package wmbus

import (
	"fmt"
	"time"
)

// LengthForDIF returns the data length encoded in the lower nibble of the DIF
// byte. The boolean indicates whether the DIF value is supported.
func LengthForDIF(dif byte) (int, bool) {
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

// DecodeBCDLittleEndian converts a BCD payload (little endian nibble order) to
// an integer.
func DecodeBCDLittleEndian(b []byte) (int, error) {
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

// DecodeTypeFDateTime decodes the four-byte Type F timestamp used by many
// Wireless M-Bus meters.
func DecodeTypeFDateTime(b []byte) (time.Time, error) {
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
		return time.Time{}, fmt.Errorf("invalid type F datetime encoding: %02X%02X%02X%02X", b[0], b[1], b[2], b[3])
	}
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC), nil
}
