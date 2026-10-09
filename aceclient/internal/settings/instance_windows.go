//go:build windows

package settings

import (
	"syscall"
)

// errorSharingViolation is ERROR_SHARING_VIOLATION (32).
const errorSharingViolation syscall.Errno = 32

// tryLockFile opens path with share mode 0, so any second open (from another
// process) fails with ERROR_SHARING_VIOLATION until the handle is closed.
func tryLockFile(path string) (release func() error, busy bool, err error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, false, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil,
		syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if err == errorSharingViolation {
			return nil, true, nil
		}
		return nil, false, err
	}
	return func() error { return syscall.CloseHandle(h) }, false, nil
}
