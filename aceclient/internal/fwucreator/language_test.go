package fwucreator

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testdata/lang_golden.json was produced by running the decompiled
// ICUFWUCreator.AddLanguage (compiled unmodified, minus the System.Drawing
// members, with a Serilog stub) over the installer's Lang_Eve_Mini_*.csv and
// the edge cases in testdata/lang. Header = bytes [0:8] | [16:24] | [24:28]
// of the object (the compressed-length fields depend on the deflater).
type langGolden struct {
	File      string `json:"file"`
	Language  string `json:"language"`
	RawLen    int    `json:"rawLen"`
	RawSHA256 string `json:"rawSHA256"`
	Header    string `json:"header"`
	RawHex    string `json:"rawHex"`
	Error     string `json:"error"`
}

const installerFiles = "../../../firmware/msi_work/files3"

func langPath(file string) string {
	if strings.HasPrefix(file, "Lang_Eve_Mini_") {
		return filepath.Join(installerFiles, file)
	}
	return filepath.Join("testdata", "lang", file)
}

func TestAddLanguageMatchesDecompiledCSharp(t *testing.T) {
	raw, err := os.ReadFile("testdata/lang_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden []langGolden
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	for _, g := range golden {
		t.Run(g.File, func(t *testing.T) {
			p := langPath(g.File)
			if _, err := os.Stat(p); err != nil {
				t.Skipf("fixture missing: %v", err)
			}
			var w bytes.Buffer
			err := AddLanguage(&w, ObjectLanguage, p)
			if g.Error != "" {
				// e.g. "ArgumentException: Requested value '…' was not found."
				msg := g.Error[strings.Index(g.Error, ": ")+2:]
				if err == nil || err.Error() != msg {
					t.Fatalf("err %v, C# %s", err, g.Error)
				}
				if w.Len() != 0 {
					t.Fatalf("wrote %d bytes on error", w.Len())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			objs, err := ParseObjects(append(w.Bytes(), 0, 0, 0, 0))
			if err != nil || len(objs) != 1 {
				t.Fatalf("parse: %v (%d objs)", err, len(objs))
			}
			o := objs[0]
			data, err := o.Decompress()
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(data)
			if len(data) != g.RawLen || hex.EncodeToString(sum[:]) != g.RawSHA256 {
				if wantRaw, _ := hex.DecodeString(g.RawHex); len(wantRaw) == len(data) {
					for i := range data {
						if data[i] != wantRaw[i] {
							t.Fatalf("raw differs at %d (record %d byte %d): %02X, C# %02X", i, i/LanguageRecordSize, i%LanguageRecordSize, data[i], wantRaw[i])
						}
					}
				}
				t.Fatalf("raw %d bytes sha %x, C# %d %s", len(data), sum, g.RawLen, g.RawSHA256)
			}
			if o.Language != g.Language {
				t.Fatalf("language %q, C# %q", o.Language, g.Language)
			}
			b := w.Bytes()
			hdr := strings.ToUpper(hex.EncodeToString(b[0:8]) + "|" + hex.EncodeToString(b[16:24]) + "|" + hex.EncodeToString(b[24:28]))
			if hdr != g.Header {
				t.Fatalf("header %s, C# %s", hdr, g.Header)
			}
			// Remaining fields: size = 32 + compressed + name, 4-aligned.
			size := binary.LittleEndian.Uint32(b[8:])
			if size%4 != 0 || int(size) != len(b) || int(size) < objectHeaderSize+len(o.Compressed)+len(o.Language) {
				t.Fatalf("size %d, object %d bytes", size, len(b))
			}
			if _, err := ParseLanguageRecords(data); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestLanguageEdgeRecords spells out what the edge-case CSV must produce
// (independently of the golden hash).
func TestLanguageEdgeRecords(t *testing.T) {
	lang, data, err := buildLanguageData("testdata/lang/Lang_Edge_XX.csv")
	if err != nil {
		t.Fatal(err)
	}
	if lang != "xx_YY" {
		t.Fatalf("language %q", lang)
	}
	recs, err := ParseLanguageRecords(data)
	if err != nil {
		t.Fatal(err)
	}
	text := func(r LanguageRecord) string { return string(bytes.TrimRight(r.Text, "\x00")) }
	want := []struct {
		id    DisplayString
		text  string
		font  byte
		x, y  uint16
		check bool
	}{
		{DispCardAccepted, "Hello, world, with commas", 29, 160, 28, true},
		{5, "numeric id", 1, 2, 3, true},
		{DispGoodbye, "tab\tquote", 7, 8, 65535, true},
		{DispError, "\xdcn\xefc\xf6d\xe9 \xf1   \n newline", 29, 160, 28, true},
		{255, "empty id", 0, 0, 0, true},
		{DispError, "", 1, 1, 1, false},
		{DispCharging, "x", 44, 1, 1, true}, // (byte)300
		{DispTotal, "tabs", 1, 2, 3, true},
	}
	if len(recs) != len(want) {
		t.Fatalf("%d records, want %d", len(recs), len(want))
	}
	for i, w := range want {
		r := recs[i]
		if r.ID != w.id || r.Font != w.font || r.X != w.x || r.Y != w.y || (w.check && text(r) != w.text) {
			t.Errorf("record %d = {%v %q %d %d %d}, want %+v", i, r.ID, text(r), r.Font, r.X, r.Y, w)
		}
	}
	long := recs[5]
	if long.Text[78] == 0 || long.Text[79] != 0 {
		t.Errorf("long text not truncated to 79 bytes + NUL: %q", long.Text)
	}
}
