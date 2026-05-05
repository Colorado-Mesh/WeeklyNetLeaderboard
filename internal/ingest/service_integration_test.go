package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"meshmonday/internal/config"
	"meshmonday/internal/storage"

	_ "modernc.org/sqlite"
)

func TestIntegrationIngestDedupeAndMondayCheckin(t *testing.T) {
	dbPath := "./test_ingest.db"
	removeSQLiteFiles(dbPath)
	defer removeSQLiteFiles(dbPath)

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	cfg := config.Config{
		IATADefault:   "SEA",
		TZ:            "America/Los_Angeles",
		TrackFromDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}

	svc := NewService(cfg, store, slog.Default())
	observed := time.Date(2026, 5, 4, 16, 0, 0, 0, time.UTC)
	payloadHex := "110041766572793a20496e746567726174696f6e207465737420236d6573686d6f6e646179"

	svc.HandleMessage(context.Background(), "meshcore/SEA/dev1/packets", payloadHex, observed)
	svc.HandleMessage(context.Background(), "meshcore/SEA/dev1/packets", payloadHex, observed.Add(5*time.Second))

	week := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	rows, err := store.ListCheckinsByWeek(context.Background(), []string{"SEA"}, week)
	if err != nil {
		t.Fatalf("list checkins: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 checkin after dedupe, got %d", len(rows))
	}
	if rows[0].ObserverCount != 1 {
		t.Fatalf("expected 1 observer, got %d", rows[0].ObserverCount)
	}
}

func TestIntegrationObserverAggregationFromEnvelopeOrigin(t *testing.T) {
	dbPath := "./test_ingest_observers.db"
	removeSQLiteFiles(dbPath)
	defer removeSQLiteFiles(dbPath)

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	cfg := config.Config{
		IATADefault:   "SEA",
		TZ:            "America/Los_Angeles",
		TrackFromDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}
	svc := NewService(cfg, store, slog.Default())

	observed := time.Date(2026, 5, 4, 16, 30, 0, 0, time.UTC)
	packetHex := "110041766572793a20496e746567726174696f6e207465737420236d6573686d6f6e646179"

	envelope := func(origin string) string {
		return fmt.Sprintf(`{"origin":"%s","raw":"%s","type":"PACKET","direction":"rx"}`, origin, packetHex)
	}

	svc.HandleMessage(context.Background(), "meshcore/SEA/dev1/packets", envelope("Krzychu-RPT"), observed)
	svc.HandleMessage(context.Background(), "meshcore/SEA/dev2/packets", envelope("Olympia-RPT"), observed.Add(2*time.Second))
	svc.HandleMessage(context.Background(), "meshcore/SEA/dev3/packets", envelope("Krzychu-RPT"), observed.Add(4*time.Second))

	week := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	rows, err := store.ListCheckinsByWeek(context.Background(), []string{"SEA"}, week)
	if err != nil {
		t.Fatalf("list checkins: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 checkin, got %d", len(rows))
	}
	if rows[0].ObserverCount != 2 {
		t.Fatalf("expected 2 unique observers, got %d", rows[0].ObserverCount)
	}
	if len(rows[0].ObserverNames) != 2 {
		t.Fatalf("expected 2 observer names, got %d (%v)", len(rows[0].ObserverNames), rows[0].ObserverNames)
	}
}

func TestIntegrationUsernameDedupWithinWeek(t *testing.T) {
	dbPath := "./test_ingest_username_dedupe.db"
	removeSQLiteFiles(dbPath)
	defer removeSQLiteFiles(dbPath)

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	cfg := config.Config{
		IATADefault:   "SEA",
		TZ:            "America/Los_Angeles",
		TrackFromDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}
	svc := NewService(cfg, store, slog.Default())

	observed := time.Date(2026, 5, 4, 18, 0, 0, 0, time.UTC)
	// Same username "Avery", different packet hashes/messages.
	firstPacketHex := "110041766572793a20436865636b696e203120236d6573686d6f6e646179"
	secondPacketHex := "110041766572793a20436865636b696e203220236d6573686d6f6e646179"

	svc.HandleMessage(context.Background(), "meshcore/SEA/dev1/packets", firstPacketHex, observed)
	svc.HandleMessage(context.Background(), "meshcore/PDX/dev2/packets", secondPacketHex, observed.Add(10*time.Second))

	week := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	rows, err := store.ListCheckinsByWeek(context.Background(), nil, week)
	if err != nil {
		t.Fatalf("list checkins: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 globally deduped checkin by username, got %d", len(rows))
	}
}

func TestIntegrationObserverBadgeAggregatesAcrossUserPackets(t *testing.T) {
	dbPath := "./test_ingest_observer_aggregate.db"
	removeSQLiteFiles(dbPath)
	defer removeSQLiteFiles(dbPath)

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	cfg := config.Config{
		IATADefault:   "SEA",
		TZ:            "America/Los_Angeles",
		TrackFromDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}
	svc := NewService(cfg, store, slog.Default())

	observed := time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)
	firstPacketHex := "110041766572793a20416c70686120236d6573686d6f6e646179"
	secondPacketHex := "110041766572793a20427261766f20236d6573686d6f6e646179"

	// Same user/week, different packets and observers across IATAs.
	svc.HandleMessage(context.Background(), "meshcore/SEA/obs1/packets", firstPacketHex, observed)
	svc.HandleMessage(context.Background(), "meshcore/PDX/obs2/packets", secondPacketHex, observed.Add(2*time.Second))

	week := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	rows, err := store.ListCheckinsByWeek(context.Background(), nil, week)
	if err != nil {
		t.Fatalf("list checkins: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 deduped checkin row, got %d", len(rows))
	}
	if rows[0].ObserverCount != 2 {
		t.Fatalf("expected observer count aggregated across packets to be 2, got %d", rows[0].ObserverCount)
	}
	if len(rows[0].ObserverNames) != 2 {
		t.Fatalf("expected 2 observer names, got %d (%v)", len(rows[0].ObserverNames), rows[0].ObserverNames)
	}
}

func TestIntegrationRestoreCheckinPacketsFromRaw(t *testing.T) {
	dbPath := "./test_restore_checkin_packets.db"
	removeSQLiteFiles(dbPath)
	defer removeSQLiteFiles(dbPath)

	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close()

	cfg := config.Config{
		IATADefault:   "SEA",
		TZ:            "America/Los_Angeles",
		TrackFromDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
	}
	svc := NewService(cfg, store, slog.Default())

	observed := time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)
	firstPacketHex := "110041766572793a20416c70686120236d6573686d6f6e646179"
	secondPacketHex := "110041766572793a20427261766f20236d6573686d6f6e646179"

	svc.HandleMessage(context.Background(), "meshcore/SEA/obs1/packets", firstPacketHex, observed)
	svc.HandleMessage(context.Background(), "meshcore/PDX/obs2/packets", secondPacketHex, observed.Add(2*time.Second))

	week := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	rows, err := store.ListCheckinsByWeek(context.Background(), nil, week)
	if err != nil {
		t.Fatalf("list checkins before wipe: %v", err)
	}
	if len(rows) != 1 || rows[0].ObserverCount != 2 {
		t.Fatalf("expected initial observer aggregation of 2, got rows=%d observer_count=%d", len(rows), rows[0].ObserverCount)
	}

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite direct: %v", err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("DELETE FROM checkin_packets;"); err != nil {
		t.Fatalf("wipe checkin_packets: %v", err)
	}

	rows, err = store.ListCheckinsByWeek(context.Background(), nil, week)
	if err != nil {
		t.Fatalf("list checkins after wipe: %v", err)
	}
	if len(rows) != 1 || rows[0].ObserverCount != 0 {
		t.Fatalf("expected observer aggregation to drop to 0 after wipe, got rows=%d observer_count=%d", len(rows), rows[0].ObserverCount)
	}

	restoreResult, err := svc.RestoreCheckinPacketsFromRaw(context.Background())
	if err != nil {
		t.Fatalf("restore checkin packets: %v", err)
	}
	if restoreResult.RawScanned < 2 || restoreResult.LinksInserted < 2 {
		t.Fatalf("expected restore to scan and reinsert packet links, got %+v", restoreResult)
	}

	rows, err = store.ListCheckinsByWeek(context.Background(), nil, week)
	if err != nil {
		t.Fatalf("list checkins after restore: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 checkin after restore, got %d", len(rows))
	}
	if rows[0].ObserverCount != 2 {
		t.Fatalf("expected observer aggregation restored to 2, got %d", rows[0].ObserverCount)
	}

}

func removeSQLiteFiles(dbPath string) {
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
}
