// Package web provides web based micro services
package web

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/v3/api"
)

// Service is a web service with service discovery built in
type Service interface {
	Client() *http.Client
	Init(opts ...Option) error
	Options() Options
	Handle(pattern string, handler http.Handler, endpoints ...*api.Endpoint)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), endpoints ...*api.Endpoint)
	Run() error
}

type Option func(o *Options)

var (
	// For serving
	DefaultName    = "go-web"
	DefaultVersion = "latest"
	DefaultId      = uuid.New().String()
	DefaultAddress = ":0"

	// for registration
	DefaultRegisterTTL      = time.Minute
	DefaultRegisterInterval = time.Second * 30

	// static directory
	DefaultStaticDir = "html"
)

// NewService returns a new web.Service
func NewService(opts ...Option) Service {
	return newService(opts...)
}
