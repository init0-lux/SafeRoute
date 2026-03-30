package main

import (
	"log/slog"
	"os"

	"saferoute-backend/config"
	"saferoute-backend/internal/app"
)

func main() {
	cfg := config.Load()
	server := app.New(cfg)
	addr := cfg.Address()

	slog.Info("starting SafeRoute backend", "addr", addr, "env", cfg.Environment)

	if err := server.Listen(addr); err != nil {
		slog.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}
