package main

import (
	"fmt"
	"net/http"

	"github.com/hb-go/micro-plugins/web"
	"github.com/micro/go-micro/api"
	"github.com/micro/go-micro/util/log"
)

func main() {
	// create new web service
	service := web.NewService(
		web.Name("go.micro.api.console.web"),
		web.Version("latest"),
	)

	// initialise service
	if err := service.Init(); err != nil {
		log.Fatal(err)
	}

	service.Handle("/console/", &handler{}, &api.Endpoint{
		Name:    "console",
		Host:    []string{"localhost:8080"},
		Path:    []string{"^/console"},
		Method:  []string{"POST", "GET", "DELETE"},
		Handler: "proxy",
	})

	// run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

type handler struct {
}

func (*handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("request success, path: %v", r.URL.Path)))
}
