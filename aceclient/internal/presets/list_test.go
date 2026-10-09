package presets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, content string, mod time.Time) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
	return p
}

const presetA = `<?xml version="1.0" encoding="utf-8"?>
<Settings>
  <Version>1.2</Version>
  <Model>Twin 4.0</Model>
  <NumberOfSockets>2</NumberOfSockets>
  <Information>Info A</Information>
  <Properties>
    <Property Id="2129_0" Value="16" />
    <Property Id="2129_00" Value="99" />
    <Property Id="2129_0" Value="32" />
  </Properties>
</Settings>`

const presetB = `<Settings><Model>EVe-single</Model><Properties/></Settings>`

func TestLoadPresetsFromFolder(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)
	writeFile(t, dir, "a.xml", presetA, now.Add(-time.Hour))
	writeFile(t, dir, "B.IIP", presetB, now)
	writeFile(t, dir, "f.xml", `<Other><Version>9</Version></Other>`, now.Add(-2*time.Hour))
	writeFile(t, dir, "c.exml", "not listed", now)
	writeFile(t, dir, "d.json", "{}", now)
	if err := os.Mkdir(filepath.Join(dir, "e.xml"), 0o755); err != nil {
		t.Fatal(err)
	}
	rows, err := LoadPresetsFromFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		got = append(got, strings.Join(r.Columns(), "|"))
		if !filepath.IsAbs(r.Path) || !strings.HasSuffix(r.Path, r.Name+filepath.Ext(r.Path)) {
			t.Errorf("Path = %q", r.Path)
		}
	}
	want := []string{
		"B||EVe-single|||",
		"a|1.2|Twin 4.0|2|32|Info A", // last 2129_0 wins; "2129_00" is not the power id
		"f|||||",                     // no <Settings>: empty row
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestLoadPresetsFromFolderStopsAtBrokenFile(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for name, broken := range map[string]string{
		"malformed":       `<Settings><Version>1`,
		"no properties":   `<Settings><Version>1</Version></Settings>`,
		"property no id":  `<Settings><Properties><Property Value="1"/></Properties></Settings>`,
		"power no value":  `<Settings><Properties><Property Id="2129_0"/></Properties></Settings>`,
		"multiple roots":  `<Settings><Properties/></Settings><Settings/>`,
		"unknown charset": `<?xml version="1.0" encoding="ebcdic-x"?><Settings><Properties/></Settings>`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "new.xml", presetB, now)
			writeFile(t, dir, "broken.xml", broken, now.Add(-time.Minute))
			writeFile(t, dir, "old.xml", presetA, now.Add(-time.Hour))
			rows, err := LoadPresetsFromFolder(dir)
			if err == nil {
				t.Fatal("expected an error")
			}
			if len(rows) != 1 || rows[0].Name != "new" {
				t.Errorf("rows = %+v (scan must stop at the broken file)", rows)
			}
		})
	}
}

func TestLoadPresetsFromFolderEncodings(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)
	// UTF-16 LE with BOM
	src := `<?xml version="1.0" encoding="utf-16"?><Settings><Information>é€</Information><Properties/></Settings>`
	var u16 []byte
	u16 = append(u16, 0xFF, 0xFE)
	for _, r := range src {
		u16 = append(u16, byte(r), byte(r>>8))
	}
	writeFile(t, dir, "u16.xml", string(u16), now)
	// Windows-1252 declared, 0x80 is the euro sign
	writeFile(t, dir, "cp.xml", "<?xml version=\"1.0\" encoding=\"windows-1252\"?><Settings><Information>\xE9\x80</Information><Properties/></Settings>", now.Add(-time.Minute))
	// UTF-8 BOM
	writeFile(t, dir, "bom.xml", "\xEF\xBB\xBF<Settings><Information>é€</Information><Properties/></Settings>", now.Add(-2*time.Minute))
	rows, err := LoadPresetsFromFolder(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	for _, r := range rows {
		if r.Information != "é€" {
			t.Errorf("%s: Information = %q", r.Name, r.Information)
		}
	}
}

func TestLoadPresetsFromFolderCreatesFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "x", "Presets")
	rows, err := LoadPresetsFromFolder(dir)
	if err != nil || len(rows) != 0 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Error("folder must be created")
	}
	file := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPresetsFromFolder(file); err == nil {
		t.Error("a file in place of the folder must fail like Directory.CreateDirectory")
	}
}

func TestFillPresetList(t *testing.T) {
	root := t.TempDir()
	f := NewFolders(filepath.Join(root, "appdata"), filepath.Join(root, "program", "Presets"))
	now := time.Now().Truncate(time.Second)
	for _, d := range []string{f.ProgramFilesPresets, f.LocalTCPPresets, f.LocalBackofficePresets} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, f.ProgramFilesPresets, "pf.xml", presetA, now.Add(-time.Hour))
	writeFile(t, f.LocalTCPPresets, "tcp.xml", presetB, now)
	writeFile(t, f.LocalTCPPresets, "meter.json", "{}", now)
	writeFile(t, f.LocalBackofficePresets, "bad.iip", "<", now)
	rows, err := FillPresetList(f)
	if err == nil {
		t.Error("the broken backoffice file must be reported")
	}
	if len(rows) != 2 || rows[0].Name != "pf" || rows[1].Name != "tcp" {
		t.Errorf("rows = %+v (folder order beats modification time)", rows)
	}
	if _, err := os.Stat(f.LocalRTUPresets); err != nil {
		t.Error("missing RTU folder must be created")
	}
}

// TestFixtureFolders documents the real preset folders shipped/synchronised
// by the installer: none holds *.xml/*.iip Settings presets.
func TestFixtureFolders(t *testing.T) {
	base := filepath.Join("..", "..", "..", "firmware")
	want := map[string][]string{
		"Presets":                   {".json"},
		"TCPPresets":                {".json"},
		"RTUPresets":                {".json"},
		"BackofficePresets":         {".fwi", ".tcf"},
		"TestBackOfficePresets":     {".fwi", ".tcf"},
		"TestBackOfficePresetsTest": {".fwi", ".tcf"},
		"TestPath":                  {".fwi", ".tcf"},
	}
	seen := 0
	for name, exts := range want {
		dir := filepath.Join(base, name)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue // LoadPresetsFromFolder would create it; never touch fixtures
		}
		seen++
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, e := range entries {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			ok := false
			for _, x := range exts {
				ok = ok || ext == x
			}
			if !ok {
				t.Errorf("%s: unexpected file %q", name, e.Name())
			}
			count++
		}
		if count == 0 {
			t.Errorf("%s: empty", name)
		}
		rows, err := LoadPresetsFromFolder(dir)
		if err != nil || len(rows) != 0 {
			t.Errorf("%s: rows=%d err=%v", name, len(rows), err)
		}
	}
	if seen == 0 {
		t.Skip("firmware fixtures not present")
	}
}
