package hydrodigit

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// Data captures the decoded contents of the hydrodigit manufacturer-specific
// record (DIF 0x0F). Variant selects between the legacy block already parsed by
// wmbusmeters today ("legacy") and the newer Hydrolink layout ("extended").
type Data struct {
	Variant string

	// Legacy fields (frame identifier 0x15/0x95).
	FrameIdentifier byte
	Contents        string
	Voltage         float64
	LeakDate        string // dd.mm.yyyy
	BackflowM3      float64
	MonthlyTotals   map[string]float64

	// Extended (Hydrolink) fields.
	BatteryPercentRaw     uint8
	BatteryPercentClamped uint8
	ErrorBits             uint32
	MSByte                byte
	OptionalSections      OptionalSections
}

// OptionalSections mirrors the bitmap defined in hydrodigit-manufacturer-data.md.
type OptionalSections struct {
	HasInstantaneous bool
	InstantaneousRaw []byte

	HasReverseFlow  bool
	ReverseFlowM3   float64
	HasEmptyPipe    bool
	EmptyPipeDate   string
	HasLeakDate     bool
	LeakEventDate   string
	HasFreezeDate   bool
	FreezeEventDate string
	MemoDay1        []byte
	MemoDay2        []byte

	HasMonthlyHistory bool
	MonthlyHistory    []float64
}

const (
	minLegacyBytes   = 1 + 1 + 4 + 12*3 // frame id + voltage + backflow + months
	minExtendedBytes = 1 + 3 + 1        // battery + error bits + MS byte
)

// ParseManufacturerData decodes the manufacturer block using either the legacy
// or extended format. Callers may pass the slice starting at DIF 0x0F or just
// the bytes that follow. volumeScale should describe the scaling used for the
// primary consumption record (e.g. 1e-3 for VIF 0x13).
func ParseManufacturerData(raw []byte, volumeScale float64) (Data, error) {
	if len(raw) == 0 {
		return Data{}, errors.New("empty manufacturer payload")
	}
	block := raw
	if raw[0] == 0x0F {
		if len(raw) == 1 {
			return Data{}, errors.New("manufacturer payload missing content")
		}
		block = raw[1:]
	}
	if len(block) >= minLegacyBytes && (block[0] == 0x15 || block[0] == 0x95) {
		return parseLegacyBlock(block, volumeScale)
	}
	if len(block) >= minExtendedBytes {
		return parseExtendedBlock(block, volumeScale)
	}
	return Data{}, fmt.Errorf("unsupported manufacturer block: %s", hex.EncodeToString(block))
}

func parseLegacyBlock(block []byte, mainScale float64) (Data, error) {
	d := Data{
		Variant:         "legacy",
		FrameIdentifier: block[0],
		MonthlyTotals:   make(map[string]float64, 12),
	}
	offset := 1
	if offset >= len(block) {
		return Data{}, errors.New("legacy block truncated before voltage byte")
	}
	d.Contents = legacyContents(block[0])
	d.Voltage = decodeVoltage(block[offset] & 0x0F)
	offset++

	if block[0] == 0x95 {
		if offset+2 >= len(block) {
			return Data{}, errors.New("legacy block truncated before leak date")
		}
		year := block[offset]
		month := block[offset+1]
		day := block[offset+2]
		d.LeakDate = fmt.Sprintf("%02X.%02X.20%02X", day, month, year)
		offset += 3
	}

	if offset+3 >= len(block) {
		return Data{}, errors.New("legacy block truncated before backflow")
	}
	d.BackflowM3 = decodeBackflow(block[offset : offset+4])
	offset += 4

	months := []string{
		"January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December",
	}
	monthlyScale := adjustedMonthlyScale(mainScale)
	for _, month := range months {
		if offset+2 >= len(block) {
			return Data{}, errors.New("legacy block truncated while decoding monthly history")
		}
		value := decodeMonthly(block[offset:offset+3], monthlyScale)
		d.MonthlyTotals[month] = value
		offset += 3
	}
	return d, nil
}

func parseExtendedBlock(block []byte, mainScale float64) (Data, error) {
	if len(block) < minExtendedBytes {
		return Data{}, errors.New("extended block too short")
	}
	d := Data{
		Variant:               "extended",
		BatteryPercentRaw:     block[0],
		BatteryPercentClamped: clampBattery(block[0]),
	}
	offset := 1
	d.ErrorBits = uint32(block[offset])<<16 | uint32(block[offset+1])<<8 | uint32(block[offset+2])
	offset += 3
	d.MSByte = block[offset]
	offset++

	sections := OptionalSections{}
	for bit := 0; bit < 8; bit++ {
		if (d.MSByte>>bit)&0x01 == 0 {
			continue
		}
		switch bit {
		case 0:
			if offset+6 >= len(block) {
				return Data{}, errors.New("instantaneous block truncated")
			}
			sections.HasInstantaneous = true
			sections.InstantaneousRaw = append([]byte(nil), block[offset:offset+7]...)
			offset += 7
		case 1:
			if offset+2 >= len(block) {
				return Data{}, errors.New("reverse-flow block truncated")
			}
			sections.HasReverseFlow = true
			sections.ReverseFlowM3 = float64(uint32(block[offset])|uint32(block[offset+1])<<8|uint32(block[offset+2])<<16) / 1000.0
			offset += 3
		case 2:
			date, consumed, err := decodeBCDDate(block[offset:])
			if err == nil {
				sections.HasEmptyPipe = true
				sections.EmptyPipeDate = date
			}
			offset += consumed
		case 3:
			date, consumed, err := decodeBCDDate(block[offset:])
			if err == nil {
				sections.HasLeakDate = true
				sections.LeakEventDate = date
			}
			offset += consumed
		case 4:
			date, consumed, err := decodeBCDDate(block[offset:])
			if err == nil {
				sections.HasFreezeDate = true
				sections.FreezeEventDate = date
			}
			offset += consumed
		case 5:
			if offset+4 >= len(block) {
				return Data{}, errors.New("memo day 1 truncated")
			}
			sections.MemoDay1 = append([]byte(nil), block[offset:offset+5]...)
			offset += 5
		case 6:
			if offset+4 >= len(block) {
				return Data{}, errors.New("memo day 2 truncated")
			}
			sections.MemoDay2 = append([]byte(nil), block[offset:offset+5]...)
			offset += 5
		case 7:
			if offset+36 > len(block) {
				return Data{}, errors.New("monthly history truncated")
			}
			sections.HasMonthlyHistory = true
			monthlyScale := adjustedMonthlyScale(mainScale)
			values := make([]float64, 12)
			for i := 0; i < 12; i++ {
				values[i] = decodeMonthly(block[offset:offset+3], monthlyScale)
				offset += 3
			}
			sections.MonthlyHistory = values
		}
	}

	d.OptionalSections = sections
	return d, nil
}

func legacyContents(frameID byte) string {
	switch frameID {
	case 0x15:
		return "Backflow, alarms and monthly data"
	case 0x95:
		return "Backflow, leak date, alarms and monthly data"
	default:
		return "unknown, please open issue with this telegram for driver improvement"
	}
}

func decodeVoltage(nibble byte) float64 {
	switch nibble {
	case 0x01:
		return 1.9
	case 0x02:
		return 2.1
	case 0x03:
		return 2.2
	case 0x04:
		return 2.3
	case 0x05:
		return 2.4
	case 0x06:
		return 2.5
	case 0x07:
		return 2.65
	case 0x08:
		return 2.8
	case 0x09:
		return 2.9
	case 0x0A:
		return 3.05
	case 0x0B:
		return 3.2
	case 0x0C:
		return 3.35
	case 0x0D:
		return 3.5
	default:
		return 3.7
	}
}

func decodeBackflow(b []byte) float64 {
	if len(b) < 4 {
		return 0
	}
	value := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	return float64(value) / 1000.0
}

func decodeMonthly(b []byte, scale float64) float64 {
	if len(b) < 3 {
		return 0
	}
	value := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16
	if scale == 0 {
		scale = 0.01
	}
	result := float64(value) * scale
	if result >= 100000 {
		return 0
	}
	return result
}

func adjustedMonthlyScale(mainScale float64) float64 {
	if mainScale <= 0 {
		return 0.01
	}
	return mainScale * 10
}

func clampBattery(raw byte) uint8 {
	if raw > 100 {
		return 100
	}
	return raw
}

func decodeBCDDate(b []byte) (string, int, error) {
	if len(b) < 3 {
		return "", 0, errors.New("BCD date requires 3 bytes")
	}
	bytes := b[:3]
	for _, by := range bytes {
		if (by&0x0F) > 9 || (by>>4) > 9 {
			return "", 3, fmt.Errorf("invalid BCD digit in %02X", by)
		}
	}
	year := int(bytes[0]&0x0F) + int(bytes[0]>>4)*10
	month := int(bytes[1]&0x0F) + int(bytes[1]>>4)*10
	day := int(bytes[2]&0x0F) + int(bytes[2]>>4)*10
	return fmt.Sprintf("20%02d-%02d-%02d", year, month, day), 3, nil
}
