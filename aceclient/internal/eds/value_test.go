package eds

import (
	"math"
	"testing"

	"alfen/aceclient/internal/api"
)

func TestParseValue(t *testing.T) {
	tests := []struct {
		sdt  api.SDT
		in   string
		kind ValueKind
		str  string // Value.ToString()
		wire string // StoreProperties text
	}{
		{api.SDTVisibleString, " a,b ", ValueString, " a,b ", " a,b "},
		{api.SDTUnsigned8, "12", ValueUint, "12", "12"},
		{api.SDTUnsigned8, " 12 ", ValueUint, "12", "12"},
		{api.SDTUnsigned8, "+1", ValueUint, "0", "0"}, // no sign allowed
		{api.SDTUnsigned8, "-1", ValueUint, "0", "0"},
		{api.SDTUnsigned8, "256", ValueUint, "0", "0"},
		{api.SDTUnsigned8, "TRUE", ValueUint, "1", "1"},
		{api.SDTUnsigned8, "false", ValueUint, "0", "0"},
		{api.SDTUnsigned8, " true", ValueUint, "0", "0"}, // true/false are not trimmed
		{api.SDTUnsigned8, "1.0", ValueUint, "0", "0"},
		{api.SDTInteger8, "-128", ValueInt, "-128", "-128"},
		{api.SDTInteger8, "+5", ValueInt, "5", "5"},
		{api.SDTInteger8, "128", ValueInt, "0", "0"},
		{api.SDTInteger8, "True", ValueInt, "1", "1"},
		{api.SDTUnsigned16, "65535", ValueUint, "65535", "65535"},
		{api.SDTUnsigned16, "65536", ValueUint, "0", "0"},
		{api.SDTUnsigned32, "4294967295", ValueUint, "4294967295", "4294967295"},
		{api.SDTUnsigned64, "18446744073709551615", ValueUint, "18446744073709551615", "18446744073709551615"},
		{api.SDTUnsigned64, "1.84467440737096E+19", ValueUint, "0", "0"},
		{api.SDTInteger16, "-32768", ValueInt, "-32768", "-32768"},
		{api.SDTInteger32, " -7 ", ValueInt, "-7", "-7"},
		{api.SDTInteger32, "- 7", ValueInt, "0", "0"},
		{api.SDTInteger64, "-9223372036854775808", ValueInt, "-9223372036854775808", "-9223372036854775808"},
		{api.SDTReal32, "12,5", ValueFloat32, "12.5", "12.5"},
		{api.SDTReal32, "0.1", ValueFloat32, "0.1", "0.1"},
		{api.SDTReal32, ".5", ValueFloat32, "0.5", "0.5"},
		{api.SDTReal32, "123456789", ValueFloat32, "1.234568E+08", "1.234568E+08"},
		{api.SDTReal32, "3.4028235E+38", ValueFloat32, "3.402823E+38", "3.402823E+38"},
		{api.SDTReal32, "1e39", ValueFloat32, "0", "0"}, // float overflow fails
		{api.SDTReal32, "NaN", ValueFloat32, "NaN", "NaN"},
		{api.SDTReal32, "1/3", ValueFloat32, "0", "0"},
		{api.SDTReal64, "0.1", ValueFloat64, "0.1", "0.1"},
		{api.SDTReal64, "1.8446744073709552E+19", ValueFloat64, "1.84467440737096E+19", "1.84467440737096E+19"},
		{api.SDTReal64, "1e-5", ValueFloat64, "1E-05", "1E-05"},
		{api.SDTReal64, "100", ValueFloat64, "100", "100"},
		{api.SDTReal64, "1e309", ValueFloat64, "0", "0"},
		{api.SDTReal64, "-Infinity", ValueFloat64, "-Infinity", "-Infinity"},
		{api.SDTBoolean, "true", ValueBool, "True", "True"},
		{api.SDTBoolean, " FALSE ", ValueBool, "False", "False"},
		{api.SDTBoolean, "1", ValueUint, "0", "0"}, // bool.TryParse fails -> (byte)0
		{api.SDTByteArray, "1,a,FF", ValueBytes, "System.Byte[]", "01,0A,FF"},
		{api.SDTByteArray, "", ValueBytes, "System.Byte[]", "00"},
		{api.SDTByteArray, "zz, 10,100", ValueBytes, "System.Byte[]", "00,10,00"},
		{api.SDTArray16, "1,ffff,10000", ValueWords, "System.UInt16[]", "0001,FFFF,0000"},
		{api.SDTLowLimit, "whatever", ValueString, "", ""},
		{api.SDTOctetString, "x", ValueNull, "", ""},
		{api.SDTUnicodeString, "x", ValueNull, "", ""},
		{api.SDTDomain, "x", ValueNull, "", ""},
		{api.SDTInteger24, "5", ValueNull, "", ""},
		{api.SDTTimeOfDay, "5", ValueNull, "", ""},
	}
	for _, tc := range tests {
		v := ParseValue(tc.sdt, tc.in)
		if v.Kind != tc.kind || v.String() != tc.str || v.Wire() != tc.wire {
			t.Errorf("ParseValue(%d, %q) = kind %d %q wire %q; want kind %d %q wire %q",
				tc.sdt, tc.in, v.Kind, v.String(), v.Wire(), tc.kind, tc.str, tc.wire)
		}
		if ValueSupported(tc.sdt) == (tc.kind == ValueNull) {
			t.Errorf("ValueSupported(%d) inconsistent", tc.sdt)
		}
	}
}

func TestReal32IsFloat32Exact(t *testing.T) {
	v := ParseValue(api.SDTReal32, "0.1")
	if v.Float != float64(float32(0.1)) {
		t.Fatalf("REAL32 must hold the float32 value, got %v", v.Float)
	}
}

func TestEqualAndChanged(t *testing.T) {
	u5 := ParseValue(api.SDTUnsigned8, "5")
	tests := []struct {
		a, b  Value
		equal bool
	}{
		{u5, ParseValue(api.SDTUnsigned8, " 5"), true},
		{u5, ParseValue(api.SDTInteger8, "5"), false}, // byte vs sbyte
		{ParseValue(api.SDTBoolean, "true"), ParseValue(api.SDTBoolean, "True"), true},
		{ParseValue(api.SDTBoolean, "x"), ParseValue(api.SDTBoolean, "false"), false}, // (byte)0 vs false
		{ParseValue(api.SDTByteArray, "01"), ParseValue(api.SDTByteArray, "01"), false},
		{ParseValue(api.SDTReal32, "NaN"), ParseValue(api.SDTReal32, "NaN"), true},
		{ParseValue(api.SDTVisibleString, "a"), ParseValue(api.SDTVisibleString, "A"), false},
		{Value{}, Value{}, true},
	}
	for i, tc := range tests {
		if got := tc.a.Equal(tc.b); got != tc.equal {
			t.Errorf("%d: Equal = %v, want %v", i, got, tc.equal)
		}
	}
	if Changed(u5, ParseValue(api.SDTUnsigned8, "5")) || !Changed(u5, ParseValue(api.SDTUnsigned8, "6")) || Changed(u5, Value{}) {
		t.Error("Changed")
	}
}

func TestConversions(t *testing.T) {
	i32 := []struct {
		v    Value
		want int32
		ok   bool
	}{
		{ParseValue(api.SDTReal32, "0.5"), 0, true},
		{ParseValue(api.SDTReal32, "1.5"), 2, true},
		{ParseValue(api.SDTReal64, "2.5"), 2, true},
		{ParseValue(api.SDTReal64, "-1.5"), -2, true},
		{ParseValue(api.SDTUnsigned32, "4294967295"), 0, false},
		{ParseValue(api.SDTBoolean, "true"), 1, true},
		{ParseValue(api.SDTVisibleString, " 42 "), 42, true},
		{ParseValue(api.SDTVisibleString, "x"), 0, false},
		{ParseValue(api.SDTByteArray, "1"), 0, false},
	}
	for i, tc := range i32 {
		got, err := tc.v.ToInt32()
		if got != tc.want || (err == nil) != tc.ok {
			t.Errorf("%d: ToInt32 = %d,%v", i, got, err)
		}
	}
	if f, err := ParseValue(api.SDTUnsigned64, "18446744073709551615").ToFloat64(); err != nil || f != math.MaxUint64 {
		t.Errorf("ToFloat64 = %v,%v", f, err)
	}
}

func TestNetFormatting(t *testing.T) {
	g := []struct {
		v    float64
		prec int
		want string
	}{
		{12.5, 15, "12.5"},
		{1e15, 15, "1E+15"},
		{123456789012345, 15, "123456789012345"},
		{1234567890123456, 15, "1.23456789012346E+15"},
		{0.0001, 15, "0.0001"},
		{0.00001, 15, "1E-05"},
		{0.1 + 0.2, 15, "0.3"},
		{float64(float32(1.0 / 3)), 7, "0.3333333"},
		{math.Copysign(0, -1), 15, "0"},
		{math.Inf(1), 15, "Infinity"},
	}
	for _, tc := range g {
		if got := netFormatG(tc.v, tc.prec); got != tc.want {
			t.Errorf("netFormatG(%v,%d) = %q, want %q", tc.v, tc.prec, got, tc.want)
		}
	}
	f := []struct {
		v    float64
		dec  int
		want string
	}{
		{0.05, 1, "0.1"},
		{0.15, 1, "0.2"}, // .NET Framework: 15 digits first, then half away from zero
		{1.005, 2, "1.01"},
		{2.5, 0, "3"},
		{-0.04, 1, "0.0"},
		{-1.25, 1, "-1.3"},
		{12.5, 3, "12.500"},
		{999.95, 1, "1000.0"},
		{0.0049, 2, "0.00"},
		{1e20, 2, "100000000000000000000.00"},
		{123.456, 0, "123"},
		{0, 3, "0.000"},
		{-12345.678, 1, "-12345.7"},
	}
	for _, tc := range f {
		if got := netFormatFixed(tc.v, tc.dec); got != tc.want {
			t.Errorf("netFormatFixed(%v,%d) = %q, want %q", tc.v, tc.dec, got, tc.want)
		}
	}
	r := []struct {
		v    float64
		d    int
		want float64
	}{
		{2.5, 0, 2}, {3.5, 0, 4}, {-2.5, 0, -2}, {1.005, 2, 1}, {12.3456, 3, 12.346}, {1e17 + 0.5, 0, 1e17 + 0.5},
	}
	for _, tc := range r {
		if got := netMathRound(tc.v, tc.d); got != tc.want {
			t.Errorf("netMathRound(%v,%d) = %v, want %v", tc.v, tc.d, got, tc.want)
		}
	}
}

func TestNetParseNumberStyles(t *testing.T) {
	tests := []struct {
		s     string
		style numStyle
		want  string
		ok    bool
	}{
		{" 5 ", styleUnsigned, "5", true},
		{"+5", styleUnsigned, "", false},
		{" -5", styleInteger, "-5", true},
		{"- 5", styleInteger, "", false},
		{"5-", styleNumber, "-5", true},
		{"5 -", styleNumber, "-5", true},
		{"-5-", styleNumber, "", false},
		{"5.", styleNumber, "5", true},
		{".5", styleNumber, "0.5", true},
		{".", styleNumber, "", false},
		{"1e3", styleNumber, "", false},
		{"1e3", styleFloat, "1e3", true},
		{"1e", styleFloat, "", false},
		{"5\x00\x00", styleInteger, "5", true},
	}
	for _, tc := range tests {
		got, ok := netParseNumber(tc.s, tc.style)
		if got != tc.want || ok != tc.ok {
			t.Errorf("netParseNumber(%q) = %q,%v want %q,%v", tc.s, got, ok, tc.want, tc.ok)
		}
	}
	if v, ok := netParseHexInt32(" 7fffffff "); !ok || v != math.MaxInt32 {
		t.Errorf("hex = %d,%v", v, ok)
	}
	if _, ok := netParseHexInt32("0x10"); ok {
		t.Error("HexNumber must reject a 0x prefix")
	}
	if _, ok := netParseHexInt32("100000000"); ok {
		t.Error("hex overflow must fail")
	}
}
