package api

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf16"
)

// New-password validation errors of DlgDeviceNewPassword.OnOkClicked
// (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceNewPassword.cs). The
// texts are the exact warnings the dialog shows.
var (
	ErrPasswordTooShort    = errors.New("The new password is too short, it needs to be at least 10 characters. Please enter another password.")
	ErrPasswordTooLong     = errors.New("The new password is too long, the maximum length is 40 characters. Please enter another password.")
	ErrPasswordInvalidChar = errors.New("The new password contains an invalid character, the characters '\\', '\"' and ',' are not allowed. Please enter another password.")
	ErrPasswordMismatch    = errors.New("The new passwords do not match! Please ensure that the new passwords are the same.")
)

// NewPasswordHint is the rule text DlgDeviceNewPassword shows under the
// password fields.
const NewPasswordHint = "The password must be between 10 and 40 characters long.\nCharacters '\\', '\"' and ',' are not allowed.\n"

// ValidateNewPassword ports DlgDeviceNewPassword.OnOkClicked: the checks run
// in the C# order (length counted in UTF-16 code units like string.Length)
// and the first failure is returned; nil means the password is accepted.
// ErrPasswordMismatch is the only one that concerns the confirmation field.
func ValidateNewPassword(password, confirm string) error {
	n := len(utf16.Encode([]rune(password)))
	switch {
	case n < 10:
		return ErrPasswordTooShort
	case n > 40:
		return ErrPasswordTooLong
	case strings.ContainsAny(password, "\\\","):
		return ErrPasswordInvalidChar
	case password != confirm:
		return ErrPasswordMismatch
	}
	return nil
}

// fwFileVersionRe is the DlgUpload.OnBtnUploadClicked pattern (the dots are
// unescaped in the C# too).
var fwFileVersionRe = regexp.MustCompile(`[_ ](\d+.\d+.\d+)[_.-]`)

// NewPasswordRequiredForUpgrade ports the DlgUpload.OnBtnUploadClicked check
// that opens DlgDeviceNewPassword: a non-AHP charger on firmware major < 5
// receiving a file whose lower-cased name carries a version (pattern
// "[_ ](\d+.\d+.\d+)[_.-]") with major >= 5. A version the C# Version.Parse
// would reject (it throws there) counts as no version.
func NewPasswordRequiredForUpgrade(isAHP bool, currentFWMajor int, fileName string) bool {
	name := fileName
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	m := fwFileVersionRe.FindStringSubmatch(strings.ToLower(name))
	if m == nil {
		return false
	}
	major, ok := parseVersionMajor(m[1])
	if !ok {
		return false
	}
	return !isAHP && currentFWMajor < 5 && major >= 5
}

// parseVersionMajor ports System.Version.Parse for "a.b.c" and returns a.
func parseVersionMajor(s string) (int, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return 0, false
	}
	vals := make([]int, len(parts))
	for i, p := range parts {
		n, ok := parseNetInt(p, 32)
		if !ok || n < 0 {
			return 0, false
		}
		vals[i] = int(n)
	}
	return vals[0], true
}
