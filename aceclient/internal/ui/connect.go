package ui

import (
	"errors"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/ui/core"
)

// ConnectParams is what the connection bar or the login dialog collect.
type ConnectParams struct {
	Address  string
	Port     int
	Insecure bool
	Username string // user level id: "admin", "temp" or "service"
	Password string
	// Protocol forces "https"/"http"; "" applies the C# rule
	// (ICULanDevice.IsHTTPS => Port == 443).
	Protocol string
	// What discovery knows about the device (shown in dialogs/messages).
	Identity, SerialNumber, HostName string
	// From an mDNS announcement (ICULanDevice.ReInitialize): the "type" TXT
	// socket count and the raw "fwversion" TXT; zero values mean unknown.
	NumberOfSockets int
	FirmwareVersion string
}

// LoginStore persists what DlgDeviceLogin keeps in Settings.Default
// (LastDeviceUsername, LastDevicePassword, LastPasswordStoreTime,
// StorePasswords) and the login display name (LastUserName). The settings
// agent provides a persistent implementation via State.SetLoginStore; the
// default keeps everything in memory for the session only.
type LoginStore interface {
	// DeviceLoginDefaults ports the DlgDeviceLogin constructor prefill.
	DeviceLoginDefaults(now time.Time) (username, password string, store bool)
	// RecordDeviceLogin ports OnOkClicked's Settings writes after a login.
	RecordDeviceLogin(username, password string, store bool, now time.Time)
	// DisplayName is Settings.Default.LastUserName (login "displayname").
	DisplayName() string
}

type memoryLoginStore struct {
	mu        sync.Mutex
	username  string
	password  string
	store     bool
	storeTime time.Time
}

// DeviceLoginDefaults ports ValidateStoredPassword (reset to "admin"/"" when
// expired, see core.StoredPasswordExpired) followed by the constructor's
// `if (StorePasswords && level != "service")` prefill.
func (m *memoryLoginStore) DeviceLoginDefaults(now time.Time) (string, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if core.StoredPasswordExpired(m.storeTime, now) {
		m.username, m.password = core.UserLevelAdmin, ""
	}
	if !m.store {
		return core.UserLevelAdmin, "", false
	}
	u := m.username
	if u == "" {
		u = core.UserLevelAdmin
	}
	return u, m.password, true
}

// RecordDeviceLogin ports OnOkClicked: store the credentials when the check
// box is on, else reset them to "admin"/"".
func (m *memoryLoginStore) RecordDeviceLogin(username, password string, store bool, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if store {
		m.username, m.password = username, password
	} else {
		m.username, m.password = core.UserLevelAdmin, ""
	}
	m.storeTime = now
	m.store = store
}

func (m *memoryLoginStore) DisplayName() string { return "" }

func (s *State) currentLoginStore() LoginStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loginStore
}

func (s *State) loginDisplayName() string {
	s.mu.RLock()
	name, ls := s.displayName, s.loginStore
	s.mu.RUnlock()
	if name != "" || ls == nil {
		return name
	}
	return ls.DisplayName()
}

// connectFromBar is the Connect button: without a password the login dialog
// asks for one (OnLoginRequest shows DlgDeviceLogin when no credentials are
// known); otherwise the bar's values log in directly.
func (s *State) connectFromBar() {
	p, _ := s.barParams()
	if p.Password == "" {
		s.ShowLoginDialog()
		return
	}
	s.Connect(p, nil)
}

// Connect ports the device-selection login of MainWindow.OnDeviceSelectionChanged
// and DlgDeviceLogin.OnOkClicked: validate the address (DlgManualIP), offer to
// save pending changes of the previous device, log it out, then LoginRequest
// the new one off the UI goroutine. A failed login shows the
// HandleUnsuccessfulLoginRequest text; a successful one collects the
// "generic"/"generic2" categories, runs the OnConnect hooks and the tamper
// check. done (optional) receives the outcome on the UI goroutine.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1038-1240, DlgDeviceLogin.cs:222-320
func (s *State) Connect(p ConnectParams, done func(core.LoginResult, error)) {
	finish := func(res core.LoginResult, err error) {
		if done != nil {
			done(res, err)
		}
	}
	ip, uerr := core.ParseIPAddress(p.Address)
	if uerr != nil {
		s.ShowError(uerr.Primary, uerr.Secondary)
		finish(core.LoginResult{}, uerr)
		return
	}
	if uerr := core.ValidatePort(p.Port); uerr != nil {
		s.ShowError(uerr.Primary, uerr.Secondary)
		finish(core.LoginResult{}, uerr)
		return
	}
	p.Address = ip.String()
	prev := s.Session()
	if prev != nil && s.Connected() && prev.Props.HasChanges() {
		s.AskQuestion("You have changed some properties for device '"+prev.Identification()+"', do you want to save the changes?", func(yes bool) {
			if yes {
				s.SaveAllChanges(func(core.SaveResult, error) { s.connectNow(p, prev, finish) })
				return
			}
			prev.Props.Revert()
			s.connectNow(p, prev, finish)
		})
		return
	}
	s.connectNow(p, prev, finish)
}

func (s *State) connectNow(p ConnectParams, prev *core.Session, finish func(core.LoginResult, error)) {
	s.mu.RLock()
	protocol, eds := s.protocol, s.eds
	s.mu.RUnlock()
	if p.Protocol != "" {
		protocol = p.Protocol
	}
	sess := core.NewSession(p.Address, p.Port, p.Insecure, protocol)
	sess.Props.SetEDS(eds)
	sess.SetIdentity(p.Identity, p.SerialNumber, p.HostName)
	sess.SetSockets(p.NumberOfSockets, 0)
	if v, ok := core.AnnouncedFirmwareVersion(p.FirmwareVersion); ok && p.FirmwareVersion != "" {
		sess.SetFirmwareVersion(v)
	}
	ld := api.LoginData{Username: p.Username, Password: p.Password, DisplayName: s.loginDisplayName()}
	s.SetStatus("Logging in to %s...", p.Address)
	s.bar.connect.Disable()

	var res core.LoginResult
	var afterErr error
	s.RunAsync(func() {
		if prev != nil {
			prev.Logout()
		}
		res = sess.LoginRequest(ld)
		if res.LoggedIn {
			afterErr = sess.AfterLogin()
		}
	}, func() {
		if prev != nil && s.Session() == prev {
			s.dropSession()
		}
		if !res.LoggedIn {
			msg := core.LoginErrorMessage(res.StatusCode, res.Content, sess.Identification(), p.Address)
			s.setConnectedUI()
			s.SetStatus("Login to %s failed: %s", p.Address, msg)
			s.ShowError(msg)
			finish(res, errors.New(msg))
			return
		}
		s.mu.Lock()
		s.session = sess
		s.connected = true
		s.mu.Unlock()
		s.setConnectedUI()
		if afterErr != nil {
			s.SetStatus("Logged in to %s, reading properties failed: %s", sess.Identification(), requestErrorText(afterErr))
		} else {
			s.SetStatus("Logged in to %s (%s) as %s.", sess.Identification(), p.Address, ld.Username)
		}
		for _, fn := range s.hooks(&s.onConnect) {
			fn(s)
		}
		s.NotifyPropertiesChanged()
		s.firePanelShown(s.tabs.Selected())
		if core.TamperAlarm(sess.Props) {
			s.AskQuestion(core.TamperQuestionPrimary+"\n\n"+core.TamperQuestionSecondary, func(yes bool) {
				if !yes {
					return
				}
				var err error
				s.RunAsync(func() { err = sess.AcknowledgeTamper() }, func() {
					if err != nil {
						s.ShowError(requestErrorText(err))
					}
					s.NotifyPropertiesChanged()
				})
			})
		}
		finish(res, nil)
	})
}

// dropSession forgets the current device and fires the OnDisconnect hooks.
func (s *State) dropSession() {
	s.mu.Lock()
	was := s.connected
	s.session = nil
	s.connected = false
	s.mu.Unlock()
	s.setConnectedUI()
	if was {
		for _, fn := range s.hooks(&s.onDisconnect) {
			fn(s)
		}
	}
}

// Disconnect ports PanelInformation.OnLogout / ICULanDevice.Logout: POST
// /api/logout off the UI goroutine; when the device acknowledged it the
// stored login data (here: the bar's password) is cleared. done is optional.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs (OnLogout), firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1770-1794
func (s *State) Disconnect(done func()) {
	sess := s.Session()
	if sess == nil {
		if done != nil {
			done()
		}
		return
	}
	s.SetStatus("Logging out of %s...", sess.Identification())
	ok := false
	s.RunAsync(func() { ok = sess.Logout() }, func() {
		if ok {
			s.bar.pass.SetText("")
		}
		s.dropSession()
		s.SetStatus("Logged out of %s.", sess.Identification())
		if done != nil {
			done()
		}
	})
}

// ShowLoginDialog ports DlgDeviceLogin for the address in the connection
// bar: user level combo, password, the "store password" check box (hidden for
// "service"), and a progress bar that fills over the 15 s login timeout while
// the login runs. A failed login keeps the dialog open; success closes it.
// The "Forgot password" button (DlgResetPassword → /api/prc) is not ported.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs:47-320
func (s *State) ShowLoginDialog() {
	p, _ := s.barParams()
	if _, uerr := core.ParseIPAddress(p.Address); uerr != nil {
		s.ShowError(uerr.Primary, uerr.Secondary)
		return
	}
	prompt := widget.NewLabelWithStyle(core.LoginPrompt, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	ident := widget.NewLabel(core.LoginIdentityLine(p.Identity, p.SerialNumber))
	ident.Importance = widget.LowImportance
	labels := make([]string, len(core.DeviceUserLevels))
	for i, l := range core.DeviceUserLevels {
		labels[i] = l.Label
	}
	store := widget.NewCheck(core.LoginStoreCaption, nil)
	user := widget.NewSelect(labels, nil)
	pass := widget.NewPasswordEntry()
	progress := widget.NewProgressBar()
	progress.Hide()

	user.OnChanged = func(string) {
		if i := user.SelectedIndex(); i >= 0 && !core.StorePasswordAllowed(core.DeviceUserLevels[i].ID) {
			store.SetChecked(false)
			store.Hide()
		} else {
			store.Show()
		}
	}
	user.SetSelectedIndex(0)
	if u, pw, st := s.currentLoginStore().DeviceLoginDefaults(time.Now()); st {
		store.SetChecked(true)
		if i := core.UserLevelIndex(u); i >= 0 {
			user.SetSelectedIndex(i)
		}
		pass.SetText(pw)
	}
	if p.Password != "" {
		pass.SetText(p.Password)
	}

	var dlg *dialog.CustomDialog
	var okBtn, cancelBtn *widget.Button
	setSensitive := func(on bool) {
		for _, w := range []fyne.Disableable{user, pass, okBtn, cancelBtn} {
			if on {
				w.Enable()
			} else {
				w.Disable()
			}
		}
	}
	onOK := func() {
		level := core.UserLevelAdmin
		if i := user.SelectedIndex(); i >= 0 {
			level = core.DeviceUserLevels[i].ID
		}
		cp := p
		cp.Username, cp.Password = level, pass.Text
		setSensitive(false)
		progress.SetValue(0)
		progress.Show()
		start := time.Now()
		stop := make(chan struct{})
		go func() {
			t := time.NewTicker(100 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					f := core.LoginProgressFraction(time.Since(start), int(core.LoginTimeout/time.Second))
					s.uiDo(func() { progress.SetValue(f) })
					if f >= 1 {
						return
					}
				}
			}
		}()
		s.Connect(cp, func(res core.LoginResult, err error) {
			close(stop)
			if err != nil {
				progress.Hide()
				progress.SetValue(0)
				setSensitive(true)
				return
			}
			s.currentLoginStore().RecordDeviceLogin(cp.Username, cp.Password, store.Checked, time.Now())
			s.bar.user.SetSelectedIndex(core.UserLevelIndex(cp.Username))
			dlg.Hide()
		})
	}
	okBtn = widget.NewButton("Ok", onOK)
	okBtn.Importance = widget.HighImportance
	cancelBtn = widget.NewButton("Cancel", func() {
		s.SetStatus("%s", core.LoginCancelled)
		dlg.Hide()
	})
	pass.OnSubmitted = func(string) { onOK() }

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel(core.LoginUserLevel), user,
		widget.NewLabel(core.LoginPassword), pass)
	content := container.NewVBox(prompt, ident, form, store, progress,
		container.NewHBox(layout.NewSpacer(), cancelBtn, okBtn))
	dlg = dialog.NewCustomWithoutButtons(AppName, content, s.Window)
	dlg.Resize(fyne.NewSize(460, dlg.MinSize().Height))
	dlg.Show()
	s.Window.Canvas().Focus(pass)
}
