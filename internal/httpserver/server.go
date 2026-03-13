package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/servereye/servereyebot/internal/logger"
)

// Server represents HTTP server for health checks
type HttpServer struct {
	server *http.Server
	logger logger.Logger
}

// New creates a new HTTP server
func New(port int, log logger.Logger) *HttpServer {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	// Ready check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &HttpServer{
		server: server,
		logger: log,
	}
}

// Start starts the HTTP server
func (s *HttpServer) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP server", "port", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server gracefully
func (s *HttpServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}
