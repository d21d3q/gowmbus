package frame

import (
	"encoding/binary"
	"fmt"
)

// Telegram represents a decoded Wireless M-Bus frame stripped from transport
// details. The structure will expand as parsing capabilities are added.
type Telegram struct {
	Raw          []byte
	Length       byte
	Control      byte
	Manufacturer uint16
	MeterID      [4]byte
	Version      byte
	DeviceType   byte
	CI           byte
	AccessNumber byte
	Status       byte
	TPL          TPLInfo
	StatusFlags  map[string]bool
	Payload      []byte
}

type TPLInfo struct {
	Present         bool
	AccessField     byte
	StatusField     byte
	Config          uint16
	SecurityMode    byte
	EncryptedBlocks int
}

// Parse extracts the standard short (T1) header from a raw frame.
func Parse(raw []byte) (Telegram, error) {
	if len(raw) < 13 {
		return Telegram{}, fmt.Errorf("telegram too short: %d bytes", len(raw))
	}
	length := raw[0]
	if int(length)+1 != len(raw) {
		return Telegram{}, fmt.Errorf("declared length %d does not match actual length %d", length, len(raw))
	}
	t := Telegram{
		Raw:          raw,
		Length:       length,
		Control:      raw[1],
		Manufacturer: binary.LittleEndian.Uint16(raw[2:4]),
	}
	copy(t.MeterID[:], raw[4:8])
	t.Version = raw[8]
	t.DeviceType = raw[9]
	t.CI = raw[10]
	cursor := 13
	t.AccessNumber = raw[11]
	t.Status = raw[12]
	t.StatusFlags = decodeStatusFlags(t.Status)

	var tpl TPLInfo
	if t.CI == 0x7A {
		if shortTPLPresent(raw, 11) {
			parsed, consumed, err := parseShortTPL(raw, 11)
			if err != nil {
				return Telegram{}, err
			}
			tpl = parsed
			cursor = 11 + consumed
		} else {
			t.AccessNumber = 0
			t.Status = 0
			t.StatusFlags = map[string]bool{}
			cursor = 11
		}
	}
	if cursor > len(raw) {
		return Telegram{}, fmt.Errorf("payload offset %d exceeds telegram length %d", cursor, len(raw))
	}
	t.TPL = tpl
	t.Payload = raw[cursor:]
	return t, nil
}

// MeterIDString returns the EN 13757 display format (MSB first).
func (t Telegram) MeterIDString() string {
	return fmt.Sprintf("%02X%02X%02X%02X", t.MeterID[3], t.MeterID[2], t.MeterID[1], t.MeterID[0])
}

var statusFlagDefs = []struct {
	mask byte
	key  string
}{
	{0x80, "status_empty_pipe"},
	{0x40, "status_reverse_flow"},
	{0x20, "status_freezing"},
	{0x10, "status_temp_alarm"},
	{0x08, "status_perm_alarm"},
	{0x04, "status_battery_alarm"},
	{0x02, "status_hw_alarm"},
}

func decodeStatusFlags(status byte) map[string]bool {
	flags := make(map[string]bool)
	for _, def := range statusFlagDefs {
		if status&def.mask != 0 {
			flags[def.key] = true
		}
	}
	return flags
}

func parseShortTPL(data []byte, offset int) (TPLInfo, int, error) {
	if len(data) < offset+4 {
		return TPLInfo{}, 0, fmt.Errorf("short TPL header truncated")
	}
	tpl := TPLInfo{
		Present:     true,
		AccessField: data[offset],
		StatusField: data[offset+1],
	}
	cfg := binary.LittleEndian.Uint16(data[offset+2 : offset+4])
	tpl.Config = cfg
	tpl.SecurityMode = byte((cfg >> 8) & 0x1F)
	if tpl.SecurityMode == 5 {
		tpl.EncryptedBlocks = int((cfg >> 4) & 0x0F)
	}
	return tpl, 4, nil
}

func shortTPLPresent(data []byte, offset int) bool {
	if len(data) < offset+4 {
		return false
	}
	if data[offset] == 0x2F && data[offset+1] == 0x2F {
		return false
	}
	return true
}
