package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {})
	r.Post("/users", func(w http.ResponseWriter, r *http.Request) {})
	_ = http.ListenAndServe(":8080", r)
}
