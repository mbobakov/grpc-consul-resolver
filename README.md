# GRPC consul resolver

Feature rich and easy-to-use resolver which return endpoints for service from the [Hashicorp Consul](https://www.consul.io) and watch for the changes.

This library is *production ready* and will always *save backward-compatibility*

## Quick Start

For using resolving endpoints from your [Hashicorp Consul](https://www.consul.io) just import this library with `import _ "github.com/mbobakov/grpc-consul-resolver"` and pass valid connection string to the `grpc.NewClient`.

Requires Go 1.26 or newer.

For full example see [this section](#example)

## Connection string
`consul://[user:password@]127.0.0.127:8555/my-service?[healthy=]&[wait=]&[near=]&[insecure=]&[limit=]&[tag=]&[token=]`

*Parameters:*

| Name               | Format                   | Description                                                                                                                   |
|--------------------|--------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| tag                | string                   | Select endpoints only with this tag                                                                                           |
| healthy            | true/false               | Return only endpoints which pass all health-checks. Default: false                                                            |
| wait               | as in time.ParseDuration | Wait time for watch changes. Due this time period endpoints will be force refreshed. Default: inherits agent property         |
| insecure           | true/false               | Allow insecure communication with Consul. Default: true                                                                       |
| near               | string                   | Sort endpoints by response duration. Can be efficient combine with `limit` parameter default: "_agent"                        |
| limit              | int                      | Limit number of endpoints for the service. Default: no limit                                                                  |
| timeout            | as in time.ParseDuration | Http-client timeout. Default: 60s                                                                                             |
| max-backoff        | as in time.ParseDuration | Max backoff time for reconnect to consul. Reconnects will start from 10ms to _max-backoff_ exponentialy with factor 2.  Default: 1s |
| token              | string                   | Consul token                                                                                                                  |
| dc                 | string                   | Consul datacenter to choose. Optional                                                                                         |
| allow-stale        | true/false               | Allow stale results from the agent. https://www.consul.io/api/features/consistency.html#stale                                 |
| require-consistent | true/false               | RequireConsistent forces the read to be fully consistent. This is more expensive but prevents ever performing a stale read.   |

## Example
```go
package main

import (
	"log"

	_ "github.com/mbobakov/grpc-consul-resolver" // It's important

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
    conn, err := grpc.NewClient(
        "consul://127.0.0.1:8500/whoami?wait=14s&tag=manual",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin": {}}]}`),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    ...
}
```

## Development

```sh
make lint              # golangci-lint for the library and the integration tests
make test              # unit tests with the race detector
make test-integration  # integration tests against Consul in Docker. Image can be set with CONSUL_IMAGE
make generate          # regenerate mocks
```

Integration tests live in the separate `tests` module, so their dependencies (testcontainers, Docker client, etc.)
don't leak into the library `go.mod`.

## License

MIT-LICENSE. See the [LICENCE](LICENCE) file provided in the repository for details.
