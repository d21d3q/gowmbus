package records

// Record represents a decoded DIF/VIF entry from the application payload.
type Record struct {
	DIF  byte
	VIF  byte
	VIFE []byte
	Data []byte
}

// Decoder extracts DIF/VIF records from manufacturer payloads. Implementation
// will follow once the parser migrates from wmbusmeters logic.
type Decoder interface {
	Decode(payload []byte) ([]Record, error)
}
