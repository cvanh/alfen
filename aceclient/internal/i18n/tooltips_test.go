package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

const fixtureCSV = "../../../firmware/msi_work/files3/tooltip_en_GB.csv"

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	return b
}

func TestEmbeddedMatchesInstallerFile(t *testing.T) {
	want := readFixture(t, fixtureCSV)
	got, err := embedded.ReadFile("tooltip_en_GB.csv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("embedded tooltip_en_GB.csv differs from the installer's copy")
	}
}

func TestRealCSVStructure(t *testing.T) {
	reg := NewEmbedded()
	c := reg.Collection(LanguageEnglish)
	if c == nil {
		t.Fatal("no English collection")
	}
	if c.Language != LanguageEnglish {
		t.Fatalf("Language = %v", c.Language)
	}
	// 403 lines = 383 tips + 8 "// section" comments + 12 blank lines.
	if len(c.Tips) != 383 {
		t.Fatalf("tips = %d; want 383", len(c.Tips))
	}
	idRe := regexp.MustCompile(`^[0-9A-F]{4}_[0-9A-F]{2}$`)
	seen := map[string]int{}
	for _, tip := range c.Tips {
		if !idRe.MatchString(tip.ID) {
			t.Errorf("non-standard ID %q", tip.ID)
		}
		if tip.Name == "" || tip.LabelText == "" || tip.Tooltip == "" {
			t.Errorf("empty column in %+v", tip)
		}
		seen[tip.ID]++
	}
	if len(seen) != 380 {
		t.Errorf("unique IDs = %d; want 380", len(seen))
	}
	var dups []string
	for id, n := range seen {
		if n > 1 {
			dups = append(dups, id)
		}
	}
	if len(dups) != 3 {
		t.Errorf("duplicate IDs = %v; want 2541_00, 2542_00, 3221_0C", dups)
	}
	first := c.Tips[0]
	if first.ID != "1008_00" || first.Name != "OD_manufacturerDeviceName" || first.LabelText != "Platform type" {
		t.Errorf("first tip = %+v", first)
	}
	// No other language ships with the installer.
	for l := LanguageDutch; l <= LanguageIcelandic; l++ {
		if reg.Collection(l) != nil {
			t.Errorf("unexpected collection for %v", l)
		}
	}
}

func TestRealCSVLookups(t *testing.T) {
	reg := NewEmbedded()
	tests := []struct {
		name         string
		id           string
		label        string
		tipPrefix    string
		tipContains  string
		tipNotSubstr string
	}{
		{"backslash-n becomes newline", "2051_00", "Object Number",
			"Product unique ID used to purchase a new LicenseKey.",
			".\n(Back office config key 'FeatureObjectId')", `\n`},
		{"duplicate: first wins", "2541_00", "Public Key",
			"Public encryption key which is used to encrypt meter values for socket 1.", "", ""},
		{"duplicate: first wins (2)", "3221_0C", "Current L3 (A)", "Current L3 (A)", "", ""},
		{"stray comma kept in tooltip", "2189_00", "Maximum Allowed Phases",
			", The maximum number of phases that the CP is allowed to charge on.", "", ""},
		{"inner tabs kept", "2137_00", "Disconnect action", "The behavior of the CS",
			"\n- Continue:\n\tWhen the cable", ""},
		{"label inner tabs trimmed at end", "3182_00", "Bootloader version", "Bootloader version", "", ""},
		{"literal double quote", "213C_00", "Online action", "Select the action",
			`is online":` + "\n- Pre Authorize", ""},
		{"curly quotes (UTF-8)", "2530_03", "Mode (EMS)", "Choose whether",
			"‘Socket’ or ‘Smart Charging Network’", ""},
		{"degree sign (UTF-8)", "2201_00", "Temperature °C", "Measured temperature (°C) inside CS.", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := reg.GetLabelText(tc.id, LanguageUnknown); got != tc.label {
				t.Errorf("GetLabelText(%s) = %q; want %q", tc.id, got, tc.label)
			}
			tip := reg.GetTooltip(tc.id, LanguageEnglish)
			if !strings.HasPrefix(tip, tc.tipPrefix) {
				t.Errorf("GetTooltip(%s) = %q; want prefix %q", tc.id, tip, tc.tipPrefix)
			}
			if tc.tipContains != "" && !strings.Contains(tip, tc.tipContains) {
				t.Errorf("GetTooltip(%s) = %q; want to contain %q", tc.id, tip, tc.tipContains)
			}
			if tc.tipNotSubstr != "" && strings.Contains(tip, tc.tipNotSubstr) {
				t.Errorf("GetTooltip(%s) still contains %q", tc.id, tc.tipNotSubstr)
			}
		})
	}
	// GetTooltip never returns a literal `\n`; the raw Tip keeps it.
	c := reg.Collection(LanguageEnglish)
	raw := 0
	for _, tip := range c.Tips {
		if strings.Contains(tip.Tooltip, `\n`) {
			raw++
		}
		if strings.Contains(reg.GetTooltip(tip.ID, LanguageUnknown), `\n`) {
			t.Errorf("GetTooltip(%s) has a literal \\n", tip.ID)
		}
	}
	if raw == 0 {
		t.Error("expected raw tooltips to contain literal \\n sequences")
	}
	// Misses fall back to "".
	for _, id := range []string{"", "2067_01", "2062_0", "2062_00 ", "ffff_ff", "0000_00"} {
		if got := reg.GetTooltip(id, LanguageUnknown); got != "" {
			t.Errorf("GetTooltip(%q) = %q; want empty", id, got)
		}
		if got := reg.GetLabelText(id, LanguageUnknown); got != "" {
			t.Errorf("GetLabelText(%q) = %q; want empty", id, got)
		}
	}
}

func TestLanguageSelection(t *testing.T) {
	fsys := fstest.MapFS{
		"tooltip_en_GB.csv":     {Data: []byte("X_00, OD_x , Label EN , Tip EN\\nline2\n")},
		"tooltip_nl_NL.csv":     {Data: []byte("\xef\xbb\xbfX_00, OD_x , Label NL , Tip NL\r\n")},
		"tooltip_en_GB.csv.csv": {Data: []byte("X_00, OD_x , Label EN2 , Tip EN2\n")},
		"tooltip_xx_XX.csv":     {Data: []byte("X_00, OD_x , Label XX , Tip XX\n")},
		"tooltip_de.csv":        {Data: []byte("X_00, OD_x , Label DE , Tip DE\n")},
		"other.csv":             {Data: []byte("X_00, OD_x , Label O , Tip O\n")},
		"tooltip_fr_FR.csv/a":   {Data: []byte("dir, not, a, file\n")},
	}
	reg := NewFS(fsys)
	if reg.Language() != LanguageEnglish {
		t.Fatalf("initial language = %v; want English", reg.Language())
	}
	if got := reg.GetTooltip("X_00", LanguageUnknown); got != "Tip EN\nline2" {
		t.Errorf("current-language tooltip = %q", got)
	}
	// Two English files: GetCollection returns the first by name order.
	if got := reg.GetLabelText("X_00", LanguageEnglish); got != "Label EN" {
		t.Errorf("first English collection label = %q", got)
	}
	if got := reg.GetLabelText("X_00", LanguageDutch); got != "Label NL" {
		t.Errorf("explicit Dutch label = %q", got)
	}
	if reg.Collection(LanguageGerman) != nil || reg.Collection(LanguageFrench) != nil {
		t.Error("unexpected German/French collection")
	}
	reg.SetLanguage(LanguageDutch)
	if got := reg.GetTooltip("X_00", LanguageUnknown); got != "Tip NL" {
		t.Errorf("after SetLanguage(Dutch) = %q", got)
	}
	if got := reg.GetTooltip("X_00", LanguageEnglish); got != "Tip EN\nline2" {
		t.Errorf("explicit English after switch = %q", got)
	}
	reg.SetLanguage(LanguageGerman) // no collection -> ""
	if got := reg.GetTooltip("X_00", LanguageUnknown); got != "" {
		t.Errorf("missing language = %q; want empty", got)
	}
	reg.SetLanguage(LanguageUnknown) // accepted, matches nothing
	if got := reg.GetLabelText("X_00", LanguageUnknown); got != "" {
		t.Errorf("Unknown current language = %q; want empty", got)
	}
}

func TestInitIsLazyAndReinitReloads(t *testing.T) {
	dir := t.TempDir()
	write := func(s string) {
		if err := os.WriteFile(filepath.Join(dir, "tooltip_en_GB.csv"), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("A_00, n , L1 , T1\n")
	reg := NewFolder(dir)
	if got := reg.GetLabelText("A_00", LanguageUnknown); got != "L1" {
		t.Fatalf("label = %q", got)
	}
	write("A_00, n , L2 , T2\n")
	reg.Init() // already loaded: no-op
	if got := reg.GetLabelText("A_00", LanguageUnknown); got != "L1" {
		t.Fatalf("Init reloaded: label = %q", got)
	}
	reg.Reinit()
	if got := reg.GetLabelText("A_00", LanguageUnknown); got != "L2" {
		t.Fatalf("after Reinit label = %q", got)
	}
}

func TestMissingFolder(t *testing.T) {
	reg := NewFolder(filepath.Join(t.TempDir(), DefaultFolder))
	if reg.Collection(LanguageUnknown) != nil {
		t.Fatal("missing folder must yield no collections")
	}
	if got := reg.GetTooltip("2062_00", LanguageUnknown); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTooltipFile(t *testing.T) {
	fsys := fstest.MapFS{"tooltip_xx_XX.csv": {Data: []byte("a,b,c,d\n")}}
	if _, err := ParseTooltipFile(fsys, "tooltip_xx_XX.csv"); err != ErrUnknownLanguage {
		t.Errorf("err = %v; want ErrUnknownLanguage", err)
	}
	if _, err := ParseTooltipFile(fsys, "tooltip_nl_NL.csv"); err == nil {
		t.Error("missing file: want error")
	}
}

func TestPackageDefault(t *testing.T) {
	if CurrentLanguage() != LanguageEnglish {
		t.Fatalf("CurrentLanguage = %v", CurrentLanguage())
	}
	if got := LabelText("2062_00"); got != "Station maximum current (A)" {
		t.Errorf("LabelText = %q", got)
	}
	if got := Tooltip("2062_00"); !strings.HasSuffix(got, "\n(Back office config key 'Station-MaxCurrent')") {
		t.Errorf("Tooltip = %q", got)
	}
	SetLanguage(LanguageDutch)
	defer SetLanguage(LanguageEnglish)
	if got := Tooltip("2062_00"); got != "" {
		t.Errorf("Dutch (not shipped) Tooltip = %q; want empty", got)
	}
}

func TestConcurrentLookups(t *testing.T) {
	reg := NewEmbedded()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				reg.GetTooltip("2062_00", LanguageUnknown)
				if i == 0 && j%10 == 0 {
					reg.Reinit()
					reg.SetLanguage(LanguageEnglish)
				}
			}
		}(i)
	}
	wg.Wait()
}

// A Tooltips without a source behaves like a missing "Tooltips" folder
// instead of panicking on the first lookup.
func TestNoSource(t *testing.T) {
	var zero Tooltips
	nilFS := NewFS(nil)
	for name, reg := range map[string]*Tooltips{"zero value": &zero, "NewFS(nil)": nilFS} {
		t.Run(name, func(t *testing.T) {
			reg.Init()
			reg.Reinit()
			if c := reg.Collection(LanguageEnglish); c != nil {
				t.Errorf("Collection = %+v; want nil", c)
			}
			if got := reg.GetTooltip("2062_00", LanguageUnknown); got != "" {
				t.Errorf("GetTooltip = %q", got)
			}
			if got := reg.GetLabelText("2062_00", LanguageEnglish); got != "" {
				t.Errorf("GetLabelText = %q", got)
			}
			if reg.HasTooltip("2062_00") {
				t.Error("HasTooltip = true")
			}
			if got := reg.MakeTooltip("x", "2062_00"); got != "" {
				t.Errorf("MakeTooltip = %q", got)
			}
		})
	}
	if zero.Language() != LanguageUnknown {
		t.Errorf("zero value Language = %v; want Unknown (documented)", zero.Language())
	}
	if nilFS.Language() != LanguageEnglish {
		t.Errorf("NewFS(nil) Language = %v; want English", nilFS.Language())
	}
}

// HasTooltip and MakeTooltip on an instance use that instance's source and
// current language, not Default's.
func TestInstanceHasAndMakeTooltip(t *testing.T) {
	reg := NewFS(fstest.MapFS{
		"tooltip_en_GB.csv": {Data: []byte("9999_01, OD_x , Label , Line1\\nLine2\n")},
		"tooltip_nl_NL.csv": {Data: []byte("9999_02, OD_y , Label , NL only\n")},
	})
	if !reg.HasTooltip("9999_01") || HasTooltip("9999_01") {
		t.Fatalf("instance HasTooltip = %v, Default HasTooltip = %v; want true, false",
			reg.HasTooltip("9999_01"), HasTooltip("9999_01"))
	}
	if reg.HasTooltip("9999_02") {
		t.Error("9999_02 is Dutch-only; English current language must not see it")
	}
	if got := reg.MakeTooltip("anything", "9999_01"); got != "Line1\nLine2" {
		t.Errorf("MakeTooltip = %q", got)
	}
	if got := MakeTooltip("anything", "9999_01"); got != "" {
		t.Errorf("Default MakeTooltip = %q; want empty", got)
	}
	if got := reg.MakeTooltip("", "9999_01"); got != "" {
		t.Errorf("empty text: %q", got)
	}
	reg.SetLanguage(LanguageDutch)
	if reg.HasTooltip("9999_01") || !reg.HasTooltip("9999_02") {
		t.Error("HasTooltip must follow the instance's current language")
	}
	if got := reg.MakeTooltip("x", "9999_02"); got != "NL only" {
		t.Errorf("Dutch MakeTooltip = %q", got)
	}
}
