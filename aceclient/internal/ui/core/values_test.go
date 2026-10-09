package core

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"alfen/aceclient/internal/api"
)

func TestConvertValue(t *testing.T) {
	cases := []struct {
		t    api.SDT
		in   string
		want string
		ok   bool
	}{
		{api.SDTVisibleString, " abc ", " abc ", true},
		{api.SDTUnsigned8, "true", "1", true},
		{api.SDTUnsigned8, "FALSE", "0", true},
		{api.SDTUnsigned8, " 200 ", "200", true},
		{api.SDTUnsigned8, "256", "0", true}, // overflow → 0
		{api.SDTUnsigned8, "-1", "0", true},  // no sign allowed
		{api.SDTInteger8, "-128", "-128", true},
		{api.SDTInteger8, "True", "1", true},
		{api.SDTUnsigned16, "+5", "0", true}, // AllowLeadingSign not set
		{api.SDTUnsigned16, "65535", "65535", true},
		{api.SDTInteger16, "+5", "5", true},
		{api.SDTInteger32, "1.5", "0", true},
		{api.SDTUnsigned64, "18446744073709551615", "18446744073709551615", true},
		{api.SDTInteger64, "-9223372036854775808", "-9223372036854775808", true},
		{api.SDTReal32, "16", "16", true},
		{api.SDTReal32, "16.700001", "16.7", true},
		{api.SDTReal32, "25,5", "25.5", true},
		{api.SDTReal32, "1e-5", "1E-05", true},
		{api.SDTReal32, "1e40", "0", true}, // .NET Framework: overflow fails
		{api.SDTReal32, "abc", "0", true},
		{api.SDTReal64, "0.1", "0.1", true},
		{api.SDTReal64, "123456789012345678", "1.23456789012346E+17", true},
		{api.SDTReal64, "-0", "0", true},
		{api.SDTBoolean, "true", "True", true},
		{api.SDTBoolean, " False ", "False", true},
		{api.SDTBoolean, "1", "0", true}, // bool.TryParse fails → (byte)0
		{api.SDTByteArray, "0a,1F,zz", "0A,1F,00", true},
		{api.SDTByteArray, "", "00", true},
		{api.SDTArray16, "1,ABCD", "0001,ABCD", true},
		{api.SDTLowLimit, "x", "", true},
		{api.SDTUnicodeString, "x", "", false},
		{api.SDTOctetString, "x", "", false},
		{api.SDTDomain, "x", "", false},
		{api.SDTUnsigned24, "1", "", false},
	}
	for _, c := range cases {
		got, ok := ConvertValue(c.t, c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ConvertValue(%s, %q) = %q,%v; want %q,%v", SDTName(c.t), c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestCheckboxValueMirrorsInstaller(t *testing.T) {
	cases := []struct {
		t      api.SDT
		active bool
		want   string
	}{
		{api.SDTUnsigned8, true, "1"},
		{api.SDTInteger8, false, "0"},
		{api.SDTBoolean, true, "True"},
		{api.SDTBoolean, false, "False"},
		{api.SDTUnsigned16, true, "0"}, // bool text fails ushort.TryParse in the C#
		{api.SDTInteger32, true, "0"},
	}
	for _, c := range cases {
		if got := CheckboxValue(c.t, c.active); got != c.want {
			t.Errorf("CheckboxValue(%s,%v) = %q, want %q", SDTName(c.t), c.active, got, c.want)
		}
	}
	for in, want := range map[string]bool{"True": true, "False": false, "0": false, "5": true, "": false} {
		if got := CheckState(in); got != want {
			t.Errorf("CheckState(%q) = %v", in, got)
		}
	}
}

func TestNumberInput(t *testing.T) {
	cases := []struct {
		t       api.SDT
		maxLen  uint64
		digits  int
		in      string
		want    string
		clamped bool
		err     bool
	}{
		{api.SDTUnsigned16, 0, 0, "300", "300", false, false},
		{api.SDTUnsigned16, 0, 0, "70000", "65535", true, false},
		{api.SDTUnsigned16, 100, 0, "300", "100", true, false}, // MaxLength caps MaximumValue
		{api.SDTInteger8, 0, 0, "-200", "-128", true, false},
		{api.SDTReal32, 0, 3, "25,5", "25.5", false, false},
		{api.SDTReal32, 0, 3, "1.23456", "1.235", false, false},
		{api.SDTUnsigned8, 0, 0, "x", "", false, true},
	}
	for _, c := range cases {
		got, clamped, err := NumberInput(c.t, c.maxLen, c.digits, c.in)
		if (err != nil) != c.err || got != c.want || clamped != c.clamped {
			t.Errorf("NumberInput(%s,%d,%d,%q) = %q,%v,%v; want %q,%v,err=%v", SDTName(c.t), c.maxLen, c.digits, c.in, got, clamped, err, c.want, c.clamped, c.err)
		}
	}
}

func TestTextInputMaxLength(t *testing.T) {
	p := Property{Property: api.Property{DataType: api.SDTVisibleString, MaxLength: 6}}
	if got := TextInput(p, nil, "ABCDEFGH"); got != "ABCDE" { // MaxLength-1
		t.Fatalf("TextInput = %q", got)
	}
	p.MaxLength = 0
	if got := TextMaxLength(p, nil); got != 256 {
		t.Fatalf("default max = %d", got)
	}
	if got := TextMaxLength(p, &EDSParam{MaxLength: 33}); got != 33 {
		t.Fatalf("EDS max = %d", got)
	}
}

func TestVersions(t *testing.T) {
	if v, ok := ParseVersion("7.4.6"); !ok || v.String() != "7.4.6" || v.Build != 6 || v.Revision != -1 {
		t.Fatalf("ParseVersion = %v %v", v, ok)
	}
	for _, bad := range []string{"7", "1.2.3.4.5", "1.-2", "a.b", ""} {
		if _, ok := ParseVersion(bad); ok {
			t.Errorf("ParseVersion(%q) accepted", bad)
		}
	}
	if !MustVersion("4.3").Less(MustVersion("4.3.0")) {
		t.Error("undefined build must sort before 0")
	}
	cases := []struct {
		in    string
		https bool
		want  string
	}{
		{"7.4.6-4416", true, "7.4.6"},
		{"6.X.0-123", true, "6.99.0"},
		{"", true, "4.10.0"},
		{"", false, "4.9.0"},
		{"garbage-1", true, "0.0"},
	}
	for _, c := range cases {
		if got := FirmwareVersionNumber(c.in, c.https).String(); got != c.want {
			t.Errorf("FirmwareVersionNumber(%q,%v) = %s, want %s", c.in, c.https, got, c.want)
		}
	}
}

func TestLoginErrorMessage(t *testing.T) {
	cases := []struct {
		status  int
		content string
		want    string
	}{
		{401, "", "Secure Service Access is disabled for device 'CP1'"},
		{403, "", "The password for device 'CP1' is incorrect! Please provide a valid password."},
		{429, `{"lockout_remaining_seconds":120}`, "Locked out of device 'CP1' for 2.0 minutes due to multiple incorrect password attempts."},
		{429, `{"lockout_remaining_seconds":60}`, "Locked out of device 'CP1' for 1.0 minute due to multiple incorrect password attempts."},
		{429, ``, "Locked out of device 'CP1' due to multiple incorrect password attempts."},
		{429, `{"lockout_remaining_seconds":0}`, "Locked out of device 'CP1' due to multiple incorrect password attempts."},
		{429, `<html>`, "Login failed"},
		{408, "", "Login request timeout for CP1 and IP 10.0.0.2"},
		{503, "", "Device CP1/10.0.0.2 is not reachable"},
		{500, "", "Unknown error"},
	}
	for _, c := range cases {
		if got := LoginErrorMessage(c.status, c.content, "CP1", "10.0.0.2"); got != c.want {
			t.Errorf("LoginErrorMessage(%d,%q) = %q, want %q", c.status, c.content, got, c.want)
		}
	}
	if _, show := ConnectFailureText(403, "1.2.3.4", "x"); show {
		t.Error("403 must not show the generic failure popup")
	}
	if msg, show := ConnectFailureText(500, "1.2.3.4", "boom"); !show || msg != "Failed to communicate with device at IP address 1.2.3.4\nError: boom" {
		t.Errorf("ConnectFailureText = %q %v", msg, show)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestStatusFromError(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{fmt.Errorf("login failed: HTTP 403: nope"), 403},
		{fmt.Errorf("prop read HTTP 401: {}"), 401},
		{&url.Error{Op: "Get", URL: "https://x", Err: timeoutErr{}}, StatusRequestTimeout},
		{&url.Error{Op: "Get", URL: "https://x", Err: errors.New("weird")}, StatusUnused},
		{errors.New("decode prop page: bad"), 0},
	}
	for _, c := range cases {
		if got, _ := StatusFromError(c.err); got != c.want {
			t.Errorf("StatusFromError(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestStoredPasswordExpiredIsReversedLikeInstaller(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if StoredPasswordExpired(now.Add(-48*time.Hour), now) {
		t.Error("a store time in the past never expires (C# subtraction is reversed)")
	}
	if !StoredPasswordExpired(now.Add(48*time.Hour), now) {
		t.Error("a store time > 24 h in the future expires")
	}
	if f := LoginProgressFraction(7500*time.Millisecond, 15); f != 0.5 {
		t.Errorf("LoginProgressFraction = %v", f)
	}
}

func TestDeviceListOrderingAndFilter(t *testing.T) {
	l := NewDeviceList()
	l.Upsert(Device{Identity: "Zeta", Address: "10.0.0.9", Discovered: false})
	l.Upsert(Device{Identity: "Beta", Address: "10.0.0.3", SCNNetwork: "B-net", Discovered: true})
	l.Upsert(Device{Identity: "Alpha", Address: "10.0.0.4", SCNNetwork: "B-net", Discovered: true})
	l.Upsert(Device{Identity: "Gamma", Address: "10.0.0.5", SCNNetwork: "A-net", HostName: "ng910-60023-ace01", Discovered: true})
	rows := l.Rows("")
	var got []string
	for _, d := range rows {
		got = append(got, d.Identity)
	}
	want := "[Gamma Alpha Beta Zeta]" // HasSCNNetwork desc, SCNNetwork, Identification
	if fmt.Sprint(got) != want {
		t.Fatalf("order = %v, want %s", got, want)
	}
	for filter, n := range map[string]int{"": 4, "ace01": 1, "10.0.0.": 4, "ALPHA": 1, "nomatch": 0} {
		if c := len(l.Rows(filter)); c != n {
			t.Errorf("Rows(%q) = %d, want %d", filter, c, n)
		}
	}
	l.Clear()
	if l.Len() != 0 {
		t.Fatal("Clear")
	}
}

func TestParseIPAddressAndPort(t *testing.T) {
	if _, err := ParseIPAddress("192.168.1.10"); err != nil {
		t.Fatal(err)
	}
	_, err := ParseIPAddress("192.168.1")
	if err == nil || err.Primary != "Invalid IP addres!" || err.Secondary != "Please enter an ip address in the following form: 192.168.1.10" {
		t.Fatalf("err = %#v", err)
	}
	if ValidatePort(443) != nil || ValidatePort(0) == nil || ValidatePort(70000) == nil {
		t.Fatal("ValidatePort")
	}
	if ProtocolForPort(443) != "https" || ProtocolForPort(80) != "http" || ProtocolForPort(8443) != "http" {
		t.Fatal("ProtocolForPort")
	}
}

func TestAnnouncedFirmwareVersion(t *testing.T) {
	for in, want := range map[string]string{"7.4.6-4416": "7.4.6", " 6.X.0-12 ": "6.99.0", "2.7.0": "2.7.0"} {
		if v, ok := AnnouncedFirmwareVersion(in); !ok || v.String() != want {
			t.Errorf("AnnouncedFirmwareVersion(%q) = %v,%v want %s", in, v, ok, want)
		}
	}
	if _, ok := AnnouncedFirmwareVersion("garbage"); ok {
		t.Error("garbage accepted")
	}
}
