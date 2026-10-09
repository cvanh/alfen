package backoffice

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"alfen/aceclient/internal/api"
)

// Messages shown by PanelConnectivity.OnSaveChanges.
const (
	// MsgUniquePriorities is the MessageDialog.ShowError text for duplicate
	// network-profile priorities.
	MsgUniquePriorities = "Network profiles must have unique priorities."
	// MsgWiredOnly is the m_lblConnectionMethod warning text.
	MsgWiredOnly = "Back office preset has only GPRS connection. But device supports only wired connections."
)

// ErrUniquePriorities is returned by PlanSave for MsgUniquePriorities.
var ErrUniquePriorities = errors.New(MsgUniquePriorities)

// RebootQuestion ports the MessageDialog.AskQuestion text asked after a preset
// upload or a network-profile priority change (Yes -> DlgReboot with
// fAutoStartReboot and hardReboot).
func RebootQuestion(identity string) string {
	return "You have changed the Backoffice settings. The new settings will only be active after a restart. Do you want to restart " + identity + " now?"
}

// clearNetworkProfile ports PanelConnectivity.ClearNetworkProfileCategory.
func clearNetworkProfile(w *writeSet, index int, hasExtendedFieldLengths bool) {
	if index < 0 || index >= NetworkProfileCount {
		return
	}
	id := PropNetworkProfile + uint16(index)
	user, pin := NPSubAPNUsernameDeprecated, NPSubSimPinDeprecated
	if hasExtendedFieldLengths {
		user, pin = NPSubAPNUsername, NPSubSimPin
	}
	w.set(id, NPSubPriority, "0")        // (byte)0
	w.set(id, NPSubInterface, "0")       // (byte)0
	w.set(id, NPSubProtocolVersion, "1") // (byte)1
	w.set(id, NPSubURL, "")
	w.set(id, NPSubSecurityProfile, "0") // 0
	w.set(id, NPSubAPNName, "")
	w.set(id, user, "")
	w.set(id, NPSubAPNPassword, "")
	w.set(id, pin, "")
	w.set(id, NPSubMessageTimeout, "10") // 10
}

// clearAll ports PanelConnectivity.ClearAllBackOfficeSettings.
func clearAll(w *writeSet, hasExtendedFieldLengths bool) {
	w.set(PropBackOfficeURLWired, 1, "")
	w.set(PropBackOfficeURLWired, 2, "")
	w.set(PropBackOfficeURL, 1, "")
	w.set(PropBackOfficeURL, 2, "")
	w.set(PropAPNName, 0, "")
	w.set(PropAPNUser, 0, "")
	w.set(PropAPNPassword, 0, "")
	for i := 0; i < NetworkProfileCount; i++ {
		clearNetworkProfile(w, i, hasExtendedFieldLengths)
	}
}

// ClearAllBackOfficeSettings ports PanelConnectivity.ClearAllBackOfficeSettings:
// the propList that blanks the wired/mobile backoffice URLs, the APN
// credentials and all four network profiles. Only properties the device
// exposes are included; values are typed per the device's DataType.
func ClearAllBackOfficeSettings(device Lookup, hasExtendedFieldLengths bool) []api.Property {
	w := newWriteSet(device)
	clearAll(w, hasExtendedFieldLengths)
	return w.all()
}

// SaveInput is the backoffice-related panel/device state
// PanelConnectivity.OnSaveChanges reads.
type SaveInput struct {
	Device Lookup // the device's cached properties (GetProperty)

	Selection        string // m_cmbBackOffices.GetValue().ToString().Trim()
	SelectionChanged bool   // m_cmbBackOffices.IsChanged
	CurrentPreset    string // m_bopresetName (ParseShortName(8310).preset)
	MeterName        string // m_meterName (ParseShortName(8310).meter)
	Presets          Catalog
	Firmware         Version // ICULanDevice.FirmwareVersionNumber
	IsAHP            bool

	HasNetworkProfiles      bool // HasNetworkProfiles(Device)
	HasExtendedFieldLengths bool // HasExtendedFieldLengths(Device)

	// Network-profile selects (read only when HasNetworkProfiles).
	ProfilePriorities       [NetworkProfileCount]byte      // m_cmbNP_Priority[i]
	ProfilePriorityChanged  [NetworkProfileCount]bool      // m_cmbNP_Priority[i].IsChanged
	ProfileOCPPVersions     [NetworkProfileCount]NPVersion // m_cmbNP_OcppVersion[i]
	ProfileSecurityProfiles [NetworkProfileCount]string    // m_cmbNP_SecurityProfile[i] ("0".."3")

	// Legacy connect-method select (read only without network profiles).
	ConnectMode        string // m_cmbConnectMode.GetValue()
	ConnectModeChanged bool   // m_cmbConnectMode.IsChanged
}

// SavePlan is the ordered set of device writes PanelConnectivity.OnSaveChanges
// performs for the backoffice section.
type SavePlan struct {
	// Selection is the preset select value after saving ("_SA" when every
	// network profile is off).
	Selection string
	// ClearFirst is stored unconditionally (StoreProperties, not only changed
	// values) BEFORE UploadFile; empty when nothing is uploaded.
	ClearFirst []api.Property
	// ClearRetry are the ClearFirst entries that differ from the device value:
	// if storing ClearFirst fails they stay "changed" and go out with Writes.
	ClearRetry []api.Property
	// UploadFile is the preset file to upload verbatim via /api/firmware.
	UploadFile string
	// AfterUpload (8310 = "<preset>[,<meter>]") is written only when the
	// upload succeeded.
	AfterUpload []api.Property
	// Writes are the remaining properties that differ from the device value
	// (StoreChangedProperties).
	Writes []api.Property
	// DisableProxy mirrors m_chkProxyEnabled.SetValue(false) (8471 is in Writes).
	DisableProxy bool
	// ProfilesChanged is the C# `flag`: a network-profile priority changed in
	// manual mode, which (like an upload) triggers RebootQuestion.
	ProfilesChanged bool
}

// SecurityProfileError is PlanSave's error for an AHP network profile that
// uses OCPP 2.0.1 with security profile "0: Default".
type SecurityProfileError struct{ Profile int } // 1-based

func (e SecurityProfileError) Error() string {
	return fmt.Sprintf("You cannot use \"0: Default\" as security profile in Network Profile %d when using OCPP 2.0.1 or higher", e.Profile)
}

// PlanSave ports the backoffice section of PanelConnectivity.OnSaveChanges:
//
//   - "_MAN"/"_SA": 8310 = CombineShortName("", meter), 8321 = "ocpp/json";
//     "_SA" also clears every backoffice property.
//   - a preset that changed (or differs from the device's 8310 name): resolve
//     the preset file; if found, clear everything (stored first), upload the
//     file, then 8310 = CombineShortName(preset, meter).
//   - the unchanged preset with network profiles but all priorities off:
//     falls back to "_SA" (8310/8321 + clear).
//   - without network profiles: 8471 (proxy) = false unless wired, and 8311 =
//     the connect method when it changed.
//   - manual mode with network profiles: priorities must be unique, and AHP
//     devices may not combine OCPP 2.0.1 with security profile 0.
//
// The proxy password, authorization key, Wi-Fi PSK and Eichrecht parts of
// OnSaveChanges are not backoffice logic and are not covered.
func PlanSave(in SaveInput) (SavePlan, error) {
	text := csTrim(in.Selection)
	plan := SavePlan{Selection: text}
	manual := text == KeyManual
	standalone := text == KeyStandAlone
	w := newWriteSet(in.Device)
	var after *writeSet

	switch {
	case manual || standalone:
		w.set(PropShortName, 0, CombineShortName("", in.MeterName))
		w.set(PropProtocolName, 0, "ocpp/json")
		if standalone {
			clearAll(w, in.HasExtendedFieldLengths)
		}
	case in.SelectionChanged || text != in.CurrentPreset:
		if file := in.Presets.Resolve(text, in.Firmware); file != "" {
			cw := newWriteSet(in.Device)
			clearAll(cw, in.HasExtendedFieldLengths)
			plan.ClearFirst = cw.all()
			plan.ClearRetry = cw.changed()
			plan.UploadFile = file
			after = newWriteSet(in.Device)
			after.set(PropShortName, 0, CombineShortName(text, in.MeterName))
		}
	case in.HasNetworkProfiles:
		anyOn := false
		for _, p := range in.ProfilePriorities {
			if p != 0 {
				anyOn = true
				break
			}
		}
		if !anyOn {
			plan.Selection = KeyStandAlone
			w.set(PropShortName, 0, CombineShortName("", in.MeterName))
			w.set(PropProtocolName, 0, "ocpp/json")
			clearAll(w, in.HasExtendedFieldLengths)
		}
	}

	if !in.HasNetworkProfiles {
		text3 := in.ConnectMode
		strB := ""
		if in.Device != nil {
			if p, ok := in.Device(PropConnectMethod, 0); ok {
				strB = typedString(p)
			}
		}
		if text3 != ConnectMethodWired {
			w.set(PropProxyEnabled, 0, "False")
			plan.DisableProxy = true
		}
		if in.ConnectModeChanged || !strings.EqualFold(text3, strB) {
			w.set(PropConnectMethod, 0, text3)
		}
	} else if manual {
		seen := map[byte]bool{}
		nonZero := 0
		for i, p := range in.ProfilePriorities {
			if in.ProfilePriorityChanged[i] {
				plan.ProfilesChanged = true
			}
			if p > 0 {
				nonZero++
				seen[p] = true
			}
		}
		if len(seen) != nonZero {
			return SavePlan{}, ErrUniquePriorities
		}
		if in.IsAHP {
			for i := 0; i < NetworkProfileCount; i++ {
				if in.ProfileOCPPVersions[i] >= NPVersionOCPP201 && in.ProfileSecurityProfiles[i] == "0" {
					return SavePlan{}, SecurityProfileError{Profile: i + 1}
				}
			}
		}
	}

	plan.Writes = w.changed()
	if after != nil {
		plan.AfterUpload = after.changed()
	}
	return plan, nil
}

// Uploader uploads a preset file (ICUDevice.UploadFirmware(null, path)).
type Uploader func(path string) error

// UploadPresetFile returns a minimal Uploader: the file is read and posted
// verbatim with api.Client.UploadFirmware (POST /api/firmware, the path the
// C# uses for presets). It does NOT perform the rest of
// ICULanDevice.UploadFirmware(null, path) -> StartUpload: the device-date
// sync when 13824_1 != 3, the "upload already in progress" check, the
// 3-attempt loop with 401/403 re-login and the 900 s timeout. Pass a full
// StartUpload port as Execute's uploader when one is available.
func UploadPresetFile(c *api.Client) Uploader {
	return func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		resp, err := c.UploadFirmware(data)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("Couldn't communicate with the device, please reboot the device. (HTTP %d)", resp.StatusCode)
		}
		return nil
	}
}

// ExecResult reports what Execute did.
type ExecResult struct {
	ClearErr  error // StoreProperties(ClearFirst) failed (ignored by the C#)
	Uploaded  bool  // flag2: the preset upload succeeded
	UploadErr error
	StoreErr  error // StoreChangedProperties failed (ignored by the C#)
	// AskReboot: the C# now asks RebootQuestion (upload done or profile
	// priorities changed).
	AskReboot bool
}

// Execute runs a SavePlan in the C# order: store ClearFirst, upload the
// preset, then store the changed properties (Writes, AfterUpload when the
// upload succeeded, ClearRetry when the clear failed, plus any other changed
// panel properties passed as extra) in one StoreProperties call. upload may be
// nil to use UploadPresetFile(c). Like the C#, a failing step does not stop
// the following ones.
func Execute(c *api.Client, plan SavePlan, upload Uploader, extra ...api.Property) ExecResult {
	var res ExecResult
	if upload == nil {
		upload = UploadPresetFile(c)
	}
	var writes []api.Property
	if plan.UploadFile != "" {
		if len(plan.ClearFirst) > 0 {
			res.ClearErr = c.StoreProperties(plan.ClearFirst...)
			if res.ClearErr != nil {
				writes = append(writes, plan.ClearRetry...)
			}
		}
		if err := upload(plan.UploadFile); err != nil {
			res.UploadErr = err
		} else {
			res.Uploaded = true
		}
	}
	writes = mergeWrites(writes, plan.Writes)
	if res.Uploaded {
		writes = mergeWrites(writes, plan.AfterUpload)
	}
	writes = mergeWrites(writes, extra)
	if len(writes) > 0 {
		res.StoreErr = c.StoreProperties(writes...)
	}
	res.AskReboot = res.Uploaded || plan.ProfilesChanged
	return res
}

// mergeWrites appends b to a, replacing entries for the same property.
func mergeWrites(a, b []api.Property) []api.Property {
	for _, p := range b {
		replaced := false
		for i := range a {
			if a[i].ID == p.ID && a[i].Sub == p.Sub {
				a[i] = p
				replaced = true
				break
			}
		}
		if !replaced {
			a = append(a, p)
		}
	}
	return a
}

// SelectionEffects is what PanelConnectivity.OnBackOfficeSelectChanged does
// to the other controls when the preset select changes.
type SelectionEffects struct {
	// Ignored: the new value equals the device's preset (case-insensitive).
	Ignored bool
	// SetFirstProfilePriority: "_MAN" with all profiles off -> profile 1
	// gets priority 1.
	SetFirstProfilePriority bool
	// SetConnectModeNone: "_SA" without network profiles -> connect method "0".
	SetConnectModeNone bool
	// ClearProfilePriorities: "_SA" with network profiles -> all priorities 0.
	ClearProfilePriorities bool
	// ClearAPNFields: switching from a preset to "_MAN"/"_SA" blanks the APN
	// name/user/password and SIM pin inputs.
	ClearAPNFields bool
	// WasPreset is the new prevSelectionPreset to pass next time (when
	// Ignored, keep the previous value, which may still be unset).
	WasPreset bool
}

// SelectionChanged ports PanelConnectivity.OnBackOfficeSelectChanged.
// prevWasPreset is the nullable prevSelectionPreset (nil before the first
// change); priorities are the current m_cmbNP_Priority values.
func SelectionChanged(selection, currentPreset string, prevWasPreset *bool, hasNetworkProfiles bool, priorities [NetworkProfileCount]byte) SelectionEffects {
	var e SelectionEffects
	if strings.EqualFold(selection, currentPreset) {
		e.Ignored = true
		return e
	}
	isPreset := selection != KeyManual && selection != KeyStandAlone
	prev := isPreset
	if prevWasPreset != nil {
		prev = *prevWasPreset
	}
	switch selection {
	case KeyManual:
		if hasNetworkProfiles {
			anyOn := false
			for _, p := range priorities {
				if p != 0 {
					anyOn = true
					break
				}
			}
			e.SetFirstProfilePriority = !anyOn && NetworkProfileCount != 0
		}
		e.ClearAPNFields = prev
	case KeyStandAlone:
		if !hasNetworkProfiles {
			e.SetConnectModeNone = true
		} else {
			e.ClearProfilePriorities = true
		}
		e.ClearAPNFields = prev
	}
	e.WasPreset = isPreset
	return e
}

// ControlsInput is the panel state PanelConnectivity.OnUpdateControls reads.
type ControlsInput struct {
	Selection          string  // preset select value
	SupportsModem      bool    // ICULanDevice.SupportsModem()
	Firmware           Version // FirmwareVersionNumber
	HasNetworkProfiles bool
	Protocol           string // m_cmbProtocol value (legacy firmware), e.g. "1.5"
	ConnectMode        string // m_cmbConnectMode value (legacy firmware)
}

// ControlState is the backoffice-dependent part of
// PanelConnectivity.OnUpdateControls.
type ControlState struct {
	// ConnectMethodOptions is the connect-method list (SetCustomList).
	ConnectMethodOptions []Option
	// ConnectMethodEnabled: more than one option.
	ConnectMethodEnabled bool
	// ForceConnectMethodNone: "_SA" forces the select to "0".
	ForceConnectMethodNone bool
	// ShowWiredOnlyWarning shows MsgWiredOnly (only when no option is left;
	// unreachable in practice because "0" is never removed).
	ShowWiredOnlyWarning bool
	// URLFieldsEnabled: the four backoffice URL/path inputs are editable
	// (firmware major < 5, or manual mode).
	URLFieldsEnabled bool
	// WiredURLHidden: the wired URL/path inputs are hidden ("_SA").
	WiredURLHidden bool
	// APNFieldsHidden: APN name/user/password are only shown in manual mode.
	APNFieldsHidden bool
	// SimPinHidden: the SIM pin input is hidden in manual mode (sic).
	SimPinHidden bool
	// ShowSmartChargingProfiles / ShowSecurityExtensions: the "SC profiles"
	// and "Back office security" categories.
	ShowSmartChargingProfiles bool
	ShowSecurityExtensions    bool
	// ShowMobile / ShowProxy: the "Mobile" and proxy categories.
	ShowMobile bool
	ShowProxy  bool
}

// Controls ports the backoffice-dependent rules of PanelConnectivity.OnUpdateControls.
func Controls(in ControlsInput) ControlState {
	manual := in.Selection == KeyManual
	standalone := in.Selection == KeyStandAlone
	var opts []Option
	for _, o := range ConnectMethodTitles {
		if standalone && o.Key != ConnectMethodNone {
			continue
		}
		if !in.SupportsModem && (o.Key == ConnectMethodGPRS || o.Key == ConnectMethodAuto) {
			continue
		}
		opts = append(opts, o)
	}
	s := ControlState{ForceConnectMethodNone: standalone}
	if len(opts) == 0 {
		opts = []Option{{ConnectMethodNone, "None"}}
		s.ShowWiredOnlyWarning = !in.HasNetworkProfiles // label only exists on legacy firmware
	}
	s.ConnectMethodOptions = opts
	s.ConnectMethodEnabled = len(opts) > 1
	if in.HasNetworkProfiles {
		s.ShowSmartChargingProfiles = !standalone
		s.ShowSecurityExtensions = !standalone
		s.ShowMobile = true
		s.ShowProxy = true
	} else {
		s.ShowSmartChargingProfiles = !standalone && in.Protocol == "1.5"
		s.ShowSecurityExtensions = !standalone && in.Protocol != "1.5"
		cm := in.ConnectMode
		if standalone {
			cm = ConnectMethodNone
		}
		title := ""
		for _, o := range ConnectMethodTitles {
			if o.Key == cm {
				title = o.Title
			}
		}
		s.ShowMobile = title == "Mobile" || title == "Autodetect"
		s.ShowProxy = title == "Wired"
	}
	s.URLFieldsEnabled = in.Firmware.Major < 5 || manual
	s.WiredURLHidden = standalone
	s.APNFieldsHidden = !manual
	s.SimPinHidden = manual
	return s
}
