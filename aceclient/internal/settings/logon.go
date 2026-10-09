package settings

import (
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"strings"
)

// LogonUserName ports the DlgLogon constructor's user-name pre-fill:
// Settings.LastUserName, or AppProperties.DefaultUserName ("User") when empty.
//
// ports DlgLogon..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs)
func (st *Store) LogonUserName() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if st.s.LastUserName == "" {
		return DefaultUserName
	}
	return st.s.LastUserName
}

// SetLastUserName ports DlgLogon.OnOkClicked after a successful logon:
// Settings.LastUserName = m_txtUsername.Text; Save().
//
// ports DlgLogon.OnOkClicked (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs)
func (st *Store) SetLastUserName(name string) error {
	return st.Update(func(s *Settings) { s.LastUserName = name })
}

// DisplayName returns Settings.LastUserName unmodified. MainWindow.OnLoginRequest
// copies it into the device's ACEWebLoginData.DisplayName, but only AFTER the
// login attempt (MainWindow.cs: after the awaited LoginRequest in the
// non-unique-password branch, and after DlgDeviceLogin closed, whose Ok
// handler already logged in). So this is not the value to send as
// "displayname" in /api/login: the first login for each device sends "",
// later ones send what the previous OnLoginRequest copied. Use
// SessionCredentials.DisplayName for the value to send, and
// SessionCredentials.AssignDisplayName / FinishLoginDialog for the copy.
//
// ports Settings.Default.LastUserName as read by MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (st *Store) DisplayName() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s.LastUserName
}

// HashUserName ports the user-name hash DlgLogon (and ICUUser.CreateHashedUser)
// uses as ICUUser.User and as the LocalPasswords key:
// Convert.ToBase64String(SHA256(UTF8(name.ToLowerInvariant()))).
// strings.ToLower stands in for ToLowerInvariant (identical for ASCII).
//
// ports DlgLogon.ValidatePassword / ICUUser.CreateHashedUser (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs, ACESettings/ICUSettings/ICUUser.cs)
func HashUserName(name string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(name)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// LocalPasswordJSONs ports the LocalPasswords scan in DlgLogon.ValidatePassword:
// each entry is Split(';'); entries with more than one part whose first part
// equals hashedUser contribute their second part (an ICUUser JSON object). The
// ICUUser/ICUConfig handling itself is not part of this package.
//
// The caller must try the results in list order and keep the C# abort rule.
// The first entry whose password validates logs the user on. If deserialising
// an entry or constructing its ICUUser throws, ValidatePassword logs the error
// and returns null at once. Later entries are not tried and the "Incorrect
// password" warning is not shown. Only when every entry was tried without a
// match does the warning appear. Because of the mirrored `Count() > 2` bug in
// ReplaceLocalPassword, entries pile up, so this abort rule matters.
// (Settings.Get().LocalPasswords gives the raw list for a caller that does the
// whole scan itself.)
//
// ports DlgLogon.ValidatePassword (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs)
func (st *Store) LocalPasswordJSONs(hashedUser string) []string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	var out []string
	for _, e := range st.s.LocalPasswords {
		parts := strings.Split(e, ";")
		if len(parts) > 1 && parts[0] == hashedUser {
			out = append(out, parts[1])
		}
	}
	return out
}

// ReplaceLocalPassword ports the LocalPasswords update in
// DlgLogon.OnChangePassword: create the list if null, remove the first entry
// whose Split(';') has MORE THAN TWO parts and whose first part equals
// hashedUser, then append hashedUser + ";" + userJSON and save. userJSON is
// ICUUser.Json of the user created by ICUUser.CreateHashedUser.
//
// Fidelity note: the C# removal test is "array.Count() > 2", while a normal
// entry "<hash>;<json>" splits into exactly two parts (base64 and the JSON
// contain no ';'), so in practice nothing is removed and every password
// change appends another entry. ValidatePassword then tries them in order.
// This is mirrored, not fixed.
//
// ports DlgLogon.OnChangePassword (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs)
func (st *Store) ReplaceLocalPassword(hashedUser, userJSON string) error {
	return st.Update(func(s *Settings) {
		if s.LocalPasswords == nil {
			s.LocalPasswords = []string{}
		}
		for i, e := range s.LocalPasswords {
			parts := strings.Split(e, ";")
			if len(parts) > 2 && parts[0] == hashedUser {
				s.LocalPasswords = slices.Delete(s.LocalPasswords, i, i+1)
				break
			}
		}
		s.LocalPasswords = append(s.LocalPasswords, hashedUser+";"+userJSON)
	})
}

// Texts DlgLogon shows around local password changes.
const (
	// MsgLogonPasswordsMismatch is shown when the new and confirm passwords differ.
	MsgLogonPasswordsMismatch = "The new password and the confirm-new-passwords do not match!"
	// MsgLogonPasswordChanged / MsgLogonPasswordChangedDetail are shown after
	// ReplaceLocalPassword.
	MsgLogonPasswordChanged       = "Your password has been changed!"
	MsgLogonPasswordChangedDetail = "Note that it is only changed on this PC\nAnd that your original password still works."
)
