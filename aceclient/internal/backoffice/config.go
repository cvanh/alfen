package backoffice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// decodeArray decodes a JSON array the way JavaScriptSerializer.DeserializeObject
// does for these files: objects -> map[string]any, numbers kept verbatim.
func decodeArray(raw json.RawMessage, what string) ([]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	if v == nil { // foreach over null -> NullReferenceException
		return nil, fmt.Errorf("%s: null", what)
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: %T is not an array", what, v)
	}
	return arr, nil
}

// ParseBackoffices ports the `foreach (dynamic item in val["Backoffices"])
// new ICUBackOffice(item)` loop of ICUConfig.ReadInstallerSettings. raw is the
// value of the "Backoffices" key; an absent key (empty raw) is an error, as
// the C# indexer throws and the whole config load fails.
func ParseBackoffices(raw json.RawMessage) ([]*BackOffice, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("Backoffices: key not present")
	}
	arr, err := decodeArray(raw, "Backoffices")
	if err != nil {
		return nil, err
	}
	out := make([]*BackOffice, 0, len(arr))
	for i, it := range arr {
		obj, ok := it.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Backoffices[%d]: %T is not an object", i, it)
		}
		b := newBackOffice(false) // ICUBackOffice(dynamic): base() -> not dirty
		if err := b.loadFromMap(obj); err != nil {
			return nil, fmt.Errorf("Backoffices[%d]: %w", i, err)
		}
		out = append(out, b)
	}
	return out, nil
}

// ParsePMBackoffices ports the optional `if (ContainsKey("PMBackOffices"))
// foreach ... new ICUPMBackOffice(item)` loop of ReadInstallerSettings. An
// absent key (empty raw) yields no entries.
func ParsePMBackoffices(raw json.RawMessage) ([]*PMBackOffice, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	arr, err := decodeArray(raw, "PMBackOffices")
	if err != nil {
		return nil, err
	}
	out := make([]*PMBackOffice, 0, len(arr))
	for i, it := range arr {
		obj, ok := it.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("PMBackOffices[%d]: %T is not an object", i, it)
		}
		p := newPMBackOffice(false)
		if err := p.loadFromMap(obj); err != nil {
			return nil, fmt.Errorf("PMBackOffices[%d]: %w", i, err)
		}
		out = append(out, p)
	}
	return out, nil
}

// BackofficesJSON ports `string.Join(",", BackOffices.Select(a => a.Json))`
// (text5 of ICUConfig.WriteInstallerSettings).
func BackofficesJSON(bos []*BackOffice) string {
	parts := make([]string, len(bos))
	for i, b := range bos {
		parts[i] = b.JSON()
	}
	return strings.Join(parts, ",")
}

// PMBackofficesJSON ports `string.Join(",", PMBackOffices.Select(a => a.Json))`
// (text6 of ICUConfig.WriteInstallerSettings).
func PMBackofficesJSON(pms []*PMBackOffice) string {
	parts := make([]string, len(pms))
	for i, p := range pms {
		parts[i] = p.JSON()
	}
	return strings.Join(parts, ",")
}

// MarshalBackoffices returns the exact bytes WriteInstallerSettings emits as
// the value of "Backoffices": "[" + text5 + "\n]". Pass nil for the
// addBackoffices=false case ("[\n]").
func MarshalBackoffices(bos []*BackOffice) json.RawMessage {
	return json.RawMessage("[" + BackofficesJSON(bos) + "\n]")
}

// MarshalPMBackoffices returns the exact bytes WriteInstallerSettings emits as
// the value of "PMBackOffices": "[" + text6 + "\n]".
func MarshalPMBackoffices(pms []*PMBackOffice) json.RawMessage {
	return json.RawMessage("[" + PMBackofficesJSON(pms) + "\n]")
}

// CopyBackOffice ports ICUConfig.AddBackoffice(copyFrom) minus the collection
// bookkeeping: a new (dirty) backoffice holding all of copyFrom's properties
// with "(copy)" appended to the four titles. The caller appends it to its list
// and marks the collection dirty.
func CopyBackOffice(copyFrom *BackOffice) *BackOffice {
	b := NewBackOffice()
	b.CopyFrom(copyFrom, true)
	b.SetTitle(b.Title() + "(copy)")
	b.SetTitleNL(b.TitleNL() + "(copy)")
	b.SetTitleDE(b.TitleDE() + "(copy)")
	b.SetTitleFR(b.TitleFR() + "(copy)")
	return b
}

// CopyPMBackOffice ports ICUConfig.AddPMBackoffice(copyFrom) minus the
// collection bookkeeping.
func CopyPMBackOffice(copyFrom *PMBackOffice) *PMBackOffice {
	p := NewPMBackOffice()
	p.CopyFrom(copyFrom, true)
	p.SetTitle(p.Title() + "(copy)")
	return p
}

func findByTitle(bos []*BackOffice, title string) *BackOffice {
	for _, b := range bos {
		if b.Title() == title {
			return b
		}
	}
	return nil
}

// DerivePMBackOffices ports ICUConfig.UpdatePMBackOffices: it rebuilds the
// part-management list from the backoffice titles ("<name> - production
// gprs|wired|auto", "<name> - sandbox gprs|wired|auto"), merging a GPRS and a
// wired variant into one PMBackOffice. Like the C# (Substring with
// LastIndexOf('-') == -1) it fails on any title without a '-'.
func DerivePMBackOffices(bos []*BackOffice) ([]*PMBackOffice, error) {
	var out []*PMBackOffice
	for _, bo := range bos {
		title := bo.Title()
		text := csTrim(strings.ToLower(title))
		idx := strings.LastIndexByte(title, '-')
		if idx < 0 {
			return nil, fmt.Errorf("UpdatePMBackOffices: title %q has no '-' (ArgumentOutOfRangeException)", title)
		}
		text2 := csTrim(title[:idx])
		switch {
		case strings.HasSuffix(text, "production auto") || strings.HasSuffix(text, "productionauto"):
			out = addPMBackOffice(out, text2, bo, bo)
		case strings.HasSuffix(text, "production gprs") || strings.HasSuffix(text, "productiongprs"):
			if findByTitle(bos, strings.ReplaceAll(title, "gprs", "auto")) == nil {
				wired := findByTitle(bos, strings.ReplaceAll(title, "gprs", "wired"))
				out = addPMBackOffice(out, text2, bo, wired)
			}
		case strings.HasSuffix(text, "production wired") || strings.HasSuffix(text, "productionwired"):
			if findByTitle(bos, strings.ReplaceAll(title, "wired", "auto")) == nil &&
				findByTitle(bos, strings.ReplaceAll(title, "wired", "gprs")) == nil {
				out = addPMBackOffice(out, text2, nil, bo)
			}
		case strings.HasSuffix(text, "sandbox auto"):
			out = addPMBackOffice(out, text2+" sandbox", bo, bo)
		case strings.HasSuffix(text, "sandbox gprs"):
			if findByTitle(bos, strings.ReplaceAll(title, "gprs", "auto")) == nil {
				wired := findByTitle(bos, strings.ReplaceAll(title, "gprs", "wired"))
				out = addPMBackOffice(out, text2+" sandbox", bo, wired)
			}
		case strings.HasSuffix(text, "sandbox wired"):
			if findByTitle(bos, strings.ReplaceAll(title, "wired", "auto")) == nil &&
				findByTitle(bos, strings.ReplaceAll(title, "wired", "gprs")) == nil {
				out = addPMBackOffice(out, text2+" sandbox", nil, bo)
			}
		}
	}
	return out, nil
}

// addPMBackOffice ports ICUConfig.AddPMBackOffice (private). The assignment
// order is the C# object-initializer order; assignments to properties the PM
// model has no entry for are no-ops, as in the C#.
func addPMBackOffice(out []*PMBackOffice, title string, gprs, wired *BackOffice) []*PMBackOffice {
	lan, gprsOn := true, true
	if wired == nil {
		lan = false
		wired = gprs
	}
	if gprs == nil {
		gprsOn = false
		gprs = wired
	}
	if wired == nil && gprs == nil {
		return out
	}
	p := NewPMBackOffice()
	p.SetTitle(title)
	p.SetAPNName(gprs.APNName())
	p.SetAPNUser(gprs.APNUser())
	p.SetAPNPassword(gprs.APNPassword())
	p.SetBackOfficeURLwired_Domain(wired.BackOfficeURLwired_Domain())
	p.SetBackOfficeURLwired_Path(wired.BackOfficeURLwired_Path())
	p.SetBackOfficeURL_Domain(gprs.BackOfficeURL_Domain())
	p.SetBackOfficeURL_Path(gprs.BackOfficeURL_Path())
	p.SetCentralMeterValueAlignment(gprs.CentralMeterValueAlignment())
	p.SetDNS1_1(gprs.DNS1_1())
	p.SetDNS1_2(gprs.DNS1_2())
	p.SetDNS2_1(wired.DNS2_1())
	p.SetDNS2_2(wired.DNS2_2())
	p.SetEVDisconnectAction(gprs.EVDisconnectAction())
	p.SetEVDisconnectTimeout(gprs.EVDisconnectTimeout())
	p.SetIntensityAuto(gprs.IntensityAuto())
	p.SetIntensityIntensity(gprs.IntensityIntensity())
	p.SetLanguage(gprs.Language())
	p.SetOCPP15SmartChargingType(gprs.OCPP15SmartChargingType())
	p.SetOfflineNFCAuthorization(gprs.OfflineNFCAuthorization())
	p.SetOnlineNFCAuthorization(gprs.OnlineNFCAuthorization())
	p.SetPingPongInterval(gprs.PingPongInterval())
	p.SetProtocolName(gprs.ProtocolName())
	p.SetProtocolVersion(gprs.ProtocolVersion())
	p.SetSendStationStatus(gprs.SendStationStatus())
	p.SetSimPin(gprs.SimPin())
	p.SetTimezoneMinutes(gprs.TimezoneMinutes())
	p.SetTransactionMessageAttempts(gprs.TransactionMessageAttempts())
	p.SetTransactionMessageRetryInterval(gprs.TransactionMessageRetryInterval())
	p.SetGPRSEnabled(gprsOn)
	p.SetLANEnabled(lan)
	return append(out, p)
}

// xmlAttrEscaper/xmlTextEscaper mirror XmlWriter's escaping of attribute
// values and element text.
var (
	xmlAttrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;",
		"\t", "&#x9;", "\n", "&#xA;", "\r", "&#xD;")
	xmlTextEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
)

// PMBackOfficeSettingsXML ports the per-backoffice body of
// ICUConfig.WritePMBackOfficeSettingsXML: the file name
// (Title.Trim().Replace(' ', '-') + ".xml") and the file content — the
// indented XDocument with its XML declaration line removed, rewritten by
// File.WriteAllLines (UTF-8 without BOM, CRLF after every line).
func PMBackOfficeSettingsXML(pm *PMBackOffice) (fileName string, content []byte) {
	lines := []string{"<Config>", "  <Setting>"}
	var objs []string
	for _, q := range pm.Properties {
		if e, ok := q.Element(); ok {
			objs = append(objs, fmt.Sprintf(`      <Object Id="%s" Value="%s">%s</Object>`,
				xmlAttrEscaper.Replace(e.ID), xmlAttrEscaper.Replace(e.Value), xmlTextEscaper.Replace(e.Text)))
		}
	}
	if len(objs) == 0 {
		lines = append(lines, `    <Product Model="NG9xx" Device="NG9xx" />`)
	} else {
		lines = append(lines, `    <Product Model="NG9xx" Device="NG9xx">`)
		lines = append(lines, objs...)
		lines = append(lines, "    </Product>")
	}
	lines = append(lines, "  </Setting>", "</Config>")
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString("\r\n")
	}
	return strings.ReplaceAll(csTrim(pm.Title()), " ", "-") + ".xml", []byte(b.String())
}

// WritePMBackOfficeSettingsXML ports ICUConfig.WritePMBackOfficeSettingsXML:
// one XML file per part-management backoffice in directory. Like the C# it
// stops at the first failure.
func WritePMBackOfficeSettingsXML(directory string, pms []*PMBackOffice) error {
	for _, pm := range pms {
		name, data := PMBackOfficeSettingsXML(pm)
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Feature is one backoffice user-rights feature (ICUFeature of type
// ICUFeatureType.Backoffice, default rights ReadOnly).
type Feature struct {
	ID   string // "BO_<TITLE>"
	Name string // "BackOffice '<Title>'"
}

// FeatureID ports the ID built by ICUFeature.CreateBackOfficeFeature and
// ICUConfig.AddDefaultGroups: $"BO_{title.Replace(' ', '_').Trim().ToUpperInvariant()}".
func FeatureID(title string) string {
	return "BO_" + strings.ToUpper(csTrim(strings.ReplaceAll(title, " ", "_")))
}

// FeatureName ports the name ICUConfig.AddBackofficesToFeatures passes:
// $"BackOffice '{Title}'".
func FeatureName(title string) string { return "BackOffice '" + title + "'" }

// Features ports ICUConfig.AddBackofficesToFeatures: one feature per
// backoffice with a non-empty title, in list order, skipping IDs already added
// (AddFeature dedupes by ID; the caller must also dedupe against its other
// features).
func Features(bos []*BackOffice) []Feature {
	var out []Feature
	seen := map[string]bool{}
	for _, b := range bos {
		t := b.Title()
		if t == "" {
			continue
		}
		id := FeatureID(t)
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, Feature{ID: id, Name: FeatureName(t)})
	}
	return out
}

// CompanyFeatureIDs ports the list2 query of ICUConfig.AddDefaultGroups: the
// backoffice feature IDs a generated "Extern_<company>" group gets ReadOnly
// rights on (backoffices whose Groups contains company.ToUpperInvariant(),
// ordinal substring match); every other backoffice feature gets None.
func CompanyFeatureIDs(bos []*BackOffice, company string) []string {
	var out []string
	up := strings.ToUpper(company)
	for _, b := range bos {
		if strings.Contains(b.Groups(), up) {
			out = append(out, FeatureID(b.Title()))
		}
	}
	return out
}
