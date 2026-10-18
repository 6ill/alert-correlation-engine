package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	RedisAddr           string
	Port                string
	TCOPWebhookToken    string
	TCOPWebhookUsername string // empty disables the BasicAuth layer
	TCOPWebhookPassword string
	AdminUsername       string
	AdminPassword       string
}

const minAdminPasswordLength = 16

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Not found .env, use default env variables")
	}
	return Config{
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		RedisAddr:           getEnv("REDIS_ADDR", "localhost:6379"),
		Port:                getEnv("PORT", "8080"),
		TCOPWebhookToken:    getEnv("TCOP_WEBHOOK_TOKEN", ""),
		TCOPWebhookUsername: getEnv("TCOP_WEBHOOK_USERNAME", ""),
		TCOPWebhookPassword: getEnv("TCOP_WEBHOOK_PASSWORD", ""),
		AdminUsername:       getEnv("API_ADMIN_USERNAME", ""),
		AdminPassword:       getEnv("API_ADMIN_PASSWORD", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Validate reports configuration that must never be left unset. The API
// refuses to start rather than run with its admin routes unprotected.
func (c Config) Validate() error {
	var problems []string
	if c.AdminUsername == "" {
		problems = append(problems, "API_ADMIN_USERNAME is required")
	}
	if len(c.AdminPassword) < minAdminPasswordLength {
		problems = append(problems, fmt.Sprintf("API_ADMIN_PASSWORD must be at least %d characters", minAdminPasswordLength))
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
