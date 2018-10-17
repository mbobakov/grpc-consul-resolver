### Connection string
`consul://[user:password@]127.0.0.127:8555/my-service?[healthy=]&[wait=]&[near=]&[insecure=]&[limit=]&[tag=]`

*Parameters:*

| Name     	| Format                   	| Description                                                                                                           	|
|----------	|--------------------------	|-----------------------------------------------------------------------------------------------------------------------	|
| tag      	| string                   	| Select endpoints only with this tag                                                                                   	|
| healthy  	| true/false               	| Return only endpoints which pass all health-checks. Default: false                                                    	|
| wait     	| as in time.ParseDuration 	| Wait time for watch changes. Due this time period endpoints will be force refreshed. Default: inherits agent property 	|
| insecure 	| true/false               	| Allow insecure communication with Consul. Default: true                                                               	|
| near     	| string                   	| Sort endpoints by response duration. Can be efficient combine with `limit` parameter default: "_agent"                	|
| limit    	| int                      	| Limit number of endpoints for the service. Default: no limit                                                          	|
| timeout  	| as in time.ParseDuration 	| Http-client timeout. Default: 60s                                                                                     	|

### Example
```go
package main

import (
	"time"
	
	_ "github.com/mbobakov/grpc-consul-resolver" // It's important
	
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial( "consul://127.0.0.1:8500/whoami?wait=14s&tag=manual",  // See connection string section !
        grpc.WithInsecure(), 
        grpc.WithBalancerName("round_robin"),
    )
	if err != nil {
		t.Fatal(err)
	}
    defer conn.Close()
    ...
}
```
