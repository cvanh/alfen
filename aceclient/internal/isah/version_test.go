package isah

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in      string
		want    Version
		wantStr string
		errSub  string
	}{
		{"3.4.0", Version{3, 4, 0, -1}, "3.4.0", ""},
		{"3.4", Version{3, 4, -1, -1}, "3.4", ""},
		{"1.2.3.4", Version{1, 2, 3, 4}, "1.2.3.4", ""},
		{" 6 . 4 .0 ", Version{6, 4, 0, -1}, "6.4.0", ""}, // Int32.TryParse allows surrounding white space
		{"+3.4", Version{3, 4, -1, -1}, "3.4", ""},
		{"4.99.0", Version{4, 99, 0, -1}, "4.99.0", ""},
		{"3", Version{}, "", "too short or too long"},
		{"1.2.3.4.5", Version{}, "", "too short or too long"},
		{"", Version{}, "", "too short or too long"},
		{"a.b", Version{}, "", "not in a correct format"},
		{"1..2", Version{}, "", "not in a correct format"},
		{"-1.0", Version{}, "", "greater than or equal to zero"},
		{"99999999999.0", Version{}, "", "too large or too small"},
	}
	for _, c := range cases {
		got, err := ParseVersion(c.in)
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("ParseVersion(%q) error = %v, want containing %q", c.in, err, c.errSub)
			}
			continue
		}
		if err != nil || got != c.want || got.String() != c.wantStr {
			t.Errorf("ParseVersion(%q) = %+v (%s), %v; want %+v (%s)", c.in, got, got, err, c.want, c.wantStr)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		a, b Version
		want int
	}{
		{NewVersion(3, 4), NewVersion(3, 4, 0), -1}, // undefined build sorts first
		{NewVersion(3, 4, 0), NewVersion(3, 4, 0), 0},
		{NewVersion(3, 4, 0), NewVersion(3, 4, 1), -1},
		{NewVersion(3, 10, 0), NewVersion(3, 9, 0), 1},
		{NewVersion(4, 0), NewVersion(3, 99, 99, 99), 1},
		{NewVersion(1, 2, 3, 4), NewVersion(1, 2, 3), 1},
		{ZeroVersion, NewVersion(0, 0, 0), -1},
	}
	for _, c := range cases {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", c.a, c.b, got, c.want)
		}
		if got := c.a.Less(c.b); got != (c.want < 0) {
			t.Errorf("%s.Less(%s) = %v", c.a, c.b, got)
		}
	}
	if s := ZeroVersion.String(); s != "0.0" {
		t.Errorf("ZeroVersion = %q", s)
	}
}

func TestNewVersionPanics(t *testing.T) {
	for _, f := range []func(){
		func() { NewVersion(1, 2, 3, 4, 5) },
		func() { NewVersion(-1, 0) },
		func() { NewVersion(1, 0, -1) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("expected panic")
				}
			}()
			f()
		}()
	}
}

func TestFirmwareVersionNumber(t *testing.T) {
	cases := []struct {
		sw    string
		https bool
		want  string
	}{
		{"6.4.0-4166", true, "6.4.0"},
		{"4.X.0-1", false, "4.99.0"},     // "X" -> "99"
		{"1.2.3.4-x-y", true, "1.2.3.4"}, // only the first part counts
		{"garbage-1", true, "0.0"},       // parse failure -> new Version()
		{"5-1", true, "0.0"},             // a single component is a parse failure
		{"", true, "4.10.0"},             // no '-' -> HTTPS default
		{"", false, "4.9.0"},             // no '-' -> HTTP default
		{"5.1.0", true, "4.10.0"},        // no '-' even if it looks like a version
	}
	for _, c := range cases {
		if got := FirmwareVersionNumber(c.sw, c.https).String(); got != c.want {
			t.Errorf("FirmwareVersionNumber(%q, %v) = %s, want %s", c.sw, c.https, got, c.want)
		}
	}
}

// TestPropertyIDsAgainstEDS checks the property IDs against the shipped
// EDS.xml, for the IDs that it defines (0x21A0..0x21A2 are not in it).
func TestPropertyIDsAgainstEDS(t *testing.T) {
	data, err := os.ReadFile("../../../firmware/msi_work/files3/EDS.xml")
	if err != nil {
		t.Skipf("EDS.xml fixture not available: %v", err)
	}
	for _, c := range []struct {
		id    uint16
		hex   string
		param string
	}{
		{PropSoftwareVersion, "100A", "Manufacturer software version"},
		{PropObjectID, "2051", "sysChargePointSerialNumber"},
	} {
		re := regexp.MustCompile(`<Object Id="` + c.hex + `" ParameterName="([^"]*)"`)
		m := re.FindSubmatch(data)
		if m == nil {
			t.Errorf("EDS.xml has no Object %s", c.hex)
			continue
		}
		if string(m[1]) != c.param {
			t.Errorf("EDS Object %s ParameterName = %q, want %q", c.hex, m[1], c.param)
		}
		if got := strings.ToUpper(strings.TrimLeft(hex16(c.id), "0")); got != c.hex {
			t.Errorf("constant %d is %s, want %s", c.id, got, c.hex)
		}
	}
}

func hex16(v uint16) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>12], digits[v>>8&0xF], digits[v>>4&0xF], digits[v&0xF]})
}
