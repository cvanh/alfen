package isah

import (
	"encoding/json"
	"math"
	"math/big"
	"testing"
)

func TestNewArticle(t *testing.T) {
	a := NewArticle("P", "row", 3, "d")
	if *a != (Article{Name: "P", PartCode: "P", DetailCode: "row", Quantity: 3, Description: "d"}) {
		t.Errorf("NewArticle = %+v", a)
	}
}

func TestValueStringDotNetFormats(t *testing.T) {
	b, _ := new(big.Int).SetString("-123456789012345678901234567890", 10)
	cases := []struct {
		v    any
		want string
	}{
		{nil, ""},
		{"abc", "abc"},
		{true, "True"},
		{false, "False"},
		{int64(-42), "-42"},
		{b, "-123456789012345678901234567890"},
		{json.RawMessage(`{"a":1}`), `{"a":1}`},
		// Double.ToString() on .NET Framework is "G" (15 significant digits).
		{0.1, "0.1"},
		{0.1 + 0.2, "0.3"},
		{1.5, "1.5"},
		{-2.25, "-2.25"},
		{100.0, "100"},
		{1e14, "100000000000000"},
		{1e15, "1E+15"},
		{1e20, "1E+20"},
		{123456789012345678.0, "1.23456789012346E+17"},
		{0.0001, "0.0001"},
		{0.00001, "1E-05"},
		{1.0 / 3, "0.333333333333333"},
		{math.Copysign(0, -1), "0"},
		{math.NaN(), "NaN"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
	}
	for _, c := range cases {
		p := IWSPropertyValue{ID: 0x21A1, SubID: 0, Value: c.v}
		if got := p.ValueString(); got != c.want {
			t.Errorf("ValueString(%#v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestDebuggerDisplay(t *testing.T) {
	cases := []struct {
		p    IWSPropertyValue
		want string
	}{
		{IWSPropertyValue{ID: 8609, SubID: 0, Value: "KEY"}, "21A1_00: KEY"},
		{IWSPropertyValue{ID: 0x2A, SubID: 0xB, Value: int64(1)}, "002A_0B: 1"},
		{IWSPropertyValue{ID: 0xFFFF, SubID: 0xFF, Value: nil}, "FFFF_FF: "},
	}
	for _, c := range cases {
		if got := c.p.DebuggerDisplay(); got != c.want {
			t.Errorf("DebuggerDisplay = %q, want %q", got, c.want)
		}
		if got := c.p.String(); got != c.want {
			t.Errorf("String = %q, want %q", got, c.want)
		}
	}
}

func TestIWSObjectHelpers(t *testing.T) {
	o := &IWSObject{}
	if o.IsLanguageDefined() || o.HasPersonalizedLogo() || o.FindProperty(PropLicenseKey) != nil {
		t.Error("zero object helpers")
	}
	o.Logo = "aGVsbG8="
	if o.HasPersonalizedLogo() {
		t.Error("logo without IsPersonalizedDisplay")
	}
	o.IsPersonalizedDisplay = true
	if !o.HasPersonalizedLogo() {
		t.Error("logo with IsPersonalizedDisplay")
	}
	for _, in := range []string{"aGVsbG8=", " aGVs\r\n\tbG8= ", "aGVs bG8="} {
		o.Logo = in
		if b, err := o.LogoData(); err != nil || string(b) != "hello" {
			t.Errorf("LogoData(%q) = %q, %v", in, b, err)
		}
	}
	for _, in := range []string{"aGVsbG8", "!!!!"} {
		o.Logo = in
		if _, err := o.LogoData(); err == nil {
			t.Errorf("LogoData(%q) should fail", in)
		}
	}
	o.Properties = []*IWSPropertyValue{nil, {ID: 8609, Value: "a"}, {ID: 8609, Value: "b"}}
	if p := o.FindProperty(8609); p == nil || p.Value != "a" {
		t.Errorf("FindProperty should return the first match: %v", p)
	}
}

func TestHTTPResponseSuccess(t *testing.T) {
	for code, want := range map[int]bool{199: false, 200: true, 204: true, 299: true, 300: false, 404: false, 408: false} {
		if got := (&HTTPResponse{StatusCode: code}).IsSuccessStatusCode(); got != want {
			t.Errorf("IsSuccessStatusCode(%d) = %v", code, got)
		}
	}
	var nilResp *HTTPResponse
	if nilResp.IsSuccessStatusCode() {
		t.Error("nil response is not a success")
	}
	r := requestTimeoutResponse()
	if r.StatusCode != 408 || r.Status != "Request Timeout" || len(r.Body) != 0 {
		t.Errorf("requestTimeoutResponse = %+v", r)
	}
}
