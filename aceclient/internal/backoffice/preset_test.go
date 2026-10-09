package backoffice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateFileTypesAndKeySuffix(t *testing.T) {
	if got := UpdateFileTypes(false); strings.Join(got, ",") != ".fwi,.fwu" {
		t.Errorf("NG9xx: %v", got)
	}
	if got := UpdateFileTypes(true); strings.Join(got, ",") != ".tfw,.tcf" {
		t.Errorf("AHP: %v", got)
	}
	for in, want := range map[string]string{
		"acme-a": "acme", "Acme-B": "Acme", "x-c": "x", "x-d": "x-d", "-a": "", "a": "a", "abc": "abc",
	} {
		if got := RemoveTrailingEncryptionKey(in); got != want {
			t.Errorf("RemoveTrailingEncryptionKey(%q) = %q; want %q", in, got, want)
		}
	}
}

func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCatalog(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "acme-a.fwi", "acme-b.fwi", "old.fwu", "Mixed-B.FWI", "ahp.tcf", "notes.txt", "dotted.name.fwi")
	if err := os.Mkdir(filepath.Join(dir, "sub.fwi"), 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := ScanPresetFolder(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != 5 {
		t.Fatalf("catalog: %v", c)
	}
	if !filepath.IsAbs(c["acme-a"]) || filepath.Base(c["dotted.name"]) != "dotted.name.fwi" {
		t.Errorf("paths: %v", c)
	}
	opts, err := c.Options()
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, o := range opts {
		keys = append(keys, o.Key)
	}
	if got := strings.Join(keys, "|"); got != "_MAN|_SA|acme|dotted.name|Mixed|old" {
		t.Errorf("option order: %s", got)
	}
	if opts[0].Title != TitleManual || opts[1].Title != TitleStandAlone || opts[2].Title != "acme" {
		t.Errorf("titles: %+v", opts[:3])
	}

	resolve := []struct {
		name string
		fw   Version
		want string
	}{
		{"acme", V(4, 12, 0), "acme-b.fwi"},
		{"acme", V(6, 6, 2), "acme-b.fwi"},
		{"acme", V(4, 11, 9), "acme-a.fwi"},
		{"old", V(7, 0, 0), "old.fwu"},
		{"Mixed", V(7, 0, 0), ""}, // case-sensitive: "Mixed-b" != "Mixed-B", like the C# Dictionary
		{"missing", V(7, 0, 0), ""},
	}
	for _, r := range resolve {
		got := c.Resolve(r.name, r.fw)
		if got != "" {
			got = filepath.Base(got)
		}
		if got != r.want {
			t.Errorf("Resolve(%q, %v) = %q; want %q", r.name, r.fw, got, r.want)
		}
	}

	ahp, err := ScanPresetFolder(dir, true)
	if err != nil || len(ahp) != 1 || ahp.Resolve("ahp", V(2, 4, 0)) == "" {
		t.Errorf("AHP catalog: %v %v", ahp, err)
	}
	if missing, err := ScanPresetFolder(filepath.Join(dir, "nope"), false); err != nil || len(missing) != 0 {
		t.Errorf("missing folder: %v %v", missing, err)
	}
	if _, err := NewCatalog([]string{"/a/x.fwi", "/b/x.fwu"}, false); err == nil {
		t.Error("duplicate names must fail like ToDictionary")
	}
	bad, _ := NewCatalog([]string{"/a/_MAN.fwi"}, false)
	if _, err := bad.Options(); err == nil {
		t.Error("a preset named _MAN clashes with dictionary.Add")
	}
}

// TestRealPresetFolder scans firmware/BackofficePresets (an FTP mirror with
// uppercase -A/-B names).
func TestRealPresetFolder(t *testing.T) {
	dir := "../../../firmware/BackofficePresets"
	if _, err := os.Stat(dir); err != nil {
		t.Skip("fixture folder not available")
	}
	ng, err := ScanPresetFolder(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	ahp, err := ScanPresetFolder(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(ng) != 790 || len(ahp) != 168 {
		t.Fatalf("catalog sizes: NG9xx %d, AHP %d", len(ng), len(ahp))
	}
	opts, err := ng.Options()
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) < 2+len(ng)/2 || opts[0].Key != KeyManual || opts[1].Key != KeyStandAlone {
		t.Errorf("options: %d, first %+v", len(opts), opts[:2])
	}
	for _, o := range opts[2:] {
		if strings.HasSuffix(o.Key, "-A") || strings.HasSuffix(o.Key, "-B") {
			t.Errorf("key suffix not stripped: %q", o.Key)
		}
	}
	// As in the C#, uppercase "-B" files are not found via name+"-b" (the
	// installer's FTP sync stores files lowercased); AHP .tcf presets resolve
	// by exact name.
	if ng.Resolve("Allego", V(6, 0, 0)) != "" {
		t.Error("case-sensitive lookup expected")
	}
	if ahp.Resolve("Allego", V(2, 4, 0)) == "" {
		t.Error("AHP preset Allego.tcf must resolve by exact name")
	}
}

func TestShortName(t *testing.T) {
	cases := []struct{ in, preset, meter string }{
		{"acme", "acme", ""},
		{"acme,ABB B23", "acme", "ABB B23"},
		{"acme,m,extra", "acme", "m"},
		{"", "", ""},
	}
	for _, c := range cases {
		p, m := ParseShortName(c.in)
		if p != c.preset || m != c.meter {
			t.Errorf("ParseShortName(%q) = %q, %q", c.in, p, m)
		}
	}
	if CombineShortName("acme", "") != "acme" || CombineShortName("acme", "m") != "acme,m" || CombineShortName("", "m") != ",m" {
		t.Error("CombineShortName basics")
	}
	long := strings.Repeat("p", 45)
	if CombineShortName(long, "1234") != long+",1234" || CombineShortName(long, "12345") != long {
		t.Error("CombineShortName 50-char limit")
	}
	if NormalizeConnectMethod(99) != "3" || NormalizeConnectMethod(2) != "2" {
		t.Error("NormalizeConnectMethod")
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		s, t string
		d    int
	}{
		{"", "abc", 3}, {"abc", "", 3}, {"kitten", "sitting", 3}, {"flaw", "lawn", 2}, {"same", "same", 0},
	}
	for _, c := range cases {
		if got := LevenshteinDistance(c.s, c.t); got != c.d {
			t.Errorf("LevenshteinDistance(%q, %q) = %d; want %d", c.s, c.t, got, c.d)
		}
	}
}

func TestSelectionFromDevice(t *testing.T) {
	opts := []Option{{KeyManual, TitleManual}, {KeyStandAlone, TitleStandAlone}, {"acme", "acme"}, {"tnm", "tnm"}}
	cases := []struct {
		name string
		in   SelectionInput
		want Selection
	}{
		{"empty legacy none", SelectionInput{ShortName: "", ConnectMethod: 0}, Selection{KeyStandAlone, "0"}},
		{"standalone legacy wired", SelectionInput{ShortName: "Standalone - all cards", ConnectMethod: 1}, Selection{KeyManual, "1"}},
		{"legacy 99 is auto", SelectionInput{ShortName: "stand-alone", ConnectMethod: 99}, Selection{KeyManual, "3"}},
		{"profiles all off", SelectionInput{ShortName: ",meter", HasNetworkProfiles: true}, Selection{Preset: KeyStandAlone}},
		{"profiles one on", SelectionInput{ShortName: "", HasNetworkProfiles: true, ProfilePriorities: [4]int{0, 2, 0, 0}}, Selection{Preset: KeyManual}},
		{"preset lowercased", SelectionInput{ShortName: " ACME ,meter"}, Selection{Preset: "acme"}},
		{"part management exact", SelectionInput{ShortName: "TNM - production gprs", PartManagementMode: true, Options: opts}, Selection{Preset: "tnm"}},
		{"part management closest", SelectionInput{ShortName: "acmee", PartManagementMode: true, Options: opts}, Selection{Preset: "acme"}},
	}
	for _, c := range cases {
		if got := SelectionFromDevice(c.in); got != c.want {
			t.Errorf("%s: got %+v; want %+v", c.name, got, c.want)
		}
	}
}

func TestCultureCompare(t *testing.T) {
	ordered := []string{"_MAN", "_SA", "4th-dimension", "abel&co", "abel&co-(wired)", "Abel&co-(wired)", "acme", "Zed"}
	for i := 0; i+1 < len(ordered); i++ {
		if cultureCompare(ordered[i], ordered[i+1]) >= 0 {
			t.Errorf("%q should sort before %q", ordered[i], ordered[i+1])
		}
	}
	if cultureCompare("co-op", "coop") == 0 {
		t.Error("final ordinal tie-break expected")
	}
}
