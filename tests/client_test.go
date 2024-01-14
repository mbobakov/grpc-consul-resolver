//go:build integration

package tests

import (
	"context"
	"strconv"
	"testing"

	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/grpclog"
)

func TestClient(t *testing.T) {
	logger := logrus.New()
	grpclog.SetLoggerV2(&grpcLog{logger})

	const grpcServiceConfig = `{"loadBalancingConfig": [ { "round_robin": {} } ]}`
	// Context for the whole test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spin up a Consul server
	consulURI, teardown, err := SpinUpConsul(t)
	defer teardown(ctx)
	require.NoError(t, err)

	// counter map of destination ports
	portCounter := make(map[int]int)
	// Spin up a 2 gRPC servers
	g := errgroup.Group{}
	for i := 0; i < 2; i++ {
		port, err := findNextFreePort()
		require.NoError(t, err)

		portCounter[port] = 0

		g.Go(func() error { return rungRPC(ctx, port) })
		// Register the gRPC server with Consul
		err = registerService(t, consulURI, "helloworld", port)
		require.NoError(t, err)
	}

	conn, err := grpc.Dial(
		"consul://"+consulURI+"/helloworld?wait=14s&tag=public",
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultServiceConfig(grpcServiceConfig),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(128e+6)),
	)
	defer conn.Close()
	require.NoError(t, err)

	// create a client and call the server
	client := pb.NewGreeterClient(conn)

	// call the server several times to make sure that the load balancing works
	for i := 0; i < 10; i++ {
		resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: "world"})
		require.NoError(t, err)
		port, err := strconv.Atoi(resp.Message)
		require.NoError(t, err)
		portCounter[port]++
	}

	for p, c := range portCounter {
		t.Logf("port counter %d got %d requests", p, c)
		require.GreaterOrEqual(t, c, 2)
	}

}

type grpcLog struct {
	*logrus.Logger
}

func (l *grpcLog) V(lvl int) bool {
	return true
}
