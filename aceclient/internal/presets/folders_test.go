package presets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFolders(t *testing.T) {
	f := NewFolders(filepath.FromSlash("/appdata"), ProgramFilesPresetsFolder)
	want := Folders{
		ProgramFilesPresets:    "Presets",
		LocalOldPresets:        filepath.FromSlash("/appdata/ACE Service Installer/Presets"),
		LocalTCPPresets:        filepath.FromSlash("/appdata/ACE Service Installer/TCPPresets"),
		LocalRTUPresets:        filepath.FromSlash("/appdata/ACE Service Installer/RTUPresets"),
		LocalBackofficePresets: filepath.FromSlash("/appdata/ACE Service Installer/BackofficePresets"),
	}
	if f != want {
		t.Errorf("%+v", f)
	}
	order := f.PresetListFolders()
	if len(order) != 4 || order[0] != want.ProgramFilesPresets || order[1] != want.LocalTCPPresets ||
		order[2] != want.LocalRTUPresets || order[3] != want.LocalBackofficePresets {
		t.Errorf("PresetListFolders = %v", order)
	}
}

func TestDefaultFolders(t *testing.T) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	f, err := DefaultFolders()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(f.LocalTCPPresets, filepath.Join(cfg, LocalInstallerFolder)) || f.ProgramFilesPresets != "Presets" {
		t.Errorf("%+v", f)
	}
}

func TestRemoveOldPresetFolder(t *testing.T) {
	if err := RemoveOldPresetFolder(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Errorf("missing folder: %v", err)
	}
	f := NewFolders(t.TempDir(), ProgramFilesPresetsFolder)
	old := f.LocalOldPresets
	if err := os.MkdirAll(filepath.Join(old, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"carlo gavazzi modbus.json", "x.json", filepath.Join("sub", "y.json")} {
		if err := os.WriteFile(filepath.Join(old, n), []byte("{}"), 0o400); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(f.LocalTCPPresets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RemoveOldPresetFolder(old); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old Presets folder must be gone")
	}
	if _, err := os.Stat(f.LocalTCPPresets); err != nil {
		t.Error("other preset folders must stay")
	}
}
