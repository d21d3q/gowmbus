package gowmbus

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/d21d3q/gowmbus/internal/testutil"
)

func TestHydrodigitGolden(t *testing.T) {
	fixtures := []struct {
		name        string
		opts        AnalyzeOptions
		expectError bool
		expectFile  string
	}{
		{name: "hydrodigit_water"},
		{name: "hydrodigit_unknown"},
		{name: "hydro3"},
		{name: "hydro4"},
		{name: "hydrolink_worked_example", expectFile: "hydrodigit/hydrolink_worked_example_partial.json"},
		{name: "hydrolink_worked_example", opts: AnalyzeOptions{KeyHex: strings.Repeat("0", 32)}},
	}
	for _, tc := range fixtures {
		tc := tc
		testName := tc.name
		if tc.opts.KeyHex != "" {
			testName += "_with_key"
		}
		t.Run(testName, func(t *testing.T) {
			hexStr := testutil.LoadHex(t, "hydrodigit/"+tc.name+".hex")
			result, err := AnalyzeHexWithOptions(context.Background(), hexStr, tc.opts)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "encrypted")
				return
			}
			require.NoError(t, err)
			path := "hydrodigit/" + tc.name + ".json"
			if tc.expectFile != "" {
				path = tc.expectFile
			}
			var expected map[string]any
			testutil.LoadJSON(t, path, &expected)
			require.Equal(t, "", diffMaps(expected, result.Fields))
		})
	}
}

func diffMaps(expected, actual map[string]any) string {
	if len(expected) != len(actual) {
		return fmt.Sprintf("len mismatch expected %d actual %d", len(expected), len(actual))
	}
	for k, v := range expected {
		av, ok := actual[k]
		if !ok {
			return fmt.Sprintf("missing key %s", k)
		}
		switch ev := v.(type) {
		case float64:
			avFloat, ok := av.(float64)
			if !ok || math.Abs(ev-avFloat) > 1e-6 {
				return fmt.Sprintf("key %s mismatch expected %v got %v", k, v, av)
			}
		default:
			if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", av) {
				return fmt.Sprintf("key %s mismatch expected %v got %v", k, v, av)
			}
		}
	}
	return ""
}
