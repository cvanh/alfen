package presets

import (
	"bytes"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// decodeUTF8Lenient mirrors Encoding.UTF8.GetString: invalid byte sequences
// become U+FFFD instead of failing.
func decodeUTF8Lenient(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	for len(b) > 0 {
		r, n := utf8.DecodeRune(b)
		sb.WriteRune(r) // RuneError for an invalid byte
		b = b[n:]
	}
	return sb.String()
}

// toValidUTF8 replaces invalid UTF-8 in a Go string with U+FFFD, which is what
// a .NET string (UTF-16) would contain for the same data.
func toValidUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return decodeUTF8Lenient([]byte(s))
}

// decodeUTF16 decodes UTF-16 bytes (without BOM) in the given byte order.
func decodeUTF16(b []byte, bigEndian bool) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		if bigEndian {
			u[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		} else {
			u[i] = uint16(b[2*i+1])<<8 | uint16(b[2*i])
		}
	}
	s := string(utf16.Decode(u))
	if len(b)%2 == 1 {
		s += "\uFFFD"
	}
	return s
}

// decodeUTF32 decodes UTF-32 bytes (without BOM) in the given byte order.
func decodeUTF32(b []byte, bigEndian bool) string {
	var sb strings.Builder
	for i := 0; i+4 <= len(b); i += 4 {
		var r uint32
		if bigEndian {
			r = uint32(b[i])<<24 | uint32(b[i+1])<<16 | uint32(b[i+2])<<8 | uint32(b[i+3])
		} else {
			r = uint32(b[i+3])<<24 | uint32(b[i+2])<<16 | uint32(b[i+1])<<8 | uint32(b[i])
		}
		if !utf8.ValidRune(rune(r)) {
			r = utf8.RuneError
		}
		sb.WriteRune(rune(r))
	}
	if len(b)%4 != 0 {
		sb.WriteRune(utf8.RuneError)
	}
	return sb.String()
}

// readAllText ports File.ReadAllText(path) decoding: the byte order mark
// selects UTF-8, UTF-16 LE/BE or UTF-32 LE/BE (detectEncodingFromByteOrderMarks)
// and is removed; without a BOM the bytes are read as UTF-8 with replacement.
func readAllText(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return decodeUTF8Lenient(b[3:])
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE, 0x00, 0x00}):
		return decodeUTF32(b[4:], false)
	case bytes.HasPrefix(b, []byte{0x00, 0x00, 0xFE, 0xFF}):
		return decodeUTF32(b[4:], true)
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return decodeUTF16(b[2:], false)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return decodeUTF16(b[2:], true)
	}
	return decodeUTF8Lenient(b)
}
