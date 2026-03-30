package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"saferoute-backend/config"
	"saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"
	dbconn "saferoute-backend/internal/common/db"
	"saferoute-backend/internal/evidence"
	"saferoute-backend/internal/reports"
	"saferoute-backend/internal/safety"
	"saferoute-backend/internal/sos"
	"saferoute-backend/internal/trust"
	"saferoute-backend/internal/workers"
)

func main() {
	cfg := config.Load()
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := dbconn.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := database.DB()
	if err != nil {
		slog.Error("failed to access database handle", "error", err)
		os.Exit(1)
	}

	if err := sqlDB.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	sessionManager, err := auth.NewSessionManager(auth.SessionConfig{
		AccessSecret:      cfg.JWTAccessSecret,
		RefreshSecret:     cfg.JWTRefreshSecret,
		AccessTTL:         cfg.JWTAccessTTL,
		RefreshTTL:        cfg.JWTRefreshTTL,
		AccessCookieName:  cfg.AuthAccessCookieName,
		RefreshCookieName: cfg.AuthRefreshCookieName,
		CookieDomain:      cfg.AuthCookieDomain,
		CookieSameSite:    cfg.AuthCookieSameSite,
		CookieSecure:      cfg.AuthCookieSecure,
	})
	if err != nil {
		slog.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}

	authService := auth.NewService(auth.NewRepository(database))
	authHandler := auth.NewHandler(authService, sessionManager)
	authMiddleware := auth.NewMiddleware(authService, sessionManager)
	trustService := trust.NewService(trust.NewRepository(database))
	reportsService := reports.NewService(reports.NewRepository(database), reports.ServiceConfig{
		DefaultNearbyLimit: cfg.ReportsNearbyDefaultLimit,
		MaxNearbyLimit:     cfg.ReportsNearbyMaxLimit,
		MaxNearbyRadiusM:   cfg.ReportsNearbyMaxRadiusM,
	}, trustService)
	reportsHandler := reports.NewHandler(
		reportsService,
		authMiddleware.VerifyUser(),
	)
	trustHandler := trust.NewHandler(trustService, authMiddleware.VerifyUser())
	safetyHandler := safety.NewHandler(
		safety.NewService(
			safety.NewRepository(database),
			safety.ServiceConfig{
				DefaultRadiusM: cfg.SafetyDefaultRadiusM,
				MaxRadiusM:     cfg.SafetyMaxRadiusM,
				RecentWindow:   cfg.SafetyRecentWindow,
			},
		),
	)
	evidenceHandler := evidence.NewHandler(
		evidence.NewService(
			evidence.NewRepository(database),
			evidence.NewLocalStorage(cfg.EvidenceStorageRoot),
			reportsService,
			sos.NewRepository(database),
			evidence.ServiceConfig{
				MaxFileSizeBytes: cfg.MaxEvidenceSizeBytes,
			},
		),
		authMiddleware.VerifyUser(),
	)
	workerManager := workers.NewManager(
		workers.NewCleanupJob(cfg.EvidenceStorageRoot, cfg.WorkerCleanupInterval, cfg.WorkerCleanupMaxAge),
		workers.NewSafetyCacheJob(nil, cfg.WorkerSafetyRefreshInterval),
		workers.NewIPFSUploadJob(nil, cfg.WorkerIPFSPollInterval),
	)

	server := app.New(cfg, authHandler.RegisterRoutes, reportsHandler.RegisterRoutes, trustHandler.RegisterRoutes, safetyHandler.RegisterRoutes, evidenceHandler.RegisterRoutes)
	addr := cfg.Address()

	workerManager.Start(appCtx)

	go func() {
		<-appCtx.Done()
		if err := server.Shutdown(); err != nil {
			slog.Error("failed to shut down server", "error", err)
		}
	}()

	slog.Info("starting SafeRoute backend", "addr", addr)

	if err := server.Listen(addr); err != nil && appCtx.Err() == nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
