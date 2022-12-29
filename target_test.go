package consul

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  target
		err   bool
	}{
		{"simple", "consul://127.0.0.127:8555/my-service",
			target{
				Type:       "service",
				Addr:       "127.0.0.127:8555",
				Target:     "my-service",
				Near:       "_agent",
				MaxBackoff: time.Second,
			},
			false,
		},
		{"all-args", "consul://user:password@127.0.0.127:8555/my-service?wait=14s&near=host&insecure=true&limit=1&tag=production&token=test_token&max-backoff=2s&dc=xx&allow-stale=true&require-consistent=true",
			target{
				Type:              "service",
				Addr:              "127.0.0.127:8555",
				User:              "user",
				Password:          "password",
				Target:            "my-service",
				Near:              "host",
				Wait:              14 * time.Second,
				TLSInsecure:       true,
				Limit:             1,
				Tag:               "production",
				Token:             "test_token",
				MaxBackoff:        2 * time.Second,
				Dc:                "xx",
				AllowStale:        true,
				RequireConsistent: true,
			},
			false,
		},
		{"prepared-query", "consul://127.0.0.127:8555/my-query?type=prepared_query",
			target{
				Type:         "prepared_query",
				Addr:         "127.0.0.127:8555",
				Target:       "my-query",
				Near:         "_agent",
				MaxBackoff:   time.Second,
				PollInterval: 30 * time.Second,
			},
			false,
		},
		{"prepared-query-poll-interval", "consul://127.0.0.127:8555/my-query?type=prepared_query&poll-interval=10s",
			target{
				Type:         "prepared_query",
				Addr:         "127.0.0.127:8555",
				Target:       "my-query",
				Near:         "_agent",
				MaxBackoff:   time.Second,
				PollInterval: 10 * time.Second,
			},
			false,
		},
		{"prepared-query-with-wait", "consul://127.0.0.127:8555/my-query?type=prepared_query&wait=10s",
			target{},
			true,
		},
		{"prepared-query-with-tag", "consul://127.0.0.127:8555/my-query?type=prepared_query&tag=my-tag",
			target{},
			true,
		},
		{"prepared-query-with-healthy", "consul://127.0.0.127:8555/my-query?type=prepared_query&healthy=true",
			target{},
			true,
		},
		{"bad-scheme", "127.0.0.127:8555/my-service",
			target{},
			true,
		},
		{"no-service", "consul://127.0.0.127:8555",
			target{},
			true,
		},
		{"bad-arg", "consul://127.0.0.127:8555/s?insecure=BADDD",
			target{},
			true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.input)
			require.Equal(t, tt.err, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
