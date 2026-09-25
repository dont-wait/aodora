package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/your-org/aodora/server/internal/application/health"
	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	httpapi "github.com/your-org/aodora/server/internal/interfaces/http"
	"github.com/your-org/aodora/server/internal/logger"
)

func main() {
	cfg, err := config.LoadAllConfig()
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		var configErrors config.ConfigErrors
		if errors.As(err, &configErrors) {
			for _, configError := range configErrors {
				log.Error().Err(configError.Err).Str("config", configError.Name).Msg("load configuration")
			}
		} else {
			log.Error().Err(err).Msg("load configuration")
		}
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.OpenMongo(ctx, database.Config{
		URI: cfg.Mongo.URI,
		DB:  cfg.Mongo.DB,
	})
	if err != nil {
		log.Error().Err(err).Msg("open database")
		os.Exit(1)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Close(closeCtx); err != nil {
			log.Error().Err(err).Msg("close mongodb")
		}
	}()

	service := health.NewService(db, *log)
	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           httpapi.NewRouter(service, *log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.HTTP.Addr).Msg("http server started")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("http server stopped unexpectedly")
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("shutdown http server")
	}
	log.Info().Msg("http server stopped")
}
