package presets

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Preset folder names of ICUServiceInstaller.AppProperties
// (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs).
const (
	AppName = "ACE Service Installer" // AppProperties.AppName

	LocalInstallerFolder             = "ACE Service Installer" // under %APPDATA%
	LocalOldPresetsFolderName        = "Presets"               // legacy, removed by RemoveOldPresetFolder
	LocalTCPPresetsFolderName        = "TCPPresets"            // Modbus TCP meter presets (*.json)
	LocalRTUPresetsFolderName        = "RTUPresets"            // Modbus RTU meter presets (*.json)
	LocalBackofficePresetsFolderName = "BackofficePresets"     // backoffice presets (*.fwi/*.tcf)

	// FTP folder names the local folders are synchronised from (the FTP
	// UpdateManager itself is not ported).
	FTPTCPPresetsFolder        = "TCPPresets"
	FTPRTUPresetsFolder        = "RTUPresets"
	FTPBackofficePresetsFolder = "BackofficePresets"

	// ProgramFilesPresetsFolder is AppProperties.ProgramFilesPresetsFolder:
	// the relative path "Presets", i.e. next to the installed executable when
	// started from its install directory (the process working directory).
	ProgramFilesPresetsFolder = "Presets"
)

// Folders holds the resolved preset folders (the AppProperties
// Local*PresetsFolder / ProgramFilesPresetsFolder properties). Every path is
// injectable; use NewFolders or DefaultFolders to fill it like the C#.
type Folders struct {
	ProgramFilesPresets    string // AppProperties.ProgramFilesPresetsFolder
	LocalOldPresets        string // AppProperties.LocalOldPresetsFolder
	LocalTCPPresets        string // AppProperties.LocalTCPPresetsFolder
	LocalRTUPresets        string // AppProperties.LocalRTUPresetsFolder
	LocalBackofficePresets string // AppProperties.LocalBackofficePresetsFolder
}

// NewFolders ports the AppProperties folder properties:
// Path.Combine(applicationData, "ACE Service Installer", <name>) for the
// local folders, and programFilesPresets for ProgramFilesPresetsFolder
// (pass ProgramFilesPresetsFolder for the C# relative default).
func NewFolders(applicationData, programFilesPresets string) Folders {
	base := filepath.Join(applicationData, LocalInstallerFolder)
	return Folders{
		ProgramFilesPresets:    programFilesPresets,
		LocalOldPresets:        filepath.Join(base, LocalOldPresetsFolderName),
		LocalTCPPresets:        filepath.Join(base, LocalTCPPresetsFolderName),
		LocalRTUPresets:        filepath.Join(base, LocalRTUPresetsFolderName),
		LocalBackofficePresets: filepath.Join(base, LocalBackofficePresetsFolderName),
	}
}

// DefaultFolders is NewFolders with os.UserConfigDir() in the role of
// Environment.SpecialFolder.ApplicationData (on Windows both are %APPDATA%)
// and the relative ProgramFilesPresetsFolder.
func DefaultFolders() (Folders, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return Folders{}, err
	}
	return NewFolders(dir, ProgramFilesPresetsFolder), nil
}

// PresetListFolders returns the folders DlgPresets.FillPresetList scans, in
// order: ProgramFiles, TCP, RTU, Backoffice.
func (f Folders) PresetListFolders() []string {
	return []string{f.ProgramFilesPresets, f.LocalTCPPresets, f.LocalRTUPresets, f.LocalBackofficePresets}
}

// RemoveOldPresetFolder ports MainWindow.RemoveOldPresetFolder: the legacy
// %APPDATA%\ACE Service Installer\Presets folder (Modbus meter presets from
// before the TCP/RTU split) is deleted. Nothing happens when it does not
// exist. Each top-level file is deleted first; an I/O failure there is only
// logged in the C# (ignored here), while an access-denied failure aborts
// (UnauthorizedAccessException is not an IOException). Finally the folder is
// deleted recursively and that error is returned.
//
// In the C# this runs from CheckForUpdates_Settings, only when the FTP site
// was reachable, before the TCP/RTU preset download check.
func RemoveOldPresetFolder(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil // DirectoryInfo.Exists == false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue // DirectoryInfo.GetFiles lists files only
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && errors.Is(err, fs.ErrPermission) {
			return err
		}
	}
	return os.RemoveAll(dir)
}
