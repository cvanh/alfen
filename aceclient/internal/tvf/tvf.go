// Package tvf covers the AHP `.tfw` (TVF) container.
//
// Two parts, mirroring the .fwi split:
//
//   - BuildHeader(): a port of ACEFWUCreator.TvfHeader (docs/decompiled, class
//     TvfHeader, lines ~8819-8893) — the installer's LOCAL resource-TVF header.
//     This local variant is NOT encrypted.
//   - Sniff(): a read-only inspector for the REAL Alfen AHP releases, which are
//     encrypted and ECDSA-P384 signed by Alfen's build pipeline (see
//     docs/01-firmware-formats.md). It locates the ASCII manifest and the
//     plaintext/ciphertext boundary without assuming an exact layout.
package tvf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"unicode"

	"alfen/aceclient/internal/alfencrc"
)

// MagicTVF is the u32 at offset 0x04 of the local TvfHeader (0xB8ED7AD5).
const MagicTVF uint32 = 0xB8ED7AD5

// baseHeaderLength is TvfHeader._baseHeaderLength.
const baseHeaderLength = 1350

// BuildHeader ports TvfHeader.GetBytes + CreateFinalTvfHeader for the local
// (unencrypted) resource-TVF variant.
func BuildHeader(manifestVersion int, description string, dataLength int) []byte {
	mv := fmt.Sprintf("%d", manifestVersion)
	var b bytes.Buffer
	le := binary.LittleEndian
	w16 := func(v uint16) { var t [2]byte; le.PutUint16(t[:], v); b.Write(t[:]) }
	w32 := func(v uint32) { var t [4]byte; le.PutUint32(t[:], v); b.Write(t[:]) }
	writeUTF8Len := func(s string) { b.WriteByte(byte(len(s))); b.WriteString(s) }

	headerLen := uint16(baseHeaderLength + len(mv) + len(description))

	w16(1)         // version marker
	w16(headerLen) // CalculateHeaderLength()
	w32(MagicTVF)  // 3102571221
	w32(uint32(dataLength))
	w32(0)
	b.WriteByte(0)
	b.WriteByte(0)
	b.WriteByte(0)
	b.WriteByte(0)
	w32(0)
	w32(0)
	writeUTF8Len(mv)
	writeUTF8Len(description)
	b.Write(make([]byte, 32))
	w16(0)
	b.Write(make([]byte, 256))
	w16(0)
	b.Write(make([]byte, 1024))

	headerBody := b.Bytes()
	crc := alfencrc.ComputeChecksum(headerBody, 0)
	out := make([]byte, 0, len(headerBody)+4)
	out = append(out, headerBody...)
	var t [4]byte
	le.PutUint32(t[:], crc)
	out = append(out, t[:]...)
	return out
}

// SniffResult summarises what Sniff found in a real .tfw.
type SniffResult struct {
	Manifest       string // ASCII manifest string near the top
	ManifestOffset int
	CertPresent    bool
	CertOffset     int
	PayloadStart   int // first offset that looks like the encrypted payload
}

// Sniff inspects a real .tfw: it extracts the leading ASCII manifest, notes an
// embedded X.509 certificate, and estimates where the encrypted payload begins.
// It never assumes a fixed layout; the offsets are best-effort.
func Sniff(data []byte) SniffResult {
	res := SniffResult{PayloadStart: -1, CertOffset: -1, ManifestOffset: -1}

	// Manifest: the first run of >= 16 printable ASCII bytes in the first 512.
	limit := 512
	if limit > len(data) {
		limit = len(data)
	}
	start, run := -1, 0
	for i := 0; i < limit; i++ {
		if isPrintable(data[i]) {
			if start < 0 {
				start = i
			}
			run++
		} else {
			if run >= 16 {
				res.Manifest = string(data[start : start+run])
				res.ManifestOffset = start
				break
			}
			start, run = -1, 0
		}
	}
	if res.Manifest == "" && run >= 16 {
		res.Manifest = string(data[start : start+run])
		res.ManifestOffset = start
	}

	// X.509 certificate: look for the SEQUENCE/… marker "0\x82" followed by a
	// certificate OID is overkill; the AHP cert subject string is a reliable tag.
	if idx := bytes.Index(data, []byte("AHP firmware signing")); idx >= 0 {
		res.CertPresent = true
		res.CertOffset = idx
	}

	// Payload: the notes place the encrypted region near 0x800. Report the first
	// 512-byte window past 0x400 whose Shannon entropy looks like ciphertext.
	res.PayloadStart = firstHighEntropyOffset(data, 0x400, 512, 7.5)
	return res
}

func isPrintable(b byte) bool {
	return b == '\t' || (b >= 0x20 && b < 0x7f) || unicode.IsSpace(rune(b)) && b < 0x80
}

// firstHighEntropyOffset scans in `window`-sized steps from `from` and returns
// the first offset whose window has Shannon entropy >= minBits bits/byte (a
// proxy for the AES payload). Returns -1 if none.
func firstHighEntropyOffset(data []byte, from, window int, minBits float64) int {
	for off := from; off+window <= len(data); off += window {
		if shannon(data[off:off+window]) >= minBits {
			return off
		}
	}
	return -1
}

// shannon returns the entropy of b in bits per byte (0..8).
func shannon(b []byte) float64 {
	var counts [256]int
	for _, c := range b {
		counts[c]++
	}
	n := float64(len(b))
	e := 0.0
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		e -= p * math.Log2(p)
	}
	return e
}
