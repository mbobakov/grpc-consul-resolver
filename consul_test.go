package consul

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/consul/api"
	"github.com/mbobakov/grpc-consul-resolver/internal/mocks"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/resolver"
)

func TestPopulateEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		input     [][]string
		wantCalls [][]resolver.Address
	}{
		{"one",
			[][]string{{"127.0.0.1:50051"}},
			[][]resolver.Address{
				[]resolver.Address{
					{Addr: "127.0.0.1:50051"},
				},
			},
		},
		{"sorted",
			[][]string{
				{"227.0.0.1:50051", "127.0.0.1:50051"},
			},
			[][]resolver.Address{
				[]resolver.Address{
					{Addr: "127.0.0.1:50051"},
					{Addr: "227.0.0.1:50051"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			var (
				in = make(chan []string, len(tt.input))
			)

			fcc := mocks.NewMockClientConn(ctrl)
			for _, aa := range tt.wantCalls {
				fcc.EXPECT().UpdateState(resolver.State{Addresses: aa}).Times(1)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go populateEndpoints(ctx, fcc, in)
			for _, i := range tt.input {
				in <- i
			}
			time.Sleep(time.Millisecond)
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
		{"simple", target{Service: "svc", Wait: time.Second},
			[]*api.ServiceEntry{
				&api.ServiceEntry{
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

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
			fconsul := mocks.NewMockservicer(ctrl)
			fconsul.EXPECT().Service(tt.tgt.Service, tt.tgt.Tag, tt.tgt.Healthy, &api.QueryOptions{
				WaitIndex:         0,
				Near:              tt.tgt.Near,
				WaitTime:          tt.tgt.Wait,
				Datacenter:        tt.tgt.Dc,
				AllowStale:        tt.tgt.AllowStale,
				RequireConsistent: tt.tgt.RequireConsistent,
			}).
				Times(1).
				Return(tt.services, &api.QueryMeta{LastIndex: 1}, tt.errorFromService)
			fconsul.EXPECT().Service(tt.tgt.Service, tt.tgt.Tag, tt.tgt.Healthy, &api.QueryOptions{
				WaitIndex:         1,
				Near:              tt.tgt.Near,
				WaitTime:          tt.tgt.Wait,
				Datacenter:        tt.tgt.Dc,
				AllowStale:        tt.tgt.AllowStale,
				RequireConsistent: tt.tgt.RequireConsistent,
			}).
				Do(
					func(svc string, tag string, h bool, opt *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
						if opt.WaitIndex > 0 {
							select {}
						}
						return tt.services, &api.QueryMeta{LastIndex: 1}, tt.errorFromService
					},
				).Times(1).
				Return(tt.services, &api.QueryMeta{LastIndex: 1}, tt.errorFromService)

			go watchConsulService(ctx, fconsul, tt.tgt, out)
			time.Sleep(5 * time.Millisecond)

			require.Equal(t, tt.want, got)
		})
	}
}
