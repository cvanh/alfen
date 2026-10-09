package config

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

// The seeds are synthetic only, so a crasher written to testdata/fuzz can
// never contain data from a real config.

// FuzzParseJSON checks that the config reader never panics, and that both a
// successful and a partial read leave a config that Marshal and the rights
// helpers can use.
func FuzzParseJSON(f *testing.F) {
	small, err := smallConfig().Marshal(EncryptNone, true, "D")
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(small))
	f.Add(minimalJSON("[]", ""))
	f.Add(minimalJSON("["+boJSON("X", "ACME")+"]", `[{"Title":"P","GPRSEnabled":"true","LANEnabled":0,"Values":[{"Key":"k","Value":1}]}]`))
	f.Add([]byte(`{"Type":"ICUConfigFile","Version":"v","Date":"d","Backoffices":"","Users":{},"Firmwares":[],` +
		`"Groups":[{"Name":"g","Comment":"","FeatureRights":[{"ID":"PAGE_LOG","Rights":"None"}]}]}`))
	f.Add([]byte("{\"Type\":\"ICUConfigFile\",\"Version\":1,\"Date\":true,\"Backoffices\":[],\"Users\":[{\"User\":\"u\",\"Password\":\"p\",\"Group\":\"x\",\"Fullname\":\"\t\",\"Company\":\"C\"}],\"Firmwares\":[]}"))
	f.Fuzz(func(t *testing.T, data []byte) {
		c := NewConfig()
		err := c.readJSON(data)
		for _, enc := range []EncryptionType{EncryptNone, EncryptBase64} {
			if _, merr := c.Marshal(enc, true, "D"); merr != nil {
				t.Fatalf("Marshal(%v): %v", enc, merr)
			}
		}
		for _, g := range c.Groups {
			_ = c.Rights(g.Name, PageLog)
			_ = g.JSON(true)
		}
		for _, u := range c.Users {
			_ = VisiblePages(u, false)
			_ = u.JSON()
		}
		if err == nil {
			if _, perr := ParseJSON(data); perr != nil {
				t.Fatalf("readJSON succeeded but ParseJSON failed: %v", perr)
			}
		}
	})
}

// localEntryFixture returns the synthetic hashed config and the ICUUser JSON
// of a local-password entry for "alice" with password synthNewPass.
func localEntryFixture(tb testing.TB) (*Config, string) {
	tb.Helper()
	c, _ := hashedFixture(tb)
	local, err := c.ChangeLocalPassword("alice", synthAlicePass, synthNewPass, synthNewPass, nil)
	if err != nil {
		tb.Fatal(err)
	}
	_, js, ok := strings.Cut(local[0], ";")
	if !ok {
		tb.Fatal("local entry without ';'")
	}
	return c, js
}

// FuzzLogonLocalEntry feeds arbitrary "<hash>;<json>" local-password entries
// (Settings.LocalPasswords is a file the user can edit) to Logon.
func FuzzLogonLocalEntry(f *testing.F) {
	c, js := localEntryFixture(f)
	f.Add(js, synthNewPass)
	f.Add("not json", "x")
	f.Add("{}", "x")
	f.Add("{\"User\":\"u\",\"Password\":\"01:00:x\",\"Fullname\":\"\t\",\"Group\":\"Service\"}", "pw")
	f.Add(`{"User":"u","Password":"FFFF:00:x","Fullname":"","Group":"Service"}`, "pw")
	hash := HashUserName("alice")
	f.Fuzz(func(t *testing.T, entry, password string) {
		u, err := c.Logon("alice", password, []string{hash + ";" + entry})
		if err == nil && u == nil {
			t.Fatal("nil user without an error")
		}
		if u != nil {
			_ = u.GetRights(PageLog)
		}
	})
}

var commentValueRE = regexp.MustCompile(`"Comment":"([^"]{8})`)

// TestLogonLocalEntryControlChars covers the tryLocalPassword fallback for
// raw control characters inside JSON strings, which JavaScriptSerializer
// accepts and encoding/json does not.
func TestLogonLocalEntryControlChars(t *testing.T) {
	c, js := localEntryFixture(t)
	if !commentValueRE.MatchString(js) {
		t.Fatal("fixture entry has no Comment value")
	}
	tests := []struct {
		name   string
		entry  string
		wantOK bool
	}{
		{"as written", js, true},
		{"raw tab in Fullname", strings.Replace(js, `"Fullname":""`, "\"Fullname\":\"\t\"", 1), true},
		{"raw tab after the group name", strings.Replace(js, `"Group":"Service"`, "\"Group\":\"Service\t\"", 1), true},
		// Convert.FromBase64String ignores the white space inside the value.
		{"raw CRLF inside the Comment base64", commentValueRE.ReplaceAllString(js, "\"Comment\":\"${1}\r\n"), true},
		{"raw control char outside a string", "\x01" + js, false},
		{"raw control char in the group name", strings.Replace(js, `"Group":"Service"`, "\"Group\":\"Serv\x01ice\"", 1), false},
	}
	hash := HashUserName("alice")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.entry == js && tc.name != "as written" {
				t.Fatal("test edit did not apply")
			}
			u, err := c.Logon("alice", synthNewPass, []string{hash + ";" + tc.entry})
			if tc.wantOK {
				if err != nil {
					t.Fatalf("Logon: %v", err)
				}
				if u.Group.Name != "Service" || u.Group.HTTPPassword != synthSvcPass || u.Company != synthISAH {
					t.Fatal("group, credentials or ISAH data not recovered")
				}
				return
			}
			// The C# catches the exception, logs it and aborts the logon
			// without the "Incorrect password" warning.
			if err == nil || errors.Is(err, ErrIncorrectPassword) {
				t.Fatalf("want an aborting error, got %v", err)
			}
		})
	}
}
