package i18n

import (
	"fmt"
	"strings"
)

// TooltipID returns the tooltip key of a property: the tooltipID == 0 branch
// of UIPropertyBase.MakeToolTipID (UIPropertyBase.cs),
// $"{m_usId:X4}_{m_bSubId:X2}" — e.g. (0x2062, 0) -> "2062_00",
// (0x2180, 0x0A) -> "2180_0A".
//
// Note this differs from api.Property.IDSub ("%X_%X", unpadded), which is
// the wire format, not the tooltip key.
func TooltipID(id uint16, sub uint8) string {
	return fmt.Sprintf("%04X_%02X", id, sub)
}

// MakeTooltipID ports UIPropertyBase.MakeToolTipID (UIPropertyBase.cs). The
// UIProperty* constructors take an optional packed tooltipID = index<<8 | sub
// that overrides the control's own (id, sub); 0 means "use the property":
//
//	if (id == 0) return $"{m_usId:X4}_{m_bSubId:X2}";
//	return $"{id >> 8:X4}_{id & 0xFF:X2}";
//
// E.g. PanelSCNSettings passes 2195461u (0x218005) -> "2180_05". Controls
// built from a label only have id = sub = 0, so without an override their
// key is "0000_00".
func MakeTooltipID(id uint16, sub uint8, tooltipID uint32) string {
	if tooltipID == 0 {
		return TooltipID(id, sub)
	}
	return fmt.Sprintf("%04X_%02X", tooltipID>>8, tooltipID&0xFF)
}

// HasTooltip is Tooltips.HasTooltip on Default: the information-button rule
// of the UIPropertyBase constructors (m_btnTooltip.Visible).
func HasTooltip(tooltipID string) bool {
	return Default.HasTooltip(tooltipID)
}

// MakeTooltip is Tooltips.MakeTooltip on Default
// (UIPropertyBase.MakeToolTip).
func MakeTooltip(text, tooltipID string) string {
	return Default.MakeTooltip(text, tooltipID)
}

// ChangedTooltip is the hover text of the "changed" (pencil) icon that every
// UIPropertyBase sets on m_imvChanged.
const ChangedTooltip = "This property is changed"

// InvalidValueMessage ports the DlgInfo text built by
// UIPropertyBase.OnPropertyValueException (UIPropertyBase.cs) when a device
// property raises ExceptionOccured; title is ICUProperty.Title, invalidValue
// is ValueExceptionEventArgs.InvalidValue and excMessage its
// Exception.Message. The trailing space is in the original.
func InvalidValueMessage(title, invalidValue, excMessage string) string {
	return "Configuration item: " + title + " has an invalid value: " + invalidValue + ". Exception: " + excMessage + " "
}

// EDSLabel carries the two EDSParameter fields (ICUSettings.EDSParameter)
// that UIPropertyBase.LabelText reads. Units is nil when the EDS entry has no
// <Units> element; an empty but present <Units/> is a non-nil "".
type EDSLabel struct {
	Title string
	Units *string
}

// PropertyLabelText ports the UIPropertyBase.LabelText getter
// (UIPropertyBase.cs). eds is DataSheet.FindParameter(id, sub) (nil when not
// found) and labelText the constructor's label (m_sLabelText):
//
//   - EDS entry found: its Title, plus " (<Units>)" when Units is non-nil;
//     unknownTitle = false.
//   - else id == 0 && sub == 0 (label-only control): labelText; m_fUnknownTitle
//     is left untouched, i.e. false on a freshly built control.
//   - else: "XXXX" when sub == 0, otherwise "XXXX_<sub in DECIMAL>"
//     ($"{m_usId:X4}_{m_bSubId}"), unknownTitle = true. The controls later
//     replace an unknown title with ICUProperty.Title once the device
//     reports a value.
func PropertyLabelText(id uint16, sub uint8, eds *EDSLabel, labelText string) (text string, unknownTitle bool) {
	switch {
	case eds != nil:
		text = eds.Title
		if eds.Units != nil {
			text = fmt.Sprintf("%s (%s)", text, *eds.Units)
		}
		return text, false
	case id == 0 && sub == 0:
		return labelText, false
	case sub == 0:
		return fmt.Sprintf("%04X", id), true
	default:
		return fmt.Sprintf("%04X_%d", id, sub), true
	}
}

// ApplyOverwriteLabel ports the label override shared by the Initialize
// methods of UIString, UIPassword, UICheckbox, UIPropertyString,
// UIPropertyReadOnlyString and UIPropertyCheckbox:
//
//	m_lblTitle = new Label(LabelText);
//	if (!string.IsNullOrWhiteSpace(overwriteLabelText)) {
//	    m_lblTitle.Text = overwriteLabelText;
//	    m_fUnknownTitle = false;
//	}
//
// label/unknownTitle are PropertyLabelText's results. (UIPropertyNumber
// differs: a non-empty customLabel — string.IsNullOrEmpty, so whitespace
// counts — replaces the label without evaluating LabelText at all, so
// unknownTitle stays false.)
func ApplyOverwriteLabel(label string, unknownTitle bool, overwriteLabelText string) (string, bool) {
	if strings.TrimSpace(overwriteLabelText) != "" {
		return overwriteLabelText, false
	}
	return label, unknownTitle
}
