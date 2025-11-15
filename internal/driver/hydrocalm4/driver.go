package hydrocalm4

import (
	"context"
	"fmt"

	"gitlab.com/d21d3q/gowmbus/internal/driver"
	"gitlab.com/d21d3q/gowmbus/internal/driver/wmbus"
	"gitlab.com/d21d3q/gowmbus/internal/frame"
)

const (
	manufacturerBMT  = 0x09B4
	ciHydrocalm4     = 0x8C
	deviceTypeHeat   = 0x0D
	defaultTimestamp = "1111-11-11T11:11:11Z"
	mediaHeat        = "heat/cooling load"
)

func init() {
	driver.Register(driver.Detection{
		Manufacturer: manufacturerBMT,
		CI:           ciHydrocalm4,
		DeviceTypes:  []byte{deviceTypeHeat},
	}, Driver{})
}

// Driver implements decoding for Hydrocalm4 heat meters.
type Driver struct{}

var _ driver.PartialReporter = Driver{}

// Name returns the canonical driver name.
func (Driver) Name() string { return "hydrocalm4" }

// PartialFields exposes basic metadata when parsing fails.
func (Driver) PartialFields(t *frame.Telegram) map[string]any {
	return map[string]any{
		"_":      "telegram",
		"id":     t.MeterIDString(),
		"meter":  "hydrocalm4",
		"media":  mediaHeat,
		"status": statusString(t),
	}
}

// Process parses the telegram payload into structured fields.
func (Driver) Process(_ context.Context, t *frame.Telegram) (map[string]any, error) {
	payload := trimToApplication(t.Payload)
	records, err := wmbus.ParseRecords(payload)
	if err != nil {
		return nil, err
	}
	values, err := aggregate(records)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{
		"_":         "telegram",
		"id":        t.MeterIDString(),
		"meter":     "hydrocalm4",
		"media":     mediaHeat,
		"timestamp": defaultTimestamp,
		"status":    statusString(t),
	}
	if values.DeviceDateTime != "" {
		fields["device_datetime"] = values.DeviceDateTime
	}
	if values.TotalHeatingKWh != nil {
		fields["total_heating_kwh"] = *values.TotalHeatingKWh
	}
	if values.TotalCoolingKWh != nil {
		fields["total_cooling_kwh"] = *values.TotalCoolingKWh
	}
	if values.TotalHeatingM3 != nil {
		fields["total_heating_m3"] = *values.TotalHeatingM3
	}
	if values.TotalCoolingM3 != nil {
		fields["total_cooling_m3"] = *values.TotalCoolingM3
	}
	if values.C1VolumeM3 != nil {
		fields["c1_volume_m3"] = *values.C1VolumeM3
	}
	if values.C2VolumeM3 != nil {
		fields["c2_volume_m3"] = *values.C2VolumeM3
	}
	if values.SupplyTempC != nil {
		fields["supply_temperature_c"] = *values.SupplyTempC
	}
	if values.ReturnTempC != nil {
		fields["return_temperature_c"] = *values.ReturnTempC
	}
	if values.VolumeFlowM3h != nil {
		fields["volume_flow_m3h"] = *values.VolumeFlowM3h
	}
	if values.PowerKW != nil {
		fields["power_kw"] = *values.PowerKW
	}
	return fields, nil
}

func statusString(t *frame.Telegram) string {
	return "OK"
}

type aggregateValues struct {
	DeviceDateTime  string
	TotalHeatingKWh *float64
	TotalCoolingKWh *float64
	TotalHeatingM3  *float64
	TotalCoolingM3  *float64
	C1VolumeM3      *float64
	C2VolumeM3      *float64
	SupplyTempC     *float64
	ReturnTempC     *float64
	VolumeFlowM3h   *float64
	PowerKW         *float64
}

func aggregate(recs []wmbus.Record) (aggregateValues, error) {
	var out aggregateValues
	for _, rec := range recs {
		switch {
		case rec.VIF == 0x6D:
			ts, err := wmbus.DecodeTypeFDateTime(rec.Data)
			if err != nil {
				return out, err
			}
			out.DeviceDateTime = ts.Format("2006-01-02 15:04")
		case isEnergyVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			if rec.Tariff == 1 {
				out.TotalCoolingKWh = ptr(val)
			} else {
				out.TotalHeatingKWh = ptr(val)
			}
		case isVolumeVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			switch {
			case rec.Subunit == 1:
				out.C1VolumeM3 = ptr(val)
			case rec.Subunit == 2:
				out.C2VolumeM3 = ptr(val)
			case rec.Tariff == 1:
				out.TotalCoolingM3 = ptr(val)
			default:
				out.TotalHeatingM3 = ptr(val)
			}
		case isVolumeFlowVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			out.VolumeFlowM3h = ptr(val)
		case isPowerVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			out.PowerKW = ptr(val)
		case isFlowTempVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			out.SupplyTempC = ptr(val)
		case isReturnTempVIF(rec.VIF):
			val, err := decodeValue(rec)
			if err != nil {
				return out, err
			}
			out.ReturnTempC = ptr(val)
		}
	}
	return out, nil
}

func isEnergyVIF(v int) bool { return v >= 0x00 && v <= 0x0F }
func isVolumeVIF(v int) bool { return v >= 0x10 && v <= 0x17 }

func isVolumeFlowVIF(v int) bool {
	return (v >= 0x38 && v <= 0x3F) || (v >= 0x40 && v <= 0x4F)
}

func isPowerVIF(v int) bool {
	return (v >= 0x28 && v <= 0x2F) || (v >= 0x30 && v <= 0x37)
}

func isFlowTempVIF(v int) bool   { return v >= 0x58 && v <= 0x5B }
func isReturnTempVIF(v int) bool { return v >= 0x5C && v <= 0x5F }

type unitKind int

const (
	unitUnknown unitKind = iota
	unitKWh
	unitMJ
	unitM3
	unitM3h
	unitMJh
	unitKW
	unitCelsius
)

func scaleForVIF(v int) (float64, unitKind, bool) {
	switch v {
	case 0x00:
		return 1_000_000, unitKWh, true
	case 0x01:
		return 100_000, unitKWh, true
	case 0x02:
		return 10_000, unitKWh, true
	case 0x03:
		return 1_000, unitKWh, true
	case 0x04:
		return 100, unitKWh, true
	case 0x05:
		return 10, unitKWh, true
	case 0x06:
		return 1, unitKWh, true
	case 0x07:
		return 0.1, unitKWh, true
	case 0x08:
		return 1_000_000, unitMJ, true
	case 0x09:
		return 100_000, unitMJ, true
	case 0x0A:
		return 10_000, unitMJ, true
	case 0x0B:
		return 1_000, unitMJ, true
	case 0x10:
		return 1_000_000, unitM3, true
	case 0x13:
		return 1_000, unitM3, true
	case 0x14:
		return 100, unitM3, true
	case 0x15:
		return 10, unitM3, true
	case 0x16:
		return 1, unitM3, true
	case 0x17:
		return 0.1, unitM3, true
	case 0x28:
		return 1_000_000, unitKW, true
	case 0x29:
		return 100_000, unitKW, true
	case 0x2A:
		return 10_000, unitKW, true
	case 0x2B:
		return 1_000, unitKW, true
	case 0x2C:
		return 100, unitKW, true
	case 0x2D:
		return 10, unitKW, true
	case 0x2E:
		return 1, unitKW, true
	case 0x2F:
		return 0.1, unitKW, true
	case 0x3B:
		return 1_000, unitM3h, true
	case 0x3C:
		return 100, unitM3h, true
	case 0x3D:
		return 10, unitM3h, true
	case 0x3E:
		return 1, unitM3h, true
	case 0x40:
		return 600_000_000, unitM3h, true
	case 0x48:
		return 1_000_000_000 * 3600, unitM3h, true
	case 0x30:
		return 1_000_000, unitMJh, true
	case 0x31:
		return 100_000, unitMJh, true
	case 0x32:
		return 10_000, unitMJh, true
	case 0x59:
		return 100, unitCelsius, true
	case 0x5D:
		return 100, unitCelsius, true
	default:
		return 0, unitUnknown, false
	}
}

func decodeValue(rec wmbus.Record) (float64, error) {
	raw, err := wmbus.DecodeBCDLittleEndian(rec.Data)
	if err != nil {
		return 0, fmt.Errorf("decode VIF 0x%02X: %w", rec.VIF, err)
	}
	scale, unit, ok := scaleForVIF(rec.VIF)
	if !ok || scale == 0 {
		return 0, fmt.Errorf("unsupported VIF 0x%02X", rec.VIF)
	}
	value := float64(raw) / scale
	switch unit {
	case unitKWh, unitM3, unitM3h, unitCelsius, unitKW:
		return value, nil
	case unitMJ:
		return value / 3.6, nil
	case unitMJh:
		return value * (1000.0 / 3.6), nil
	default:
		return 0, fmt.Errorf("unsupported unit for VIF 0x%02X", rec.VIF)
	}
}

func ptr(v float64) *float64 {
	return &v
}

func trimToApplication(payload []byte) []byte {
	for i := 0; i+1 < len(payload); i++ {
		if payload[i] == 0x04 && payload[i+1] == 0x6D {
			return payload[i:]
		}
	}
	return payload
}
