package eds

import (
	"fmt"
	"regexp"
	"strings"
)

// Texts and rules of ICUServiceInstaller.DlgObjectID (DlgObjectID.cs), the
// dialog PanelInformation.CheckIfCSIsInitialize opens when the Object Number
// (0x2051 sysChargePointSerialNumber) is empty or "UNKNOWN".
const (
	ObjectIDDialogTitle  = "Device Object ID"
	ObjectIDDialogPrompt = "Empty or invalid Object ID found, please enter a valid one."
	ObjectIDDialogLabel  = "Object ID:"

	// ObjectIDFormat is m_regexObjectNrFormat (matched case-insensitively).
	ObjectIDFormat = "^[0-9]{5,6}r[0-9]{2,3}$"
	// NewObjectIDFormat is m_regexNewObjectNrFormat (case-insensitive).
	NewObjectIDFormat = "^ACE[0-9]{7}$"

	// MsgInvalidObjectIDFmt is the error dialog for a malformed id; %s is the
	// lower-cased, trimmed input.
	MsgInvalidObjectIDFmt = "Invalid Object ID: %s!\nPlease provide a correct one."
	// MsgObjectIDUnverifiedFmt asks (Yes/No/Cancel) when ISAH cannot be
	// reached or there is no logged-in user.
	MsgObjectIDUnverifiedFmt = "Unable to verify the entered Object ID with ISAH. Are you sure to change the Object ID to %s?"
	// MsgObjectIDChangedFmt is shown when ISAH knows the object.
	MsgObjectIDChangedFmt = "The Object ID of the Charge Station will be changed to %s."
	// MsgObjectIDInvalidAskFmt asks (Yes/No/Cancel) when ISAH does not know it.
	MsgObjectIDInvalidAskFmt = "The entered Object ID seems to be invalid. Are you sure to change the Object ID to %s?"

	// ObjectIDPropID is the property the accepted id is stored to
	// (lanDev.storeProperty(8273, 0, NewObjectID)).
	ObjectIDPropID uint16 = 0x2051
	// ObjectIDSerialPropID is read (GetPropertyUInt64(8608)) and sent to ISAH
	// with the id, after UpdateCategories("generic").
	ObjectIDSerialPropID uint16 = 0x21A0
)

var (
	reObjectID    = regexp.MustCompile("(?i)" + ObjectIDFormat)
	reNewObjectID = regexp.MustCompile("(?i)" + NewObjectIDFormat)
)

// ValidateObjectID ports the format check of DlgObjectID.OnCommandActivated:
// the input is lower-cased and trimmed and must match ObjectIDFormat or
// NewObjectIDFormat (case-insensitive). On success the C# keeps the entry
// text as typed (NewObjectID = m_txtObjectID.Text), returned here as accepted;
// otherwise the error carries MsgInvalidObjectIDFmt.
func ValidateObjectID(input string) (accepted, checked string, err error) {
	checked = strings.TrimSpace(strings.ToLower(input))
	if !reObjectID.MatchString(checked) && !reNewObjectID.MatchString(checked) {
		return "", checked, fmt.Errorf(MsgInvalidObjectIDFmt, checked)
	}
	return input, checked, nil
}

// ObjectIDCheck holds the I/O DlgObjectID.CheckIfObjectIDExists performs, so
// the decision logic stays pure.
type ObjectIDCheck struct {
	// LoggedIn is currentUser != null.
	LoggedIn bool
	// Reachable ports IWSConnection.TestIWSConnection(AppProperties.IsahSite).
	Reachable func() bool
	// Lookup ports UpdateCategories("generic") + IWSConnection.GetObjectData:
	// true when ISAH returned an object with a success status.
	Lookup func(objectID string) bool
	// Ask shows a Yes/No/Cancel question and reports Yes.
	Ask func(msg string) bool
	// Show shows an informational message.
	Show func(msg string)
}

// CheckObjectIDExists ports DlgObjectID.CheckIfObjectIDExists; objectID is
// the lower-cased, trimmed text (ValidateObjectID's checked value). It reports
// whether the dialog may accept the id.
func CheckObjectIDExists(objectID string, c ObjectIDCheck) bool {
	ask := func(msg string) bool { return c.Ask != nil && c.Ask(msg) }
	if !c.LoggedIn || c.Reachable == nil || !c.Reachable() {
		return ask(fmt.Sprintf(MsgObjectIDUnverifiedFmt, objectID))
	}
	if c.Lookup != nil && c.Lookup(objectID) {
		if c.Show != nil {
			c.Show(fmt.Sprintf(MsgObjectIDChangedFmt, objectID))
		}
		return true
	}
	return ask(fmt.Sprintf(MsgObjectIDInvalidAskFmt, objectID))
}
