package fwucreator

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"alfen/aceclient/internal/alfencrc"
	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/logo"
)

// goldenPaletteImage mirrors the C# harness's MakeImage/logo inputs
// (testdata/object_golden.json was produced by the decompiled
// AddObjectToStream with stubbed Image/ColorPalette types).
func goldenPaletteImage(entries int) *image.Paletted {
	pal := make(color.Palette, entries)
	for i := range pal {
		a := uint8(255)
		if i >= 16 && i < 40 {
			a = 0
		}
		pal[i] = color.NRGBA{uint8(i), uint8(255 - i), uint8(i * 3), a}
	}
	img := image.NewPaletted(image.Rect(0, 0, 300, 140), pal)
	for i := range img.Pix {
		img.Pix[i] = uint8((i * 7) % 128)
	}
	return img
}

func TestAddObjectToStreamMatchesDecompiledCSharp(t *testing.T) {
	raw, err := os.ReadFile("testdata/object_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var golden []struct{ Name, Object string }
	if err := json.Unmarshal(raw, &golden); err != nil {
		t.Fatal(err)
	}
	type call struct {
		data          []byte
		typ           ObjectType
		format        ImageFormat
		w, h, stride  int
		maxColors, vo int
		img           *image.Paletted
	}
	logoData := GetImage(goldenPaletteImage(256))
	calls := map[string]call{
		"logo_accepted":      {LogoAcceptedL1, ObjectLogoAccepted, ImageFormatL1, 56, 59, 7, 0, 0, nil},
		"logo_charging":      {LogoChargingL1, ObjectLogoCharging, ImageFormatL1, 80, 50, 10, 0, 0, nil},
		"logo_error":         {LogoErrorL1, ObjectLogoSocketerror, ImageFormatL1, 56, 56, 7, 0, 0, nil},
		"logo_communicating": {LogoCommunicatingL1, ObjectLogoCommunicating, ImageFormatL1, 56, 78, 7, 0, 0, nil},
		"font_28":            {FontRobotica28, ObjectRoboticaRegular28, ImageFormatL4, 20, 28, 10, 0, -2, nil},
		"font_29":            {FontRobotica29, ObjectRoboticaRegular29, ImageFormatL4, 22, 30, 11, 0, -2, nil},
		"font_30":            {FontRobotica30, ObjectRoboticaRegular30, ImageFormatL4, 30, 41, 15, 0, -4, nil},
		"customer_256_128":   {logoData, ObjectLogoCustomer, ImagePaletted8, 300, 140, 300, 128, 0, goldenPaletteImage(256)},
		"customer_16_128":    {logoData, ObjectLogoCustomer, ImagePaletted8, 300, 140, 300, 128, 0, goldenPaletteImage(16)},
		"customer_256_1":     {logoData, ObjectLogoCustomer, ImagePaletted8, 300, 140, 300, 1, 0, goldenPaletteImage(256)},
	}
	le := binary.LittleEndian
	for _, g := range golden {
		t.Run(g.Name, func(t *testing.T) {
			c, ok := calls[g.Name]
			if !ok {
				t.Fatalf("no Go call for %s", g.Name)
			}
			cs, _ := hex.DecodeString(g.Object)
			var buf bytes.Buffer
			if err := AddObjectToStream(&buf, c.data, c.typ, c.format, c.w, c.h, c.stride, c.maxColors, c.vo, c.img); err != nil {
				t.Fatal(err)
			}
			goObj := buf.Bytes()
			if !bytes.Equal(goObj[0:8], cs[0:8]) {
				t.Fatalf("[0:8] %x, C# %x", goObj[0:8], cs[0:8])
			}
			if !bytes.Equal(goObj[16:28], cs[16:28]) {
				t.Fatalf("[16:28] %x, C# %x", goObj[16:28], cs[16:28])
			}
			palLen := 0
			if c.img != nil {
				palLen = min(len(c.img.Palette), c.maxColors) * 3
			}
			if !bytes.Equal(goObj[32:32+palLen], cs[32:32+palLen]) {
				t.Fatalf("palette bytes differ")
			}
			// Everything that depends on the deflater: compare the
			// deflater-independent part of the size, and inflate both.
			base := func(o []byte) (int, []byte) {
				comp := int(le.Uint32(o[12:]))
				data := o[32+palLen : 32+palLen+comp]
				pad := len(o) - 32 - palLen - comp
				return int(le.Uint32(o[8:])) - comp - pad, data
			}
			gb, gcomp := base(goObj)
			cb, ccomp := base(cs)
			if gb != cb {
				t.Fatalf("size-compressed-pad %d, C# %d", gb, cb)
			}
			if le.Uint32(goObj[8:])%4 != 0 || alfencrc.ComputeChecksum(gcomp, 0) != le.Uint32(goObj[28:]) {
				t.Fatalf("size alignment / compressed CRC wrong")
			}
			for _, comp := range [][]byte{gcomp, ccomp} {
				if comp[0] != 0x78 || comp[1] != 0x9C {
					t.Fatalf("zlib header %x", comp[:2])
				}
				got, err := io.ReadAll(flate.NewReader(bytes.NewReader(comp[2:])))
				if err != nil || !bytes.Equal(got, c.data) {
					t.Fatalf("inflate: %v", err)
				}
			}
		})
	}
}

// copyLanguages copies the installer's Lang_Eve_Mini_*.csv into a temp
// "UILanguages" folder (AddAllLanguageFilesFromFolder takes every file).
func copyLanguages(t *testing.T) string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(installerFiles, "Lang_Eve_Mini_*.csv"))
	if len(matches) == 0 {
		t.Skip("installer language CSVs missing")
	}
	dir := filepath.Join(t.TempDir(), UILanguagesFolder)
	os.MkdirAll(dir, 0o755)
	for _, m := range matches {
		b, _ := os.ReadFile(m)
		os.WriteFile(filepath.Join(dir, filepath.Base(m)), b, 0o644)
	}
	return dir
}

func convertedLogo(t *testing.T, w, h int) *image.Paletted {
	t.Helper()
	img, _, err := logo.LoadImage(filepath.Join(installerFiles, DefaultLogoFile))
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	out, err := logo.ConvertImage(img, w, h, MaxColors)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestCreateFWUDataRoundTrip builds a display resource exactly as the logo
// dialog does, unwraps it with internal/fwi and checks every object.
func TestCreateFWUDataRoundTrip(t *testing.T) {
	langDir := copyLanguages(t)
	img := convertedLogo(t, MaxLogoWidth-2*DefaultImageMargin, MaxLogoHeight-2*DefaultImageMargin)
	for _, large := range []bool{false, true} {
		file, err := CreateFWUData(img, MaxColors, langDir, false, "", large)
		if err != nil {
			t.Fatal(err)
		}
		h, err := fwi.Parse(file)
		if err != nil || !h.LooksLikeDisplayResource() {
			t.Fatalf("header %v %v", h, err)
		}
		if ok, _, _ := fwi.VerifyCRC(file); !ok {
			t.Fatal("FWI header CRC")
		}
		if int(h.PayloadLength) != len(file)-fwi.HeaderLen {
			t.Fatalf("payload length %d, file %d", h.PayloadLength, len(file))
		}
		stream, err := fwi.UnwrapDisplayPayload(file)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasSuffix(stream, []byte{0, 0, 0, 0}) {
			t.Fatal("no u32 0 terminator")
		}
		objs, err := ParseObjects(stream)
		if err != nil {
			t.Fatal(err)
		}
		wantTypes := []ObjectType{ObjectLogoCustomer, ObjectLogoAccepted, ObjectLogoCharging, ObjectLogoSocketerror,
			ObjectLogoCommunicating, ObjectRoboticaRegular28, ObjectRoboticaRegular29}
		if large {
			wantTypes = append(wantTypes, ObjectRoboticaRegular30)
		}
		nLang := len(objs) - len(wantTypes)
		if nLang != 10 {
			t.Fatalf("large=%v: %d language objects", large, nLang)
		}
		end := 0
		for i, o := range objs {
			if i < len(wantTypes) && o.Type != wantTypes[i] {
				t.Fatalf("object %d is %v, want %v", i, o.Type, wantTypes[i])
			}
			if o.Size%4 != 0 || o.Offset%4 != 0 {
				t.Fatalf("object %d misaligned", i)
			}
			data, err := o.Decompress()
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case i == 0:
				if o.Format != ImagePaletted8 || int(o.Width) != img.Rect.Dx() || int(o.Height) != img.Rect.Dy() ||
					o.Stride != o.Width || o.Count != MaxColors-1 || o.PaletteOffset != 32 || o.DataOffset != 32+MaxColors*3 {
					t.Fatalf("customer header %+v", o)
				}
				if !bytes.Equal(data, img.Pix) {
					t.Fatal("customer pixels differ")
				}
				for k, p := range o.Palette {
					c := img.Palette[k].(color.NRGBA)
					if p != [3]byte{c.R, c.G, c.B} {
						t.Fatalf("palette %d", k)
					}
				}
			case i < len(wantTypes):
				b := builtinRobotica30
				if i-1 < len(builtinObjects) {
					b = builtinObjects[i-1]
				}
				if !bytes.Equal(data, b.Data) || int(o.Width) != b.Width || int(o.Height) != b.Height ||
					int(o.Stride) != b.Stride || int(o.VerticalOffset) != b.VerticalOffset || o.Format != b.Format ||
					o.Count != 0 || o.PaletteOffset != 0 || o.DataOffset != 32 {
					t.Fatalf("builtin %v header/data mismatch: %+v", o.Type, o)
				}
			default:
				if o.Type != ObjectLanguage || o.Format != LanguageFormat || len(data) != 50*LanguageRecordSize {
					t.Fatalf("language object %d: %+v (%d bytes)", i, o, len(data))
				}
			}
			end = o.Offset + int(o.Size)
		}
		if end+4 != len(stream) {
			t.Fatalf("terminator at %d, stream %d", end, len(stream))
		}
		var langs []string
		for _, o := range objs[len(wantTypes):] {
			langs = append(langs, o.Language)
		}
		want := "nl_NL,en_GB,de_DE,es_ES,fi_FI,fr_FR,it_IT,nn_NO,pt_PT,sv_SE"
		if got := strings.Join(langs, ","); got != want {
			t.Fatalf("language order %s, want %s", got, want)
		}
	}
}

func TestCreateFWUDataCFile(t *testing.T) {
	langDir := filepath.Join(t.TempDir(), "missing", UILanguagesFolder)
	img := goldenPaletteImage(256)
	cdir := t.TempDir()
	file, err := CreateFWUData(img, MaxColors, langDir, true, cdir, false)
	if err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(langDir); err != nil || !st.IsDir() {
		t.Fatal("language folder not created")
	}
	stream, err := fwi.UnwrapDisplayPayload(file)
	if err != nil {
		t.Fatal(err)
	}
	objs, err := ParseObjects(stream)
	if err != nil || len(objs) != 8 || objs[7].Type != ObjectRoboticaRegular30 {
		t.Fatalf("createCFile must add Robotica 30 and no languages: %d objs %v", len(objs), err)
	}
	c, err := os.ReadFile(filepath.Join(cdir, "default_objects.c"))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	AppendCFileBlock(&sb, "object_data", stream)
	if string(c) != sb.String() {
		t.Fatal("C file differs from AppendCFileBlock(stream)")
	}
}

func TestAppendCFileBlock(t *testing.T) {
	var sb strings.Builder
	data := make([]byte, 18)
	for i := range data {
		data[i] = byte(i * 15)
	}
	AppendCFileBlock(&sb, "x", data)
	want := "const uint8_t x[18] = {\n0x00, 0x0F, 0x1E, 0x2D, 0x3C, 0x4B, 0x5A, 0x69, 0x78, 0x87, 0x96, 0xA5, 0xB4, 0xC3, 0xD2, 0xE1, \r\n0xF0, 0xFF\n};\r\n\r\n"
	if sb.String() != want {
		t.Fatalf("got %q", sb.String())
	}
}

func TestAddAllLanguageFilesOrderAndAbort(t *testing.T) {
	src, err := os.ReadFile("testdata/lang/Lang_NoHeader_QQ.csv")
	if err != nil {
		t.Fatal(err)
	}
	bad, _ := os.ReadFile("testdata/lang/Lang_BadEnum_ZZ.csv")
	dir := t.TempDir()
	for _, n := range []string{"b_FR.csv", "x_DE.csv", "a_AA.csv", "y_EN.csv", "z_NL.csv", "c_PT.csv"} {
		os.WriteFile(filepath.Join(dir, n), []byte("LANGUAGE: "+n+"\n"+string(src)[strings.Index(string(src), "\r")+1:]), 0o644)
	}
	var w bytes.Buffer
	if err := AddAllLanguageFilesFromFolder(&w, dir); err != nil {
		t.Fatal(err)
	}
	objs, err := ParseObjects(append(w.Bytes(), 0, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, o := range objs {
		got = append(got, o.Language)
	}
	if strings.Join(got, ",") != "z_NL.csv,y_EN.csv,x_DE.csv,a_AA.csv,b_FR.csv,c_PT.csv" {
		t.Fatalf("order %v", got)
	}
	// A failing file stops the remaining ones (the C# catch is outside the loop).
	os.WriteFile(filepath.Join(dir, "b0_bad.csv"), bad, 0o644)
	w.Reset()
	if err := AddAllLanguageFilesFromFolder(&w, dir); err == nil {
		t.Fatal("bad file not reported")
	}
	objs, _ = ParseObjects(append(w.Bytes(), 0, 0, 0, 0))
	got = got[:0]
	for _, o := range objs {
		got = append(got, o.Language)
	}
	if strings.Join(got, ",") != "z_NL.csv,y_EN.csv,x_DE.csv,a_AA.csv" {
		t.Fatalf("after abort %v", got)
	}
}

func TestListFilesLikeWindows(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"b.csv", "A.csv", "a_.csv", "C.csv"} {
		os.WriteFile(filepath.Join(dir, n), nil, 0o644)
	}
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)
	got, err := listFilesLikeWindows(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A.csv", "a_.csv", "b.csv", "C.csv"}
	if !sort.StringsAreSorted([]string{strings.ToUpper(got[0]), strings.ToUpper(got[1])}) || strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestParseObjectsErrors(t *testing.T) {
	if _, err := ParseObjects([]byte{1, 2, 3}); err == nil {
		t.Fatal("short stream accepted")
	}
	bad := make([]byte, 40)
	bad[0] = 2
	if _, err := ParseObjects(bad); err == nil {
		t.Fatal("bad version accepted")
	}
}
