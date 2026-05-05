package config

import "testing"

func TestChannelSecretKeysIncludesPublicDerivedAndPrivate(t *testing.T) {
	cfg := Config{
		HashtagChannels:   []string{"#bot", "seattle"},
		PrivateChannelKeys: []string{"00112233445566778899aabbccddeeff"},
	}
	keys := cfg.ChannelSecretKeys()

	want := map[string]bool{
		"8b3387e9c5cdea6ac9e5edbaa115cd72": true, // public fixed key
		"eb50a1bcb3e4e5d7bf69a57c9dada211": true, // #bot
		"ef627a9bbbb549347fdb76bf0cd3bc14": true, // #seattle
		"00112233445566778899aabbccddeeff": true, // private key
	}

	for _, key := range keys {
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing expected keys: %v", want)
	}
}

func TestValidateDiceBearStyle(t *testing.T) {
	cfg := Config{
		MeshName:          "CascadiaMesh",
		DiceBearStyle:     "rings",
		UIPollSeconds:     15,
		IATADefault:       "SEA",
		MQTTTopicTemplate: "meshcore/+/+/packets",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cfg.DiceBearStyle = "not a style"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid DICEBEAR_STYLE to fail validation")
	}

	cfg.DiceBearStyle = "adventurer"
	cfg.UIPollSeconds = -1
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected negative UI_POLL_SECONDS to fail validation")
	}
}

func TestParseIATAFilters(t *testing.T) {
	got := parseIATAFilters("ALL", "SEA")
	if got != nil {
		t.Fatalf("expected ALL to disable filters, got %v", got)
	}

	got = parseIATAFilters("sea,pdx,yvr", "SEA")
	want := []string{"SEA", "PDX", "YVR"}
	if len(got) != len(want) {
		t.Fatalf("unexpected filter len: got=%d want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected filter at %d: got=%s want=%s", i, got[i], want[i])
		}
	}

	got = parseIATAFilters("", "SEA")
	if len(got) != 1 || got[0] != "SEA" {
		t.Fatalf("expected fallback SEA filter, got %v", got)
	}
}

