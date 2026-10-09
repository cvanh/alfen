package alfencrc

import (
	"hash/crc32"
	"math/rand"
	"testing"
)

// refCSharp is a faithful port of ACEFWUCreator.Crc32 with poly 0xEDB88320,
// used only to prove ComputeChecksum matches the decompiled C# loop.
func refCSharp(b []byte, start int) uint32 {
	var t [256]uint32
	for n := uint32(0); n < 256; n++ {
		c := n
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = (c >> 1) ^ 0xEDB88320
			} else {
				c >>= 1
			}
		}
		t[n] = c
	}
	num := uint32(0xFFFFFFFF)
	for i := start; i < len(b); i++ {
		bb := byte(num&0xFF) ^ b[i]
		num = (num >> 8) ^ t[bb]
	}
	return ^num
}

// TestEquivalence checks ComputeChecksum == the C# port == crc32.ChecksumIEEE
// across random inputs and start offsets.
func TestEquivalence(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for trial := 0; trial < 20000; trial++ {
		n := r.Intn(300)
		b := make([]byte, n)
		r.Read(b)
		start := 0
		if n > 0 {
			start = r.Intn(n + 1)
		}
		want := refCSharp(b, start)
		if got := ComputeChecksum(b, start); got != want {
			t.Fatalf("ComputeChecksum mismatch: got %08X want %08X (n=%d start=%d)", got, want, n, start)
		}
		if got := crc32.ChecksumIEEE(b[start:]); got != want {
			t.Fatalf("stdlib mismatch: got %08X want %08X", got, want)
		}
	}
}

// TestPolyIsIEEE guards the assumption that the installer's poly is IEEE.
func TestPolyIsIEEE(t *testing.T) {
	if PolyAlfen != 0xEDB88320 {
		t.Fatalf("PolyAlfen = %08X, want EDB88320", PolyAlfen)
	}
}
