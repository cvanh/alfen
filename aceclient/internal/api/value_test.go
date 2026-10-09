package api

import (
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		t    SDT
		in   string
		want any
		ok   bool
	}{
		{SDTVisibleString, " as is ", " as is ", true},
		{SDTUnsigned8, "TRUE", uint8(1), true},
		{SDTUnsigned8, "False", uint8(0), true},
		{SDTUnsigned8, " 255 ", uint8(255), true},
		{SDTUnsigned8, "256", uint8(0), true},
		{SDTUnsigned8, "+1", uint8(0), true}, // no sign allowed
		{SDTUnsigned8, "1.0", uint8(0), true},
		{SDTInteger8, "-128", int8(-128), true},
		{SDTInteger8, "+5", int8(5), true},
		{SDTInteger8, "true", int8(1), true},
		{SDTInteger8, "128", int8(0), true},
		{SDTUnsigned16, "65535", uint16(65535), true},
		{SDTUnsigned32, "\t4294967295\n", uint32(4294967295), true},
		{SDTUnsigned64, "18446744073709551615", uint64(18446744073709551615), true},
		{SDTInteger16, "-32768", int16(-32768), true},
		{SDTInteger32, "2147483648", int32(0), true},
		{SDTInteger64, "-9223372036854775808", int64(math.MinInt64), true},
		{SDTReal32, "1,5", float32(1.5), true},
		{SDTReal32, " .5 ", float32(0.5), true},
		{SDTReal32, "5.", float32(5), true},
		{SDTReal32, "1e39", float32(0), true}, // overflow fails on .NET Framework
		{SDTReal32, "Infinity", float32(math.Inf(1)), true},
		{SDTReal32, "inf", float32(0), true},
		{SDTReal32, "0x1p-2", float32(0), true},
		{SDTReal64, "-1.25E+2", float64(-125), true},
		{SDTReal64, "1e309", float64(0), true},
		{SDTReal64, "1,234.5", float64(0), true}, // no thousands in NumberStyles.Float
		{SDTBoolean, "true", true, true},
		{SDTBoolean, " FALSE\x00", false, true},
		{SDTBoolean, "1", uint8(0), true},
		{SDTByteArray, "", []byte{0}, true},
		{SDTByteArray, "0a,FF, 7 ,1ff,zz", []byte{0x0A, 0xFF, 0x07, 0, 0}, true},
		{SDTArray16, "1234,zz,FFFF", []uint16{0x1234, 0, 0xFFFF}, true},
		{SDTLowLimit, "whatever", "", true},
		{SDTDomain, "x", nil, false},
		{SDTUnicodeString, "x", nil, false},
		{SDTInteger24, "1", nil, false},
		{SDTOctetString, "1", nil, false},
	}
	for _, tc := range tests {
		got, ok := ParseValue(tc.t, tc.in)
		if ok != tc.ok || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParseValue(%d, %q) = %#v %v, want %#v %v", tc.t, tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestValueString(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{nil, ""},
		{"x", "x"},
		{true, "True"},
		{uint8(7), "7"},
		{int64(-3), "-3"},
		{float32(0.1), "0.1"},
		{float32(16777217), "1.677722E+07"},
		{float64(1.0 / 3), "0.333333333333333"},
		{float64(1e15), "1E+15"},
		{float64(1e-5), "1E-05"},
		{float64(0.0001), "0.0001"},
		{math.Copysign(0, -1), "0"},
		{math.NaN(), "NaN"},
		{math.Inf(-1), "-Infinity"},
		{[]byte{1}, "System.Byte[]"},
		{[]uint16{1}, "System.UInt16[]"},
	}
	for _, tc := range tests {
		if got := ValueString(tc.v); got != tc.want {
			t.Errorf("ValueString(%#v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestPropertyAccessors(t *testing.T) {
	p := func(t SDT, v string) Property { return Property{DataType: t, Value: v} }
	ints := []struct {
		p    Property
		mask int
		want int
	}{
		{p(SDTReal64, "2.5"), 0, 2}, // banker's rounding
		{p(SDTReal64, "3.5"), 0, 4},
		{p(SDTReal64, "-2.5"), 0, -2},
		{p(SDTReal64, "1e10"), 0, 0}, // overflow -> 0
		{p(SDTUnsigned32, "4294967295"), 0, 0},
		{p(SDTUnsigned8, "7"), 2, 2},
		{p(SDTBoolean, "True"), 0, 1},
		{p(SDTVisibleString, " 42 "), 0, 42},
		{p(SDTVisibleString, "abc"), 0, 0},
		{p(SDTByteArray, "01"), 0, 0},
		{p(SDTDomain, "5"), 0, 0},
	}
	for _, tc := range ints {
		if got := tc.p.Int(tc.mask); got != tc.want {
			t.Errorf("%+v.Int(%d) = %d, want %d", tc.p, tc.mask, got, tc.want)
		}
	}
	if got := p(SDTUnsigned32, "4294967295").UInt(0); got != 4294967295 {
		t.Errorf("UInt = %d", got)
	}
	if got := p(SDTUnsigned32, "255").UInt(0x0F); got != 0x0F {
		t.Errorf("UInt mask = %d", got)
	}
	if got := p(SDTReal64, "-1.5").UInt(0); got != 0 {
		t.Errorf("UInt negative = %d", got)
	}
	if got := p(SDTReal64, "-0.4").UInt(0); got != 0 {
		t.Errorf("UInt -0.4 = %d", got)
	}
	bools := []struct {
		p    Property
		def  bool
		want bool
	}{
		{p(SDTBoolean, "true"), false, true},
		{p(SDTUnsigned8, "2"), false, true},
		{p(SDTUnsigned8, "0"), true, false},
		{p(SDTVisibleString, "abc"), true, true},
		{p(SDTVisibleString, "false"), true, false},
		{p(SDTBoolean, "garbage"), true, false}, // value is byte 0
		{p(SDTDomain, ""), true, true},
	}
	for _, tc := range bools {
		if got := tc.p.Bool(tc.def); got != tc.want {
			t.Errorf("%+v.Bool(%v) = %v", tc.p, tc.def, got)
		}
	}
	if got := p(SDTReal64, "NaN").UInt64(7); got != 7 {
		t.Errorf("UInt64 NaN = %d", got)
	}
	if got := p(SDTReal64, "2.5").UInt64(0); got != 2 {
		t.Errorf("UInt64 2.5 = %d", got)
	}
	if got := p(SDTUnsigned64, "18446744073709551615").UInt64(0); got != math.MaxUint64 {
		t.Errorf("UInt64 max = %d", got)
	}
	if got := p(SDTVisibleString, "1,234.5").Double(); got != 1234.5 {
		t.Errorf("Double thousands = %v", got)
	}
	if got := p(SDTReal32, "0.1").Double(); got != float64(float32(0.1)) {
		t.Errorf("Double float32 widening = %v", got)
	}
	if got := p(SDTByteArray, "01").Double(); got != 0 {
		t.Errorf("Double array = %v", got)
	}
	if got := p(SDTReal32, "1,5").ToString(); got != "1.5" {
		t.Errorf("ToString = %q", got)
	}
	if got := p(SDTUnicodeString, "x").ToString(); got != "" {
		t.Errorf("ToString of a null value = %q", got)
	}
}

func TestDeviceValueString(t *testing.T) {
	tests := []struct {
		t    SDT
		v    string
		want string
	}{
		{SDTReal64, "2.0005", "2.001"}, // 15-digit rounding, half away from zero
		{SDTReal64, "-1.2345", "-1.235"},
		{SDTReal64, "-0.0001", "0.000"},
		{SDTReal64, "1234567.89051", "1234567.891"},
		{SDTReal64, "9.9996", "10.000"},
		{SDTReal64, "0.0006", "0.001"},
		{SDTReal64, "1e20", "100000000000000000000.000"},
		{SDTReal32, "0.1", "0.100"},
		{SDTReal32, "NaN", "NaN"},
		{SDTUnsigned8, "5", "5"},
		{SDTDomain, "5", ""},
	}
	for _, tc := range tests {
		if got := (Property{DataType: tc.t, Value: tc.v}).DeviceValueString(); got != tc.want {
			t.Errorf("DeviceValueString(%d, %q) = %q, want %q", tc.t, tc.v, got, tc.want)
		}
	}
	for in, want := range map[float64]string{1.5: "1.5", 0.05: "0.1", 0.25: "0.3", 1.0: "1.0", 0.0: "0.0"} {
		if got := formatNetFixed(in, 1); got != want {
			t.Errorf("F1(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestElementValue(t *testing.T) {
	tests := []struct {
		t    SDT
		v    string
		want string
		ok   bool
	}{
		{SDTBoolean, "true", "true", true},
		{SDTReal32, "0.1", "0.1", true},
		{SDTReal64, "0.333333333333333314829616256247", "0.33333333333333331", true},
		{SDTReal64, "-0", "-0", true},
		{SDTReal64, "1e300", "1E+300", true},
		{SDTReal64, "Infinity", "INF", true},
		{SDTByteArray, "1,2", "01,02", true},
		{SDTArray16, "1", "0001", true},
		{SDTUnsigned16, "7", "7", true},
		{SDTVisibleString, "x y", "x y", true},
		{SDTDomain, "x", "", false},
	}
	for _, tc := range tests {
		got, ok := (Property{DataType: tc.t, Value: tc.v}).ElementValue()
		if got != tc.want || ok != tc.ok {
			t.Errorf("ElementValue(%d, %q) = %q %v, want %q", tc.t, tc.v, got, ok, tc.want)
		}
	}
}

// TestEDSDataTypes checks that every data type the shipped EDS.xml uses has
// a defined SetValue outcome, and pins the ones the C# cannot hold.
func TestEDSDataTypes(t *testing.T) {
	data, err := os.ReadFile("../../../firmware/msi_work/files3/EDS.xml")
	if err != nil {
		t.Skip("EDS.xml fixture not available:", err)
	}
	seen := map[SDT]int{}
	for _, m := range regexp.MustCompile(`DataType="0x([0-9A-Fa-f]+)"`).FindAllStringSubmatch(string(data), -1) {
		n, _ := strconv.ParseUint(m[1], 16, 16)
		seen[SDT(n)]++
	}
	if len(seen) == 0 {
		t.Fatal("no DataType attributes found")
	}
	for typ, count := range seen {
		_, ok := ParseValue(typ, "1")
		if typ == SDTDomain {
			if ok {
				t.Errorf("DOMAIN (%d params) must have no SetValue branch", count)
			}
			continue
		}
		if !ok {
			t.Errorf("EDS data type %d (%d params) has no SetValue branch", typ, count)
		}
	}
}
