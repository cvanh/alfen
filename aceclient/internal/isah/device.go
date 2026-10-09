package isah

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// User carries the two ICUSettings.ICUUser fields that the IWS callers read
// (ACESettings/ICUSettings/ICUUser.cs). Both values come from the user's
// configuration, never from this package.
type User struct {
	// User is ICUUser.User. LoadSettingsFromIWS sends it as AdditionalInfo.
	User string
	// Company is ICUUser.Company. Every installer call site passes it as the
	// IWS "security"/IsahCredentials string (sent as Basic base64(ASCII(...))).
	Company string
}

// IWSDevice carries the ICULanDevice values that LoadSettingsFromIWS reads.
type IWSDevice struct {
	// ObjectID is GetPropertyString(8273, 0, 0) (PropObjectID), as read from
	// the device. LoadSettingsFromIWS lower-cases and trims it.
	ObjectID string
	// NumberOfSockets is ICUDevice.NumberOfSockets.
	NumberOfSockets int
	// ProcessorID is GetPropertyUInt64(8608, 0, 0) (PropProcessorID); 0 if absent.
	ProcessorID uint64
}

// UnknownObjectError is returned by LoadSettingsFromIWS when IWS did not give
// a 2xx response with an object. It is the C# path that logs "The current
// {ObjectId} is unknown! Please contact Alfen!" and returns false. Response
// is the IWS result, for diagnostics.
type UnknownObjectError struct {
	ObjectID string
	Response *IsahResponse
}

// Error renders the C# Serilog message (a string property is quoted).
func (e *UnknownObjectError) Error() string {
	return fmt.Sprintf("The current %q is unknown! Please contact Alfen!", e.ObjectID)
}

// ErrNoCurrentUser is the Go form of the NullReferenceException that
// LoadSettingsFromIWS catches when currentUser is null.
var ErrNoCurrentUser = errors.New("isah: no current user (ICUUser is null)")

// LoadSettingsFromIWS ports ICULanDevice.LoadSettingsFromIWS
// (ACENetwork/ICUNetwork/ICULanDevice.cs). The C# returns bool plus a ref
// IWSObject. This port returns the object, or an error where C# returns false.
//
// Request: GetObjectData(isahSite, isahCredentials,
// objectId = dev.ObjectID.ToLowerInvariant().Trim(),
// NumberOfSockets = dev.NumberOfSockets, ProcessorId = dev.ProcessorID as decimal,
// AdditionalInfo = currentUser.User, and the defaults for the rest).
//
// The C# does not pass includeLogo to GetObjectData, so the request always has
// IncludeLogo=True. This port keeps that, and includeLogo is accepted but has
// no effect. (PanelInformation's "update license key" passes includeLogo: false.)
// The C# also reads property 8611 and discards it. That has no effect here.
//
// Callers pass isahCredentials = currentUser.Company and isahSite = DefaultSite.
func (c *Connection) LoadSettingsFromIWS(ctx context.Context, currentUser *User, isahSite, isahCredentials string, dev IWSDevice, includeLogo bool) (*IWSObject, error) {
	_ = includeLogo // not forwarded, as in C#
	text := strings.TrimSpace(strings.ToLower(dev.ObjectID))
	if currentUser == nil {
		return nil, ErrNoCurrentUser
	}
	req := NewObjectDataRequest(text)
	req.NumberOfSockets = dev.NumberOfSockets
	req.ProcessorID = strconv.FormatUint(dev.ProcessorID, 10)
	req.AdditionalInfo = currentUser.User
	r, err := c.GetObjectData(ctx, isahSite, isahCredentials, req)
	if err != nil {
		return nil, err
	}
	if r.IsahObject != nil && r.HTTPResponse.IsSuccessStatusCode() {
		return r.IsahObject, nil
	}
	return nil, &UnknownObjectError{ObjectID: text, Response: r}
}

// LoadSettingsFromIWS calls Connection.LoadSettingsFromIWS on the default connection.
func LoadSettingsFromIWS(ctx context.Context, currentUser *User, isahSite, isahCredentials string, dev IWSDevice, includeLogo bool) (*IWSObject, error) {
	return defaultConnection.LoadSettingsFromIWS(ctx, currentUser, isahSite, isahCredentials, dev, includeLogo)
}

// Object ID formats from DlgObjectID (ACEServiceInstaller/DlgObjectID.cs),
// matched with RegexOptions.IgnoreCase: the old "12345r01" style and the new
// "ACE0001234" style.
const (
	ObjectNrFormat    = `^[0-9]{5,6}r[0-9]{2,3}$` // m_regexObjectNrFormat
	NewObjectNrFormat = `^ACE[0-9]{7}$`           // m_regexNewObjectNrFormat
)

var (
	objectNrRe    = regexp.MustCompile(`(?i)` + ObjectNrFormat)
	newObjectNrRe = regexp.MustCompile(`(?i)` + NewObjectNrFormat)
)

// NormalizeObjectID ports `m_txtObjectID.Text.ToLower().Trim()` in
// DlgObjectID.OnCommandActivated. The dialog validates and checks this
// normalized text, but its result (NewObjectID) is the raw entered Text.
func NormalizeObjectID(text string) string {
	return strings.TrimSpace(strings.ToLower(text))
}

// ValidateObjectID ports the regex check in DlgObjectID.OnCommandActivated on
// the normalized text. On a mismatch it returns the dialog's error message.
func ValidateObjectID(normalized string) error {
	if !objectNrRe.MatchString(normalized) && !newObjectNrRe.MatchString(normalized) {
		return fmt.Errorf("Invalid Object ID: %s!\nPlease provide a correct one.", normalized)
	}
	return nil
}

// ObjectIDCheck is the result of DlgObjectID.CheckIfObjectIDExists, before the
// user answers.
type ObjectIDCheck int

const (
	// ObjectIDUnverified: there is no current user, or TestIWSConnection
	// failed. The dialog asks Yes/No/Cancel, and Yes accepts the ID.
	ObjectIDUnverified ObjectIDCheck = iota
	// ObjectIDKnown: IWS returned an object. The dialog shows an info message
	// and accepts the ID.
	ObjectIDKnown
	// ObjectIDUnknown: IWS did not return an object. The dialog asks
	// Yes/No/Cancel, and Yes accepts the ID.
	ObjectIDUnknown
)

// String returns the Go constant name.
func (c ObjectIDCheck) String() string {
	switch c {
	case ObjectIDUnverified:
		return "ObjectIDUnverified"
	case ObjectIDKnown:
		return "ObjectIDKnown"
	case ObjectIDUnknown:
		return "ObjectIDUnknown"
	}
	return "ObjectIDCheck(" + strconv.Itoa(int(c)) + ")"
}

// NeedsConfirmation reports whether DlgObjectID asks a Yes/No/Cancel question
// (only Yes accepts the ID) instead of an info message.
func (c ObjectIDCheck) NeedsConfirmation() bool { return c != ObjectIDKnown }

// Message returns the text that DlgObjectID.CheckIfObjectIDExists shows for
// the result.
func (c ObjectIDCheck) Message(objectID string) string {
	switch c {
	case ObjectIDKnown:
		return "The Object ID of the Charge Station will be changed to " + objectID + "."
	case ObjectIDUnknown:
		return "The entered Object ID seems to be invalid. Are you sure to change the Object ID to " + objectID + "?"
	default:
		return "Unable to verify the entered Object ID with ISAH. Are you sure to change the Object ID to " + objectID + "?"
	}
}

// CheckIfObjectIDExists ports the non-UI part of
// DlgObjectID.CheckIfObjectIDExists:
//   - currentUser == nil, or TestIWSConnection(isahSite) fails: ObjectIDUnverified;
//   - else GetObjectData(isahSite, currentUser.Company, objectID,
//     numberOfSockets, processorID as decimal) with the C# defaults
//     (AdditionalInfo is "" here): ObjectIDKnown if it gives a 2xx response
//     with an object, else ObjectIDUnknown.
//
// Before the call, the C# refreshes the "generic" category
// (lanDevice.UpdateCategories("generic")) and reads processorID with
// GetPropertyUInt64(8608, 0, 0). The caller must do that. The error is
// non-nil only if isahSite is not an absolute URI, where the dialog would throw.
func (c *Connection) CheckIfObjectIDExists(ctx context.Context, isahSite string, currentUser *User, objectID string, numberOfSockets int, processorID uint64) (ObjectIDCheck, error) {
	if currentUser == nil {
		return ObjectIDUnverified, nil
	}
	ok, err := c.TestIWSConnection(ctx, isahSite)
	if err != nil {
		return ObjectIDUnverified, err
	}
	if !ok {
		return ObjectIDUnverified, nil
	}
	req := NewObjectDataRequest(objectID)
	req.NumberOfSockets = numberOfSockets
	req.ProcessorID = strconv.FormatUint(processorID, 10)
	r, err := c.GetObjectData(ctx, isahSite, currentUser.Company, req)
	if err != nil {
		return ObjectIDUnknown, err
	}
	if r.IsahObject != nil && r.HTTPResponse.IsSuccessStatusCode() {
		return ObjectIDKnown, nil
	}
	return ObjectIDUnknown, nil
}

// CheckIfObjectIDExists calls Connection.CheckIfObjectIDExists on the default connection.
func CheckIfObjectIDExists(ctx context.Context, isahSite string, currentUser *User, objectID string, numberOfSockets int, processorID uint64) (ObjectIDCheck, error) {
	return defaultConnection.CheckIfObjectIDExists(ctx, isahSite, currentUser, objectID, numberOfSockets, processorID)
}
