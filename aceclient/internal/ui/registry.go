package ui

import (
	"fmt"
	"sort"
	"sync"

	"fyne.io/fyne/v2"
)

// Page feature IDs: the PageID each installer panel is constructed with in
// MainWindow's constructor (m_allPanels / m_allSCNPanels). They are the keys
// of the feature rights in the installer configuration (ICUGroup.GetRights).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:229-249
const (
	FeatureSocket        = "PAGE_SOCKET"        // PanelSockets
	FeatureInformation   = "PAGE_INFORMATION"   // PanelInformation
	FeaturePower         = "PAGE_POWER"         // PanelPower
	FeatureNetwork       = "PAGE_NETWORK"       // PanelLoadbalancing
	FeatureWhitelist     = "PAGE_WHITELIST"     // PanelAuthorization
	FeatureTransactions  = "PAGE_TRANSACTIONS"  // PanelTransactions
	FeatureBackoffice    = "PAGE_BACKOFFICE"    // PanelConnectivity
	FeatureUI            = "PAGE_UI"            // PanelInterface
	FeatureAlerts        = "PAGE_ALERTS"        // PanelAlerts
	FeatureLog           = "PAGE_LOG"           // PanelLog
	FeatureStates        = "PAGE_STATES"        // PanelMonitoring
	FeatureAllProperties = "PAGE_ALLPROPERTIES" // PanelAllProperties
	FeatureSCNOverview   = "PAGE_SCN_OVERVIEW"  // PanelSCNOverview
	FeatureSCNSettings   = "PAGE_SCN_SETTINGS"  // PanelSCNSettings
)

// Tab orders. The device pages follow MainWindow.FillPanelList (the order of
// m_allPanels, then m_allSCNPanels) in steps of 100 so later agents can slot
// pages in between. Discovery (the C# device tree) comes first; the firmware
// upload page (DlgUpload, a dialog in the C#) comes last.
const (
	OrderDiscover      = 0
	OrderSockets       = 100
	OrderInformation   = 200
	OrderPower         = 300
	OrderNetwork       = 400
	OrderWhitelist     = 500
	OrderTransactions  = 600
	OrderBackoffice    = 700
	OrderUI            = 800
	OrderAlerts        = 900
	OrderLog           = 1000
	OrderStates        = 1100
	OrderAllProperties = 1200
	OrderSCNOverview   = 1300
	OrderSCNSettings   = 1400
	OrderFirmware      = 1500
)

// PanelMode says in which tab set a panel appears. MainWindow.FillPanelList
// shows either the device panels or, when an SCN node is selected in the
// device tree, the SCN panels.
type PanelMode int

const (
	PanelModeDevice PanelMode = iota // m_allPanels (default)
	PanelModeSCN                     // m_allSCNPanels
	PanelModeAlways                  // shell pages that are always shown
)

// PanelSpec describes one tab. Panel files register theirs from init() via
// RegisterPanel, so app.go never needs editing to add a page.
type PanelSpec struct {
	ID    string // unique key (e.g. "information")
	Title string // tab text (PanelBase.Title)
	Order int    // sort key, see the Order* constants
	// FeatureID is the C# PageID used for feature-rights gating; "" means
	// the page is not subject to rights.
	FeatureID string
	// RequiresConnection shows the PanelNotLoggedIn placeholder instead of
	// the content while no device is logged in.
	RequiresConnection bool
	Mode               PanelMode
	// Build creates the page once, when the window is built. Pages subscribe
	// to State hooks (OnConnect, OnDisconnect, OnPanelShown, …) from here.
	Build func(*State) fyne.CanvasObject
}

// MenuItemSpec describes one main-menu entry (MainWindow's _File, _Device
// and _Help menus). Command, preset and about agents register theirs from
// init() via RegisterMenuItem.
type MenuItemSpec struct {
	Menu  string // "File", "Device", "Help" (others are appended after these)
	Label string
	Order int
	// RequiresConnection disables the entry while no device is logged in
	// (m_mnuDevice.Sensitive = m_currentDevice != null && IsLoggedIn).
	RequiresConnection bool
	// FeatureID optionally hides the entry when the rights hook returns None.
	FeatureID string
	Action    func(*State)
}

// menuOrder is the C# top-level menu order.
var menuOrder = []string{"File", "Device", "Help"}

// SaveCheckFunc is one panel's OnSaveChanges hook, evaluated (in
// registration order) before the shell's Save stores anything.
type SaveCheckFunc func(*State) (errorText, question string)

type registry struct {
	mu     sync.Mutex
	panels map[string]PanelSpec
	menus  []MenuItemSpec
	checks []SaveCheckFunc
}

func newRegistry() *registry { return &registry{panels: map[string]PanelSpec{}} }

var defaultRegistry = newRegistry()

func (r *registry) registerPanel(p PanelSpec) {
	if p.ID == "" || p.Build == nil {
		panic("ui: RegisterPanel needs an ID and a Build func")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.panels[p.ID]; dup {
		panic(fmt.Sprintf("ui: panel %q registered twice", p.ID))
	}
	r.panels[p.ID] = p
}

func (r *registry) panelList() []PanelSpec {
	r.mu.Lock()
	out := make([]PanelSpec, 0, len(r.panels))
	for _, p := range r.panels {
		out = append(out, p)
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (r *registry) registerMenuItem(m MenuItemSpec) {
	if m.Menu == "" || m.Label == "" || m.Action == nil {
		panic("ui: RegisterMenuItem needs Menu, Label and Action")
	}
	r.mu.Lock()
	r.menus = append(r.menus, m)
	r.mu.Unlock()
}

// menuList returns the menu names in C# order (unknown menus alphabetically
// after File/Device/Help) and their items sorted by Order then Label.
func (r *registry) menuList() ([]string, map[string][]MenuItemSpec) {
	r.mu.Lock()
	items := append([]MenuItemSpec(nil), r.menus...)
	r.mu.Unlock()
	by := map[string][]MenuItemSpec{}
	for _, m := range items {
		by[m.Menu] = append(by[m.Menu], m)
	}
	var names []string
	known := map[string]bool{}
	for _, n := range menuOrder {
		known[n] = true
		if len(by[n]) > 0 {
			names = append(names, n)
		}
	}
	var extra []string
	for n := range by {
		if !known[n] {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	names = append(names, extra...)
	for n := range by {
		l := by[n]
		sort.SliceStable(l, func(i, j int) bool {
			if l[i].Order != l[j].Order {
				return l[i].Order < l[j].Order
			}
			return l[i].Label < l[j].Label
		})
	}
	return names, by
}

func (r *registry) registerSaveCheck(f SaveCheckFunc) {
	r.mu.Lock()
	r.checks = append(r.checks, f)
	r.mu.Unlock()
}

func (r *registry) saveChecks() []SaveCheckFunc {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]SaveCheckFunc(nil), r.checks...)
}

// RegisterPanel adds a tab. Call it from a panel file's init(). It panics on
// a duplicate ID or a missing Build func.
func RegisterPanel(p PanelSpec) { defaultRegistry.registerPanel(p) }

// Panels returns every registered panel sorted by Order (then ID).
func Panels() []PanelSpec { return defaultRegistry.panelList() }

// RegisterMenuItem adds a main-menu entry. Call it from init().
func RegisterMenuItem(m MenuItemSpec) { defaultRegistry.registerMenuItem(m) }

// MenuItems returns the registered menus in display order and their items.
func MenuItems() ([]string, map[string][]MenuItemSpec) { return defaultRegistry.menuList() }

// RegisterSaveCheck adds a panel OnSaveChanges hook (see SaveCheckFunc).
func RegisterSaveCheck(f SaveCheckFunc) { defaultRegistry.registerSaveCheck(f) }

// Rights mirrors ICUSettings.ICURights.
//
// Source: firmware/decompiled/ACESettings/ICUSettings/ICURights.cs
type Rights int

const (
	RightsNone     Rights = iota // page hidden (PanelBase.IsVisible == false)
	RightsReadOnly               // page shown, properties read-only (CheckAccessRights)
	RightsFull
)

// RightsFunc resolves the rights of the logged-on installer user for a page
// feature ID (ICUUser.GetRights → ICUGroup.GetRights, which defaults to Full
// for features the group does not list, and ReadOnly when the user has no
// group). The config/rights agent installs it with State.SetRights; until
// then every page has Full rights.
//
// Source: firmware/decompiled/ACESettings/ICUSettings/ICUUser.cs:319-326, ICUGroup.cs:260-263; firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelBase.cs:219-246
type RightsFunc func(featureID string) Rights
