package consul

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"google.golang.org/grpc/resolver"
)

// schemeName for the urls
// All target URLs like 'consul://.../...' will be resolved by this resolver
const schemeName = "consul"

// builder implements resolver.Builder and use for constructing all consul resolvers
type builder struct{}

func (b *builder) Build(url resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	tgt, err := parseURL(url.Authority, url.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "Wrong consul URL")
	}
	cli, err := api.NewClient(&tgt.Config)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't connect to the Consul API")
	}
	r := resolvr{
		client:               cli,
		target:               tgt,
		done:                 make(chan struct{}),
		cc:                   cc,
		disableServiceConfig: opts.DisableServiceConfig,
	}

	go r.watch()
	return &r, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return schemeName
}

func parseURL(athority, u string) (target, error) {
	const (
		waitTimeParamName   = "wait"
		onlyTagsParamName   = "only_tags"
		onlyHealthParamName = "only_health"
		listSeparator       = ","
	)
	rawURL, err := url.Parse(strings.Join([]string{athority, u}, "/"))
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL")
	}
	var creds api.HttpBasicAuth
	passwd, ok := rawURL.User.Password()
	if ok {
		creds.Password = passwd
		creds.Username = rawURL.User.Username()
	}

	waitTime, err := time.ParseDuration(rawURL.Query().Get(waitTimeParamName))
	if err != nil {
		return target{}, errors.Wrapf(err, "Malformed URL parameter: %s", waitTimeParamName)
	}

	var onlyHealth bool
	if oh := rawURL.Query().Get(onlyHealthParamName); len(oh) > 0 {
		onlyHealth, err = strconv.ParseBool(oh)
		if err != nil {
			return target{}, errors.Wrapf(err, "Malformed URL parameter: %s", onlyHealthParamName)
		}
	}
	return target{
		service:    rawURL.Opaque,
		onlyHealth: onlyHealth,
		onlyTags:   strings.Split(rawURL.Query().Get(onlyTagsParamName), listSeparator),
		Config: api.Config{
			Address:  rawURL.Host,
			WaitTime: waitTime,
			HttpAuth: &creds,
		},
	}, nil
}
