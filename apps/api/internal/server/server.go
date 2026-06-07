package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"rx-rz/rectangle-api/internal/auth"
	"rx-rz/rectangle-api/internal/helpers"
	"strings"
	"time"
)

type Options struct {
	Port               int
	AuthService        *auth.AuthService
	Logger             *slog.Logger
	CORSAllowedOrigins string
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
		Handler:           corsMiddleware(mux, opts.CORSAllowedOrigins),
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
	opts.Mux.HandleFunc("POST /auth/login/email", opts.AuthHandler.LoginWithEmail)
	opts.Mux.HandleFunc("POST /auth/otp/send", opts.AuthHandler.SendOTP)
	opts.Mux.HandleFunc("POST /auth/otp/verify", opts.AuthHandler.VerifyOTP)
}

func corsMiddleware(next http.Handler, allowedOrigins string) http.Handler {
	origins := map[string]struct{}{}
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := origins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	if err := helpers.WriteData(w, http.StatusOK, helpers.Envelope{
		"status": "ok",
	}, nil); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}
