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
	"saferoute-backend/internal/notify"
	"saferoute-backend/internal/reports"
	"saferoute-backend/internal/safety"
	"saferoute-backend/internal/sos"
	"saferoute-backend/internal/trust"
	"saferoute-backend/internal/trustedcontacts"
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
	sosRepo := sos.NewRepository(database)
	trustedContactsRepo := trustedcontacts.NewRepository(database)

	trustService := trust.NewService(trust.NewRepository(database))
	reportsService := reports.NewService(reports.NewRepository(database), reports.ServiceConfig{
		DefaultNearbyLimit: cfg.ReportsNearbyDefaultLimit,
		MaxNearbyLimit:     cfg.ReportsNearbyMaxLimit,
		MaxNearbyRadiusM:   cfg.ReportsNearbyMaxRadiusM,
	}, trustService)
	reportsHandler := reports.NewHandler(reportsService, authMiddleware.VerifyUser())
	trustHandler := trust.NewHandler(trustService, authMiddleware.VerifyUser())
	safetyHandler := safety.NewHandler(
		safety.NewService(
			safety.NewRepository(database),
			safety.NewGoogleRoutesProvider(safety.GoogleRoutesConfig{
				APIKey:  cfg.GoogleRoutesAPIKey,
				BaseURL: cfg.GoogleRoutesBaseURL,
			}),
			safety.ServiceConfig{
				DefaultRadiusM:       cfg.SafetyDefaultRadiusM,
				MaxRadiusM:           cfg.SafetyMaxRadiusM,
				RecentWindow:         cfg.SafetyRecentWindow,
				RouteCorridorRadiusM: cfg.SafetyRouteCorridorRadiusM,
				RouteSegmentLengthM:  cfg.SafetyRouteSegmentLengthM,
				RouteMaxDistanceM:    cfg.SafetyRouteMaxDistanceM,
				TimeRisk: safety.TimeRiskConfig{
					Timezone:         cfg.SafetyTimeRiskTimezone,
					HighStartHour:    cfg.SafetyTimeRiskHighStartHour,
					HighEndHour:      cfg.SafetyTimeRiskHighEndHour,
					MorningStartHour: cfg.SafetyTimeRiskMorningStartHour,
					MorningEndHour:   cfg.SafetyTimeRiskMorningEndHour,
					EveningStartHour: cfg.SafetyTimeRiskEveningStartHour,
					EveningEndHour:   cfg.SafetyTimeRiskEveningEndHour,
				},
			},
		),
	)
	evidenceHandler := evidence.NewHandler(
		evidence.NewService(
			evidence.NewRepository(database),
			evidence.NewLocalStorage(cfg.EvidenceStorageRoot),
			reportsService,
			sosRepo,
			evidence.ServiceConfig{
				MaxFileSizeBytes: cfg.MaxEvidenceSizeBytes,
			},
		),
		authMiddleware.VerifyUser(),
	)

	notificationSender := notify.NewMultiSender()
	notificationSender.Register(notify.ChannelPush, notify.NewExpoPushSender())

	devSender := notify.NewDevSender(slog.Default())
	notificationSender.Register(notify.ChannelSMS, devSender)
	notificationSender.Register(notify.ChannelEmail, devSender)

	slog.Info("initialized notification senders", "push", "expo", "fallback", "dev-log")

	trustedContactsService := trustedcontacts.NewService(trustedContactsRepo, auth.NewRepository(database))
	trustedContactsHandler := trustedcontacts.NewHandler(
		trustedContactsService,
		authMiddleware,
	)

	sosService := sos.NewService(sosRepo)
	sosService.SetNotifyTrustedContactsFunc(func(ctx context.Context, session *sos.SOSSession) error {
		if session == nil || session.UserID == nil {
			return nil
		}

		slog.Info("triggering notifications for SOS session", "session_id", session.ID, "user_id", *session.UserID)

		summary, err := sosService.NotifyTrustedContacts(
			ctx,
			trustedContactsService,
			notificationSender,
			session,
			cfg.SOSViewerBaseURL,
		)
		if err != nil {
			slog.Error("failed to notify trusted contacts", "error", err, "session_id", session.ID)
			return nil
		}

		slog.Info(
			"notifications sent",
			"session_id", summary.SessionID,
			"successful", summary.Successful,
			"failed", summary.Failed,
		)
		return nil
	})
	sosHandler := sos.NewHandler(sosService, authMiddleware)

	workerManager := workers.NewManager(
		workers.NewCleanupJob(cfg.EvidenceStorageRoot, cfg.WorkerCleanupInterval, cfg.WorkerCleanupMaxAge),
		workers.NewSafetyCacheJob(nil, cfg.WorkerSafetyRefreshInterval),
		workers.NewIPFSUploadJob(nil, cfg.WorkerIPFSPollInterval),
	)

	server := app.New(
		cfg,
		authHandler.RegisterRoutes,
		reportsHandler.RegisterRoutes,
		trustHandler.RegisterRoutes,
		safetyHandler.RegisterRoutes,
		evidenceHandler.RegisterRoutes,
		trustedContactsHandler.RegisterRoutes,
		sosHandler.RegisterRoutes,
	)
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
