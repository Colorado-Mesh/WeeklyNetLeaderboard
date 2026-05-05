package mqtt

import (
	"context"
	"log/slog"
	"testing"
)

func TestBuildMessageDropsOversizedPayload(t *testing.T) {
	c := &Consumer{
		logger:          slog.Default(),
		maxPayloadBytes: 8,
	}
	if _, ok := c.buildMessage("meshcore/SEA/device/packets", []byte("123456789")); ok {
		t.Fatal("expected oversized payload to be dropped")
	}
}

func TestBuildMessageAcceptsPayloadAtLimit(t *testing.T) {
	c := &Consumer{
		logger:          slog.Default(),
		maxPayloadBytes: 8,
	}
	msg, ok := c.buildMessage("meshcore/SEA/device/packets", []byte("12345678"))
	if !ok {
		t.Fatal("expected payload at limit to pass")
	}
	if msg.PayloadHex != "12345678" {
		t.Fatalf("unexpected payload: %q", msg.PayloadHex)
	}
}

func TestSubscribeCachesSubscriptionsForReconnect(t *testing.T) {
	c := NewConsumer(slog.Default(), "tcp://127.0.0.1:1883", "test-client", "", "", 1024)

	err := c.Subscribe("meshcore/+/+/packets", func(context.Context, Message) {})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(c.subs) != 1 {
		t.Fatalf("expected 1 cached subscription, got %d", len(c.subs))
	}
}
