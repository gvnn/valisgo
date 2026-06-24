package main

import (
	"log"
	"net/http"

	"valisgo/internal/registry"
	"valisgo/internal/registry/pypi"
	"valisgo/internal/server"
)

func main() {

	srv := server.NewServer()

	srv.RegisterProtocol("pypi", &pypi.PyPIProtocol{})
	srv.RegisterRepository(registry.Repository{Name: "my-pypi", Format: "pypi"})

	r := srv.SetupRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
