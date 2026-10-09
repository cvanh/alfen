package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func smallConfig() *Config {
	c := NewConfig()
	c.Version = "1.2.3"
	c.Features = []*Feature{
		NewPageFeature("LOG", "Page Log", RightsFull),
		NewIDFeature(8290, "Max Station Current", RightsFull),
	}
	g := NewGroup(c.Features, "Service", "Line1\r\nLine2", "hu", "hp")
	g.Features[0].Rights = RightsNone
	c.Groups = []*Group{g}
	c.Users = []*User{{User: "alice", Password: "pw", Group: g, Fullname: "Alice A", Company: "ACME"}}
	c.Firmwares = []*Firmware{{Filename: "fw.bin", Version: "1.0.0", Date: "27-2-2019", Comments: "a\nb"}}
	c.BackOffices = []json.RawMessage{json.RawMessage(boJSON("X", ""))}
	return c
}

// boJSON is a complete backoffice element with the keys ICUBackOffice.Json
// writes. new ICUBackOffice(item) needs Title, TitleNL, TitleDE, TitleFR,
// Groups and Values.
func boJSON(title, groups string) string {
	return fmt.Sprintf(`{"Title":%q,"TitleNL":"","TitleDE":"","TitleFR":"","Groups":%q,"Values":[],"ValuesEx":[]}`, title, groups)
}

// TestMarshalLayout pins the exact text WriteInstallerSettings builds with
// string.Format / ValidString.
func TestMarshalLayout(t *testing.T) {
	c := smallConfig()
	want := "{\n\"Type\":\"ICUConfigFile\",\n\"Version\":\"1.2.3\",\n\"Date\":\"D\",\n" +
		"\"Features\":[\n\t{\"Name\":\"Page Log\",\"ID\":\"PAGE_LOG\",\"Type\":\"Page\",\"Default\":\"Full\",\"Comment\":\"\"}," +
		"\n\t{\"Name\":\"Max Station Current\",\"ID\":\"ID_2062\",\"Type\":\"Property\",\"Default\":\"Full\",\"Comment\":\"\"}\n],\n" +
		"\"Groups\":[\n\t{\"Name\":\"Service\",\"Comment\":\"Line1<br>Line2\",\"HTTPUser\":\"hu\",\"HTTPPassword\":\"hp\",\"FeatureRights\":[\n\t\t{\"ID\":\"PAGE_LOG\",\"Rights\":\"None\"}]}\n],\n" +
		"\"Users\":[\n\t{\"User\":\"alice\",\"Password\":\"pw\",\"Group\":\"Service\",\"Fullname\":\"Alice A\",\"Comment\":\"\",\"Company\":\"ACME\"}\n],\n" +
		"\"Backoffices\":[\n\t" + boJSON("X", "") + "\n],\n" +
		"\"PMBackOffices\":[\n],\n" +
		"\"Firmwares\":[\n\t{\"Filename\":\"fw.bin\",\"Version\":\"1.0.0\",\"Date\":\"27-2-2019\",\"Comments\":\"a<br>b\"}\n]}"
	got, err := c.Marshal(EncryptNone, true, "D")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("layout mismatch at %d\ngot:  %q\nwant: %q", firstDiff([]byte(got), []byte(want)), got, want)
	}
	// addBackoffices=false empties both backoffice arrays.
	got, _ = c.Marshal(EncryptNone, false, "D")
	if !strings.Contains(got, "\"Backoffices\":[\n],\n\"PMBackOffices\":[\n],") {
		t.Fatal("addBackoffices=false still wrote backoffices")
	}
	// EncryptRijndael keeps the group credentials (only Hashed blanks them).
	enc, _ := c.Marshal(EncryptRijndael, true, "D")
	plain, err := Decrypt(enc, ConfigPassPhrase)
	if err != nil || !strings.Contains(plain, "\"HTTPUser\":\"hu\"") {
		t.Fatalf("Rijndael envelope: %v", err)
	}
}

func TestParseRoundTripSmall(t *testing.T) {
	text, _ := smallConfig().Marshal(EncryptBase64, true, "D")
	c, err := Parse(text, EncryptBase64)
	if err != nil {
		t.Fatal(err)
	}
	if c.Groups[0].Comment != "Line1\nLine2" || c.Firmwares[0].Comments != "a\nb" {
		t.Fatalf("<br> not decoded: %q %q", c.Groups[0].Comment, c.Firmwares[0].Comments)
	}
	if c.Rights("Service", "PAGE_LOG") != RightsNone || c.Rights("Service", "ID_2062") != RightsFull {
		t.Fatal("feature rights not restored")
	}
	if c.FindFeature("BO_X") == nil {
		t.Fatal("BO_X feature not derived from backoffice title")
	}
	if c.Groups[0].NumberOfUsers != 1 {
		t.Fatalf("NumberOfUsers = %d", c.Groups[0].NumberOfUsers)
	}
}

func TestParseRequiredKeys(t *testing.T) {
	base := map[string]string{
		"Type": `"ICUConfigFile"`, "Version": `"v"`, "Date": `"d"`,
		"Backoffices": `[]`, "Users": `[]`, "Firmwares": `[]`,
	}
	build := func(drop string, override map[string]string) []byte {
		parts := []string{}
		for k, v := range base {
			if k == drop {
				continue
			}
			if o, ok := override[k]; ok {
				v = o
			}
			parts = append(parts, `"`+k+`":`+v)
		}
		for k, v := range override {
			if _, ok := base[k]; !ok {
				parts = append(parts, `"`+k+`":`+v)
			}
		}
		return []byte("{" + strings.Join(parts, ",") + "}")
	}
	if _, err := ParseJSON(build("", nil)); err != nil {
		t.Fatalf("minimal config: %v", err)
	}
	for _, key := range []string{"Type", "Version", "Date", "Backoffices", "Users", "Firmwares"} {
		if _, err := ParseJSON(build(key, nil)); err == nil {
			t.Errorf("missing %s accepted", key)
		}
	}
	if _, err := ParseJSON(build("", map[string]string{"Type": `"Other"`})); !errors.Is(err, ErrIncorrectFileType) {
		t.Errorf("wrong Type: %v", err)
	}
	if c, err := ParseJSON(build("", map[string]string{"Version": `2`})); err != nil || c.Version != "2" {
		t.Errorf("numeric Version: %v", err)
	}
	if _, err := ParseJSON(build("", map[string]string{"Users": `null`})); err == nil {
		t.Error("null Users accepted")
	}
	bad := map[string]string{
		"feature without Comment":      `{"Features":[{"Name":"n","ID":"X"}]}`,
		"feature with bad Default":     `{"Features":[{"Name":"n","ID":"X","Default":"Maybe","Comment":""}]}`,
		"user without Fullname":        `{"Users":[{"User":"u","Password":"p","Group":"g"}]}`,
		"firmware without Date":        `{"Firmwares":[{"Filename":"f","Version":"v","Comments":""}]}`,
		"group without Comment":        `{"Groups":[{"Name":"g"}]}`,
		"right with bad value":         `{"Groups":[{"Name":"g","Comment":"","FeatureRights":[{"ID":"PAGE_LOG","Rights":"All"}]}]}`,
		"backoffice without Title key": `{"Backoffices":[{"Groups":""}]}`,
		"backoffice without TitleNL":   `{"Backoffices":[{"Title":"X","TitleDE":"","TitleFR":"","Groups":"","Values":[]}]}`,
		"backoffice without Values":    `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":""}]}`,
		"backoffice not an object":     `{"Backoffices":["X"]}`,
		"Backoffices not enumerable":   `{"Backoffices":5}`,
		"Values entry without Key":     `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[{"Value":1}]}]}`,
		"Values entry without Value":   `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[{"Key":"k"}]}]}`,
		"Values entry numeric Key":     `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[{"Key":1,"Value":1}]}]}`,
		"Values entry not an object":   `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[["Key","Value"]]}]}`,
		"Values non-empty string":      `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":"ab"}]}`,
		"Values null":                  `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":null}]}`,
		"Values non-empty object":      `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":{"k":1}}]}`,
		"ValuesEx entry without Key":   `{"Backoffices":[{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[],"ValuesEx":[{}]}]}`,
		"PM backoffice without Title":  `{"PMBackOffices":[{"Values":[]}]}`,
		"PM backoffice without Values": `{"PMBackOffices":[{"Title":"X"}]}`,
		"PM GPRSEnabled not a boolean": `{"PMBackOffices":[{"Title":"X","GPRSEnabled":"yes","Values":[]}]}`,
		"PM GPRSEnabled string 1":      `{"PMBackOffices":[{"Title":"X","GPRSEnabled":"1","Values":[]}]}`,
		"PM LANEnabled array":          `{"PMBackOffices":[{"Title":"X","LANEnabled":[],"Values":[]}]}`,
		"PM Values entry without Key":  `{"PMBackOffices":[{"Title":"X","Values":[{"Value":1}]}]}`,
		"PMBackOffices null":           `{"PMBackOffices":null}`,
	}
	for name, js := range bad {
		var o map[string]json.RawMessage
		_ = json.Unmarshal([]byte(js), &o)
		ov := map[string]string{}
		for k, v := range o {
			ov[k] = string(v)
		}
		if _, err := ParseJSON(build("", ov)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	// A rights value for an unknown feature ID is never parsed (C# only
	// parses Rights for the matched feature).
	if _, err := ParseJSON(build("", map[string]string{"Groups": `[{"Name":"g","Comment":"","FeatureRights":[{"ID":"NOPE","Rights":"All"}]}]`})); err != nil {
		t.Errorf("unknown feature id rejected: %v", err)
	}
}

func TestParseDefaultsAndQuirks(t *testing.T) {
	js := `{"Type":"ICUConfigFile","Version":"v","Date":"d",
"Features":[{"Name":" Page Log ","ID":" PAGE_LOG ","Comment":" c<br>d "},{"Name":"old bo","ID":"BO_OLD","Type":"Backoffice","Default":"None","Comment":""}],
"Backoffices":[` + boJSON("Fleet One", "ACME;OTHER") + `,` + boJSON("Other", "XYZ") + `,` + boJSON("", "ACME") + `],
"Users":[{"User":"a","Password":"p","Group":"Nope","Fullname":"","Company":"ACME"},
         {"User":"b","Password":"p","Group":"Nope","Fullname":"","Company":"ACME"},
         {"User":"c","Password":"p","Group":"Service","Fullname":"","Company":""},
         {"User":"d","Password":"p","Group":"Nope","Fullname":"","Company":"ZZZ"},
         {"User":"e","Password":"p","Group":"Admin","Fullname":""}],
"Firmwares":[]}`
	c, err := ParseJSON([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	f := c.FindFeature("PAGE_LOG")
	if f == nil || f.Name != "Page Log" || f.Comment != "c\nd" || f.Type != FeatureTypeNormal || f.Default != RightsFull {
		t.Fatalf("feature defaults: %+v", f)
	}
	if c.FindFeature("BO_OLD") != nil {
		t.Fatal("Backoffice-type feature from JSON should be replaced by AddBackofficesToFeatures")
	}
	if c.FindFeature("BO_FLEET_ONE") == nil || c.FindFeature("BO_OTHER") == nil {
		t.Fatal("missing derived BO features")
	}
	var names []string
	for _, g := range c.Groups {
		names = append(names, g.Name)
	}
	if got := strings.Join(names, ","); got != "Admin,Production,Service,Customer,Extern_ACME,Extern_ZZZ" {
		t.Fatalf("default groups = %s", got)
	}
	if c.Rights("Admin", "PAGE_PRODUCTION") != RightsFull || c.Rights("Service", "PAGE_PRODUCTION") != RightsNone {
		t.Fatal("Admin default group must start Full, others from feature defaults")
	}
	if c.Rights("Extern_ACME", "BO_FLEET_ONE") != RightsReadOnly || c.Rights("Extern_ACME", "BO_OTHER") != RightsNone {
		t.Fatal("Extern_ backoffice rights not derived from backoffice Groups")
	}
	if c.Rights("Service", "BO_OTHER") != RightsReadOnly {
		t.Fatal("non-extern group should keep the BO default (ReadOnly)")
	}
	want := map[string]string{"a": "Extern_ACME", "b": "Extern_ACME", "c": "Service", "d": "Extern_ZZZ", "e": "Admin"}
	for _, u := range c.Users {
		if u.Group == nil || u.Group.Name != want[u.User] {
			t.Errorf("user %s resolved to %v", u.User, u.Group)
		}
	}
	if c.FindUser("E").Company != "Admin" {
		t.Error("missing Company must default to the group name")
	}
	if c.Rights("Nope", "PAGE_LOG") != RightsReadOnly {
		t.Error("unknown group should behave like a user without group")
	}

	// A JSON-loaded "Admin" group starts from feature defaults (the C#
	// assigns Name after InitializeFeatures).
	js2 := strings.Replace(js, `"Firmwares":[]`, `"Firmwares":[],"Groups":[{"Name":"Admin","Comment":""}]`, 1)
	c2, err := ParseJSON([]byte(js2))
	if err != nil {
		t.Fatal(err)
	}
	if c2.Rights("Admin", "PAGE_PRODUCTION") != RightsNone {
		t.Fatal("JSON Admin group should not be forced to Full")
	}
	if g := c2.FindUser("a").Group; g != nil {
		t.Fatalf("no Extern_* group exists, user should have none, got %s", g.Name)
	}
}

func TestParseToleratesRawControlChars(t *testing.T) {
	js := "{\"Type\":\"ICUConfigFile\",\"Version\":\"v\",\"Date\":\"d\",\"Backoffices\":[],\"Users\":[],\"Firmwares\":[{\"Filename\":\"f\",\"Version\":\"1\",\"Date\":\"x\",\"Comments\":\"a\tb\"}]}"
	c, err := ParseJSON([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	if c.Firmwares[0].Comments != "a\tb" {
		t.Fatalf("Comments = %q", c.Firmwares[0].Comments)
	}
}

func TestReadAllTextBOM(t *testing.T) {
	if readAllText([]byte("\xEF\xBB\xBFabc")) != "abc" {
		t.Error("UTF-8 BOM")
	}
	if readAllText([]byte{0xFF, 0xFE, 'a', 0, 'b', 0}) != "ab" {
		t.Error("UTF-16LE BOM")
	}
	if readAllText([]byte{0xFE, 0xFF, 0, 'a', 0, 'b'}) != "ab" {
		t.Error("UTF-16BE BOM")
	}
}

func TestLookups(t *testing.T) {
	c := smallConfig()
	for in, ok := range map[string]bool{"fw.bin": true, `C:\fw\FW.BIN`: true, "/tmp/Fw.Bin": true, "fw.bin.old": false} {
		if (c.FindFirmware(in) != nil) != ok {
			t.Errorf("FindFirmware(%q)", in)
		}
	}
	if c.FindUser("ALICE") == nil || c.FindUser("bob") != nil {
		t.Error("FindUser")
	}
	if c.FindGroup("service") != nil || c.FindGroup("Service") == nil {
		t.Error("FindGroup is exact")
	}
	g := c.AddGroup()
	if g.Name != "" || len(g.Features) != len(c.Features) || g.GetRights("PAGE_LOG") != RightsFull {
		t.Error("AddGroup")
	}
	u := c.AddUser()
	if len(u.Password) != PasswordLength || u.Group != nil || u.GetRights("PAGE_LOG") != RightsReadOnly {
		t.Errorf("AddUser: %q", u.Password)
	}
	f := c.AddFirmware()
	if f.Version != "0.0.0" {
		t.Error("AddFirmware")
	}
	c.RemoveGroup(g)
	c.RemoveUser(u)
	c.RemoveFirmware(f)
	if len(c.Groups) != 1 || len(c.Users) != 1 || len(c.Firmwares) != 1 {
		t.Error("Remove*")
	}
}

func TestCreatePassword(t *testing.T) {
	for i := 0; i < 200; i++ {
		p := CreatePassword(PasswordLength)
		digits := 0
		for _, ch := range p {
			switch {
			case strings.ContainsRune(passwordDigit, ch):
				digits++
			case !strings.ContainsRune(passwordChars, ch):
				t.Fatalf("unexpected char in %q", p)
			}
		}
		if len(p) != PasswordLength || digits != 1 {
			t.Fatalf("CreatePassword = %q", p)
		}
	}
}

func TestSaveWritesBackup(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "InstallerSettings.dat")
	c := smallConfig()
	if err := c.Save(fn, EncryptRijndael, true); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(fn)
	if _, err := os.Stat(filepath.Join(dir, "InstallerSettings.bak")); !os.IsNotExist(err) {
		t.Fatal("no .bak expected on first save")
	}
	c.Version = "2.0.0"
	if err := c.Save(fn, EncryptRijndael, true); err != nil {
		t.Fatal(err)
	}
	bak, err := os.ReadFile(filepath.Join(dir, "InstallerSettings.bak"))
	if err != nil || string(bak) != string(first) {
		t.Fatalf(".bak should hold the previous file: %v", err)
	}
	back, err := Load(fn, EncryptRijndael)
	if err != nil || back.Version != "2.0.0" {
		t.Fatalf("Load after Save: %v", err)
	}
}

// minimalJSON is a loadable config with the given Backoffices / PMBackOffices
// array texts.
func minimalJSON(backoffices, pmBackoffices string) []byte {
	js := `{"Type":"ICUConfigFile","Version":"v","Date":"d","Backoffices":` + backoffices
	if pmBackoffices != "" {
		js += `,"PMBackOffices":` + pmBackoffices
	}
	return []byte(js + `,"Users":[],"Firmwares":[]}`)
}

// TestBackOfficeConstructorAccepts covers values new ICUBackOffice(item) /
// new ICUPMBackOffice(item) accept, and the Title they produce (Convert.ToString
// + Trim, overwritten by Values/ValuesEx entries keyed "Title").
func TestBackOfficeConstructorAccepts(t *testing.T) {
	const rest = `"TitleNL":"","TitleDE":"","TitleFR":"","Groups":""`
	tests := []struct {
		name    string
		bo      string
		wantBOs []string // derived BO_ feature IDs
	}{
		{"complete", boJSON(" Fleet One ", "ACME"), []string{"BO_FLEET_ONE"}},
		{"no ValuesEx", `{"Title":"X",` + rest + `,"Values":[]}`, []string{"BO_X"}},
		{"array Title", `{"Title":[1],` + rest + `,"Values":[]}`, []string{BackOfficeFeatureID(csArrayTypeName)}},
		{"object Title", `{"Title":{"a":1},` + rest + `,"Values":[]}`, []string{BackOfficeFeatureID(csDictTypeName)}},
		{"numeric Title", `{"Title":42,` + rest + `,"Values":[]}`, []string{"BO_42"}},
		{"boolean Title", `{"Title":true,` + rest + `,"Values":[]}`, []string{"BO_TRUE"}},
		{"null Title", `{"Title":null,` + rest + `,"Values":[]}`, nil},
		{"array Groups", `{"Title":"X","TitleNL":"","TitleDE":"","TitleFR":"","Groups":["A"],"Values":[]}`, []string{"BO_X"}},
		{"empty object Values", `{"Title":"X",` + rest + `,"Values":{}}`, []string{"BO_X"}},
		{"empty string Values", `{"Title":"X",` + rest + `,"Values":""}`, []string{"BO_X"}},
		{"null Key", `{"Title":"X",` + rest + `,"Values":[{"Key":null,"Value":1}]}`, []string{"BO_X"}},
		{"Values overrides Title", `{"Title":"X",` + rest + `,"Values":[{"Key":"Title","Value":" Y "}]}`, []string{"BO_Y"}},
		{"ValuesEx overrides last", `{"Title":"X",` + rest + `,"Values":[{"Key":"Title","Value":"Y"}],"ValuesEx":[{"Key":"Title","Value":"Z"}]}`, []string{"BO_Z"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := ParseJSON(minimalJSON("["+tc.bo+"]", ""))
			if err != nil {
				t.Fatalf("rejected: %v", err)
			}
			var got []string
			for _, f := range c.Features {
				if f.Type == FeatureTypeBackoffice {
					got = append(got, f.ID)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.wantBOs, ",") {
				t.Fatalf("BO features = %v, want %v", got, tc.wantBOs)
			}
		})
	}
	pms := []struct{ name, pm string }{
		{"complete", `{"Title":"X","GPRSEnabled":true,"LANEnabled":false,"Values":[{"Key":"k","Value":"v"}]}`},
		{"flags absent", `{"Title":"X","Values":[]}`},
		{"flags as numbers", `{"Title":"X","GPRSEnabled":0,"LANEnabled":2.5,"Values":[]}`},
		{"flags as strings", `{"Title":"X","GPRSEnabled":" TRUE\u0000","LANEnabled":"false","Values":[]}`},
		{"null flag and Title", `{"Title":null,"GPRSEnabled":null,"Values":[]}`},
	}
	for _, tc := range pms {
		if _, err := ParseJSON(minimalJSON("[]", "["+tc.pm+"]")); err != nil {
			t.Errorf("PM %s rejected: %v", tc.name, err)
		}
	}
	// Empty string / object enumerate nothing, also for the top-level arrays.
	if c, err := ParseJSON(minimalJSON(`""`, `{}`)); err != nil || len(c.BackOffices)+len(c.PMBackOffices) != 0 {
		t.Errorf("empty enumerables: %v", err)
	}
}

func TestConvertToBoolean(t *testing.T) {
	tests := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{`true`, true, false},
		{`false`, false, false},
		{`null`, false, false},
		{`0`, false, false},
		{`-3`, true, false},
		{`0.0`, false, false},
		{`"True"`, true, false},
		{`"fAlSe"`, false, false},
		{`" true \u0000"`, true, false},
		{`"1"`, false, true},
		{`"yes"`, false, true},
		{`""`, false, true},
		{`[]`, false, true},
		{`{}`, false, true},
	}
	for _, tc := range tests {
		got, err := convertToBoolean(json.RawMessage(tc.in))
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("convertToBoolean(%s) = %v, %v", tc.in, got, err)
		}
	}
}

// TestLoadRejectsIncompleteBackoffice: ReadInstallerSettings fails when a
// backoffice lacks a key LoadFromDynamic indexes unconditionally.
func TestLoadRejectsIncompleteBackoffice(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "cfg.dat")
	good := minimalJSON("["+boJSON("X", "")+"]", "")
	bad := minimalJSON(`[{"Title":"X","TitleDE":"","TitleFR":"","Groups":"","Values":[]}]`, "")
	if err := os.WriteFile(fn, good, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(fn, EncryptNone); err != nil {
		t.Fatalf("complete backoffice: %v", err)
	}
	if err := os.WriteFile(fn, bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if c, err := Load(fn, EncryptNone); err == nil || c != nil || !strings.Contains(err.Error(), "TitleNL") {
		t.Fatalf("backoffice without TitleNL: %v", err)
	}
}

// TestModelSetters pins the normalisation of the C# property setters.
func TestModelSetters(t *testing.T) {
	f := &Feature{}
	f.SetID("  page_log ")
	f.SetName(" n ")
	f.SetComment(" c ")
	g := &Group{}
	g.SetName(" g ")
	g.SetComment(" c ")
	g.SetHTTPUser(" u ")
	g.SetHTTPPassword(" p ")
	u := &User{}
	u.SetUser(" a ")
	u.SetPassword(" b ")
	u.SetFullname(" c ")
	u.SetComment(" d ")
	u.SetCompany(" e ")
	fw := &Firmware{}
	fw.SetFilename(" f.bin ")
	fw.SetVersion(" 1.0 ")
	tests := []struct{ name, got, want string }{
		{"Feature.ID", f.ID, "PAGE_LOG"},
		{"Feature.Name", f.Name, "n"},
		{"Feature.Comment", f.Comment, "c"},
		{"Group.Name", g.Name, "g"},
		{"Group.Comment", g.Comment, "c"},
		{"Group.HTTPUser", g.HTTPUser, "u"},
		{"Group.HTTPPassword", g.HTTPPassword, "p"},
		{"User.User", u.User, "a"},
		{"User.Password", u.Password, "b"},
		{"User.Fullname", u.Fullname, "c"},
		{"User.Comment", u.Comment, "d"},
		{"User.Company", u.Company, "e"},
		{"Firmware.Filename", fw.Filename, "f.bin"},
		{"Firmware.Version", fw.Version, "1.0"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
	// The upper-cased ID is what GetRights matches.
	f.Default = RightsNone
	if NewGroup([]*Feature{f}, "Service", "", "", "").GetRights(PageLog) != RightsNone {
		t.Error("SetID must make the feature match PAGE_LOG")
	}
}
