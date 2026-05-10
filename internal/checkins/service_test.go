package checkins

import (
	"testing"
	"time"
)

func TestExtractFromPayload(t *testing.T) {
	c, ok := ExtractFromPayload("41766572793a2068656c6c6f206d65736820236d6573686d6f6e646179", "#weeklynet")
	if !ok {
		t.Fatal("expected extraction success")
	}
	if c.Username != "avery" {
		t.Fatalf("expected avery, got %s", c.Username)
	}
}

func TestExtractFromPayloadRequiresWeeklyNetTag(t *testing.T) {
	_, ok := ExtractFromPayload("41766572793a2068656c6c6f206d657368", "#weeklynet")
	if ok {
		t.Fatal("expected extraction to fail without #weeklynet tag")
	}
}

func TestExtractFromPayloadAllowsUnicodeAndSymbolsInUsername(t *testing.T) {
	// "TR&RS🎸: Marvelous #WeeklyNet!"
	payloadHex := "5452265253f09f8eb83a204d617276656c6f757320234d6573684d6f6e64617921"
	c, ok := ExtractFromPayload(payloadHex, "#weeklynet")
	if !ok {
		t.Fatal("expected extraction success for unicode/symbol sender")
	}
	if c.DisplayName != "TR&RS🎸" {
		t.Fatalf("expected display name TR&RS🎸, got %q", c.DisplayName)
	}
	if c.Username != "tr_rs" {
		t.Fatalf("expected normalized username tr_rs, got %q", c.Username)
	}
}

func TestDecryptGroupTextPayloadPublicChannelVector(t *testing.T) {
	payloadHex := "11C3C1354D619BAE9590E4D177DB7EEAF982F5BDCF78005D75157D9535FA90178F785D"
	sender, message, ok := decryptGroupTextPayload(payloadHex, []string{"8b3387e9c5cdea6ac9e5edbaa115cd72"})
	if !ok {
		t.Fatal("expected decrypt success")
	}
	if sender != "🌲 Tree" {
		t.Fatalf("expected sender 🌲 Tree, got %q", sender)
	}
	if message != "☁️" {
		t.Fatalf("expected message ☁️, got %q", message)
	}
}

func TestWeekdayHelpers(t *testing.T) {
	tm := time.Date(2026, 5, 4, 16, 0, 0, 0, time.UTC)
	if !IsWeekdayInTZ(tm, "America/Los_Angeles", time.Monday) {
		t.Fatal("expected monday true")
	}
	if IsWeekdayInTZ(tm, "America/Los_Angeles", time.Tuesday) {
		t.Fatal("expected tuesday false")
	}
	weekStart := WeekStartForWeekday(tm, "America/Los_Angeles", time.Monday)
	if weekStart.Weekday() != time.Monday {
		t.Fatalf("expected monday, got %s", weekStart.Weekday())
	}
}
