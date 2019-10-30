package main

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// HTTPWorker implements the svc interface for livecycle management
type HTTPWorker struct {
	port    int
	version string
	handler http.HandlerFunc
	server  *http.Server
}

// Option represents an configuration alternative for HTTPWorker
type Option func(w *HTTPWorker)

// WithPort starts the server on port
func WithPort(port int) Option {
	return func(w *HTTPWorker) {
		w.port = port
	}
}

// WithHandlerFunc adds a handler func so that the response isn't "not here yet"
func WithHandlerFunc(f http.HandlerFunc) Option {
	return func(w *HTTPWorker) {
		w.handler = f
	}
}

// NewHTTPWorker yeilds a new http worker
func NewHTTPWorker(version string, ops ...Option) *HTTPWorker {
	worker := &HTTPWorker{
		port:    8080,
		version: version,
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte("not here yet"))
		},
	}
	for _, op := range ops {
		op(worker)
	}
	return worker
}

func (worker *HTTPWorker) Init(*zap.Logger) error {
	worker.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", worker.port),
		Handler: worker.handler,
	}
	return nil
}

func (worker *HTTPWorker) Terminate() error {
	return worker.server.Shutdown(context.Background())
}

func (worker *HTTPWorker) Run() error {
	if err := worker.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
