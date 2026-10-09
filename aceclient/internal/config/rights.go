package config

// Page IDs of the installer panels, in MainWindow's m_allPanels /
// m_allSCNPanels order (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs).
// PAGE_ALERTS and the SCN pages have no config feature, so
// Group.GetRights returns RightsFull for them.
const (
	PageSocket        = "PAGE_SOCKET"        // PanelSockets
	PageInformation   = "PAGE_INFORMATION"   // PanelInformation
	PagePower         = "PAGE_POWER"         // PanelPower
	PageNetwork       = "PAGE_NETWORK"       // PanelLoadbalancing
	PageWhitelist     = "PAGE_WHITELIST"     // PanelAuthorization
	PageTransactions  = "PAGE_TRANSACTIONS"  // PanelTransactions
	PageBackoffice    = "PAGE_BACKOFFICE"    // PanelConnectivity
	PageUI            = "PAGE_UI"            // PanelInterface
	PageAlerts        = "PAGE_ALERTS"        // PanelAlerts
	PageLog           = "PAGE_LOG"           // PanelLog
	PageStates        = "PAGE_STATES"        // PanelMonitoring
	PageAllProperties = "PAGE_ALLPROPERTIES" // PanelAllProperties
	PageSCNOverview   = "PAGE_SCN_OVERVIEW"  // PanelSCNOverview
	PageSCNSettings   = "PAGE_SCN_SETTINGS"  // PanelSCNSettings
)

// Non-page feature IDs the installer checks.
const (
	FeatureCreateFWU = "FEATURE_CREATEFWU" // DlgUploadResources "Create FWU" button (needs RightsFull)
	FeatureColors    = "FEATURE_COLORS"    // PanelInterface "LED colors" color holder
)

// MainPageIDs is m_allPanels' page order (MainWindow constructor).
var MainPageIDs = []string{
	PageSocket, PageInformation, PagePower, PageNetwork, PageWhitelist, PageTransactions,
	PageBackoffice, PageUI, PageAlerts, PageLog, PageStates, PageAllProperties,
}

// SCNPageIDs is m_allSCNPanels' page order (MainWindow constructor).
var SCNPageIDs = []string{PageSCNOverview, PageSCNSettings}

// IsPageVisible ports PanelBase.IsVisible(ICUUser): false without a user or
// when the user's rights on the page are RightsNone.
func IsPageVisible(u *User, pageID string) bool {
	if u == nil {
		return false
	}
	return u.GetRights(pageID) != RightsNone
}

// VisiblePages ports MainWindow.FillPanelList(fSCN): the page IDs (in panel
// order) of the main panels, or of the SCN panels when scn is set, that
// IsPageVisible allows.
func VisiblePages(u *User, scn bool) []string {
	ids := MainPageIDs
	if scn {
		ids = SCNPageIDs
	}
	var out []string
	for _, id := range ids {
		if IsPageVisible(u, id) {
			out = append(out, id)
		}
	}
	return out
}

// PropertyAccess is the outcome of UIPropertyBase.CheckAccessRights for one
// property widget.
type PropertyAccess struct {
	ReadOnly bool // m_fForceReadOnly
	Hidden   bool // m_fHidden
}

// CheckAccessRights ports UIPropertyBase.CheckAccessRights(newGroup,
// parentRights): read-only when the property is always read-only
// (ForceReadonly), the page is ReadOnly or the group has ReadOnly on the
// property's FeatureRightID; hidden when the page is None or the group has
// None on it. A nil group (never passed by the C#) applies no per-feature
// restriction.
func CheckAccessRights(g *Group, parentRights Rights, featureRightID string, alwaysReadOnly bool) PropertyAccess {
	fr := RightsFull
	if g != nil {
		fr = g.GetRights(featureRightID)
	}
	return PropertyAccess{
		ReadOnly: alwaysReadOnly || parentRights == RightsReadOnly || fr == RightsReadOnly,
		Hidden:   parentRights == RightsNone || fr == RightsNone,
	}
}

// PanelPropertyAccess ports PanelBase.SetUser for one property: the page's
// rights for the user are the parent rights for CheckAccessRights. ok is false
// when SetUser returns before checking (no user, or a user without group), in
// which case the widget keeps its previous state.
func PanelPropertyAccess(u *User, pageID, featureRightID string, alwaysReadOnly bool) (access PropertyAccess, ok bool) {
	if u == nil || u.Group == nil {
		return PropertyAccess{}, false
	}
	return CheckAccessRights(u.Group, u.GetRights(pageID), featureRightID, alwaysReadOnly), true
}

// CanCreateFWU ports DlgUploadResources: the "Create FWU" button is visible
// only with RightsFull on FEATURE_CREATEFWU.
func CanCreateFWU(u *User) bool {
	return u != nil && u.GetRights(FeatureCreateFWU) == RightsFull
}

// CanUseLogCommands ports PanelLog.OnUpdateControls: the 8th log filter button
// and the command button are shown only with RightsFull on PAGE_LOG.
func CanUseLogCommands(u *User) bool {
	return u != nil && u.GetRights(PageLog) == RightsFull
}

// MainWindowTitle ports the window titles set by MainWindow.ShowLogon:
//
//	no user:        "<AppName> <version> - Settings: <config version>"
//	user:           "... - <Fullname> (<Group.Name>)"
//	user, no FTP:   "... - <Fullname> (<Group.Name>) *"
//
// fileVersion is formatted with FormatAppVersion.
func MainWindowTitle(fileVersion, configVersion string, u *User, ftpAvailable bool) string {
	t := AppName + " " + FormatAppVersion(fileVersion) + " - Settings: " + configVersion
	if u == nil {
		return t
	}
	group := ""
	if u.Group != nil {
		group = u.Group.Name
	}
	t += " - " + u.Fullname + " (" + group + ")"
	if !ftpAvailable {
		t += " *"
	}
	return t
}

// LoggedOutTitle ports MainWindow.OnLogout's title "<AppName> <config version>".
func LoggedOutTitle(configVersion string) string {
	return AppName + " " + configVersion
}
