package settings

import (
	"encoding/base64"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPreferences(t *testing.T) {
	st := openTemp(t)
	if p := st.Preferences(); !p.AskConfirmExit || !p.AskSaveChanges {
		t.Fatalf("defaults %+v", p)
	}
	if err := st.SetPreferences(Preferences{AskConfirmExit: false, AskSaveChanges: true}); err != nil {
		t.Fatal(err)
	}
	st2, _ := Open(st.Dir())
	if p := st2.Preferences(); p.AskConfirmExit || !p.AskSaveChanges {
		t.Errorf("persisted %+v", p)
	}
}

func TestExitPrompt(t *testing.T) {
	on, off := Preferences{AskConfirmExit: true}, Preferences{AskConfirmExit: false}
	cases := []struct {
		p                     Preferences
		user, device, changes bool
		want                  ExitPrompt
	}{
		{on, false, true, true, ExitPrompt{}},
		{off, true, true, true, ExitPrompt{}},
		{on, true, true, true, ExitPrompt{true, MsgUnsavedChanges, MsgConfirmClose}},
		{on, true, false, true, ExitPrompt{true, MsgConfirmClose, ""}},
		{on, true, true, false, ExitPrompt{true, MsgConfirmClose, ""}},
	}
	for i, c := range cases {
		if got := c.p.ExitPrompt(c.user, c.device, c.changes); got != c.want {
			t.Errorf("case %d: %+v, want %+v", i, got, c.want)
		}
	}
}

func TestPendingChangesOnSwitch(t *testing.T) {
	ask, noask := Preferences{AskSaveChanges: true}, Preferences{AskSaveChanges: false}
	cases := []struct {
		p              Preferences
		changes, allow bool
		want           PendingChangesAction
	}{
		{ask, false, true, PendingNone},
		{ask, true, false, PendingNone},
		{ask, true, true, PendingAsk},
		{noask, true, true, PendingRevert},
		{noask, false, true, PendingNone},
	}
	for i, c := range cases {
		if got := c.p.PendingChangesOnSwitch(c.changes, c.allow); got != c.want {
			t.Errorf("case %d: %v, want %v", i, got, c.want)
		}
	}
	if q := SaveChangesQuestion("ACE0001"); q != "You have changed some properties for device 'ACE0001', do you want to save the changes?" {
		t.Errorf("question %q", q)
	}
}

func TestDialogTextsMatchSource(t *testing.T) {
	for file, texts := range map[string][]string{
		"ACEServiceInstaller/ICUServiceInstaller/DlgSettings.cs": {DlgSettingsTitle, LabelAskConfirmExit, LabelAskSaveChanges},
		"ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs": {MsgUnsavedChanges, MsgConfirmClose,
			"\"You have changed some properties for device '\" + m_currentDevice.Identity + \"', do you want to save the changes?\"",
			"Device with IP: {dlg.IPAddress} is already present in the overview and cannot be added.",
			"Device '{lanDev.Identification}' with IP: {dlg.IPAddress} sucessfully added to the overview.",
			"Device with IP: {dlg.IPAddress} could not be added to the overview.", MsgDeviceReAdded},
		"ACEServiceInstaller/ICUServiceInstaller/DlgLogon.cs": {MsgLogonPasswordsMismatch, MsgLogonPasswordChanged,
			strings.ReplaceAll(MsgLogonPasswordChangedDetail, "\n", `\n`)},
		"ACEServiceInstaller/ICUServiceInstaller/DlgDeviceLogin.cs": {strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(LabelStorePassword),
			`Items.Add("admin", "Owner")`, `Items.Add("temp", "Temporary")`, `Items.Add("service", "Secure Service Access")`},
		"ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs": {LabelManualIPHeader, LabelManualIPAddress, LabelManualPort,
			LabelManualModelType, LabelManualNumberOfSockets, LabelManualLoginRequired, PlaceholderManualIPAddress,
			`new Button("` + LabelManualSearch + `")`, MsgInvalidIPTitle, MsgInvalidIPDetail, MsgNoLinkLocalDevice,
			"A new device is found on ip: {item.Key} do you want to add that manually?",
			`"Manual IP address " + AppProperties.AppName`,
			"m_spbPort.MaximumValue = 65535.0", "m_spbPort.Value = 443.0", "m_spbPort.MinimumValue = 0.0",
			"m_spbNumberOfSockets.MinimumValue = 1.0", "m_spbNumberOfSockets.MaximumValue = 2.0", "m_spbNumberOfSockets.Value = 1.0"},
	} {
		src := readFixture(t, file)
		for _, s := range texts {
			if !strings.Contains(src, s) {
				t.Errorf("%s lacks %q", file, s)
			}
		}
	}
	for i, l := range DeviceUserLevels {
		want := []string{"admin", "temp", "service"}[i]
		if l.ID != want {
			t.Errorf("user level %d = %s", i, l.ID)
		}
	}
}

func TestLogonUserName(t *testing.T) {
	st := openTemp(t)
	if st.LogonUserName() != DefaultUserName || st.DisplayName() != "" {
		t.Fatalf("empty: %q %q", st.LogonUserName(), st.DisplayName())
	}
	if err := st.SetLastUserName("Installer1"); err != nil {
		t.Fatal(err)
	}
	if st.LogonUserName() != "Installer1" || st.DisplayName() != "Installer1" {
		t.Errorf("after set: %q %q", st.LogonUserName(), st.DisplayName())
	}
}

func TestHashUserName(t *testing.T) {
	h := HashUserName("SomeUser")
	if h != HashUserName("someuser") {
		t.Error("hash must be case-insensitive (ToLowerInvariant)")
	}
	raw, err := base64.StdEncoding.DecodeString(h)
	if err != nil || len(raw) != 32 {
		t.Errorf("not base64 SHA-256: %q", h)
	}
	if h == HashUserName("other") {
		t.Error("collision")
	}
}

func TestLocalPasswords(t *testing.T) {
	st := openTemp(t)
	u := HashUserName("u")
	if got := st.LocalPasswordJSONs(u); got != nil {
		t.Fatalf("empty store: %v", got)
	}
	// Normal entries ("hash;json") are never removed by the C# (> 2 parts
	// test), so a second change appends.
	_ = st.ReplaceLocalPassword(u, `{"User":"a"}`)
	_ = st.ReplaceLocalPassword(u, `{"User":"b"}`)
	if got := st.LocalPasswordJSONs(u); !reflect.DeepEqual(got, []string{`{"User":"a"}`, `{"User":"b"}`}) {
		t.Errorf("got %v", got)
	}
	// An entry with three parts and the same key is replaced (first match only).
	_ = st.Update(func(s *Settings) {
		s.LocalPasswords = []string{"x;1", u + ";old;extra", u + ";old2;extra", "y"}
	})
	_ = st.ReplaceLocalPassword(u, "new")
	want := []string{"x;1", u + ";old2;extra", "y", u + ";new"}
	if got := st.Get().LocalPasswords; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
	// Lookup uses parts[1] of entries with > 1 parts.
	if got := st.LocalPasswordJSONs(u); !reflect.DeepEqual(got, []string{"old2", "new"}) {
		t.Errorf("lookup %v", got)
	}
	if got := st.LocalPasswordJSONs("nobody"); got != nil {
		t.Errorf("unknown user %v", got)
	}
}

var (
	past   = time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	now    = time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)
	future = now.Add(25 * time.Hour)
)

func TestRecordDeviceLogin(t *testing.T) {
	st := openTemp(t)
	if err := st.RecordDeviceLogin(UserLevelTemp, "pw1", true, past); err != nil {
		t.Fatal(err)
	}
	s := st.Get()
	if s.LastDeviceUsername != "temp" || s.LastDevicePassword != "pw1" || !s.StorePasswords || !s.LastPasswordStoreTime.Equal(past) {
		t.Errorf("store on: %+v", s)
	}
	if err := st.RecordDeviceLogin(UserLevelTemp, "pw2", false, now); err != nil {
		t.Fatal(err)
	}
	s = st.Get()
	if s.LastDeviceUsername != "admin" || s.LastDevicePassword != "" || s.StorePasswords || !s.LastPasswordStoreTime.Equal(now) {
		t.Errorf("store off: %+v", s)
	}
}

func TestValidateStoredPassword(t *testing.T) {
	st := openTemp(t)
	_ = st.RecordDeviceLogin("temp", "pw", true, past)
	// Stored long ago: the reversed C# subtraction never expires it.
	if ok, err := st.ValidateStoredPassword(now); !ok || err != nil {
		t.Errorf("past: %v %v", ok, err)
	}
	if s := st.Get(); s.LastDevicePassword != "pw" {
		t.Error("password reset for a past timestamp")
	}
	// Exactly 24 h ahead: TotalHours > 24.0 is false.
	_ = st.Update(func(s *Settings) { s.LastPasswordStoreTime = now.Add(24 * time.Hour) })
	if ok, _ := st.ValidateStoredPassword(now); !ok {
		t.Error("24h ahead must still be valid")
	}
	// More than 24 h in the future: reset, false, StorePasswords untouched.
	_ = st.Update(func(s *Settings) { s.LastPasswordStoreTime = future })
	if ok, err := st.ValidateStoredPassword(now); ok || err != nil {
		t.Errorf("future: %v %v", ok, err)
	}
	s := st.Get()
	if s.LastDeviceUsername != "admin" || s.LastDevicePassword != "" || !s.StorePasswords {
		t.Errorf("after reset %+v", s)
	}
	st2, _ := Open(st.Dir())
	if st2.Get().LastDevicePassword != "" {
		t.Error("reset not saved")
	}
	// StorePasswords off -> false.
	_ = st.RecordDeviceLogin("admin", "x", false, past)
	if ok, _ := st.ValidateStoredPassword(now); ok {
		t.Error("StorePasswords off must return false")
	}
}

func TestLoginPrefill(t *testing.T) {
	st := openTemp(t)
	sess := &Credentials{Username: "temp", Password: "sess"}
	// StorePasswords off: nothing.
	if _, ok, _ := st.LoginPrefill(now, false, true, sess); ok {
		t.Error("prefill with StorePasswords off")
	}
	_ = st.RecordDeviceLogin("temp", "stored", true, past)
	cases := []struct {
		force, auto bool
		sess        *Credentials
		want        Credentials
		ok          bool
	}{
		{false, true, nil, Credentials{"temp", "stored"}, true},
		{false, true, sess, *sess, true},
		{true, true, sess, Credentials{}, false},
		{false, false, nil, Credentials{}, false},
	}
	for i, c := range cases {
		got, ok, err := st.LoginPrefill(now, c.force, c.auto, c.sess)
		if got != c.want || ok != c.ok || err != nil {
			t.Errorf("case %d: %+v %v %v", i, got, ok, err)
		}
	}
}

func TestDeviceLoginDefaults(t *testing.T) {
	st := openTemp(t)
	f, err := st.DeviceLoginDefaults(now, Credentials{})
	want := DeviceLoginForm{UserLevel: "admin", StorePasswordVisible: true}
	if f != want || err != nil {
		t.Errorf("fresh: %+v %v", f, err)
	}
	_ = st.RecordDeviceLogin("temp", "stored", true, past)
	f, _ = st.DeviceLoginDefaults(now, Credentials{})
	want = DeviceLoginForm{UserLevel: "temp", Password: "stored", StorePassword: true, StorePasswordVisible: true}
	if f != want {
		t.Errorf("stored: %+v", f)
	}
	// Preset (device LoginData) wins and auto-submits.
	f, _ = st.DeviceLoginDefaults(now, Credentials{Username: "admin", Password: "p"})
	want = DeviceLoginForm{UserLevel: "admin", Password: "p", StorePassword: true, StorePasswordVisible: true, AutoSubmit: true}
	if f != want {
		t.Errorf("preset: %+v", f)
	}
	// Incomplete preset is ignored.
	f, _ = st.DeviceLoginDefaults(now, Credentials{Username: "admin"})
	if f.AutoSubmit || f.Password != "stored" {
		t.Errorf("half preset: %+v", f)
	}
	// "service" unticks and hides the store box.
	f, _ = st.DeviceLoginDefaults(now, Credentials{Username: "service", Password: "s"})
	want = DeviceLoginForm{UserLevel: "service", Password: "s", AutoSubmit: true}
	if f != want {
		t.Errorf("service: %+v", f)
	}
	_ = st.Update(func(s *Settings) { s.LastDeviceUsername = "service" })
	f, _ = st.DeviceLoginDefaults(now, Credentials{})
	if f.StorePassword || f.StorePasswordVisible || f.UserLevel != "service" {
		t.Errorf("stored service: %+v", f)
	}
	// "service" then a preset "admin": the box shows again but stays unticked.
	f, _ = st.DeviceLoginDefaults(now, Credentials{Username: "admin", Password: "p"})
	want = DeviceLoginForm{UserLevel: "admin", Password: "p", StorePasswordVisible: true, AutoSubmit: true}
	if f != want {
		t.Errorf("service then admin: %+v", f)
	}
}

// A user level that is not a combo item (e.g. a group HTTP user left in
// LoginData) leaves no selection; the store-password box keeps its state
// (the SelectionChanged handler throws before acting) and Submit refuses,
// as the C# NullReferenceException path cancels the dialog.
func TestDeviceLoginUnknownUserLevel(t *testing.T) {
	st := openTemp(t)
	_ = st.RecordDeviceLogin("temp", "stored", true, past)
	f, _ := st.DeviceLoginDefaults(now, Credentials{Username: "grouphttpuser", Password: "p"})
	want := DeviceLoginForm{UserLevel: "", Password: "p", StorePassword: true, StorePasswordVisible: true, AutoSubmit: true}
	if f != want {
		t.Errorf("unknown preset: %+v", f)
	}
	if _, err := f.Submit(); !errors.Is(err, ErrNoUserLevel) {
		t.Errorf("Submit with no selection: %v", err)
	}
	_ = st.Update(func(s *Settings) { s.LastDeviceUsername = "olduser" })
	f, _ = st.DeviceLoginDefaults(now, Credentials{})
	want = DeviceLoginForm{UserLevel: "", Password: "stored", StorePassword: true, StorePasswordVisible: true}
	if f != want {
		t.Errorf("unknown stored level: %+v", f)
	}
	// Picking a level in the view selects it again.
	f.SelectUserLevel(UserLevelTemp)
	if cr, err := f.Submit(); err != nil || cr != (Credentials{"temp", "stored"}) {
		t.Errorf("after selection: %+v %v", cr, err)
	}
	// Moving from "service" to an unknown level leaves the box hidden.
	f.SelectUserLevel(UserLevelService)
	f.SelectUserLevel("bogus")
	if f.UserLevel != "" || f.StorePassword || f.StorePasswordVisible {
		t.Errorf("service then unknown: %+v", f)
	}
}

func TestUpdateStoredDevicePassword(t *testing.T) {
	st := openTemp(t)
	if err := st.UpdateStoredDevicePassword(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(st.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Error("empty password must be a no-op")
	}
	if err := st.UpdateStoredDevicePassword("new"); err != nil {
		t.Fatal(err)
	}
	if s := st.Get(); s.LastDevicePassword != "new" || s.StorePasswords {
		t.Errorf("%+v", s)
	}
}

func TestSessionCredentials(t *testing.T) {
	st := openTemp(t)
	c := NewSessionCredentials()
	if c.Get("10.0.0.1") != nil {
		t.Fatal("non-empty")
	}
	c.RememberAfterDialog(st, "10.0.0.1", Credentials{"admin", "x"})
	if c.Get("10.0.0.1") != nil {
		t.Error("remembered with StorePasswords off")
	}
	_ = st.RecordDeviceLogin("admin", "x", true, past)
	c.RememberAfterDialog(st, "10.0.0.1", Credentials{}) // cancelled dialog stores the empty pair
	if got := c.Get("10.0.0.1"); got == nil || *got != (Credentials{}) {
		t.Errorf("cancelled: %v", got)
	}
	c.Set("10.0.0.1", Credentials{"admin", "y"})
	if got := c.Get("10.0.0.1"); got == nil || got.Password != "y" {
		t.Errorf("set: %v", got)
	}
	c.Delete("10.0.0.1")
	if c.Get("10.0.0.1") != nil {
		t.Error("delete")
	}
	if s, _ := Open(st.Dir()); s.Get().LastDevicePassword != "x" {
		t.Error("session credentials must not be persisted")
	}
}

func TestSessionCredentialsZeroValue(t *testing.T) {
	var c SessionCredentials
	c.Set("d", Credentials{"admin", "x"})
	c.SetLoginData("d", Credentials{"admin", "x"})
	c.AfterPasswordChange("e", "y")
	c.Logout("e")
	c.Delete("e")
	if got := c.Get("d"); got == nil || got.Password != "x" || c.LoginData("d").Password != "x" {
		t.Errorf("zero value: %v %v", got, c.LoginData("d"))
	}
	var c2 SessionCredentials
	c2.FinishLoginDialog(openTemp(t), "d", true, Credentials{"admin", "x"})
	var c3 SessionCredentials
	c3.AssignDisplayName(openTemp(t), "d")
}

// MainWindow.OnLoginRequest leaves LoginData alone when LoginPrefill does not
// apply (StorePasswords off), so the next login dialog for the same device
// submits itself with the previous dialog credentials; only a cancelled
// dialog clears them.
func TestLoginDataAutoSubmitsWithStorePasswordsOff(t *testing.T) {
	st := openTemp(t)
	c := NewSessionCredentials()
	const dev = "10.0.0.1"
	f, err := c.PrepareDeviceLogin(st, dev, now, false, true)
	if err != nil || f.AutoSubmit || f.UserLevel != "admin" || f.Password != "" {
		t.Fatalf("first dialog: %+v %v", f, err)
	}
	// User enters temp/pw, store box unticked; login succeeds.
	f.SelectUserLevel(UserLevelTemp)
	f.Password = "pw"
	cred, err := f.Submit()
	if err != nil {
		t.Fatal(err)
	}
	c.SetLoginData(dev, cred)
	if err := st.RecordDeviceLogin(cred.Username, cred.Password, f.StorePassword, now); err != nil {
		t.Fatal(err)
	}
	c.FinishLoginDialog(st, dev, true, cred)
	if c.Get(dev) != nil {
		t.Error("UserData set with StorePasswords off")
	}
	if s := st.Get(); s.LastDevicePassword != "" || s.StorePasswords {
		t.Errorf("password persisted: %+v", s)
	}
	// Re-login (session expiry, re-selecting the device): automatic and not.
	for _, auto := range []bool{true, false} {
		f, err = c.PrepareDeviceLogin(st, dev, now, false, auto)
		want := DeviceLoginForm{UserLevel: "temp", Password: "pw", StorePasswordVisible: true, AutoSubmit: true}
		if err != nil || f != want {
			t.Errorf("auto=%v: %+v %v", auto, f, err)
		}
	}
	// Another device has no LoginData yet.
	if f, _ = c.PrepareDeviceLogin(st, "10.0.0.2", now, false, true); f.AutoSubmit {
		t.Errorf("other device: %+v", f)
	}
	// A cancelled dialog clears LoginData.
	c.FinishLoginDialog(st, dev, false, Credentials{"temp", "pw"})
	if c.LoginData(dev) != (Credentials{}) {
		t.Errorf("cancel kept %+v", c.LoginData(dev))
	}
	if f, _ = c.PrepareDeviceLogin(st, dev, now, false, true); f.AutoSubmit {
		t.Errorf("after cancel: %+v", f)
	}
}

func TestPrepareDeviceLoginUsesPrefill(t *testing.T) {
	st := openTemp(t)
	c := NewSessionCredentials()
	_ = st.RecordDeviceLogin("temp", "stored", true, past)
	// StorePasswords on, automatic attempt: Last* become LoginData.
	f, err := c.PrepareDeviceLogin(st, "d", now, false, true)
	want := DeviceLoginForm{UserLevel: "temp", Password: "stored", StorePassword: true, StorePasswordVisible: true, AutoSubmit: true}
	if err != nil || f != want || c.LoginData("d") != (Credentials{"temp", "stored"}) {
		t.Errorf("prefill: %+v %v %+v", f, err, c.LoginData("d"))
	}
	// UserData wins over the stored slot.
	c.Set("d", Credentials{"admin", "sess"})
	if f, _ = c.PrepareDeviceLogin(st, "d", now, false, true); f.UserLevel != "admin" || f.Password != "sess" || !f.AutoSubmit {
		t.Errorf("user data: %+v", f)
	}
	// Forced dialog: LoginData is left as it is (still auto-submits).
	if f, _ = c.PrepareDeviceLogin(st, "d", now, true, true); !f.AutoSubmit || f.Password != "sess" {
		t.Errorf("forced: %+v", f)
	}
}

func TestFinishLoginDialogStoresUserData(t *testing.T) {
	st := openTemp(t)
	c := NewSessionCredentials()
	_ = st.SetLastUserName("Installer1")
	_ = st.RecordDeviceLogin("admin", "pw", true, now)
	c.FinishLoginDialog(st, "d", true, Credentials{"admin", "pw"})
	if got := c.Get("d"); got == nil || *got != (Credentials{"admin", "pw"}) {
		t.Errorf("user data %v", got)
	}
	c.FinishLoginDialog(st, "d", false, Credentials{"admin", "pw"})
	if got := c.Get("d"); got == nil || *got != (Credentials{}) {
		t.Errorf("cancelled user data %v", got)
	}
}

// DisplayName is copied from LastUserName only after a login, so the first
// /api/login for a device sends an empty displayname.
func TestDisplayNameAssignedAfterLogin(t *testing.T) {
	st := openTemp(t)
	c := NewSessionCredentials()
	_ = st.SetLastUserName("Installer1")
	if c.DisplayName("d") != "" {
		t.Fatal("display name before the first login")
	}
	c.FinishLoginDialog(st, "d", true, Credentials{"admin", "pw"})
	if c.DisplayName("d") != "Installer1" {
		t.Errorf("after dialog: %q", c.DisplayName("d"))
	}
	// Group (non-unique password) branch.
	c.SetLoginData("g", Credentials{"grp", "x"})
	if c.DisplayName("g") != "" {
		t.Error("group: display name before login")
	}
	c.AssignDisplayName(st, "g")
	_ = st.SetLastUserName("Installer2")
	if c.DisplayName("g") != "Installer1" {
		t.Errorf("group: %q (must keep the value copied at the last login)", c.DisplayName("g"))
	}
	// SetLoginData and Logout keep DisplayName; Delete forgets it.
	c.SetLoginData("d", Credentials{"temp", "t"})
	c.Logout("d")
	if c.DisplayName("d") != "Installer1" || c.LoginData("d") != (Credentials{}) {
		t.Errorf("logout: %q %+v", c.DisplayName("d"), c.LoginData("d"))
	}
	c.Delete("d")
	if c.DisplayName("d") != "" {
		t.Error("delete kept display name")
	}
}

func TestLogoutAndPasswordChange(t *testing.T) {
	c := NewSessionCredentials()
	c.Set("d", Credentials{"admin", "old"})
	c.SetLoginData("d", Credentials{"admin", "old"})
	c.AfterPasswordChange("d", "new")
	if c.LoginData("d").Password != "new" || c.Get("d").Password != "old" {
		t.Errorf("admin change: %+v %+v", c.LoginData("d"), c.Get("d"))
	}
	c.SetLoginData("t", Credentials{"temp", "old"})
	c.AfterPasswordChange("t", "new")
	if c.LoginData("t").Password != "old" {
		t.Error("non-admin password replaced")
	}
	c.Logout("d")
	if c.Get("d") != nil || c.LoginData("d") != (Credentials{}) {
		t.Errorf("logout: %v %+v", c.Get("d"), c.LoginData("d"))
	}
}
