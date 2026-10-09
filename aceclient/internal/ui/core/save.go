package core

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"
)

// IdentityOrModelChangedText is SaveAllChanges' message when 8272/8273/8275
// changed (the C# also removes the device so it is re-added by discovery).
const IdentityOrModelChangedText = "The Identity or model has changed, the device will be re-added automatically to the Service Installer"

// SaveResult reports the side effects of SaveAllChanges.
type SaveResult struct {
	// IdentityOrModelChanged is SaveAllChanges' flag: 8273, 8275 and 8272
	// all exist and at least one of them was changed.
	IdentityOrModelChanged bool
	// ConfigurationNotice is ValidateCSConfiguration's DlgInfo text (empty
	// when nothing was adjusted).
	ConfigurationNotice string
}

// SaveAllChanges ports the device half of MainWindow.SaveAllChanges (the
// panels' OnSaveChanges checks run before it, see SaveCheck):
//
//  1. 8273 (object number) changed → store it on its own;
//  2. 8609 (feature license key) changed → store it on its own;
//  3. 8583 (last configuration change) present → set it to the current Unix
//     time in ms (Math.Round(TotalMilliseconds)) and store it on its own;
//  4. ValidateCSConfiguration;
//  5. StoreChangedProperties (all remaining changed properties, 15 per POST).
//
// Any failing store aborts. Fidelity note: with the EDS pre-populating the C#
// dictionary, step 3 fires for every device whose EDS knows 8583; here it
// fires when the device reported 8583.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1589-1660
func (s *Session) SaveAllChanges(now time.Time) (SaveResult, error) {
	var res SaveResult
	pc := s.Props
	p8273, ok1 := pc.Get(8273, 0)
	p8275, ok2 := pc.Get(8275, 0)
	p8272, ok3 := pc.Get(8272, 0)
	if ok1 && ok2 && ok3 && (p8273.Changed || p8275.Changed || p8272.Changed) {
		res.IdentityOrModelChanged = true
	}
	if ok1 && p8273.Changed {
		if err := s.StoreProperties(p8273); err != nil {
			return res, err
		}
	}
	if p, ok := pc.Get(8609, 0); ok && p.Changed {
		if err := s.StoreProperties(p); err != nil {
			return res, err
		}
	}
	if _, ok := pc.Get(8583, 0); ok {
		ms := math.Round(float64(now.UTC().UnixNano()) / 1e6)
		p, err := pc.SetValue(8583, 0, FormatDotNetFloat(ms, 64))
		if err == nil {
			if err := s.StoreProperties(p); err != nil {
				return res, err
			}
		}
	}
	res.ConfigurationNotice = ValidateCSConfiguration(pc, s.NumberOfSockets(), s.numberOfFeederCablesValue(), s.Identification())
	if changed := pc.Changed(); len(changed) > 0 {
		if err := s.StoreProperties(changed...); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (s *Session) numberOfFeederCablesValue() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.numberOfFeederCables
}

// ValidateCSConfiguration ports MainWindow.ValidateCSConfiguration (with
// updateValues = false): on a multi-socket, single-feeder station whose two
// connector maximum currents (8489, 12585) exceed the station current (8290),
// and when neither an SCN (8576_1 empty) nor static load balancing
// (8292 & 1 == 0) is configured, both connector maxima are set to half the
// station current (and 33793_1/_2 to half*230*3 when present). It returns the
// DlgInfo text, or "" when nothing changed.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1860-1888
func ValidateCSConfiguration(pc *PropertyCache, numberOfSockets, numberOfFeederCables int, identification string) string {
	if numberOfSockets <= 1 || numberOfFeederCables > 1 {
		return ""
	}
	a := pc.Int(8489, 0)
	b := pc.Int(12585, 0)
	station := pc.Int(8290, 0)
	if a+b <= station {
		return ""
	}
	noSCN := len(pc.String(8576, 1)) == 0
	noStatic := pc.Int(8292, 0, 1) == 0
	if !(noSCN && noStatic) {
		return ""
	}
	half := station / 2
	_, _ = pc.SetValue(8489, 0, strconv.Itoa(half))
	_, _ = pc.SetValue(12585, 0, strconv.Itoa(half))
	if pc.Has(33793, 1) {
		_, _ = pc.SetValue(33793, 1, strconv.Itoa(half*230*3))
		_, _ = pc.SetValue(33793, 2, strconv.Itoa(half*230*3))
	}
	return fmt.Sprintf("The total sum of the connectors maximum current (%dA) is more than the station current (%dA)!\n", a+b, station) +
		"This is only allowed in combination with Static Loadbalancing or Smart Charging Network.\n\n" +
		fmt.Sprintf("NOTE: The charing station %s connector maximum currents will be set to %dA.", identification, half)
}

// SaveCheck is one panel's OnSaveChanges, evaluated before SaveAllChanges
// touches the device. A non-empty Error aborts the save (MessageDialog
// .ShowError); a non-empty Question must be answered Yes to continue
// (MessageDialog.AskQuestion).
type SaveCheck struct {
	Error    string
	Question string
}

// InformationSaveCheck ports PanelInformation.OnSaveChanges: a changed
// Secure Service Access flag (8626) asks for confirmation (and, as in the
// C#, skips the identity check); otherwise a changed identity (8275) must
// match ^[a-zA-Z0-9*\-_=:+|@.]{1,20}$ (48 on AHP).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs:582-604
func InformationSaveCheck(pc *PropertyCache, isAHP bool, identity string) SaveCheck {
	if p, ok := pc.Get(8626, 0); ok && p.Changed {
		if !CheckState(p.Value) {
			return SaveCheck{Question: "Are you sure you want to disable Secure Service Access?\n\nBy disabling SSA, Alfen certified service engineers will need your (temporary) password to service your charging station " + identity + ".\n\nAdditionally they cannot help you with resetting your password in case it gets lost."}
		}
		return SaveCheck{Question: "Are you sure you want to enable Secure Service Access?\n\nWith SSA enabled, an Alfen certified service engineer can configure this charging station (" + identity + ") without the need to share any password.\n\nThey can also recover access in case of a lost password."}
	}
	if p, ok := pc.Get(8275, 0); ok && p.Changed {
		if msg := ValidateIdentity(p.Value, isAHP); msg != "" {
			return SaveCheck{Error: msg}
		}
	}
	return SaveCheck{}
}

var (
	identityRe20 = regexp.MustCompile(`^[a-zA-Z0-9*\-_=:+|@.]{1,20}$`)
	identityRe48 = regexp.MustCompile(`^[a-zA-Z0-9*\-_=:+|@.]{1,48}$`)
)

// ValidateIdentity ports the "Customer Ident. Number" rule of
// PanelInformation.OnSaveChanges; it returns the error text or "".
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs:582-604
func ValidateIdentity(identity string, isAHP bool) string {
	re := identityRe20
	if isAHP {
		re = identityRe48
	}
	if re.MatchString(identity) {
		return ""
	}
	msg := "The field \"Customer Ident. Number\" is invalid.\nOnly characters, digits and *-_=:+|@. are allowed"
	if isAHP {
		return msg + "\nand the maximum length is 48 positions."
	}
	return msg + "."
}

// TamperState mirrors ICUServiceInstaller.ETamperState.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/ETamperState.cs
type TamperState int

const (
	TamperUnknown TamperState = iota
	TamperNotTampered
	TamperActive
	Tampered
	TamperedAck
)

// Tamper alarm texts from MainWindow.OnDeviceSelectionChanged.
const (
	TamperQuestionPrimary   = "The charging station might have been tampered"
	TamperQuestionSecondary = "Do you want to acknowledge the tamper alarm?"
)

// TamperAlarm ports the check in OnDeviceSelectionChanged: HasProperty(8784)
// and its value is Tampered or TamperActive.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1038-1240 (tamper check)
func TamperAlarm(pc *PropertyCache) bool {
	if !pc.Has(8784, 0) {
		return false
	}
	st := TamperState(pc.Int(8784, 0))
	return st == Tampered || st == TamperActive
}

// AcknowledgeTamper stores 8784_0 = 4 (Tampered_Ack), as the C# does after a
// Yes to the tamper question.
func (s *Session) AcknowledgeTamper() error {
	p, err := s.Props.SetValue(8784, 0, "4")
	if err != nil {
		return err
	}
	return s.StoreProperties(p)
}
