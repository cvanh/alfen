// Package settings ports the ACE Service Installer's application settings,
// startup options and connection persistence:
//
//   - ICUServiceInstaller.Properties.Settings (Settings.cs / app.config): every
//     user-scoped setting key with its C# type and DefaultSettingValue, except
//     the AppInsights telemetry connection string (out of scope).
//   - AppProperties (AppProperties.cs): names, folders and file names that a
//     cross-platform port still needs (see Paths).
//   - AppOptions / App / ACEServiceInstaller.MainClass: start-up arguments
//     ("large", "multimode"), the single-instance guard and the log folder.
//   - DlgSettings, DlgManualIP, DlgLogon (LocalPasswords / LastUserName) and
//     DlgDeviceLogin (the "store password" mechanism): the pure logic behind
//     those dialogs, extracted from the Xwt widgets.
//   - LANConnection.AddManualDevice / RemoveManualDevice + MainWindow
//     OnAddManualDeviceClicked: manual-device duplicate rules.
//
// Persistence is Go-native: one JSON document (FileName) in a per-user
// directory (DefaultDir, injectable for tests), written atomically (temp
// file + rename) with 0600 permissions. The JSON keys of the ported settings
// are the C# setting names, so they map 1:1 onto user.config. Missing keys
// load as their C# defaults, like ApplicationSettingsBase does.
//
// Passwords are remembered exactly as the C# does it and nowhere else: a
// single, global "last device user level + password" slot (LastDeviceUsername
// / LastDevicePassword), only while the user ticks "store password" in the
// device-login dialog, stored in plain text (user.config is plain XML; this
// file is plain JSON with 0600 permissions). Connection profiles and manual
// devices never hold a password. See devicelogin.go.
//
// Go-native additions that have no C# counterpart (and are documented as
// such): connection Profiles, LastDevice and the persisted ManualDevices list
// (the C# keeps manually added devices in memory for the session only and
// persists just the last dialog entry, LastManual*).
//
// No Fyne imports: a thin view calls these functions and renders the results.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// DirName is the per-user folder (under os.UserConfigDir) that holds the Go
// port's settings file. The C# installer keeps its data files (presets,
// firmware, InstallerConfigV3.dat) in AppProperties.LocalInstallerFolder
// ("ACE Service Installer") under %APPDATA%; DefaultPaths points there.
const DirName = "ACEServiceInstaller"

// FileName is the settings document inside the settings directory.
const FileName = "settings.json"

// ErrCorrupt is matched (errors.Is) by the *CorruptFileError Open returns when
// the settings file exists but cannot be decoded. The C# ApplicationSettingsBase
// throws ConfigurationErrorsException in that case and never overwrites the
// file. Open instead moves the undecodable file aside (see CorruptFileError)
// and returns a usable Store holding the defaults, so the next save cannot
// destroy the user's data.
var ErrCorrupt = errors.New("settings: corrupt settings file")

// CorruptFileError is returned by Open, together with a usable defaults Store,
// when the settings file cannot be decoded (one wrongly typed field is
// enough). errors.Is(err, ErrCorrupt) reports true.
type CorruptFileError struct {
	// Path is the settings file.
	Path string
	// Backup is where the undecodable file was moved
	// ("<Path>.corrupt-<unix seconds>"), or "" when the move failed. In that
	// case Err includes the move error, and the next save replaces the file.
	Backup string
	// Err is the decode error (joined with the move error, if any).
	Err error
}

func (e *CorruptFileError) Error() string {
	msg := ErrCorrupt.Error() + ": " + e.Path + ": " + e.Err.Error()
	if e.Backup != "" {
		msg += " (moved to " + e.Backup + ")"
	}
	return msg
}

// Is makes errors.Is(err, ErrCorrupt) true.
func (e *CorruptFileError) Is(target error) bool { return target == ErrCorrupt }

// Unwrap returns the decode (and move) error.
func (e *CorruptFileError) Unwrap() error { return e.Err }

// backupCorrupt renames path to a free "<path>.corrupt-<unix seconds>[-n]" name.
func backupCorrupt(path string, now time.Time) (string, error) {
	base := fmt.Sprintf("%s.corrupt-%d", path, now.Unix())
	for i := 0; i < 1000; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := os.Lstat(name); errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(path, name); err != nil {
				return "", err
			}
			return name, nil
		}
	}
	return "", fmt.Errorf("settings: no free backup name for %s", path)
}

// Settings holds every persisted value.
//
// The first block ports ICUServiceInstaller.Properties.Settings field by
// field (ICUServiceInstaller.Properties/Settings.cs): same names (also used as
// JSON keys), same types (string, int, bool, DateTime -> time.Time,
// StringCollection -> []string) and the same defaults (see Defaults).
// AppInsightsConnStrEncrypted is deliberately not ported (telemetry).
//
// ports ICUServiceInstaller.Properties.Settings (ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs)
type Settings struct {
	// LastUserName is the installer (app logon) user name typed in DlgLogon.
	// MainWindow.OnLoginRequest copies it into the device-login DisplayName
	// after each login (see SessionCredentials.DisplayName).
	LastUserName string `json:"LastUserName"`
	// LocalPasswords holds DlgLogon's locally changed app-logon passwords,
	// one "<hashed user>;<ICUUser.Json>" string each (see logon.go). Default:
	// null (no DefaultSettingValue).
	LocalPasswords []string `json:"LocalPasswords"`
	// LastManualIPAddress .. LastManualModelType remember the last accepted
	// DlgManualIP entry (see manualip.go).
	LastManualIPAddress string `json:"LastManualIPAddress"`
	LastManualIPPort    int    `json:"LastManualIPPort"`
	// LastCommand pre-fills DlgCommand. The C# only ever reads it (nothing
	// writes it), so it stays at its default unless set via Update.
	LastCommand string `json:"LastCommand"`
	// AskConfirmExit / AskSaveChanges are the two DlgSettings check boxes.
	AskConfirmExit bool `json:"AskConfirmExit"`
	AskSaveChanges bool `json:"AskSaveChanges"`
	// LastFTPActivity / LastFTPConnectionWarning drive the FTP update-server
	// reachability warning (MainWindow.UpdateManager_CanConnect). Kept for
	// key completeness; the FTP UpdateManager is not ported.
	LastFTPActivity          time.Time `json:"LastFTPActivity"`
	LastFTPConnectionWarning time.Time `json:"LastFTPConnectionWarning"`
	// LastDevicePassword / LastDeviceUsername / StorePasswords /
	// LastPasswordStoreTime are the DlgDeviceLogin "store password" slot
	// (see devicelogin.go). LastDevicePassword is plain text, as in C#.
	LastDevicePassword    string    `json:"LastDevicePassword"`
	LastDeviceUsername    string    `json:"LastDeviceUsername"`
	StorePasswords        bool      `json:"StorePasswords"`
	LastPasswordStoreTime time.Time `json:"LastPasswordStoreTime"`
	// LastManualHostname is declared in Settings.cs but never read or written
	// by the installer; kept for key completeness.
	LastManualHostname        string `json:"LastManualHostname"`
	LastManualNumberOfSockets int    `json:"LastManualNumberOfSockets"`
	// LastManualLoginRequired is written by DlgManualIP but never read (the
	// dialog always starts with "Login required" ticked).
	LastManualLoginRequired bool   `json:"LastManualLoginRequired"`
	LastManualModelType     string `json:"LastManualModelType"`

	// ---- Go-native additions (no C# counterpart) ----

	// Profiles are named connection profiles (no passwords). See profiles.go.
	Profiles []Profile `json:"Profiles,omitempty"`
	// LastDevice is the device the GUI last connected to (no password).
	LastDevice *Profile `json:"LastDevice,omitempty"`
	// ManualDevices persists manually added devices across restarts. The C#
	// LANConnection keeps them in memory only. See manualip.go.
	ManualDevices []ManualDevice `json:"ManualDevices,omitempty"`
}

// defaultDate is DefaultSettingValue("2020-01-01") for the DateTime settings
// (parsed by .NET as a local, Kind=Unspecified date).
func defaultDate() time.Time { return time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local) }

// Defaults returns the C# DefaultSettingValue of every ported setting
// (ICUServiceInstaller.Properties/Settings.cs, mirrored in app.config).
//
// ports Settings [DefaultSettingValue] attributes (ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs, ACEServiceInstaller/app.config)
func Defaults() Settings {
	return Settings{
		LastUserName:              "",
		LocalPasswords:            nil,
		LastManualIPAddress:       "",
		LastManualIPPort:          443,
		LastCommand:               "",
		AskConfirmExit:            true,
		AskSaveChanges:            true,
		LastFTPActivity:           defaultDate(),
		LastFTPConnectionWarning:  defaultDate(),
		LastDevicePassword:        "",
		LastDeviceUsername:        "",
		StorePasswords:            false,
		LastPasswordStoreTime:     defaultDate(),
		LastManualHostname:        "",
		LastManualNumberOfSockets: 2,
		LastManualLoginRequired:   true,
		LastManualModelType:       "",
	}
}

// clone returns a deep copy so callers never alias the store's slices.
func (s Settings) clone() Settings {
	c := s
	c.LocalPasswords = slices.Clone(s.LocalPasswords)
	c.Profiles = slices.Clone(s.Profiles)
	c.ManualDevices = slices.Clone(s.ManualDevices)
	if s.LastDevice != nil {
		ld := *s.LastDevice
		c.LastDevice = &ld
	}
	return c
}

// DefaultDir returns os.UserConfigDir()/ACEServiceInstaller.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, DirName), nil
}

// Store is the thread-safe, file-backed equivalent of Settings.Default
// (ApplicationSettingsBase wrapped in SettingsBase.Synchronized). All methods
// may be called from any goroutine.
//
// ports Settings.Default (ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs)
type Store struct {
	dir  string
	path string

	mu sync.RWMutex
	s  Settings
}

// Open loads dir/settings.json. A missing file yields Defaults() (nothing is
// written until the first change). Keys missing from the file keep their
// defaults. Any other read error returns a nil Store.
//
// If the file cannot be decoded, Open moves it to a backup name and returns a
// Store holding Defaults() together with a *CorruptFileError (matching
// ErrCorrupt) that names the backup.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("settings: empty directory")
	}
	st := &Store{dir: dir, path: filepath.Join(dir, FileName), s: Defaults()}
	data, err := os.ReadFile(st.path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: read %s: %w", st.path, err)
	}
	loaded := Defaults()
	if err := json.Unmarshal(data, &loaded); err != nil {
		cerr := &CorruptFileError{Path: st.path, Err: err}
		backup, mvErr := backupCorrupt(st.path, time.Now())
		if mvErr != nil {
			cerr.Err = errors.Join(err, fmt.Errorf("settings: back up corrupt file: %w", mvErr))
		}
		cerr.Backup = backup
		return st, cerr
	}
	st.s = loaded
	return st, nil
}

// OpenDefault opens the store in DefaultDir().
func OpenDefault() (*Store, error) {
	dir, err := DefaultDir()
	if err != nil {
		return nil, err
	}
	return Open(dir)
}

// Dir returns the settings directory.
func (st *Store) Dir() string { return st.dir }

// Path returns the settings file path.
func (st *Store) Path() string { return st.path }

// Get returns a deep copy of the current settings.
func (st *Store) Get() Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s.clone()
}

// Update applies fn to the settings and saves them, mirroring the C# pattern
// "Settings.Default.X = v; Settings.Default.Save();". As in C#, the in-memory
// change stands even when the save fails; the save error is returned.
//
// ports Settings.Default[...] = v; SettingsBase.Save() (ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs)
func (st *Store) Update(fn func(*Settings)) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	fn(&st.s)
	return st.saveLocked()
}

// Save writes the current settings to disk (Settings.Default.Save()).
//
// ports SettingsBase.Save() on Settings.Default (ACEServiceInstaller/ICUServiceInstaller.Properties/Settings.cs)
func (st *Store) Save() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.saveLocked()
}

// saveLocked serialises and atomically replaces the settings file. The
// caller holds st.mu (write lock), which also orders concurrent saves.
func (st *Store) saveLocked() error {
	data, err := json.MarshalIndent(st.s, "", "  ")
	if err != nil {
		return fmt.Errorf("settings: encode: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(st.dir, 0o700); err != nil {
		return fmt.Errorf("settings: create %s: %w", st.dir, err)
	}
	if err := writeFileAtomic(st.path, data, 0o600); err != nil {
		return fmt.Errorf("settings: write %s: %w", st.path, err)
	}
	return nil
}

// writeFileAtomic writes data to a temp file in the target directory, syncs
// it and renames it over path, so readers see either the old or the new file.
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
