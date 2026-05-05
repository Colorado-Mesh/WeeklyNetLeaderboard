package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteSetsBusyTimeoutAndPoolLimits(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "store.db")
	store, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	var busyTimeout int
	if err := store.db.QueryRowContext(context.Background(), "PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout < 5000 {
		t.Fatalf("expected busy_timeout >= 5000, got %d", busyTimeout)
	}

	stats := store.db.Stats()
	if stats.MaxOpenConnections != 4 {
		t.Fatalf("expected max open conns 4, got %d", stats.MaxOpenConnections)
	}
}
