package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gotunnel-pro/internal/config"
	"gotunnel-pro/internal/crypto"
	"gotunnel-pro/internal/health"
	"gotunnel-pro/internal/logging"
	"gotunnel-pro/internal/metrics"
	"gotunnel-pro/internal/tunnel"
)

var (
	logger *logging.Logger
	cfg    *config.ServerConfig
)

func main() {
	// Initialize configuration
	configPath := flag.String("config", "config/server.yaml", "Path to configuration file")
	flag.Parse()

	var err error
	cfg, err = config.LoadServerConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger = logging.NewLogger("gotunnel-server", cfg.Environment, parseLogLevel(cfg.LogLevel))
	ctx := context.Background()

	// Initialize health service
	healthService := health.NewHealthService()
	healthService.SetReady(true)

	// Load mTLS configuration
	tlsConfig, err := crypto.LoadMTLSConfig(
		cfg.Server.CertFile,
		cfg.Server.KeyFile,
		cfg.Server.CAFile,
		true,
	)
	if err != nil {
		logger.Fatal(ctx, "Failed to load mTLS configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create tunnel server
	server := tunnel.NewServer(&tunnel.ServerConfig{
		ListenAddr: cfg.Server.ListenAddr,
		TLSConfig:  tlsConfig,
		Logger:     logger,
		Health:     healthService,
	})

	// Setup HTTP server for metrics and health checks
	httpServer := setupHTTPServer(healthService)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(2)

	// Start tunnel server
	go func() {
		defer wg.Done()
		logger.Info(ctx, "Starting tunnel server", map[string]interface{}{
			"address": cfg.Server.ListenAddr,
		})
		if err := server.Start(); err != nil {
			logger.Error(ctx, "Tunnel server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Start HTTP server
	go func() {
		defer wg.Done()
		logger.Info(ctx, "Starting HTTP server", map[string]interface{}{
			"address": cfg.Server.MetricsAddr,
		})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "HTTP server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info(ctx, "Shutdown signal received, initiating graceful shutdown", nil)

	// Initiate graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Mark as shutting down
	healthService.SetShuttingDown(true)
	healthService.SetReady(false)

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "HTTP server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Shutdown tunnel server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Tunnel server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Wait for all goroutines to finish
	wg.Wait()
	logger.Info(ctx, "Graceful shutdown completed", nil)
}

func setupHTTPServer(healthService *health.HealthService) *http.Server {
	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		result := healthService.Check(r.Context())
		status := http.StatusOK

		if result["status"] == "unhealthy" || healthService.IsShuttingDown() {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(result)
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if healthService.IsShuttingDown() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("shutting down"))
			return
		}

		if !healthService.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// Metrics endpoint
	mux.Handle("/metrics", metrics.MetricsHandler())

	return &http.Server{
		Addr:    cfg.Server.MetricsAddr,
		Handler: mux,
	}
}

func parseLogLevel(level string) logging.Level {
	switch level {
	case "debug":
		return logging.DEBUG
	case "info":
		return logging.INFO
	case "warn":
		return logging.WARN
	case "error":
		return logging.ERROR
	default:
		return logging.INFO
	}
}
