package presets

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"alfen/aceclient/internal/api"
)

// vectorDevice / vectorProps reproduce the inputs of the .NET reference
// document (see dotnet_vectors_test.go).
var vectorDevice = DeviceInfo{
	ModelType:       ModelTwin_4_0,
	Model:           "Twin 4.0",
	NumberOfSockets: 2,
	IPAddress:       "192.168.1.20",
	Port:            443,
	HostName:        "twin-4-0-ng910-60023 & <x>",
}

var vectorNow = time.Date(2024, 5, 1, 12, 34, 56, 123450000, time.FixedZone("CEST", 2*3600))

func vectorDictionary() *Dictionary {
	return NewDictionary(
		[]DataSheetParameter{{ID: 0x3000, DataType: api.SDTUnsigned8, Name: "edsOnlyNeverRead"}},
		[]api.Property{
			{ID: 0x2129, Sub: 0, DataType: api.SDTReal32, Value: "16.1"},
			{ID: 0x2129, Sub: 1, DataType: api.SDTReal32, Value: "1e-5"},
			{ID: 0x2051, Sub: 0, DataType: api.SDTVisibleString, Value: "forbidden 8273"},
			{ID: 0x2060, Sub: 0, DataType: api.SDTVisibleString, Value: "a&b<c>d\"e'f\tg\nh\ri\r\nj"},
			{ID: 0x2061, Sub: 0, DataType: api.SDTVisibleString, Value: "héllo € 😀"},
			{ID: 0x2062, Sub: 0, DataType: api.SDTUnsigned8, Value: "5"},
			{ID: 0x2070, Sub: 0, DataType: api.SDTUnsigned8, Value: "1", ReadOnly: true},
			{ID: 0x2063, Sub: 0, DataType: api.SDTInteger8, Value: "-5"},
			{ID: 0x2064, Sub: 0, DataType: api.SDTInteger16, Value: "-300"},
			{ID: 0x2065, Sub: 0, DataType: api.SDTUnsigned32, Value: "4000000000"},
			{ID: 0x2071, Sub: 0, DataType: api.SDTInteger24, Value: "12"},
			{ID: 0x2066, Sub: 0, DataType: api.SDTInteger64, Value: "-9000000000"},
			{ID: 0x2067, Sub: 0, DataType: api.SDTUnsigned64, Value: "18446744073709551615"},
			{ID: 0x2068, Sub: 0, DataType: api.SDTBoolean, Value: "true"},
			{ID: 0x2068, Sub: 1, DataType: api.SDTBoolean, Value: "False"},
			{ID: 0x2069, Sub: 0, DataType: api.SDTBoolean, Value: "yes"},
			{ID: 0x2072, Sub: 0, DataType: api.SDTLowLimit, Value: "x"},
			{ID: 0x206A, Sub: 0, DataType: api.SDTByteArray, Value: "a,ff,0"},
			{ID: 0x206B, Sub: 0, DataType: api.SDTArray16, Value: "1234,a"},
			{ID: 0x206C, Sub: 0, DataType: api.SDTVisibleString, Value: ""},
		})
}

func mustB64(t *testing.T, s string) string {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildSettingsXMLMatchesDotNet(t *testing.T) {
	got, err := BuildSettingsXML(vectorDevice, vectorDictionary(), vectorNow)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustB64(t, dotnetSettingsDocB64); got != want {
		t.Errorf("document mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestBuildSettingsXMLEmptyProperties(t *testing.T) {
	got, err := BuildSettingsXML(DeviceInfo{Model: "Unknown"}, NewDictionary(nil, nil), vectorNow)
	if err != nil {
		t.Fatal(err)
	}
	// Same element shapes as the .NET <Information></Information> / <Properties /> vector.
	ref := mustB64(t, dotnetEmptyPropsDocB64)
	for _, frag := range []string{"  <Information></Information>\r\n", "  <Properties />\r\n</Settings>", "<Identity />", "<IPAddress />", "<HostName />", "<Port>0</Port>"} {
		if !strings.Contains(got, frag) {
			t.Errorf("missing %q in %q", frag, got)
		}
	}
	if !strings.HasSuffix(ref, "  <Properties />\r\n</Settings>") {
		t.Errorf("reference vector changed: %q", ref)
	}
}

func TestBuildSettingsXMLInvalidCharacter(t *testing.T) {
	d := NewDictionary(nil, []api.Property{{ID: 0x2060, DataType: api.SDTVisibleString, Value: "bad\x01"}})
	if _, err := BuildSettingsXML(vectorDevice, d, vectorNow); err == nil || !strings.Contains(err.Error(), "0x01") {
		t.Errorf("err = %v", err)
	}
}

func TestXMLDateTimeLocal(t *testing.T) {
	cest := time.FixedZone("CEST", 2*3600)
	cet := time.FixedZone("CET", 3600)
	in := []time.Time{
		time.Date(2024, 5, 1, 12, 34, 56, 123450000, cest),
		time.Date(2024, 1, 15, 8, 0, 0, 0, cet),
		time.Date(2024, 1, 15, 8, 0, 0, 100000000, cet),
		time.Date(2024, 1, 15, 8, 0, 0, 100, cet),
	}
	for i, tm := range in {
		if got := xmlDateTimeLocal(tm); got != dotnetDates[i] {
			t.Errorf("date %d = %q, want %q", i, got, dotnetDates[i])
		}
	}
	if got := xmlDateTimeLocal(time.Date(2024, 1, 2, 3, 4, 5, 99, time.UTC)); got != "2024-01-02T03:04:05+00:00" {
		t.Errorf("UTC local = %q (sub-tick nanoseconds are dropped, offset never Z)", got)
	}
	if got := xmlDateTimeLocal(time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("", -(5*3600+1800)))); got != "2024-01-02T03:04:05-05:30" {
		t.Errorf("negative offset = %q", got)
	}
}

func TestSavePropertiesMatchesDotNet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ACE0001.exml")
	if err := SaveProperties(path, vectorDevice, vectorDictionary(), vectorNow); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != dotnetSettingsDocEnc {
		t.Errorf("encrypted file differs from .NET output")
	}
}

// loadDictionary is the "device" the .NET vector file is loaded into.
func loadDictionary() *Dictionary {
	return NewDictionary(nil, []api.Property{
		{ID: 0x2129, Sub: 0, Name: "maxCurrent", DataType: api.SDTReal32, Value: "16"},            // -> 16.1, changed
		{ID: 0x2129, Sub: 1, DataType: api.SDTReal32, Value: "0.00001"},                           // same
		{ID: 0x2060, Sub: 0, DataType: api.SDTVisibleString, Value: "a&b<c>d\"e'f\tg\nh\ri\r\nj"}, // same after unescape
		{ID: 0x2062, Sub: 0, Name: "u8", DataType: api.SDTUnsigned8, Value: "6"},                  // -> 5, changed
		{ID: 0x2067, Sub: 0, DataType: api.SDTUnsigned64, Value: "18446744073709551615"},          // same
		{ID: 0x2068, Sub: 0, DataType: api.SDTBoolean, Value: "true"},                             // same
		{ID: 0x2069, Sub: 0, DataType: api.SDTBoolean, Value: "x"},                                // (byte)0 == (byte)0
		{ID: 0x206A, Sub: 0, Name: "bytes", DataType: api.SDTByteArray, Value: "0A,FF,00"},        // array: always changed
		{ID: 0x206B, Sub: 0, Name: "words", DataType: api.SDTArray16, Value: "1234,000A"},         // array: always changed
		{ID: 0x206C, Sub: 0, DataType: api.SDTVisibleString, Value: ""},                           // same
	})
}

func TestLoadPropertiesFromDotNetFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "saved.exml")
	if err := os.WriteFile(path, []byte(dotnetSettingsDocEnc), 0o600); err != nil {
		t.Fatal(err)
	}
	d := loadDictionary()
	asked := false
	n, err := LoadProperties(path, vectorDevice, d, func(string, string) bool { asked = true; return false })
	if err != nil {
		t.Fatal(err)
	}
	if asked {
		t.Error("same model must not ask")
	}
	if n != 4 {
		t.Errorf("changed = %d, want 4", n)
	}
	var got []string
	for _, p := range d.ChangedProperties() {
		got = append(got, p.IDSub()+"="+p.Value)
	}
	want := "2129_0=16.1 2062_0=5 206A_0=0A,FF,00 206B_0=1234,000A"
	if strings.Join(got, " ") != want {
		t.Errorf("writes = %q, want %q", strings.Join(got, " "), want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rt.exml")
	src := vectorDictionary()
	src.SetValue(0x2062, 0, "42") // unsaved edits are saved too (current Value)
	if err := SaveProperties(path, vectorDevice, src, vectorNow); err != nil {
		t.Fatal(err)
	}
	dst := vectorDictionary()
	n, err := LoadProperties(path, vectorDevice, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	// 2062 (42) + the two arrays
	if n != 3 {
		t.Errorf("changed = %d, want 3", n)
	}
	if p, _ := dst.Property(0x2062, 0); p.Value != "42" {
		t.Errorf("2062 = %q", p.Value)
	}
	if dst.IsChanged(0x2051, 0) {
		t.Error("forbidden property must not be loaded")
	}
}

const settingsHead = `<Settings><XMLVersion>1.0</XMLVersion><Model>Twin 4.0</Model>`

func TestLoadPropertiesTextChecks(t *testing.T) {
	newDict := func() *Dictionary {
		return NewDictionary(nil, []api.Property{
			{ID: 0x2129, DataType: api.SDTReal32, Value: "16"},
			{ID: 0x2051, DataType: api.SDTVisibleString, Value: "obj"}, // 8273: forbidden
			{ID: 0x2070, DataType: api.SDTUnsigned8, Value: "1", ReadOnly: true},
		})
	}
	tests := []struct {
		name    string
		xml     string
		n       int
		err     error
		anyErr  bool
		changed []PropKey
	}{
		{name: "no settings root", xml: `<Other/>`},
		{name: "wrong version", xml: `<Settings><XMLVersion>2.0</XMLVersion></Settings>`, err: ErrIncorrectVersion},
		{name: "missing version", xml: `<Settings/>`, anyErr: true},
		{name: "missing model", xml: `<Settings><XMLVersion>1.0</XMLVersion></Settings>`, anyErr: true},
		{name: "missing properties", xml: settingsHead + `</Settings>`, anyErr: true},
		{name: "apply", n: 1, changed: []PropKey{{0x2129, 0}},
			xml: settingsHead + `<Properties><Property Id="2129_00" Value="20"/><Property Id="9999_0" Value="1"/><Property Id="2129" Value="1"/></Properties></Settings>`},
		{name: "forbidden skipped",
			xml: settingsHead + `<Properties><Property Id="2051_0" Value="new"/></Properties></Settings>`},
		{name: "read-only applied", n: 1, changed: []PropKey{{0x2070, 0}},
			xml: settingsHead + `<Properties><Property Id="2070_0" Value="2"/></Properties></Settings>`},
		{name: "same value not counted",
			xml: settingsHead + `<Properties><Property Id="2129_0" Value="16.0"/></Properties></Settings>`},
		{name: "id overflow keeps earlier assignments", anyErr: true, changed: []PropKey{{0x2129, 0}},
			xml: settingsHead + `<Properties><Property Id="2129_0" Value="20"/><Property Id="10000_0" Value="1"/></Properties></Settings>`},
		{name: "missing value", anyErr: true,
			xml: settingsHead + `<Properties><Property Id="2129_0"/></Properties></Settings>`},
		{name: "missing id", anyErr: true,
			xml: settingsHead + `<Properties><Property Value="1"/></Properties></Settings>`},
		{name: "malformed", anyErr: true, xml: `<Settings>`},
		{name: "namespaced root ignored", xml: `<Settings xmlns="urn:x"><XMLVersion>2.0</XMLVersion></Settings>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newDict()
			n, err := LoadPropertiesText(tt.xml, vectorDevice, d, nil)
			if tt.err != nil && !errors.Is(err, tt.err) {
				t.Errorf("err = %v, want %v", err, tt.err)
			}
			if tt.err == nil && (err != nil) != tt.anyErr {
				t.Errorf("err = %v", err)
			}
			if n != tt.n {
				t.Errorf("n = %d, want %d", n, tt.n)
			}
			var changed []PropKey
			for _, p := range d.ChangedProperties() {
				changed = append(changed, PropKey{p.ID, p.Sub})
			}
			if len(changed) != len(tt.changed) {
				t.Fatalf("changed = %v, want %v", changed, tt.changed)
			}
			for i := range changed {
				if changed[i] != tt.changed[i] {
					t.Errorf("changed = %v, want %v", changed, tt.changed)
				}
			}
		})
	}
}

func TestLoadPropertiesAlreadyChangedNotCounted(t *testing.T) {
	d := NewDictionary(nil, []api.Property{{ID: 0x2129, DataType: api.SDTReal32, Value: "16"}})
	d.SetValue(0x2129, 0, "10")
	n, err := LoadPropertiesText(settingsHead+`<Properties><Property Id="2129_0" Value="20"/></Properties></Settings>`, vectorDevice, d, nil)
	if err != nil || n != 0 || !d.IsChanged(0x2129, 0) {
		t.Errorf("n=%d err=%v changed=%v", n, err, d.IsChanged(0x2129, 0))
	}
	// assigning the device value back clears IsChanged (the user's edit is lost)
	n, _ = LoadPropertiesText(settingsHead+`<Properties><Property Id="2129_0" Value="16"/></Properties></Settings>`, vectorDevice, d, nil)
	if n != 0 || d.IsChanged(0x2129, 0) {
		t.Errorf("n=%d changed=%v", n, d.IsChanged(0x2129, 0))
	}
}

func TestLoadPropertiesModelMismatch(t *testing.T) {
	xml := `<Settings><XMLVersion>1.0</XMLVersion><Model>ICU Eve Mini</Model><Properties><Property Id="2129_0" Value="20"/></Properties></Settings>`
	// An Eve Mini with fixed cable saves "ICU Eve Mini", which From() maps
	// back to Eve_Mini: the C# asks even for its own file.
	dev := NewDeviceInfo("eve-mini-x", 1, 0, "ACE1", "10.0.0.2", 443)
	if dev.ModelType != ModelEve_Mini_FC || dev.Model != "ICU Eve Mini" {
		t.Fatalf("dev = %+v", dev)
	}
	for _, answer := range []bool{false, true} {
		d := NewDictionary(nil, []api.Property{{ID: 0x2129, DataType: api.SDTReal32, Value: "16"}})
		var q, detail string
		n, err := LoadPropertiesText(xml, dev, d, func(a, b string) bool { q, detail = a, b; return answer })
		if err != nil {
			t.Fatal(err)
		}
		if q != "Warning! The setting you are trying to load are for a 'ICU Eve Mini', you currently have selected a 'ICU Eve Mini'." ||
			detail != "Are you sure you want to load this settings file?" {
			t.Errorf("question = %q / %q", q, detail)
		}
		if want := map[bool]int{false: 0, true: 1}[answer]; n != want {
			t.Errorf("answer %v: n = %d, want %d", answer, n, want)
		}
	}
	// nil confirm counts as "No"
	d := NewDictionary(nil, []api.Property{{ID: 0x2129, DataType: api.SDTReal32, Value: "16"}})
	if n, _ := LoadPropertiesText(xml, dev, d, nil); n != 0 || d.IsChanged(0x2129, 0) {
		t.Error("nil confirm must not apply")
	}
}

func TestReadSettingsTextExtensions(t *testing.T) {
	dir := t.TempDir()
	plain := settingsHead + `<Properties><Property Id="2129_0" Value="20"/></Properties></Settings>`
	files := map[string]string{
		"plain.xml":  "\xEF\xBB\xBF" + plain, // UTF-8 BOM stripped by ReadAllText
		"enc.exml":   encryptString(plain, settingsPassPhrase),
		"upper.EXML": encryptString(plain, settingsPassPhrase), // not decrypted: case-sensitive compare
		"wrapped.exml": func() string {
			e := encryptString(plain, settingsPassPhrase)
			return e[:10] + "\r\n " + e[10:] + "\n"
		}(),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for name, wantOK := range map[string]bool{"plain.xml": true, "enc.exml": true, "upper.EXML": false, "wrapped.exml": true} {
		d := NewDictionary(nil, []api.Property{{ID: 0x2129, DataType: api.SDTReal32, Value: "16"}})
		n, err := LoadProperties(filepath.Join(dir, name), vectorDevice, d, nil)
		if wantOK && (err != nil || n != 1) {
			t.Errorf("%s: n=%d err=%v", name, n, err)
		}
		if !wantOK && err == nil {
			t.Errorf("%s: expected a parse error", name)
		}
	}
	if _, err := LoadProperties(filepath.Join(dir, "missing.exml"), vectorDevice, NewDictionary(nil, nil), nil); err == nil {
		t.Error("missing file must fail")
	}
}

func TestUIStrings(t *testing.T) {
	if DefaultSettingsFileName("ACE0123456") != "ACE0123456.exml" {
		t.Error("DefaultSettingsFileName")
	}
	if SettingsLoadedDetail(3) != "Changed 3 properties.\nPlease save the changes to the device." {
		t.Error("SettingsLoadedDetail")
	}
	if PresetsDialogTitle != "Property Presets ACE Service Installer" {
		t.Error("PresetsDialogTitle")
	}
	if len(ForbiddenProperties()) != 38 || !IsForbidden(8273) || !IsForbidden(21016) || IsForbidden(8489) {
		t.Error("forbidden list")
	}
}
