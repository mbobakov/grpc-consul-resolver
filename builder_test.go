package consul

import (
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
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
				service: "my-service",
				Config:  api.Config{Address: "127.0.0.127:8555"},
			},
			false},
		{"bad scheme", "127.0.0.127:8555/my-service",
			target{},
			true},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseURL(tt.input)
			assert.Equal(t, tt.err, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
