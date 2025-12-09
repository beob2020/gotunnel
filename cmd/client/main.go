package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gotunnel-pro/internal/config"
	"gotunnel-pro/internal/crypto"
	"gotunnel-pro/internal/logging"
	"gotunnel-pro/internal/tunnel"
)

func main() {
	// Initialize configuration
	configPath := os.Getenv("GOTUNNEL_CONFIG")
	if configPath == "" {
		configPath = "config/client.yaml"
	}

	cfg, err := config.LoadClientConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := logging.NewLogger("gotunnel-client", cfg.Environment, parseLogLevel(cfg.LogLevel))
	ctx := context.Background()

	// Load mTLS configuration
	tlsConfig, err := crypto.LoadMTLSConfig(
		cfg.Client.CertFile,
		cfg.Client.KeyFile,
		cfg.Client.CAFile,
		false,
	)
	if err != nil {
		logger.Fatal(ctx, "Failed to load mTLS configuration", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create tunnel client
	client := tunnel.NewClient(&tunnel.ClientConfig{
		ServerAddr: cfg.Server.Address,
		TLSConfig:  tlsConfig,
		Tunnels:    cfg.Tunnels,
		Logger:     logger,
		Reconnect: tunnel.ReconnectConfig{
			Enabled:     true,
			MaxAttempts: 10,
			Interval:    5 * time.Second,
			Backoff:     2.0,
			MaxBackoff:  60 * time.Second,
		},
	})

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	// Start client
	go func() {
		defer wg.Done()
		logger.Info(ctx, "Starting tunnel client", map[string]interface{}{
			"server": cfg.Server.Address,
		})
		if err := client.Start(); err != nil {
			logger.Error(ctx, "Client error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info(ctx, "Shutdown signal received", nil)

	// Shutdown client
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Client shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	wg.Wait()
	logger.Info(ctx, "Client stopped gracefully", nil)
}

func parseLogLevel(level string) logging.Level {
	// Same as server implementation
	return logging.INFO
}
