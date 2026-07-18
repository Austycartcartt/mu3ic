package library

import (
	"crypto/rand"
	"fmt"
)

// NewUUID returns a random (v4) UUID string, e.g.
// "550e8400-e29b-41d4-a716-446655440000". Hand-rolled rather than pulling
// in github.com/google/uuid — the Phase 2 spec permits only one new
// dependency (dhowden/tag), and a v4 UUID is a dozen lines of stdlib.
func NewUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating uuid: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
