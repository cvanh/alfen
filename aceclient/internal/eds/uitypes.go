package eds

import (
	"fmt"
	"image/color"
)

// UIPropertyNumberType ports ICUServiceInstaller.UIPropertyNumberType
// (UIPropertyNumberType.cs). Declared but not referenced by the installer.
type UIPropertyNumberType int

const (
	NumberTypeUnknown UIPropertyNumberType = iota
	NumberTypeUInt8
	NumberTypeUInt16
	NumberTypeUInt32
	NumberTypeUInt64
	NumberTypeInt8
	NumberTypeInt16
	NumberTypeInt32
	NumberTypeInt64
	NumberTypeFloat
)

var numberTypeNames = [...]string{"Unknown", "UInt8", "UInt16", "UInt32", "UInt64", "Int8", "Int16", "Int32", "Int64", "Float"}

// String returns the C# enum member name.
func (t UIPropertyNumberType) String() string {
	if t >= 0 && int(t) < len(numberTypeNames) {
		return numberTypeNames[t]
	}
	return fmt.Sprintf("UIPropertyNumberType(%d)", int(t))
}

// UIPropertyStringType ports ICUServiceInstaller.UIPropertyStringType
// (UIPropertyStringType.cs): how a string/read-only-string control formats or
// filters its value.
type UIPropertyStringType int

const (
	StringTypeString UIPropertyStringType = iota
	StringTypeIP
	StringTypeDateTime
	StringTypeControllerVersion
	StringTypePowerVersion
	StringTypeNFC1SW
	StringTypeNFC1HW
	StringTypeNFC2SW
	StringTypeNFC2HW
	StringTypePWM
	StringTypePublicKey
	StringTypeFeatures
	StringTypeFloat
	StringTypeFloat0_001
	StringTypeFloat0_01
	StringTypeFloat0_1
	StringTypeFloat10
	StringTypeFloat2
	StringTypeFloat2_0_001
	StringTypeFloat2_0_01
	StringTypeFloat2_0_1
	StringTypeFloat2_10
)

var stringTypeNames = [...]string{
	"String", "IP", "DateTime", "ControllerVersion", "PowerVersion",
	"NFC1_SW", "NFC1_HW", "NFC2_SW", "NFC2_HW", "PWM", "PublicKey", "Features",
	"Float", "Float_0_001", "Float_0_01", "Float_0_1", "Float_10",
	"Float2", "Float2_0_001", "Float2_0_01", "Float2_0_1", "Float2_10",
}

// String returns the C# enum member name.
func (t UIPropertyStringType) String() string {
	if t >= 0 && int(t) < len(stringTypeNames) {
		return stringTypeNames[t]
	}
	return fmt.Sprintf("UIPropertyStringType(%d)", int(t))
}

// EUILabelType ports ICUServiceInstaller.EUILabelType (EUILabelType.cs).
type EUILabelType int

const (
	LabelNormal EUILabelType = iota
	LabelHeader
	LabelSmallHeader
	LabelWarning
	LabelInfo
	LabelError
	LabelLargeWarning
	LabelLargeInfo
	LabelLargeError
)

var labelTypeNames = [...]string{"Normal", "Header", "SmallHeader", "Warning", "Info", "Error", "LargeWarning", "LargeInfo", "LargeError"}

// String returns the C# enum member name.
func (t EUILabelType) String() string {
	if t >= 0 && int(t) < len(labelTypeNames) {
		return labelTypeNames[t]
	}
	return fmt.Sprintf("EUILabelType(%d)", int(t))
}

// EQuestion ports ICUServiceInstaller.EQuestion (EQuestion.cs).
type EQuestion int

const (
	QuestionUnknown EQuestion = iota
	QuestionOK
	QuestionNOK
	QuestionNA
)

var questionNames = [...]string{"Unknown", "OK", "NOK", "NA"}

// String returns the C# enum member name.
func (q EQuestion) String() string {
	if q >= 0 && int(q) < len(questionNames) {
		return questionNames[q]
	}
	return fmt.Sprintf("EQuestion(%d)", int(q))
}

// EQuestionType ports ICUServiceInstaller.EQuestionType (EQuestionType.cs).
type EQuestionType int

const (
	QuestionTypeNormal EQuestionType = iota
	QuestionTypeImportant
)

// String returns the C# enum member name.
func (t EQuestionType) String() string {
	switch t {
	case QuestionTypeNormal:
		return "Normal"
	case QuestionTypeImportant:
		return "Important"
	}
	return fmt.Sprintf("EQuestionType(%d)", int(t))
}

// Named colors (Xwt.Drawing.Colors) the property widgets use.
var (
	ColorSteelBlue = color.RGBA{0x46, 0x82, 0xB4, 0xFF}
	ColorOrange    = color.RGBA{0xFF, 0xA5, 0x00, 0xFF}
	ColorRed       = color.RGBA{0xFF, 0x00, 0x00, 0xFF}
	ColorBlack     = color.RGBA{0x00, 0x00, 0x00, 0xFF}
	ColorDarkGray  = color.RGBA{0xA9, 0xA9, 0xA9, 0xFF}
	ColorBlue      = color.RGBA{0x00, 0x00, 0xFF, 0xFF}
	ColorDarkBlue  = color.RGBA{0x00, 0x00, 0x8B, 0xFF}
	// AppProperties.Color_Disabled = Color.FromBytes(250, 250, 250): the
	// background of editable entries (sic: named "disabled" in the C#).
	ColorEditableBackground = color.RGBA{250, 250, 250, 0xFF}
	// AppProperties.Color_Border = Color.FromBytes(171, 173, 179).
	ColorBorder = color.RGBA{171, 173, 179, 0xFF}
)

// FontWeight names the Xwt font weights used by the property widgets.
type FontWeight int

const (
	WeightNormal FontWeight = iota
	WeightSemibold
	WeightBold
)

// LabelStyle is the presentation UIPropertyLabel applies per EUILabelType.
type LabelStyle struct {
	Color     color.RGBA // text color when enabled (DarkGray when disabled)
	Scale     float64    // font size relative to AppProperties.Font_BaseLabel
	Weight    FontWeight
	Centered  bool // TextAlignment.Center (Normal) vs Start
	ColSpan   int  // table columns the label spans (1 for Normal, 3 otherwise)
	WrapWords bool // WrapMode.Word (all types)
}

// StyleFor ports UIPropertyLabel.Initialize + SetType.
func StyleFor(t EUILabelType) LabelStyle {
	s := LabelStyle{Scale: 1.0, Weight: WeightNormal, ColSpan: 3, WrapWords: true}
	switch t {
	case LabelHeader:
		s.Color, s.Scale, s.Weight = ColorSteelBlue, 1.5, WeightSemibold
	case LabelSmallHeader:
		s.Color, s.Scale, s.Weight = ColorSteelBlue, 1.1, WeightSemibold
	case LabelInfo, LabelLargeInfo:
		s.Color = ColorSteelBlue
	case LabelWarning:
		s.Color = ColorOrange
	case LabelLargeWarning:
		s.Color, s.Scale, s.Weight = ColorOrange, 1.1, WeightBold
	case LabelError, LabelLargeError:
		s.Color, s.Scale, s.Weight = ColorRed, 1.1, WeightSemibold
	default:
		s.Color, s.Centered, s.ColSpan = ColorBlack, true, 1
	}
	return s
}

// LabelLines ports the reserved line count of UIPropertyLabel.Initialize:
// max(number of '\n'-separated lines, reserveLines).
func LabelLines(text string, reserveLines int) int {
	n := 1
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			n++
		}
	}
	if n < reserveLines {
		return reserveLines
	}
	return n
}

// UICustomField ports ICUServiceInstaller.UICustomField (UICustomField.cs): a
// table cell with its own text and colors (rendered by UICustomCell).
type UICustomField struct {
	Text    string
	TxColor color.RGBA
	BgColor color.RGBA
}

// OkNokNaLabels are the toggle captions of UIPropertyOkNokNa, indexed by
// EQuestion (Unknown has none).
var OkNokNaLabels = map[EQuestion]string{QuestionOK: "OK", QuestionNOK: "NOK", QuestionNA: "N.V.T."}

// OkNokNaButtonColor ports the toggle background of UIPropertyOkNokNa: Orange
// for EQuestionType.Important, AppProperties.Color_Disabled otherwise.
func OkNokNaButtonColor(t EQuestionType) color.RGBA {
	if t == QuestionTypeImportant {
		return ColorOrange
	}
	return ColorEditableBackground
}

// OkNokNaActive ports UIPropertyOkNokNa.OnRefreshDisplay: which of the three
// toggles (OK, NOK, N.V.T.) is pressed for value v.
func OkNokNaActive(v EQuestion) (ok, nok, na bool) {
	return v == QuestionOK, v == QuestionNOK, v == QuestionNA
}
