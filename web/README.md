# Web Service

在go-micro/web基础上为`Handle()`和`HandleFunc()`方法增加了`endpoints ...*api.Endpoint`选项，并通过自定义的`api.Endpoint`来指定路由，
实现`rpc`和`http`服务统一使用默认的`handler=meta`网关。

## 示例

**网关**
```bash
$ micro api
```

**[example](/web/example/main.go)**

```bash
$ go run main.go
```

浏览器:http://localhost:8080/console

```bash
curl -HHost:hbchen.com 'http://localhost:8080/console/path'
request success, path: /console/path
```
 
