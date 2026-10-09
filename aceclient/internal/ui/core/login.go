package core

import "time"

// Device-login user levels (ICUNetwork.ACEWebLoginData constants and the
// DlgDeviceLogin m_cmbUserName items).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ACEWebLoginData.cs
const (
	UserLevelAdmin   = "admin"   // ACEWebLoginData.AdminUser, "Owner" (default)
	UserLevelTemp    = "temp"    // ACEWebLoginData.TempUser, "Temporary"
	UserLevelService = "service" // ACEWebLoginData.ServiceUser, "Secure Service Access"
)

// UserLevel is one DlgDeviceLogin combo entry (value, label).
type UserLevel struct {
	ID    string // the /api/login username
	Label string // text shown in the combo
}

// DeviceUserLevels ports the DlgDeviceLogin combo items in order.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs:47-168
var DeviceUserLevels = []UserLevel{
	{ID: UserLevelAdmin, Label: "Owner"},
	{ID: UserLevelTemp, Label: "Temporary"},
	{ID: UserLevelService, Label: "Secure Service Access"},
}

// UserLevelIndex returns the combo index of a user level id (-1 if unknown).
func UserLevelIndex(id string) int {
	for i, l := range DeviceUserLevels {
		if l.ID == id {
			return i
		}
	}
	return -1
}

// DlgDeviceLogin texts.
const (
	LoginPrompt       = "Please select the user level and enter the password to login\n"
	LoginUserLevel    = "User level:"
	LoginPassword     = "Password:"
	LoginStoreCaption = "Apply if all chargers have the same password.\r\nThe last used user level and password will be used for the next device you log into."
	// LoginCancelled is ACEWebLoginData.LoginError after Cancel (OnLoginRequest).
	LoginCancelled = "User cancelled the login"
)

// LoginIdentityLine ports DlgDeviceLogin's gray identity label.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs:47-168
func LoginIdentityLine(identity, serialNumber string) string {
	return "Charging Station identity: " + identity + " (Serial number: " + serialNumber + ")\n"
}

// StorePasswordAllowed ports the m_cmbUserName.SelectionChanged handler: the
// "store password" check box is unchecked and hidden for "service".
func StorePasswordAllowed(userLevel string) bool { return userLevel != UserLevelService }

// StoredPasswordExpired ports the test in DlgDeviceLogin.ValidateStoredPassword:
// (LastPasswordStoreTime - DateTime.Now).TotalHours > 24.0. The subtraction is
// reversed in the C#, so only a store time more than 24 h in the future
// expires; that is mirrored.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs:170-180
func StoredPasswordExpired(lastStore, now time.Time) bool {
	return lastStore.Sub(now).Hours() > 24.0
}

// LoginProgressFraction ports TimeOutProgressBarTask: the fraction shown after
// elapsed time, stepping every 100 ms over timeoutSeconds*10 steps.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs:207-220
func LoginProgressFraction(elapsed time.Duration, timeoutSeconds int) float64 {
	total := float64(timeoutSeconds * 10)
	if total <= 0 {
		return 1
	}
	i := float64(elapsed / (100 * time.Millisecond))
	if i > total {
		i = total
	}
	return i / total
}

// ConnectFailureText ports MainWindow.OnDeviceSelectionChanged's error for a
// failed login that was not cancelled; it is suppressed for OK, Forbidden,
// RequestTimeout, TooManyRequests (popups already shown) and
// HttpVersionNotSupported. ok is false when no popup is shown.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1038-1240
func ConnectFailureText(status int, ip, loginError string) (string, bool) {
	switch status {
	case StatusOK, StatusForbidden, StatusRequestTimeout, StatusTooManyRequests, StatusHTTPVersionNotSupp:
		return "", false
	}
	return "Failed to communicate with device at IP address " + ip + "\nError: " + loginError, true
}
