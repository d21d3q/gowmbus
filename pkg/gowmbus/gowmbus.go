package gowmbus

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"gitlab.com/d21d3q/gowmbus/internal/crypto"
	"gitlab.com/d21d3q/gowmbus/internal/driver"
	_ "gitlab.com/d21d3q/gowmbus/internal/driver/hydrocalm4" // register driver
	_ "gitlab.com/d21d3q/gowmbus/internal/driver/hydrodigit" // register driver
	"gitlab.com/d21d3q/gowmbus/internal/frame"
)

// Result captures the outcome of AnalyzeHex.
type Result struct {
	Driver    string
	RawHex    string
	ByteCount int
	Telegram  *frame.Telegram
	Fields    map[string]any
}

// String renders a human-readable representation of the result.
func (r Result) String() string {
	summary := map[string]any{
		"driver":     r.Driver,
		"byte_count": r.ByteCount,
		"raw_hex":    r.RawHex,
	}
	if r.Telegram != nil {
		summary["meter_id"] = r.Telegram.MeterIDString()
		summary["manufacturer"] = fmt.Sprintf("0x%04X", r.Telegram.Manufacturer)
		summary["ci"] = fmt.Sprintf("0x%02X", r.Telegram.CI)
	}
	if len(r.Fields) > 0 {
		summary["fields"] = r.Fields
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Sprintf("driver: %s bytes:%d raw:%s (marshal error: %v)", r.Driver, r.ByteCount, r.RawHex, err)
	}
	return string(data)
}

// AnalyzeHex parses the frame, selects a driver, and returns decoded data.
func AnalyzeHex(ctx context.Context, raw string) (Result, error) {
	return AnalyzeHexWithOptions(ctx, raw, AnalyzeOptions{})
}

// AnalyzeHexWithOptions parses the frame with custom options.
func AnalyzeHexWithOptions(ctx context.Context, raw string, opts AnalyzeOptions) (Result, error) {
	ctxWithKey, key, err := opts.toInternal(ctx)
	if err != nil {
		return Result{}, err
	}
	data, err := decodeHex(raw)
	if err != nil {
		return Result{}, err
	}
	telegram, err := frame.Parse(data)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Driver:    "unknown",
		RawHex:    strings.ToUpper(stripWhitespace(raw)),
		ByteCount: len(data),
		Telegram:  &telegram,
	}

	drv, err := driver.Lookup(&telegram)
	if err != nil {
		return result, nil
	}
	if err := crypto.Decrypt(&telegram, key); err != nil {
		if errors.Is(err, crypto.ErrKeyRequired) {
			if reporter, ok := drv.(driver.PartialReporter); ok {
				fields := reporter.PartialFields(&telegram)
				fields["encryption"] = err.Error()
				result.Driver = drv.Name()
				result.Fields = fields
				return result, nil
			}
		}
		return result, err
	}

	fields, err := drv.Process(ctxWithKey, &telegram)
	if err != nil {
		if reporter, ok := drv.(driver.PartialReporter); ok {
			partial := reporter.PartialFields(&telegram)
			partial["error"] = err.Error()
			result.Driver = drv.Name()
			result.Fields = partial
			return result, nil
		}
		return result, err
	}
	result.Driver = drv.Name()
	result.Fields = fields
	return result, nil
}

func decodeHex(input string) ([]byte, error) {
	clean := stripWhitespace(input)
	if strings.HasPrefix(clean, "0X") {
		clean = clean[2:]
	}
	if len(clean)%2 != 0 {
		return nil, fmt.Errorf("hex telegram must contain an even number of digits, got %d", len(clean))
	}
	decoded := make([]byte, len(clean)/2)
	if _, err := hex.Decode(decoded, []byte(clean)); err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}
	return decoded, nil
}

func stripWhitespace(s string) string {
	builder := strings.Builder{}
	builder.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) || r == '|' || r == '_' {
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
