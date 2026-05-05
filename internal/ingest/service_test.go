package ingest

import (
	"context"
	"errors"
	"testing"
)

func TestDecodeIncomingEnvelopeRejectsInvalidJSONEnvelope(t *testing.T) {
	_, _, ok := decodeIncomingEnvelope("{invalid", "observer", 1024, 64)
	if ok {
		t.Fatal("expected invalid JSON envelope to be rejected")
	}
}

func TestDecodeIncomingEnvelopeRejectsOversizedObserver(t *testing.T) {
	_, _, ok := decodeIncomingEnvelope(`{"origin":"this-observer-name-is-way-too-long","raw":"aa11"}`, "fallback", 1024, 8)
	if ok {
		t.Fatal("expected oversized origin to be rejected")
	}
}

func TestDecodeIncomingEnvelopeAcceptsValidEnvelope(t *testing.T) {
	packetHex, observer, ok := decodeIncomingEnvelope(`{"origin":"Krzychu-RPT","raw":"aa11"}`, "fallback", 1024, 64)
	if !ok {
		t.Fatal("expected valid envelope")
	}
	if packetHex != "aa11" {
		t.Fatalf("unexpected packet hex: %q", packetHex)
	}
	if observer != "Krzychu-RPT" {
		t.Fatalf("unexpected observer: %q", observer)
	}
}

func TestWithBusyRetryRetriesTransientBusyErrors(t *testing.T) {
	attempts := 0
	err := withBusyRetry(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("database is locked (5) (SQLITE_BUSY)")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestWithBusyRetryDoesNotRetryNonBusyErrors(t *testing.T) {
	attempts := 0
	err := withBusyRetry(context.Background(), func() error {
		attempts++
		return errors.New("permanent error")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt for non-busy error, got %d", attempts)
	}
}
