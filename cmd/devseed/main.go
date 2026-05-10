package main

import (
	"context"
	"fmt"
	"time"

	"weeklynet/internal/checkins"
	"weeklynet/internal/config"
	"weeklynet/internal/meshcore"
	"weeklynet/internal/models"
	"weeklynet/internal/storage"
)

func main() {
	_ = config.LoadEnvFile(".env.local")
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	store, err := storage.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			panic(err)
		}
	}()

	now := time.Now()
	users := []string{"Avery", "Blake", "Casey", "Drew", "Emerson", "Finley", "Gray", "Harper", "Indigo", "Jordan", "Kai", "Logan"}
	for i, name := range users {
		weeksAgo := i % 6
		observed := checkins.WeekStartForWeekday(now.AddDate(0, 0, -(weeksAgo*7)), cfg.TZ, cfg.DayOfWeek).Add(9 * time.Hour)
		packetHash := meshcore.HashPacket(fmt.Sprintf("%s-%s", name, observed.Format(time.RFC3339)))
		topic := fmt.Sprintf("meshcore/%s/%s/packets", cfg.IATADefault, "seed-device")
		_, _ = store.InsertRawPacket(context.Background(), models.RawPacket{
			PacketHash:      packetHash,
			Topic:           topic,
			IATA:            cfg.IATADefault,
			DevicePublicKey: "seed-device",
			PayloadHex:      "040041766572793a20436865636b696e21",
			PayloadType:     4,
			PayloadTypeName: "TXT_MSG",
			RouteType:       0,
			RouteTypeName:   "TRANSPORT_FLOOD",
			PathLen:         0,
			ObservedAt:      observed,
			ReceivedAt:      observed,
		})
		_, _ = store.InsertCheckin(context.Background(), models.Checkin{
			PacketHash:  packetHash,
			Username:    normalize(name),
			DisplayName: name,
			Message:     "Checking in from seed data",
			IATA:        cfg.IATADefault,
			CheckinDate: observed,
			WeekStart:   checkins.WeekStartForWeekday(observed, cfg.TZ, cfg.DayOfWeek),
			CreatedAt:   time.Now().UTC(),
		})
	}
	fmt.Println("seed data inserted")
}

func normalize(name string) string {
	out := []rune{}
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+'a'-'A')
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		}
	}
	return string(out)
}
