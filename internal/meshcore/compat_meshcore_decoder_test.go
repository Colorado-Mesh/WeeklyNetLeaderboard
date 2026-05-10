package meshcore

import "testing"

func TestCompatibilityWithMichaelHartVectors(t *testing.T) {
	cases := []struct {
		name            string
		hexData         string
		wantRouteType   int
		wantPayloadType int
		wantPathLen     int
		wantPathHex     string
		wantPayloadName string
	}{
		{
			name:            "response packet structure fixture",
			hexData:         "0600DE1FDFCAD56E6C38B756FEE81C24199C6043AC5B",
			wantRouteType:   2,
			wantPayloadType: 1,
			wantPathLen:     0,
			wantPathHex:     "",
			wantPayloadName: "RESPONSE",
		},
		{
			name:            "group text fixture",
			hexData:         "150011C3C1354D619BAE9590E4D177DB7EEAF982F5BDCF78005D75157D9535FA90178F785D",
			wantRouteType:   1,
			wantPayloadType: 5,
			wantPathLen:     0,
			wantPathHex:     "",
			wantPayloadName: "GROUP_TEXT",
		},
		{
			name:            "text message fixture",
			hexData:         "09046F17C47ED00A13E16AB5B94B1CC2D1A5059C6E5A6253C60D",
			wantRouteType:   1,
			wantPayloadType: 2,
			wantPathLen:     4,
			wantPathHex:     "6f17c47e",
			wantPayloadName: "TEXT_MESSAGE",
		},
		{
			name:            "advert fixture",
			hexData:         "11007E7662676F7F0850A8A355BAAFBFC1EB7B4174C340442D7D7161C9474A2C94006CE7CF682E58408DD8FCC51906ECA98EBF94A037886BDADE7ECD09FD92B839491DF3809C9454F5286D1D3370AC31A34593D569E9A042A3B41FD331DFFB7E18599CE1E60992A076D50238C5B8F85757375354522F50756765744D65736820436F75676172",
			wantRouteType:   1,
			wantPayloadType: 4,
			wantPathLen:     0,
			wantPathHex:     "",
			wantPayloadName: "ADVERT",
		},
		{
			name:            "trace fixture",
			hexData:         "260130A24D89BD0000000000FB",
			wantRouteType:   2,
			wantPayloadType: 9,
			wantPathLen:     1,
			wantPathHex:     "30",
			wantPayloadName: "TRACE",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packet, err := DecodePacket(tc.hexData)
			if err != nil {
				t.Fatalf("decode err: %v", err)
			}
			if packet.RouteType != tc.wantRouteType {
				t.Fatalf("route type mismatch: got %d want %d", packet.RouteType, tc.wantRouteType)
			}
			if packet.PayloadType != tc.wantPayloadType {
				t.Fatalf("payload type mismatch: got %d want %d", packet.PayloadType, tc.wantPayloadType)
			}
			if packet.PathLen != tc.wantPathLen {
				t.Fatalf("path len mismatch: got %d want %d", packet.PathLen, tc.wantPathLen)
			}
			if packet.PathHex != tc.wantPathHex {
				t.Fatalf("path hex mismatch: got %q want %q", packet.PathHex, tc.wantPathHex)
			}
			if packet.PayloadTypeName != tc.wantPayloadName {
				t.Fatalf("payload name mismatch: got %q want %q", packet.PayloadTypeName, tc.wantPayloadName)
			}
		})
	}
}

func TestTransportCodesDecode(t *testing.T) {
	// Header: routeType=TRANSPORT_FLOOD(0), payloadType=RESPONSE(1), version=0 => 0x04
	packet, err := DecodePacket("043412785600aabb")
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if packet.TransportCodes == nil {
		t.Fatal("expected transport codes")
	}
	if packet.TransportCodes[0] != 0x1234 || packet.TransportCodes[1] != 0x5678 {
		t.Fatalf("unexpected transport codes: %#v", *packet.TransportCodes)
	}
	if packet.PathLen != 0 || packet.PayloadHex != "aabb" {
		t.Fatalf("unexpected parsed body: path=%d payload=%s", packet.PathLen, packet.PayloadHex)
	}
}
