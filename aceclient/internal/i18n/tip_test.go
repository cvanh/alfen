package i18n

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParseTipLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Tip
		ok   bool
	}{
		{"empty", "", Tip{}, false},
		{"comment", "// General", Tip{}, false},
		{"three fields", "2062_00, OD_x , Label", Tip{}, false},
		{"exactly four", "2062_00, OD_x , Label , Tip",
			Tip{ID: "2062_00", Name: "OD_x", LabelText: "Label", Tooltip: "Tip"}, true},
		{"commas in tooltip are re-joined", "2062_00,OD_x,Label, a, b ,c ",
			Tip{ID: "2062_00", Name: "OD_x", LabelText: "Label", Tooltip: "a, b ,c"}, true},
		// tooltip_en_GB.csv line 33: a stray comma after the label shifts it
		// into the tooltip column.
		{"stray comma after label", "2189_00, OD_mainMaxAllowedPhases , Maximum Allowed Phases, , The maximum",
			Tip{ID: "2189_00", Name: "OD_mainMaxAllowedPhases", LabelText: "Maximum Allowed Phases", Tooltip: ", The maximum"}, true},
		{"tabs and unicode whitespace trimmed only at ends", "\t1 , 　n\t, l a\tb  ,\t t\tt \u0085",
			Tip{ID: "1", Name: "n", LabelText: "l a\tb", Tooltip: "t\tt"}, true},
		{"zero-width space is not whitespace", "​X,n,l,t",
			Tip{ID: "​X", Name: "n", LabelText: "l", Tooltip: "t"}, true},
		{"quotes are literal (no RFC 4180)", `"A,B",n,l,"x"`,
			Tip{ID: `"A`, Name: `B"`, LabelText: "n", Tooltip: `l,"x"`}, true},
		{"comment with three commas is a tip", "// a, b, c, d",
			Tip{ID: "// a", Name: "b", LabelText: "c", Tooltip: "d"}, true},
		{"empty fields", ",,,",
			Tip{}, true},
		{"backslash-n kept raw", `X,n,l,a\nb`,
			Tip{ID: "X", Name: "n", LabelText: "l", Tooltip: `a\nb`}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseTipLine(tc.line)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("parseTipLine(%q) = %+v, %v; want %+v, %v", tc.line, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestReadLines(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a"}},
		{"a\n\n", []string{"a", ""}},
		{"\n", []string{""}},
		{"a\r\nb\rc\nd", []string{"a", "b", "c", "d"}},
		{"a\r\r\nb", []string{"a", "", "b"}},
		{"a\n\rb", []string{"a", "", "b"}},
		{"a\r", []string{"a"}},
	}
	for _, tc := range tests {
		if got := readLines(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("readLines(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestDecodeStreamText(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want string
	}{
		{"plain utf8", []byte("a°"), "a°"},
		{"utf8 bom", []byte("\xef\xbb\xbfa°"), "a°"},
		{"utf16le bom", []byte{0xFF, 0xFE, 'a', 0, 0xB0, 0}, "a°"},
		{"utf16be bom", []byte{0xFE, 0xFF, 0, 'a', 0, 0xB0}, "a°"},
		{"utf16le surrogate pair", []byte{0xFF, 0xFE, 0x3D, 0xD8, 0x00, 0xDE}, "\U0001F600"},
		{"utf16le bom only", []byte{0xFF, 0xFE}, ""},
		{"utf16le lone low surrogate", []byte{0xFF, 0xFE, 0x00, 0xDE, 'a', 0}, "�a"},
		{"utf16le high surrogate then BMP", []byte{0xFF, 0xFE, 0x3D, 0xD8, 'a', 0}, "�a"},
		// .NET 4.8 StreamReader never flushes its Decoder at EOF, so an
		// incomplete trailing unit stays pending and is silently dropped.
		{"utf16le odd trailing byte dropped", []byte{0xFF, 0xFE, 'a', 0, 'b'}, "a"},
		{"utf16le trailing high surrogate dropped", []byte{0xFF, 0xFE, 'a', 0, 0x3D, 0xD8}, "a"},
		{"utf16le trailing high surrogate + odd byte dropped", []byte{0xFF, 0xFE, 'a', 0, 0x3D, 0xD8, 0x00}, "a"},
		{"utf16le two trailing high surrogates", []byte{0xFF, 0xFE, 0x3D, 0xD8, 0x3D, 0xD8}, "�"},
		{"utf16be odd trailing byte dropped", []byte{0xFE, 0xFF, 0, 'a', 0}, "a"},
		{"utf32le bom", []byte{0xFF, 0xFE, 0, 0, 'a', 0, 0, 0}, "a"},
		{"utf32be bom", []byte{0, 0, 0xFE, 0xFF, 0, 0, 0, 'a'}, "a"},
		{"utf32le supplementary", []byte{0xFF, 0xFE, 0, 0, 0x00, 0xF6, 0x01, 0x00}, "\U0001F600"},
		{"utf32le surrogate code point", []byte{0xFF, 0xFE, 0, 0, 0x00, 0xD8, 0, 0, 'a', 0, 0, 0}, "�a"},
		{"utf32le above U+10FFFF", []byte{0xFF, 0xFE, 0, 0, 0x00, 0x00, 0x11, 0x00}, "�"},
		{"utf32le 0xFFFFFFFF", []byte{0xFF, 0xFE, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF}, "�"},
		{"utf32be above U+10FFFF", []byte{0, 0, 0xFE, 0xFF, 0x00, 0x11, 0x00, 0x00}, "�"},
		{"utf32le 1-byte tail dropped", []byte{0xFF, 0xFE, 0, 0, 'a', 0, 0, 0, 'b'}, "a"},
		{"utf32le 3-byte tail dropped", []byte{0xFF, 0xFE, 0, 0, 'a', 0, 0, 0, 'b', 0, 0}, "a"},
		{"invalid utf8 replaced", []byte("a\xffb"), "a�b"},
		{"invalid utf8 lead at EOF replaced", []byte("a\xff"), "a�"},
		{"non-shortest lead at EOF replaced", []byte("a\xc0"), "a�"},
		{"truncated utf8 at EOF dropped", []byte("a\xe2\x82"), "a"},
		{"truncated 4-byte utf8 at EOF dropped", []byte("a\xf0\x9f\x98"), "a"},
		{"truncated utf8 after newline dropped", []byte("a\n\xe2"), "a\n"},
		{"partial utf8 bom only", []byte("\xef\xbb"), ""},
		{"lone utf8 lead byte", []byte("\xef"), ""},
		{"bom only", []byte("\xef\xbb\xbf"), ""},
		{"single byte", []byte("x"), "x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decodeStreamText(tc.in); got != tc.want {
				t.Fatalf("decodeStreamText(% x) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseTooltips(t *testing.T) {
	in := "\xef\xbb\xbf// Header\r\n" +
		"1008_00, OD_a , Label A , Tip A\r\n" +
		"\r\n" +
		"bad line, only, three\n" +
		"1008_00, OD_dup , Label dup , Tip dup\r" +
		"2062_00, OD_b , Label B , Tip, with, commas"
	c, err := ParseTooltips(strings.NewReader(in), LanguageDutch)
	if err != nil {
		t.Fatal(err)
	}
	if c.Language != LanguageDutch {
		t.Errorf("Language = %v", c.Language)
	}
	want := []Tip{
		{"1008_00", "OD_a", "Label A", "Tip A"},
		{"1008_00", "OD_dup", "Label dup", "Tip dup"},
		{"2062_00", "OD_b", "Label B", "Tip, with, commas"},
	}
	if !reflect.DeepEqual(c.Tips, want) {
		t.Fatalf("Tips = %+v; want %+v", c.Tips, want)
	}
	if got := c.Get("1008_00"); got == nil || got.Name != "OD_a" {
		t.Errorf("Get first-wins: got %+v", got)
	}
	// Tip is read-only in the C#: Get hands out a copy.
	if got := c.Get("2062_00"); got != nil {
		got.Tooltip = "changed"
		if c.Tips[2].Tooltip != "Tip, with, commas" || c.Get("2062_00").Tooltip != "Tip, with, commas" {
			t.Error("writing to the Tip returned by Get changed the collection")
		}
	} else {
		t.Error("Get(2062_00) = nil")
	}
	if got := c.Get("1008_0"); got != nil {
		t.Errorf("Get must be exact: got %+v", got)
	}
	if got := c.Get("2062_00 "); got != nil {
		t.Errorf("Get must not trim: got %+v", got)
	}
	var nilColl *TooltipCollection
	if nilColl.Get("x") != nil {
		t.Error("nil collection Get must be nil")
	}
}

func TestParseTooltipsReadError(t *testing.T) {
	boom := errors.New("boom")
	if c, err := ParseTooltips(iotest.ErrReader(boom), LanguageEnglish); c != nil || !errors.Is(err, boom) {
		t.Fatalf("got %v, %v; want nil, boom", c, err)
	}
}
