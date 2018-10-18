package consul

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/resolver"
)

func TestPopulateEndpoints(t *testing.T) {
	tests := []struct {
		name  string
		input [][]string
		want  []resolver.Address
	}{
		{"one", [][]string{{"127.0.0.1:50051"}}, []resolver.Address{{Addr: "127.0.0.1:50051"}}},
		{"sorted",
			[][]string{
				{"227.0.0.1:50051", "127.0.0.1:50051"},
			},
			[]resolver.Address{
				{Addr: "127.0.0.1:50051"},
				{Addr: "227.0.0.1:50051"},
			},
		},
		{"multy",
			[][]string{
				{"127.0.0.1:50051"},
				{"127.0.0.1:50052"},
				{"127.0.0.1:50053"},
			},
			[]resolver.Address{
				{Addr: "127.0.0.1:50053"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got []resolver.Address
				in  = make(chan []string, len(tt.input))
			)
			fcc := NewFakeClientConnDefaultFatal(t)
			fcc.NewAddressHook = func(cc []resolver.Address) {
				got = cc
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go populateEndpoints(ctx, fcc, in)
			for _, i := range tt.input {
				in <- i
			}
			time.Sleep(time.Millisecond)
			require.True(t, fcc.NewAddressCalledN(len(tt.input)))
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWatchConsulService(t *testing.T) {
	tests := []struct {
		name             string
		tgt              target
		addr             []string // port increased with 1 per invocation
		startPort        int
		times            uint64
		errorFromService bool
		want             []string
	}{
		{"simple", target{Service: "svc", Wait: time.Second}, []string{"127.0.0.1"}, 100, 3, false, []string{"127.0.0.1:102"}},
		{"error", target{}, []string{"127.0.0.1"}, 100, 3, true, nil},
		{"limit", target{Limit: 1}, []string{"127.0.0.1", "127.0.0.2", "127.0.0.3"}, 100, 1, false, []string{"127.0.0.1:100"}},
		{"limitOver", target{Limit: 10}, []string{"127.0.0.1"}, 100, 1, false, []string{"127.0.0.1:100"}},
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
			fconsul := NewFakeservicerDefaultFatal(t)

			fconsul.ServiceHook = func(s, tag string, hlz bool, q *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
				require.Equal(t, tt.tgt.Service, s)
				require.Equal(t, tt.tgt.Tag, tag)
				require.Equal(t, tt.tgt.Healthy, hlz)
				require.Equal(t, tt.tgt.Wait, q.WaitTime)
				if q.WaitIndex >= tt.times {
					select {}
				}
				if tt.errorFromService {
					return nil, &api.QueryMeta{LastIndex: q.WaitIndex + 1}, errors.New("Error")
				}
				var rr []*api.ServiceEntry
				for _, a := range tt.addr {
					rr = append(rr, &api.ServiceEntry{Service: &api.AgentService{Address: a, Port: tt.startPort + int(q.WaitIndex)}})
				}
				return rr, &api.QueryMeta{LastIndex: q.WaitIndex + 1}, nil
			}

			go watchConsulService(ctx, fconsul, tt.tgt, out)
			time.Sleep(10 * time.Millisecond)

			require.True(t, fconsul.ServiceCalledN(int(tt.times)))
			require.Equal(t, tt.want, got)
		})
	}
}
