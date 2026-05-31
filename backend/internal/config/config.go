package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Port                 string
	PostgresURL          string
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Info().Msg("no .env file, reading environment variables")
	}
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		PostgresURL:          mustEnv("POSTGRES_URL"),
		JWTSecret:            mustEnv("JWT_SECRET"),
		AccessTokenDuration:  getDuration("ACCESS_TOKEN_DURATION", 15*time.Minute),
		RefreshTokenDuration: getDuration("REFRESH_TOKEN_DURATION", 30*24*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Fatal().Str("key", key).Str("value", v).Msg("invalid duration")
	}
	return d
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatal().Str("key", key).Msg("required environment variable is not set")
	}
	return v
}
