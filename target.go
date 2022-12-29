package consul

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/form"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
)

const (
	targetTypeService       = "service"
	targetTypePreparedQuery = "prepared_query"

	defaultPollInterval = 30 * time.Second
)

type target struct {
	Type              string        `form:"type"`
	Addr              string        `form:"-"`
	User              string        `form:"-"`
	Password          string        `form:"-"`
	Target            string        `form:"-"`
	Wait              time.Duration `form:"wait"`
	Timeout           time.Duration `form:"timeout"`
	MaxBackoff        time.Duration `form:"max-backoff"`
	PollInterval      time.Duration `form:"poll-interval"`
	Tag               string        `form:"tag"`
	Near              string        `form:"near"`
	Limit             int           `form:"limit"`
	Healthy           bool          `form:"healthy"`
	TLSInsecure       bool          `form:"insecure"`
	Token             string        `form:"token"`
	Dc                string        `form:"dc"`
	AllowStale        bool          `form:"allow-stale"`
	RequireConsistent bool          `form:"require-consistent"`
	// TODO(mbobakov): custom parameters for the http-transport
	// TODO(mbobakov): custom parameters for the TLS subsystem
}

func (t *target) String() string {
	return fmt.Sprintf("%s='%s' healthy='%t' tag='%s'", t.Type, t.Target, t.Healthy, t.Tag)
}

//	parseURL with parameters
//
// see README.md for the actual format
// URL schema will stay stable in the future for backward compatibility
func parseURL(u string) (target, error) {
	rawURL, err := url.Parse(u)
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL")
	}

	if rawURL.Scheme != schemeName ||
		len(rawURL.Host) == 0 || len(strings.TrimLeft(rawURL.Path, "/")) == 0 {
		return target{},
			errors.Errorf("Malformed URL('%s'). Must be in the next format: 'consul://[user:passwd]@host/service?param=value'", u)
	}

	var tgt target
	tgt.User = rawURL.User.Username()
	tgt.Password, _ = rawURL.User.Password()
	tgt.Addr = rawURL.Host
	tgt.Target = strings.TrimLeft(rawURL.Path, "/")
	decoder := form.NewDecoder()
	decoder.RegisterCustomTypeFunc(func(vals []string) (interface{}, error) {
		return time.ParseDuration(vals[0])
	}, time.Duration(0))

	err = decoder.Decode(&tgt, rawURL.Query())
	if err != nil {
		return target{}, errors.Wrap(err, "Malformed URL parameters")
	}
	if tgt.Type == "" {
		tgt.Type = targetTypeService
	}
	if tgt.Type != targetTypeService && tgt.Type != targetTypePreparedQuery {
		return target{}, errors.Errorf(`"type" must be either %q or %q`, targetTypeService, targetTypePreparedQuery)
	}
	if tgt.Type == targetTypeService && tgt.PollInterval > 0 {
		return target{}, errors.Errorf(`"poll-interval" can only be set when type=%q`, targetTypePreparedQuery)
	}
	if tgt.Type == targetTypePreparedQuery && tgt.Wait > 0 {
		return target{}, errors.Errorf(`"wait" can only be set when type=%q`, targetTypeService)
	}
	if tgt.Type == targetTypePreparedQuery && tgt.Healthy {
		return target{}, errors.Errorf(`"healthy" can only be set when type=%q`, targetTypeService)
	}
	if tgt.Type == targetTypePreparedQuery && tgt.Tag != "" {
		return target{}, errors.Errorf(`"tag" can only be set when type=%q`, targetTypeService)
	}
	if len(tgt.Near) == 0 {
		tgt.Near = "_agent"
	}
	if tgt.MaxBackoff == 0 {
		tgt.MaxBackoff = time.Second
	}
	if tgt.Type == targetTypePreparedQuery && tgt.PollInterval == 0 {
		tgt.PollInterval = defaultPollInterval
	}
	return tgt, nil
}

// consulConfig returns config based on the parsed target.
// It uses custom http-client.
func (t *target) consulConfig() *api.Config {
	var creds *api.HttpBasicAuth
	if len(t.User) > 0 && len(t.Password) > 0 {
		creds = new(api.HttpBasicAuth)
		creds.Password = t.Password
		creds.Username = t.User
	}
	// custom http.Client
	c := &http.Client{
		Timeout: t.Timeout,
	}
	return &api.Config{
		Address:    t.Addr,
		HttpAuth:   creds,
		WaitTime:   t.Wait,
		HttpClient: c,
		TLSConfig: api.TLSConfig{
			InsecureSkipVerify: t.TLSInsecure,
		},
		Token: t.Token,
	}
}
