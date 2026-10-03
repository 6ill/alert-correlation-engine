package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	RedisAddr           string
	Port                string
	TCOPWebhookToken    string
	TCOPWebhookUsername string // empty disables the BasicAuth layer
	TCOPWebhookPassword string
}

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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
