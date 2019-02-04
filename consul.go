package consul

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
)

// init function needs for  auto-register in resolvers registry
func init() {
	resolver.Register(&builder{})
}

// consulResolver watch for enpoints changes and push to the GRPC only diffs
// consulResolver implements resolver.Resolver from the GRPC package
type resolvr struct {
	cancelFunc context.CancelFunc
}

// ResolveNow will be skipped due unnecessary in this case
func (r *resolvr) ResolveNow(resolver.ResolveNowOption) {}

// Close closes the resolver.
func (r *resolvr) Close() {
	r.cancelFunc()
}

type servicer interface {
	Service(string, string, bool, *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error)
}

func watchConsulService(ctx context.Context, s servicer, tgt target, out chan<- []string) {
	res := make(chan []string)
	go func() {
		var lastIndex uint64
		for {
			ss, meta, err := s.Service(
				tgt.Service,
				tgt.Tag,
				tgt.Healthy,
				&api.QueryOptions{
					WaitIndex: lastIndex,
					Near:      tgt.Near,
					WaitTime:  tgt.Wait,
				},
			)
			if err != nil {
				grpclog.Errorf("[Consul resolver] Couldn't fetch endpoints. target={%s}", tgt.String())
				continue
			}
			lastIndex = meta.LastIndex
			grpclog.Infof("[Consul resolver] %d endpoints fetched in(+wait) %s for target={%s}",
				len(ss),
				meta.RequestTime,
				tgt.String(),
			)

			ee := make([]string, 0, len(ss))
			for _, s := range ss {
				ee = append(ee, fmt.Sprintf("%s:%d", s.Node.Address, s.Service.Port))
			}
			if tgt.Limit != 0 && len(ee) > tgt.Limit {
				ee = ee[:tgt.Limit]
			}
			res <- ee
		}
	}()

	for {
		select {
		case ee := <-res:
			out <- ee
		case <-ctx.Done():
			return
		}
	}
}

func populateEndpoints(ctx context.Context, clientConn resolver.ClientConn, input <-chan []string) {
	for {
		select {
		case cc := <-input:
			connsSet := make(map[string]struct{}, len(cc))
			for _, c := range cc {
				connsSet[c] = struct{}{}
			}
			conns := make([]resolver.Address, 0, len(connsSet))
			for c := range connsSet {
				conns = append(conns, resolver.Address{Addr: c})
			}
			sort.Sort(byAddressString(conns)) // Don't replace the same address list in the balancer
			clientConn.NewAddress(conns)
		case <-ctx.Done():
			grpclog.Info("[Consul resolver] Watch has been finished")
			return
		}
	}
}

// byAddressString sorts resolver.Address by Address Field  sorting in increasing order.
type byAddressString []resolver.Address

func (p byAddressString) Len() int           { return len(p) }
func (p byAddressString) Less(i, j int) bool { return p[i].Addr < p[j].Addr }
func (p byAddressString) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
