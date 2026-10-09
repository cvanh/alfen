package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AppOptions ports the ICUServiceInstaller.AppOptions enum
// (ACEServiceInstaller/ICUServiceInstaller/AppOptions.cs). MainClass passes it
// to App.Run, which hands it to MainWindow (stored in m_aoOptions; the
// decompiled source never reads it back).
type AppOptions int

// AppOptions values (same names and values as the C# enum).
const (
	AoNormal AppOptions = 0 // aoNormal
	AoLarge  AppOptions = 1 // aoLarge
)

// String returns the C# enum member name.
func (o AppOptions) String() string {
	switch o {
	case AoNormal:
		return "aoNormal"
	case AoLarge:
		return "aoLarge"
	}
	return fmt.Sprintf("AppOptions(%d)", int(o))
}

// Start-up constants from ACEServiceInstaller.MainClass
// (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs).
const (
	// InstanceName ports MainClass._appName, the name of the single-instance
	// Mutex ("ACE Service Installer 4.0").
	InstanceName = "ACE Service Installer 4.0"
	// InstanceWait ports _mutex.WaitOne(TimeSpan.FromSeconds(2.0)).
	InstanceWait = 2 * time.Second
	// MultiModeArg is the exact (case-sensitive) args[0] that skips the
	// single-instance check.
	MultiModeArg = "multimode"
	// LargeArg is args[0] after ToLowerInvariant().Trim(' ', '-') that
	// selects AoLarge (so "-large", "--LARGE" and " large " all match).
	LargeArg = "large"
	// MsgAnotherInstance is the error logged when the mutex is held.
	MsgAnotherInstance = "Another instance of the app is running. Bye!"
	// LogFolderName ports the "SiaLogs" folder of GetOrCreateLogFolder.
	LogFolderName = "SiaLogs"
	// MsgLogFolderFailed prefixes the message GetOrCreateLogFolder shows when
	// the log folder cannot be created (followed by Path.GetTempPath(), which
	// ends in a directory separator).
	MsgLogFolderFailed = "Logging not enabled, could not find or create a folder at location: "
)

// ErrAnotherInstance is returned by AcquireSingleInstance when another
// process holds the instance lock for longer than the wait time.
var ErrAnotherInstance = errors.New(MsgAnotherInstance)

// StartupOptions is the result of ParseArgs.
type StartupOptions struct {
	// Options is AoLarge when args[0] is "large" (see LargeArg).
	Options AppOptions
	// MultiMode skips the single-instance guard (args[0] == "multimode").
	MultiMode bool
}

// ParseArgs ports the argument handling of MainClass.Main. Only args[0] is
// inspected (args excludes the program name, as in C#'s Main(string[])):
//
//	flag    = args.Length != 0 && args[0] == "multimode"
//	options = args[0].ToLowerInvariant().Trim(' ', '-') == "large" ? aoLarge : aoNormal
//
// ports MainClass.Main (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs)
func ParseArgs(args []string) StartupOptions {
	var o StartupOptions
	if len(args) == 0 {
		return o
	}
	o.MultiMode = args[0] == MultiModeArg
	if strings.Trim(strings.ToLower(args[0]), " -") == LargeArg {
		o.Options = AoLarge
	}
	return o
}

// InstanceLock is a held single-instance lock (the C# named Mutex).
//
// ports MainClass._mutex (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs)
type InstanceLock struct {
	path    string
	release func() error
}

// heldLocks keeps every acquired InstanceLock reachable until Release, as the
// C# keeps its Mutex in the static field MainClass._mutex. Without it, a
// caller that drops the *InstanceLock would lose the lock on the next GC
// (the *os.File finalizer closes the descriptor, which drops the flock).
var heldLocks = struct {
	sync.Mutex
	m map[*InstanceLock]struct{}
}{m: map[*InstanceLock]struct{}{}}

// Path returns the lock file path.
func (l *InstanceLock) Path() string { return l.path }

// Release frees the lock (_mutex.ReleaseMutex()). It is safe to call more
// than once.
func (l *InstanceLock) Release() error {
	if l == nil {
		return nil
	}
	heldLocks.Lock()
	r := l.release
	l.release = nil
	delete(heldLocks.m, l)
	heldLocks.Unlock()
	if r == nil {
		return nil
	}
	return r()
}

// AcquireSingleInstance ports MainClass's single-instance guard: the C#
// opens Mutex(false, "ACE Service Installer 4.0") and gives up with
// MsgAnotherInstance when WaitOne(2 s) fails. The Go port locks the file
// dir/"ACE Service Installer 4.0.lock" instead (flock on Unix, an unshared
// CreateFile handle on Windows), retrying until wait elapses, then returns
// ErrAnotherInstance. The OS drops the lock if the process dies. The lock is
// held until Release or process exit, even if the caller drops the returned
// value (like the static C# Mutex). Callers skip this when
// StartupOptions.MultiMode is set, as the C# does.
//
// ports MainClass.Main mutex guard (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs)
func AcquireSingleInstance(dir string, wait time.Duration) (*InstanceLock, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, InstanceName+".lock")
	deadline := time.Now().Add(wait)
	for {
		release, busy, err := tryLockFile(path)
		if err != nil {
			return nil, err
		}
		if !busy {
			l := &InstanceLock{path: path, release: release}
			heldLocks.Lock()
			heldLocks.m[l] = struct{}{}
			heldLocks.Unlock()
			return l, nil
		}
		if !time.Now().Before(deadline) {
			return nil, ErrAnotherInstance
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// LogFolder ports MainClass.GetOrCreateLogFolder: Path.Combine(
// Path.GetTempPath(), "SiaLogs"), created when missing. On failure the C#
// shows MsgLogFolderFailed + Path.GetTempPath() (which ends in a directory
// separator) and runs without file logging; here the same message is
// returned as the error.
//
// Hardening (Go-native): Path.GetTempPath() is per-user on Windows, but
// os.TempDir() is the shared /tmp on Linux. So the folder is created with
// mode 0700, and on Unix an existing folder must be a real directory (not a
// symlink) owned by the current user; it is then tightened to 0700. A folder
// that fails this check is treated like one that cannot be created.
//
// ports MainClass.GetOrCreateLogFolder (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs)
func LogFolder() (string, error) {
	tmp := os.TempDir()
	dir := filepath.Join(tmp, LogFolderName)
	failed := errors.New(MsgLogFolderFailed + withTrailingSeparator(tmp))
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", failed
	}
	if err := checkPrivateDir(dir); err != nil {
		return "", failed
	}
	return dir, nil
}

// withTrailingSeparator mirrors Path.GetTempPath(), which always ends in a
// directory separator.
func withTrailingSeparator(dir string) string {
	if dir == "" || os.IsPathSeparator(dir[len(dir)-1]) {
		return dir
	}
	return dir + string(filepath.Separator)
}

// LogFileName ports the file name MainClass passes to the Serilog file sink:
// $"service_installer_{DateTime.Now:yyyyMMdd-HHmmss}.json". That is the base
// path only. The sink uses rollingInterval: RollingInterval.Day, so Serilog
// inserts the day (yyyyMMdd) before ".json", plus "_NNN" on size rolls; the
// file on disk is service_installer_<yyyyMMdd-HHmmss><yyyyMMdd>.json. File
// logging itself is out of scope for the port.
//
// ports MainClass.Main Serilog file sink path (ACEServiceInstaller/ACEServiceInstaller/MainClass.cs)
func LogFileName(t time.Time) string {
	return "service_installer_" + t.Format("20060102-150405") + ".json"
}
