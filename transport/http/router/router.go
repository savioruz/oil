// Package router provides HTTP routing utilities.
package router

import (
	"oil/internal/handlers/auth"
	"oil/internal/handlers/gallery"
	"oil/internal/handlers/todo"

	"github.com/go-chi/chi/v5"
)

// DomainHandlers holds all the domain handlers for the application.
type DomainHandlers struct {
	Todo    todo.Handler
	Auth    auth.Handler
	Gallery gallery.Handler
}

// Router holds the domain handlers for route configuration.
type Router struct {
	DomainHandlers DomainHandlers
}

// SetupRoutes configures all routes for the application.
func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/api", func(routerGroup chi.Router) {
		r.DomainHandlers.Todo.Router(routerGroup)
		r.DomainHandlers.Auth.Router(routerGroup)
		r.DomainHandlers.Gallery.Router(routerGroup)
	})
}

// New creates a new Router instance.
func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
