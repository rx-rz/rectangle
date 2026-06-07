package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rx-rz/rectangle-api/internal/auth"
	"rx-rz/rectangle-api/internal/config"
	"rx-rz/rectangle-api/internal/db"
	"rx-rz/rectangle-api/internal/server"
	"rx-rz/rectangle-api/internal/user"
	"rx-rz/rectangle-api/platform/logger"
	"rx-rz/rectangle-api/platform/mail"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	appLogger := logger.New(cfg.AppEnv)
	database, err := db.Open(cfg.DbUrl)

	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	defer func(database *sqlx.DB) {
		err := database.Close()
		if err != nil {
			appLogger.Error("db closing failed", "error", err)
		}
	}(database)

	userRepo := user.NewRepository(database)
	authRepo := auth.NewRepository(database)
	mailer := mail.NewMailer(cfg)
	authService := auth.NewService(auth.ServiceOptions{
		UserRepository: userRepo,
		OTPRepository:  authRepo,
		Mailer:         mailer,
		Config:         cfg,
		Logger:         appLogger,
	})

	apiServer := server.New(server.Options{
		Port:        cfg.Port,
		AuthService: authService,
		Logger:      appLogger,
	})

	shutdownErr := make(chan error, 1)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		shutdownErr <- apiServer.Shutdown(ctx)
	}()

	appLogger.Info("api server started", "addr", apiServer.Addr, "env", cfg.AppEnv)
	err = apiServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		appLogger.Error("api server failed", "error", err)
		os.Exit(1)
	}

	if err := <-shutdownErr; err != nil {
		appLogger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}

	appLogger.Info("api server stopped")
}
