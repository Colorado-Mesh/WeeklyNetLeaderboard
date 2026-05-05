package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"meshmonday/internal/checkins"
	"meshmonday/internal/config"
	"meshmonday/internal/leaderboard"
	"meshmonday/internal/models"
	"meshmonday/internal/storage"
)

type Server struct {
	cfg       config.Config
	store     *storage.SQLiteStore
	logger    *slog.Logger
	templates *template.Template
}

func NewServer(cfg config.Config, store *storage.SQLiteStore, logger *slog.Logger) (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"observerTooltip": observerTooltip,
	}).ParseGlob(filepath.Join("web", "templates", "*.html"))
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		store:     store,
		logger:    logger,
		templates: tmpl,
	}, nil
}

func observerTooltip(count int, names []string) string {
	if count <= 0 {
		return "No observers recorded yet."
	}
	if len(names) == 0 {
		return "Observer data unavailable."
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	if len(sorted) == 1 {
		return "1 observer: " + sorted[0]
	}
	return fmt.Sprintf("%d observers: %s", len(sorted), strings.Join(sorted, " • "))
}

func configuredHashtagChannels(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, raw := range values {
		channel := strings.TrimSpace(raw)
		if channel == "" {
			continue
		}
		if !strings.HasPrefix(channel, "#") {
			channel = "#" + channel
		}
		channel = strings.ToLower(channel)
		if channel == "#meshmonday" {
			continue
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		out = append(out, channel)
	}
	return out
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/api/checkins", s.handleAPIWeekCheckins)
	mux.HandleFunc("/api/leaderboard", s.handleAPILeaderboard)
	mux.HandleFunc("/leaderboard", s.handleLeaderboardPage)
	mux.HandleFunc("/", s.handleMondayPage)
	return loggingMiddleware(mux, s.logger)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleMondayPage(w http.ResponseWriter, r *http.Request) {
	weekStart := checkins.WeekStartMonday(time.Now(), s.cfg.TZ)
	if raw := strings.TrimSpace(r.URL.Query().Get("week")); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			weekStart = parsed
		}
	}
	items, err := s.store.ListCheckinsByWeek(r.Context(), s.cfg.IATAFilters, weekStart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		WeekStart string
		MeshName  string
		ListenChannels []string
		DiceBearStyle string
		UIPollSeconds int
		IATA      string
		Count     int
		Checkins  []models.Checkin
	}{
		WeekStart: weekStart.Format("2006-01-02"),
		MeshName:  s.cfg.MeshName,
		ListenChannels: configuredHashtagChannels(s.cfg.HashtagChannels),
		DiceBearStyle: s.cfg.DiceBearStyle,
		UIPollSeconds: s.cfg.UIPollSeconds,
		IATA:      s.cfg.IATAFilterLabel(),
		Count:     len(items),
		Checkins:  items,
	}
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleLeaderboardPage(w http.ResponseWriter, r *http.Request) {
	entries, err := s.computeLeaderboard(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Entries     []models.LeaderboardEntry
		MeshName    string
		TrackedFrom string
		DiceBearStyle string
		UIPollSeconds int
	}{
		Entries:     entries,
		MeshName:    s.cfg.MeshName,
		TrackedFrom: s.cfg.TrackFromDate.Format("2006-01-02"),
		DiceBearStyle: s.cfg.DiceBearStyle,
		UIPollSeconds: s.cfg.UIPollSeconds,
	}
	if err := s.templates.ExecuteTemplate(w, "leaderboard.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAPIWeekCheckins(w http.ResponseWriter, r *http.Request) {
	weekStart := checkins.WeekStartMonday(time.Now(), s.cfg.TZ)
	if raw := strings.TrimSpace(r.URL.Query().Get("week")); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			weekStart = parsed
		}
	}
	items, err := s.store.ListCheckinsByWeek(r.Context(), s.cfg.IATAFilters, weekStart)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"week_start": weekStart.Format("2006-01-02"),
		"iata":       s.cfg.IATAFilterLabel(),
		"count":      len(items),
		"checkins":   items,
	})
}

func (s *Server) handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := s.computeLeaderboard(r.Context())
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"tracked_from": s.cfg.TrackFromDate.Format("2006-01-02"),
		"entries":      entries,
	})
}

func (s *Server) computeLeaderboard(ctx context.Context) ([]models.LeaderboardEntry, error) {
	checkinRows, err := s.store.ListCheckinsSince(ctx, s.cfg.IATAFilters, s.cfg.TrackFromDate)
	if err != nil {
		return nil, err
	}
	entries := leaderboard.Compute(checkinRows, s.cfg.TrackFromDate)
	if err := s.store.ReplaceLeaderboardSnapshots(ctx, s.cfg.TrackFromDate, entries, time.Now().UTC()); err != nil {
		s.logger.Warn("replace leaderboard snapshots failed", "error", err.Error())
	}
	return entries, nil
}

func loggingMiddleware(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http_request", "path", r.URL.Path, "method", r.Method, "duration_ms", time.Since(start).Milliseconds())
	})
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
