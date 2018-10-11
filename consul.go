package consul

import (
	"fmt"
	"sort"
	"time"

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
	target               target
	client               *api.Client
	done                 chan struct{}
	cc                   resolver.ClientConn
	disableServiceConfig bool
}

type target struct {
	tag     string
	healthy bool
	service string
	api.Config
	// TODO(mbobakov): custom parameters for the http-transport
	// TODO(mbobakov): custom parameters for the TLS subsystem
}

func (t *target) String() string {
	return fmt.Sprintf("service=%s healthy=%t tag=%s", t.service, t.healthy, t.tag)
}

// ResolveNow will be skipped due unnecessary in this case
func (r *resolvr) ResolveNow(resolver.ResolveNowOption) {}

// Close closes the resolver.
func (r *resolvr) Close() {
	close(r.done)
}

func (r *resolvr) watch() {
	t := r.target
	connsCh := make(chan []resolver.Address)
	go func() {
		var lastIndex uint64
		for {
			ts := time.Now()
			ss, meta, err := r.client.Health().Service(
				t.service,
				t.tag,
				t.healthy,
				&api.QueryOptions{
					WaitIndex: lastIndex,
				},
			)
			if err != nil {
				grpclog.Errorf("[Consul resolver] Couldn't fetch endpoints. target={%s}", t.String())
				continue
			}
			grpclog.Infof("[Consul resolver] %d endpoints fetched in(+wait) %s for target={%s}",
				len(ss),
				time.Since(ts),
				t.String(),
			)
			lastIndex = meta.LastIndex
			if len(ss) == 0 {
				continue
			}

			connsSet := make(map[string]struct{}, len(ss))
			for _, s := range ss {
				connsSet[fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port)] = struct{}{}
			}
			conns := make([]resolver.Address, 0, len(connsSet))
			for c := range connsSet {
				conns = append(conns, resolver.Address{Addr: c})
			}

			sort.Sort(byAddressString(conns)) // Don't replace the same address list in the balancer
			connsCh <- conns
		}
	}()
	for {
		select {
		case cc := <-connsCh:
			r.cc.NewAddress(cc)
		case <-r.done:
			grpclog.Info("[Consul resolver] Watch has been finiched")
			return
		}
	}
}

// byAddressString sorts resolver.Address by Address Field  sorting in increasing order.
type byAddressString []resolver.Address

func (p byAddressString) Len() int           { return len(p) }
func (p byAddressString) Less(i, j int) bool { return p[i].Addr < p[j].Addr }
func (p byAddressString) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
