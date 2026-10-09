package settings

import (
	"errors"
	"slices"
	"sync"
	"time"
)

// Device-login user levels (DlgDeviceLogin m_cmbUserName items: value, label).
const (
	UserLevelAdmin   = "admin"   // "Owner" (default selection)
	UserLevelTemp    = "temp"    // "Temporary"
	UserLevelService = "service" // "Secure Service Access" (password never stored)
)

// UserLevel is one DlgDeviceLogin user-level combo item.
type UserLevel struct {
	ID    string // value sent as the /api/login username
	Label string // text shown in the combo
}

// DeviceUserLevels lists the DlgDeviceLogin user levels in combo order.
//
// ports DlgDeviceLogin..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
var DeviceUserLevels = []UserLevel{
	{ID: UserLevelAdmin, Label: "Owner"},
	{ID: UserLevelTemp, Label: "Temporary"},
	{ID: UserLevelService, Label: "Secure Service Access"},
}

// isDeviceUserLevel reports whether level is one of the m_cmbUserName items
// (Xwt ItemCollection.IndexOf(level) != -1).
func isDeviceUserLevel(level string) bool {
	return slices.ContainsFunc(DeviceUserLevels, func(l UserLevel) bool { return l.ID == level })
}

// LabelStorePassword is the DlgDeviceLogin m_chkStorePassword caption.
const LabelStorePassword = "Apply if all chargers have the same password.\r\nThe last used user level and password will be used for the next device you log into."

// storedPasswordMaxAge is the 24 h limit tested by ValidateStoredPassword.
const storedPasswordMaxAge = 24 * time.Hour

// ErrNoUserLevel is returned by DeviceLoginForm.Submit when no user level is
// selected. The C# property DlgDeviceLogin.Username reads
// m_cmbUserName.SelectedItem.ToString() without a null check. OnOkClicked
// catches the NullReferenceException, sets LoginData.IsLoggedIn = false and
// LoginData.LoginError to the exception message, and responds Command.Cancel.
// So the dialog closes as cancelled and no login request is sent.
//
// ports DlgDeviceLogin.Username / OnOkClicked catch block (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
var ErrNoUserLevel = errors.New("settings: no user level selected")

// Credentials is the username/password pair of ICUNetwork.ACEWebLoginData
// (ACENetwork/ICUNetwork/ACEWebLoginData.cs) as used by the login flow.
//
// ports ACEWebLoginData (ACENetwork/ICUNetwork/ACEWebLoginData.cs)
type Credentials struct {
	Username string
	Password string
}

// StorePasswords returns Settings.StorePasswords (the "store password"
// check box state of the last successful device login).
//
// ports Settings.StorePasswords reads in DlgDeviceLogin / MainWindow (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (st *Store) StorePasswords() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.s.StorePasswords
}

// ValidateStoredPassword ports DlgDeviceLogin.ValidateStoredPassword:
//
//	if ((LastPasswordStoreTime - DateTime.Now).TotalHours > 24.0) {
//	    LastDeviceUsername = "admin"; LastDevicePassword = ""; Save(); return false;
//	}
//	return StorePasswords;
//
// Fidelity note: the subtraction is reversed, so the reset only triggers when
// LastPasswordStoreTime lies more than 24 h in the FUTURE (e.g. after a clock
// change). A stored password therefore never expires by age. This is mirrored,
// not fixed. The returned error is the save error of the reset branch.
//
// ports DlgDeviceLogin.ValidateStoredPassword (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (st *Store) ValidateStoredPassword(now time.Time) (bool, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.validateStoredPasswordLocked(now)
}

func (st *Store) validateStoredPasswordLocked(now time.Time) (bool, error) {
	if st.s.LastPasswordStoreTime.Sub(now) > storedPasswordMaxAge {
		st.s.LastDeviceUsername = UserLevelAdmin
		st.s.LastDevicePassword = ""
		return false, st.saveLocked()
	}
	return st.s.StorePasswords, nil
}

// LoginPrefill ports the unique-password branch of MainWindow.OnLoginRequest
// that decides which credentials are written into the device's LoginData
// before DlgDeviceLogin is shown:
//
//	if (DlgDeviceLogin.ValidateStoredPassword() && !forceDialog && automaticAttempt)
//	    use lanDevice.UserData (session) if the device has one,
//	    else Settings.LastDeviceUsername / LastDevicePassword.
//
// session is the device's UserData (SessionCredentials.Get), nil when the
// device has none. When ok is true the caller must store cred as the device's
// LoginData (SessionCredentials.SetLoginData). When ok is false the C# leaves
// LoginData unchanged: it still holds the credentials of the previous dialog
// for that device (or nothing for a new device), and DlgDeviceLogin submits
// them automatically when both are non-empty. This happens even when
// StorePasswords is off. SessionCredentials.PrepareDeviceLogin does all of this.
//
// The C# clears automaticAttempt after this call
// (m_fAutomaticDeviceLoginAttempt = false) until AllowAutoLogonOnce or a
// device selection re-arms it; that state lives in the caller.
//
// ports MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (st *Store) LoginPrefill(now time.Time, forceDialog, automaticAttempt bool, session *Credentials) (cred Credentials, ok bool, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	valid, err := st.validateStoredPasswordLocked(now)
	if !valid || forceDialog || !automaticAttempt {
		return Credentials{}, false, err
	}
	if session != nil {
		return *session, true, err
	}
	return Credentials{Username: st.s.LastDeviceUsername, Password: st.s.LastDevicePassword}, true, err
}

// DeviceLoginForm is the state of DlgDeviceLogin.
//
// ports DlgDeviceLogin..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
type DeviceLoginForm struct {
	// UserLevel is the selected m_cmbUserName value (see DeviceUserLevels).
	// "" means no selection (SelectedIndex -1).
	UserLevel string
	// Password pre-fills m_txtPassword. In C# the first key press in the
	// password box clears a pre-filled password (m_isPasswordChanged).
	Password string
	// StorePassword is the m_chkStorePassword state.
	StorePassword bool
	// StorePasswordVisible is false while the "service" level is selected.
	StorePasswordVisible bool
	// AutoSubmit is true when the device's LoginData already carries a user
	// name and password: the dialog presses Ok itself once shown.
	AutoSubmit bool
}

// SelectUserLevel ports setting m_cmbUserName.SelectedItem with the
// SelectionChanged handler attached. A view calls it when the user picks a
// level.
//
// Xwt sets SelectedIndex = Items.IndexOf(level). A level that is not in
// DeviceUserLevels (for example a group HTTP user left in LoginData) leaves the
// combo with no selection, so UserLevel becomes "". The handler then throws a
// NullReferenceException (SelectedItem is null). Xwt reports it as an
// unhandled exception and continues, so the store-password box keeps its
// state. For a known level the handler runs: "service" unticks and hides the
// store-password box, and any other level shows it again.
//
// ports DlgDeviceLogin..ctor m_cmbUserName.SelectionChanged handler (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (f *DeviceLoginForm) SelectUserLevel(level string) {
	if !isDeviceUserLevel(level) {
		f.UserLevel = ""
		return
	}
	f.UserLevel = level
	if level == UserLevelService {
		f.StorePassword = false
		f.StorePasswordVisible = false
	} else {
		f.StorePasswordVisible = true
	}
}

// Submit ports the start of DlgDeviceLogin.OnOkClicked, which writes the
// dialog values into the device's LoginData before /api/login:
//
//	m_lanDevice.LoginData.Username = Username; // m_cmbUserName.SelectedItem.ToString()
//	m_lanDevice.LoginData.Password = Password;
//
// The caller stores cred with SessionCredentials.SetLoginData and logs in. On
// success it calls Store.RecordDeviceLogin(cred.Username, cred.Password,
// f.StorePassword, now) and then SessionCredentials.FinishLoginDialog with
// ok = true. On a failed login the C# dialog stays open.
//
// With no selection Submit returns ErrNoUserLevel. The caller must then send
// nothing and close the dialog as cancelled (FinishLoginDialog with ok =
// false), as the C# NullReferenceException path does.
//
// ports DlgDeviceLogin.OnOkClicked (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (f DeviceLoginForm) Submit() (Credentials, error) {
	if !isDeviceUserLevel(f.UserLevel) {
		return Credentials{}, ErrNoUserLevel
	}
	return Credentials{Username: f.UserLevel, Password: f.Password}, nil
}

// DeviceLoginDefaults ports the DlgDeviceLogin constructor. preset is the
// device's current LoginData username/password (SessionCredentials.LoginData).
// LoginPrefill overwrites LoginData only when it returns ok. Otherwise
// LoginData still holds the credentials of the previous dialog for that
// device. Use SessionCredentials.PrepareDeviceLogin to get this right.
//
//  1. ValidateStoredPassword();
//  2. select "admin";
//  3. if StorePasswords (and the selection is not "service"): tick the box,
//     select LastDeviceUsername when non-empty, pre-fill LastDevicePassword;
//     else untick the box;
//  4. if preset has both username and password: select preset.Username,
//     pre-fill preset.Password and submit automatically when shown.
//
// A user level that is not a combo item leaves no selection (see
// SelectUserLevel). An automatic submit then cancels the dialog (see Submit).
//
// ports DlgDeviceLogin..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (st *Store) DeviceLoginDefaults(now time.Time, preset Credentials) (DeviceLoginForm, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, err := st.validateStoredPasswordLocked(now)
	f := DeviceLoginForm{UserLevel: UserLevelAdmin, StorePasswordVisible: true}
	if st.s.StorePasswords && f.UserLevel != UserLevelService {
		f.StorePassword = true
		if st.s.LastDeviceUsername != "" {
			f.SelectUserLevel(st.s.LastDeviceUsername)
		}
		f.Password = st.s.LastDevicePassword
	} else {
		f.StorePassword = false
	}
	if preset.Username != "" && preset.Password != "" {
		f.SelectUserLevel(preset.Username)
		f.Password = preset.Password
		f.AutoSubmit = true
	}
	return f, err
}

// RecordDeviceLogin ports the success branch of DlgDeviceLogin.OnOkClicked:
//
//	if (store) { LastDeviceUsername = username; LastDevicePassword = password; }
//	else       { LastDeviceUsername = "admin";  LastDevicePassword = ""; }
//	LastPasswordStoreTime = now; StorePasswords = store; Save();
//
// Call it only after /api/login succeeded. The password is persisted in plain
// text exactly when store is true (C# user.config is plain XML as well).
//
// ports DlgDeviceLogin.OnOkClicked (ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs)
func (st *Store) RecordDeviceLogin(username, password string, store bool, now time.Time) error {
	return st.Update(func(s *Settings) {
		if store {
			s.LastDeviceUsername = username
			s.LastDevicePassword = password
		} else {
			s.LastDeviceUsername = UserLevelAdmin
			s.LastDevicePassword = ""
		}
		s.LastPasswordStoreTime = now
		s.StorePasswords = store
	})
}

// UpdateStoredDevicePassword ports DlgUpload's post-upload step: when the
// firmware upload set a new device password (m_newPassword non-empty), the C#
// writes it to Settings.LastDevicePassword and saves — regardless of
// StorePasswords — then logs in again. An empty password is a no-op.
//
// ports DlgUpload upload-completed handler (ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs)
func (st *Store) UpdateStoredDevicePassword(newPassword string) error {
	if newPassword == "" {
		return nil
	}
	return st.Update(func(s *Settings) { s.LastDevicePassword = newPassword })
}

// loginData is the part of ICULanDevice.LoginData (an ACEWebLoginData) that
// the login flow reads back: user name, password and DisplayName.
type loginData struct {
	cred        Credentials
	displayName string
}

// SessionCredentials holds the per-device login state that the C# keeps on
// each ICULanDevice object, in memory only, for the lifetime of the device.
// Nothing here is ever persisted. Keys are chosen by the caller (e.g. the
// device address). The zero value is ready to use and safe for concurrent use.
//
// It tracks two separate things, as the C# does:
//
//   - UserData (Get / Set / RememberAfterDialog). MainWindow.OnLoginRequest
//     sets it after the login dialog, but only while Settings.StorePasswords
//     is set, and LoginPrefill prefers it over the stored slot.
//   - LoginData (LoginData / SetLoginData / DisplayName). This holds the
//     credentials sent to /api/login. DlgDeviceLogin.OnOkClicked writes them
//     on every Ok, whatever StorePasswords is, and only a cancelled dialog or
//     a logout clears them. While both are non-empty the next DlgDeviceLogin
//     for that device submits itself.
//
// ports ICULanDevice.UserData + ICULanDevice.LoginData (ACENetwork/ICUNetwork/ICULanDevice.cs) as used by MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
type SessionCredentials struct {
	mu       sync.Mutex
	userData map[string]Credentials
	login    map[string]loginData
}

// NewSessionCredentials returns an empty cache (equivalent to the zero value).
func NewSessionCredentials() *SessionCredentials {
	return &SessionCredentials{}
}

// Get returns the device's UserData credentials, or nil when it has none
// (UserData == null).
func (c *SessionCredentials) Get(device string) *Credentials {
	c.mu.Lock()
	defer c.mu.Unlock()
	cr, ok := c.userData[device]
	if !ok {
		return nil
	}
	return &cr
}

// Set stores the device's UserData (lanDevice.UserData = ...).
func (c *SessionCredentials) Set(device string, cr Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.userData == nil {
		c.userData = map[string]Credentials{}
	}
	c.userData[device] = cr
}

// RememberAfterDialog ports only the UserData step after the login dialog in
// MainWindow.OnLoginRequest. When st.StorePasswords() is set, the dialog
// result becomes the device's UserData. This includes the empty pair of a
// cancelled dialog:
//
//	if (Settings.Default.StorePasswords) lanDevice.UserData = new ACEWebLoginData { Username, Password };
//
// FinishLoginDialog ports the whole post-dialog block and should normally be
// used instead.
//
// ports MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (c *SessionCredentials) RememberAfterDialog(st *Store, device string, cr Credentials) {
	if st.StorePasswords() {
		c.Set(device, cr)
	}
}

// LoginData returns the user name and password of the device's LoginData.
// They are empty for a device that has not been logged in yet (a new
// ACEWebLoginData has null strings).
//
// ports ICULanDevice.LoginData (ACENetwork/ICUNetwork/ICULanDevice.cs)
func (c *SessionCredentials) LoginData(device string) Credentials {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.login[device].cred
}

// SetLoginData sets the user name and password of the device's LoginData and
// keeps its DisplayName. The C# writers are:
//
//   - OnLoginRequest: the LoginPrefill result when ok, or the group HTTP
//     credentials in the non-unique-password branch;
//   - DlgDeviceLogin.OnOkClicked: the dialog values (DeviceLoginForm.Submit)
//     just before /api/login, even if that login then fails;
//   - ICULanDevice.ResetPassword: the password is cleared.
//
// ports writes to ICULanDevice.LoginData.Username/Password (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs, DlgDeviceLogin.cs; ACENetwork/ICUNetwork/ICULanDevice.cs)
func (c *SessionCredentials) SetLoginData(device string, cr Credentials) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLoginLocked(device, cr)
}

func (c *SessionCredentials) setLoginLocked(device string, cr Credentials) {
	if c.login == nil {
		c.login = map[string]loginData{}
	}
	ld := c.login[device]
	ld.cred = cr
	c.login[device] = ld
}

// DisplayName returns the device's LoginData.DisplayName: the value to send
// as "displayname" in /api/login (api.LoginData.DisplayName).
//
// The C# assigns it only after a login attempt (AssignDisplayName), and
// ACEWebLoginData.DisplayName starts as null. So the first /api/login for
// each device sends "displayname":"", and later logins send the
// LastUserName copied at the end of the previous OnLoginRequest.
//
// ports ACEWebLoginData.DisplayName as read by ICULanDevice.LoginRequest / ClassicLogin (ACENetwork/ICUNetwork/ICULanDevice.cs)
func (c *SessionCredentials) DisplayName(device string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.login[device].displayName
}

// AssignDisplayName ports `loginData.DisplayName =
// Settings.Default.LastUserName`. MainWindow.OnLoginRequest runs it at the
// end of both branches: after the group-credential login in the
// non-unique-password branch, and after the login dialog closed in the
// unique-password branch (FinishLoginDialog does that one). It does not run
// when the dialog was not shown because another one was already open.
//
// ports MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (c *SessionCredentials) AssignDisplayName(st *Store, device string) {
	name := st.DisplayName()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.login == nil {
		c.login = map[string]loginData{}
	}
	ld := c.login[device]
	ld.displayName = name
	c.login[device] = ld
}

// PrepareDeviceLogin ports the unique-password branch of
// MainWindow.OnLoginRequest up to showing DlgDeviceLogin:
//
//  1. LoginPrefill with the device's UserData. When it returns ok, the
//     result becomes the device's LoginData;
//  2. DeviceLoginDefaults with the device's LoginData as the preset.
//
// LoginData is not cleared when LoginPrefill returns ok = false. So a device
// that was logged in through the dialog before gets AutoSubmit with those
// credentials again, even when StorePasswords is off.
//
// ports MainWindow.OnLoginRequest + DlgDeviceLogin..ctor (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs, DlgDeviceLogin.cs)
func (c *SessionCredentials) PrepareDeviceLogin(st *Store, device string, now time.Time, forceDialog, automaticAttempt bool) (DeviceLoginForm, error) {
	cred, ok, err := st.LoginPrefill(now, forceDialog, automaticAttempt, c.Get(device))
	if ok {
		c.SetLoginData(device, cred)
	}
	f, err2 := st.DeviceLoginDefaults(now, c.LoginData(device))
	return f, errors.Join(err, err2)
}

// FinishLoginDialog ports the block of MainWindow.OnLoginRequest that runs
// after DlgDeviceLogin closed. ok is true for Command.Ok, and cred is then
// the dialog's user name and password:
//
//	if (Ok) { loginData.Username = dlg.Username; loginData.Password = dlg.Password; }
//	else    { loginData.Username = ""; loginData.Password = ""; } // cancelled
//	if (Settings.Default.StorePasswords) lanDevice.UserData = copy of loginData;
//	loginData.DisplayName = Settings.Default.LastUserName;
//
// StorePasswords is read after the dialog, so a RecordDeviceLogin from the
// same dialog counts.
//
// ports MainWindow.OnLoginRequest (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (c *SessionCredentials) FinishLoginDialog(st *Store, device string, ok bool, cred Credentials) {
	if !ok {
		cred = Credentials{}
	}
	store := st.StorePasswords()
	name := st.DisplayName()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setLoginLocked(device, cred)
	if store {
		if c.userData == nil {
			c.userData = map[string]Credentials{}
		}
		c.userData[device] = cred
	}
	ld := c.login[device]
	ld.displayName = name
	c.login[device] = ld
}

// AfterPasswordChange ports PanelInformation.OnChangePassword after
// ICULanDevice.ChangePassword succeeded: the new password replaces the
// LoginData password only when the logged-in user is "admin"
// (LoginData.IsAdminUser). UserData is not touched.
//
// ports PanelInformation.OnChangePassword (ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs)
func (c *SessionCredentials) AfterPasswordChange(device, newPassword string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ld := c.login[device]
	if ld.cred.Username != UserLevelAdmin {
		return
	}
	ld.cred.Password = newPassword
	c.login[device] = ld
}

// Logout ports PanelInformation.OnLogout after ICULanDevice.Logout()
// returned true: UserData = null and the LoginData user name and password
// are cleared. DisplayName is kept.
//
// ports PanelInformation.OnLogout (ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs)
func (c *SessionCredentials) Logout(device string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.userData, device)
	if _, ok := c.login[device]; ok {
		c.setLoginLocked(device, Credentials{})
	}
}

// Delete forgets all state of the device (the ICULanDevice object is
// deallocated or removed from the overview).
func (c *SessionCredentials) Delete(device string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.userData, device)
	delete(c.login, device)
}
