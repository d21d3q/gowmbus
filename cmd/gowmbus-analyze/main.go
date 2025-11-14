package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"gitlab.com/d21d3q/gowmbus/pkg/gowmbus"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gowmbus-analyze [hex]",
		Short: "Decode Wireless M-Bus telegrams",
		Long:  "gowmbus-analyze decodes Wireless M-Bus telegrams using the gowmbus library.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := gowmbus.AnalyzeOptions{KeyHex: keyHex}
			ctx := cmd.Context()
			if len(args) == 0 {
				return runInteractive(ctx, opts)
			}
			return runAnalyze(ctx, opts, args[0])
		},
	}

	keyHex string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&keyHex, "key", "", "hex-encoded 16-byte AES key (32 hex chars)")
}

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	ctx := context.Background()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logrus.Fatal(err)
	}
}

func runInteractive(ctx context.Context, opts gowmbus.AnalyzeOptions) error {
	scanner := bufio.NewScanner(os.Stdin)
	logrus.Info("gowmbus analyze mode. Paste a hex telegram and press Enter (Ctrl+D to exit).")
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := runAnalyze(ctx, opts, line); err != nil {
			logrus.WithError(err).Error("failed to decode telegram")
		}
	}
	return scanner.Err()
}

func runAnalyze(ctx context.Context, opts gowmbus.AnalyzeOptions, hex string) error {
	result, err := gowmbus.AnalyzeHexWithOptions(ctx, hex, opts)
	if err != nil {
		return err
	}
	fmt.Println(result.String())
	return nil
}
