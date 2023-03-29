//go:build integration

package tests

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

// rungRPC starts a gRPC greeting server on the given port
// more info: https://github.com/grpc/grpc-go/tree/master/examples/helloworld
// This is blocking, so it should be run in a goroutine
// gRPC server is stopped when the context is cancelled
// gRPC server is using reflection
func rungRPC(ctx context.Context, port int) error {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{port: port})
	reflection.Register(s)

	go stopWhenDone(ctx, s)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

func stopWhenDone(ctx context.Context, server *grpc.Server) {
	<-ctx.Done()
	server.GracefulStop()
}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
	port int
}

// SayHello implements helloworld.GreeterServer
// It just returns port number as greeting
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: strconv.Itoa(s.port)}, nil
}

func findNextFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to listen: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
