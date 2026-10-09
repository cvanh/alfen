// Package fwi parses and builds Alfen NG9xx `.fwi` firmware containers.
//
// Two things live here, and they are NOT the same:
//
//  1. Parse(): a generic reader for the 160-byte header of ANY .fwi on disk
//     (controller firmware AND display-resource packages). It reads every
//     header word at its offset and interprets only the fields the installer
//     source confirms, leaving variant words raw. This is what feeds the key
//     tester.
//
//  2. BuildDisplayResource()/the AES + wrapper pipeline: a faithful port of
//     ICUFWUCreator.WriteFWUFile from docs/decompiled/ICUFWUCreator.cs. This is
//     the ONE path where the key is known (the hardcoded display key), so it is
//     end-to-end verifiable. Controller/AHP firmware uses a DIFFERENT,
//     device-held key and is NOT decryptable with this key.
//
// Header layout (confirmed against real files AND CreateFWIHeader):
//
//	0x00 u32  CRC32 over bytes [0x04:EOF]            (alfencrc, IEEE)
//	0x04 u32  header length (always 0x000000A0 = 160)
//	0x08 u16  type      (0x13 display/4.x-6.x, 0x21 7.x)
//	0x0A u16  magic     (0xA1FE)
//	0x0C u32  word0C    (display builder: 0xFFFFFFEF; controller fw: 0x080101F0)
//	0x10 u32  word10    (display builder: 0x0027A201; controller fw: 0x0047B001)
//	0x14 u32  payload length
//	0x18 u32  word18    (0xFFFFFFFF on 7.x; valued on 6.6.2)
//	0x1C u32  word1C
//	0x20 [64] wrapped-key block (all-0xFF = absent; present from ~6.0.0)
//	0x60 [16] section map (4x u32; present on 7.x, else 0xFF)
//	0x70 [48] padding (0xFF)
//	0xA0 ...   encrypted payload
package fwi

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"

	"alfen/aceclient/internal/alfencrc"
)

// HeaderLen is the fixed header size (0xA0).
const HeaderLen = 160

// MagicA1FE is the constant magic at offset 0x0A.
const MagicA1FE uint16 = 0xA1FE

// Known header type values at offset 0x08.
const (
	TypeLegacy uint16 = 0x13 // 4.12.0 / 5.6.1 / 6.6.2 and display resources
	TypeV7     uint16 = 0x21 // 7.x controller firmware
)

// DisplayKey is ICUFWUCreator.AESEncrypt's rgbKey (the display-resource key).
// Bytes {41,198,106,32,174,22,196,186,4,106,33,213,122,120,102,79}.
// Hex: 29C66A20AE16C4BA046A21D57A78664F. AES-128-CBC, zero IV, PKCS7.
var DisplayKey = []byte{41, 198, 106, 32, 174, 22, 196, 186, 4, 106, 33, 213, 122, 120, 102, 79}

// Header is a parsed .fwi header. Raw words are exposed so nothing is assumed.
type Header struct {
	CRC32         uint32    // 0x00 stored checksum
	HeaderLength  uint32    // 0x04
	Type          uint16    // 0x08
	Magic         uint16    // 0x0A
	Word0C        uint32    // 0x0C
	Word10        uint32    // 0x10
	PayloadLength uint32    // 0x14
	Word18        uint32    // 0x18
	Word1C        uint32    // 0x1C
	KeyBlock      [64]byte  // 0x20
	SectionMap    [4]uint32 // 0x60 (16 bytes)
	Pad70         [48]byte  // 0x70
}

// Parse reads the 160-byte header from data (which must be the whole file or at
// least the header + payload).
func Parse(data []byte) (*Header, error) {
	if len(data) < HeaderLen {
		return nil, fmt.Errorf("short file: %d bytes, need >= %d", len(data), HeaderLen)
	}
	h := &Header{}
	le := binary.LittleEndian
	h.CRC32 = le.Uint32(data[0x00:])
	h.HeaderLength = le.Uint32(data[0x04:])
	h.Type = le.Uint16(data[0x08:])
	h.Magic = le.Uint16(data[0x0A:])
	h.Word0C = le.Uint32(data[0x0C:])
	h.Word10 = le.Uint32(data[0x10:])
	h.PayloadLength = le.Uint32(data[0x14:])
	h.Word18 = le.Uint32(data[0x18:])
	h.Word1C = le.Uint32(data[0x1C:])
	copy(h.KeyBlock[:], data[0x20:0x60])
	for i := 0; i < 4; i++ {
		h.SectionMap[i] = le.Uint32(data[0x60+i*4:])
	}
	copy(h.Pad70[:], data[0x70:0xA0])
	return h, nil
}

// VerifyCRC recomputes the header checksum and compares it to the stored value.
//
// The checksum covers the 160-byte HEADER ONLY (bytes [0x04:0xA0]), not the
// payload: CreateFWIHeader computes ComputeChecksum(headerBytes, 4) while the
// MemoryStream still holds just the header. Verified empirically against real
// 5.6.1 / 6.6.2 / 7.4.6 images (stored == CRC(header[4:160]); CRC over
// [4:EOF] does not match). The docs' "0x04..EOF" claim is incorrect.
func VerifyCRC(data []byte) (ok bool, stored, computed uint32) {
	le := binary.LittleEndian
	stored = le.Uint32(data[0x00:])
	computed = alfencrc.ComputeChecksum(data[:HeaderLen], 4)
	return stored == computed, stored, computed
}

// Payload returns the encrypted payload slice (everything after the header).
// It uses PayloadLength when it fits, otherwise the remainder of the file.
func (h *Header) Payload(data []byte) []byte {
	end := HeaderLen + int(h.PayloadLength)
	if end <= 0 || end > len(data) {
		end = len(data)
	}
	return data[HeaderLen:end]
}

// HasWrappedKey reports whether the 0x20 block holds a real wrapped key (i.e.
// it is not the all-0xFF "absent" marker).
func (h *Header) HasWrappedKey() bool { return !allFF(h.KeyBlock[:]) }

// HasSectionMap reports whether the 0x60 section map is populated (7.x).
func (h *Header) HasSectionMap() bool {
	for _, v := range h.SectionMap {
		if v != 0xFFFFFFFF {
			return true
		}
	}
	return false
}

// LooksLikeDisplayResource reports whether the header matches the exact
// constants CreateFWIHeader writes for display-resource packages, as opposed to
// controller firmware (which shares type 0x13 but uses different words).
func (h *Header) LooksLikeDisplayResource() bool {
	return h.Type == TypeLegacy && h.Magic == MagicA1FE &&
		h.Word0C == 0xFFFFFFEF && h.Word10 == 0x0027A201 && !h.HasWrappedKey()
}

func allFF(b []byte) bool {
	for _, v := range b {
		if v != 0xFF {
			return false
		}
	}
	return true
}

func (h *Header) String() string {
	return fmt.Sprintf(
		"type=0x%02X magic=0x%04X word0C=0x%08X word10=0x%08X payloadLen=%d(0x%X) "+
			"word18=0x%08X word1C=0x%08X wrappedKey=%v sectionMap=%v(%08X,%08X,%08X,%08X)",
		h.Type, h.Magic, h.Word0C, h.Word10, h.PayloadLength, h.PayloadLength,
		h.Word18, h.Word1C, h.HasWrappedKey(), h.HasSectionMap(),
		h.SectionMap[0], h.SectionMap[1], h.SectionMap[2], h.SectionMap[3])
}

// -------------------------------------------------------------------------
// Display-resource build pipeline (port of ICUFWUCreator). Key is known here.
// -------------------------------------------------------------------------

// GetDataInBin ports ICUFWUCreator.GetDataInBin: prepend a 16-byte wrapper
// [CRC32 u32][16+len u32][0 u32][0 u32] then the data, with the CRC32 taken
// over everything from offset 4 onward.
func GetDataInBin(original []byte) []byte {
	buf := make([]byte, 16+len(original))
	le := binary.LittleEndian
	le.PutUint32(buf[0:], 0)                        // crc placeholder
	le.PutUint32(buf[4:], uint32(16+len(original))) // total length
	le.PutUint32(buf[8:], 0)
	le.PutUint32(buf[12:], 0)
	copy(buf[16:], original)
	le.PutUint32(buf[0:], alfencrc.ComputeChecksum(buf, 4))
	return buf
}

// DeflateData ports ICUFWUCreator.DeflateData: raw DEFLATE, then prepend the
// two-byte zlib header 0x78 0x9C. (Used per-object; kept for completeness.)
func DeflateData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	// CompressionLevel.Optimal ~ flate.BestCompression.
	w, err := flate.NewWriter(&b, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	out := make([]byte, 0, b.Len()+2)
	out = append(out, 0x78, 0x9C)
	out = append(out, b.Bytes()...)
	return out, nil
}

// AESEncryptDisplay ports ICUFWUCreator.AESEncrypt: AES-128-CBC, zero IV, PKCS7
// padding (RijndaelManaged defaults), using DisplayKey.
func AESEncryptDisplay(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(DisplayKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	padded := pkcs7Pad(plain, aes.BlockSize)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return out, nil
}

// AESDecryptDisplay is the inverse: AES-128-CBC, zero IV, PKCS7 unpad.
func AESDecryptDisplay(ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(DisplayKey)
	if err != nil {
		return nil, err
	}
	if len(ct)%aes.BlockSize != 0 || len(ct) == 0 {
		return nil, fmt.Errorf("ciphertext not block-aligned: %d", len(ct))
	}
	iv := make([]byte, aes.BlockSize)
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	return pkcs7Unpad(out, aes.BlockSize)
}

// CreateFWIHeader ports ICUFWUCreator.CreateFWIHeader for the display-resource
// variant (type 0x13, word0C=0xFFFFFFEF, word10=0x0027A201, empty key block).
func CreateFWIHeader(dataLength int) []byte {
	buf := make([]byte, HeaderLen)
	le := binary.LittleEndian
	le.PutUint32(buf[0x00:], 0)          // crc placeholder
	le.PutUint32(buf[0x04:], 160)        // header length
	le.PutUint32(buf[0x08:], 2717777939) // 0xA1FE0013 (magic<<16 | type)
	le.PutUint32(buf[0x0C:], 4294967279) // 0xFFFFFFEF
	le.PutUint16(buf[0x10:], 41473)      // 0xA201
	le.PutUint16(buf[0x12:], 39)         // 0x0027
	le.PutUint32(buf[0x14:], uint32(dataLength))
	le.PutUint32(buf[0x18:], 0)
	le.PutUint32(buf[0x1C:], 0xFFFFFFFF)
	for i := 0x20; i < HeaderLen; i += 4 {
		le.PutUint32(buf[i:], 0xFFFFFFFF)
	}
	le.PutUint32(buf[0x00:], alfencrc.ComputeChecksum(buf, 4))
	return buf
}

// WrapDisplayPayload ports WriteFWUFile: GetDataInBin -> AESEncrypt ->
// CreateFWIHeader -> concat. allData is the already-assembled object stream.
func WrapDisplayPayload(allData []byte) ([]byte, error) {
	bin := GetDataInBin(allData)
	ct, err := AESEncryptDisplay(bin)
	if err != nil {
		return nil, err
	}
	hdr := CreateFWIHeader(len(ct))
	out := make([]byte, 0, len(hdr)+len(ct))
	out = append(out, hdr...)
	out = append(out, ct...)
	return out, nil
}

// UnwrapDisplayPayload reverses WrapDisplayPayload for a display-resource .fwi:
// AES-decrypt the payload and strip the 16-byte GetDataInBin wrapper, returning
// the inner object stream. Errors (bad padding / CRC mismatch) mean this is not
// a display-resource image (e.g. it is controller firmware with another key).
func UnwrapDisplayPayload(file []byte) (objects []byte, err error) {
	h, err := Parse(file)
	if err != nil {
		return nil, err
	}
	plain, err := AESDecryptDisplay(h.Payload(file))
	if err != nil {
		return nil, fmt.Errorf("AES/padding: %w", err)
	}
	if len(plain) < 16 {
		return nil, errors.New("plaintext shorter than wrapper")
	}
	le := binary.LittleEndian
	storedCRC := le.Uint32(plain[0:])
	total := le.Uint32(plain[4:])
	if int(total) > len(plain) {
		return nil, fmt.Errorf("wrapper length %d > plaintext %d", total, len(plain))
	}
	// CRC is computed over bytes [4:total] (GetDataInBin computes over [4:EOF]
	// of the exact buffer; PKCS7 padding may extend plain, so bound by total).
	if got := alfencrc.ComputeChecksum(plain[:total], 4); got != storedCRC {
		return nil, fmt.Errorf("inner CRC mismatch: stored=%08X computed=%08X", storedCRC, got)
	}
	return plain[16:total], nil
}

func pkcs7Pad(b []byte, bs int) []byte {
	n := bs - len(b)%bs
	out := make([]byte, len(b)+n)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(n)
	}
	return out
}

func pkcs7Unpad(b []byte, bs int) ([]byte, error) {
	if len(b) == 0 || len(b)%bs != 0 {
		return nil, errors.New("pkcs7: invalid length")
	}
	n := int(b[len(b)-1])
	if n == 0 || n > bs || n > len(b) {
		return nil, errors.New("pkcs7: invalid padding")
	}
	for _, v := range b[len(b)-n:] {
		if int(v) != n {
			return nil, errors.New("pkcs7: bad padding bytes")
		}
	}
	return b[:len(b)-n], nil
}
