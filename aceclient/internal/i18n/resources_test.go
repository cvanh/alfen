package i18n

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"image/png"
	"os"
	"strings"
	"testing"
)

const fixtureResx = "../../../firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller.Properties.Resources.resx"

// pngFromResx re-extracts the bitmap's PNG stream from the BinaryFormatter
// blob: the System.Drawing.Bitmap "Data" member is an
// ArraySinglePrimitive record (0x0F, objectId int32, length int32,
// PrimitiveType 2 = Byte) immediately followed by the PNG bytes.
func pngFromResx(t *testing.T, resx []byte) (name string, img []byte) {
	t.Helper()
	var root struct {
		Data []struct {
			Name     string `xml:"name,attr"`
			MimeType string `xml:"mimetype,attr"`
			Value    string `xml:"value"`
		} `xml:"data"`
	}
	if err := xml.Unmarshal(resx, &root); err != nil {
		t.Fatal(err)
	}
	if len(root.Data) != 1 {
		t.Fatalf("resx has %d data entries; want exactly 1", len(root.Data))
	}
	d := root.Data[0]
	if d.MimeType != "application/x-microsoft.net.object.binary.base64" {
		t.Fatalf("mimetype = %q", d.MimeType)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(d.Value), ""))
	if err != nil {
		t.Fatal(err)
	}
	i := bytes.Index(raw, []byte("\x89PNG\r\n\x1a\n"))
	if i < 10 {
		t.Fatalf("PNG signature at %d", i)
	}
	rec := raw[i-10 : i]
	if rec[0] != 0x0F || rec[9] != 2 {
		t.Fatalf("unexpected record header % x", rec)
	}
	n := int(binary.LittleEndian.Uint32(rec[5:9]))
	if i+n > len(raw) {
		t.Fatalf("PNG length %d overruns blob", n)
	}
	return d.Name, raw[i : i+n]
}

func TestLogoAlfenSmallMatchesResx(t *testing.T) {
	resx, err := os.ReadFile(fixtureResx)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	name, want := pngFromResx(t, resx)
	if name != "logo_alfen_small" {
		t.Errorf("resource name = %q", name)
	}
	if !bytes.Equal(LogoAlfenSmall(), want) {
		t.Fatal("embedded logo_alfen_small.png differs from the resx bitmap")
	}
}

func TestLogoAlfenSmallDecodes(t *testing.T) {
	b := LogoAlfenSmall()
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 126 || cfg.Height != 32 {
		t.Errorf("size = %dx%d; want 126x32", cfg.Width, cfg.Height)
	}
	b[0] = 0 // returned slice is a copy
	if LogoAlfenSmall()[0] != 0x89 {
		t.Error("LogoAlfenSmall must return a copy")
	}
}
