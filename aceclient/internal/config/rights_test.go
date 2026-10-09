package config

import (
	"slices"
	"testing"
)

func TestFeatureIDs(t *testing.T) {
	tests := []struct{ name, got, want string }{
		{"PropertyFeatureID(8290)", PropertyFeatureID(8290), "ID_2062"},
		{"PropertyFeatureID(12824)", PropertyFeatureID(12824), "ID_3218"},
		{"PropertyFeatureID(1)", PropertyFeatureID(1), "ID_0001"},
		{"BackOfficeFeatureID(Fleet One)", BackOfficeFeatureID("Fleet One"), "BO_FLEET_ONE"},
		{"BackOfficeFeatureID( my bo )", BackOfficeFeatureID(" my bo "), "BO__MY_BO_"}, // Replace(' ', '_') happens before Trim
		{"NewPageFeature", NewPageFeature("LOG", " x ", RightsFull).ID, "PAGE_LOG"},
		{"NewNormalFeature", NewNormalFeature("CREATEFWU", "", RightsFull).ID, "FEATURE_CREATEFWU"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
	if f := NewBackOfficeFeature("A B", "BackOffice 'A B'", RightsReadOnly); f.Type != FeatureTypeBackoffice || f.Default != RightsReadOnly {
		t.Error("NewBackOfficeFeature")
	}
}

func rightsUser() *User {
	feats := []*Feature{
		NewPageFeature("LOG", "Page Log", RightsFull),
		NewPageFeature("PRODUCTION", "Page Production", RightsNone),
		NewPageFeature("POWER", "Page Power", RightsReadOnly),
		NewIDFeature(8290, "Max Station Current", RightsFull),
		NewIDFeature(8292, "Load Balancing Mode", RightsReadOnly),
	}
	return &User{User: "u", Group: NewGroup(feats, "Service", "", "", "")}
}

func TestGetRights(t *testing.T) {
	u := rightsUser()
	tests := []struct {
		id   string
		want Rights
	}{
		{PageLog, RightsFull},
		{"PAGE_PRODUCTION", RightsNone},
		{PagePower, RightsReadOnly},
		{PageAlerts, RightsFull}, // not a config feature
		{PageSCNOverview, RightsFull},
	}
	for _, tc := range tests {
		if got := u.GetRights(tc.id); got != tc.want {
			t.Errorf("GetRights(%s) = %v", tc.id, got)
		}
	}
	if (&User{}).GetRights(PageLog) != RightsReadOnly {
		t.Error("user without group must be ReadOnly")
	}
	g := NewGroup([]*Feature{NewPageFeature("PRODUCTION", "", RightsNone)}, "ADMIN", "", "", "")
	if g.GetRights("PAGE_PRODUCTION") != RightsFull {
		t.Error("NewGroup named admin (any case) starts Full")
	}
}

func TestVisiblePages(t *testing.T) {
	u := rightsUser()
	if got := VisiblePages(u, false); !slices.Equal(got, MainPageIDs) {
		t.Errorf("all main pages should be visible: %v", got)
	}
	u.Group.Features[0].Rights = RightsNone // PAGE_LOG
	got := VisiblePages(u, false)
	if slices.Contains(got, PageLog) || len(got) != len(MainPageIDs)-1 {
		t.Errorf("PAGE_LOG should be hidden: %v", got)
	}
	if !slices.Equal(VisiblePages(u, true), SCNPageIDs) {
		t.Error("SCN pages")
	}
	if VisiblePages(nil, false) != nil || IsPageVisible(nil, PageLog) {
		t.Error("no user, no pages")
	}
	if !IsPageVisible(&User{}, PageLog) {
		t.Error("group-less user gets ReadOnly, i.e. visible")
	}
}

func TestCheckAccessRights(t *testing.T) {
	g := rightsUser().Group
	tests := []struct {
		parent         Rights
		feature        string
		alwaysReadOnly bool
		want           PropertyAccess
	}{
		{RightsFull, PropertyFeatureID(8290), false, PropertyAccess{}},
		{RightsFull, PropertyFeatureID(8290), true, PropertyAccess{ReadOnly: true}},
		{RightsReadOnly, PropertyFeatureID(8290), false, PropertyAccess{ReadOnly: true}},
		{RightsNone, PropertyFeatureID(8290), false, PropertyAccess{Hidden: true}},
		{RightsFull, PropertyFeatureID(8292), false, PropertyAccess{ReadOnly: true}},
		{RightsFull, "PAGE_PRODUCTION", false, PropertyAccess{Hidden: true}},
		{RightsFull, "", false, PropertyAccess{}}, // no FeatureRightID -> Full
		{RightsNone, PropertyFeatureID(8292), true, PropertyAccess{ReadOnly: true, Hidden: true}},
	}
	for _, tc := range tests {
		if got := CheckAccessRights(g, tc.parent, tc.feature, tc.alwaysReadOnly); got != tc.want {
			t.Errorf("%v/%s/%v = %+v", tc.parent, tc.feature, tc.alwaysReadOnly, got)
		}
	}
	u := rightsUser()
	if a, ok := PanelPropertyAccess(u, PagePower, PropertyFeatureID(8290), false); !ok || !a.ReadOnly {
		t.Error("ReadOnly page makes its properties read-only")
	}
	if _, ok := PanelPropertyAccess(&User{}, PagePower, "", false); ok {
		t.Error("SetUser skips users without group")
	}
	if _, ok := PanelPropertyAccess(nil, PagePower, "", false); ok {
		t.Error("SetUser skips nil user")
	}
}

func TestFeatureGates(t *testing.T) {
	u := rightsUser()
	if !CanCreateFWU(u) || !CanUseLogCommands(u) {
		t.Error("unknown FEATURE_CREATEFWU => Full; PAGE_LOG Full")
	}
	if CanCreateFWU(nil) || CanUseLogCommands(&User{}) {
		t.Error("nil / group-less user")
	}
}

func TestTitles(t *testing.T) {
	u := &User{Fullname: "Jane", Group: &Group{Name: "Service"}}
	tests := []struct{ name, got, want string }{
		{"no user", MainWindowTitle("4.1.2.345", "2.3.0", nil, true), "ACE Service Installer 4.1.2-345 - Settings: 2.3.0"},
		{"user", MainWindowTitle("4.1.2.345", "2.3.0", u, true), "ACE Service Installer 4.1.2-345 - Settings: 2.3.0 - Jane (Service)"},
		{"unsaved", MainWindowTitle("4.1.2.345", "2.3.0", u, false), "ACE Service Installer 4.1.2-345 - Settings: 2.3.0 - Jane (Service) *"},
		{"no group", MainWindowTitle("4.1", "2.3.0", &User{Fullname: "J"}, true), "ACE Service Installer 4.1 - Settings: 2.3.0 - J ()"},
		{"logged out", LoggedOutTitle("2.3.0"), "ACE Service Installer 2.3.0"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, tc.got, tc.want)
		}
	}
}
