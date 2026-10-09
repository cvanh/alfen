package presets

import (
	"math"
	"testing"

	"alfen/aceclient/internal/api"
)

func TestFloatFormattingMatchesDotNet(t *testing.T) {
	for _, v := range dotnetFloat32 {
		f := float64(math.Float32frombits(v.bits))
		if got := netFormatR(f, 7, 9, true); got != v.r {
			t.Errorf("float32 %08X R = %q, want %q", v.bits, got, v.r)
		}
		if got := netFormatG(f, 7, "Infinity", "-Infinity"); got != v.g {
			t.Errorf("float32 %08X G = %q, want %q", v.bits, got, v.g)
		}
	}
	for _, v := range dotnetFloat64 {
		f := math.Float64frombits(v.bits)
		if got := netFormatR(f, 15, 17, false); got != v.r {
			t.Errorf("float64 %016X R = %q, want %q", v.bits, got, v.r)
		}
		if got := netFormatG(f, 15, "Infinity", "-Infinity"); got != v.g {
			t.Errorf("float64 %016X G = %q, want %q", v.bits, got, v.g)
		}
	}
}

func TestXMLConvertFloatSpecials(t *testing.T) {
	tests := []struct {
		f    float64
		want string
	}{
		{math.Inf(1), "INF"},
		{math.Inf(-1), "-INF"},
		{math.NaN(), "NaN"},
		{math.Copysign(0, -1), "-0"},
		{0, "0"},
	}
	for _, tt := range tests {
		if got := xmlConvertFloat(tt.f, 7, 9, true); got != tt.want {
			t.Errorf("xmlConvertFloat(%v) = %q, want %q", tt.f, got, tt.want)
		}
	}
	// default ToString: -0 prints as "0" on .NET Framework
	if got := netFormatG(math.Copysign(0, -1), 7, "Infinity", "-Infinity"); got != "0" {
		t.Errorf("G(-0) = %q", got)
	}
}

func TestParseValue(t *testing.T) {
	tests := []struct {
		name    string
		dt      api.SDT
		text    string
		element string // ICUProperty.Element Value attribute
		store   string // StoreProperties value text
		kind    valueKind
	}{
		{"u8", api.SDTUnsigned8, "5", "5", "5", kindByte},
		{"u8 white", api.SDTUnsigned8, " \t7 \r\n", "7", "7", kindByte},
		{"u8 true", api.SDTUnsigned8, "TRUE", "1", "1", kindByte},
		{"u8 false", api.SDTUnsigned8, "False", "0", "0", kindByte},
		{"u8 overflow", api.SDTUnsigned8, "256", "0", "0", kindByte},
		{"u8 sign rejected", api.SDTUnsigned8, "+5", "0", "0", kindByte},
		{"u8 trailing NUL", api.SDTUnsigned8, "9\x00\x00", "9", "9", kindByte},
		{"i8", api.SDTInteger8, "-128", "-128", "-128", kindSByte},
		{"i8 true", api.SDTInteger8, "true", "1", "1", kindSByte},
		{"i8 overflow", api.SDTInteger8, "128", "0", "0", kindSByte},
		{"i8 space after sign", api.SDTInteger8, "- 5", "0", "0", kindSByte},
		{"u16", api.SDTUnsigned16, "65535", "65535", "65535", kindUInt16},
		{"u16 no bool", api.SDTUnsigned16, "true", "0", "0", kindUInt16},
		{"u32", api.SDTUnsigned32, "4294967295", "4294967295", "4294967295", kindUInt32},
		{"u64", api.SDTUnsigned64, "18446744073709551615", "18446744073709551615", "18446744073709551615", kindUInt64},
		{"u64 overflow", api.SDTUnsigned64, "18446744073709551616", "0", "0", kindUInt64},
		{"i16", api.SDTInteger16, "+300", "300", "300", kindInt16},
		{"i32 min", api.SDTInteger32, "-2147483648", "-2147483648", "-2147483648", kindInt32},
		{"i64 min", api.SDTInteger64, "-9223372036854775808", "-9223372036854775808", "-9223372036854775808", kindInt64},
		{"i32 decimal rejected", api.SDTInteger32, "1.0", "0", "0", kindInt32},
		{"real32", api.SDTReal32, "16.1", "16.1", "16.1", kindFloat32},
		{"real32 comma", api.SDTReal32, "16,5", "16.5", "16.5", kindFloat32},
		{"real32 exp", api.SDTReal32, "1e-5", "1E-05", "1E-05", kindFloat32},
		{"real32 R vs G", api.SDTReal32, "0.333333343", "0.333333343", "0.3333333", kindFloat32},
		{"real32 overflow", api.SDTReal32, "1e39", "0", "0", kindFloat32},
		{"real32 infinity symbol", api.SDTReal32, " Infinity ", "INF", "Infinity", kindFloat32},
		{"real32 nan", api.SDTReal32, "NaN", "NaN", "NaN", kindFloat32},
		{"real32 neg zero", api.SDTReal32, "-0", "0", "0", kindFloat32},
		{"real32 thousands", api.SDTReal32, "1,000.5", "0", "0", kindFloat32},
		{"real32 dangling exp", api.SDTReal32, "5e", "0", "0", kindFloat32},
		{"real32 leading dot", api.SDTReal32, ".5", "0.5", "0.5", kindFloat32},
		{"real64", api.SDTReal64, "0.1", "0.1", "0.1", kindFloat64},
		{"real64 overflow", api.SDTReal64, "1e999", "0", "0", kindFloat64},
		{"real64 G15", api.SDTReal64, "0.30000000000000004", "0.30000000000000004", "0.3", kindFloat64},
		{"bool", api.SDTBoolean, "True", "true", "True", kindBool},
		{"bool trimmed", api.SDTBoolean, " false\x00", "false", "False", kindBool},
		{"bool fallback byte", api.SDTBoolean, "1", "0", "0", kindByte},
		{"string", api.SDTVisibleString, " a,b ", " a,b ", " a,b ", kindString},
		{"bytes", api.SDTByteArray, "a, ff ,0", "0A,FF,00", "0A,FF,00", kindBytes},
		{"bytes bad", api.SDTByteArray, "0x1,100,zz", "00,00,00", "00,00,00", kindBytes},
		{"bytes empty", api.SDTByteArray, "", "00", "00", kindBytes},
		{"words", api.SDTArray16, "1234,a,FFFF", "1234,000A,FFFF", "1234,000A,FFFF", kindWords},
		{"words overflow", api.SDTArray16, "10000", "0000", "0000", kindWords},
		{"low limit", api.SDTLowLimit, "x", "", "", kindString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := parseValue(tt.dt, tt.text)
			if r.keep || r.v == nil {
				t.Fatalf("unexpected keep")
			}
			if r.v.kind != tt.kind {
				t.Errorf("kind = %v, want %v", r.v.kind, tt.kind)
			}
			if got := r.v.elementString(); got != tt.element {
				t.Errorf("element = %q, want %q", got, tt.element)
			}
			if got := r.v.storeString(); got != tt.store {
				t.Errorf("store = %q, want %q", got, tt.store)
			}
		})
	}
}

func TestParseValueUnhandledTypesKeepValue(t *testing.T) {
	for _, dt := range []api.SDT{api.SDTUnicodeString, api.SDTDomain, api.SDTOctetString, api.SDTInteger24, api.SDTUnsigned40, api.SDTTimeOfDay} {
		if r := parseValue(dt, "1"); !r.keep {
			t.Errorf("SDT %d: expected value to be left unchanged", dt)
		}
	}
	if r := parseValue(api.SDTLowLimit, "1"); !r.readOnly {
		t.Error("LOW_LIMIT must set ReadOnly")
	}
}

func TestValueEquals(t *testing.T) {
	f1 := parseValue(api.SDTReal32, "NaN").v
	f2 := parseValue(api.SDTReal32, "NaN").v
	if !f1.equals(f2) {
		t.Error("NaN.Equals(NaN) must be true")
	}
	b1 := parseValue(api.SDTByteArray, "01").v
	b2 := parseValue(api.SDTByteArray, "01").v
	if b1.equals(b2) || !b1.equals(b1) {
		t.Error("arrays must compare by reference")
	}
	byteZero := parseValue(api.SDTBoolean, "x").v // (byte)0
	boolFalse := parseValue(api.SDTBoolean, "false").v
	if byteZero.equals(boolFalse) {
		t.Error("(byte)0 must not equal false")
	}
	if !byteZero.equals(parseValue(api.SDTUnsigned8, "0").v) {
		t.Error("(byte)0 must equal (byte)0")
	}
	if parseValue(api.SDTUnsigned16, "1").v.equals(parseValue(api.SDTUnsigned32, "1").v) {
		t.Error("different CLR types must not be equal")
	}
	var null *propValue
	if null.equals(byteZero) || byteZero.equals(nil) {
		t.Error("null comparisons")
	}
}

func TestNetParseHex(t *testing.T) {
	tests := []struct {
		s    string
		max  uint64
		want uint64
		ok   bool
	}{
		{"2129", 0xFFFFFFFF, 0x2129, true},
		{" 0ff ", 0xFF, 0xFF, true},
		{"000000000001", 0xFF, 1, true},
		{"100", 0xFF, 0, false},
		{"FFFFFFFF", 0xFFFFFFFF, 0xFFFFFFFF, true},
		{"100000000", 0xFFFFFFFF, 0, false},
		{"0x10", 0xFF, 0, false},
		{"", 0xFF, 0, false},
		{"-1", 0xFF, 0, false},
	}
	for _, tt := range tests {
		got, ok := netParseHex(tt.s, tt.max)
		if got != tt.want || ok != tt.ok {
			t.Errorf("netParseHex(%q) = %d,%v want %d,%v", tt.s, got, ok, tt.want, tt.ok)
		}
	}
}
