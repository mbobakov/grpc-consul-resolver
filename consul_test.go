package consul

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/consul/api"
	"github.com/mbobakov/grpc-consul-resolver/internal/mocks"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
)

func TestPopulateEndpoints(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name     string
		nodeName string
		input    []*api.ServiceEntry
		want     []resolver.Address
	}{
		{
			name:     "one",
			nodeName: "node-1",
			input: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address:     "127.0.0.1",
						Port:        50051,
						CreateIndex: 100,
						ModifyIndex: 100,
					},
				},
			},
			want: []resolver.Address{
				{
					Addr:       "127.0.0.1:50051",
					Attributes: attributes.New(createIndexKey, uint64(100), modifyIndexKey, uint64(100)),
				},
			},
		},
		{
			name:     "two",
			nodeName: "node-2",
			input: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address:     "127.0.0.1",
						Port:        50051,
						CreateIndex: 100,
						ModifyIndex: 100,
					},
				},
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address:     "227.0.0.1",
						Port:        50051,
						CreateIndex: 101,
						ModifyIndex: 101,
					},
				},
			},
			want: []resolver.Address{
				{
					Addr:       "227.0.0.1:50051",
					Attributes: attributes.New(createIndexKey, uint64(101), modifyIndexKey, uint64(101)),
				},
				{
					Addr:       "127.0.0.1:50051",
					Attributes: attributes.New(createIndexKey, uint64(100), modifyIndexKey, uint64(100)),
				},
			},
		},
	}
	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			clientConnMock := mocks.NewMockClientConn(ctrl)
			clientConnMock.EXPECT().UpdateState(resolver.State{Addresses: tc.want})

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			in := make(chan []*api.ServiceEntry, 1)
			in <- tc.input

			go populateEndpoints(ctx, clientConnMock, in, tc.nodeName)

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
		want             []*api.ServiceEntry
	}{
		{
			name: "no limit",
			tgt: target{
				Service: "svc",
				Wait:    time.Second,
			},
			services: []*api.ServiceEntry{
				{
					Service: &api.AgentService{Address: "127.0.0.1", Port: 1024},
				},
			},
			want: []*api.ServiceEntry{
				{
					Service: &api.AgentService{Address: "127.0.0.1", Port: 1024},
				},
			},
		},
		{
			name: "with limit",
			tgt: target{
				Service: "svc",
				Wait:    time.Second,
				Limit:   1,
			},
			services: []*api.ServiceEntry{
				{
					Service: &api.AgentService{Address: "129.0.0.1", Port: 1024},
				},
				{
					Service: &api.AgentService{Address: "128.0.0.1", Port: 1024},
				},
				{
					Service: &api.AgentService{Address: "127.0.0.1", Port: 1024},
				},
			},
			want: []*api.ServiceEntry{
				{
					Service: &api.AgentService{Address: "129.0.0.1", Port: 1024},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			ctrl := gomock.NewController(t)

			var (
				got []*api.ServiceEntry
				out = make(chan []*api.ServiceEntry)
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

func TestSortSameNodeFirst(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name     string
		nodeName string
		in       []*api.ServiceEntry
		expect   []*api.ServiceEntry
	}{
		{
			name:     "one service on agent node",
			nodeName: "node-1",
			in: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
			expect: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
		},
		{
			name:     "one service on different node",
			nodeName: "node-1",
			in: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
			expect: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
		},
		{
			name:     "two services on agent node",
			nodeName: "node-1",
			in: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "227.0.0.1",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
			expect: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "227.0.0.1",
						Port:    50051,
					},
				},
			},
		},
		{
			name:     "two services on different nodes",
			nodeName: "node-1",
			in: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "227.0.0.1",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
			expect: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "227.0.0.1",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "127.0.0.1",
						Port:    50051,
					},
				},
			},
		},
		{
			name:     "three services on different nodes",
			nodeName: "node-1",
			in: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "192.168.235.110",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "192.168.235.116",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-3",
					},
					Service: &api.AgentService{
						Address: "192.168.235.115",
						Port:    50051,
					},
				},
			},
			expect: []*api.ServiceEntry{
				{
					Node: &api.Node{
						Node: "node-1",
					},
					Service: &api.AgentService{
						Address: "192.168.235.110",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-3",
					},
					Service: &api.AgentService{
						Address: "192.168.235.115",
						Port:    50051,
					},
				},
				{
					Node: &api.Node{
						Node: "node-2",
					},
					Service: &api.AgentService{
						Address: "192.168.235.116",
						Port:    50051,
					},
				},
			},
		},
	}

	for i := range tt {
		tc := tt[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sort.Sort(sameNodeFirst{
				agentNodeName: tc.nodeName,
				in:            tc.in,
			})

			require.Equal(t, tc.expect, tc.in)
		})
	}
}
