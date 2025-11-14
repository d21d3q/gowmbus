package hydrodigit

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/d21d3q/gowmbus/internal/frame"
)

func TestLegacyManufacturerBlock(t *testing.T) {
	raw := mustLoadHex(t, filepath.Join("..", "..", "..", "testdata", "hydrodigit", "hydrodigit_water.hex"))
	tg, err := frame.Parse(raw)
	if err != nil {
		t.Fatalf("frame.Parse: %v", err)
	}
	readings, block, err := parseStandardReadings(tg.Payload)
	if err != nil {
		t.Fatalf("parseStandardReadings: %v", err)
	}
	data, err := ParseManufacturerData(block, readings.VolumeScale)
	if err != nil {
		t.Fatalf("ParseManufacturerData: %v", err)
	}
	if data.Variant != "legacy" {
		t.Fatalf("expected legacy variant, got %s", data.Variant)
	}
	if data.Contents != "Backflow, alarms and monthly data" {
		t.Fatalf("unexpected contents: %s", data.Contents)
	}
	if data.Voltage < 3.69 || data.Voltage > 3.71 {
		t.Fatalf("unexpected voltage %.2f", data.Voltage)
	}
	wantMonths := map[string]float64{
		"January":   1.93,
		"April":     2.53,
		"September": 3.60,
		"December":  1.79,
	}
	for month, want := range wantMonths {
		got, ok := data.MonthlyTotals[month]
		if !ok {
			t.Fatalf("missing month %s", month)
		}
		if diff := abs(got - want); diff > 0.01 {
			t.Fatalf("month %s mismatch: got %.2f want %.2f", month, got, want)
		}
	}
}

func TestLegacyLeakDate(t *testing.T) {
	raw := mustLoadHex(t, filepath.Join("..", "..", "..", "testdata", "hydrodigit", "hydro4.hex"))
	tg, err := frame.Parse(raw)
	if err != nil {
		t.Fatalf("frame.Parse: %v", err)
	}
	readings, block, err := parseStandardReadings(tg.Payload)
	if err != nil {
		t.Fatalf("parseStandardReadings: %v", err)
	}
	data, err := ParseManufacturerData(block, readings.VolumeScale)
	if err != nil {
		t.Fatalf("ParseManufacturerData: %v", err)
	}
	if data.LeakDate != "25.04.2024" {
		t.Fatalf("unexpected leak date %s", data.LeakDate)
	}
	if data.BackflowM3 < 0.006 || data.BackflowM3 > 0.008 {
		t.Fatalf("unexpected backflow %.3f", data.BackflowM3)
	}
}

func TestExtendedManufacturerBlock(t *testing.T) {
	raw := mustLoadHex(t, filepath.Join("..", "..", "..", "testdata", "hydrodigit", "hydrolink_worked_example.hex"))
	tg, err := frame.Parse(raw)
	if err != nil {
		t.Fatalf("frame.Parse: %v", err)
	}
	readings, block, err := parseStandardReadings(tg.Payload)
	if err != nil {
		t.Fatalf("parseStandardReadings: %v", err)
	}
	data, err := ParseManufacturerData(block, readings.VolumeScale)
	if err != nil {
		t.Fatalf("ParseManufacturerData: %v", err)
	}
	if data.Variant != "extended" {
		t.Fatalf("expected extended variant, got %s", data.Variant)
	}
	if data.BatteryPercentRaw != 0x84 {
		t.Fatalf("unexpected battery raw %02X", data.BatteryPercentRaw)
	}
	if data.BatteryPercentClamped != 100 {
		t.Fatalf("expected clamped battery = 100, got %d", data.BatteryPercentClamped)
	}
	if data.ErrorBits != 0x290357 {
		t.Fatalf("unexpected error bits 0x%06X", data.ErrorBits)
	}
	if data.MSByte != 0x5D {
		t.Fatalf("unexpected MSByte 0x%02X", data.MSByte)
	}
}

func mustLoadHex(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	clean := bytesTrimSpace(data)
	decoded := make([]byte, hex.DecodedLen(len(clean)))
	n, err := hex.Decode(decoded, clean)
	if err != nil {
		t.Fatalf("hex decode %s: %v", rel, err)
	}
	return decoded[:n]
}

func bytesTrimSpace(b []byte) []byte {
	var out []byte
	out = make([]byte, 0, len(b))
	for _, c := range b {
		switch c {
		case '\n', '\r', ' ', '\t':
			continue
		default:
			out = append(out, c)
		}
	}
	return out
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
