package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"rx-rz/rectangle-api/internal/auth"
	"rx-rz/rectangle-api/internal/helpers"
	"time"
)

type Options struct {
	Port        int
	AuthService *auth.AuthService
	Logger      *slog.Logger
}

type routeOptions struct {
	Mux         *http.ServeMux
	AuthHandler *auth.Handler
}

func New(opts Options) *http.Server {
	mux := http.NewServeMux()

	authHandler := auth.NewHandler(auth.HandlerOptions{
		Service: opts.AuthService,
		Logger:  opts.Logger,
	})

	registerRoutes(routeOptions{
		Mux:         mux,
		AuthHandler: authHandler,
	})

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", opts.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func registerRoutes(opts routeOptions) {
	opts.Mux.HandleFunc("GET /health", healthHandler)

	//auth
	opts.Mux.HandleFunc("POST /auth/signup/email", opts.AuthHandler.SignupWithEmail)
	opts.Mux.HandleFunc("POST /auth/otp/send", opts.AuthHandler.SendOTP)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	if err := helpers.WriteJSON(w, http.StatusOK, helpers.Envelope{
		"status": "ok",
	}, nil); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
