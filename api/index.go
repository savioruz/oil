// Package handler is for handling serverless function requests.
package handler

import (
	"net/http"
	"oil/config"
	"oil/di"
	"oil/shared/logger"
)

// Handler is the entry point for handling incoming HTTP requests in a serverless environment.
func Handler(w http.ResponseWriter, r *http.Request) {
	r.RequestURI = r.URL.String()

	cfg := config.Get()

	logger.InitLogger()

	logger.SetLogLevel(cfg)

	handler := di.InitializeService()
	handler.ServeHTTP(w, r)
}
