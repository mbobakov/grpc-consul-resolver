//go:build integration

package tests

import (
	"context"
	"fmt"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	consulImage = "hashicorp/consul:1.13"
)

// SpinUpConsul run consul container and return options to connect to it
// It exports consul on random port and returns it. It makes this func suitable for parallel testing
// Don't forger to call terminate function when you are ready
func SpinUpConsul(t *testing.T) (string, func(context.Context) error, error) {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:           consulImage,
		ExposedPorts:    []string{"8500/tcp"},
		AlwaysPullImage: true,
		Cmd:             []string{"agent", "-ui", "-client", "0.0.0.0", "-dev"},
		WaitingFor:      wait.ForLog(`[INFO]  agent: Synced node info`),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	ip, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "8500/tcp")
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, err)

	uri := fmt.Sprintf("%s:%s", ip, port.Port())

	return uri, container.Terminate, nil
}

func registerService(t *testing.T, caddr, name string, port int) error {
	t.Helper()

	config := consulapi.DefaultConfig()
	config.Address = caddr

	consul, err := consulapi.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create consul client: %v", err)
	}

	registration := &consulapi.AgentServiceRegistration{
		Name:    name,
		ID:      name + "-service-" + fmt.Sprintf("%d", port),
		Port:    port,
		Address: "localhost",
		Tags:    []string{"public"},
	}

	err = consul.Agent().ServiceRegister(registration)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	return nil

}
