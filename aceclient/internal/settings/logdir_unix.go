//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package settings

import (
	"errors"
	"os"
	"syscall"
)

// checkPrivateDir verifies that dir is a real directory (not a symlink) owned
// by the current user, then restricts it to 0700. This guards the shared
// /tmp on Linux against a folder planted by another user (see LogFolder).
func checkPrivateDir(dir string) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return errors.New("settings: " + dir + " is not a directory")
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || int(st.Uid) != os.Geteuid() {
		return errors.New("settings: " + dir + " is not owned by the current user")
	}
	if fi.Mode().Perm() != 0o700 {
		return os.Chmod(dir, 0o700)
	}
	return nil
}
