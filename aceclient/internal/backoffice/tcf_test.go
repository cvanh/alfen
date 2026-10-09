package backoffice

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"alfen/aceclient/internal/alfencrc"
)

// TestTCFLayout documents (it does not implement) the layout of the AHP
// backoffice presets in firmware/BackofficePresets/*.tcf. The C# treats them
// as opaque blobs selected by extension; this pins what the package doc says
// about them: the ACEFWUCreator.TvfHeader layout with magic 0x84FC8EB7.
func TestTCFLayout(t *testing.T) {
	files, _ := filepath.Glob("../../../firmware/BackofficePresets/*.tcf")
	if len(files) == 0 {
		t.Skip("no .tcf fixtures")
	}
	named := 0
	for _, f := range files {
		d, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		le := binary.LittleEndian
		if len(d) < 0x40 {
			t.Fatalf("%s: too short", f)
		}
		version, hdrLen := le.Uint16(d[0:]), int(le.Uint16(d[2:]))
		magic, dataLen := le.Uint32(d[4:]), int(le.Uint32(d[8:]))
		mvLen := int(d[28])
		descLen := int(d[29+mvLen])
		desc := string(d[30+mvLen : 30+mvLen+descLen])
		if version != 1 || magic != 0x84FC8EB7 {
			t.Errorf("%s: version %d magic %#x", f, version, magic)
		}
		if hdrLen != 1350+mvLen+descLen || len(d) != hdrLen+dataLen {
			t.Errorf("%s: header %d data %d size %d", f, hdrLen, dataLen, len(d))
		}
		if got, want := le.Uint32(d[hdrLen-4:]), alfencrc.ComputeChecksum(d[:hdrLen-4], 0); got != want {
			t.Errorf("%s: header CRC %#x, computed %#x", f, got, want)
		}
		sigOff := 30 + mvLen + descLen + 32
		if sl := int(le.Uint16(d[sigOff:])); sl == 0 || sl > 256 {
			t.Errorf("%s: signature length %d", f, sl)
		}
		if strings.TrimSuffix(filepath.Base(f), ".tcf") == desc {
			named++
		}
	}
	t.Logf("%d .tcf presets, %d with description == file name", len(files), named)
	if named == 0 {
		t.Error("expected the description to carry the preset name")
	}
}
