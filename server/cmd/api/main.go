package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authapp "github.com/your-org/aodora/server/internal/application/auth"
	cartapp "github.com/your-org/aodora/server/internal/application/cart"
	"github.com/your-org/aodora/server/internal/application/catalog"
	admin "github.com/your-org/aodora/server/internal/application/catalogadmin"
	customerapp "github.com/your-org/aodora/server/internal/application/customer"
	"github.com/your-org/aodora/server/internal/application/health"
	orderapp "github.com/your-org/aodora/server/internal/application/order"
	reviewapp "github.com/your-org/aodora/server/internal/application/review"
	"github.com/your-org/aodora/server/internal/config"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	catalogMongo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
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
	catalogService := catalog.NewService(catalogMongo.NewCatalogRepository(db.DB))
	adminService := admin.NewService(catalogMongo.NewAdminCatalogRepository(db.DB))
	authService := authapp.NewService(catalogMongo.NewAuthRepository(db.DB), cfg.Auth.SessionTTL)
	customerService := customerapp.NewService(catalogMongo.NewCustomerRepository(db.DB))
	cartService := cartapp.NewService(catalogMongo.NewCartRepository(db.DB))

	var quoteSigner *domainorder.QuoteSigner
	if cfg.Order.QuoteSecret != nil {
		var err error
		quoteSigner, err = domainorder.NewQuoteSigner(cfg.Order.QuoteSecret)
		if err != nil {
			log.Error().Err(err).Msg("create quote signer")
			os.Exit(1)
		}
	} else {
		log.Warn().Msg("ORDER_QUOTE_SECRET is not configured; order quoting and creation will be unavailable")
	}
	if cfg.Order.CODShippingFee == nil {
		log.Warn().Msg("ORDER_COD_SHIPPING_FEE is not configured; COD orders will be unavailable")
	}

	orderRepo := catalogMongo.NewOrderRepository(db.Client, db.DB)
	orderAdminService := orderapp.NewAdminService(orderRepo)
	orderService := orderapp.NewService(orderRepo, orderRepo, quoteSigner, cfg.Order.QuoteTTL, cfg.Order.CODShippingFee, nil, nil)
	reviewService := reviewapp.NewService(catalogMongo.NewReviewRepository(db.DB), orderService)

	authGuard := httpapi.NewAuthGuard(authService, httpapi.AuthCookieConfig{
		Name: cfg.Auth.CookieName, Secure: cfg.Auth.CookieSecure, TTL: cfg.Auth.SessionTTL, AllowedOrigin: cfg.Auth.AllowedOrigin,
	})
	server := &http.Server{
		Addr: cfg.HTTP.Addr,
		Handler: httpapi.NewRouter(service, *log,
			httpapi.WithAuth(authGuard),
			httpapi.WithCustomer(customerService, authGuard),
			httpapi.WithCart(cartService, authGuard),
			httpapi.WithCatalog(catalogService, *log),
			httpapi.WithAdminCatalog(adminService, authGuard.RequireAdmin, *log, os.Getenv("ADMIN_ALLOWED_ORIGIN")),
			httpapi.WithOrders(orderService, authGuard, *log),
			httpapi.WithReviews(reviewService, authGuard, catalogService, orderService, *log),
			httpapi.WithAdminOrders(orderAdminService, authGuard.RequireAdmin, *log, os.Getenv("ADMIN_ALLOWED_ORIGIN"))),
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
