package main

import (
	"context"
	"os"
	"time"

	"weeklynet/internal/config"
	"weeklynet/internal/ingest"
	"weeklynet/internal/logging"
	"weeklynet/internal/storage"
)

func main() {
	_ = config.LoadEnvFile(".env.local")
	logger := logging.NewJSONLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err.Error())
		os.Exit(1)
	}

	store, err := storage.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		logger.Error("open sqlite", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Warn("close sqlite", "error", err.Error())
		}
	}()

	svc := ingest.NewService(cfg, store, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := svc.RestoreCheckinPacketsFromRaw(ctx)
	if err != nil {
		logger.Error("restore checkin packets failed", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("restore checkin packets complete",
		"raw_scanned", result.RawScanned,
		"decoded", result.Decoded,
		"checkins_found", result.CheckinsFound,
		"qualifying_checkins", result.QualifyingCheckins,
		"links_inserted", result.LinksInserted,
		"decode_errors", result.DecodeErrors,
		"insert_link_errors", result.InsertLinkErrors,
	)
}
