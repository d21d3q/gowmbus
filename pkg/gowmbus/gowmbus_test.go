package gowmbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeHex(t *testing.T) {
	raw := " |4E44_B409 86868686| "
	data, err := decodeHex(raw)
	require.NoError(t, err)
	require.Len(t, data, 8)
}

func TestDecodeHexOddLength(t *testing.T) {
	_, err := decodeHex("ABC")
	require.Error(t, err)
}

func TestAnalyzeHexHydrodigit(t *testing.T) {
	ctx := context.Background()
	frame := "4E44B4098686868613077AF00040052F2F0C1366380000046D27287E2A0F150E00000000C10000D10000E60000FD00000C01002F0100410100540100680100890000A00000B30000002F2F2F2F2F2F"
	result, err := AnalyzeHex(ctx, frame)
	require.NoError(t, err)
	require.Equal(t, "hydrodigit", result.Driver)
	require.NotNil(t, result.Telegram)
	require.Equal(t, "86868686", result.Telegram.MeterIDString())
}
