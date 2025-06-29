package rest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Config struct {
	Port string `env:"SERVER_PORT"`
}

// CORS middleware для разрешения кросс-доменных запросов
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CreateServer(cfg Config, handler *Handler) *http.Server {
	r := mux.NewRouter()

	r.Use(corsMiddleware)

	// API маршруты
	r.HandleFunc("/order/{order_uid}", handler.GetOrder()).Methods("GET")

	// Статические файлы для веб-интерфейса
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("web")))

	server := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}
