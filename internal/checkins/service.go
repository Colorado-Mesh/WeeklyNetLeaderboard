package checkins

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	wordPattern         = regexp.MustCompile(`[A-Za-z0-9_]+`)
)

const payloadTypeGroupText = 0x05

type Candidate struct {
	Username    string
	DisplayName string
	Message     string
}

func ExtractFromPacket(payloadType int, payloadHex string, channelKeys []string) (Candidate, bool) {
	if payloadType == payloadTypeGroupText {
		if sender, message, ok := decryptGroupTextPayload(payloadHex, channelKeys); ok {
			message = strings.TrimSpace(message)
			if sender != "" {
				if !hasMeshMondayTag(message) {
					return Candidate{}, false
				}
				return Candidate{
					Username:    normalizeUsername(sender),
					DisplayName: strings.TrimSpace(sender),
					Message:     message,
				}, true
			}
			if full := strings.TrimSpace(message); full != "" {
				return parseCandidateFromText(full)
			}
		}
	}
	return ExtractFromPayload(payloadHex)
}

func ExtractFromPayload(payloadHex string) (Candidate, bool) {
	raw, err := hex.DecodeString(strings.TrimSpace(payloadHex))
	if err != nil {
		return Candidate{}, false
	}

	// Try JSON first in case observers publish already-decoded payloads.
	var jsonBody map[string]any
	if err := json.Unmarshal(raw, &jsonBody); err == nil {
		name := stringValue(jsonBody, "username", "sender", "name")
		message := stringValue(jsonBody, "message", "text", "body")
		if name != "" && message != "" && hasMeshMondayTag(message) {
			u := normalizeUsername(name)
			return Candidate{
				Username:    u,
				DisplayName: strings.TrimSpace(name),
				Message:     strings.TrimSpace(message),
			}, true
		}
	}

	text := sanitizeText(string(raw))
	if text == "" {
		return Candidate{}, false
	}
	return parseCandidateFromText(text)
}

func IsMondayInTZ(t time.Time, tz string) bool {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Weekday() == time.Monday
}

func WeekStartMonday(t time.Time, tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	local := t.In(loc)
	delta := (int(local.Weekday()) + 6) % 7
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return dayStart.AddDate(0, 0, -delta)
}

func normalizeUsername(input string) string {
	clean := strings.ToLower(strings.TrimSpace(input))
	parts := wordPattern.FindAllString(clean, -1)
	if len(parts) == 0 {
		return "anonymous"
	}
	return strings.Join(parts, "_")
}

func sanitizeText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) > 280 {
		return strings.TrimSpace(string(runes[:280]))
	}
	return value
}

func stringValue(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := body[key]; ok {
			s, ok := v.(string)
			if ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func hasMeshMondayTag(message string) bool {
	return strings.Contains(strings.ToLower(message), "#meshmonday")
}

func parseCandidateFromText(text string) (Candidate, bool) {
	display, msg := parseSenderAndMessage(text)
	if display != "" {
		if !hasMeshMondayTag(msg) {
			return Candidate{}, false
		}
		return Candidate{
			Username:    normalizeUsername(display),
			DisplayName: display,
			Message:     msg,
		}, true
	}
	return Candidate{}, false
}

func decryptGroupTextPayload(payloadHex string, channelKeys []string) (sender string, message string, ok bool) {
	payloadBytes, err := hex.DecodeString(strings.TrimSpace(payloadHex))
	if err != nil || len(payloadBytes) < 3 {
		return "", "", false
	}

	channelHash := payloadBytes[0]
	cipherMAC := payloadBytes[1:3]
	ciphertext := payloadBytes[3:]
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", "", false
	}

	for _, keyHex := range channelKeys {
		key, err := hex.DecodeString(strings.TrimSpace(keyHex))
		if err != nil || len(key) != 16 {
			continue
		}
		hash := sha256.Sum256(key)
		if hash[0] != channelHash {
			continue
		}
		if !verifyMeshcoreMAC(key, ciphertext, cipherMAC) {
			continue
		}
		plaintext, err := decryptAES128ECB(key, ciphertext)
		if err != nil || len(plaintext) < 5 {
			continue
		}
		text := decodeNullTerminatedUTF8(plaintext[5:])
		if text == "" {
			continue
		}
		sender, message = parseSenderAndMessage(text)
		if message == "" {
			message = text
		}
		return sender, message, true
	}
	return "", "", false
}

func verifyMeshcoreMAC(key16, ciphertext, expected []byte) bool {
	if len(expected) < 2 {
		return false
	}
	channelSecret := make([]byte, 32)
	copy(channelSecret[:16], key16)

	mac := hmac.New(sha256.New, channelSecret)
	mac.Write(ciphertext)
	sum := mac.Sum(nil)
	return len(sum) >= 2 && sum[0] == expected[0] && sum[1] == expected[1]
}

func decryptAES128ECB(key16, ciphertext []byte) ([]byte, error) {
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not block aligned")
	}
	block, err := aes.NewCipher(key16)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += aes.BlockSize {
		block.Decrypt(out[i:i+aes.BlockSize], ciphertext[i:i+aes.BlockSize])
	}
	return out, nil
}

func decodeNullTerminatedUTF8(raw []byte) string {
	idx := len(raw)
	for i, b := range raw {
		if b == 0 {
			idx = i
			break
		}
	}
	content := raw[:idx]
	if !utf8.Valid(content) {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func parseSenderAndMessage(raw string) (sender string, message string) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", ""
	}
	if idx := strings.Index(text, ": "); idx > 0 && idx < 80 {
		potentialSender := strings.TrimSpace(text[:idx])
		if potentialSender != "" && !strings.ContainsAny(potentialSender, "[]:") {
			return potentialSender, strings.TrimSpace(text[idx+2:])
		}
	}
	return "", text
}
