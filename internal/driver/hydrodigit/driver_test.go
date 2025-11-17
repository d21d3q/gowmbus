package hydrodigit

import (
	"context"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/d21d3q/gowmbus/internal/frame"
)

func TestDriverProcess(t *testing.T) {
	raw := mustReadHex(t, filepath.Join("..", "..", "..", "testdata", "hydrodigit", "hydrodigit_water.hex"))
	tg, err := frame.Parse(raw)
	if err != nil {
		t.Fatalf("frame.Parse: %v", err)
	}
	fields, err := (Driver{}).Process(context.Background(), &tg)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if fields["id"] != "86868686" {
		t.Fatalf("unexpected id: %v", fields["id"])
	}
	if total, ok := fields["total_m3"].(float64); !ok || math.Abs(total-3.866) > 0.001 {
		t.Fatalf("unexpected total_m3: %v", fields["total_m3"])
	}
	if month, ok := fields["April_total_m3"].(float64); !ok || math.Abs(month-2.53) > 0.01 {
		t.Fatalf("unexpected April_total_m3: %v", fields["April_total_m3"])
	}
}

func mustReadHex(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	clean := bytesTrimSpace(data)
	buf := make([]byte, hex.DecodedLen(len(clean)))
	n, err := hex.Decode(buf, clean)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return buf[:n]
}
