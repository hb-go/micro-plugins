## go-mciro plugins

- `JWT`+`Casbin`权限验证
	- micro/auth
- `CORS`跨域
	- micro/cors
- 流量染色
	- micro/chain
	- wrapper/select/chain
	
- Istio Http 插件，将go-micro服务加入service mesh，Client、Server不需要Registry、Selector、Transport等
	- client/istio_http
	- server/istio_http	

### TODO
- Istio Http
    - FQDN Server/Service address定义
    - 支持stream
- Istio gRPC
    - Istio部署测试
    - API service支持server.HandlerOption定义api.Endpoint
    
### Attention
- http broker依赖registry，所以不可用
       
### [go-micro istio示例](https://github.com/hb-go/micro/tree/master/istio)
