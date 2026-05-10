package leaderboard

import (
	"testing"
	"time"

	"weeklynet/internal/models"
)

func TestCompute(t *testing.T) {
	trackFrom := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	checkins := []models.Checkin{
		{Username: "finley", DisplayName: "Finley", WeekStart: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)},
		{Username: "finley", DisplayName: "Finley", WeekStart: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)},
		{Username: "gray", DisplayName: "Gray", WeekStart: time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)},
	}
	rows := Compute(checkins, trackFrom)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Username != "finley" {
		t.Fatalf("expected finley first, got %s", rows[0].Username)
	}
	if rows[0].LongestStreak != 2 {
		t.Fatalf("expected streak 2, got %d", rows[0].LongestStreak)
	}
}
