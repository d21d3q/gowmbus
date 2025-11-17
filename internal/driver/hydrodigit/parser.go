package hydrodigit

import (
	"fmt"
	"time"

	"github.com/d21d3q/gowmbus/internal/driver/wmbus"
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
		length, ok := wmbus.LengthForDIF(dataDIF & 0x0F)
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
			digits, err := wmbus.DecodeBCDLittleEndian(data)
			if err != nil {
				return readings, nil, err
			}
			if scale, ok := volumeScaleFromVIF(vif); ok {
				readings.TotalVolumeM3 = float64(digits) * scale
				readings.VolumeScale = scale
			}
		case (dataDIF&0x0F) == 0x04 && vif == 0x6D && readings.MeterDateTime.IsZero():
			ts, err := wmbus.DecodeTypeFDateTime(data)
			if err != nil {
				return readings, nil, err
			}
			readings.MeterDateTime = ts
		}
	}
	return readings, manufacturerBlock, nil
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
