package gowmbus

import (
	"context"

	internalopts "gitlab.com/d21d3q/gowmbus/internal/options"
)

// AnalyzeOptions configures parsing.
type AnalyzeOptions struct {
	KeyHex string
}

func (opts AnalyzeOptions) toInternal(ctx context.Context) (context.Context, []byte, error) {
	key, err := internalopts.ParseKeyHex(opts.KeyHex)
	if err != nil {
		return ctx, nil, err
	}
	ctx = internalopts.WithSecurityKey(ctx, key)
	return ctx, key, nil
}
