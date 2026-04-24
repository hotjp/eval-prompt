// Package gateway implements L5-Gateway layer: TLS termination, protocol adaptation,
// middleware (Recover/Metrics/CORS), and request routing.
package gateway

import (
	"context"
)

// Gateway handles HTTP/gRPC protocol adaptation and middleware.
type Gateway struct {
	// TODO: Add dependencies (authz, service layers)
}

// New creates a new Gateway instance.
func New() *Gateway {
	return &Gateway{}
}

// Start starts the gateway server.
func (g *Gateway) Start(ctx context.Context) error {
	// TODO: Implement server startup
	return nil
}

// Stop gracefully stops the gateway server.
func (g *Gateway) Stop(ctx context.Context) error {
	// TODO: Implement graceful shutdown
	return nil
}
