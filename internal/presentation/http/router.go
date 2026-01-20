package http

import (
	"net/http"

	"github.com/bengobox/game-stats-api/internal/config"
	"github.com/bengobox/game-stats-api/internal/presentation/http/handlers"
	"github.com/bengobox/game-stats-api/internal/presentation/http/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/bengobox/game-stats-api/docs"
)

type RouterOptions struct {
	Config      *config.Config
	AuthHandler *handlers.AuthHandler
}

func NewRouter(opts RouterOptions) chi.Router {
	r := chi.NewRouter()

	// Base middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Default().Handler)

	// Root redirect to Swagger
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", opts.AuthHandler.Login)
			r.Post("/refresh", opts.AuthHandler.Refresh)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(opts.Config.JWTSecret))

			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				// Return user info from context
			})

			// Admin only routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RoleMiddleware("admin"))
				// ...
			})
		})
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The url pointing to API definition
	))

	return r
}

// ChiRouter alias for return type convenience if needed
func GetRouter(opts RouterOptions) chi.Router {
	return NewRouter(opts)
}
