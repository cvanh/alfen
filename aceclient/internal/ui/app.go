// Package ui is the Fyne v2 GUI of the Go port of the ACE Service Installer
// (ACEServiceInstaller.exe, .NET/Xwt). It ports the MainWindow shell — the
// connection controls, the tab host (MainWindowBase/FillPanelList), the
// Save/Revert/Exit bar and the status line — and hosts one view per installer
// panel or dialog, each in its own file and registered via RegisterPanel.
//
// Views are thin: device and protocol logic lives in internal/ui/core (no
// Fyne) and the internal/* backend packages. Views never perform HTTP/UDP on
// the UI goroutine; they use State.RunAsync and touch widgets only in the
// completion callback, which runs on the UI goroutine via fyne.Do.
package ui

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/ui/core"
)

// AppName is AppProperties.AppName (the window title).
const AppName = "ACE Service Installer"

// AppID is the neutral application ID (preferences/storage key, FyneApp.toml).
const AppID = "io.github.aceclient.installer"

//go:embed assets/Icon.png
var iconPNG []byte

// Icon is the application icon (ACEServiceInstaller Resources Icon.png).
var Icon = fyne.NewStaticResource("Icon.png", iconPNG)

func init() {
	// MainWindow's "_File > _Close" (Close → HandleCloseRequested).
	RegisterMenuItem(MenuItemSpec{Menu: "File", Label: "Close", Order: 900, Action: func(s *State) { s.handleCloseRequested() }})
}

// Options configure the shell.
type Options struct {
	// AutoDiscover starts the SCN listener when the window is built, like
	// MainWindow starting the LAN search after logon.
	AutoDiscover bool
}

// NewApp creates the Fyne application with the installer's ID and icon.
func NewApp() fyne.App {
	a := app.NewWithID(AppID)
	a.SetIcon(Icon)
	return a
}

// State is the shared shell state handed to every panel: the current device
// session, connection status, hooks, the status line and async helpers.
// Methods that touch widgets must be called on the UI goroutine unless noted.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs (class MainWindow)
type State struct {
	App     fyne.App
	Window  fyne.Window
	Devices *core.DeviceList // discovered devices (MainWindow.m_colDevices)

	opts Options
	reg  *registry

	mu          sync.RWMutex
	session     *core.Session
	connected   bool
	rights      RightsFunc
	eds         core.EDSLookup
	scnName     string
	loginStore  LoginStore
	displayName string
	// protocol forces the session protocol ("" = core.ProtocolForPort).
	protocol string

	hooksMu      sync.Mutex
	onConnect    []func(*State)
	onDisconnect []func(*State)
	onProps      []func(*State)
	onShown      map[string][]func(*State)
	onClose      []func()

	wg   sync.WaitGroup
	busy atomic.Int32
	uiMu sync.Mutex

	bar       *connectionBar
	status    *widget.Label
	connInfo  *widget.Label
	activity  *widget.ProgressBarInfinite
	saveBtn   *widget.Button
	revertBtn *widget.Button
	tabs      *container.AppTabs
	hosts     map[string]*panelHost
	views     map[string]any
	order     []*panelHost
	saving    bool
}

// panelHost wraps one registered page: its built content and, for pages that
// need a device, the PanelNotLoggedIn placeholder.
type panelHost struct {
	spec        PanelSpec
	item        *container.TabItem
	content     fyne.CanvasObject
	placeholder fyne.CanvasObject
	stack       *fyne.Container
}

// NewMainWindow builds the main window (MainWindow's constructor) on a and
// returns its State; call State.Window.ShowAndRun() to start.
func NewMainWindow(a fyne.App, opts Options) *State {
	return newMainWindow(a, opts, defaultRegistry)
}

func newMainWindow(a fyne.App, opts Options, reg *registry) *State {
	s := &State{
		App:        a,
		Devices:    core.NewDeviceList(),
		opts:       opts,
		reg:        reg,
		loginStore: &memoryLoginStore{username: core.UserLevelAdmin},
		onShown:    map[string][]func(*State){},
		hosts:      map[string]*panelHost{},
		views:      map[string]any{},
	}
	w := a.NewWindow(AppName)
	w.SetIcon(Icon)
	s.Window = w

	s.status = widget.NewLabel("")
	s.status.Truncation = fyne.TextTruncateEllipsis
	s.connInfo = widget.NewLabel("Not connected")
	s.activity = widget.NewProgressBarInfinite()
	s.activity.Hide()
	s.revertBtn = widget.NewButton("Revert", s.revertAll)
	s.saveBtn = widget.NewButton("Save", s.saveAll)
	s.revertBtn.Disable()
	s.saveBtn.Disable()

	s.bar = s.newConnectionBar()
	s.tabs = container.NewAppTabs()
	s.tabs.SetTabLocation(container.TabLocationTop)
	s.tabs.OnSelected = func(ti *container.TabItem) { s.firePanelShown(ti) }
	for _, spec := range reg.panelList() {
		h := &panelHost{spec: spec}
		h.content = spec.Build(s)
		if spec.RequiresConnection {
			h.placeholder = s.notLoggedInPlaceholder()
		}
		h.stack = container.NewStack()
		h.item = container.NewTabItem(spec.Title, h.stack)
		s.hosts[spec.ID] = h
		s.order = append(s.order, h)
	}
	s.fillPanelList()
	s.refreshPanelContent()

	statusBar := container.NewBorder(nil, nil, nil,
		container.NewHBox(s.activity, s.connInfo, s.revertBtn, s.saveBtn),
		s.status)
	w.SetContent(container.NewBorder(
		container.NewVBox(s.bar.obj, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), statusBar),
		nil, nil, s.tabs))
	s.rebuildMenu()
	w.SetCloseIntercept(s.handleCloseRequested)
	w.Resize(fyne.NewSize(1100, 720))
	return s
}

// notLoggedInPlaceholder ports PanelNotLoggedIn: the label and a "Login..."
// button ("Login to the charging station") that re-opens the login dialog
// (MainWindow.ReselectCurrentItem(showLoginDialog: true)).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelNotLoggedIn.cs
func (s *State) notLoggedInPlaceholder() fyne.CanvasObject {
	lbl := widget.NewLabel(core.NotLoggedInText)
	lbl.Wrapping = fyne.TextWrapWord
	btn := widget.NewButton("Login...", func() { s.ShowLoginDialog() })
	return container.NewVBox(lbl, container.NewHBox(btn, layout.NewSpacer()))
}

// ---------------------------------------------------------------- accessors

// Session returns the current device session, nil when none.
func (s *State) Session() *core.Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session
}

// Client returns the current api client, nil when not connected.
func (s *State) Client() *api.Client {
	if sess := s.Session(); sess != nil {
		return sess.Client
	}
	return nil
}

// Connected reports whether a device is logged in.
func (s *State) Connected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// LoginData returns the credentials of the current session.
func (s *State) LoginData() api.LoginData {
	if sess := s.Session(); sess != nil {
		return sess.LoginData()
	}
	return api.LoginData{}
}

// Props returns the property cache of the current session (nil when none).
func (s *State) Props() *core.PropertyCache {
	if sess := s.Session(); sess != nil {
		return sess.Props
	}
	return nil
}

// SetEDS installs the EDS lookup (internal/eds, Phase 2) for this and all
// future sessions. Safe from any goroutine.
func (s *State) SetEDS(l core.EDSLookup) {
	s.mu.Lock()
	s.eds = l
	sess := s.session
	s.mu.Unlock()
	if sess != nil {
		sess.Props.SetEDS(l)
	}
}

// SetLoginStore installs the persistence for the device-login dialog
// (Settings.Default LastDeviceUsername/Password/StorePasswords/LastUserName).
func (s *State) SetLoginStore(ls LoginStore) {
	s.mu.Lock()
	s.loginStore = ls
	s.mu.Unlock()
}

// SetDisplayName overrides the login "displayname" (Settings.LastUserName).
func (s *State) SetDisplayName(name string) {
	s.mu.Lock()
	s.displayName = name
	s.mu.Unlock()
}

// SetRights installs the feature-rights hook (ICUUser.GetRights) and
// re-filters the tabs (FillPanelList). UI goroutine.
func (s *State) SetRights(f RightsFunc) {
	s.mu.Lock()
	s.rights = f
	s.mu.Unlock()
	s.fillPanelList()
	s.rebuildMenu()
}

// Rights returns the rights for a page feature ID: Full without a hook or
// without a feature ID.
func (s *State) Rights(featureID string) Rights {
	s.mu.RLock()
	f := s.rights
	s.mu.RUnlock()
	if f == nil || featureID == "" {
		return RightsFull
	}
	return f(featureID)
}

// CurrentSCN returns the SCN network whose pages are shown ("" in device mode).
func (s *State) CurrentSCN() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scnName
}

// ShowSCN switches the tab set to the SCN pages for networkName
// (MainWindow.UpdatePanels → FillPanelList(fSCN: true)); "" switches back to
// the device pages. UI goroutine.
func (s *State) ShowSCN(networkName string) {
	s.mu.Lock()
	s.scnName = networkName
	s.mu.Unlock()
	s.fillPanelList()
}

// ------------------------------------------------------------------- hooks

// OnConnect registers fn to run (UI goroutine) after every successful login.
func (s *State) OnConnect(fn func(*State)) {
	s.hooksMu.Lock()
	s.onConnect = append(s.onConnect, fn)
	s.hooksMu.Unlock()
}

// OnDisconnect registers fn to run (UI goroutine) after the device is logged out.
func (s *State) OnDisconnect(fn func(*State)) {
	s.hooksMu.Lock()
	s.onDisconnect = append(s.onDisconnect, fn)
	s.hooksMu.Unlock()
}

// OnPropertiesChanged registers fn to run (UI goroutine) whenever
// NotifyPropertiesChanged is called (reads merged, edits, saves, reverts).
func (s *State) OnPropertiesChanged(fn func(*State)) {
	s.hooksMu.Lock()
	s.onProps = append(s.onProps, fn)
	s.hooksMu.Unlock()
}

// OnPanelShown registers fn to run when the page panelID becomes the selected
// tab, and after a login while it is selected (PanelBase.OnShowPanel).
func (s *State) OnPanelShown(panelID string, fn func(*State)) {
	s.hooksMu.Lock()
	s.onShown[panelID] = append(s.onShown[panelID], fn)
	s.hooksMu.Unlock()
}

func (s *State) hooks(list *[]func(*State)) []func(*State) {
	s.hooksMu.Lock()
	defer s.hooksMu.Unlock()
	return append(([]func(*State))(nil), (*list)...)
}

// NotifyPropertiesChanged runs the OnPropertiesChanged hooks and updates the
// Save/Revert buttons (MainWindow.CheckIfPropertiesHasChanged). UI goroutine.
func (s *State) NotifyPropertiesChanged() {
	s.updateSaveButtons()
	for _, fn := range s.hooks(&s.onProps) {
		fn(s)
	}
}

func (s *State) firePanelShown(ti *container.TabItem) {
	if ti == nil {
		return
	}
	for id, h := range s.hosts {
		if h.item == ti {
			s.hooksMu.Lock()
			fns := append(([]func(*State))(nil), s.onShown[id]...)
			s.hooksMu.Unlock()
			for _, fn := range fns {
				fn(s)
			}
		}
	}
}

// setView/view keep each page's view object (for menu actions and tests).
func (s *State) setView(id string, v any) {
	s.mu.Lock()
	s.views[id] = v
	s.mu.Unlock()
}

func (s *State) view(id string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.views[id]
}

// SelectPanel makes the page panelID the current tab (if visible).
func (s *State) SelectPanel(panelID string) {
	if h, ok := s.hosts[panelID]; ok {
		for _, it := range s.tabs.Items {
			if it == h.item {
				s.tabs.Select(it)
				return
			}
		}
	}
}

// --------------------------------------------------------- status & async

// SetStatus writes the status line. UI goroutine.
func (s *State) SetStatus(format string, args ...any) {
	s.status.SetText(fmt.Sprintf(format, args...))
}

// ShowError shows MessageDialog.ShowError(primary[, secondary]). UI goroutine.
func (s *State) ShowError(primary string, secondary ...string) {
	msg := primary
	if len(secondary) > 0 && secondary[0] != "" {
		msg += "\n" + secondary[0]
	}
	dialog.ShowError(fmt.Errorf("%s", msg), s.Window)
}

// ShowMessage shows MessageDialog.ShowMessage. UI goroutine.
func (s *State) ShowMessage(text string) {
	dialog.ShowInformation(AppName, text, s.Window)
}

// AskQuestion shows a Yes/No question (MessageDialog.AskQuestion with
// Command.Yes/No) and calls answer(true) only on Yes. UI goroutine.
func (s *State) AskQuestion(text string, answer func(yes bool)) {
	d := dialog.NewConfirm(AppName, text, answer, s.Window)
	d.SetConfirmText("Yes")
	d.SetDismissText("No")
	d.Show()
}

// RunAsync runs work on a new goroutine (never the UI goroutine) and then
// done, if non-nil, on the UI goroutine via fyne.Do. While work runs the
// status-bar activity indicator is shown. Results travel through the
// closures. Call it from the UI goroutine (handlers, hooks, completions).
func (s *State) RunAsync(work func(), done func()) {
	s.wg.Add(1)
	s.busy.Add(1)
	s.refreshActivity()
	go func() {
		work()
		s.uiDo(func() {
			defer s.wg.Done()
			s.busy.Add(-1)
			s.refreshActivity()
			if done != nil {
				done()
			}
		})
	}()
}

// uiDo runs fn on the UI goroutine via fyne.Do, serialised with every other
// asynchronous completion (the headless test driver runs fyne.Do callbacks
// on the calling goroutine, so they are not otherwise serialised). Call it
// only from background goroutines.
func (s *State) uiDo(fn func()) {
	fyne.Do(func() {
		s.uiMu.Lock()
		defer s.uiMu.Unlock()
		fn()
	})
}

// Wait blocks until every RunAsync started so far has finished (tests).
func (s *State) Wait() { s.wg.Wait() }

func (s *State) refreshActivity() {
	if s.busy.Load() > 0 {
		s.activity.Show()
		s.activity.Start()
	} else {
		s.activity.Stop()
		s.activity.Hide()
	}
}

// ------------------------------------------------------- tabs and menus

// fillPanelList ports MainWindow.FillPanelList: the visible pages for the
// current mode whose feature rights are not None, in registry order.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:839-878
func (s *State) fillPanelList() {
	scn := s.CurrentSCN() != ""
	var items []*container.TabItem
	for _, h := range s.order {
		switch h.spec.Mode {
		case PanelModeDevice:
			if scn {
				continue
			}
		case PanelModeSCN:
			if !scn {
				continue
			}
		}
		if h.spec.FeatureID != "" && s.Rights(h.spec.FeatureID) == RightsNone {
			continue
		}
		items = append(items, h.item)
	}
	cur := s.tabs.Selected()
	s.tabs.SetItems(items)
	for _, it := range items {
		if it == cur {
			s.tabs.Select(it)
		}
	}
}

// refreshPanelContent swaps every page between its content and the
// PanelNotLoggedIn placeholder according to the connection state.
func (s *State) refreshPanelContent() {
	connected := s.Connected()
	for _, h := range s.order {
		obj := h.content
		if h.spec.RequiresConnection && !connected {
			obj = h.placeholder
		}
		if len(h.stack.Objects) != 1 || h.stack.Objects[0] != obj {
			h.stack.Objects = []fyne.CanvasObject{obj}
			h.stack.Refresh()
		}
	}
}

// rebuildMenu builds the main menu from the menu registry; entries that need
// a device are disabled while disconnected.
func (s *State) rebuildMenu() {
	names, by := s.reg.menuList()
	connected := s.Connected()
	var menus []*fyne.Menu
	for _, n := range names {
		var items []*fyne.MenuItem
		for _, m := range by[n] {
			if m.FeatureID != "" && s.Rights(m.FeatureID) == RightsNone {
				continue
			}
			spec := m
			it := fyne.NewMenuItem(spec.Label, func() { spec.Action(s) })
			it.Disabled = spec.RequiresConnection && !connected
			items = append(items, it)
		}
		if len(items) > 0 {
			menus = append(menus, fyne.NewMenu(n, items...))
		}
	}
	if len(menus) > 0 {
		s.Window.SetMainMenu(fyne.NewMainMenu(menus...))
	}
}

// --------------------------------------------------------- save / revert

func (s *State) updateSaveButtons() {
	pc := s.Props()
	changed := s.Connected() && pc != nil && pc.HasChanges() && !s.saving
	if changed {
		s.saveBtn.Enable()
		s.revertBtn.Enable()
	} else {
		s.saveBtn.Disable()
		s.revertBtn.Disable()
	}
}

// revertAll ports MainWindow.RevertAllChanges (ICUDevice.RevertChanges).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1562-1587
func (s *State) revertAll() {
	if pc := s.Props(); pc != nil {
		pc.Revert()
	}
	s.NotifyPropertiesChanged()
}

// saveAll ports MainWindow.SaveAllChanges: every registered panel check
// (OnSaveChanges) first — an error aborts, a question must be answered Yes —
// then core.Session.SaveAllChanges off the UI goroutine.
func (s *State) saveAll() {
	s.SaveAllChanges(nil)
}

// SaveAllChanges runs the shell's Save (see saveAll); done (optional) gets the
// outcome on the UI goroutine. UI goroutine.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1589-1660
func (s *State) SaveAllChanges(done func(core.SaveResult, error)) {
	sess := s.Session()
	if sess == nil || !s.Connected() {
		return
	}
	checks := s.reg.saveChecks()
	var step func(i int)
	step = func(i int) {
		if i == len(checks) {
			s.runSave(sess, done)
			return
		}
		errText, question := checks[i](s)
		switch {
		case errText != "":
			s.ShowError(errText)
			if done != nil {
				done(core.SaveResult{}, fmt.Errorf("%s", errText))
			}
		case question != "":
			s.AskQuestion(question, func(yes bool) {
				if yes {
					step(i + 1)
				} else if done != nil {
					done(core.SaveResult{}, fmt.Errorf("save cancelled"))
				}
			})
		default:
			step(i + 1)
		}
	}
	step(0)
}

func (s *State) runSave(sess *core.Session, done func(core.SaveResult, error)) {
	s.saving = true
	s.updateSaveButtons()
	var res core.SaveResult
	var err error
	s.SetStatus("Saving changes to %s...", sess.Identification())
	s.RunAsync(func() {
		res, err = sess.SaveAllChanges(time.Now())
	}, func() {
		s.saving = false
		if err != nil {
			s.SetStatus("Save failed: %v", err)
			s.ShowError(requestErrorText(err))
		} else {
			s.SetStatus("Changes saved to %s.", sess.Identification())
			if res.ConfigurationNotice != "" {
				s.ShowMessage(res.ConfigurationNotice)
			}
			if res.IdentityOrModelChanged {
				s.Devices.Remove(sess.Address())
				s.ShowMessage(core.IdentityOrModelChangedText)
			}
		}
		s.NotifyPropertiesChanged()
		if done != nil {
			done(res, err)
		}
	})
}

// requestErrorText prefers the installer's popup text of a RequestError.
func requestErrorText(err error) string {
	if re, ok := err.(*core.RequestError); ok && re.Message != "" {
		return re.Message
	}
	return err.Error()
}

// handleCloseRequested ports MainWindow.HandleCloseRequested with
// AskConfirmExit (default True): ask, log the device out, quit.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1504-1531
func (s *State) handleCloseRequested() {
	q := "Are you sure you want to close this application?"
	if pc := s.Props(); s.Connected() && pc != nil && pc.HasChanges() {
		q = "You have unsaved changed!\n" + q
	}
	s.AskQuestion(q, func(yes bool) {
		if !yes {
			return
		}
		s.runCloseHooks()
		sess := s.Session()
		if sess == nil {
			s.App.Quit()
			return
		}
		s.RunAsync(func() { sess.Logout() }, func() { s.App.Quit() })
	})
}

// OnClose registers fn to run (UI goroutine) when the application is about
// to quit, e.g. to release the SCN socket.
func (s *State) OnClose(fn func()) {
	s.hooksMu.Lock()
	s.onClose = append(s.onClose, fn)
	s.hooksMu.Unlock()
}

func (s *State) runCloseHooks() {
	s.hooksMu.Lock()
	fns := append([]func(){}, s.onClose...)
	s.hooksMu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// ----------------------------------------------------- connection bar UI

// connectionBar is the shell's connection controls: IP, port (443), user
// level, password, insecure TLS, Connect/Disconnect.
type connectionBar struct {
	ip         *widget.Entry
	port       *widget.Entry
	user       *widget.Select
	pass       *widget.Entry
	insecure   *widget.Check
	connect    *widget.Button
	disconnect *widget.Button
	obj        fyne.CanvasObject

	target core.Device // last discovery selection (identity/serial/hostname)
}

func (s *State) newConnectionBar() *connectionBar {
	b := &connectionBar{}
	b.ip = widget.NewEntry()
	b.ip.SetPlaceHolder("xxx.xxx.xxx.xxx") // DlgManualIP placeholder
	b.port = widget.NewEntry()
	b.port.SetText("443") // Settings.LastManualIPPort default
	labels := make([]string, len(core.DeviceUserLevels))
	for i, l := range core.DeviceUserLevels {
		labels[i] = l.Label
	}
	b.user = widget.NewSelect(labels, nil)
	b.user.SetSelectedIndex(0) // "admin" (Owner)
	b.pass = widget.NewPasswordEntry()
	b.pass.SetPlaceHolder("Password")
	b.insecure = widget.NewCheck("Insecure TLS", nil)
	b.insecure.SetChecked(true)
	b.connect = widget.NewButton("Connect", func() { s.connectFromBar() })
	b.connect.Importance = widget.HighImportance
	b.disconnect = widget.NewButton("Disconnect", func() { s.Disconnect(nil) })
	b.disconnect.Disable()
	b.pass.OnSubmitted = func(string) { s.connectFromBar() }
	b.ip.OnSubmitted = func(string) { s.connectFromBar() }

	portBox := container.NewGridWrap(fyne.NewSize(70, b.port.MinSize().Height), b.port)
	ipBox := container.NewGridWrap(fyne.NewSize(160, b.ip.MinSize().Height), b.ip)
	passBox := container.NewGridWrap(fyne.NewSize(160, b.pass.MinSize().Height), b.pass)
	b.obj = container.NewHBox(
		widget.NewLabel("IP address:"), ipBox,
		widget.NewLabel("Port:"), portBox,
		widget.NewLabel("User level:"), b.user,
		widget.NewLabel("Password:"), passBox,
		b.insecure, layout.NewSpacer(), b.connect, b.disconnect)
	return b
}

// barParams reads the connection bar into ConnectParams.
func (s *State) barParams() (ConnectParams, error) {
	b := s.bar
	p := ConnectParams{
		Address:  strings.TrimSpace(b.ip.Text),
		Insecure: b.insecure.Checked,
		Password: b.pass.Text,
		Username: core.UserLevelAdmin,
	}
	if i := b.user.SelectedIndex(); i >= 0 && i < len(core.DeviceUserLevels) {
		p.Username = core.DeviceUserLevels[i].ID
	}
	port, err := strconv.Atoi(strings.TrimSpace(b.port.Text))
	if err != nil {
		port = 0
	}
	p.Port = port
	if b.target.Address == p.Address {
		p.Identity, p.SerialNumber, p.HostName = b.target.Identity, b.target.SerialNumber, b.target.HostName
		p.NumberOfSockets, p.FirmwareVersion = b.target.NumberOfSockets, b.target.FirmwareVersion
	}
	return p, nil
}

// SetConnectionTarget fills the connection bar from a discovered device
// (selecting a row in the device list). UI goroutine.
func (s *State) SetConnectionTarget(d core.Device) {
	s.bar.target = d
	s.bar.ip.SetText(d.Address)
	if d.Port > 0 {
		s.bar.port.SetText(strconv.Itoa(d.Port))
	}
}

// setConnectedUI flips the shell between connected and disconnected.
func (s *State) setConnectedUI() {
	connected := s.Connected()
	if connected {
		s.bar.connect.Disable()
		s.bar.disconnect.Enable()
		sess := s.Session()
		ld := sess.LoginData()
		s.connInfo.SetText(fmt.Sprintf("Connected: %s (%s) as %s", sess.Identification(), sess.Address(), ld.Username))
	} else {
		s.bar.connect.Enable()
		s.bar.disconnect.Disable()
		s.connInfo.SetText("Not connected")
	}
	s.refreshPanelContent()
	s.rebuildMenu()
	s.updateSaveButtons()
}
