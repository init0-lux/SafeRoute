package main

import (
	"log/slog"
	"os"

	"saferoute-backend/config"
	dbconn "saferoute-backend/internal/common/db"
	"saferoute-backend/internal/schema"
)

func main() {
	cfg := config.Load()

	db, err := dbconn.Open(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	if err := schema.Sync(db); err != nil {
		slog.Error("failed to sync database schema", "error", err)
		os.Exit(1)
	}

	slog.Info(
		"database schema sync complete",
		"tables",
		[]string{
			"users",
			"trusted_contacts",
			"trusted_contact_requests",
			"user_verifications",
			"reports",
			"evidence",
			"complaint_events",
			"sos_sessions",
			"location_pings",
			"sos_viewer_grants",
		},
	)
}
