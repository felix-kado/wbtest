package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Пока можно не смотреть я не делал сервер ещё

func NewRouter() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})
	return router
}

func StartServer(port string) {
	router := NewRouter()
	http.ListenAndServe(":"+port, router)
}
