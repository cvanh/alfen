package presets

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// forbiddenProperties ports PropertyStorage.s_forbiddenProperties
// (ACEServiceInstaller/ICUServiceInstaller/PropertyStorage.cs): object indexes
// (any subindex) that are never written to nor loaded from a settings file.
var forbiddenProperties = []uint16{
	8271, 8273, 8274, 8275, 8277, 8281, 8287, 8309, 8317, 8451,
	8549, 8550, 8551, 8576, 9506, 9507, 9520, 8583, 8609, 8728,
	8729, 9568, 9569, 9570, 9571, 9572, 9573, 9584, 9585, 9586,
	9587, 9588, 9589, 12824, 16919, 16920, 21015, 21016,
}

// ForbiddenProperties returns a copy of PropertyStorage.s_forbiddenProperties.
func ForbiddenProperties() []uint16 { return slices.Clone(forbiddenProperties) }

// IsForbidden reports s_forbiddenProperties.Contains(id).
func IsForbidden(id uint16) bool { return slices.Contains(forbiddenProperties, id) }

// UI strings of the MainWindow settings/preset flows
// (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs: menu set-up,
// OnStoreSettings, OnLoadSettings, LoadProperties, OnLoadPresets).
const (
	MenuSaveSettingsAs = "_Save Settings As..."
	MenuLoadSettings   = "_Load Settings..."
	MenuLoadPreset     = "_Load Preset..."

	SaveSettingsDialogTitle = "Save Settings As"
	OpenSettingsDialogTitle = "Open Settings File"

	// SettingsFileExtension is the extension LoadProperties decrypts and the
	// save dialog proposes.
	SettingsFileExtension = ".exml"

	// SettingsLoadedMessage is shown (with SettingsLoadedDetail) when
	// LoadProperties changed at least one property.
	SettingsLoadedMessage = "Settings file successfully loaded."

	// IncorrectVersionMessage is LoadProperties' error for XMLVersion != "1.0".
	IncorrectVersionMessage = "Incorrect version of setting file"
)

// FileFilter is an Xwt FileDialogFilter (name + glob pattern).
type FileFilter struct {
	Name    string
	Pattern string
}

// SaveSettingsFilters are the filters of OnStoreSettings' SaveFileDialog.
var SaveSettingsFilters = []FileFilter{
	{"Encrypted Setting files", "*.exml"},
	{"All files", "*.*"},
}

// OpenSettingsFilters are the filters of OnLoadSettings' OpenFileDialog.
var OpenSettingsFilters = []FileFilter{
	{"Encrypted Setting files", "*.exml"},
	{"Setting files", "*.xml"},
	{"All files", "*.*"},
}

// DefaultSettingsFileName ports OnStoreSettings' InitialFileName:
// Identification + ".exml".
func DefaultSettingsFileName(identification string) string {
	return identification + SettingsFileExtension
}

// SettingsLoadedDetail ports the second line of MainWindow.LoadProperties'
// success message.
func SettingsLoadedDetail(changed int) string {
	return fmt.Sprintf("Changed %d properties.\nPlease save the changes to the device.", changed)
}

// ErrIncorrectVersion is returned by LoadProperties when the file's
// XMLVersion is not "1.0" (the C# shows IncorrectVersionMessage).
var ErrIncorrectVersion = errors.New(IncorrectVersionMessage)

// ModelMismatchQuestion ports the MessageDialog.AskQuestion text of
// PropertyStorage.LoadProperties when the file's model differs from the
// device's: fileModel is From(<Model>) and deviceModel is ICULanDevice.Model.
func ModelMismatchQuestion(fileModel DeviceModel, deviceModel string) (question, detail string) {
	return "Warning! The setting you are trying to load are for a '" + fileModel.Display("") +
			"', you currently have selected a '" + deviceModel + "'.",
		"Are you sure you want to load this settings file?"
}

// ConfirmFunc asks the user a Yes/No/Cancel question and reports whether
// "Yes" was chosen (MessageDialog.AskQuestion(...) == Command.Yes).
type ConfirmFunc func(question, detail string) bool

// ---- saving ----

// xmlWriter reproduces the output of XDocument.ToString() on Windows
// (XmlWriter with Indent = true, OmitXmlDeclaration = true, two-space
// indentation, "\r\n" new lines, NewLineHandling.Replace, CheckCharacters).
type xmlWriter struct {
	sb  strings.Builder
	err error
}

const xmlNewLine = "\r\n"

// isXMLChar reports whether r is allowed in XML 1.0 content.
func isXMLChar(r rune) bool {
	return r == 0x9 || r == 0xA || r == 0xD ||
		(r >= 0x20 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF)
}

func (w *xmlWriter) invalid(r rune) {
	if w.err == nil {
		w.err = fmt.Errorf("'%c', hexadecimal value 0x%02X, is an invalid character.", r, r)
	}
}

// text writes element content (XmlEncodedRawTextWriter.WriteElementTextBlock).
func (w *xmlWriter) text(s string) {
	s = toValidUTF8(s)
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '<':
			w.sb.WriteString("&lt;")
		case r == '>':
			w.sb.WriteString("&gt;")
		case r == '&':
			w.sb.WriteString("&amp;")
		case r == '\n':
			w.sb.WriteString(xmlNewLine)
		case r == '\r':
			if i+1 < len(s) && s[i+1] == '\n' {
				n++
			}
			w.sb.WriteString(xmlNewLine)
		case !isXMLChar(r):
			w.invalid(r)
		default:
			w.sb.WriteString(s[i : i+n])
		}
		i += n
	}
}

// attr writes an attribute value (WriteAttributeTextBlock).
func (w *xmlWriter) attr(s string) {
	s = toValidUTF8(s)
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '<':
			w.sb.WriteString("&lt;")
		case r == '>':
			w.sb.WriteString("&gt;")
		case r == '&':
			w.sb.WriteString("&amp;")
		case r == '"':
			w.sb.WriteString("&quot;")
		case r == '\t':
			w.sb.WriteString("&#x9;")
		case r == '\n':
			w.sb.WriteString("&#xA;")
		case r == '\r':
			w.sb.WriteString("&#xD;")
		case !isXMLChar(r):
			w.invalid(r)
		default:
			w.sb.WriteString(s[i : i+n])
		}
		i += n
	}
}

// line starts a new indented line (no new line before the root element).
func (w *xmlWriter) line(depth int) {
	if w.sb.Len() > 0 {
		w.sb.WriteString(xmlNewLine)
	}
	w.sb.WriteString(strings.Repeat("  ", depth))
}

// textElement writes <name>value</name>.
func (w *xmlWriter) textElement(depth int, name, value string) {
	w.line(depth)
	w.sb.WriteString("<" + name + ">")
	w.text(value)
	w.sb.WriteString("</" + name + ">")
}

// nullableElement writes new XElement(name, value) where an empty value
// stands for a C# null (<name />).
func (w *xmlWriter) nullableElement(depth int, name, value string) {
	if value == "" {
		w.line(depth)
		w.sb.WriteString("<" + name + " />")
		return
	}
	w.textElement(depth, name, value)
}

// xmlDateTimeLocal ports XmlConvert.ToString(DateTime.Now,
// XmlDateTimeSerializationMode.RoundtripKind) for a DateTimeKind.Local value:
// yyyy-MM-ddTHH:mm:ss, a 100 ns fraction without trailing zeros, and the
// local UTC offset as +hh:mm / -hh:mm (never "Z").
func xmlDateTimeLocal(t time.Time) string {
	var sb strings.Builder
	sb.WriteString(t.Format("2006-01-02T15:04:05"))
	if ticks := t.Nanosecond() / 100; ticks != 0 {
		frac := fmt.Sprintf("%07d", ticks)
		sb.WriteString("." + strings.TrimRight(frac, "0"))
	}
	_, off := t.Zone()
	sign := byte('+')
	if off < 0 {
		sign = '-'
		off = -off
	}
	fmt.Fprintf(&sb, "%c%02d:%02d", sign, off/3600, (off%3600)/60)
	return sb.String()
}

// elementIDSub is ICUProperty.ID_SUB ("{Id:X4}_{SubId:X2}"), the Id attribute
// written by ICUProperty.Element.
func elementIDSub(id uint16, sub byte) string {
	return fmt.Sprintf("%04X_%02X", id, sub)
}

// BuildSettingsXML ports the document PropertyStorage.SaveProperties builds,
// returned as XDocument.ToString() text (before encryption). now plays
// DateTime.Now. Properties are written in dictionary order when
// !ReadOnly && Value != null && !forbidden, each as
// <Property Id="IIII_SS" Value="..." /> (ICUProperty.Element).
//
// The C# first refreshes the device (lanDevice.UpdateCategories()); callers
// must build dict from a fresh read.
func BuildSettingsXML(dev DeviceInfo, dict *Dictionary, now time.Time) (string, error) {
	w := &xmlWriter{}
	w.line(0)
	w.sb.WriteString("<Settings>")
	w.textElement(1, "XMLVersion", "1.0")
	w.textElement(1, "Version", "1.0")
	w.textElement(1, "Date", xmlDateTimeLocal(now))
	w.textElement(1, "Model", dev.Model)
	w.textElement(1, "NumberOfSockets", strconv.Itoa(dev.NumberOfSockets))
	w.textElement(1, "Information", "")
	w.line(1)
	w.sb.WriteString("<Device>")
	w.nullableElement(2, "Identity", dev.Identification)
	w.nullableElement(2, "IPAddress", dev.IPAddress)
	w.textElement(2, "Port", strconv.Itoa(dev.Port))
	w.nullableElement(2, "HostName", dev.HostName)
	w.line(1)
	w.sb.WriteString("</Device>")
	var props []*dictEntry
	if dict != nil {
		for _, e := range dict.entries {
			if !e.readOnly && e.value != nil && !IsForbidden(e.id) {
				props = append(props, e)
			}
		}
	}
	w.line(1)
	if len(props) == 0 {
		w.sb.WriteString("<Properties />")
	} else {
		w.sb.WriteString("<Properties>")
		for _, e := range props {
			w.line(2)
			w.sb.WriteString(`<Property Id="`)
			w.attr(elementIDSub(e.id, e.sub))
			w.sb.WriteString(`" Value="`)
			w.attr(e.value.elementString())
			w.sb.WriteString(`" />`)
		}
		w.line(1)
		w.sb.WriteString("</Properties>")
	}
	w.line(0)
	w.sb.WriteString("</Settings>")
	if w.err != nil {
		return "", w.err
	}
	return w.sb.String(), nil
}

// SaveProperties ports PropertyStorage.SaveProperties: the settings document
// is always encrypted (EncryptDecrypt with pass phrase "Alfen", Base64) and
// written to fileName, whatever its extension. The C# shows any error in a
// message box and returns false; here it is returned.
func SaveProperties(fileName string, dev DeviceInfo, dict *Dictionary, now time.Time) error {
	doc, err := BuildSettingsXML(dev, dict, now)
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, []byte(encryptString(doc, settingsPassPhrase)), 0o666)
}

// ---- loading ----

// ReadSettingsText ports the first half of PropertyStorage.LoadProperties:
// File.ReadAllText, then EncryptDecrypt.Decrypt when the extension is exactly
// ".exml" (case-sensitive, as string.Compare). Other files are plain XML.
func ReadSettingsText(fileName string) (string, error) {
	raw, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	text := readAllText(raw)
	if netGetExtension(fileName) == SettingsFileExtension {
		return decryptString(text, settingsPassPhrase)
	}
	return text, nil
}

// netGetExtension ports Path.GetExtension: "" when the name has no '.' after
// the last separator or ends with '.'.
func netGetExtension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		switch path[i] {
		case '.':
			if i == len(path)-1 {
				return ""
			}
			return path[i:]
		case '/', '\\', ':':
			return ""
		}
	}
	return ""
}

// errMissing stands in for the NullReferenceException the C# hits when a
// required element or attribute is absent.
func errMissing(what string) error {
	return fmt.Errorf("settings file: missing %s", what)
}

// LoadProperties ports PropertyStorage.LoadProperties: read (and for .exml
// decrypt) fileName, parse it, and assign every <Property Id Value> of
// Settings/Properties to the matching dictionary entry (ICUProperty.Value =
// value), skipping unknown ids and s_forbiddenProperties. It returns the
// number of properties that went from unchanged to changed.
//
// Checks performed, as in the C#: no <Settings> root -> 0, nil;
// XMLVersion != "1.0" -> ErrIncorrectVersion; when
// ParseDeviceModel(<Model>, 1, false) != dev.ModelType the user is asked
// ModelMismatchQuestion via confirm (nil confirm counts as "No") and a
// refusal returns 0, nil. NumberOfSockets and Version are not checked.
// Read-only properties are not skipped on load.
//
// Errors after some assignments (a <Property> without Id/Value, an id that
// overflows Convert.ToUInt16/ToByte) return 0 and an error while the earlier
// assignments stay applied, exactly like the C# exception path.
//
// After a successful load the caller shows SettingsLoadedMessage /
// SettingsLoadedDetail(n) when n > 0; the writes for "Save" are
// dict.ChangedProperties() (or dict.StoreChangedProperties).
func LoadProperties(fileName string, dev DeviceInfo, dict *Dictionary, confirm ConfirmFunc) (int, error) {
	text, err := ReadSettingsText(fileName)
	if err != nil {
		return 0, err
	}
	return LoadPropertiesText(text, dev, dict, confirm)
}

// LoadPropertiesText is LoadProperties on already read/decrypted text
// (XDocument.Parse onwards).
func LoadPropertiesText(text string, dev DeviceInfo, dict *Dictionary, confirm ConfirmFunc) (int, error) {
	doc, err := xDocumentParse(text)
	if err != nil {
		return 0, err
	}
	settings := doc.Element("Settings")
	if settings == nil {
		return 0, nil
	}
	xmlVersion := settings.Element("XMLVersion")
	if xmlVersion == nil {
		return 0, errMissing("<XMLVersion>")
	}
	if xmlVersion.Value() != "1.0" {
		return 0, ErrIncorrectVersion
	}
	modelEl := settings.Element("Model")
	if modelEl == nil {
		return 0, errMissing("<Model>")
	}
	fileModel := ParseDeviceModel(modelEl.Value(), 1, false)
	if fileModel != dev.ModelType {
		question, detail := ModelMismatchQuestion(fileModel, dev.Model)
		if confirm == nil || !confirm(question, detail) {
			return 0, nil
		}
	}
	propsEl := settings.Element("Properties")
	if propsEl == nil {
		return 0, errMissing("<Properties>")
	}
	num := 0
	for _, item := range propsEl.Elements("Property") {
		id, ok := item.Attribute("Id")
		if !ok {
			return 0, errMissing("Property/@Id")
		}
		if dict == nil {
			continue
		}
		key, found, err := dict.LookupIDSub(id)
		if err != nil {
			return 0, err
		}
		if !found || IsForbidden(key.ID) {
			continue
		}
		e := dict.find(key.ID, key.Sub)
		wasChanged := e.changed
		value, ok := item.Attribute("Value")
		if !ok {
			return 0, errMissing("Property/@Value")
		}
		e.setText(value)
		if !wasChanged && e.changed {
			num++
		}
	}
	return num, nil
}
