package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Logon messages, verbatim from DlgLogon (ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs).
var (
	// ErrUnknownUser is the warning shown when no config user matches the hashed name.
	ErrUnknownUser = errors.New("Unknown user name.\nPlease enter a valid user name.")
	// ErrIncorrectPassword is the warning shown when neither the config entry
	// nor any local password entry validates.
	ErrIncorrectPassword = errors.New("Incorrect password.\nPlease enter the correct password.")
	// ErrNewPasswordMismatch is OnChangePassword's error when new != confirm.
	ErrNewPasswordMismatch = errors.New("The new password and the confirm-new-passwords do not match!")
)

// Messages DlgLogon.OnChangePassword shows after a successful change
// (MessageDialog.ShowMessage(primary, secondary)).
const (
	PasswordChangedMessage = "Your password has been changed!"
	PasswordChangedDetail  = "Note that it is only changed on this PC\nAnd that your original password still works."
)

// LogonWelcomeText is the DlgLogon header label.
const LogonWelcomeText = "Welcome to the ACE Service Installer.\nPlease enter your user name and password below.\n\n"

// InitialLogonUserName ports DlgLogon's user-name prefill:
// Settings.Default.LastUserName, or AppProperties.DefaultUserName when empty.
// After a successful Logon the C# stores the name as typed into LastUserName.
// That setting lives in internal/settings (Store.LogonUserName /
// Store.SetLastUserName); this package keeps no user-settings store.
func InitialLogonUserName(lastUserName string) string {
	if lastUserName == "" {
		return DefaultUserName
	}
	return lastUserName
}

// LogonVersionText ports the grey version label of DlgLogon:
// "Settings version: {config Version}\nApplication version: {file version}".
func LogonVersionText(configVersion, fileVersion string) string {
	return fmt.Sprintf("Settings version: %s\nApplication version: %s", configVersion, FormatAppVersion(fileVersion))
}

// FormatAppVersion ports the FileVersion formatting in DlgLogon / MainWindow:
// with at least four dot-separated parts "a.b.c.d" becomes "a.b.c-d",
// otherwise the string is returned unchanged.
func FormatAppVersion(fileVersion string) string {
	p := strings.Split(fileVersion, ".")
	if len(p) >= 4 {
		return fmt.Sprintf("%s.%s.%s-%s", p[0], p[1], p[2], p[3])
	}
	return fileVersion
}

// Logon ports DlgLogon.ValidatePassword + CreateNewUser: the user name is
// hashed (HashUserName) and must equal a config user's User exactly, else
// ErrUnknownUser. The password is verified with ValidateHashedPassword; when
// that fails, every localPasswords entry ("<hashed user>;<ICUUser json>",
// Settings.Default.LocalPasswords, i.e. internal/settings
// Store.Get().LocalPasswords) whose first field matches is parsed and tried in
// order; otherwise ErrIncorrectPassword.
//
// The returned user is the CreateNewUser product: a copy of the group with
// the original user's rights, Fullname/Company decrypted and the group's
// HTTPUser/HTTPPassword taken from the decrypted Comment ("user:password",
// only when it splits into exactly two parts). Company carries the ISAH
// credentials (PanelInformation / DlgObjectID pass it as IsahCredentials).
//
// Errors other than the two sentinels mirror exceptions in the C#: from a
// local-password entry they are caught there (logged, logon aborted without a
// message); from the config entry they propagate.
func (c *Config) Logon(username, password string, localPasswords []string) (*User, error) {
	hashed := HashUserName(username)
	var orig *User
	for _, u := range c.Users {
		if u.User == hashed {
			orig = u
			break
		}
	}
	if orig == nil {
		return nil, ErrUnknownUser
	}
	ok, err := orig.ValidateHashedPassword(password)
	if err != nil {
		return nil, err
	}
	if ok {
		return createNewUser(orig, password)
	}
	for _, entry := range localPasswords {
		parts := strings.Split(entry, ";")
		if len(parts) > 1 && parts[0] == hashed {
			u, err := c.tryLocalPassword(parts[1], password)
			if err != nil {
				return nil, fmt.Errorf("config: local password entry: %w", err)
			}
			if u != nil {
				return u, nil
			}
		}
	}
	return nil, ErrIncorrectPassword
}

// tryLocalPassword is the try-block body of ValidatePassword's local-password
// loop: new ICUUser(Groups, JavaScriptSerializer.DeserializeObject(json)).
func (c *Config) tryLocalPassword(userJSON, password string) (*User, error) {
	raw := json.RawMessage(userJSON)
	if !json.Valid(raw) {
		raw = escapeControlChars(raw)
	}
	u, err := userFromJSON(c.Groups, raw, "LocalPasswords")
	if err != nil {
		return nil, err
	}
	ok, err := u.ValidateHashedPassword(password)
	if err != nil || !ok {
		return nil, err
	}
	return createNewUser(u, password)
}

// createNewUser ports DlgLogon.CreateNewUser.
func createNewUser(orig *User, userPassword string) (*User, error) {
	if orig.Group == nil {
		return nil, errors.New("config: user has no group") // NullReferenceException in the C#
	}
	features := make([]*Feature, 0, len(orig.Group.Features))
	for _, fr := range orig.Group.Features {
		features = append(features, fr.Feature)
	}
	g := NewGroup(features, orig.Group.Name, "", "", "")
	for _, fr := range g.Features {
		fr.Rights = orig.GetRights(fr.Feature.ID)
	}
	parts := strings.Split(orig.Password, ":")
	if len(parts) != 3 {
		return nil, ErrIncorrectPassword // CreateNewUser returns null; ValidatePassword already required 3 parts
	}
	passPhrase := parts[1] + userPassword
	u := &User{Group: g, User: orig.User, Password: orig.Password}
	fullname, err := Decrypt(orig.Fullname, passPhrase)
	if err != nil {
		return nil, err
	}
	u.Fullname = strings.TrimSpace(fullname)
	text, err := Decrypt(orig.Comment, passPhrase)
	if err != nil {
		return nil, err
	}
	company, err := Decrypt(orig.Company, passPhrase)
	if err != nil {
		return nil, err
	}
	u.Company = strings.TrimSpace(company)
	if cred := strings.Split(text, ":"); len(cred) == 2 {
		g.HTTPUser = strings.TrimSpace(cred[0])
		g.HTTPPassword = strings.TrimSpace(cred[1])
	}
	return u, nil
}

// ChangeLocalPassword ports DlgLogon.OnChangePassword: the current
// credentials must pass Logon; newPassword must equal confirmPassword
// (ErrNewPasswordMismatch); then a new hashed user is created
// (CreateHashedUser(lower(username), newPassword, group, Company,
// "HTTPUser:HTTPPassword") from the logged-on user) and appended to the local
// password list as "<User>;<User.JSON()>". The returned slice replaces
// Settings.Default.LocalPasswords: the caller stores it through internal/settings
// (Store.Update(func(s *settings.Settings) { s.LocalPasswords = list })) and
// shows PasswordChangedMessage / PasswordChangedDetail. The original password
// keeps working.
//
// Like the C#, an existing entry for the user is removed only when it splits
// into more than two ';' fields, which a normal entry never does - so old
// entries accumulate and are tried in order by Logon.
func (c *Config) ChangeLocalPassword(username, password, newPassword, confirmPassword string, localPasswords []string) ([]string, error) {
	u, err := c.Logon(username, password, localPasswords)
	if err != nil {
		return localPasswords, err
	}
	if newPassword != confirmPassword {
		return localPasswords, ErrNewPasswordMismatch
	}
	hashed := HashUserName(username)
	out := append([]string(nil), localPasswords...)
	for i, entry := range out {
		if parts := strings.Split(entry, ";"); len(parts) > 2 && parts[0] == hashed {
			out = append(out[:i], out[i+1:]...)
			break
		}
	}
	httpData := u.Group.HTTPUser + ":" + u.Group.HTTPPassword
	nu, err := CreateHashedUser(strings.ToLower(username), newPassword, u.Group, u.Company, httpData)
	if err != nil {
		return localPasswords, err
	}
	return append(out, nu.User+";"+nu.JSON()), nil
}
