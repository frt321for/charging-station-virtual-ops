package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charging-ops/backend/internal/app"
	"charging-ops/backend/internal/common/config"
)

func main() {
	host := flag.String("host", "", "HTTP listen host")
	port := flag.String("port", "", "HTTP listen port")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	if *host != "" {
		cfg.HTTPHost = *host
	}
	if *port != "" {
		cfg.HTTPPort = *port
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := app.NewServer(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed to initialize server", "module", "bootstrap", "traceId", "startup", "error", err)
		os.Exit(1)
	}
	defer server.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	logger.Info("api server started", "module", "bootstrap", "traceId", "startup", "host", cfg.HTTPHost, "port", cfg.HTTPPort)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown failed", "module", "bootstrap", "traceId", "shutdown", "error", err)
			os.Exit(1)
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "module", "bootstrap", "traceId", "runtime", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("api server stopped", "module", "bootstrap", "traceId", "shutdown")
}
