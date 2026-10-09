package fwucreator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alfen/aceclient/internal/alfencrc"
	"alfen/aceclient/internal/tvf"
)

func TestNewtonsoftString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"logo", `"logo"`},
		{`a"b\c`, `"a\"b\\c"`},
		{"t\tn\nr\rf\fb\b", `"t\tn\nr\rf\fb\b"`},
		{"\x01\x1f", `"\u0001\u001f"`},
		{"<&>' é €", `"<&>' é €"`},
		{"\u0085\u2028\u2029", "\"\\u0085\\u2028\\u2029\""},
	}
	for _, tc := range tests {
		if got := newtonsoftString(tc.in); got != tc.want {
			t.Errorf("%q -> %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestManifestJSON(t *testing.T) {
	now := time.Date(2026, 10, 10, 8, 5, 3, 999, time.UTC)
	m := NewAhpManifest(AppName, "logo_alfen", 1, now)
	want := `{"manifest-version":1,"meta":{"description":"logo_alfen","created-by":"ACE Service Installer","created-at":"2026-10-10T08:05:03Z"},"options":{"reboot-scb-after-stage-install":false,"wait-for-charging-sessions-to-finish":false}}`
	if got := m.JSON(); got != want {
		t.Fatalf("manifest\n got %s\nwant %s", got, want)
	}
	v := NewAhpVideoResource("logo_alfen.png", 8)
	if got := v.JSON(); got != `{"logo":"logo_alfen.png","logoMargins":[8,8,8,8]}` {
		t.Fatalf("video resource %s", got)
	}
}

func TestSharpTarHeaderFormat(t *testing.T) {
	mt := time.Unix(0x5F5E100, 0)
	ar := writeSharpTar([]sharpTarEntry{{Name: "a.json", Data: []byte("hello"), ModTime: mt}})
	if len(ar) != tarRecordSize {
		t.Fatalf("archive %d bytes, want one 10240-byte record", len(ar))
	}
	h := ar[:512]
	checks := []struct {
		off, n int
		want   string
	}{
		{0, 7, "a.json\x00"},
		{100, 8, "0000644\x00"},
		{108, 8, "0000000\x00"},
		{116, 8, "0000000\x00"},
		{124, 12, "00000000005\x00"},
		{136, 12, "00575360400\x00"},
		{155, 2, " 0"}, // checksum trailer kept as ' ', then typeflag '0'
		{257, 8, "ustar\x00 \x00"},
		{265, 5, "user\x00"},
		{297, 5, "None\x00"},
	}
	for _, c := range checks {
		if got := string(h[c.off : c.off+c.n]); got != c.want {
			t.Errorf("header[%d:%d] = %q, want %q", c.off, c.off+c.n, got, c.want)
		}
	}
	var sum int
	for i, b := range h {
		if i >= 148 && i < 156 {
			b = ' '
		}
		sum += int(b)
	}
	if got := string(h[148:155]); got != strings.Repeat("0", 6-len(itoa8(sum)))+itoa8(sum)+"\x00" {
		t.Errorf("checksum field %q, sum %o", got, sum)
	}
	// Go's reader accepts the SharpZipLib layout.
	tr := tar.NewReader(bytes.NewReader(ar))
	hdr, err := tr.Next()
	if err != nil || hdr.Name != "a.json" || hdr.Size != 5 || hdr.Mode != 0o644 || !hdr.ModTime.Equal(mt) || hdr.Uname != "user" || hdr.Gname != "None" {
		t.Fatalf("reader: %+v %v", hdr, err)
	}
	// A name over 100 characters gets a GNU ././@LongLink entry first.
	long := strings.Repeat("n", 120)
	ar = writeSharpTar([]sharpTarEntry{{Name: long, Data: []byte("x"), ModTime: mt}})
	if ar[156] != 'L' || string(ar[0:13]) != "././@LongLink" || string(ar[512:632]) != long {
		t.Fatal("LongLink layout")
	}
	hdr, err = tar.NewReader(bytes.NewReader(ar)).Next()
	if err != nil || hdr.Name != long {
		t.Fatalf("long name: %v %v", hdr, err)
	}
}

func itoa8(v int) string {
	s := ""
	for v > 0 {
		s = string(rune('0'+v%8)) + s
		v /= 8
	}
	return s
}

func TestCreateTvfData(t *testing.T) {
	src := filepath.Join(installerFiles, DefaultLogoFile)
	png, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	st, _ := os.Stat(src)
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	out, err := createTvfData(src, AppName, 8, DefaultManifestVersion, now)
	if err != nil {
		t.Fatal(err)
	}
	hdrLen := int(binary.LittleEndian.Uint16(out[2:])) // CalculateHeaderLength
	if hdrLen != 1350+len("1")+len("logo_alfen") {
		t.Fatalf("header length %d", hdrLen)
	}
	if binary.LittleEndian.Uint32(out[4:]) != tvf.MagicTVF {
		t.Fatal("TVF magic")
	}
	body := out[:hdrLen-4]
	if binary.LittleEndian.Uint32(out[hdrLen-4:]) != alfencrc.ComputeChecksum(body, 0) {
		t.Fatal("TVF header CRC")
	}
	dataLength := int(binary.LittleEndian.Uint32(out[8:]))
	if !bytes.Equal(out[:hdrLen], tvf.BuildHeader(DefaultManifestVersion, "logo_alfen", dataLength)) {
		t.Fatal("header differs from tvf.BuildHeader")
	}
	if sn := tvf.Sniff(out); sn.CertPresent {
		t.Fatal("local resource TVF must be unsigned")
	}
	gz := out[hdrLen:]
	if !bytes.Equal(gz[:4], []byte{0x1f, 0x8b, 8, 0}) || binary.LittleEndian.Uint32(gz[4:]) != uint32(now.Unix()) || gz[8] != 0 || gz[9] != 255 {
		t.Fatalf("gzip header % x", gz[:10])
	}
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatal(err)
	}
	outer, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if len(outer)%tarRecordSize != 0 {
		t.Fatalf("outer tar %d not record aligned", len(outer))
	}
	tr := tar.NewReader(bytes.NewReader(outer))
	h, err := tr.Next()
	if err != nil || h.Name != InnerPackageName || h.Mode != 0o644 {
		t.Fatalf("outer entry %+v %v", h, err)
	}
	inner, _ := io.ReadAll(tr)
	if _, err := tr.Next(); err != io.EOF {
		t.Fatal("outer tar has more than one entry")
	}
	if len(inner) != dataLength || len(inner)%tarRecordSize != 0 {
		t.Fatalf("inner tar %d bytes, header dataLength %d", len(inner), dataLength)
	}
	ir := tar.NewReader(bytes.NewReader(inner))
	wantNames := []string{"logo_alfen.png", VideoResourceFileName, ManifestFileName}
	for i, name := range wantNames {
		h, err := ir.Next()
		if err != nil || h.Name != name || h.Mode != 0o644 || h.Uid != 0 || h.Typeflag != tar.TypeReg {
			t.Fatalf("inner entry %d: %+v %v", i, h, err)
		}
		data, _ := io.ReadAll(ir)
		switch i {
		case 0:
			if !bytes.Equal(data, png) || h.ModTime.Unix() != st.ModTime().Unix() {
				t.Fatal("image entry differs from the source file")
			}
		case 1:
			if string(data) != "{\"logo\":\"logo_alfen.png\",\"logoMargins\":[8,8,8,8]}\r\n" {
				t.Fatalf("videoresources.json %q", data)
			}
		case 2:
			if string(data) != NewAhpManifest(AppName, "logo_alfen", 1, now).JSON()+"\r\n" || !h.ModTime.Equal(now) {
				t.Fatalf("manifest.json %q", data)
			}
		}
	}
	if _, err := ir.Next(); err != io.EOF {
		t.Fatal("inner tar has extra entries")
	}
}

func TestCreateTvfDataRejects(t *testing.T) {
	dir := t.TempDir()
	notImage := filepath.Join(dir, "x.png")
	os.WriteFile(notImage, []byte("not an image"), 0o644)
	if _, err := CreateTvfData(notImage, AppName, 8, 1); err == nil {
		t.Fatal("non-image accepted")
	}
	// A GIF decodes but is not PNG.
	gif := filepath.Join(dir, "x.gif")
	os.WriteFile(gif, []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;"), 0o644)
	if _, err := CreateTvfData(gif, AppName, 8, 1); err == nil || err.Error() != "Only the png image format is supported on this CS platform" {
		t.Fatalf("gif: %v", err)
	}
	if _, err := validateTvfOutput(make([]byte, MaxTvfFileSize+1)); err == nil || err.Error() != "Compressed image size exceeds the limit off 500 kB" {
		t.Fatalf("size limit: %v", err)
	}
	if _, err := validateTvfOutput(make([]byte, MaxTvfFileSize)); err != nil {
		t.Fatal(err)
	}
}
