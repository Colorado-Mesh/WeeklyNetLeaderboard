package meshcore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	PayloadVersionV1 = 0

	RouteTypeFlood  = 0
	RouteTypeDirect = 3
)

var payloadTypeNames = map[int]string{
	0x00: "REQUEST",
	0x01: "RESPONSE",
	0x02: "TEXT_MESSAGE",
	0x03: "ACK",
	0x04: "ADVERT",
	0x05: "GROUP_TEXT",
	0x06: "GROUP_DATA",
	0x07: "ANON_REQUEST",
	0x08: "PATH",
	0x09: "TRACE",
	0x0A: "MULTIPART",
	0x0B: "CONTROL",
	0x0F: "RAW_CUSTOM",
}

var routeTypeNames = map[int]string{
	0: "TRANSPORT_FLOOD",
	1: "FLOOD",
	2: "DIRECT",
	3: "TRANSPORT_DIRECT",
}

type Packet struct {
	Header          byte
	PayloadVersion  int
	PayloadType     int
	PayloadTypeName string
	RouteType       int
	RouteTypeName   string
	TransportCodes  *[2]uint16
	PathLen         int
	BytesPerHop     int
	PathByteLength  int
	Path            []string
	PathHex         string
	PayloadHex      string
	PayloadBytes    []byte
}

func DecodePathLenByte(value byte) (pathLen int, bytesPerHop int, err error) {
	pathLen = int(value & 0x3F)
	sizeCode := int((value >> 6) & 0x03)
	bytesPerHop = sizeCode + 1
	if bytesPerHop < 1 || bytesPerHop > 3 {
		return 0, 0, fmt.Errorf("invalid bytes per hop: %d", bytesPerHop)
	}
	return pathLen, bytesPerHop, nil
}

func DecodePacket(payloadHex string) (*Packet, error) {
	normalized := strings.TrimPrefix(strings.TrimSpace(payloadHex), "0x")
	if normalized == "" {
		return nil, errors.New("empty payload hex")
	}
	raw, err := hex.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}
	if len(raw) < 2 {
		return nil, errors.New("packet too short")
	}

	offset := 0
	header := raw[offset]
	offset++

	payloadVersion := int((header >> 6) & 0x03)
	if payloadVersion != PayloadVersionV1 {
		return nil, fmt.Errorf("unsupported payload version %d", payloadVersion)
	}
	payloadType := int((header >> 2) & 0x0F)
	routeType := int(header & 0x03)
	var transportCodes *[2]uint16

	if routeType == RouteTypeFlood || routeType == RouteTypeDirect {
		if len(raw) < offset+4 {
			return nil, errors.New("missing transport codes")
		}
		code1 := uint16(raw[offset]) | (uint16(raw[offset+1]) << 8)
		code2 := uint16(raw[offset+2]) | (uint16(raw[offset+3]) << 8)
		transportCodes = &[2]uint16{code1, code2}
		offset += 4
	}

	if len(raw) <= offset {
		return nil, errors.New("missing path length byte")
	}
	pathLen, bytesPerHop, err := DecodePathLenByte(raw[offset])
	if err != nil {
		return nil, err
	}
	offset++

	pathByteLength := pathLen * bytesPerHop
	if len(raw) < offset+pathByteLength {
		return nil, errors.New("path bytes out of bounds")
	}

	pathBytes := raw[offset : offset+pathByteLength]
	offset += pathByteLength
	payloadBytes := raw[offset:]
	groupedPath := make([]string, 0, pathLen)
	for i := 0; i < pathLen; i++ {
		start := i * bytesPerHop
		end := start + bytesPerHop
		groupedPath = append(groupedPath, hex.EncodeToString(pathBytes[start:end]))
	}

	packet := &Packet{
		Header:          header,
		PayloadVersion:  payloadVersion,
		PayloadType:     payloadType,
		PayloadTypeName: payloadTypeName(payloadType),
		RouteType:       routeType,
		RouteTypeName:   routeTypeName(routeType),
		TransportCodes:  transportCodes,
		PathLen:         pathLen,
		BytesPerHop:     bytesPerHop,
		PathByteLength:  pathByteLength,
		Path:            groupedPath,
		PathHex:         hex.EncodeToString(pathBytes),
		PayloadHex:      hex.EncodeToString(payloadBytes),
		PayloadBytes:    payloadBytes,
	}
	return packet, nil
}

func HashPacket(payloadHex string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(payloadHex))))
	return hex.EncodeToString(sum[:])
}

func payloadTypeName(value int) string {
	name, ok := payloadTypeNames[value]
	if !ok {
		return fmt.Sprintf("UNKNOWN_%d", value)
	}
	return name
}

func routeTypeName(value int) string {
	name, ok := routeTypeNames[value]
	if !ok {
		return fmt.Sprintf("UNKNOWN_%d", value)
	}
	return name
}
