package wmbus

import "fmt"

// Record represents a parsed DIF/VIF entry from a telegram payload.
type Record struct {
	DIF     byte
	DIFE    []byte
	VIF     int
	RawVIF  []byte
	Data    []byte
	Storage int
	Tariff  int
	Subunit int
}

// ParseRecords iterates over the payload and returns the DIF/VIF records until
// manufacturer-specific data is reached (DIF 0x0F) or the buffer ends.
func ParseRecords(payload []byte) ([]Record, error) {
	records := make([]Record, 0, 8)
	i := 0
	for i < len(payload) {
		dif := payload[i]
		i++
		if dif == 0x2F {
			continue
		}
		if dif == 0x0F {
			break
		}
		if dif == 0x00 {
			continue
		}
		rec := Record{DIF: dif}
		storage := int((dif >> 6) & 0x01)
		tariff := 0
		subunit := 0
		difenr := 0

		hasDIFE := (dif & 0x80) != 0
		for hasDIFE {
			if i >= len(payload) {
				return nil, fmt.Errorf("unexpected end of payload while reading DIFE")
			}
			dife := payload[i]
			i++
			rec.DIFE = append(rec.DIFE, dife)
			subunit |= int((dife>>6)&0x01) << difenr
			tariff |= int((dife>>4)&0x03) << (difenr * 2)
			storage |= int(dife&0x0F) << (1 + difenr*4)
			hasDIFE = (dife & 0x80) != 0
			difenr++
		}
		if i >= len(payload) {
			return nil, fmt.Errorf("unexpected end of payload before VIF")
		}
		vifByte := payload[i]
		i++
		rec.RawVIF = append(rec.RawVIF, vifByte)
		if vifByte == 0xFB || vifByte == 0xFD || vifByte == 0xEF || vifByte == 0xFF {
			return nil, fmt.Errorf("extended VIF 0x%02X not supported", vifByte)
		}
		if (vifByte & 0x80) != 0 {
			return nil, fmt.Errorf("VIF extensions not supported (saw 0x%02X)", vifByte)
		}
		fullVIF := int(vifByte & 0x7F)

		length, ok := LengthForDIF(dif & 0x0F)
		if !ok {
			break
		}
		if length == 0 {
			continue
		}
		if i+length > len(payload) {
			return nil, fmt.Errorf("payload truncated for DIF 0x%02X", dif)
		}
		rec.Data = append(rec.Data, payload[i:i+length]...)
		i += length

		rec.VIF = fullVIF
		rec.Storage = storage
		rec.Tariff = tariff
		rec.Subunit = subunit
		records = append(records, rec)
	}
	return records, nil
}
