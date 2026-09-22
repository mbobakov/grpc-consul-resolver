//go:build integration

package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// defaultConsulImage can be overridden with the CONSUL_IMAGE environment variable
const defaultConsulImage = "hashicorp/consul:2.0"

// SpinUpConsul run consul container and return address to connect to it
// It exports consul on random port and returns it. It makes this func suitable for parallel testing
// The container is terminated automatically when the test finishes
func SpinUpConsul(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	image := os.Getenv("CONSUL_IMAGE")
	if image == "" {
		image = defaultConsulImage
	}

	container, err := testcontainers.Run(ctx, image,
		testcontainers.WithExposedPorts("8500/tcp"),
		testcontainers.WithCmd("agent", "-ui", "-client", "0.0.0.0", "-dev"),
		testcontainers.WithWaitStrategy(
			// Wait for the leader election. Before that the agent can't serve catalog requests
			wait.ForHTTP("/v1/status/leader").
				WithPort("8500/tcp").
				WithResponseMatcher(func(body io.Reader) bool {
					b, err := io.ReadAll(body)
					return err == nil && !bytes.Equal(bytes.TrimSpace(b), []byte(`""`))
				}),
		),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err)

	uri, err := container.PortEndpoint(ctx, "8500/tcp", "")
	require.NoError(t, err)

	return uri
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
