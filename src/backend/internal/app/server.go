package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"charging-ops/backend/internal/common/config"
	"charging-ops/backend/internal/common/health"
	"charging-ops/backend/internal/domain/command"
	"charging-ops/backend/internal/domain/gateway"
	"charging-ops/backend/internal/domain/session"
	"charging-ops/backend/internal/domain/simcontrol"
	"charging-ops/backend/internal/domain/site"
	"charging-ops/backend/internal/platform/cache"
	"charging-ops/backend/internal/platform/database"
	"charging-ops/backend/internal/platform/events"
)

// Server owns the HTTP server and shared platform clients.
type Server struct {
	httpServer *http.Server
	database   *database.Client
	cache      *cache.Client
	events     *events.Client
	simulator  *simcontrol.Service
}

// NewServer creates an API server with all required infrastructure clients.
func NewServer(ctx context.Context, cfg config.Config, logger *slog.Logger) (*Server, error) {
	dbClient, err := database.Connect(ctx, cfg.DatabaseURL())
	if err != nil {
		return nil, err
	}

	cacheClient, err := cache.Connect(ctx, cache.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
		DB:       cfg.ValkeyDB,
	})
	if err != nil {
		dbClient.Close()
		return nil, err
	}

	eventClient, err := events.Connect(cfg.NATSURL)
	if err != nil {
		cacheClient.Close()
		dbClient.Close()
		return nil, err
	}

	healthHandler := health.NewHandler(health.Dependencies{
		Database: dbClient,
		Cache:    cacheClient,
		Events:   eventClient,
	})
	siteHandler := site.NewHandler(site.NewService(site.NewRepository(dbClient)))
	sessionHandler := session.NewHandler(session.NewService(session.NewRepository(dbClient)))
	commandHandler := command.NewHandler(command.NewService(command.NewRepository(dbClient)))
	gatewayHandler := gateway.NewHandler(gateway.NewService(gateway.NewRepository(dbClient)))
	simulatorService := simcontrol.NewService(localAPIBase(cfg.HTTPHost, cfg.HTTPPort), logger)
	simulatorHandler := simcontrol.NewHandler(simulatorService)

	mux := http.NewServeMux()
	registerRoutes(mux, healthHandler, siteHandler, sessionHandler, commandHandler, gatewayHandler, simulatorHandler)

	httpServer := &http.Server{
		Addr:              cfg.HTTPHost + ":" + cfg.HTTPPort,
		Handler:           traceMiddleware(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		database:   dbClient,
		cache:      cacheClient,
		events:     eventClient,
		simulator:  simulatorService,
	}, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Close releases infrastructure clients.
func (s *Server) Close() {
	s.simulator.Stop()
	s.events.Close()
	s.cache.Close()
	s.database.Close()
}

func localAPIBase(host string, port string) string {
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}
