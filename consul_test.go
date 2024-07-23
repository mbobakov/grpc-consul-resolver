package consul

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/resolver"
)

func TestPopulateEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		wantCall []resolver.Address
	}{
		{"one",
			[]string{"127.0.0.1:50051"},
			[]resolver.Address{
				{Addr: "127.0.0.1:50051"},
			},
		},
		{"sorted",
			[]string{"227.0.0.1:50051", "127.0.0.1:50051"},
			[]resolver.Address{
				{Addr: "127.0.0.1:50051"},
				{Addr: "227.0.0.1:50051"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				in = make(chan []string, len(tt.input))
			)

			fcc := &ClientConnMock{
				UpdateStateFunc: func(state resolver.State) error {
					require.Equal(t, tt.wantCall, state.Addresses)
					return nil
				},
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go populateEndpoints(ctx, fcc, in)
			in <- tt.input
			time.Sleep(time.Millisecond)

			require.Equal(t, 1, len(fcc.UpdateStateCalls()))
		})
	}
}

func TestWatchConsulService(t *testing.T) {
	tests := []struct {
		name             string
		tgt              target
		services         []*api.ServiceEntry
		errorFromService error
		want             []string
	}{
		{"simple", target{Service: "svc", Tags: []string{"foo", "bar"}, Wait: time.Second},
			[]*api.ServiceEntry{
				{
					Service: &api.AgentService{Address: "127.0.0.1", Port: 1024},
				},
			},
			nil,
			[]string{"127.0.0.1:1024"},
		},
		// TODO: Add more tests-cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var (
				got []string
				out = make(chan []string)
			)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case got = <-out:
					}
				}
			}()
			fconsul := &servicerMock{
				ServiceMultipleTagsFunc: func(s1 string, s2 []string, b bool, queryOptions *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
					require.Equal(t, tt.tgt.Service, s1)
					require.Equal(t, tt.tgt.Tags, s2)
					require.Equal(t, tt.tgt.Healthy, b)
					require.Equal(t, tt.tgt.Near, queryOptions.Near)
					require.Equal(t, tt.tgt.Wait, queryOptions.WaitTime)
					require.Equal(t, tt.tgt.Dc, queryOptions.Datacenter)
					require.Equal(t, tt.tgt.AllowStale, queryOptions.AllowStale)
					require.Equal(t, tt.tgt.RequireConsistent, queryOptions.RequireConsistent)

					return tt.services, &api.QueryMeta{LastIndex: 1}, tt.errorFromService
				},
			}

			go watchConsulService(ctx, fconsul, tt.tgt, out)
			time.Sleep(5 * time.Millisecond)

			require.Equal(t, tt.want, got)
		})
	}
}
