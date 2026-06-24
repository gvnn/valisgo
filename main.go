package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Router *chi.Mux
}

func CreateNewServer() *Server {
	return &Server{
		Router: chi.NewRouter(),
	}
}

func (s *Server) MountHandlers() {
	s.Router.Use(middleware.Logger)
	s.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
}

func main() {
	s := CreateNewServer()
	s.MountHandlers()

	if err := http.ListenAndServe(":3000", s.Router); err != nil {
		log.Fatal(err)
	}
}
