## go-mciro plugins

将go-micro服务加入service mesh，Client、Server不需要Registry、Selector、Transport等
- client/istio_http
- server/istio_http

> TODO
- Server/Service address定义
- 支持server.HandlerOption定义API
- 支持stream

```go
package main

import (
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
    "github.com/micro/go-plugins/registry/noop"
	
    httpClient "github.com/hb-go/micro-plugins/client/istio_http"
	httpServer "github.com/hb-go/micro-plugins/server/istio_http"
)

func main() {
	c := httpClient.NewClient(
		client.ContentType("application/json"),
	)
	s := httpServer.NewServer(
		server.Address("localhost:8081"),
	)

	// New Service
	service := micro.NewService(
		micro.Name("go.micro.api.http"),
		micro.Version("latest"),
		micro.Registry(noop.NewRegistry()),
		micro.Client(c),
		micro.Server(s),
	)

	// ……
}
```
