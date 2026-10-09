package i18n

import "testing"

func TestTooltipID(t *testing.T) {
	tests := []struct {
		id   uint16
		sub  uint8
		want string
	}{
		{0x2062, 0x00, "2062_00"},
		{0x2180, 0x0A, "2180_0A"},
		{0x0001, 0x01, "0001_01"},
		{0xFFFF, 0xFF, "FFFF_FF"},
		{0, 0, "0000_00"},
	}
	for _, tc := range tests {
		if got := TooltipID(tc.id, tc.sub); got != tc.want {
			t.Errorf("TooltipID(%#x, %#x) = %q; want %q", tc.id, tc.sub, got, tc.want)
		}
		if got := MakeTooltipID(tc.id, tc.sub, 0); got != tc.want {
			t.Errorf("MakeTooltipID(%#x, %#x, 0) = %q; want %q", tc.id, tc.sub, got, tc.want)
		}
	}
}

// Every packed tooltipID literal in the installer: all 48 reach the
// UIProperty* constructors through the PanelBase.Add* helpers (PanelBase.cs),
// whose trailing `uint tooltipID` is forwarded unchanged; no control is built
// with one directly. (The other 7-digit literals in the panels and dialogs go
// to UpdateProperties/GetProperty — packed property IDs, not tooltip keys.)
// csvLabel is the label column of the shipped CSV, "" when it has no tip,
// which hides the information button. 47 of 48 call sites (43 of the 44
// distinct values) have a tip; only PanelPower 2123521 -> "2067_01" has none.
func TestMakeTooltipIDFromPanels(t *testing.T) {
	tests := []struct {
		src      string // file:line under ACEServiceInstaller/ICUServiceInstaller
		helper   string // PanelBase helper
		csLabel  string // label text passed by the panel
		packed   uint32
		want     string
		csvLabel string
	}{
		{"PanelAlerts.cs:74", "AddCustomText", "Setpoint X,Y,Z values", 2232320, "2210_00", "Setpoint X.Y.Z values"},
		{"PanelAlerts.cs:100", "AddCustomSelect", "Tamper sensor", 2248704, "2250_00", "Tamper state"},
		{"PanelAuthorization.cs:343", "AddPlainSelect", "Offline action", 2172672, "2127_00", "Offline action"},
		{"PanelConnectivity.cs:444", "AddPlainSelect", "Backoffice preset", 2127360, "2076_00", "Backoffice preset"},
		{"PanelConnectivity.cs:448", "AddPlainSelect", "Connect method", 2127616, "2077_00", "Connect method"},
		{"PanelConnectivity.cs:462", "AddCustomNumber", "Network profile connection attempts", 2159616, "20F4_00", "Network profile connection attempts"},
		{"PanelConnectivity.cs:529", "AddPlainPassword", "Password", 3312384, "328B_00", "WiFi PSK (password)"},
		{"PanelConnectivity.cs:566", "AddPlainPassword", "Back office authorization Key", 2564865, "2723_01", "Authorization key"},
		{"PanelConnectivity.cs:596", "AddPlainPassword", "Proxy password", 2168321, "2116_01", "Proxy password"},
		{"PanelConnectivity.cs:731", "AddCustomSelect", "Priority", 2158606, "20F0_0E", "Priority"},
		{"PanelConnectivity.cs:733", "AddCustomSelect", "Connect method", 2158598, "20F0_06", "Connect method"},
		{"PanelConnectivity.cs:735", "AddCustomSelect", "Protocol", 2158593, "20F0_01", "Protocol"},
		{"PanelConnectivity.cs:737", "AddText", "CSMS URL", 2158595, "20F0_03", "Network profile URL"},
		{"PanelConnectivity.cs:738", "AddCustomSelect", "Security Profile", 2158597, "20F0_05", "Security profile"},
		{"PanelConnectivity.cs:739", "AddWriteOnlyText", "APN name", 2162688, "2100_00", "APN name"},
		{"PanelConnectivity.cs:740", "AddWriteOnlyText", "APN user", 2162944, "2101_00", "APN user"},
		{"PanelConnectivity.cs:741", "AddWriteOnlyText", "APN password", 2163200, "2102_00", "APN password"},
		{"PanelConnectivity.cs:742", "AddWriteOnlyText", "SIM pin", 2163456, "2103_00", "Mobile SIM Pin"},
		{"PanelConnectivity.cs:743", "AddCustomNumber", "Websocket timeout (s)", 2158596, "20F0_04", "Websocket timeout (s)"},
		{"PanelInformation.cs:316", "AddCustomText", "Hardware version SCB", 2116865, "204D_01", "Hardware version controller board"},
		{"PanelInformation.cs:321", "AddCustomText", "Hardware version controller board", 2116865, "204D_01", "Hardware version controller board"},
		{"PanelInformation.cs:322", "AddCustomText", "Hardware version power board", 2116867, "204D_03", "Hardware version power board"},
		{"PanelInformation.cs:329", "AddCustomText", "WiFi supported", 3313408, "328F_00", "WiFi supported"},
		{"PanelInformation.cs:331", "AddCustomText", "Tamper detection supported", 2248961, "2251_01", "Tamper detection available"},
		{"PanelInformation.cs:346", "AddReadOnlyText", "NFC-RFID reader 1 hardw. version", 3244032, "3180_00", "NFC-RFID reader 1 hardware version"},
		{"PanelInformation.cs:347", "AddReadOnlyText", "NFC-RFID reader 1 softw. version", 3244033, "3180_01", "NFC-RFID reader 1 software version"},
		{"PanelInformation.cs:350", "AddReadOnlyText", "NFC-RFID reader 2 hardw. version", 3244288, "3181_00", "NFC-RFID reader 2 hardware version"},
		{"PanelInformation.cs:351", "AddReadOnlyText", "NFC-RFID reader 2 softw. version", 3244289, "3181_01", "NFC-RFID reader 2 software version"},
		{"PanelInformation.cs:356", "AddReadOnlyText", "NFC-RFID reader hardware version", 3244032, "3180_00", "NFC-RFID reader 1 hardware version"},
		{"PanelInformation.cs:357", "AddReadOnlyText", "NFC-RFID reader software version", 3244033, "3180_01", "NFC-RFID reader 1 software version"},
		{"PanelInformation.cs:461", "AddCheckBox", "Enable Secure Service Access", 2208256, "21B2_00", "Enable Secure Service Access"},
		{"PanelInformation.cs:499", "AddCheckBox", "Allow alpha releases", 2494464, "2610_00", "Allow alpha releases"},
		{"PanelLoadbalancing.cs:611", "AddCheckBox", "Static Load Balancing", 2122753, "2064_01", "Static Load Balancing"},
		{"PanelLoadbalancing.cs:615", "AddCheckBox", "Static Load Balancing", 2122753, "2064_01", "Static Load Balancing"},
		{"PanelLoadbalancing.cs:650", "AddCheckBox", "Active Load Balancing", 2122754, "2064_02", "Active Load Balancing"},
		{"PanelLoadbalancing.cs:752", "AddPlainSelect", "Mode", 2437123, "2530_03", "Mode (EMS)"},
		{"PanelMonitoring.cs:256", "AddCustomText", "Device state", 2425241, "2501_99", "Device state"},
		{"PanelMonitoring.cs:260", "AddManualSelect", "Status", 2121475, "205F_03", "Status"},
		{"PanelMonitoring.cs:302", "AddCustomText", "Device state", 2425497, "2502_99", "Device state"},
		{"PanelMonitoring.cs:306", "AddManualSelect", "Status", 2121477, "205F_05", "Status"},
		{"PanelMonitoring.cs:491", "AddCustomText", "Tilt sensor (X,Y,Z)", 2230016, "2207_00", "Tilt sensor (X.Y.Z)"},
		{"PanelPower.cs:226", "AddCustomNumber", "Installation maximum current (A)", 2123521, "2067_01", ""},
		{"PanelPower.cs:258", "AddPlainBool", "Enable Max. Imbalance current", 2192385, "2174_01", "Enable Max. Imbalance current"},
		{"PanelPower.cs:259", "AddCustomNumber", "Maximum Imbalance current (A)", 2192386, "2174_02", "Maximum Imbalance Current (A)"},
		{"PanelSCNSettings.cs:167", "AddCustomNumber", "Total Max. current (A)", 2195461, "2180_05", "Total current (A)"},
		{"PanelSCNSettings.cs:168", "AddCustomNumber", "Alternating period (s)", 2195460, "2180_04", "Alternating period (s)"},
		{"PanelSCNSettings.cs:172", "AddCustomNumber", "Socket Safe current (A)", 2195462, "2180_06", "Safe current (A)"},
		{"PanelSCNSettings.cs:173", "AddCustomNumber", "Total Safe current (A)", 2195466, "2180_0A", "Total safe current (A)"},
	}
	distinct := map[uint32]bool{}
	withTip := 0
	for _, tc := range tests {
		distinct[tc.packed] = true
		if tc.csvLabel != "" {
			withTip++
		}
		t.Run(tc.src+" "+tc.helper, func(t *testing.T) {
			// The override wins over the control's own (id, sub).
			got := MakeTooltipID(0x1234, 0x56, tc.packed)
			if got != tc.want {
				t.Fatalf("MakeTooltipID(.., %d) = %q; want %q", tc.packed, got, tc.want)
			}
			hasTip := tc.csvLabel != ""
			if HasTooltip(got) != hasTip {
				t.Errorf("HasTooltip(%q) = %v; want %v", got, !hasTip, hasTip)
			}
			if lbl := LabelText(got); lbl != tc.csvLabel {
				t.Errorf("LabelText(%q) = %q; want %q (panel label %q)", got, lbl, tc.csvLabel, tc.csLabel)
			}
		})
	}
	if len(tests) != 48 || len(distinct) != 44 || withTip != 47 {
		t.Errorf("call sites = %d, distinct = %d, with tip = %d; want 48, 44, 47", len(tests), len(distinct), withTip)
	}
	// Packed values wider than 24 bits keep all high digits, as {id >> 8:X4}.
	if got := MakeTooltipID(0, 0, 0x12345678); got != "123456_78" {
		t.Errorf("wide packed = %q", got)
	}
}

func TestMakeTooltip(t *testing.T) {
	if got := MakeTooltip("", "2062_00"); got != "" {
		t.Errorf("empty text: %q", got)
	}
	if got := MakeTooltip("anything", "2062_00"); got != Tooltip("2062_00") || got == "" {
		t.Errorf("non-empty text must return the tooltip, got %q", got)
	}
	if got := MakeTooltip("anything", "0000_00"); got != "" {
		t.Errorf("unknown id: %q", got)
	}
}

func TestPropertyLabelText(t *testing.T) {
	str := func(s string) *string { return &s }
	tests := []struct {
		name        string
		id          uint16
		sub         uint8
		eds         *EDSLabel
		label       string
		want        string
		wantUnknown bool
	}{
		{"eds title only", 0x2062, 0, &EDSLabel{Title: "Max current"}, "ignored", "Max current", false},
		{"eds title + units", 0x2062, 0, &EDSLabel{Title: "Max current", Units: str("A")}, "", "Max current (A)", false},
		{"eds empty units present", 0x2062, 0, &EDSLabel{Title: "T", Units: str("")}, "", "T ()", false},
		{"label-only control", 0, 0, nil, "Device state", "Device state", false},
		{"eds wins even for 0/0", 0, 0, &EDSLabel{Title: "E"}, "L", "E", false},
		{"unknown, sub 0", 0x2062, 0, nil, "L", "2062", true},
		{"unknown, sub decimal", 0x2180, 10, nil, "", "2180_10", true},
		{"unknown, small id padded", 0x0012, 255, nil, "", "0012_255", true},
		{"unknown, id 0 sub 1", 0, 1, nil, "L", "0000_1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, unknown := PropertyLabelText(tc.id, tc.sub, tc.eds, tc.label)
			if got != tc.want || unknown != tc.wantUnknown {
				t.Fatalf("= %q, %v; want %q, %v", got, unknown, tc.want, tc.wantUnknown)
			}
		})
	}
}

func TestApplyOverwriteLabel(t *testing.T) {
	tests := []struct {
		label, overwrite string
		unknown          bool
		want             string
		wantUnknown      bool
	}{
		{"2062", "", true, "2062", true},
		{"2062", " \t ", true, "2062", true}, // IsNullOrWhiteSpace
		{"2062", "Custom", true, "Custom", false},
		{"Title", " Custom ", false, " Custom ", false}, // not trimmed
	}
	for _, tc := range tests {
		got, unknown := ApplyOverwriteLabel(tc.label, tc.unknown, tc.overwrite)
		if got != tc.want || unknown != tc.wantUnknown {
			t.Errorf("ApplyOverwriteLabel(%q, %v, %q) = %q, %v; want %q, %v",
				tc.label, tc.unknown, tc.overwrite, got, unknown, tc.want, tc.wantUnknown)
		}
	}
}

func TestUIPropertyBaseStrings(t *testing.T) {
	if ChangedTooltip != "This property is changed" {
		t.Errorf("ChangedTooltip = %q", ChangedTooltip)
	}
	got := InvalidValueMessage("Max current", "abc", "Input string was not in a correct format.")
	want := "Configuration item: Max current has an invalid value: abc. Exception: Input string was not in a correct format. "
	if got != want {
		t.Errorf("InvalidValueMessage = %q; want %q", got, want)
	}
}
