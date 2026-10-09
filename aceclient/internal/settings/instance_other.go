//go:build !windows && !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package settings

// tryLockFile has no portable implementation on this platform; the guard is
// a no-op (always acquired).
func tryLockFile(path string) (release func() error, busy bool, err error) {
	return func() error { return nil }, false, nil
}
