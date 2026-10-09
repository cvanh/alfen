// Package alfencrc provides the CRC used throughout the ACE firmware tooling.
//
// ACEFWUCreator.Crc32 (docs/decompiled/ACEFWUCreator.cs, lines 39-67) is a
// reflected CRC with init 0xFFFFFFFF, a final bitwise-NOT, and a table built
// from the polynomial passed to its constructor. EVERY real call site uses
// `new Crc32(3988292384u)` i.e. poly 0xEDB88320 — the standard reflected
// CRC-32/IEEE polynomial. With that poly the algorithm is byte-for-byte
// identical to Go's hash/crc32 IEEE variant, so we delegate to it directly
// rather than carry a hand-rolled table. (Equivalence is pinned by the test in
// crc32_test.go, which checks a faithful port of the C# loop against
// crc32.ChecksumIEEE over random inputs and start offsets.)
package alfencrc

import "hash/crc32"

// PolyAlfen is the polynomial used at every real call site in the installer
// (3988292384 == 0xEDB88320), i.e. reflected CRC-32/IEEE.
const PolyAlfen uint32 = crc32.IEEE // 0xEDB88320

// ComputeChecksum mirrors Crc32.ComputeChecksum(bytes, startIndex): it folds in
// bytes[startIndex:] with the IEEE poly, init 0xFFFFFFFF and a final NOT. That
// is exactly crc32.ChecksumIEEE(bytes[startIndex:]).
func ComputeChecksum(bytes []byte, startIndex int) uint32 {
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex > len(bytes) {
		startIndex = len(bytes)
	}
	return crc32.ChecksumIEEE(bytes[startIndex:])
}
