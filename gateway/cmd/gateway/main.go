package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lychee-ripe/gateway/internal/config"
	"github.com/lychee-ripe/gateway/internal/handler"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/proxy"
)

func main() {
	cfgPath := flag.String("config", "", "path to gateway.yaml (default: $LYCHEE_GATEWAY_CONFIG or configs/gateway.yaml)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	// Set up structured logger.
	var logLevel slog.Level
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var logHandler slog.Handler
	opts := &slog.HandlerOptions{Level: logLevel}
	if cfg.Logging.Format == "text" {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(logHandler)

	// Build the reverse proxy.
	rp, err := proxy.New(cfg.Upstream, logger)
	if err != nil {
		logger.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	// Compose the middleware chain.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Health(cfg.Upstream, logger))
	mux.Handle("/", rp)

	// Apply middleware (outermost runs first).
	// Order: RequestID → Logging → CORS → RateLimit → Auth → handler
	var h http.Handler = mux
	h = middleware.Auth(cfg.Auth, logger)(h)
	h = middleware.RateLimit(cfg.RateLimit, logger)(h)
	h = middleware.CORS(cfg.CORS)(h)
	h = middleware.Logging(logger)(h)
	h = middleware.RequestID(h)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      h,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutS) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutS) * time.Second,
	}

	// Graceful shutdown.
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("gateway listening",
			"addr", cfg.Addr(),
			"upstream", cfg.Upstream.BaseURL,
			"auth", cfg.Auth.Enabled,
			"rate_limit", cfg.RateLimit.Enabled,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}
