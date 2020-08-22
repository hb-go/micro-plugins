module github.com/hb-go/micro-plugins/v3

go 1.13

require (
	github.com/casbin/casbin/v2 v2.1.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.0.0
	github.com/micro/go-micro/v3 v3.0.0-beta.0.20200820102804-329bc2f2655b
	github.com/micro/micro/v2 v2.0.0
	github.com/micro/micro/v3 v3.0.0-beta
	github.com/rs/cors v1.7.0
)

replace github.com/micro/micro/v3 v3.0.0-beta => github.com/hb-chen/micro/v3 v3.0.0-20200820165338-b9e3f7223f04
