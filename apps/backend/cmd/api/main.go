package main

import (
	"log/slog"
	"os"

	"saferoute-backend/config"
	"saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"
	dbconn "saferoute-backend/internal/common/db"
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

	authHandler := auth.NewHandler(
		auth.NewService(auth.NewRepository(database)),
		sessionManager,
	)

	server := app.New(cfg, authHandler.RegisterRoutes)
	addr := cfg.Address()

	slog.Info("starting SafeRoute backend", "addr", addr)

	if err := server.Listen(addr); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
