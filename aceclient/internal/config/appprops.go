package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Names from ICUServiceInstaller.AppProperties
// (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs) that this
// package needs for config-file selection, the logon dialog and the window
// title. internal/settings owns the other AppProperties folders and the
// user-scoped Properties/Settings.cs store (LastUserName, LocalPasswords).
const (
	AppName                  = "ACE Service Installer"
	LocalInstallerFolder     = "ACE Service Installer"
	ConfigFileNameOld        = "InstallerConfig.dat"
	ConfigFileNameV2         = "InstallerConfigV2.dat"
	ConfigFileName           = "InstallerConfigV3.dat"
	FallbackSettingsFilename = "InstallerSettings.dat" // also ICUConfig's default file name
	DefaultUserName          = "User"
)

// Paths resolves the AppProperties config-file locations under a base
// directory that stands in for Environment.SpecialFolder.ApplicationData.
// LocalSetupFolder() is the folder that settings.LegacyPaths() roots at.
type Paths struct {
	AppData string
}

// DefaultPaths uses os.UserConfigDir() as ApplicationData (%AppData% on
// Windows, ~/Library/Application Support on macOS, $XDG_CONFIG_HOME on Linux).
func DefaultPaths() (Paths, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("config: %w", err)
	}
	return Paths{AppData: d}, nil
}

// LocalSetupFolder ports AppProperties.LocalSetupFolder.
func (p Paths) LocalSetupFolder() string { return filepath.Join(p.AppData, LocalInstallerFolder) }

// SettingsFilename ports AppProperties.SettingsFilename (the V3 config in AppData).
func (p Paths) SettingsFilename() string { return filepath.Join(p.LocalSetupFolder(), ConfigFileName) }

// SettingsFilenameV2 ports AppProperties.SettingsFilenameV2.
func (p Paths) SettingsFilenameV2() string {
	return filepath.Join(p.LocalSetupFolder(), ConfigFileNameV2)
}

// SettingsFilenameOld ports AppProperties.SettingsFilenameOld.
func (p Paths) SettingsFilenameOld() string {
	return filepath.Join(p.LocalSetupFolder(), ConfigFileNameOld)
}

// InitializeSettings ports AppProperties.InitializeSettings
// (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs:103-124, called
// from the MainWindow constructor). It first deletes the obsolete
// InstallerConfig.dat and InstallerConfigV2.dat from the setup folder. Errors
// are ignored, because the C# only logs them. It then reads the V3 config from
// the setup folder and, if that fails, InstallerConfigV3.dat from installDir.
// The C# uses the working directory, which is the installation folder. Both
// reads use EncryptRijndaelHashed.
//
// It returns the config and the file it was read from. The C# reads both
// files into the same static AppProperties.ICUConfig, and
// ReadInstallerSettings does not roll back on failure. So when both reads
// fail, the returned config keeps whatever the reads left in it, together
// with the joined errors and an empty file name:
//   - When the fallback fails before its JSON is parsed (for example, the
//     file is missing), the state from the AppData file remains. Users and
//     groups read before a late error still allow a logon.
//   - When the fallback fails later, its own partial state replaces the
//     collections.
//   - When no read got that far, the config is empty and every logon reports
//     an unknown user.
func InitializeSettings(p Paths, installDir string) (*Config, string, error) {
	_ = os.Remove(p.SettingsFilenameOld())
	_ = os.Remove(p.SettingsFilenameV2())
	cfg := NewConfig() // AppProperties.ICUConfig = new ICUConfig()
	primary := p.SettingsFilename()
	err1 := cfg.ReadInstallerSettings(primary, EncryptRijndaelHashed)
	if err1 == nil {
		return cfg, primary, nil
	}
	fallback := filepath.Join(installDir, ConfigFileName)
	err2 := cfg.ReadInstallerSettings(fallback, EncryptRijndaelHashed)
	if err2 == nil {
		return cfg, fallback, nil
	}
	return cfg, "", errors.Join(err1, err2)
}
