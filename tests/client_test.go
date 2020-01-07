// +build integration

package tests

import (
	"testing"
	"time"

	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

func TestCLient(t *testing.T) {
	logger := logrus.New()
	grpclog.SetLoggerV2(&grpcLog{logger})
	conn, err := grpc.Dial("consul://127.0.0.1:8500/whoami?wait=14s&tag=public", grpc.WithInsecure(), grpc.WithBalancerName("round_robin"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	time.Sleep(29 * time.Second)

}

type grpcLog struct {
	*logrus.Logger
}

func (l *grpcLog) V(lvl int) bool {
	return true
}
