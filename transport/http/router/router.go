// Package router provides HTTP routing utilities.
package router

import (
	"github.com/savioruz/oil/internal/handlers/gallery"
	"github.com/savioruz/oil/internal/handlers/todo"
	"github.com/savioruz/oil/internal/handlers/userprofile"

	"github.com/go-chi/chi/v5"
)

type ModuleHandlers struct {
	Todo        todo.Handler
	Gallery     gallery.Handler
	Userprofile userprofile.Handler
}

type Router struct {
	ModuleHandlers ModuleHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/api", func(routerGroup chi.Router) {
		r.ModuleHandlers.Todo.Router(routerGroup)
		r.ModuleHandlers.Gallery.Router(routerGroup)
		r.ModuleHandlers.Userprofile.Router(routerGroup)
	})
}

func New(moduleHandlers ModuleHandlers) Router {
	return Router{
		ModuleHandlers: moduleHandlers,
	}
}
