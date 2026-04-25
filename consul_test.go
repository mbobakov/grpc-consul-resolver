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

func TestWatchConsulService_ResolveNow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tgt := target{Service: "svc", Wait: time.Second}
	out := make(chan []string)
	rn := make(chan struct{}, 1)

	callCount := 0
	fconsul := &servicerMock{
		ServiceFunc: func(s1, s2 string, b bool, queryOptions *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
			callCount++
			if callCount == 1 {
				// First call, return some data
				return []*api.ServiceEntry{
					{Service: &api.AgentService{Address: "127.0.0.1", Port: 1024}},
				}, &api.QueryMeta{LastIndex: 100}, nil
			}
			// Subsequent calls should block until context is cancelled
			<-queryOptions.Context().Done()
			return nil, &api.QueryMeta{LastIndex: 100}, queryOptions.Context().Err()
		},
	}

	go watchConsulService(ctx, fconsul, tgt, out, rn)

	// Wait for first call
	select {
	case <-out:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for first update")
	}

	// Trigger ResolveNow
	rn <- struct{}{}

	// Wait a bit for the next call to be made
	time.Sleep(20 * time.Millisecond)

	calls := fconsul.ServiceCalls()
	require.GreaterOrEqual(t, len(calls), 2)
	// Some call after the first one should have WaitIndex 0 because of ResolveNow
	foundReset := false
	for i := 1; i < len(calls); i++ {
		if calls[i].QueryOptions.WaitIndex == 0 {
			foundReset = true
			break
		}
	}
	require.True(t, foundReset, "Should have found a call with WaitIndex 0 after ResolveNow")
}

func TestWatchConsulService_RefreshInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tgt := target{Service: "svc", Wait: time.Second, RefreshInterval: 20 * time.Millisecond}
	out := make(chan []string)
	rn := make(chan struct{}, 1)

	fconsul := &servicerMock{
		ServiceFunc: func(s1, s2 string, b bool, queryOptions *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
			return nil, &api.QueryMeta{LastIndex: 100}, nil
		},
	}

	go watchConsulService(ctx, fconsul, tgt, out, rn)

	// Wait for at least 2 calls due to ticker
	time.Sleep(100 * time.Millisecond)

	calls := fconsul.ServiceCalls()
	require.GreaterOrEqual(t, len(calls), 2)
	// Some calls should have WaitIndex 0 because of ticker
	foundReset := false
	for _, call := range calls {
		if call.QueryOptions.WaitIndex == 0 {
			foundReset = true
			break
		}
	}
	require.True(t, foundReset, "Should have found a call with WaitIndex 0 due to periodic refresh")
}

func TestResolvr_ResolveNow(t *testing.T) {
	rn := make(chan struct{}, 1)
	r := &resolvr{rn: rn}
	r.ResolveNow(resolver.ResolveNowOptions{})
	select {
	case <-rn:
	default:
		t.Fatal("ResolveNow did not send to rn channel")
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
		{"simple", target{Service: "svc", Wait: time.Second},
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
				ServiceFunc: func(s1, s2 string, b bool, queryOptions *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
					require.Equal(t, tt.tgt.Service, s1)
					require.Equal(t, tt.tgt.Tag, s2)
					require.Equal(t, tt.tgt.Healthy, b)
					require.Equal(t, tt.tgt.Near, queryOptions.Near)
					require.Equal(t, tt.tgt.Wait, queryOptions.WaitTime)
					require.Equal(t, tt.tgt.Dc, queryOptions.Datacenter)
					require.Equal(t, tt.tgt.AllowStale, queryOptions.AllowStale)
					require.Equal(t, tt.tgt.RequireConsistent, queryOptions.RequireConsistent)

					return tt.services, &api.QueryMeta{LastIndex: 1}, tt.errorFromService
				},
			}

			rn := make(chan struct{})
			go watchConsulService(ctx, fconsul, tt.tgt, out, rn)
			time.Sleep(5 * time.Millisecond)

			require.Equal(t, tt.want, got)
		})
	}
}
