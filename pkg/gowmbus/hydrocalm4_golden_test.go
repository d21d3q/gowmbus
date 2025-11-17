package gowmbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/d21d3q/gowmbus/internal/testutil"
)

func TestHydrocalm4Golden(t *testing.T) {
	fixtures := []string{
		"standard_heat",
		"combined_heat_cool",
		"instantaneous_temperature",
		"instantaneous_pulses",
		"power_unit_jh",
	}
	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			hexStr := testutil.LoadHex(t, "hydrocalm4/"+name+".hex")
			result, err := AnalyzeHexWithOptions(context.Background(), hexStr, AnalyzeOptions{})
			require.NoError(t, err)

			var expected map[string]any
			testutil.LoadJSON(t, "hydrocalm4/"+name+".json", &expected)
			require.Equal(t, "", diffMaps(expected, result.Fields))
		})
	}
}
