package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	AppEnv             string
	Port               int
	DbUrl              string
	CORSAllowedOrigins string
	OtpSecret          string
	MailerApiKey       string
	MailerFrom         string
	MailerDashboardURL string
	MailerDocsURL      string
	MailerSupportURL   string
	WebAppURL          string
	GoogleClientID     string
	GoogleRedirectURI  string
	GoogleClientSecret string
}

func getString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
	}
	return parsed, nil
}

func getDatabaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return getString("DB_URL", "")
}

func Load() (Config, error) {
	_ = godotenv.Load()
	port, err := getInt("API_PORT", 4001)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:             getString("APP_ENV", "development"),
		Port:               port,
		DbUrl:              getDatabaseURL(),
		CORSAllowedOrigins: getString("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		OtpSecret:          getString("OTP_HASH_SECRET", ""),
		MailerApiKey:       getString("MAILER_API_KEY", ""),
		MailerFrom:         getString("MAILER_FROM", ""),
		MailerDashboardURL: getString("MAILER_DASHBOARD_URL", ""),
		MailerDocsURL:      getString("MAILER_DOCS_URL", ""),
		MailerSupportURL:   getString("MAILER_SUPPORT_URL", ""),
		WebAppURL:          getString("WEB_APP_URL", "http://localhost:3000"),
		GoogleClientID:     getString("GOOGLE_CLIENT_ID", ""),
		GoogleRedirectURI:  getString("GOOGLE_REDIRECT_URI", ""),
		GoogleClientSecret: getString("GOOGLE_CLIENT_SECRET", ""),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.AppEnv != "development" && c.AppEnv != "test" && c.AppEnv != "production" {
		return fmt.Errorf("APP_ENV must be one of: development, test, production")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535")
	}

	if c.DbUrl == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	parsedURL, err := url.Parse(c.DbUrl)
	if err != nil {
		return fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL must use postgres:// or postgresql://")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("DATABASE_URL must include a host")
	}

	if c.OtpSecret == "" {
		return fmt.Errorf("OTP_HASH_SECRET is required")
	}

	if c.MailerApiKey == "" {
		return fmt.Errorf("MAILER_API_KEY is required")
	}

	if c.MailerFrom == "" {
		return fmt.Errorf("MAILER_FROM is required")
	}

	if c.WebAppURL == "" {
		return fmt.Errorf("WEB_APP_URL is required")
	}

	parsedWebAppURL, err := url.Parse(c.WebAppURL)
	if err != nil {
		return fmt.Errorf("WEB_APP_URL is invalid: %w", err)
	}

	if parsedWebAppURL.Scheme != "http" && parsedWebAppURL.Scheme != "https" {
		return fmt.Errorf("WEB_APP_URL must use http:// or https://")
	}

	if parsedWebAppURL.Host == "" {
		return fmt.Errorf("WEB_APP_URL must include a host")
	}

	if c.GoogleClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	if c.GoogleClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required")
	}

	if c.GoogleRedirectURI == "" {
		return fmt.Errorf("GOOGLE_REDIRECT_URI is required")
	}

	return nil
}
