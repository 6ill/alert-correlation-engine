package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/6ill/alert-correlation-engine/internal/config"
	"github.com/6ill/alert-correlation-engine/internal/handler"
	"github.com/6ill/alert-correlation-engine/internal/httpserver"
	"github.com/6ill/alert-correlation-engine/internal/repository"
	"github.com/6ill/alert-correlation-engine/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	alertRepo := repository.NewAlertRepository(pool)
	incidentRepo := repository.NewIncidentRepository(pool)
	correlationRepo := repository.NewCorrelationRepository(redisClient)
	suppressionRepo := repository.NewSuppressionRepository(pool)
	rcaRepo := repository.NewRCARepository(pool)

	alertSvc := service.NewAlertService(alertRepo)
	correlationSvc := service.NewCorrelationService(incidentRepo, correlationRepo)
	suppressionSvc := service.NewSuppressionService(suppressionRepo)
	rcaSvc := service.NewRCAService(rcaRepo)
	incidentSvc := service.NewIncidentService(incidentRepo, alertRepo, rcaSvc)

	alertHandler := handler.NewAlertHandler(alertSvc, suppressionSvc, correlationSvc, rcaSvc)
	rcaHandler := handler.NewRCAHandler(rcaSvc)
	incidentHandler := handler.NewIncidentHandler(incidentSvc)

	app := fiber.New()
	app.Use(fiberlogger.New())
	app.Use(recover.New())

	httpserver.NewRouter(app, alertHandler, rcaHandler, incidentHandler)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := app.Listen(":" + cfg.Port); err != nil {
			slog.Error("server stopped unexpectedly", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
	}
	slog.Info("server exited gracefully")
}
