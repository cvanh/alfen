package backoffice

import (
	"encoding/json"
	"testing"

	"alfen/aceclient/internal/api"
)

func TestCSParseInt32(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"0", 0, true}, {"42", 42, true}, {" 7\t", 7, true}, {"+5", 5, true}, {"-5", -5, true},
		{"", 0, false}, {"1.0", 0, false}, {"0x10", 0, false}, {"1 2", 0, false}, {"+", 0, false},
		{"2147483647", 2147483647, true}, {"2147483648", 0, false}, {"1_000", 0, false},
	}
	for _, c := range cases {
		got, err := csParseInt32(c.in)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("csParseInt32(%q) = %d, %v; want %d ok=%v", c.in, got, err, c.want, c.ok)
		}
	}
}

func TestCSParseBool(t *testing.T) {
	cases := []struct {
		in       string
		want, ok bool
	}{
		{"True", true, true}, {"false", false, true}, {"TRUE", true, true}, {" True ", true, true},
		{"True\x00", true, true}, {"1", false, false}, {"yes", false, false}, {"", false, false},
	}
	for _, c := range cases {
		got, err := csParseBool(c.in)
		if (err == nil) != c.ok || got != c.want {
			t.Errorf("csParseBool(%q) = %v, %v; want %v ok=%v", c.in, got, err, c.want, c.ok)
		}
	}
}

// TestConvertLike pins ICUBackOffice.SetProperty's per-type conversions.
func TestConvertLike(t *testing.T) {
	cases := []struct {
		name     string
		current  any
		value    any
		want     any
		accepted bool
	}{
		{"string trims", "", "  ws://x  ", "ws://x", true},
		{"string from bool", "", true, "True", true},
		{"string from nil", "x", nil, "", true},
		{"string from number", "", json.Number("12"), "12", true},
		{"int from string", 0, "15", 15, true},
		{"int from padded string", 0, " 15 ", 15, true},
		{"int from empty fails", 7, "", 7, false},
		{"int from decimal string fails", 7, "1.5", 7, false},
		{"int from number", 0, json.Number("3"), 3, true},
		{"int from fractional number rounds half even", 0, json.Number("2.5"), 2, true},
		{"int from bool", 0, true, 1, true},
		{"int from nil", 9, nil, 0, true},
		{"bool from 0", true, "0", false, true},
		{"bool from 1", false, "1", true, true},
		{"bool from True", false, "True", true, true},
		{"bool from false", true, "false", false, true},
		{"bool from json true", false, true, true, true},
		{"bool from number 2", false, json.Number("2"), true, true},
		{"bool from garbage fails", true, "yes", true, false},
		{"bool from nil fails", true, nil, true, false},
	}
	for _, c := range cases {
		got, ok := convertLike(c.current, c.value)
		if ok != c.accepted || got != c.want {
			t.Errorf("%s: convertLike(%#v, %#v) = %#v, %v; want %#v, %v", c.name, c.current, c.value, got, ok, c.want, c.accepted)
		}
	}
}

// TestCoerceValue pins ICUProperty.SetValue + StoreProperties' ToString.
func TestCoerceValue(t *testing.T) {
	cases := []struct {
		dt         api.SDT
		text, prev string
		want       string
	}{
		{api.SDTVisibleString, " a ", "", " a "},
		{api.SDTUnsigned8, "False", "", "0"},
		{api.SDTUnsigned8, "true", "", "1"},
		{api.SDTUnsigned8, " 10 ", "", "10"},
		{api.SDTUnsigned8, "256", "", "0"},
		{api.SDTUnsigned8, "+1", "", "0"},
		{api.SDTInteger8, "-5", "", "-5"},
		{api.SDTInteger8, "3", "", "3"},
		{api.SDTInteger8, "", "", "0"},
		{api.SDTUnsigned16, "65535", "", "65535"},
		{api.SDTInteger32, "+12", "", "12"},
		{api.SDTBoolean, "False", "", "False"},
		{api.SDTBoolean, "true", "", "True"},
		{api.SDTBoolean, "0", "", "0"},
		{api.SDTReal32, "1,5", "", "1.5"},
		{api.SDTReal64, "x", "", "0"},
		{api.SDTByteArray, "1,ff,zz", "", "01,FF,00"},
		{api.SDTArray16, "1,abcd", "", "0001,ABCD"},
		{api.SDTUnicodeString, "new", "old", "old"},
		{api.SDTLowLimit, "x", "", ""},
	}
	for _, c := range cases {
		if got := coerceValue(c.dt, c.text, c.prev); got != c.want {
			t.Errorf("coerceValue(%d, %q, %q) = %q; want %q", c.dt, c.text, c.prev, got, c.want)
		}
	}
}

func TestVersion(t *testing.T) {
	cases := []struct {
		raw   string
		https bool
		want  Version
	}{
		{"6.6.2-4227", true, V(6, 6, 2)},
		{"4.12.0-3700", false, V(4, 12, 0)},
		{"4.X.0-1", true, V(4, 99, 0)},
		{"nodash", true, V(4, 10, 0)},
		{"nodash", false, V(4, 9, 0)},
		{"bad.version.x-1", true, V(0, 0)},
		{"7-1", true, V(0, 0)},
	}
	for _, c := range cases {
		if got := FirmwareVersion(c.raw, c.https); got != c.want {
			t.Errorf("FirmwareVersion(%q, %v) = %v; want %v", c.raw, c.https, got, c.want)
		}
	}
	if !V(4, 12, 0).AtLeast(V(4, 12, 0)) || V(4, 11, 9).AtLeast(V(4, 12, 0)) || !V(5, 0).AtLeast(V(4, 12, 0)) {
		t.Error("AtLeast ordering wrong")
	}
	// System.Version: an unspecified build (-1) sorts before .0
	if V(4, 12).AtLeast(V(4, 12, 0)) {
		t.Error("4.12 must be < 4.12.0 like System.Version")
	}
	if s := V(4, 12, 0).String(); s != "4.12.0" {
		t.Errorf("String() = %q", s)
	}
}
