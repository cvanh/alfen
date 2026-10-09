package i18n

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Tip ports ICUServiceInstaller.Tip (Tip.cs): one parsed tooltip line.
// Tooltip is stored raw — the literal two-character sequence `\n` is only
// turned into a newline by Tooltips.GetTooltip, never by the parser.
//
// The C# properties are get-only, so a Tip is immutable there. Here
// TooltipCollection.Get hands out a copy for the same effect; the Tips slice
// of a collection obtained from Tooltips is shared and must not be modified.
type Tip struct {
	ID        string // column 0, e.g. "2062_00" (see TooltipID)
	Name      string // column 1, object-dictionary name, e.g. "OD_sysMaxStationCurrent"
	LabelText string // column 2, UI label
	Tooltip   string // columns 3.. re-joined with ","
}

// TooltipCollection ports ICUServiceInstaller.TooltipCollection
// (TooltipCollection.cs), a List<Tip> tagged with its language. Tips keeps
// file order, duplicates included.
type TooltipCollection struct {
	Language Language
	Tips     []Tip
}

// Get ports TooltipCollection.Get: the FIRST tip whose ID equals tooltipID
// (ordinal, case-sensitive), or nil. Later duplicates are unreachable, as in
// the C# (FirstOrDefault). The result is a copy, so writing to it never
// changes the collection (the C# Tip is read-only). Safe on a nil receiver.
func (c *TooltipCollection) Get(tooltipID string) *Tip {
	i := c.index(tooltipID)
	if i < 0 {
		return nil
	}
	tip := c.Tips[i]
	return &tip
}

// index is the FirstOrDefault search behind Get: the position of the first
// tip with ID == tooltipID, or -1.
func (c *TooltipCollection) index(tooltipID string) int {
	if c == nil {
		return -1
	}
	for i := range c.Tips {
		if c.Tips[i].ID == tooltipID {
			return i
		}
	}
	return -1
}

// ErrUnknownLanguage is returned by ParseTooltipFile when the file name does
// not map to a Language (Tooltips.ParseFile returns null in that case).
var ErrUnknownLanguage = errors.New("i18n: tooltip file name has no known language")

// ParseTooltips ports the body of Tooltips.ParseFile (Tooltips.cs) after the
// language has been determined:
//
//	using StreamReader streamReader = new StreamReader(path);
//	while ((text = streamReader.ReadLine()) != null) {
//	    string[] array = text.Split(',');
//	    if (array.Length >= 4)
//	        Add(new Tip(array[0].Trim(), array[1].Trim(), array[2].Trim(),
//	                    string.Join(",", array, 3, array.Count() - 3).Trim()));
//	}
//
// Decoding follows new StreamReader(path): UTF-8 by default, with a UTF-8 /
// UTF-16 / UTF-32 byte-order mark detected and stripped. Lines end at "\r\n",
// "\r" or "\n". A read error yields (nil, err); the C# swallows it and
// returns null, which Tooltips does by skipping the file.
func ParseTooltips(r io.Reader, lang Language) (*TooltipCollection, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	c := &TooltipCollection{Language: lang}
	for _, line := range readLines(decodeStreamText(b)) {
		if tip, ok := parseTipLine(line); ok {
			c.Tips = append(c.Tips, tip)
		}
		// else: the C# evaluates text.StartsWith("//") and discards the
		// result — comments and blank lines are skipped only because they
		// have fewer than four fields.
	}
	return c, nil
}

// parseTipLine is the per-line part of Tooltips.ParseFile. A line with fewer
// than three commas is not a tip. A "//" comment WITH three commas is parsed
// as a tip, exactly as the C# does.
func parseTipLine(text string) (Tip, bool) {
	a := strings.Split(text, ",")
	if len(a) < 4 {
		return Tip{}, false
	}
	return Tip{
		ID:        netTrim(a[0]),
		Name:      netTrim(a[1]),
		LabelText: netTrim(a[2]),
		Tooltip:   netTrim(strings.Join(a[3:], ",")),
	}, true
}

// netTrim is .NET String.Trim(): it strips char.IsWhiteSpace characters.
// That set equals Go's unicode.IsSpace (ASCII \t\n\v\f\r and space, U+0085,
// U+00A0, U+1680, U+2000–U+200A, U+2028, U+2029, U+202F, U+205F, U+3000).
func netTrim(s string) string { return strings.TrimSpace(s) }

// readLines splits like repeated StreamReader.ReadLine(): terminators are
// "\r\n", "\r" and "\n"; a terminator at end of input does not produce an
// extra empty line.
func readLines(s string) []string {
	var lines []string
	for len(s) > 0 {
		i := strings.IndexAny(s, "\r\n")
		if i < 0 {
			lines = append(lines, s)
			break
		}
		lines = append(lines, s[:i])
		if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			s = s[i+2:]
		} else {
			s = s[i+1:]
		}
	}
	return lines
}

// decodeStreamText mirrors the decoding of new StreamReader(path)
// (encoding UTF-8, detectEncodingFromByteOrderMarks = true). BOM detection
// follows StreamReader.DetectEncoding: FE FF -> UTF-16BE; FF FE 00 00 ->
// UTF-32LE; FF FE -> UTF-16LE; EF BB BF -> UTF-8; 00 00 FE FF -> UTF-32BE.
//
// Invalid sequences decode to U+FFFD (the .NET replacement fallback); for
// malformed UTF-8 this emits one U+FFFD per invalid byte, which can differ
// from .NET's grouping of an invalid multi-byte subsequence into a single
// U+FFFD (unverified without .NET Framework 4.8).
//
// An incomplete unit at END of input is dropped, not replaced: the .NET
// Framework 4.8 StreamReader.ReadBuffer calls Decoder.GetChars without
// flush and returns at EOF without ever flushing, so the decoder keeps those
// bytes pending forever. That covers a truncated UTF-8 sequence, an odd
// UTF-16 byte, a trailing UTF-16 high surrogate and a 1–3 byte UTF-32 tail.
func decodeStreamText(b []byte) string {
	switch {
	case len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF:
		return decodeUTF16(b[2:], binary.BigEndian)
	case len(b) >= 4 && b[0] == 0xFF && b[1] == 0xFE && b[2] == 0 && b[3] == 0:
		return decodeUTF32(b[4:], binary.LittleEndian)
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE:
		return decodeUTF16(b[2:], binary.LittleEndian)
	case len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF:
		return decodeUTF8(b[3:])
	case len(b) >= 4 && b[0] == 0 && b[1] == 0 && b[2] == 0xFE && b[3] == 0xFF:
		return decodeUTF32(b[4:], binary.BigEndian)
	}
	return decodeUTF8(b)
}

// decodeUTF8 is Encoding.UTF8 (replacement fallback) through a decoder that
// is never flushed: a valid-so-far but truncated sequence at the end of b
// (utf8.FullRune false) stays pending and is dropped.
func decodeUTF8(b []byte) string {
	if utf8.Valid(b) {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b))
	for len(b) > 0 {
		if !utf8.FullRune(b) {
			break // truncated sequence at EOF: never flushed
		}
		r, n := utf8.DecodeRune(b)
		sb.WriteRune(r) // RuneError for each invalid byte
		b = b[n:]
	}
	return sb.String()
}

// decodeUTF16 is UnicodeEncoding (replacement fallback) through a decoder
// that is never flushed: an odd trailing byte and a high surrogate left
// waiting for its low half at the end are both dropped. Unpaired surrogates
// elsewhere become U+FFFD.
func decodeUTF16(b []byte, order binary.ByteOrder) string {
	u := make([]uint16, len(b)/2) // odd trailing byte dropped
	for i := range u {
		u[i] = order.Uint16(b[2*i:])
	}
	if n := len(u); n > 0 && u[n-1] >= 0xD800 && u[n-1] <= 0xDBFF {
		u = u[:n-1] // pending high surrogate at EOF
	}
	return string(utf16.Decode(u))
}

// decodeUTF32 is UTF32Encoding (replacement fallback) through a decoder that
// is never flushed: a 1–3 byte tail is dropped. Surrogates and values above
// U+10FFFF become U+FFFD.
func decodeUTF32(b []byte, order binary.ByteOrder) string {
	var sb strings.Builder
	for len(b) >= 4 {
		r := rune(order.Uint32(b))
		if !utf8.ValidRune(r) {
			r = utf8.RuneError
		}
		sb.WriteRune(r)
		b = b[4:]
	}
	return sb.String()
}
