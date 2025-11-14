package options

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

type contextKey struct{}

// WithSecurityKey stores the provided key inside the context.
func WithSecurityKey(ctx context.Context, key []byte) context.Context {
	if len(key) == 0 {
		return ctx
	}
	buf := make([]byte, len(key))
	copy(buf, key)
	return context.WithValue(ctx, contextKey{}, buf)
}

// SecurityKey retrieves the AES key from context if present.
func SecurityKey(ctx context.Context) []byte {
	if v := ctx.Value(contextKey{}); v != nil {
		if key, ok := v.([]byte); ok {
			return key
		}
	}
	return nil
}

// ParseKeyHex validates and decodes a 32-hex-digit AES key string.
func ParseKeyHex(input string) ([]byte, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	clean := stripWhitespace(input)
	if len(clean) != 32 {
		return nil, fmt.Errorf("AES key must be 32 hex digits (16 bytes), got %d", len(clean))
	}
	dst := make([]byte, 16)
	if _, err := hex.Decode(dst, []byte(clean)); err != nil {
		return nil, fmt.Errorf("invalid AES key hex: %w", err)
	}
	return dst, nil
}

func stripWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
