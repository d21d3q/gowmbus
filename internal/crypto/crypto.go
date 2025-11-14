package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"

	"gitlab.com/d21d3q/gowmbus/internal/frame"
)

var (
	ErrKeyRequired = errors.New("encrypted telegram: AES key required (use --key)")
	ErrInvalidKey  = errors.New("encrypted telegram: AES key rejected (bad plaintext)")
)

const securityModeAesCbcIV = 5

// Decrypt mutates the payload when the content looks encrypted.
func Decrypt(t *frame.Telegram, key []byte) error {
	if !needsDecryption(t) {
		return nil
	}
	if len(key) == 0 {
		return ErrKeyRequired
	}
	return decryptCBC(t, key)
}

func decryptCBC(t *frame.Telegram, key []byte) error {
	required := encryptedPrefixLen(t)
	if required == 0 {
		return ErrInvalidKey
	}
	if required > len(t.Payload) {
		return fmt.Errorf("encrypted section exceeds payload length (%d > %d)", required, len(t.Payload))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("invalid AES key: %w", err)
	}
	ciphertext := make([]byte, required)
	copy(ciphertext, t.Payload[:required])
	iv := buildShortIV(t)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(ciphertext, ciphertext)
	if !looksLikePlaintext(ciphertext) {
		return ErrInvalidKey
	}
	plaintext := append(ciphertext, t.Payload[required:]...)
	if len(plaintext) >= 2 && plaintext[0] == 0x2f && plaintext[1] == 0x2f {
		plaintext = plaintext[2:]
	}
	t.Payload = plaintext
	return nil
}

func buildShortIV(t *frame.Telegram) []byte {
	iv := make([]byte, 16)
	iv[0] = byte(t.Manufacturer)
	iv[1] = byte(t.Manufacturer >> 8)
	copy(iv[2:6], t.MeterID[:])
	iv[6] = t.Version
	iv[7] = t.DeviceType
	for i := 8; i < 16; i++ {
		iv[i] = t.AccessNumber
	}
	return iv
}

func looksLikePlaintext(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	first := b[0]
	if first == 0x2f {
		return true
	}
	low := first & 0x0F
	return low <= 0x0D
}

func needsDecryption(t *frame.Telegram) bool {
	if len(t.Payload) == 0 {
		return false
	}
	if len(t.Payload) >= 2 && t.Payload[0] == 0x2f && t.Payload[1] == 0x2f {
		return false
	}
	if t.TPL.Present {
		return t.TPL.SecurityMode == securityModeAesCbcIV
	}
	return !looksLikePlaintext(t.Payload)
}

func encryptedPrefixLen(t *frame.Telegram) int {
	payloadLen := len(t.Payload)
	if payloadLen == 0 {
		return 0
	}
	if t.TPL.Present && t.TPL.EncryptedBlocks > 0 {
		needed := t.TPL.EncryptedBlocks * aes.BlockSize
		if needed > payloadLen {
			needed = payloadLen
		}
		if rem := needed % aes.BlockSize; rem != 0 {
			needed -= rem
		}
		return needed
	}
	length := payloadLen - (payloadLen % aes.BlockSize)
	return length
}
