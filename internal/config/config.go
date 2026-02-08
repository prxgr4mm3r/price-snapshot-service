package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Exchange ExchangeConfig
	Poller   PollerConfig
	Logging  LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ExchangeConfig holds Binance API configuration
type ExchangeConfig struct {
	BaseURL      string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
}

// PollerConfig holds price polling configuration
type PollerConfig struct {
	Interval      time.Duration
	RetentionDays int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnvString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/snapshots?sslmode=disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			ConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Exchange: ExchangeConfig{
			BaseURL:      getEnvString("EXCHANGE_BASE_URL", "https://api.binance.com"),
			Timeout:      getEnvDuration("EXCHANGE_TIMEOUT", 10*time.Second),
			MaxRetries:   getEnvInt("EXCHANGE_MAX_RETRIES", 3),
			RetryBackoff: getEnvDuration("EXCHANGE_RETRY_BACKOFF", 100*time.Millisecond),
		},
		Poller: PollerConfig{
			Interval:      getEnvDuration("POLLER_INTERVAL", 30*time.Second),
			RetentionDays: getEnvInt("POLLER_RETENTION_DAYS", 30),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
		},
	}, nil
}

// Validate ensures configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.Poller.Interval < 5*time.Second {
		return fmt.Errorf("poller interval must be at least 5 seconds")
	}

	if c.Poller.Interval > 24*time.Hour {
		return fmt.Errorf("poller interval must be less than 24 hours")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	validLogFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", c.Logging.Format)
	}

	return nil
}

// Helper functions
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
