package main

import (
	"log/slog"
	"os"

	"saferoute-backend/config"
	"saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"
	dbconn "saferoute-backend/internal/common/db"
	"saferoute-backend/internal/notify"
	"saferoute-backend/internal/sos"
	"saferoute-backend/internal/trustedcontacts"
)

func main() {
	cfg := config.Load()

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

	// Initialize Notification Sender
	var notificationSender notify.Sender
	if cfg.Environment == "production" {
		slog.Info("initializing Expo Push notification sender")
		notificationSender = notify.NewExpoPushSender()
	} else {
		slog.Info("initializing Dev notification sender (logging only)")
		notificationSender = notify.NewDevSender(slog.Default())
	}

	authService := auth.NewService(auth.NewRepository(database))
	authMiddleware := auth.NewMiddleware(authService, sessionManager)
	authHandler := auth.NewHandler(authService, sessionManager)

	trustedContactsRepo := trustedcontacts.NewRepository(database)
	trustedContactsService := trustedcontacts.NewService(trustedContactsRepo)
	trustedContactsHandler := trustedcontacts.NewHandler(
		trustedContactsService,
		authMiddleware,
	)

	sosService := sos.NewService(sos.NewRepository(database))
	// Wire up notifications for SOS sessions
	sosService.SetNotifyTrustedContactsFunc(func(ctx context.Context, session *sos.SOSSession) error {
		if session == nil || session.UserID == nil {
			return nil
		}
		
		slog.Info("triggering notifications for SOS session", "session_id", session.ID, "user_id", *session.UserID)
		
		// Run fanout in a background goroutine or synchronously? 
		// Usually better to not block the session start, but we want to know if it fails.
		// For now, let's run it synchronously or at least capture errors in logs.
		summary, err := sosService.NotifyTrustedContacts(
			ctx,
			trustedContactsRepo, // Using repo as reader
			notificationSender,
			session,
			cfg.SOSViewerBaseURL,
		)
		if err != nil {
			slog.Error("failed to notify trusted contacts", "error", err, "session_id", session.ID)
			return nil // Don't fail the session start if notifications fail
		}
		
		slog.Info("notifications sent", 
			"session_id", summary.SessionID, 
			"successful", summary.Successful, 
			"failed", summary.Failed,
		)
		return nil
	})

	sosHandler := sos.NewHandler(sosService, authMiddleware)

	server := app.New(cfg, authHandler.RegisterRoutes, trustedContactsHandler.RegisterRoutes, sosHandler.RegisterRoutes)
	addr := cfg.Address()

	slog.Info("starting SafeRoute backend", "addr", addr)

	if err := server.Listen(addr); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
