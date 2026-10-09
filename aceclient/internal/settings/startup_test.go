package settings

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		args []string
		want StartupOptions
	}{
		{nil, StartupOptions{AoNormal, false}},
		{[]string{}, StartupOptions{AoNormal, false}},
		{[]string{"large"}, StartupOptions{AoLarge, false}},
		{[]string{"-large"}, StartupOptions{AoLarge, false}},
		{[]string{"--LARGE"}, StartupOptions{AoLarge, false}},
		{[]string{" Large- "}, StartupOptions{AoLarge, false}},
		{[]string{"/large"}, StartupOptions{AoNormal, false}},
		{[]string{"larger"}, StartupOptions{AoNormal, false}},
		{[]string{"multimode"}, StartupOptions{AoNormal, true}},
		{[]string{"MultiMode"}, StartupOptions{AoNormal, false}}, // exact, case-sensitive
		{[]string{"-multimode"}, StartupOptions{AoNormal, false}},
		{[]string{"multimode", "large"}, StartupOptions{AoNormal, true}}, // only args[0]
		{[]string{"x", "large"}, StartupOptions{AoNormal, false}},
	}
	for _, c := range cases {
		if got := ParseArgs(c.args); got != c.want {
			t.Errorf("ParseArgs(%q) = %+v, want %+v", c.args, got, c.want)
		}
	}
}

func TestAppOptionsString(t *testing.T) {
	if AoNormal.String() != "aoNormal" || AoLarge.String() != "aoLarge" || int(AoNormal) != 0 || int(AoLarge) != 1 {
		t.Error("AppOptions names/values differ from the C# enum")
	}
	if AppOptions(7).String() != "AppOptions(7)" {
		t.Error("unknown value formatting")
	}
}

func TestStartupConstantsMatchSource(t *testing.T) {
	src := readFixture(t, "ACEServiceInstaller/ACEServiceInstaller/MainClass.cs")
	for _, want := range []string{
		`_appName = "` + InstanceName + `"`,
		`TimeSpan.FromSeconds(2.0)`,
		`args[0] == "` + MultiModeArg + `"`,
		`Trim(new char[2] { ' ', '-' }) == "` + LargeArg + `"`,
		`"` + MsgAnotherInstance + `"`,
		`"` + LogFolderName + `"`,
		`"` + MsgLogFolderFailed + `"`,
		`service_installer_{DateTime.Now:yyyyMMdd-HHmmss}.json`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("MainClass.cs lacks %s", want)
		}
	}
	if InstanceWait != 2*time.Second {
		t.Error("wait differs")
	}
}

func TestSingleInstance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "lock")
	l1, err := AcquireSingleInstance(dir, InstanceWait)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(l1.Path()) != InstanceName+".lock" {
		t.Errorf("lock path %s", l1.Path())
	}
	start := time.Now()
	if _, err := AcquireSingleInstance(dir, 150*time.Millisecond); !errors.Is(err, ErrAnotherInstance) {
		t.Fatalf("second acquire: %v, want ErrAnotherInstance", err)
	} else if time.Since(start) < 150*time.Millisecond {
		t.Error("did not wait before giving up")
	}
	if ErrAnotherInstance.Error() != MsgAnotherInstance {
		t.Error("message differs")
	}
	if err := l1.Release(); err != nil {
		t.Fatal(err)
	}
	if err := l1.Release(); err != nil {
		t.Fatalf("double release: %v", err)
	}
	l2, err := AcquireSingleInstance(dir, 0)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	_ = l2.Release()

	// A lock released while the second caller waits is picked up.
	l3, _ := AcquireSingleInstance(dir, 0)
	go func() { time.Sleep(100 * time.Millisecond); _ = l3.Release() }()
	l4, err := AcquireSingleInstance(dir, 2*time.Second)
	if err != nil {
		t.Fatalf("waiting acquire: %v", err)
	}
	_ = l4.Release()
}

// The lock must survive the caller dropping the *InstanceLock (the C# keeps
// its Mutex in a static field): without the package-level reference the
// *os.File finalizer would close the descriptor and drop the flock.
func TestSingleInstanceSurvivesGC(t *testing.T) {
	if runtime.GOOS == "js" || runtime.GOOS == "wasip1" {
		t.Skip("no file locking on this platform")
	}
	dir := filepath.Join(t.TempDir(), "lock")
	func() {
		if _, err := AcquireSingleInstance(dir, 0); err != nil {
			t.Fatal(err)
		}
	}()
	for range 3 {
		runtime.GC()
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := AcquireSingleInstance(dir, 0); !errors.Is(err, ErrAnotherInstance) {
		t.Fatalf("lock lost after GC: %v", err)
	}
	// Release the dropped lock through the package-level registry.
	heldLocks.Lock()
	var mine []*InstanceLock
	for l := range heldLocks.m {
		if filepath.Dir(l.Path()) == dir {
			mine = append(mine, l)
		}
	}
	heldLocks.Unlock()
	if len(mine) != 1 {
		t.Fatalf("held locks for dir: %d", len(mine))
	}
	if err := mine[0].Release(); err != nil {
		t.Fatal(err)
	}
	l, err := AcquireSingleInstance(dir, 0)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	_ = l.Release()
	heldLocks.Lock()
	_, still := heldLocks.m[l]
	heldLocks.Unlock()
	if still {
		t.Error("released lock still registered")
	}
}

func setTempDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	return tmp
}

func TestLogFolderAndFileName(t *testing.T) {
	setTempDir(t)
	dir, err := LogFolder()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != LogFolderName {
		t.Errorf("log folder %s", dir)
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		t.Fatalf("log folder not created: %v", err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o700 {
		t.Errorf("log folder perm %v, want 0700", fi.Mode().Perm())
	}
	// An existing folder is reused (and tightened on Unix).
	if runtime.GOOS != "windows" {
		_ = os.Chmod(dir, 0o755)
	}
	if dir2, err := LogFolder(); err != nil || dir2 != dir {
		t.Errorf("reuse: %s %v", dir2, err)
	}
	if fi, _ := os.Stat(dir); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o700 {
		t.Errorf("existing folder not tightened: %v", fi.Mode().Perm())
	}
	got := LogFileName(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if got != "service_installer_20260102-030405.json" {
		t.Errorf("LogFileName = %s", got)
	}
}

// A planted symlink (or plain file) instead of the folder disables logging
// with the C# message, which ends in the temp path plus a separator.
func TestLogFolderRejectsNonDirectory(t *testing.T) {
	cases := []struct {
		name  string
		plant func(path, target string) error
	}{
		{"file", func(path, _ string) error { return os.WriteFile(path, nil, 0o600) }},
		{"symlink", func(path, target string) error { return os.Symlink(target, path) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmp := setTempDir(t)
			target := t.TempDir()
			if err := c.plant(filepath.Join(tmp, LogFolderName), target); err != nil {
				t.Skip(err)
			}
			_, err := LogFolder()
			want := MsgLogFolderFailed + withTrailingSeparator(os.TempDir())
			if err == nil || err.Error() != want || !strings.HasSuffix(err.Error(), string(filepath.Separator)) {
				t.Errorf("err = %v, want %q", err, want)
			}
		})
	}
}

func TestWithTrailingSeparator(t *testing.T) {
	sep := string(filepath.Separator)
	for in, want := range map[string]string{
		"":                            "",
		filepath.Join("a", "b"):       filepath.Join("a", "b") + sep,
		filepath.Join("a", "b") + sep: filepath.Join("a", "b") + sep,
	} {
		if got := withTrailingSeparator(in); got != want {
			t.Errorf("withTrailingSeparator(%q) = %q, want %q", in, got, want)
		}
	}
}
