package ingest

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"meshmonday/internal/checkins"
	"meshmonday/internal/config"
	"meshmonday/internal/meshcore"
	"meshmonday/internal/models"
	"meshmonday/internal/storage"
)

type Service struct {
	cfg         config.Config
	store       *storage.SQLiteStore
	logger      *slog.Logger
	channelKeys []string
}

type RestoreCheckinPacketsResult struct {
	RawScanned       int
	Decoded          int
	CheckinsFound    int
	MondayCheckins   int
	LinksInserted    int
	DecodeErrors     int
	InsertLinkErrors int
}

func NewService(cfg config.Config, store *storage.SQLiteStore, logger *slog.Logger) *Service {
	return &Service{
		cfg:         cfg,
		store:       store,
		logger:      logger,
		channelKeys: cfg.ChannelSecretKeys(),
	}
}

func (s *Service) HandleMessage(ctx context.Context, topic, payloadHex string, observedAt time.Time) {
	iata, deviceKey, ok := parseTopic(topic)
	if !ok {
		s.logger.Warn("invalid topic shape", "topic", topic)
		return
	}
	packetHex, observerKey, ok := decodeIncomingEnvelope(payloadHex, deviceKey)
	if !ok {
		s.logger.Warn("empty packet body", "topic", topic)
		return
	}

	packet, err := meshcore.DecodePacket(packetHex)
	if err != nil {
		s.logger.Warn("decode failed", "topic", topic, "error", err.Error())
		return
	}
	packetHash := meshcore.HashPacket(packetHex)

	inserted, err := s.store.InsertRawPacket(ctx, models.RawPacket{
		PacketHash:      packetHash,
		Topic:           topic,
		IATA:            iata,
		DevicePublicKey: deviceKey,
		PayloadHex:      packetHex,
		PayloadType:     packet.PayloadType,
		PayloadTypeName: packet.PayloadTypeName,
		RouteType:       packet.RouteType,
		RouteTypeName:   packet.RouteTypeName,
		PathLen:         packet.PathLen,
		ObservedAt:      observedAt.UTC(),
		ReceivedAt:      time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("insert raw packet failed", "error", err.Error())
		return
	}
	if _, err := s.store.InsertPacketObservation(ctx, packetHash, observerKey, observedAt.UTC()); err != nil {
		s.logger.Error("insert packet observation failed", "error", err.Error())
	}
	if !inserted {
		s.logger.Debug("duplicate packet ignored", "packet_hash", packetHash)
		return
	}

	candidate, ok := checkins.ExtractFromPacket(packet.PayloadType, packet.PayloadHex, s.channelKeys)
	if !ok {
		return
	}
	if !checkins.IsMondayInTZ(observedAt, s.cfg.TZ) {
		return
	}

	weekStart := checkins.WeekStartMonday(observedAt, s.cfg.TZ)
	checkinDate := observedAt.In(weekStart.Location())
	if _, err := s.store.InsertCheckinPacket(ctx, weekStart, candidate.Username, packetHash, time.Now().UTC()); err != nil {
		s.logger.Error("insert checkin packet link failed", "error", err.Error())
	}
	_, err = s.store.InsertCheckin(ctx, models.Checkin{
		PacketHash:  packetHash,
		Username:    candidate.Username,
		DisplayName: candidate.DisplayName,
		Message:     candidate.Message,
		IATA:        iata,
		CheckinDate: checkinDate,
		WeekStart:   weekStart,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		s.logger.Error("insert checkin failed", "error", err.Error())
	}
}

func (s *Service) RestoreCheckinPacketsFromRaw(ctx context.Context) (RestoreCheckinPacketsResult, error) {
	var result RestoreCheckinPacketsResult
	now := time.Now().UTC()

	err := s.store.ForEachRawPacketForRestore(ctx, func(row storage.RawPacketRestoreRow) error {
		result.RawScanned++

		packet, err := meshcore.DecodePacket(row.PayloadHex)
		if err != nil {
			result.DecodeErrors++
			return nil
		}
		result.Decoded++

		candidate, ok := checkins.ExtractFromPacket(packet.PayloadType, packet.PayloadHex, s.channelKeys)
		if !ok {
			return nil
		}
		result.CheckinsFound++

		if !checkins.IsMondayInTZ(row.ObservedAt, s.cfg.TZ) {
			return nil
		}
		result.MondayCheckins++

		weekStart := checkins.WeekStartMonday(row.ObservedAt, s.cfg.TZ)
		inserted, err := s.store.InsertCheckinPacket(ctx, weekStart, candidate.Username, row.PacketHash, now)
		if err != nil {
			result.InsertLinkErrors++
			return nil
		}
		if inserted {
			result.LinksInserted++
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

func parseTopic(topic string) (iata, deviceKey string, ok bool) {
	parts := strings.Split(strings.TrimSpace(topic), "/")
	if len(parts) != 4 {
		return "", "", false
	}
	if parts[0] != "meshcore" || parts[3] != "packets" {
		return "", "", false
	}
	return strings.ToUpper(parts[1]), parts[2], true
}

type packetEnvelope struct {
	Origin string `json:"origin"`
	Raw    string `json:"raw"`
}

func decodeIncomingEnvelope(payload, fallbackObserver string) (packetHex, observer string, ok bool) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return "", "", false
	}

	observer = fallbackObserver
	if strings.HasPrefix(trimmed, "{") {
		var env packetEnvelope
		if err := json.Unmarshal([]byte(trimmed), &env); err == nil {
			if raw := strings.TrimSpace(env.Raw); raw != "" {
				packetHex = raw
			}
			if name := strings.TrimSpace(env.Origin); name != "" {
				observer = name
			}
		}
	}
	if packetHex == "" {
		packetHex = trimmed
	}
	return packetHex, observer, true
}
