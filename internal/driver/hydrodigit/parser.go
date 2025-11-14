package hydrodigit

import (
	"encoding/hex"
	"fmt"
	"math"
	"time"
)

type standardReadings struct {
	TotalVolumeM3 float64
	MeterDateTime time.Time
	VolumeScale   float64
}

func parseStandardReadings(payload []byte) (standardReadings, []byte, error) {
	var readings standardReadings
	var manufacturerBlock []byte
	i := 0
	for i < len(payload) {
		dif := payload[i]
		i++
		if dif == 0x2F {
			continue
		}
		if dif == 0x0F {
			if i <= len(payload) {
				manufacturerBlock = payload[i-1:]
			}
			break
		}
		dataDIF := dif
		for (dif & 0x80) != 0 {
			if i >= len(payload) {
				return readings, nil, fmt.Errorf("payload ended while reading DIFE bytes")
			}
			dife := payload[i]
			i++
			dif = dife
			if (dife & 0x80) == 0 {
				break
			}
		}
		if i >= len(payload) {
			return readings, nil, fmt.Errorf("payload ended before VIF")
		}
		vif := payload[i]
		i++
		for (vif & 0x80) != 0 {
			if i >= len(payload) {
				return readings, nil, fmt.Errorf("payload ended while reading VIFE bytes")
			}
			vif = payload[i]
			i++
		}
		length, ok := lengthForDIF(dataDIF & 0x0F)
		if !ok {
			return readings, nil, fmt.Errorf("unsupported DIF 0x%02X", dataDIF)
		}
		if i+length > len(payload) {
			return readings, nil, fmt.Errorf("payload truncated for DIF 0x%02X", dataDIF)
		}
		data := payload[i : i+length]
		i += length

		switch {
		case (dataDIF&0x0F) == 0x0C && readings.TotalVolumeM3 == 0:
			digits, err := decodeBCDLittleEndian(data)
			if err != nil {
				return readings, nil, err
			}
			if scale, ok := volumeScaleFromVIF(vif); ok {
				readings.TotalVolumeM3 = roundTo(float64(digits)*scale, 3)
				readings.VolumeScale = scale
			}
		case (dataDIF&0x0F) == 0x04 && vif == 0x6D && readings.MeterDateTime.IsZero():
			ts, err := decodeTypeFDateTime(data)
			if err != nil {
				return readings, nil, err
			}
			readings.MeterDateTime = ts
		}
	}
	return readings, manufacturerBlock, nil
}

func lengthForDIF(dif byte) (int, bool) {
	switch dif & 0x0F {
	case 0x00:
		return 0, true
	case 0x01:
		return 1, true
	case 0x02:
		return 2, true
	case 0x03:
		return 3, true
	case 0x04:
		return 4, true
	case 0x05:
		return 4, true
	case 0x06:
		return 6, true
	case 0x07:
		return 8, true
	case 0x08:
		return 0, false // variable length not handled
	case 0x09:
		return 1, true
	case 0x0A:
		return 2, true
	case 0x0B:
		return 3, true
	case 0x0C:
		return 4, true
	case 0x0D:
		return 5, true
	case 0x0E:
		return 6, true
	case 0x0F:
		return 0, true
	default:
		return 0, false
	}
}

func volumeScaleFromVIF(vif byte) (float64, bool) {
	switch vif & 0x7F {
	case 0x10:
		return 1e-6, true // cm^3
	case 0x11:
		return 1e-5, true
	case 0x12:
		return 1e-4, true
	case 0x13:
		return 1e-3, true // liters
	case 0x14:
		return 1e-2, true
	case 0x15:
		return 1e-1, true
	case 0x16:
		return 1, true
	case 0x17:
		return 10, true
	default:
		return 0, false
	}
}

func decodeBCDLittleEndian(b []byte) (int, error) {
	value := 0
	multiplier := 1
	for _, by := range b {
		low := int(by & 0x0F)
		high := int((by >> 4) & 0x0F)
		if low > 9 || high > 9 {
			return 0, fmt.Errorf("invalid BCD byte: 0x%02X", by)
		}
		value += low * multiplier
		multiplier *= 10
		value += high * multiplier
		multiplier *= 10
	}
	return value, nil
}

func decodeTypeFDateTime(b []byte) (time.Time, error) {
	if len(b) != 4 {
		return time.Time{}, fmt.Errorf("type F datetime requires 4 bytes, got %d", len(b))
	}
	minute := int(b[0] & 0x3F)
	hour := int(b[1] & 0x1F)
	day := int(b[2] & 0x1F)
	month := int(b[3] & 0x0F)
	yearBitsHigh := (b[3] >> 4) & 0x0F
	yearBitsLow := (b[2] >> 5) & 0x07
	year := 2000 + int(yearBitsHigh<<3|yearBitsLow)
	if minute > 59 || hour > 23 || day == 0 || day > 31 || month == 0 || month > 12 {
		return time.Time{}, fmt.Errorf("invalid type F datetime encoding: %s", hex.EncodeToString(b))
	}
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC), nil
}

func roundTo(value float64, decimals int) float64 {
	pow := math.Pow10(decimals)
	return math.Round(value*pow) / pow
}
