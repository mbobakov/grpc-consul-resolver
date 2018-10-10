package consul

import (
	"github.com/hashicorp/consul/api"
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
	onlyTags   []string
	onlyHealth bool
	service    string
	api.Config
	// TODO(mbobakov): custom parameters for the http-transport
	// TODO(mbobakov): custom parameters for the TLS subsystem
}

// ResolveNow will be skipped due unnecessary
func (r *resolvr) ResolveNow(resolver.ResolveNowOption) {}

// Close closes the resolver.
func (r *resolvr) Close() {
	close(r.done)
}

func (r *resolvr) watch() {
	<-r.done
}
