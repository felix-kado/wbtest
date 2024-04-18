package server

import (
	"html/template"
	"net/http"

	"wbstorage/internal/db"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router *chi.Mux
	db     *db.CachedClient
}

func NewServer(dbWithCache *db.CachedClient) *Server {
	s := &Server{
		db:     dbWithCache,
		router: chi.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Get("/{orderUID}", s.handleGetOrder())
}

func (s *Server) handleGetOrder() http.HandlerFunc {
	tmpl := template.Must(template.New("order").Parse(orderTemplate))
	return func(w http.ResponseWriter, r *http.Request) {
		orderUID := chi.URLParam(r, "orderUID")
		order, err := s.db.GetOrderFromCache(orderUID)
		if err != nil {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		if err := tmpl.Execute(w, order); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

const orderTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Order Details</title>
</head>
<body>
    <h1>Order Details</h1>
    <p><strong>Order UID:</strong> {{.OrderUID}}</p>
    <p><strong>Track Number:</strong> {{.TrackNumber}}</p>
    <p><strong>Customer ID:</strong> {{.CustomerID}}</p>
    <p><strong>Locale:</strong> {{.Locale}}</p>
    <h2>Delivery Info</h2>
    {{if .Delivery}}
    <p><strong>Name:</strong> {{.Delivery.Name}}</p>
    <p><strong>Phone:</strong> {{.Delivery.Phone}}</p>
    <p><strong>Address:</strong> {{.Delivery.Address}}</p>
    {{else}}
    <p>Delivery information is not available.</p>
    {{end}}
    <h2>Items</h2>
    {{if .Items}}
    {{range .Items}}
        <div>
            <p><strong>Item Name:</strong> {{.Name}}</p>
            <p><strong>Price:</strong> {{.Price}}</p>
            <p><strong>Quantity:</strong> {{.Quantity}}</p>
            <p><strong>Total Price:</strong> {{.TotalPrice}}</p>
        </div>
    {{end}}
    {{else}}
    <p>No items available for this order.</p>
    {{end}}
</body>
</html>
`

func (s *Server) Start(address string) error {
	return http.ListenAndServe(address, s.router)
}
