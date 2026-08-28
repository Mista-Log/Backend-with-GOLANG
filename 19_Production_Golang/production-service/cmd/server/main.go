// cmd/server/main.go is deliberately THIN — per the guide's Project
// Structure section, it does exactly one job: load config, construct
// dependencies IN ORDER (manual dependency injection), start the server,
// and wait for a shutdown signal. No business logic lives here at all.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"productionservice/internal/config"
	appHTTP "productionservice/internal/transport/http"
	"productionservice/internal/repository"
	"productionservice/internal/service"
)

func main() {
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	// --- Manual dependency injection, in dependency order -----------------
	// This is exactly the guide's "main constructs everything, in order,
	// and passes it down" diagram — no DI framework, just constructor calls.
	orderRepo := repository.NewInMemoryOrderRepository()
	orderService := service.NewOrderService(orderRepo, logger)

	// The readiness check: in a real service, this would ping a real
	// database/cache. Here it's always healthy, since there's nothing
	// external to check — but the SHAPE (an injected function, not a
	// concrete dependency) is what matters for the transport layer to
	// stay decoupled.
	ready := func() error { return nil }

	router := appHTTP.NewRouter(orderService, ready)

	// A separate mux for metrics/pprof-style operational endpoints is a
	// common real pattern — keeps them off the main request router entirely.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())

	appServer := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: router}
	metricsServer := &http.Server{Addr: ":9090", Handler: metricsMux}

	go func() {
		logger.Info("app server starting", "port", cfg.Port)
		if err := appServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("app server failed", "error", err)
			os.Exit(1)
		}
	}()
	go func() {
		logger.Info("metrics server starting", "port", 9090)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	// --- Graceful shutdown -----------------------------------------------
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh // blocks here until Ctrl+C (SIGINT) or a real SIGTERM arrives

	logger.Info("shutdown signal received, finishing in-flight requests")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := appServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed, forcing close", "error", err)
		appServer.Close()
	}
	metricsServer.Shutdown(ctx)

	logger.Info("shutdown complete")
}
