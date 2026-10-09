package settings

import (
	"errors"
	"image/color"
	"os"
	"path/filepath"
)

// Application names and file/folder names from AppProperties
// (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs) that a
// cross-platform port still needs. FTP site/credentials, the
// StatusDescription/ErrorCodes spreadsheets and the Segoe UI fonts are not
// ported (FTP is out of scope; the rest is unused or Windows-only).
const (
	// AppName ports AppProperties.AppName (window/dialog titles).
	AppName = "ACE Service Installer"
	// LocalInstallerFolder ports AppProperties.LocalInstallerFolder: the C#
	// per-user folder under %APPDATA% (Environment.SpecialFolder.ApplicationData).
	LocalInstallerFolder = "ACE Service Installer"

	// LocalFirmwareFolderName ports AppProperties.LocalFirmwareFolderName.
	LocalFirmwareFolderName = "Firmware"
	// LocalLogFolderName ports AppProperties.LocalLogFolderName.
	LocalLogFolderName = "Log"
	// LocalOldPresetsFolderName ports AppProperties.LocalOldPresetsFolderName.
	LocalOldPresetsFolderName = "Presets"
	// LocalTCPPresetsFolderName ports AppProperties.LocalTCPPresetsFolderName.
	LocalTCPPresetsFolderName = "TCPPresets"
	// LocalRTUPresetsFolderName ports AppProperties.LocalRTUPresetsFolderName.
	LocalRTUPresetsFolderName = "RTUPresets"
	// LocalBackofficePresetsFolderName ports AppProperties.LocalBackofficePresetsFolderName.
	LocalBackofficePresetsFolderName = "BackofficePresets"

	// ConfigFileNameOld ports AppProperties.ConfigFileNameOld.
	ConfigFileNameOld = "InstallerConfig.dat"
	// ConfigFileNameV2 ports AppProperties.ConfigFileNameV2.
	ConfigFileNameV2 = "InstallerConfigV2.dat"
	// ConfigFileName ports AppProperties.ConfigFileName (the encrypted
	// installer config read by ICUConfig.ReadInstallerSettings).
	ConfigFileName = "InstallerConfigV3.dat"
	// FallbackSettingsFilename ports AppProperties.FallbackSettingsFilename.
	FallbackSettingsFilename = "InstallerSettings.dat"

	// ProgramFilesPresetsFolder ports AppProperties.ProgramFilesPresetsFolder
	// (relative to the working directory / install folder).
	ProgramFilesPresetsFolder = "Presets"
	// UILanguagesFolder ports AppProperties.UILanguagesFolder (relative).
	UILanguagesFolder = "UILanguages"

	// IsahSite ports AppProperties.IsahSite (ISAH/IWS web service base URL).
	IsahSite = "https://installer.alfen.com"

	// DefaultUserName ports AppProperties.DefaultUserName: the DlgLogon user
	// name shown when LastUserName is empty.
	DefaultUserName = "User"

	// ButtonHeight / ButtonWidth port AppProperties.ButtonHeight/ButtonWidth
	// (pixels).
	ButtonHeight = 24
	ButtonWidth  = 120
)

// ColorBorder ports AppProperties.Color_Border (Color.FromBytes(171,173,179)).
var ColorBorder = color.NRGBA{R: 171, G: 173, B: 179, A: 255}

// ColorDisabled ports AppProperties.Color_Disabled (Color.FromBytes(250,250,250)).
var ColorDisabled = color.NRGBA{R: 250, G: 250, B: 250, A: 255}

// Paths ports the folder/file properties of AppProperties relative to a
// setup folder (AppProperties.LocalSetupFolder). In C# that folder is
// %APPDATA%\ACE Service Installer. DefaultPaths uses the same location
// (os.UserConfigDir()/"ACE Service Installer"), so on Windows the port finds
// the presets, downloaded firmware and InstallerConfigV3.dat of an existing
// installation, and RemoveObsoleteConfigs cleans the same files as
// InitializeSettings. Only the Go port's own settings.json lives elsewhere
// (DefaultDir).
//
// AppProperties.LocalLogFolder is not ported: it is unused and (sic) returns
// the Firmware folder. Logging uses LogFolder (MainClass) instead.
//
// ports AppProperties path properties (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs)
type Paths struct {
	// Root is the setup folder (AppProperties.LocalSetupFolder).
	Root string
}

// DefaultPaths returns Paths rooted at AppProperties.LocalSetupFolder:
// Environment.SpecialFolder.ApplicationData + LocalInstallerFolder, i.e.
// os.UserConfigDir()/"ACE Service Installer" (%AppData% on Windows).
//
// ports AppProperties.LocalSetupFolder (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs)
func DefaultPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return Paths{Root: filepath.Join(base, LocalInstallerFolder)}, nil
}

// LegacyPaths is the same as DefaultPaths (the C# LocalSetupFolder). It is
// kept for callers that name the C# location explicitly.
func LegacyPaths() (Paths, error) { return DefaultPaths() }

// SetupFolder ports AppProperties.LocalSetupFolder.
func (p Paths) SetupFolder() string { return p.Root }

// SettingsFilename ports AppProperties.SettingsFilename (InstallerConfigV3.dat).
func (p Paths) SettingsFilename() string { return filepath.Join(p.Root, ConfigFileName) }

// SettingsFilenameV2 ports AppProperties.SettingsFilenameV2.
func (p Paths) SettingsFilenameV2() string { return filepath.Join(p.Root, ConfigFileNameV2) }

// SettingsFilenameOld ports AppProperties.SettingsFilenameOld.
func (p Paths) SettingsFilenameOld() string { return filepath.Join(p.Root, ConfigFileNameOld) }

// FirmwareFolder ports AppProperties.LocalFirmwareFolder.
func (p Paths) FirmwareFolder() string { return filepath.Join(p.Root, LocalFirmwareFolderName) }

// OldPresetsFolder ports AppProperties.LocalOldPresetsFolder.
func (p Paths) OldPresetsFolder() string { return filepath.Join(p.Root, LocalOldPresetsFolderName) }

// TCPPresetsFolder ports AppProperties.LocalTCPPresetsFolder.
func (p Paths) TCPPresetsFolder() string { return filepath.Join(p.Root, LocalTCPPresetsFolderName) }

// RTUPresetsFolder ports AppProperties.LocalRTUPresetsFolder.
func (p Paths) RTUPresetsFolder() string { return filepath.Join(p.Root, LocalRTUPresetsFolderName) }

// BackofficePresetsFolder ports AppProperties.LocalBackofficePresetsFolder.
func (p Paths) BackofficePresetsFolder() string {
	return filepath.Join(p.Root, LocalBackofficePresetsFolderName)
}

// InstallerConfigCandidates ports the read order of
// AppProperties.InitializeSettings: first SettingsFilename (setup folder);
// if that fails, ConfigFileName relative to the working directory. Both are
// read with ICUEncryptionType.encryptRijndaelHashed by ICUConfig (not here).
//
// ports AppProperties.InitializeSettings (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs)
func (p Paths) InstallerConfigCandidates() []string {
	return []string{p.SettingsFilename(), ConfigFileName}
}

// RemoveObsoleteConfigs ports the clean-up half of
// AppProperties.InitializeSettings: File.Delete(SettingsFilenameOld) and
// File.Delete(SettingsFilenameV2). File.Delete ignores missing files, and
// the C# only logs other failures at Debug level and carries on; here those
// failures are returned (joined) so the caller can log them.
//
// ports AppProperties.InitializeSettings (ACEServiceInstaller/ICUServiceInstaller/AppProperties.cs)
func (p Paths) RemoveObsoleteConfigs() error {
	var errs []error
	for _, f := range []string{p.SettingsFilenameOld(), p.SettingsFilenameV2()} {
		if err := os.Remove(f); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
