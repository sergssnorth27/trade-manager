package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"trade-manager/internal/config"
	"trade-manager/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found, using system env", "err", err)
	}

	fmt.Println("PG_DSN:", os.Getenv("PG_DSN"))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := store.New(ctx, cfg.PGDSN)
	if err != nil {
		slog.Error("store", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("trade-manager started")

}
