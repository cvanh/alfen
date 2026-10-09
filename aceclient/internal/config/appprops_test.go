package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaths(t *testing.T) {
	p := Paths{AppData: "/data"}
	tests := []struct {
		name, got, want string
	}{
		{"LocalSetupFolder", p.LocalSetupFolder(), "/data/ACE Service Installer"},
		{"SettingsFilename", p.SettingsFilename(), "/data/ACE Service Installer/InstallerConfigV3.dat"},
		{"SettingsFilenameV2", p.SettingsFilenameV2(), "/data/ACE Service Installer/InstallerConfigV2.dat"},
		{"SettingsFilenameOld", p.SettingsFilenameOld(), "/data/ACE Service Installer/InstallerConfig.dat"},
	}
	for _, tc := range tests {
		if tc.got != filepath.FromSlash(tc.want) {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// hashedConfigText returns the EncryptRijndael envelope of the synthetic
// hashed config (users already hashed, so they are written as they are).
// edit, when set, rewrites the plain JSON first.
func hashedConfigText(t *testing.T, version string, edit func(string) string) string {
	t.Helper()
	c, _ := hashedFixture(t)
	c.Version = version
	text, err := c.Marshal(EncryptNone, true, "D")
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		text = edit(text)
	}
	return Encrypt(text, ConfigPassPhrase)
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeHashedConfig(t *testing.T, path, version string) {
	t.Helper()
	writeFile(t, path, hashedConfigText(t, version, nil))
}

// badFirmwares makes the read fail at the first firmware entry, i.e. after
// Version, Date, the features, groups and users have been read.
func badFirmwares(s string) string {
	return strings.Replace(s, "\"Firmwares\":[\n]}", "\"Firmwares\":[{\"Filename\":\"f\"}]}", 1)
}

func TestInitializeSettings(t *testing.T) {
	p := Paths{AppData: t.TempDir()}
	install := t.TempDir()

	// Neither file: empty config + error, like the C# carrying on with new ICUConfig().
	c, from, err := InitializeSettings(p, install)
	if err == nil || from != "" || c == nil || len(c.Users) != 0 || c.Version != "" {
		t.Fatalf("no config: %v %q", err, from)
	}
	if _, err := c.Logon("alice", synthAlicePass, nil); !errors.Is(err, ErrUnknownUser) {
		t.Fatalf("empty config logon: %v", err)
	}

	// Obsolete files are deleted; fallback to the installation folder.
	for _, f := range []string{p.SettingsFilenameOld(), p.SettingsFilenameV2()} {
		writeFile(t, f, "old")
	}
	writeHashedConfig(t, filepath.Join(install, ConfigFileName), "install")
	c, from, err = InitializeSettings(p, install)
	if err != nil || c.Version != "install" || from != filepath.Join(install, ConfigFileName) {
		t.Fatalf("fallback: %v %q", err, from)
	}
	for _, f := range []string{p.SettingsFilenameOld(), p.SettingsFilenameV2()} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", f)
		}
	}

	// The AppData copy wins when it loads.
	writeHashedConfig(t, p.SettingsFilename(), "appdata")
	c, from, err = InitializeSettings(p, install)
	if err != nil || c.Version != "appdata" || from != p.SettingsFilename() {
		t.Fatalf("primary: %v %q", err, from)
	}
	if _, err := c.Logon("alice", synthAlicePass, nil); err != nil {
		t.Fatalf("logon against initialised config: %v", err)
	}

	// A corrupt AppData copy falls back to the installation folder.
	writeFile(t, p.SettingsFilename(), "garbage")
	if c, _, err = InitializeSettings(p, install); err != nil || c.Version != "install" {
		t.Fatalf("corrupt primary: %v", err)
	}
}

// TestInitializeSettingsPartialState pins the C# behaviour of reading both
// files into one static ICUConfig without rolling back on failure.
func TestInitializeSettingsPartialState(t *testing.T) {
	// AppData copy fails late (bad firmware entry), fallback missing: the
	// users and groups read before the failure stay and logon still works.
	p := Paths{AppData: t.TempDir()}
	install := t.TempDir()
	writeFile(t, p.SettingsFilename(), hashedConfigText(t, "partial", badFirmwares))
	c, from, err := InitializeSettings(p, install)
	if err == nil || from != "" {
		t.Fatalf("both reads must fail: %v %q", err, from)
	}
	if c.Version != "partial" || len(c.Users) != 2 || len(c.Groups) != 2 || len(c.Firmwares) != 0 {
		t.Fatalf("partial state lost: version %q, %d users, %d groups, %d firmwares", c.Version, len(c.Users), len(c.Groups), len(c.Firmwares))
	}
	if n := c.FindGroup("Service").NumberOfUsers; n != 0 {
		t.Fatalf("NumberOfUsers is counted only after the firmwares, got %d", n)
	}
	if _, err := c.Logon("alice", synthAlicePass, nil); err != nil {
		t.Fatalf("logon against the partial config: %v", err)
	}

	// The fallback also fails late: its own partial state replaces the
	// collections.
	writeFile(t, filepath.Join(install, ConfigFileName), hashedConfigText(t, "partial-install", func(s string) string {
		return strings.Replace(s, "\"Users\":[", "\"Users\":[{\"User\":\"x\"},", 1) // first user lacks Password
	}))
	c, _, err = InitializeSettings(p, install)
	if err == nil || c.Version != "partial-install" || len(c.Users) != 0 || len(c.Groups) != 2 {
		t.Fatalf("fallback partial state: %v version %q, %d users, %d groups", err, c.Version, len(c.Users), len(c.Groups))
	}

	// A fallback that fails after Version but before Date keeps the earlier
	// collections and only replaces Version.
	writeFile(t, filepath.Join(install, ConfigFileName), Encrypt(`{"Type":"ICUConfigFile","Version":"v-only"}`, ConfigPassPhrase))
	c, _, err = InitializeSettings(p, install)
	if err == nil || c.Version != "v-only" || c.Date != "D" || len(c.Users) != 2 {
		t.Fatalf("Version-only fallback: %v version %q date %q, %d users", err, c.Version, c.Date, len(c.Users))
	}
}

func TestReadInstallerSettingsKeepsStateOnEarlyErrors(t *testing.T) {
	c, _ := hashedFixture(t)
	users := len(c.Users)
	for name, text := range map[string]string{
		"not base64": "garbage!",
		"wrong type": Encrypt(`{"Type":"Other","Version":"x"}`, ConfigPassPhrase),
		"no Version": Encrypt(`{"Type":"ICUConfigFile"}`, ConfigPassPhrase),
	} {
		fn := filepath.Join(t.TempDir(), "cfg.dat")
		writeFile(t, fn, text)
		if err := c.ReadInstallerSettings(fn, EncryptRijndaelHashed); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
		if c.Version != "9.9.9-test" || len(c.Users) != users {
			t.Fatalf("%s: state changed before the failure point", name)
		}
	}
	if err := c.ReadInstallerSettings(filepath.Join(t.TempDir(), "missing.dat"), EncryptRijndaelHashed); err == nil {
		t.Fatal("missing file must fail")
	}
}
