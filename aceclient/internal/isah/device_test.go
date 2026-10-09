package isah

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// testUser is a placeholder identity: real values come from the caller's
// config and are never part of this package.
var testUser = &User{User: "installer.user", Company: "test-company"}

func TestLoadSettingsFromIWSRequest(t *testing.T) {
	f := newFakeIWS(t, jsonReply(200, `{"ObjectId":"ace0001234","Properties":[{"Id":8609,"SubId":0,"Value":"NEWKEY"}],"Logo":"","IsPersonalizedDisplay":false}`))
	dev := IWSDevice{ObjectID: "  ACE0001234 \n", NumberOfSockets: 2, ProcessorID: 18446744073709551615}
	for _, includeLogo := range []bool{true, false} {
		obj, err := f.conn().LoadSettingsFromIWS(context.Background(), testUser, f.srv.URL, testUser.Company, dev, includeLogo)
		if err != nil {
			t.Fatalf("includeLogo=%v: %v", includeLogo, err)
		}
		if obj.FindProperty(PropLicenseKey).ValueString() != "NEWKEY" {
			t.Errorf("license key = %v", obj.FindProperty(PropLicenseKey))
		}
	}
	reqs := f.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	for i, got := range reqs {
		if got.Method != http.MethodGet || got.Path != "/api/settings/ace0001234" {
			t.Errorf("req %d: %s %s", i, got.Method, got.Path)
		}
		// includeLogo is never forwarded: IncludeLogo is always True.
		want := "NumberOfSockets=2&ProcessorId=18446744073709551615&AdditionalInfo=installer.user&ProductionRequest=False&ObjectCode=&IncludeLogo=True"
		if got.RawQuery != want {
			t.Errorf("req %d query = %q\nwant          %q", i, got.RawQuery, want)
		}
		if h := got.Header.Get("Authorization"); h != basic("test-company") {
			t.Errorf("req %d Authorization = %q", i, h)
		}
	}
}

func TestLoadSettingsFromIWSFailures(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{"unknown object", jsonReply(404, `{}`), 404},
		{"null object", jsonReply(200, `null`), 200},
		{"bad json", jsonReply(200, `{`), 408},
	}
	for _, c := range cases {
		f := newFakeIWS(t, c.handler)
		obj, err := f.conn().LoadSettingsFromIWS(context.Background(), testUser, f.srv.URL, "x", IWSDevice{ObjectID: "12345R01", NumberOfSockets: 1}, true)
		var unk *UnknownObjectError
		if obj != nil || !errors.As(err, &unk) {
			t.Fatalf("%s: %v, %v", c.name, obj, err)
		}
		if unk.ObjectID != "12345r01" || unk.Response.HTTPResponse.StatusCode != c.wantStatus {
			t.Errorf("%s: %+v / %d", c.name, unk, unk.Response.HTTPResponse.StatusCode)
		}
		if msg := err.Error(); msg != `The current "12345r01" is unknown! Please contact Alfen!` {
			t.Errorf("%s: message %q", c.name, msg)
		}
	}

	// No user: no request at all.
	f := newFakeIWS(t, jsonReply(200, `{}`))
	if obj, err := f.conn().LoadSettingsFromIWS(context.Background(), nil, f.srv.URL, "x", IWSDevice{}, true); obj != nil || !errors.Is(err, ErrNoCurrentUser) {
		t.Errorf("nil user: %v, %v", obj, err)
	}
	if n := len(f.requests()); n != 0 {
		t.Errorf("nil user made %d requests", n)
	}

	// Invalid site: thrown by new Uri, caught by LoadSettingsFromIWS -> error.
	obj, err := LoadSettingsFromIWS(context.Background(), testUser, "not a uri", "x", IWSDevice{}, true)
	var unk *UnknownObjectError
	if obj != nil || err == nil || errors.As(err, &unk) {
		t.Errorf("invalid site: %v, %v", obj, err)
	}
}

func TestObjectIDValidation(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"12345r01", true},
		{"123456r123", true},
		{"12345R01", true}, // IgnoreCase
		{"1234r01", false},
		{"1234567r01", false},
		{"12345r1", false},
		{"12345r1234", false},
		{"ace0001234", true},
		{"ACE0001234", true},
		{"ace000123", false},
		{"ace00012345", false},
		{"xace0001234", false},
		{"", false},
		{"unknown", false},
	}
	for _, c := range cases {
		err := ValidateObjectID(c.in)
		if (err == nil) != c.valid {
			t.Errorf("ValidateObjectID(%q) = %v, want valid=%v", c.in, err, c.valid)
		}
		if err != nil && err.Error() != "Invalid Object ID: "+c.in+"!\nPlease provide a correct one." {
			t.Errorf("message = %q", err.Error())
		}
	}
	if got := NormalizeObjectID("  ACE0001234\t\n"); got != "ace0001234" {
		t.Errorf("NormalizeObjectID = %q", got)
	}
	if ValidateObjectID(NormalizeObjectID(" 12345R01 ")) != nil {
		t.Error("normalized old-style ID should validate")
	}
}

func TestObjectIDCheckMessages(t *testing.T) {
	cases := []struct {
		c       ObjectIDCheck
		confirm bool
		msg     string
		name    string
	}{
		{ObjectIDUnverified, true, "Unable to verify the entered Object ID with ISAH. Are you sure to change the Object ID to ace0001234?", "ObjectIDUnverified"},
		{ObjectIDKnown, false, "The Object ID of the Charge Station will be changed to ace0001234.", "ObjectIDKnown"},
		{ObjectIDUnknown, true, "The entered Object ID seems to be invalid. Are you sure to change the Object ID to ace0001234?", "ObjectIDUnknown"},
	}
	for _, c := range cases {
		if c.c.NeedsConfirmation() != c.confirm || c.c.Message("ace0001234") != c.msg || c.c.String() != c.name {
			t.Errorf("%v: confirm=%v msg=%q", c.c, c.c.NeedsConfirmation(), c.c.Message("ace0001234"))
		}
	}
	if s := ObjectIDCheck(9).String(); s != "ObjectIDCheck(9)" {
		t.Errorf("String = %q", s)
	}
}

func TestCheckIfObjectIDExists(t *testing.T) {
	route := func(site, api int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/settings/") {
				jsonReply(api, `{"ObjectId":"x"}`)(w, r)
				return
			}
			jsonReply(site, ``)(w, r)
		}
	}
	cases := []struct {
		name      string
		user      *User
		handler   http.HandlerFunc
		want      ObjectIDCheck
		wantCalls int
	}{
		{"no user", nil, route(200, 200), ObjectIDUnverified, 0},
		{"site down", testUser, route(503, 200), ObjectIDUnverified, 1},
		{"known", testUser, route(200, 200), ObjectIDKnown, 2},
		{"unknown", testUser, route(200, 404), ObjectIDUnknown, 2},
	}
	for _, c := range cases {
		f := newFakeIWS(t, c.handler)
		got, err := f.conn().CheckIfObjectIDExists(context.Background(), f.srv.URL, c.user, "ace0001234", 2, 77)
		if err != nil || got != c.want {
			t.Errorf("%s: %v, %v; want %v", c.name, got, err, c.want)
		}
		reqs := f.requests()
		if len(reqs) != c.wantCalls {
			t.Fatalf("%s: %d requests, want %d", c.name, len(reqs), c.wantCalls)
		}
		if c.wantCalls == 2 {
			api := reqs[1]
			if api.Path != "/api/settings/ace0001234" ||
				api.RawQuery != "NumberOfSockets=2&ProcessorId=77&AdditionalInfo=&ProductionRequest=False&ObjectCode=&IncludeLogo=True" ||
				api.Header.Get("Authorization") != basic("test-company") {
				t.Errorf("%s: api request = %s?%s auth=%q", c.name, api.Path, api.RawQuery, api.Header.Get("Authorization"))
			}
		}
	}
	if _, err := CheckIfObjectIDExists(context.Background(), "nope", testUser, "x", 1, 0); err == nil {
		t.Error("invalid site should return an error")
	}
}
