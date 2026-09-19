package config

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains only process configuration. Secrets never have a default
// outside development, making an accidental production deployment safer.
type Config struct {
	Port           string
	MongoURI       string
	MongoDatabase  string
	RedisURL       string
	JWTSecret      []byte
	JWTTTL         time.Duration
	FrontendOrigin string
	CookieSecure   bool
	CookieSameSite http.SameSite
}

func Load() (Config, error) {
	ttlHours, err := strconv.Atoi(valueOr("JWT_TTL_HOURS", "168"))
	if err != nil || ttlHours < 1 || ttlHours > 24*90 {
		return Config{}, fmt.Errorf("JWT_TTL_HOURS must be between 1 and %d", 24*90)
	}

	cookieSameSite, err := parseSameSite(valueOr("COOKIE_SAME_SITE", "lax"))
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:           valueOr("PORT", "8080"),
		MongoURI:       valueOr("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDatabase:  valueOr("MONGODB_DATABASE", "pulsepoll"),
		RedisURL:       valueOr("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:      []byte(os.Getenv("JWT_SECRET")),
		JWTTTL:         time.Duration(ttlHours) * time.Hour,
		FrontendOrigin: valueOr("FRONTEND_ORIGIN", "http://localhost:5173"),
		CookieSecure:   strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
		CookieSameSite: cookieSameSite,
	}

	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if _, err := url.ParseRequestURI(cfg.MongoURI); err != nil {
		return Config{}, fmt.Errorf("invalid MONGODB_URI: %w", err)
	}
	if _, err := url.Parse(cfg.RedisURL); err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_URL: %w", err)
	}
	return cfg, nil
}

func parseSameSite(value string) (http.SameSite, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	default:
		return 0, fmt.Errorf("COOKIE_SAME_SITE must be lax, strict, or none")
	}
}

func valueOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
