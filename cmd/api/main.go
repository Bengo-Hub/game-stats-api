package main

import (
"fmt"
"net/http"
"os"

"github.com/bengobox/game-stats-api/internal/config"
"github.com/go-chi/chi/v5"
"github.com/go-chi/chi/v5/middleware"
"github.com/rs/cors"
)

func main() {
r := chi.NewRouter()

// Middleware
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(cors.Default().Handler)

// Health check
r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
w.Write([]byte("OK"))
})

port := os.Getenv("PORT")
if port == "" {
port = "8080"
}

fmt.Printf("Server starting on port %s\n", port)
http.ListenAndServe(":"+port, r)
}
