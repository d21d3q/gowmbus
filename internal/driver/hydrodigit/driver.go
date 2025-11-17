package hydrodigit

import (
	"context"
	"fmt"

	"github.com/d21d3q/gowmbus/internal/driver"
	"github.com/d21d3q/gowmbus/internal/frame"
)

const (
	manufacturerBMT      = 0x09B4
	ciHydrodigitPrimary  = 0x7A
	ciHydrodigitExtended = 0x8C
	defaultTimestamp     = "1111-11-11T11:11:11Z"
	dateTimeFormat       = "2006-01-02 15:04"
	deviceTypeWater      = 0x07
	deviceTypeWarmWater  = 0x06
)

var monthOrder = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

func init() {
	driver.Register(driver.Detection{
		Manufacturer: manufacturerBMT,
		CI:           ciHydrodigitPrimary,
		DeviceTypes:  []byte{deviceTypeWater, deviceTypeWarmWater},
	}, Driver{})
	driver.Register(driver.Detection{
		Manufacturer: manufacturerBMT,
		CI:           ciHydrodigitExtended,
		DeviceTypes:  []byte{deviceTypeWater, deviceTypeWarmWater},
	}, Driver{})
}

// Driver implements the hydrodigit/hydrolink post-processing logic.
type Driver struct{}

var _ driver.PartialReporter = Driver{}

// Name returns the canonical driver name.
func (Driver) Name() string { return "hydrodigit" }

// PartialFields implements driver.PartialReporter.
func (Driver) PartialFields(t *frame.Telegram) map[string]any {
	fields := map[string]any{
		"_":     "telegram",
		"id":    t.MeterIDString(),
		"meter": "hydrodigit",
		"media": mediaFromDeviceType(t.DeviceType),
	}
	for k, v := range t.StatusFlags {
		fields[k] = v
	}
	return fields
}

// Process extracts manufacturer-specific data and returns a response map.
func (Driver) Process(_ context.Context, t *frame.Telegram) (map[string]any, error) {
	readings, mfctPayload, err := parseStandardReadings(t.Payload)
	if err != nil {
		return nil, err
	}
	if readings.TotalVolumeM3 == 0 && readings.MeterDateTime.IsZero() {
		return nil, fmt.Errorf("hydrodigit: telegram appears encrypted (supply meter key)")
	}
	if len(mfctPayload) == 0 {
		return nil, fmt.Errorf("hydrodigit manufacturer data missing")
	}
	mfct, err := ParseManufacturerData(mfctPayload, readings.VolumeScale)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{
		"_":         "telegram",
		"id":        t.MeterIDString(),
		"meter":     "hydrodigit",
		"media":     mediaFromDeviceType(t.DeviceType),
		"timestamp": defaultTimestamp,
	}
	if readings.TotalVolumeM3 > 0 {
		fields["total_m3"] = readings.TotalVolumeM3
	}
	if !readings.MeterDateTime.IsZero() {
		fields["meter_datetime"] = readings.MeterDateTime.Format(dateTimeFormat)
	}

	if mfct.Contents != "" {
		fields["contents"] = mfct.Contents
	}
	if mfct.Voltage > 0 {
		fields["voltage_v"] = mfct.Voltage
	}
	if mfct.BackflowM3 > 0 {
		fields["backflow_m3"] = mfct.BackflowM3
	}
	if mfct.LeakDate != "" {
		fields["leak_date"] = mfct.LeakDate
	}
	for _, month := range monthOrder {
		if value, ok := mfct.MonthlyTotals[month]; ok && value != 0 {
			key := fmt.Sprintf("%s_total_m3", month)
			fields[key] = value
		}
	}
	if mfct.Variant == "extended" {
		populateExtendedFields(fields, mfct)
	}
	for k, v := range t.StatusFlags {
		fields[k] = v
	}
	if t.StatusFlags["status_perm_alarm"] {
		fields["alarm_tamper"] = true
	}

	return fields, nil
}

func mediaFromDeviceType(device byte) string {
	switch device {
	case deviceTypeWater:
		return "water"
	case deviceTypeWarmWater:
		return "warm water"
	default:
		return "unknown"
	}
}

func populateExtendedFields(fields map[string]any, data Data) {
	fields["battery_percent_raw"] = float64(data.BatteryPercentRaw)
	fields["battery_percent_pct"] = float64(data.BatteryPercentClamped)
	fields["error_bits_hex"] = fmt.Sprintf("0x%06X", data.ErrorBits)
	fields["msb_flags_hex"] = fmt.Sprintf("0x%02X", data.MSByte)
	if data.OptionalSections.HasReverseFlow {
		fields["reverse_flow_m3"] = data.OptionalSections.ReverseFlowM3
	}
	if data.OptionalSections.HasEmptyPipe {
		fields["empty_pipe_date"] = data.OptionalSections.EmptyPipeDate
	}
	if data.OptionalSections.HasLeakDate {
		fields["leak_event_date"] = data.OptionalSections.LeakEventDate
	}
	if data.OptionalSections.HasFreezeDate {
		fields["freeze_event_date"] = data.OptionalSections.FreezeEventDate
	}
}

type statusFlag struct {
	mask  byte
	field string
}

var linkStatusFlags = []statusFlag{
	{0x80, "status_empty_pipe"},
	{0x40, "status_reverse_flow"},
	{0x20, "status_freezing"},
	{0x10, "status_temp_alarm"},
	{0x08, "status_perm_alarm"},
	{0x04, "status_battery_alarm"},
	{0x02, "status_hw_alarm"},
}
