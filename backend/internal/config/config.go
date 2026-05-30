package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type StorageMode string

const (
	StoragePostgres StorageMode = "postgres"
	StorageMemory   StorageMode = "memory"
)

type Config struct {
	StorageMode       StorageMode
	POSTGRES_DB       string
	POSTGRES_USER     string
	POSTGRES_PASSWORD string
	POSTGRES_HOST     string
	POSTGRES_PORT     string
	PORT              string
	JWTSecret         string
	JWTExpiration     time.Duration
	CookieSecure      bool
	CookieDomain      string
}

func Load() (*Config, error) {
	mode := StorageMode(strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_MODE"))))
	if mode == "" {
		mode = StoragePostgres
	}
	if mode != StoragePostgres && mode != StorageMemory {
		return nil, fmt.Errorf("STORAGE_MODE must be postgres or memory, got %q", mode)
	}

	cfg := &Config{
		StorageMode: mode,
		POSTGRES_DB: os.Getenv("POSTGRES_DB"),
		POSTGRES_USER: os.Getenv("POSTGRES_USER"),
		POSTGRES_PASSWORD: os.Getenv("POSTGRES_PASSWORD"),
		POSTGRES_HOST:     os.Getenv("POSTGRES_HOST"),
		POSTGRES_PORT:     os.Getenv("POSTGRES_PORT"),
		PORT:              os.Getenv("PORT"),
	}

	if cfg.StorageMode == StoragePostgres {
		if cfg.POSTGRES_DB == "" {
			return nil, fmt.Errorf("переменная окружения POSTGRES_DB не задана")
		}
		if cfg.POSTGRES_USER == "" {
			return nil, fmt.Errorf("переменная окружения POSTGRES_USER не задана")
		}
		if cfg.POSTGRES_PASSWORD == "" {
			return nil, fmt.Errorf("переменная окружения POSTGRES_PASSWORD не задана")
		}
	}

	if cfg.POSTGRES_HOST == "" {
		cfg.POSTGRES_HOST = "localhost"
	}
	if cfg.POSTGRES_PORT == "" {
		cfg.POSTGRES_PORT = "5432"
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("переменная окружения JWT_SECRET не задана")
	}

	cfg.JWTExpiration = 7 * 24 * time.Hour
	if ttl := os.Getenv("JWT_TTL_HOURS"); ttl != "" {
		hours, err := strconv.Atoi(ttl)
		if err != nil || hours <= 0 {
			return nil, fmt.Errorf("JWT_TTL_HOURS must be a positive integer")
		}
		cfg.JWTExpiration = time.Duration(hours) * time.Hour
	}

	cfg.CookieSecure = strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
	cfg.CookieDomain = os.Getenv("COOKIE_DOMAIN")

	return cfg, nil
}
