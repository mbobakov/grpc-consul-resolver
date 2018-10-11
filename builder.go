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
	dsn := strings.Join([]string{schemeName + ":/", url.Authority, url.Endpoint}, "/")
	tgt, err := parseURL(dsn)
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

func parseURL(u string) (target, error) {
	const (
		waitTimeParamName = "wait"
		tagParamName      = "tag"
		healthyParamName  = "healthy"
		listSeparator     = ","
	)
	rawURL, err := url.Parse(u)
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL")
	}
	if rawURL.Scheme != schemeName ||
		len(rawURL.Host) == 0 || len(strings.TrimLeft(rawURL.Path, "/")) == 0 {
		return target{},
			errors.Errorf("Malformed URL('%s'). Must be in the next format: 'consul://[user:passwd]@host/service?param=value'", u)
	}

	var creds *api.HttpBasicAuth
	passwd, ok := rawURL.User.Password()
	if ok {
		creds.Password = passwd
		creds.Username = rawURL.User.Username()
	}

	var waitTime time.Duration
	if wt := rawURL.Query().Get(waitTimeParamName); len(wt) > 0 {
		waitTime, err = time.ParseDuration(wt)
		if err != nil {
			return target{}, errors.Wrapf(err, "Malformed URL parameter: '%s'", waitTimeParamName)
		}
	}

	var onlyHealth bool
	if oh := rawURL.Query().Get(healthyParamName); len(oh) > 0 {
		onlyHealth, err = strconv.ParseBool(oh)
		if err != nil {
			return target{}, errors.Wrapf(err, "Malformed URL parameter: '%s'", healthyParamName)
		}
	}

	return target{
		service: strings.TrimLeft(rawURL.Path, "/"),
		healthy: onlyHealth,
		tag:     rawURL.Query().Get(tagParamName),
		Config: api.Config{
			Address:  rawURL.Host,
			WaitTime: waitTime,
			HttpAuth: creds,
		},
	}, nil
}
