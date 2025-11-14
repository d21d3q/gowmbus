package driver

import (
	"context"
	"fmt"
	"sync"

	"gitlab.com/d21d3q/gowmbus/internal/frame"
)

// Detection contains minimal information required to identify a driver.
type Detection struct {
	Manufacturer uint16
	CI           byte
	LinkMode     byte
}

// Driver processes telegrams once selected.
type Driver interface {
	Name() string
	Process(context.Context, *frame.Telegram) (map[string]any, error)
}

// PartialReporter can supply minimal fields when payload decryption fails.
type PartialReporter interface {
	PartialFields(*frame.Telegram) map[string]any
}

var (
	regMu    sync.RWMutex
	registry []registeredDriver
)

type registeredDriver struct {
	detect Detection
	driver Driver
}

// Register stores a driver/detection pair in memory.
func Register(det Detection, drv Driver) {
	regMu.Lock()
	defer regMu.Unlock()
	registry = append(registry, registeredDriver{detect: det, driver: drv})
}

// Lookup returns the first driver that matches the detection key.
func Lookup(det Detection) (Driver, error) {
	regMu.RLock()
	defer regMu.RUnlock()
	for _, rd := range registry {
		if rd.detect.Manufacturer == det.Manufacturer && rd.detect.CI == det.CI {
			return rd.driver, nil
		}
	}
	return nil, fmt.Errorf("driver not found for manufacturer 0x%04X CI 0x%02X", det.Manufacturer, det.CI)
}
