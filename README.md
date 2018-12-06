## go-mciro plugins

将go-micro服务加入service mesh，Client、Server不需要Registry、Selector、Transport等
- client/istio_http
- server/istio_http

### TODO
- http
    - FQDN Server/Service address定义
    - 支持stream
- gRPC
    - Istio部署测试
    - API service支持server.HandlerOption定义api.Endpoint
    
### Attention
- http broker依赖registry，所以不可用
       
### [go-micro istio示例](https://github.com/hb-go/micro/tree/master/istio)
