package fwucreator

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"
)

const decompiledDir = "../../../firmware/decompiled"

// TestObjectsMatchDecompiledCSharp re-parses ICUObjects.cs at test time and
// compares every generated table byte for byte.
func TestObjectsMatchDecompiledCSharp(t *testing.T) {
	src, err := os.ReadFile(decompiledDir + "/ACEFWUCreator/ICUFWUCreator/ICUObjects.cs")
	if err != nil {
		t.Skipf("decompiled source missing: %v", err)
	}
	gen := map[string][]byte{
		"logo_accepted_L1":      LogoAcceptedL1,
		"logo_charging_L1":      LogoChargingL1,
		"logo_error_L1":         LogoErrorL1,
		"logo_communicating_L1": LogoCommunicatingL1,
		"font_robotica_28":      FontRobotica28,
		"font_robotica_29":      FontRobotica29,
		"font_robotica_30":      FontRobotica30,
	}
	re := regexp.MustCompile(`(?s)public static byte\[\] (\w+) = new byte\[(\d+)\]\s*\{(.*?)\};`)
	num := regexp.MustCompile(`\d+`)
	ms := re.FindAllSubmatch(src, -1)
	if len(ms) != len(gen) {
		t.Fatalf("C# has %d tables, generated %d", len(ms), len(gen))
	}
	for _, m := range ms {
		name := string(m[1])
		n, _ := strconv.Atoi(string(m[2]))
		var want []byte
		for _, v := range num.FindAll(m[3], -1) {
			b, _ := strconv.Atoi(string(v))
			want = append(want, byte(b))
		}
		if len(want) != n {
			t.Fatalf("%s: parsed %d of %d", name, len(want), n)
		}
		if !bytes.Equal(gen[name], want) {
			t.Errorf("%s differs from ICUObjects.cs", name)
		}
	}
}

// TestBuiltinObjectGeometry checks the 1-bit icons are exactly stride*height
// bytes, as CreateFWUData's arguments imply.
func TestBuiltinObjectGeometry(t *testing.T) {
	for _, o := range builtinObjects[:4] {
		if len(o.Data) != o.Stride*o.Height || o.Stride != (o.Width+7)/8 {
			t.Errorf("%v: %d bytes, stride %d x height %d (width %d)", o.Type, len(o.Data), o.Stride, o.Height, o.Width)
		}
	}
}

func TestEnumValues(t *testing.T) {
	tests := []struct {
		got  int
		want int
		name string
	}{
		{int(ObjectLanguage), 7, ObjectLanguage.String()},
		{int(ObjectRoboticaRegular30), 8, ObjectRoboticaRegular30.String()},
		{int(ImageFormatL1), 1, ImageFormatL1.String()},
		{int(ImageFormatL2), 17, ImageFormatL2.String()},
		{int(ImageFormatL4), 2, ImageFormatL4.String()},
		{int(ImagePaletted8), 16, ImagePaletted8.String()},
		{int(DispEmpty), -1, DispEmpty.String()},
		{int(DispChargePointBooting), 0, DispChargePointBooting.String()},
		{int(DispIndexMax), 50, DispIndexMax.String()},
		{int(UIErrorTilt), 407, UIErrorTilt.String()},
		{int(StateWaitForDisconnectPp), 55, StateWaitForDisconnectPp.String()},
		{int(FwRolledBack), -4, FwRolledBack.String()},
	}
	names := []string{"OBJECT_LANGUAGE", "OBJECT_ROBOTICA_REGULAR_30", "FORMAT_L1", "FORMAT_L2", "FORMAT_L4", "PALETTED8",
		"DISP_EMPTY", "DISP_CHARGE_POINT_BOOTING", "DISP_INDEX_MAX", "UI_ERROR_TILT", "STATE_WAIT_FOR_DISCONNECT_PP", "ROLLED_BACK"}
	for i, tc := range tests {
		if tc.got != tc.want || tc.name != names[i] {
			t.Errorf("%s = %d, want %s = %d", tc.name, tc.got, names[i], tc.want)
		}
	}
	if s := DisplayString(99).String(); s != "99" {
		t.Errorf("undefined enum String = %q", s)
	}
}

func TestParseDisplayString(t *testing.T) {
	tests := []struct {
		in   string
		want DisplayString
		err  string
	}{
		{"DISP_CARD_ACCEPTED", DispCardAccepted, ""},
		{"  DISP_GOODBYE\t", DispGoodbye, ""},
		{"5", 5, ""},
		{"-1", DispEmpty, ""},
		{"+7", 7, ""},
		{"300", 300, ""},
		{"disp_card_accepted", 0, "Requested value 'disp_card_accepted' was not found."},
		{"", 0, "Must specify valid information for parsing in the string."},
		{"9999999999", 0, "Value was either too large or too small for an Int32."},
		{"5x", 0, "Input string was not in a correct format."},
	}
	for _, tc := range tests {
		got, err := ParseDisplayString(tc.in)
		if tc.err != "" {
			if err == nil || err.Error() != tc.err {
				t.Errorf("%q: err %v, want %q", tc.in, err, tc.err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("%q: %v %v, want %v", tc.in, got, err, tc.want)
		}
	}
}

func TestParseUInt16(t *testing.T) {
	tests := []struct {
		in   string
		want uint16
		ok   bool
	}{
		{" 15", 15, true}, {"\t\t29", 29, true}, {"65535", 65535, true}, {"+7", 7, true},
		{"-0", 0, true}, {"08 ", 8, true}, {"65536", 0, false}, {"-1", 0, false},
		{"", 0, false}, {"1 2", 0, false}, {"0x10", 0, false},
	}
	for _, tc := range tests {
		got, err := parseUInt16(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("%q: %d %v", tc.in, got, err)
		}
	}
}

func TestReadAllLines(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		raw  []byte
		want []string
	}{
		{[]byte("a\r\nb\r\n"), []string{"a", "b"}},
		{[]byte("a\nb"), []string{"a", "b"}},
		{[]byte("a\rb\r\n\nc"), []string{"a", "b", "", "c"}},
		{[]byte("\xEF\xBB\xBFx"), []string{"x"}},
		{[]byte{0xFF, 0xFE, 'h', 0, 'i', 0, '\n', 0, 'j', 0}, []string{"hi", "j"}},
		{[]byte(""), nil},
	}
	for i, tc := range tests {
		p := dir + "/f" + strconv.Itoa(i)
		os.WriteFile(p, tc.raw, 0o644)
		got, err := readAllLines(p)
		if err != nil || len(got) != len(tc.want) {
			t.Errorf("%d: %q %v", i, got, err)
			continue
		}
		for j := range got {
			if got[j] != tc.want[j] {
				t.Errorf("%d: %q, want %q", i, got, tc.want)
			}
		}
	}
}
