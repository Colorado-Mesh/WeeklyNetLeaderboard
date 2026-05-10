package leaderboard

import (
	"sort"
	"time"

	"weeklynet/internal/models"
)

func Compute(checkins []models.Checkin, trackedFrom time.Time) []models.LeaderboardEntry {
	type state struct {
		displayName string
		total       int
		weeks       []time.Time
	}
	byUser := make(map[string]*state)
	for _, c := range checkins {
		entry, ok := byUser[c.Username]
		if !ok {
			entry = &state{displayName: c.DisplayName}
			byUser[c.Username] = entry
		}
		if entry.displayName == "" {
			entry.displayName = c.DisplayName
		}
		entry.total++
		entry.weeks = append(entry.weeks, c.WeekStart)
	}

	results := make([]models.LeaderboardEntry, 0, len(byUser))
	for username, entry := range byUser {
		longest, streakStart := longestWeeklyStreak(entry.weeks)
		results = append(results, models.LeaderboardEntry{
			Username:        username,
			DisplayName:     firstNonEmpty(entry.displayName, username),
			MostCheckins:    entry.total,
			TrackedFrom:     trackedFrom.Format("2006-01-02"),
			LongestStreak:   longest,
			StreakStartDate: streakStart.Format("2006-01-02"),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].MostCheckins != results[j].MostCheckins {
			return results[i].MostCheckins > results[j].MostCheckins
		}
		if results[i].LongestStreak != results[j].LongestStreak {
			return results[i].LongestStreak > results[j].LongestStreak
		}
		return results[i].Username < results[j].Username
	})
	return results
}

func longestWeeklyStreak(weeks []time.Time) (int, time.Time) {
	if len(weeks) == 0 {
		return 0, time.Time{}
	}
	sort.Slice(weeks, func(i, j int) bool {
		return weeks[i].Before(weeks[j])
	})
	dedup := weeks[:1]
	for i := 1; i < len(weeks); i++ {
		if !sameDay(weeks[i], weeks[i-1]) {
			dedup = append(dedup, weeks[i])
		}
	}

	best := 1
	bestStart := dedup[0]
	current := 1
	currentStart := dedup[0]
	for i := 1; i < len(dedup); i++ {
		if dedup[i].Sub(dedup[i-1]) == 7*24*time.Hour {
			current++
		} else {
			current = 1
			currentStart = dedup[i]
		}
		if current > best {
			best = current
			bestStart = currentStart
		}
	}
	return best, bestStart
}

func sameDay(a, b time.Time) bool {
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
