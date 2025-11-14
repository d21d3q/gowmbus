package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadJSON loads a JSON fixture from testdata relative to the repo root.
func LoadJSON(t *testing.T, rel string, v any) {
	t.Helper()
	data := readTestdata(t, rel)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode %s: %v", rel, err)
	}
}

// LoadHex returns a trimmed hex string from testdata relative path.
func LoadHex(t *testing.T, rel string) string {
	t.Helper()
	data := readTestdata(t, rel)
	return strings.TrimSpace(string(data))
}

func readTestdata(t *testing.T, rel string) []byte {
	t.Helper()
	candidates := []string{
		filepath.Join("testdata", rel),
		filepath.Join("..", "testdata", rel),
		filepath.Join("..", "..", "testdata", rel),
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return data
		}
	}
	t.Fatalf("unable to locate testdata file %s", rel)
	return nil
}
