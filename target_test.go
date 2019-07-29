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
		{"all-args", "consul://user:password@127.0.0.127:8555/my-service?wait=14s&near=host&insecure=true&limit=1&tag=production&token=test_token&max-backoff=2s&dc=xx&allow-stale=true&require-consistent=true",
			target{
				Addr:              "127.0.0.127:8555",
				User:              "user",
				Password:          "password",
				Service:           "my-service",
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
