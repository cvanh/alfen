package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const decompiled = "../../../firmware/decompiled"

func readFixture(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(decompiled, rel))
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	return string(b)
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// TestDefaultsMatchSettingsCs checks every user setting in Settings.cs (key,
// C# type, DefaultSettingValue) against Settings/Defaults().
func TestDefaultsMatchSettingsCs(t *testing.T) {
	src := readFixture(t, "ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs")
	re := regexp.MustCompile(`\[UserScopedSetting\]\s*\[DebuggerNonUserCode\]\s*(?:\[DefaultSettingValue\("([^"]*)"\)\]\s*)?public (\w+) (\w+)`)
	matches := re.FindAllStringSubmatch(src, -1)
	if len(matches) != 18 {
		t.Fatalf("expected 18 user settings in Settings.cs, found %d", len(matches))
	}
	def := reflect.ValueOf(Defaults())
	ported := 0
	for _, m := range matches {
		dflt, typ, name := m[1], m[2], m[3]
		if name == "AppInsightsConnStrEncrypted" {
			if _, ok := def.Type().FieldByName(name); ok {
				t.Errorf("%s is telemetry and must not be ported", name)
			}
			continue
		}
		ported++
		f := def.FieldByName(name)
		if !f.IsValid() {
			t.Errorf("setting %s not ported", name)
			continue
		}
		field, _ := def.Type().FieldByName(name)
		if tag := field.Tag.Get("json"); tag != name {
			t.Errorf("%s: json key %q, want the C# name", name, tag)
		}
		switch typ {
		case "string":
			if f.Kind() != reflect.String || f.String() != dflt {
				t.Errorf("%s: got %v, want string %q", name, f, dflt)
			}
		case "int":
			want, _ := strconv.Atoi(dflt)
			if f.Kind() != reflect.Int || int(f.Int()) != want {
				t.Errorf("%s: got %v, want int %d", name, f, want)
			}
		case "bool":
			want, _ := strconv.ParseBool(dflt)
			if f.Kind() != reflect.Bool || f.Bool() != want {
				t.Errorf("%s: got %v, want bool %v", name, f, want)
			}
		case "DateTime":
			tm, ok := f.Interface().(time.Time)
			if !ok || tm.Format("2006-01-02") != dflt || tm.Hour() != 0 {
				t.Errorf("%s: got %v, want date %s", name, f, dflt)
			}
		case "StringCollection":
			if f.Kind() != reflect.Slice || !f.IsNil() || dflt != "" {
				t.Errorf("%s: got %v, want nil []string", name, f)
			}
		default:
			t.Errorf("%s: unexpected C# type %s", name, typ)
		}
	}
	if ported != 17 {
		t.Errorf("ported %d settings, want 17", ported)
	}
}

func TestOpenMissingFileGivesDefaultsAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(st.Get(), Defaults()) {
		t.Errorf("got %+v, want defaults", st.Get())
	}
	if _, err := os.Stat(st.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file written on open: %v", err)
	}
	if _, err := Open(""); err == nil {
		t.Error("Open(\"\") succeeded")
	}
}

func TestRoundTripAtomicAndPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", DirName)
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	err = st.Update(func(s *Settings) {
		s.LastUserName = "someone"
		s.AskConfirmExit = false
		s.LastManualIPPort = 8443
		s.LastPasswordStoreTime = when
		s.LocalPasswords = []string{"a;b"}
	})
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(st.Path())
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("perm %v, want 0600", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != FileName {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("leftover files: %v", names)
	}
	st2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := st2.Get()
	if got.LastUserName != "someone" || got.AskConfirmExit || got.LastManualIPPort != 8443 ||
		!got.LastPasswordStoreTime.Equal(when) || !reflect.DeepEqual(got.LocalPasswords, []string{"a;b"}) {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if !got.AskSaveChanges || got.LastManualNumberOfSockets != 2 {
		t.Errorf("untouched defaults changed: %+v", got)
	}
}

func TestMissingKeysKeepDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"LastUserName":"x","AskSaveChanges":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := Defaults()
	want.LastUserName = "x"
	want.AskSaveChanges = false
	if !reflect.DeepEqual(st.Get(), want) {
		t.Errorf("got %+v\nwant %+v", st.Get(), want)
	}
}

func TestCorruptFile(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"syntax", "{not json"},
		// One wrongly typed field makes the whole document undecodable; the
		// other keys (profiles, manual devices) must survive in the backup.
		{"wrong type", `{"LastManualIPPort":"443","Profiles":[{"Name":"a","IP":"10.0.0.1","Port":443,"User":""}],"ManualDevices":[{"IPAddress":"10.0.0.2","Port":443,"Hostname":"Lolo3","NumberOfSockets":1,"LoginRequired":true}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			if err := os.WriteFile(path, []byte(c.data), 0o600); err != nil {
				t.Fatal(err)
			}
			st, err := Open(dir)
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("err = %v, want ErrCorrupt", err)
			}
			var cerr *CorruptFileError
			if !errors.As(err, &cerr) || cerr.Path != path || cerr.Backup == "" ||
				!strings.HasPrefix(filepath.Base(cerr.Backup), FileName+".corrupt-") || !strings.Contains(err.Error(), cerr.Backup) {
				t.Fatalf("corrupt error %#v", err)
			}
			if st == nil || !reflect.DeepEqual(st.Get(), Defaults()) {
				t.Fatal("corrupt open must return a usable store with defaults")
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("corrupt file still in place: %v", err)
			}
			if b, err := os.ReadFile(cerr.Backup); err != nil || string(b) != c.data {
				t.Fatalf("backup %q: %v", b, err)
			}
			if err := st.SetLastUserName("y"); err != nil {
				t.Fatal(err)
			}
			if b, _ := os.ReadFile(cerr.Backup); string(b) != c.data {
				t.Error("save touched the backup")
			}
			st2, err := Open(dir)
			if err != nil || st2.Get().LastUserName != "y" {
				t.Errorf("save did not write a fresh file: %v %+v", err, st2)
			}
		})
	}
}

func TestCorruptBackupNamesDoNotCollide(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	when := time.Unix(1700000000, 0)
	var names []string
	for i := range 3 {
		data := fmt.Sprintf("bad %d", i)
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		name, err := backupCorrupt(path, when)
		if err != nil {
			t.Fatal(err)
		}
		if b, _ := os.ReadFile(name); string(b) != data {
			t.Errorf("backup %s = %q", name, b)
		}
		names = append(names, filepath.Base(name))
	}
	want := []string{FileName + ".corrupt-1700000000", FileName + ".corrupt-1700000000-1", FileName + ".corrupt-1700000000-2"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names %v, want %v", names, want)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	st := openTemp(t)
	if err := st.Save(); err != nil {
		t.Fatal(err)
	}
	st2, err := Open(st.Dir())
	if err != nil || !reflect.DeepEqual(st2.Get(), Defaults()) {
		t.Errorf("saved defaults did not round-trip: %v %+v", err, st2.Get())
	}
}

func TestOpenDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	st, err := OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := DefaultDir()
	if st.Dir() != dir || filepath.Base(dir) != DirName || !strings.HasPrefix(dir, home) {
		t.Errorf("dir %q (default %q)", st.Dir(), dir)
	}
}

// A failed save keeps the in-memory change, returns the error and leaves no
// temp file behind (rename onto a directory fails after the temp file exists).
func TestFailedSaveKeepsChangeAndRemovesTemp(t *testing.T) {
	st := openTemp(t)
	if err := os.MkdirAll(filepath.Join(st.Path(), "blocker"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := st.SetLastUserName("kept"); err == nil {
		t.Fatal("save over a directory succeeded")
	}
	if st.Get().LastUserName != "kept" {
		t.Error("in-memory change lost after failed save")
	}
	entries, _ := os.ReadDir(st.Dir())
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left: %s", e.Name())
		}
	}
	// settings.json being unreadable for another reason than "missing" is a
	// hard error: no store.
	if s2, err := Open(st.Dir()); err == nil || s2 != nil || errors.Is(err, ErrCorrupt) {
		t.Errorf("Open on a directory: %v %v", s2, err)
	}
}

func skipIfNoPermissionChecks(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || runtime.GOOS == "js" || runtime.GOOS == "wasip1" {
		t.Skip("POSIX permissions not enforced")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores permissions")
	}
}

func TestReadOnlyDirSaveFails(t *testing.T) {
	skipIfNoPermissionChecks(t)
	st := openTemp(t)
	if err := os.Chmod(st.Dir(), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(st.Dir(), 0o700) })
	if err := st.Update(func(s *Settings) { s.AskConfirmExit = false }); err == nil {
		t.Fatal("save into a read-only directory succeeded")
	}
	if st.Get().AskConfirmExit {
		t.Error("in-memory change lost after failed save")
	}
	if entries, _ := os.ReadDir(st.Dir()); len(entries) != 0 {
		t.Errorf("files left: %d", len(entries))
	}
}

func TestOpenUnreadableFile(t *testing.T) {
	skipIfNoPermissionChecks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"LastUserName":"x"}`), 0o000); err != nil {
		t.Fatal(err)
	}
	st, err := Open(dir)
	if err == nil || st != nil || errors.Is(err, ErrCorrupt) || !errors.Is(err, os.ErrPermission) {
		t.Errorf("Open = %v, %v; want nil store and a permission error", st, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("unreadable file must stay in place: %v", err)
	}
}

func TestGetReturnsDeepCopy(t *testing.T) {
	st := openTemp(t)
	_ = st.Update(func(s *Settings) {
		s.LocalPasswords = []string{"a"}
		s.LastDevice = &Profile{IP: "1.2.3.4", Port: 443}
	})
	g := st.Get()
	g.LocalPasswords[0] = "mutated"
	g.LastDevice.IP = "mutated"
	g2 := st.Get()
	if g2.LocalPasswords[0] != "a" || g2.LastDevice.IP != "1.2.3.4" {
		t.Error("Get aliases internal state")
	}
}

func TestConcurrentUpdates(t *testing.T) {
	st := openTemp(t)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 10 {
				_ = st.Update(func(s *Settings) { s.LastManualIPPort++ })
				_ = st.Get()
				_, _ = st.SaveProfile(Profile{Name: fmt.Sprintf("p%d", i), IP: "10.0.0.1", Port: 443 + j})
				_ = st.Preferences()
			}
		}()
	}
	wg.Wait()
	if got := st.Get().LastManualIPPort; got != 443+200 {
		t.Errorf("LastManualIPPort = %d, want %d", got, 443+200)
	}
	st2, err := Open(st.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if st2.Get().LastManualIPPort != 643 || len(st2.Profiles()) != 20 {
		t.Errorf("persisted state wrong: port %d, %d profiles", st2.Get().LastManualIPPort, len(st2.Profiles()))
	}
}

func TestJSONKeysAreCSharpNames(t *testing.T) {
	b, err := json.Marshal(Defaults())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"LastUserName", "LocalPasswords", "LastManualIPAddress", "LastManualIPPort",
		"LastCommand", "AskConfirmExit", "AskSaveChanges", "LastFTPActivity", "LastFTPConnectionWarning",
		"LastDevicePassword", "LastDeviceUsername", "StorePasswords", "LastPasswordStoreTime",
		"LastManualHostname", "LastManualNumberOfSockets", "LastManualLoginRequired", "LastManualModelType"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
	for _, k := range []string{"Profiles", "LastDevice", "ManualDevices"} {
		if _, ok := m[k]; ok {
			t.Errorf("empty Go-native key %s should be omitted", k)
		}
	}
}

func TestPaths(t *testing.T) {
	p := Paths{Root: filepath.Join("r", LocalInstallerFolder)}
	cases := []struct {
		name, got, want string
	}{
		{"SetupFolder", p.SetupFolder(), p.Root},
		{"SettingsFilename", p.SettingsFilename(), filepath.Join(p.Root, "InstallerConfigV3.dat")},
		{"SettingsFilenameV2", p.SettingsFilenameV2(), filepath.Join(p.Root, "InstallerConfigV2.dat")},
		{"SettingsFilenameOld", p.SettingsFilenameOld(), filepath.Join(p.Root, "InstallerConfig.dat")},
		{"FirmwareFolder", p.FirmwareFolder(), filepath.Join(p.Root, "Firmware")},
		{"OldPresetsFolder", p.OldPresetsFolder(), filepath.Join(p.Root, "Presets")},
		{"TCPPresetsFolder", p.TCPPresetsFolder(), filepath.Join(p.Root, "TCPPresets")},
		{"RTUPresetsFolder", p.RTUPresetsFolder(), filepath.Join(p.Root, "RTUPresets")},
		{"BackofficePresetsFolder", p.BackofficePresetsFolder(), filepath.Join(p.Root, "BackofficePresets")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if c := p.InstallerConfigCandidates(); len(c) != 2 || c[0] != p.SettingsFilename() || c[1] != "InstallerConfigV3.dat" {
		t.Errorf("candidates %v", c)
	}
	// DefaultPaths is the C# LocalSetupFolder (%APPDATA%\ACE Service
	// Installer), not the Go settings directory.
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	dp, err := DefaultPaths()
	if err != nil || dp.Root != filepath.Join(base, "ACE Service Installer") {
		t.Errorf("default root %q %v", dp.Root, err)
	}
	if lp, err := LegacyPaths(); err != nil || lp != dp {
		t.Errorf("legacy root %q %v", lp.Root, err)
	}
	if dir, _ := DefaultDir(); dir == dp.Root {
		t.Error("settings.json must not share the C# setup folder")
	}
}

func TestRemoveObsoleteConfigs(t *testing.T) {
	p := Paths{Root: t.TempDir()}
	for _, f := range []string{p.SettingsFilenameOld(), p.SettingsFilenameV2(), p.SettingsFilename()} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.RemoveObsoleteConfigs(); err != nil {
		t.Fatal(err)
	}
	if err := p.RemoveObsoleteConfigs(); err != nil {
		t.Fatalf("missing files must be ignored: %v", err)
	}
	for _, f := range []string{p.SettingsFilenameOld(), p.SettingsFilenameV2()} {
		if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s not removed", f)
		}
	}
	if _, err := os.Stat(p.SettingsFilename()); err != nil {
		t.Errorf("V3 config must stay: %v", err)
	}
}

func TestAppPropertiesConstantsMatchSource(t *testing.T) {
	src := readFixture(t, "ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs")
	for name, want := range map[string]string{
		"AppName": AppName, "LocalInstallerFolder": LocalInstallerFolder,
		"LocalFirmwareFolderName": LocalFirmwareFolderName, "LocalLogFolderName": LocalLogFolderName,
		"LocalOldPresetsFolderName": LocalOldPresetsFolderName, "LocalTCPPresetsFolderName": LocalTCPPresetsFolderName,
		"LocalRTUPresetsFolderName": LocalRTUPresetsFolderName, "LocalBackofficePresetsFolderName": LocalBackofficePresetsFolderName,
		"ConfigFileNameOld": ConfigFileNameOld, "ConfigFileNameV2": ConfigFileNameV2, "ConfigFileName": ConfigFileName,
		"IsahSite": IsahSite, "DefaultUserName": DefaultUserName,
	} {
		if !strings.Contains(src, fmt.Sprintf("public static string %s = %q;", name, want)) {
			t.Errorf("%s = %q not found in AppProperties.cs", name, want)
		}
	}
	for name, want := range map[string]string{
		"FallbackSettingsFilename": FallbackSettingsFilename, "ProgramFilesPresetsFolder": ProgramFilesPresetsFolder,
		"UILanguagesFolder": UILanguagesFolder,
	} {
		if !strings.Contains(src, fmt.Sprintf("public static string %s => %q;", name, want)) {
			t.Errorf("%s => %q not found in AppProperties.cs", name, want)
		}
	}
	if !strings.Contains(src, fmt.Sprintf("ButtonHeight = %d;", ButtonHeight)) || !strings.Contains(src, fmt.Sprintf("ButtonWidth = %d;", ButtonWidth)) {
		t.Error("button sizes differ")
	}
	if !strings.Contains(src, fmt.Sprintf("Color_Border = Color.FromBytes(%d, %d, %d)", ColorBorder.R, ColorBorder.G, ColorBorder.B)) ||
		!strings.Contains(src, fmt.Sprintf("Color_Disabled = Color.FromBytes(%d, %d, %d)", ColorDisabled.R, ColorDisabled.G, ColorDisabled.B)) {
		t.Error("colors differ")
	}
}
