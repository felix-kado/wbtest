package server

import (
	"embed"
	"html/template"
	"net/http"

	"wbstorage/internal/db"

	"github.com/go-chi/chi/v5"
)

//go:embed "templates/order.html"
var tmplFS embed.FS

type Server struct {
	db   *db.CachedClient
	tmpl *template.Template
}

func NewServer(dbWithCache *db.CachedClient) *Server {
	tmpl, _ := template.ParseFS(tmplFS, "templates/order.html")
	s := &Server{
		db:   dbWithCache,
		tmpl: tmpl,
	}
	return s
}

func NewRouter(s *Server) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/{orderUID}", s.handleGetOrder())
	return router
}

func (s *Server) handleGetOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderUID := chi.URLParam(r, "orderUID")
		order, err := s.db.SelectOrder(r.Context(), orderUID)
		if err != nil {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		if err := s.tmpl.Execute(w, order); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}
