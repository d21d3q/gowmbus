package frame

import (
	"encoding/hex"
	"testing"
)

func TestParse(t *testing.T) {
	raw := decodeHex(t, "4E44B4098686868613077AF00040052F2F0C1366380000046D27287E2A0F150E00000000C10000D10000E60000FD00000C01002F0100410100540100680100890000A00000B30000002F2F2F2F2F2F")
	tg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tg.Manufacturer != 0x09B4 {
		t.Fatalf("manufacturer mismatch: %04X", tg.Manufacturer)
	}
	if got := tg.MeterIDString(); got != "86868686" {
		t.Fatalf("meter id mismatch: %s", got)
	}
	if tg.CI != 0x7A {
		t.Fatalf("unexpected CI 0x%02X", tg.CI)
	}
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	return b
}
