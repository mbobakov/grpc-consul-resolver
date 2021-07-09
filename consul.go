package consul

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/jpillora/backoff"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
)

// init function needs for  auto-register in resolvers registry
func init() {
	resolver.Register(&builder{})
}

// resolvr implements resolver.Resolver from the gRPC package.
// It watches for endpoints changes and pushes them to the underlying gRPC connection.
type resolvr struct {
	cancelFunc context.CancelFunc
}

// ResolveNow will be skipped due unnecessary in this case
func (r *resolvr) ResolveNow(resolver.ResolveNowOptions) {}

// Close closes the resolver.
func (r *resolvr) Close() {
	r.cancelFunc()
}

//go:generate mockgen -package mocks -destination internal/mocks/resolverClientConn.go  google.golang.org/grpc/resolver ClientConn
//go:generate mockgen -package mocks -destination internal/mocks/servicer.go -source consul.go servicer
type servicer interface {
	Service(string, string, bool, *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error)
}

func watchConsulService(ctx context.Context, s servicer, tgt target, out chan<- []*api.ServiceEntry) {
	bck := &backoff.Backoff{
		Factor: 2,
		Jitter: true,
		Min:    10 * time.Millisecond,
		Max:    tgt.MaxBackoff,
	}

	var lastIndex uint64
	for {
		ss, meta, err := s.Service(
			tgt.Service,
			tgt.Tag,
			tgt.Healthy,
			&api.QueryOptions{
				WaitIndex:         lastIndex,
				Near:              tgt.Near,
				WaitTime:          tgt.Wait,
				Datacenter:        tgt.Dc,
				AllowStale:        tgt.AllowStale,
				RequireConsistent: tgt.RequireConsistent,
			},
		)
		if err != nil {
			grpclog.Errorf("[Consul resolver] Couldn't fetch endpoints. target={%s}; error={%v}", tgt.String(), err)
			time.Sleep(bck.Duration())
			continue
		}

		bck.Reset()

		if meta.LastIndex == lastIndex {
			grpclog.Info("[Consul resolver] no change")
			continue
		}

		if meta.LastIndex < lastIndex {
			// according to https://www.consul.io/api-docs/features/blocking
			// we should reset the index if it goes backward
			lastIndex = 0
		} else {
			lastIndex = meta.LastIndex
		}

		grpclog.Infof("[Consul resolver] %d endpoints fetched in(+wait) %s for target={%s}",
			len(ss),
			meta.RequestTime,
			tgt.String(),
		)

		if tgt.Limit != 0 && len(ss) > tgt.Limit {
			ss = ss[:tgt.Limit]
		}

		select {
		case out <- ss:
			continue
		case <-ctx.Done():
			return
		}
	}
}

type consulResolverKeyType string

const (
	createIndexKey = consulResolverKeyType("grpc.consul.resolver.create_index")
	modifyIndexKey = consulResolverKeyType("grpc.consul.resolver.modify_index")
)

func populateEndpoints(
	ctx context.Context,
	clientConn resolver.ClientConn,
	input <-chan []*api.ServiceEntry,
	agentNodeName string,
) {
	for {
		select {
		case in := <-input:
			// sort services to not replace the same address list in the balancer.
			sort.Sort(sameNodeFirst{
				agentNodeName: agentNodeName,
				in:            in,
			})

			addrs := make([]resolver.Address, 0, len(in))
			for _, s := range in {
				addrs = append(addrs, resolver.Address{
					Addr: fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port),
					Attributes: attributes.New(
						createIndexKey, s.Service.CreateIndex,
						modifyIndexKey, s.Service.ModifyIndex,
					),
				})
			}

			clientConn.UpdateState(resolver.State{Addresses: addrs})
		case <-ctx.Done():
			grpclog.Info("[Consul resolver] Watch has been finished")
			return
		}
	}
}

// sameNodeFirst sorts services so that services on the same
// node go first, then go others in lexicographic order.
type sameNodeFirst struct {
	agentNodeName string
	in            []*api.ServiceEntry
}

func (n sameNodeFirst) Len() int      { return len(n.in) }
func (n sameNodeFirst) Swap(i, j int) { n.in[i], n.in[j] = n.in[j], n.in[i] }

func (n sameNodeFirst) Less(i, j int) bool {
	if n.in[i].Node.Node == n.agentNodeName && n.in[j].Node.Node == n.agentNodeName {
		return n.in[i].Service.Address < n.in[j].Service.Address
	}

	if n.in[i].Node.Node == n.agentNodeName {
		return true
	}

	if n.in[j].Node.Node == n.agentNodeName {
		return false
	}

	return n.in[i].Service.Address < n.in[j].Service.Address
}
