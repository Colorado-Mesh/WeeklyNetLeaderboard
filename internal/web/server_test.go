package web

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"weeklynet/internal/config"
	"weeklynet/internal/storage"
)

func newTestServer(t *testing.T) (*Server, *storage.SQLiteStore) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(cwd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to project root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	dbPath := filepath.Join(t.TempDir(), "web_test.db")
	store, err := storage.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cfg := config.Config{
		MeshName:          "CascadiaMesh",
		DiceBearStyle:     "fun-emoji",
		UIPollSeconds:     15,
		TZ:                "America/Los_Angeles",
		TrackFromDate:     time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		MQTTTopicTemplate: "meshcore/+/+/packets",
		DayOfWeek:         time.Monday,
		CheckInHashtag:    "#meshmonday",
	}
	srv, err := NewServer(cfg, store, slog.Default())
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv, store
}

func TestRoutesIncludeSecurityHeaders(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected X-Content-Type-Options: %q", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("unexpected X-Frame-Options: %q", got)
	}
	if got := rr.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("unexpected Referrer-Policy: %q", got)
	}
}

func TestMethodNotAllowedOnReadEndpoint(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/checkins", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if got := rr.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("unexpected allow header: %q", got)
	}
}

func TestInvalidWeekReturnsBadRequest(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/checkins?week=bad-date", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid_week") {
		t.Fatalf("expected invalid_week response, got %q", rr.Body.String())
	}
}

func TestAPIDoesNotLeakInternalErrors(t *testing.T) {
	srv, store := newTestServer(t)
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/checkins?week=2026-05-04", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(strings.ToLower(body), "database") || strings.Contains(strings.ToLower(body), "sqlite") {
		t.Fatalf("internal details leaked in response: %q", body)
	}
	if !strings.Contains(body, "internal_error") {
		t.Fatalf("expected generic internal_error body, got %q", body)
	}
}
