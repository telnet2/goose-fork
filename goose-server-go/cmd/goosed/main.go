package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/block/goose-server-go/internal/config"
	"github.com/block/goose-server-go/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 0, "Server port (overrides GOOSE_PORT env var)")
	flag.Parse()

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Override port if specified via flag
	if *port != 0 {
		cfg.Port = *port
	}

	// Validate configuration
	if cfg.SecretKey == "" {
		log.Fatal().Msg("GOOSE_SERVER__SECRET_KEY environment variable is required")
	}

	log.Info().
		Int("port", cfg.Port).
		Str("data_dir", cfg.DataDir).
		Msg("Starting goose server")

	// Create and start server
	srv := server.New(cfg)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info().Msg("Shutting down server...")
		if err := srv.Shutdown(); err != nil {
			log.Error().Err(err).Msg("Server shutdown error")
		}
	}()

	// Start server (blocking)
	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
