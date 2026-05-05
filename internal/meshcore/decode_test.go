package meshcore

import "testing"

func TestDecodePathLenByte(t *testing.T) {
	pathLen, bytesPerHop, err := DecodePathLenByte(0x82)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if pathLen != 2 {
		t.Fatalf("expected pathLen 2, got %d", pathLen)
	}
	if bytesPerHop != 3 {
		t.Fatalf("expected bytesPerHop 3, got %d", bytesPerHop)
	}
}

func TestDecodePacketTXT(t *testing.T) {
	packet, err := DecodePacket("110041766572793a206869")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if packet.PayloadTypeName != "ADVERT" {
		t.Fatalf("expected ADVERT, got %s", packet.PayloadTypeName)
	}
	if packet.RouteTypeName != "FLOOD" {
		t.Fatalf("expected FLOOD, got %s", packet.RouteTypeName)
	}
}
