package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig.String())
		cancel()
	}()

	logger.Info("Server starting", "version", "0.1.0")

	// TODO: Initialize and start server
	// - Load configuration (koanf)
	// - Setup database connections (PostgreSQL + Redis)
	// - Initialize layers: Storage → Domain → Service → Authz → Gateway
	// - Start HTTP/gRPC server

	<-ctx.Done()
	logger.Info("Server stopped")
}
