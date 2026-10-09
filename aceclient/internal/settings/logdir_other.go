//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package settings

import (
	"errors"
	"os"
)

// checkPrivateDir verifies that dir is a real directory. On Windows the temp
// folder (Path.GetTempPath()) is already per-user, as in the C#.
func checkPrivateDir(dir string) error {
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return errors.New("settings: " + dir + " is not a directory")
	}
	return nil
}
