package api

import (
	"crypto/rand"
	"encoding/hex"
)

// randBoundary returns a short random hex string for a multipart boundary.
func randBoundary() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
