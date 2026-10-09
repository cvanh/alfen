package eds

import (
	"testing"

	"alfen/aceclient/internal/api"
)

func TestNumberSpin(t *testing.T) {
	const f32 = 3.4028234663852886e38
	tests := []struct {
		sdt    api.SDT
		digits int
		maxLen uint64
		want   SpinConfig
	}{
		{api.SDTUnsigned8, 0, 0, SpinConfig{0, 255, 1, 0}},
		{api.SDTUnsigned8, 0, 2, SpinConfig{0, 2, 1, 0}}, // device len overrides Maximum
		{api.SDTUnsigned16, 0, 0, SpinConfig{0, 65535, 1, 0}},
		{api.SDTUnsigned32, 0, 0, SpinConfig{0, 4294967295, 1, 0}},
		{api.SDTInteger8, 0, 0, SpinConfig{-128, 127, 1, 0}},
		{api.SDTInteger16, -1, 0, SpinConfig{-32768, 32767, 1, 0}},
		{api.SDTInteger32, 0, 0, SpinConfig{-2147483648, 2147483647, 1, 0}},
		{api.SDTInteger64, 0, 0, SpinConfig{-9.223372036854776e18, 9.223372036854776e18, 1, 0}},
		{api.SDTReal32, 3, 0, SpinConfig{-f32, f32, 0.001, 3}},
		{api.SDTReal32, -1, 0, SpinConfig{-f32, f32, 0.001, 3}},
		{api.SDTReal64, 3, 0, SpinConfig{0, 1, 0.1, 3}}, // no REAL64 case: spin defaults
		{api.SDTVisibleString, -1, 0, DefaultSpin},
	}
	for _, tc := range tests {
		if got := NumberSpin(tc.sdt, tc.digits, tc.maxLen); got != tc.want {
			t.Errorf("NumberSpin(%d,%d,%d) = %+v, want %+v", tc.sdt, tc.digits, tc.maxLen, got, tc.want)
		}
	}
}

func TestParseSpinText(t *testing.T) {
	u8 := NumberSpin(api.SDTUnsigned8, 0, 0)
	i16 := NumberSpin(api.SDTInteger16, 0, 0)
	r32 := NumberSpin(api.SDTReal32, 3, 0)
	tests := []struct {
		text string
		cfg  SpinConfig
		want float64
	}{
		{"12", u8, 12},
		{" 7 ", u8, 7},
		{"12.5", u8, 12}, // Math.Round half to even
		{"13,5", u8, 14},
		{"300", u8, 255},
		{"-5", u8, 0},
		{"abc", u8, 0}, // unparseable -> Minimum
		{"", i16, -32768},
		{"5-", i16, -5},
		{"1,234.5", r32, 1.234}, // only "1" + "." + "234" survive the split
		{"0.0005", r32, 0},
		{"0.0015", r32, 0.002},
	}
	for _, tc := range tests {
		if got := ParseSpinText(tc.text, tc.cfg); got != tc.want {
			t.Errorf("ParseSpinText(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestCommitNumber(t *testing.T) {
	tests := []struct {
		sdt    api.SDT
		spin   float64
		factor float64
		want   string
	}{
		{api.SDTUnsigned8, 12, 1, "12"},
		{api.SDTUnsigned8, 12.5, 1, "0"}, // "12.5" does not byte.TryParse
		{api.SDTInteger8, -5, 1, "-5"},
		{api.SDTUnsigned16, 100, 10, "10"},
		{api.SDTReal32, 1.5, 1, "1.5"},
		{api.SDTReal64, 0.1, 1, "0.1"},
		{api.SDTUnsigned64, 1.8446744073709552e19, 1, "0"}, // G15 has an exponent
		{api.SDTUnsigned32, 4294967295, 1, "4294967295"},
	}
	for _, tc := range tests {
		if got := CommitNumber(tc.sdt, tc.spin, tc.factor).Wire(); got != tc.want {
			t.Errorf("CommitNumber(%d,%v,%v) = %q, want %q", tc.sdt, tc.spin, tc.factor, got, tc.want)
		}
	}
}

func TestSpinDisplay(t *testing.T) {
	if got := FormatSpin(SpinValue(ParseValue(api.SDTReal32, "12.5"), 1, 3), 3); got != "12.500" {
		t.Errorf("REAL32 display = %q", got)
	}
	if got := FormatSpin(SpinValue(ParseValue(api.SDTReal32, "2.5"), 1, 0), 0); got != "2" {
		t.Errorf("rounded display = %q", got)
	}
	if got := FormatSpin(SpinValue(ParseValue(api.SDTInteger32, "-1234567"), 1, 0), 0); got != "-1234567" {
		t.Errorf("no group separators: %q", got)
	}
	if got := SpinValue(ParseValue(api.SDTUnsigned8, "7"), 0.5, 0); got != 4 {
		t.Errorf("factor: %v", got)
	}
}

func TestTextInput(t *testing.T) {
	param := &Parameter{MaxLength: 100}
	ml := []struct {
		p    *Parameter
		typ  UIPropertyStringType
		dev  uint64
		want uint64
	}{
		{nil, StringTypeString, 0, 256},
		{param, StringTypeString, 0, 100},
		{param, StringTypeString, 1, 100},
		{nil, StringTypeString, 33, 32},
		{nil, StringTypeIP, 0, 15},
		{nil, StringTypeIP, 20, 19},
	}
	for _, tc := range ml {
		if got := TextMaxLen(tc.p, tc.typ, tc.dev); got != tc.want {
			t.Errorf("TextMaxLen = %d, want %d", got, tc.want)
		}
	}
	filt := []struct {
		in   string
		typ  UIPropertyStringType
		max  uint64
		want string
	}{
		{"192.168.a1.1x", StringTypeIP, 15, "192.168.1.1"},
		{"abcdef", StringTypeString, 3, "abc"},
		{"a\U0001F600b", StringTypeString, 2, "a"}, // surrogate pair counts 2 units
		{"a\U0001F600b", StringTypeString, 3, "a\U0001F600"},
		{"short", StringTypeString, 256, "short"},
	}
	for _, tc := range filt {
		if got := FilterTextInput(tc.in, tc.typ, tc.max); got != tc.want {
			t.Errorf("FilterTextInput(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	ips := []struct{ in, want string }{
		{"192.168.001.010", "192.168.1.10"},
		{"1", "0.0.0.1"},
		{"1.2", "1.0.0.2"},
		{"1.2.3", "1.2.0.3"},
		{"1.16777215", "1.255.255.255"},
		{"1.16777216", "0.0.0.0"},
		{"4294967295", "255.255.255.255"},
		{"256.1.1.1", "0.0.0.0"},
		{"1.2.3.4.5", "0.0.0.0"},
		{"1..2", "0.0.0.0"},
		{"000", "0.0.0.0"},
		{"", ""},
	}
	for _, tc := range ips {
		if got := NormalizeIPText(tc.in); got != tc.want {
			t.Errorf("NormalizeIPText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := TextDisplay(ParseValue(api.SDTVisibleString, "\x03abc\x1e")); got != "abc" {
		t.Errorf("TextDisplay = %q", got)
	}
}

func TestSelect(t *testing.T) {
	d := Default()
	dst := SelectOptions(d.FindParameter(0x205B, 0), nil)
	if len(dst) != 4 {
		t.Fatalf("205B options = %d", len(dst))
	}
	if got := SelectOptions(d.FindParameter(0x205B, 0), map[string]bool{"0": true, "3": true}); len(got) != 2 || got[0].Title != "EU" {
		t.Errorf("hidden options = %+v", got)
	}
	if SelectOptions(d.FindParameter(0x205A, 0), nil) != nil || SelectOptions(nil, nil) != nil {
		t.Error("no options must give nil")
	}
	sel := []struct {
		v      Value
		mask   int
		cs     bool
		opts   []Option
		want   string
		wantOK bool
	}{
		{ParseValue(api.SDTInteger8, "2"), 0, true, dst, "USA", true},
		{ParseValue(api.SDTInteger8, "7"), 0, true, dst, "", false},
		{ParseValue(api.SDTUnsigned8, "19"), 3, true, dst, "Australia", true}, // 19 & 3 = 3
		{ParseValue(api.SDTVisibleString, " twin 3.0 "), 0, false, d.FindParameter(0x2050, 0).Options, "Twin 3.0", true},
		{ParseValue(api.SDTVisibleString, "twin 3.0"), 0, true, d.FindParameter(0x2050, 0).Options, "", false},
	}
	for i, tc := range sel {
		o, ok := SelectedOption(tc.v, tc.mask, tc.cs, tc.opts)
		if ok != tc.wantOK || o.Title != tc.want {
			t.Errorf("%d: SelectedOption = %+v,%v", i, o, ok)
		}
	}
	cur := ParseValue(api.SDTUnsigned8, "19")
	if got := SelectWriteText(cur, "1", true, 3); got != "17" {
		t.Errorf("masked write = %q", got)
	}
	if got := SelectWriteText(cur, "", false, 0); got != "0" {
		t.Errorf("no selection = %q", got)
	}
	if v, w := CommitSelect(api.SDTUnsigned8, cur, "2", 0, true); !w || v.Wire() != "2" {
		t.Errorf("CommitSelect = %v,%v", v, w)
	}
	if _, w := CommitSelect(api.SDTVisibleString, ParseValue(api.SDTVisibleString, "Twin 3.0"), " twin 3.0", 0, false); w {
		t.Error("case-insensitive select must not rewrite an equal value")
	}
}

func TestCheckbox(t *testing.T) {
	checked := []struct {
		v    Value
		mask int
		want bool
	}{
		{ParseValue(api.SDTBoolean, "true"), 0, true},
		{ParseValue(api.SDTBoolean, "false"), 0, false},
		{ParseValue(api.SDTBoolean, "garbage"), 0, false},
		{ParseValue(api.SDTUnsigned16, "5"), 0, true},
		{ParseValue(api.SDTReal32, "0.5"), 0, false}, // Convert.ToInt32 rounds to even
		{ParseValue(api.SDTReal32, "1.5"), 0, true},
		{ParseValue(api.SDTUnsigned8, "5"), 4, true},
		{ParseValue(api.SDTUnsigned8, "5"), 2, false},
		{ParseValue(api.SDTVisibleString, "x"), 0, false},
	}
	for i, tc := range checked {
		if got := CheckboxChecked(tc.v, tc.mask); got != tc.want {
			t.Errorf("%d: CheckboxChecked = %v", i, got)
		}
	}
	commits := []struct {
		sdt     api.SDT
		cur     string
		checked bool
		mask    int
		want    string
	}{
		{api.SDTUnsigned8, "0", true, 0, "1"},
		{api.SDTInteger8, "1", false, 0, "0"},
		{api.SDTBoolean, "false", true, 0, "True"},
		{api.SDTUnsigned16, "0", true, 0, "0"}, // "True" does not parse as ushort
		{api.SDTUnsigned8, "1", true, 4, "5"},
		{api.SDTUnsigned8, "5", false, 4, "1"},
		{api.SDTInteger8, "-1", false, 0x80, "127"},
		{api.SDTInteger8, "1", true, 0x80, "-127"},
	}
	for _, tc := range commits {
		got := CommitCheckbox(tc.sdt, ParseValue(tc.sdt, tc.cur), tc.checked, tc.mask).Wire()
		if got != tc.want {
			t.Errorf("CommitCheckbox(%d,%q,%v,%d) = %q, want %q", tc.sdt, tc.cur, tc.checked, tc.mask, got, tc.want)
		}
	}
	from := []struct {
		sdt  api.SDT
		text string
		mask int
		want bool
	}{
		{api.SDTUnsigned8, "1", 0, true},
		{api.SDTUnsigned8, "2", 0, false},
		{api.SDTUnsigned8, "6", 4, true},
		{api.SDTBoolean, " 1 ", 0, true},
		{api.SDTBoolean, "True", 0, true},
		{api.SDTBoolean, "0", 0, false},
	}
	for _, tc := range from {
		if got := CheckboxFromValue(tc.sdt, tc.text, tc.mask); got != tc.want {
			t.Errorf("CheckboxFromValue(%q) = %v", tc.text, got)
		}
	}
}

func TestFormatReadOnly(t *testing.T) {
	tests := []struct {
		typ    UIPropertyStringType
		v      Value
		offset int
		want   string
		ok     bool
	}{
		{StringTypeString, ParseValue(api.SDTUnsigned8, "7"), 0, "7", true},
		{StringTypeFloat, ParseValue(api.SDTReal32, "12.25"), 0, "12.3", true},
		{StringTypeFloat2, ParseValue(api.SDTReal32, "12.25"), 0, "12.25", true},
		{StringTypeFloat0_01, ParseValue(api.SDTUnsigned16, "1234"), 0, "12.3", true},
		{StringTypeFloat2_0_01, ParseValue(api.SDTUnsigned16, "1234"), 0, "12.34", true},
		{StringTypeFloat0_001, ParseValue(api.SDTUnsigned32, "230000"), 0, "230.0", true},
		{StringTypeFloat2_0_1, ParseValue(api.SDTInteger16, "-15"), 0, "-1.50", true},
		{StringTypeFloat10, ParseValue(api.SDTReal32, "1.25"), 0, "12.5", true},
		{StringTypePWM, ParseValue(api.SDTUnsigned16, "1600"), 0, "16.0 (9.6A)", true},
		{StringTypePublicKey, ParseValue(api.SDTVisibleString, "ABCDEF0123"), 0, "abcd ef01 23", true},
		{StringTypeFloat, ParseValue(api.SDTVisibleString, ""), 0, "", false},
		{StringTypeFeatures, ParseValue(api.SDTUnsigned32, "1"), 0, "", false},
		{StringTypeNFC1SW, ParseValue(api.SDTVisibleString, "1"), 0, "", false},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "3600000"), 0, "01:00:00", true},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "90061000"), 0, "1.01:01:01", true},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "x"), 0, "00:00:00", true},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "1700000000000"), 60, "Tuesday, 14 November 2023 23:13:20", true},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "1700000000000"), -300, "Tuesday, 14 November 2023 17:13:20", true},
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "1700000000000"), 6, "", false}, // no zone with a 6-minute offset
		{StringTypeDateTime, ParseValue(api.SDTUnsigned64, "18446744073709551615"), 0, "", false},
	}
	for i, tc := range tests {
		got, ok := FormatReadOnly(tc.typ, tc.v, tc.offset)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%d: FormatReadOnly(%s) = %q,%v want %q,%v", i, tc.typ, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDeviceUTCOffsetMinutes(t *testing.T) {
	tz := api.Property{ID: 0x205A, DataType: api.SDTInteger8, Value: "1"}
	tzm := api.Property{ID: 0x206E, DataType: api.SDTInteger8, Value: "60"}
	tzmUnsupported := api.Property{ID: 0x206E, DataType: api.SDTDomain, Value: "60"}
	tests := []struct {
		props []api.Property
		want  int
	}{
		{[]api.Property{tz, tzm}, 60},
		{[]api.Property{tz}, 6},
		{[]api.Property{tz, tzmUnsupported}, 6},
		{nil, 0},
	}
	for i, tc := range tests {
		if got := DeviceUTCOffsetMinutes(tc.props); got != tc.want {
			t.Errorf("%d: offset = %d, want %d", i, got, tc.want)
		}
	}
}

func TestLEDPattern(t *testing.T) {
	raw := []byte{0xFF, 0x80, 0x00, 0x11, 50, 0x00, 0x00, 0xFF, 0x22, 3}
	p, ok := ParseLEDPattern(raw)
	if !ok || p.Color1 != [3]byte{0xFF, 0x80, 0} || p.Time1 != 500 || p.Color2 != [3]byte{0, 0, 0xFF} || p.Time2 != 30 {
		t.Fatalf("ParseLEDPattern = %+v,%v", p, ok)
	}
	if got := p.Encode(); got != "FF,80,00,11,32,00,00,FF,22,03" {
		t.Errorf("Encode = %q", got)
	}
	p.Time1, p.Time2 = 2550, 2560
	if b := p.Bytes(); b[4] != 255 || b[9] != 0 || b[3] != 0x11 || b[8] != 0x22 {
		t.Errorf("Bytes = % X", b)
	}
	if _, ok := ParseLEDPattern(raw[:9]); ok {
		t.Error("9 bytes must be rejected")
	}
	v := ParseValue(api.SDTByteArray, "FF,80,00,11,32,00,00,FF,22,03")
	if p2, ok := ParseLEDPattern(v.Bytes); !ok || p2.Encode() != "FF,80,00,11,32,00,00,FF,22,03" {
		t.Error("BYTEARRAY round trip")
	}
}

func TestLabelStyles(t *testing.T) {
	if s := StyleFor(LabelHeader); s.Color != ColorSteelBlue || s.Scale != 1.5 || s.Weight != WeightSemibold || s.ColSpan != 3 {
		t.Errorf("Header = %+v", s)
	}
	if s := StyleFor(LabelNormal); s.Color != ColorBlack || !s.Centered || s.ColSpan != 1 {
		t.Errorf("Normal = %+v", s)
	}
	if s := StyleFor(LabelLargeWarning); s.Color != ColorOrange || s.Weight != WeightBold || s.Scale != 1.1 {
		t.Errorf("LargeWarning = %+v", s)
	}
	if s := StyleFor(LabelLargeError); s.Color != ColorRed || s.Weight != WeightSemibold {
		t.Errorf("LargeError = %+v", s)
	}
	if LabelLines("a\nb\nc", 1) != 3 || LabelLines("a", 2) != 2 {
		t.Error("LabelLines")
	}
	if StringTypeFloat2_10.String() != "Float2_10" || StringTypeNFC1SW.String() != "NFC1_SW" ||
		LabelLargeError.String() != "LargeError" || NumberTypeFloat.String() != "Float" ||
		QuestionNA.String() != "NA" || QuestionTypeImportant.String() != "Important" || KindReadOnlyText.String() != "ReadOnlyText" {
		t.Error("enum names")
	}
	if ok, nok, na := OkNokNaActive(QuestionNOK); ok || !nok || na {
		t.Error("OkNokNaActive")
	}
	if OkNokNaLabels[QuestionNA] != "N.V.T." || OkNokNaButtonColor(QuestionTypeImportant) != ColorOrange {
		t.Error("OkNokNa presentation")
	}
}
