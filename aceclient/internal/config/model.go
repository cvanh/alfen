package config

import (
	"fmt"
	"regexp"
	"strings"
)

var newlineRE = regexp.MustCompile("\\r\\n?|\\n")

// validString ports ICUBaseObject.ValidString
// (ACESettings/ICUSettings/ICUBaseObject.cs): wraps the value in double quotes
// and turns CR/LF line breaks into "<br>". Quotes and backslashes are NOT
// escaped - the C# writer emits them raw, and so does this port.
func validString(value string) string {
	return "\"" + newlineRE.ReplaceAllString(value, "<br>") + "\""
}

// fromBR undoes validString's line-break encoding the way the readers do
// (.Trim().Replace("<br>", "\n")).
func fromBR(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "<br>", "\n")
}

// Feature ports ICUSettings.ICUFeature (ACESettings/ICUSettings/ICUFeature.cs):
// a named, rights-controlled item (page, property, backoffice, other) with its
// default rights.
type Feature struct {
	Name    string
	ID      string
	Type    FeatureType
	Default Rights
	Comment string
}

// newFeature ports ICUFeature(string name, string id, ICUFeatureType, ICURights, string comment).
func newFeature(name, id string, t FeatureType, def Rights, comment string) *Feature {
	return &Feature{
		Name:    strings.TrimSpace(name),
		ID:      strings.TrimSpace(id),
		Type:    t,
		Default: def,
		Comment: strings.TrimSpace(comment),
	}
}

// NewPageFeature ports ICUFeature.CreatePageFeature: ID "PAGE_"+id, type Page.
// (C# default for def is RightsFull.)
func NewPageFeature(id, name string, def Rights) *Feature {
	return newFeature(name, "PAGE_"+id, FeatureTypePage, def, "")
}

// NewNormalFeature ports ICUFeature.CreateFeature: ID "FEATURE_"+id, type Normal.
// (C# default for def is RightsFull.)
func NewNormalFeature(id, name string, def Rights) *Feature {
	return newFeature(name, "FEATURE_"+id, FeatureTypeNormal, def, "")
}

// NewIDFeature ports ICUFeature.CreateIDFeature: ID "ID_{usId:X4}", type Property.
// (C# default for def is RightsFull.)
func NewIDFeature(id uint16, name string, def Rights) *Feature {
	return newFeature(name, PropertyFeatureID(id), FeatureTypeProperty, def, "")
}

// NewBackOfficeFeature ports ICUFeature.CreateBackOfficeFeature: ID
// BackOfficeFeatureID(title), type Backoffice. (C# default for def is RightsReadOnly.)
func NewBackOfficeFeature(title, name string, def Rights) *Feature {
	return newFeature(name, BackOfficeFeatureID(title), FeatureTypeBackoffice, def, "")
}

// PropertyFeatureID is the feature-right ID of an object-dictionary property:
// $"ID_{usId:X4}" (ICUFeature.CreateIDFeature, PanelBase.AddPropertyBase(ushort, ...)).
func PropertyFeatureID(id uint16) string {
	return fmt.Sprintf("ID_%04X", id)
}

// BackOfficeFeatureID is $"BO_{title.Replace(' ', '_').Trim().ToUpperInvariant()}"
// (ICUFeature.CreateBackOfficeFeature, ICUConfig.AddDefaultGroups).
func BackOfficeFeatureID(title string) string {
	return "BO_" + strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(title, " ", "_")))
}

// SetName ports the ICUFeature.Name setter (ACESettings/ICUSettings/ICUFeature.cs:22-33): value.Trim().
func (f *Feature) SetName(v string) { f.Name = strings.TrimSpace(v) }

// SetID ports the ICUFeature.ID setter (ACESettings/ICUSettings/ICUFeature.cs:36-47):
// value.Trim().ToUpperInvariant().
func (f *Feature) SetID(v string) { f.ID = strings.ToUpper(strings.TrimSpace(v)) }

// SetComment ports the ICUFeature.Comment setter (ACESettings/ICUSettings/ICUFeature.cs:78-89): value.Trim().
func (f *Feature) SetComment(v string) { f.Comment = strings.TrimSpace(v) }

// JSON ports ICUFeature.Json (byte-exact, including the leading "\n\t").
func (f *Feature) JSON() string {
	return fmt.Sprintf("\n\t{\"Name\":%s,\"ID\":%s,\"Type\":%s,\"Default\":%s,\"Comment\":%s}",
		validString(f.Name), validString(f.ID), validString(f.Type.String()),
		validString(f.Default.String()), validString(f.Comment))
}

// FeatureRight ports ICUSettings.ICUFeatureRight
// (ACESettings/ICUSettings/ICUFeatureRight.cs): a group's rights on one feature.
type FeatureRight struct {
	Feature *Feature
	Rights  Rights
}

// Group ports ICUSettings.ICUGroup (ACESettings/ICUSettings/ICUGroup.cs): a
// rights profile plus the HTTP credentials its members use to log in to a
// charger (MainWindow.OnLoginRequest).
type Group struct {
	Name          string
	Comment       string
	HTTPUser      string
	HTTPPassword  string
	Features      []*FeatureRight // one entry per config feature, in feature order
	NumberOfUsers int             // filled by ReadInstallerSettings
}

// NewGroup ports ICUGroup(IList<ICUFeature>, string name, string comment = "",
// string httpuser = "", string httppassword = ""). The name is stored as given
// (not trimmed) and is set before InitializeFeatures, so a group named "admin"
// (case-insensitive) starts with RightsFull on every feature; any other name
// starts with each feature's Default.
func NewGroup(features []*Feature, name, comment, httpUser, httpPassword string) *Group {
	g := &Group{Name: name, Comment: comment, HTTPUser: httpUser, HTTPPassword: httpPassword}
	g.initializeFeatures(features, strings.ToLower(name) == "admin")
	return g
}

// initializeFeatures ports ICUGroup.InitializeFeatures. admin reflects the
// Name value at the time the C# constructor calls it (null in the JSON
// constructor, so JSON-loaded groups - even "Admin" - start from defaults).
func (g *Group) initializeFeatures(features []*Feature, admin bool) {
	g.Features = make([]*FeatureRight, 0, len(features))
	for _, f := range features {
		r := f.Default
		if admin {
			r = RightsFull
		}
		g.Features = append(g.Features, &FeatureRight{Feature: f, Rights: r})
	}
}

// SetName ports the ICUGroup.Name setter (ACESettings/ICUSettings/ICUGroup.cs:26-37): value.Trim().
func (g *Group) SetName(v string) { g.Name = strings.TrimSpace(v) }

// SetComment ports the ICUGroup.Comment setter (ACESettings/ICUSettings/ICUGroup.cs:39-50): value.Trim().
func (g *Group) SetComment(v string) { g.Comment = strings.TrimSpace(v) }

// SetHTTPUser ports the ICUGroup.HTTPUser setter (ACESettings/ICUSettings/ICUGroup.cs:52-63): value.Trim().
func (g *Group) SetHTTPUser(v string) { g.HTTPUser = strings.TrimSpace(v) }

// SetHTTPPassword ports the ICUGroup.HTTPPassword setter (ACESettings/ICUSettings/ICUGroup.cs:65-76): value.Trim().
func (g *Group) SetHTTPPassword(v string) { g.HTTPPassword = strings.TrimSpace(v) }

// GetRights ports ICUGroup.GetRights: the rights on the feature with exactly
// this ID, or RightsFull when the group has no such feature.
func (g *Group) GetRights(id string) Rights {
	for _, fr := range g.Features {
		if fr.Feature.ID == id {
			return fr.Rights
		}
	}
	return RightsFull
}

// JSON ports ICUGroup.Json(bool IncludeCredentials) (byte-exact). Only rights
// that differ from the feature default are written.
func (g *Group) JSON(includeCredentials bool) string {
	var list []string
	for _, fr := range g.Features {
		if fr.Rights != fr.Feature.Default {
			list = append(list, fmt.Sprintf("\n\t\t{\"ID\":%s,\"Rights\":%s}",
				validString(fr.Feature.ID), validString(fr.Rights.String())))
		}
	}
	user, pass := "", ""
	if includeCredentials {
		user, pass = g.HTTPUser, g.HTTPPassword
	}
	return fmt.Sprintf("\n\t{\"Name\":%s,\"Comment\":%s,\"HTTPUser\":%s,\"HTTPPassword\":%s,\"FeatureRights\":[%s]}",
		validString(g.Name), validString(g.Comment), validString(user), validString(pass),
		strings.Join(list, ","))
}

// User ports ICUSettings.ICUUser (ACESettings/ICUSettings/ICUUser.cs). In
// hashed configs (EncryptRijndaelHashed) User is Base64(SHA256(lower(name))),
// Password is "NN:SALTHEX:Base64(hash)", Fullname is empty, and Comment /
// Company hold the encrypted "HTTPUser:HTTPPassword" and ISAH data.
type User struct {
	User     string
	Password string
	Fullname string
	Comment  string
	Company  string
	Group    *Group // nil when no group resolved
}

// SetUser ports the ICUUser.User setter (ACESettings/ICUSettings/ICUUser.cs:29-40): value.Trim().
func (u *User) SetUser(v string) { u.User = strings.TrimSpace(v) }

// SetPassword ports the ICUUser.Password setter (ACESettings/ICUSettings/ICUUser.cs:42-53): value.Trim().
func (u *User) SetPassword(v string) { u.Password = strings.TrimSpace(v) }

// SetFullname ports the ICUUser.Fullname setter (ACESettings/ICUSettings/ICUUser.cs:68-79): value.Trim().
func (u *User) SetFullname(v string) { u.Fullname = strings.TrimSpace(v) }

// SetComment ports the ICUUser.Comment setter (ACESettings/ICUSettings/ICUUser.cs:81-92): value.Trim().
func (u *User) SetComment(v string) { u.Comment = strings.TrimSpace(v) }

// SetCompany ports the ICUUser.Company setter (ACESettings/ICUSettings/ICUUser.cs:94-105): value.Trim().
func (u *User) SetCompany(v string) { u.Company = strings.TrimSpace(v) }

// GetRights ports ICUUser.GetRights: the group's rights, or RightsReadOnly when
// the user has no group.
func (u *User) GetRights(id string) Rights {
	if u.Group != nil {
		return u.Group.GetRights(id)
	}
	return RightsReadOnly
}

// JSON ports ICUUser.Json (byte-exact).
func (u *User) JSON() string {
	group := ""
	if u.Group != nil {
		group = u.Group.Name
	}
	return fmt.Sprintf("\n\t{\"User\":%s,\"Password\":%s,\"Group\":%s,\"Fullname\":%s,\"Comment\":%s,\"Company\":%s}",
		validString(u.User), validString(u.Password), validString(group),
		validString(u.Fullname), validString(u.Comment), validString(u.Company))
}

// CSVLine ports ICUUser.CSVLine.
func (u *User) CSVLine() string {
	return fmt.Sprintf("\"%s\", \"%s\", \"%s\"", u.Fullname, u.User, u.Password)
}

// Firmware ports ICUSettings.ICUFirmware (ACESettings/ICUSettings/ICUFirmware.cs):
// a firmware file known to the installer (DlgUpload looks entries up by file name).
type Firmware struct {
	Filename string
	Version  string
	Comments string
	Date     string
	OnFTP    bool // m_fFoundOnFtp; runtime only, never serialised
}

// NewFirmware ports the parameterless ICUFirmware() constructor.
func NewFirmware() *Firmware {
	return &Firmware{Version: "0.0.0"}
}

// SetFilename ports the ICUFirmware.Filename setter (ACESettings/ICUSettings/ICUFirmware.cs:19-30): value.Trim().
// The Comments and Date setters store the value unchanged, so those fields are
// assigned directly.
func (f *Firmware) SetFilename(v string) { f.Filename = strings.TrimSpace(v) }

// SetVersion ports the ICUFirmware.Version setter (ACESettings/ICUSettings/ICUFirmware.cs:32-43): value.Trim().
func (f *Firmware) SetVersion(v string) { f.Version = strings.TrimSpace(v) }

// JSON ports ICUFirmware.Json (byte-exact).
func (f *Firmware) JSON() string {
	return fmt.Sprintf("\n\t{\"Filename\":%s,\"Version\":%s,\"Date\":%s,\"Comments\":%s}",
		validString(f.Filename), validString(f.Version), validString(f.Date), validString(f.Comments))
}
