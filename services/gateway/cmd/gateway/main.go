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

	"github.com/lychee-ripe/gateway/internal/chain/evm"
	"github.com/lychee-ripe/gateway/internal/config"
	gatewaydb "github.com/lychee-ripe/gateway/internal/db"
	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/handler"
	"github.com/lychee-ripe/gateway/internal/middleware"
	"github.com/lychee-ripe/gateway/internal/oidc"
	"github.com/lychee-ripe/gateway/internal/proxy"
	repositorygorm "github.com/lychee-ripe/gateway/internal/repository/gorm"
	"github.com/lychee-ripe/gateway/internal/service"
)

func main() {
	cfgPath := flag.String("config", "", "path to gateway.yaml (default: $LYCHEE_GATEWAY_CONFIG or repo-root tooling/configs/gateway.yaml)")
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

	// Initialize DB and run schema migrations before serving traffic.
	gdb, err := gatewaydb.OpenGORM(context.Background(), cfg.DB)
	if err != nil {
		logger.Error("failed to open database", "driver", cfg.DB.Driver, "error", err)
		os.Exit(1)
	}
	dbConn, err := gdb.DB()
	if err != nil {
		logger.Error("failed to get sql db handle", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			logger.Error("failed to close database", "error", err)
		}
	}()

	if err := gatewaydb.AutoMigrate(context.Background(), gdb); err != nil {
		logger.Error("failed to auto migrate schema", "error", err)
		os.Exit(1)
	}

	var chainAdapter *evm.Adapter
	if cfg.Trace.Mode == domain.TraceModeBlockchain {
		chainAdapter, err = evm.NewAdapter(context.Background(), cfg.Chain)
		if err != nil {
			logger.Error("failed to initialize chain adapter", "error", err)
			os.Exit(1)
		}
		defer chainAdapter.Close()
	}

	// Build the reverse proxy.
	rp, err := proxy.New(cfg.Upstream, logger)
	if err != nil {
		logger.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	repo := repositorygorm.New(gdb)
	if err := service.SeedDefaultResources(context.Background(), repo); err != nil {
		logger.Error("failed to seed default orchard data", "error", err)
		os.Exit(1)
	}
	batchSvc := service.NewBatchCreateService(repo, chainAdapter, cfg.Trace.Mode, logger)
	traceSvc := service.NewTraceService(repo, chainAdapter, cfg.Trace.Mode)
	reconcileSvc := service.NewReconcileService(repo, repo, chainAdapter, cfg.Trace.Mode, logger)
	dashboardSvc := service.NewDashboardService(repo, repo, cfg.Trace.Mode)
	authSvc := service.NewAuthService(repo)
	userSvc := service.NewUserAdminService(repo)
	orchardSvc := service.NewOrchardService(repo)
	plotSvc := service.NewPlotService(repo)

	var validator *oidc.Validator
	if cfg.Auth.Mode == config.AuthModeOIDC {
		validator, err = oidc.NewValidator(cfg.Auth.OIDC)
		if err != nil {
			logger.Error("failed to initialize oidc validator", "error", err)
			os.Exit(1)
		}
	}

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()
	if cfg.Trace.Mode == domain.TraceModeBlockchain {
		go reconcileSvc.StartAutoReconcileWorker(appCtx)
	}

	// Compose the middleware chain.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.Health(cfg.Upstream, logger))
	mux.HandleFunc("GET /v1/auth/me", handler.GetCurrentPrincipal())
	mux.HandleFunc("POST /v1/batches", handler.CreateBatch(batchSvc, logger))
	mux.HandleFunc("GET /v1/batches/{batch_id}", handler.GetBatch(batchSvc, logger))
	mux.HandleFunc("GET /v1/orchards", handler.ListOrchards(orchardSvc))
	mux.HandleFunc("POST /v1/orchards", handler.CreateOrchard(orchardSvc))
	mux.HandleFunc("PATCH /v1/orchards/{orchard_id}", handler.UpdateOrchard(orchardSvc))
	mux.HandleFunc("DELETE /v1/orchards/{orchard_id}", handler.ArchiveOrchard(orchardSvc))
	mux.HandleFunc("GET /v1/plots", handler.ListPlots(plotSvc))
	mux.HandleFunc("POST /v1/plots", handler.CreatePlot(plotSvc))
	mux.HandleFunc("PATCH /v1/plots/{plot_id}", handler.UpdatePlot(plotSvc))
	mux.HandleFunc("DELETE /v1/plots/{plot_id}", handler.ArchivePlot(plotSvc))
	mux.HandleFunc("GET /v1/admin/users", handler.ListUsers(userSvc))
	mux.HandleFunc("POST /v1/admin/users", handler.CreateUser(userSvc))
	mux.HandleFunc("PATCH /v1/admin/users/{user_id}", handler.UpdateUser(userSvc))
	mux.HandleFunc("GET /v1/trace/{trace_code}", handler.GetPublicTrace(traceSvc, logger))
	mux.HandleFunc("GET /v1/dashboard/overview", handler.GetDashboardOverview(dashboardSvc, logger))
	mux.HandleFunc("POST /v1/batches/reconcile", handler.ReconcileBatches(reconcileSvc, logger))
	mux.Handle("/", rp)

	// Apply middleware (outermost runs first).
	// Order: RequestID → Logging → CORS → RateLimit → Auth → handler
	var h http.Handler = mux
	h = middleware.Auth(cfg.Auth, validator, authSvc, logger)(h)
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
			"db_driver", cfg.DB.Driver,
			"db_dsn", gatewaydb.SanitizeDSN(cfg.DB.Driver, cfg.DB.DSN),
			"trace_mode", cfg.Trace.Mode,
			"chain_rpc_url", cfg.Chain.RPCURL,
			"chain_id", cfg.Chain.ChainID,
			"chain_contract_address", cfg.Chain.ContractAddress,
			"auth_mode", cfg.Auth.Mode,
			"rate_limit", cfg.RateLimit.Enabled,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("shutting down...")
	appCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}
