package fwucreator

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"alfen/aceclient/internal/alfencrc"
)

// LanguageFormat is the format byte AddLanguageToStream writes (byte.MaxValue).
const LanguageFormat ImageFormat = 0xFF

// Object is one decoded entry of a CreateFWUData object stream. It is the
// inverse of AddObjectToStream / AddLanguageToStream; the C# has no reader,
// this exists to verify and inspect resources.
type Object struct {
	Offset           int // offset of the header in the stream
	Version          byte
	Type             ObjectType
	Width            uint16
	Height           uint16
	Stride           uint16
	Size             uint32 // total object size incl. header and padding
	CompressedLength uint32
	Format           ImageFormat
	Count            byte // palette entries-1 (images) or name length (languages)
	PaletteOffset    uint16
	DataOffset       uint16
	VerticalOffset   int16
	CRCRaw           uint32
	CRCCompressed    uint32
	Palette          [][3]byte // RGB triplets (image objects with Count > 0)
	Language         string    // language name (Format == LanguageFormat)
	Compressed       []byte    // 0x78 0x9C + raw DEFLATE, as stored
}

// Decompress inflates Compressed (DeflateData writes a bare zlib header with
// no Adler-32 trailer) and checks both CRCs.
func (o Object) Decompress() ([]byte, error) {
	if got := alfencrc.ComputeChecksum(o.Compressed, 0); got != o.CRCCompressed {
		return nil, fmt.Errorf("object %v: compressed CRC %08X != stored %08X", o.Type, got, o.CRCCompressed)
	}
	if len(o.Compressed) < 2 || o.Compressed[0] != 0x78 || o.Compressed[1] != 0x9C {
		return nil, fmt.Errorf("object %v: missing 78 9C header", o.Type)
	}
	raw, err := io.ReadAll(flate.NewReader(bytes.NewReader(o.Compressed[2:])))
	if err != nil {
		return nil, fmt.Errorf("object %v: inflate: %w", o.Type, err)
	}
	if got := alfencrc.ComputeChecksum(raw, 0); got != o.CRCRaw {
		return nil, fmt.Errorf("object %v: raw CRC %08X != stored %08X", o.Type, got, o.CRCRaw)
	}
	return raw, nil
}

// ParseObjects decodes an object stream (the plaintext inside a display
// resource, e.g. from fwi.UnwrapDisplayPayload) up to its u32 0 terminator.
func ParseObjects(stream []byte) ([]Object, error) {
	le := binary.LittleEndian
	var objs []Object
	off := 0
	for {
		if off+4 <= len(stream) && le.Uint32(stream[off:]) == 0 {
			return objs, nil
		}
		if off+objectHeaderSize > len(stream) {
			return objs, errors.New("object stream not terminated")
		}
		h := stream[off:]
		o := Object{
			Offset:           off,
			Version:          h[0],
			Type:             ObjectType(h[1]),
			Width:            le.Uint16(h[2:]),
			Height:           le.Uint16(h[4:]),
			Stride:           le.Uint16(h[6:]),
			Size:             le.Uint32(h[8:]),
			CompressedLength: le.Uint32(h[12:]),
			Format:           ImageFormat(h[16]),
			Count:            h[17],
			PaletteOffset:    le.Uint16(h[18:]),
			DataOffset:       le.Uint16(h[20:]),
			VerticalOffset:   int16(le.Uint16(h[22:])),
			CRCRaw:           le.Uint32(h[24:]),
			CRCCompressed:    le.Uint32(h[28:]),
		}
		if o.Version != 1 {
			return objs, fmt.Errorf("object at %d: version %d", off, o.Version)
		}
		if o.Size < objectHeaderSize || off+int(o.Size) > len(stream) {
			return objs, fmt.Errorf("object at %d: bad size %d", off, o.Size)
		}
		body := stream[off : off+int(o.Size)]
		if int(o.DataOffset)+int(o.CompressedLength) > len(body) {
			return objs, fmt.Errorf("object at %d: data %d+%d beyond size %d", off, o.DataOffset, o.CompressedLength, o.Size)
		}
		if o.Format == LanguageFormat {
			name := body[objectHeaderSize : objectHeaderSize+int(o.Count)]
			o.Language = string(name)
		} else if o.Count > 0 {
			n := int(o.Count) + 1
			if int(o.PaletteOffset)+n*3 > len(body) {
				return objs, fmt.Errorf("object at %d: palette beyond object", off)
			}
			for i := 0; i < n; i++ {
				p := body[int(o.PaletteOffset)+i*3:]
				o.Palette = append(o.Palette, [3]byte{p[0], p[1], p[2]})
			}
		}
		o.Compressed = body[o.DataOffset : int(o.DataOffset)+int(o.CompressedLength)]
		objs = append(objs, o)
		off += int(o.Size)
	}
}

// LanguageRecord is one 88-byte AddLanguage record.
type LanguageRecord struct {
	ID   DisplayString // stored as a byte; DISP_EMPTY (-1) reads back as 255
	Font byte          // (byte)Convert.ToUInt16(column[^3])
	X    uint16        // column[^2]
	Y    uint16        // column[^1]
	Text []byte        // ISO-8859-1, NUL padded to 80 bytes
}

// ParseLanguageRecords splits a decompressed OBJECT_LANGUAGE payload.
func ParseLanguageRecords(data []byte) ([]LanguageRecord, error) {
	if len(data)%LanguageRecordSize != 0 {
		return nil, fmt.Errorf("language data %d bytes is not a multiple of %d", len(data), LanguageRecordSize)
	}
	le := binary.LittleEndian
	var recs []LanguageRecord
	for off := 0; off < len(data); off += LanguageRecordSize {
		r := data[off : off+LanguageRecordSize]
		if r[1] != languageTextSize || r[3] != 0 {
			return nil, fmt.Errorf("record at %d: unexpected fixed bytes %02X %02X", off, r[1], r[3])
		}
		recs = append(recs, LanguageRecord{
			ID:   DisplayString(r[0]),
			Font: r[2],
			X:    le.Uint16(r[4:]),
			Y:    le.Uint16(r[6:]),
			Text: append([]byte(nil), r[8:]...),
		})
	}
	return recs, nil
}
