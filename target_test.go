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
				Addr:       "127.0.0.127:8555",
				Service:    "my-service",
				Near:       "_agent",
				MaxBackoff: time.Second,
			},
			false,
		},
		{"all-params", "consul://user:password@127.0.0.127:8555/my-service?allow-stale=true&dc=yy&disable-compression=true&expect-continue-timeout=3s&healthy=true&idle-conn-timeout=10s&insecure=true&limit=5&max-backoff=2s&max-idle-conns=99&near=host&require-consistent=true&tag=production&timeout=11s&tls-handshake-timeout=5s&token=test_token&wait=14s",
			target{
				Addr:                  "127.0.0.127:8555",
				User:                  "user",
				Password:              "password",
				Service:               "my-service",
				Wait:                  14 * time.Second,
				Timeout:               11 * time.Second,
				MaxBackoff:            2 * time.Second,
				Tag:                   "production",
				Near:                  "host",
				Limit:                 5,
				Healthy:               true,
				TLSInsecure:           true,
				Token:                 "test_token",
				Dc:                    "yy",
				AllowStale:            true,
				RequireConsistent:     true,
				MaxIdleConns:          99,
				IdleConnTimeout:       10 * time.Second,
				DisableCompression:    true,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 3 * time.Second,
			},
			false,
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
