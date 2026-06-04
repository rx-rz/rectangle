package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	AppEnv    string
	Port      int
	DbUrl     string
	OtpSecret string
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
		AppEnv:    getString("APP_ENV", "development"),
		Port:      port,
		DbUrl:     getDatabaseURL(),
		OtpSecret: getString("OTP_HASH_SECRET", ""),
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

	return nil
}
