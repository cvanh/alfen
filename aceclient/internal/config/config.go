package config

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// FileID is ICUConfig.s_fileID, the mandatory "Type" of a config file.
const FileID = "ICUConfigFile"

// maxJSONLength is the JavaScriptSerializer.MaxJsonLength ReadInstallerSettings sets (50 MiB).
const maxJSONLength = 52428800

// ErrIncorrectFileType is ReadInstallerSettings' ArgumentException("Incorrect file type!").
var ErrIncorrectFileType = errors.New("config: Incorrect file type!")

// Config ports ICUSettings.ICUConfig (ACESettings/ICUSettings/ICUConfig.cs):
// the installer configuration held in InstallerConfig*.dat.
type Config struct {
	Version  string
	Date     string // "Date" as read; Save writes the current time instead
	Features []*Feature
	Groups   []*Group
	Users    []*User
	// BackOffices / PMBackOffices are the "Backoffices" / "PMBackOffices"
	// array elements verbatim; internal/backoffice owns their model.
	BackOffices   []json.RawMessage
	PMBackOffices []json.RawMessage
	Firmwares     []*Firmware
}

// NewConfig returns an empty configuration (new ICUConfig()).
func NewConfig() *Config { return &Config{} }

// Load ports ICUConfig.ReadInstallerSettings(filename, encType) on a fresh
// ICUConfig: read the file (File.ReadAllText), undo the envelope for encType
// and parse the JSON. EncryptRijndael and EncryptRijndaelHashed read
// identically. On error the C# returns false (and logs); this returns the
// error and no config. Use (*Config).ReadInstallerSettings to keep the
// partially loaded state, as the C# does.
func Load(filename string, enc EncryptionType) (*Config, error) {
	c := NewConfig()
	if err := c.ReadInstallerSettings(filename, enc); err != nil {
		return nil, err
	}
	return c, nil
}

// ReadInstallerSettings ports ICUConfig.ReadInstallerSettings(filename,
// encType) (ACESettings/ICUSettings/ICUConfig.cs:290-400) on an existing
// instance. Like the C#, it changes c step by step and does not roll back on
// failure:
//   - Errors while reading, decoding or checking "Type" leave c untouched.
//   - Version and then Date are assigned once read.
//   - All six collections are then replaced by empty ones and refilled in
//     file order: Backoffices, PMBackOffices, Features (then the manual and
//     backoffice features), Groups (or the default groups), Users and
//     Firmwares. Finally NumberOfUsers is counted.
//
// An error part-way through therefore leaves everything read before it in c
// (for example the users when a firmware entry is bad). AppProperties keeps
// that partial state in its static ICUConfig, see InitializeSettings.
func (c *Config) ReadInstallerSettings(filename string, enc EncryptionType) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return c.readText(readAllText(b), enc)
}

// readAllText mirrors File.ReadAllText's BOM detection (UTF-8 / UTF-16 LE / BE;
// default UTF-8).
func readAllText(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return string(b[3:])
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}), bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		var order binary.ByteOrder = binary.LittleEndian
		if b[0] == 0xFE {
			order = binary.BigEndian
		}
		b = b[2:]
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = order.Uint16(b[2*i:])
		}
		return string(utf16.Decode(u))
	}
	return string(b)
}

// Parse ports the decoding half of ReadInstallerSettings for file text that
// has already been read: Base64 for EncryptBase64, EncryptDecrypt.Decrypt(text,
// ConfigPassPhrase) for the Rijndael types, nothing for EncryptNone. On error
// it returns no config (see ReadInstallerSettings for the partial state).
func Parse(text string, enc EncryptionType) (*Config, error) {
	c := NewConfig()
	if err := c.readText(text, enc); err != nil {
		return nil, err
	}
	return c, nil
}

// readText is ReadInstallerSettings after File.ReadAllText.
func (c *Config) readText(text string, enc EncryptionType) error {
	var plain []byte
	switch enc {
	case EncryptBase64:
		b, err := fromBase64(text)
		if err != nil {
			return err
		}
		plain = b
	case EncryptRijndael, EncryptRijndaelHashed:
		s, err := Decrypt(text, ConfigPassPhrase)
		if err != nil {
			return err
		}
		plain = []byte(s)
	default:
		plain = []byte(text)
	}
	return c.readJSON(plain)
}

// ParseJSON ports the JSON half of ReadInstallerSettings on a fresh config.
// On error it returns no config (see ReadInstallerSettings for the partial
// state).
func ParseJSON(plain []byte) (*Config, error) {
	c := NewConfig()
	if err := c.readJSON(plain); err != nil {
		return nil, err
	}
	return c, nil
}

// readJSON ports the JSON half of ReadInstallerSettings, in the C# order and
// with the C# partial-state behaviour (see ReadInstallerSettings): Type check,
// Version, Date, reset of the collections, Backoffices (required; each one
// must survive new ICUBackOffice(item)), PMBackOffices (optional; new
// ICUPMBackOffice(item)), Features (optional), AddManualFeatures,
// AddBackofficesToFeatures, Groups (optional; AddDefaultGroups when none),
// Users (required), Firmwares (required), then the per-group NumberOfUsers
// count.
func (c *Config) readJSON(plain []byte) error {
	if utf8.RuneCount(plain) > maxJSONLength {
		return errors.New("config: JSON length exceeds MaxJsonLength")
	}
	var val jsonObject
	if err := json.Unmarshal(plain, &val); err != nil {
		// JavaScriptSerializer tolerates raw control characters in strings.
		if err2 := json.Unmarshal(escapeControlChars(plain), &val); err2 != nil {
			return fmt.Errorf("config: invalid JSON: %w", err)
		}
	}
	if val == nil {
		return errors.New("config: invalid JSON: not an object")
	}
	typ, err := topScalar(val, "Type")
	if err != nil {
		return err
	}
	if typ != FileID {
		return ErrIncorrectFileType
	}
	version, err := topScalar(val, "Version")
	if err != nil {
		return err
	}
	c.Version = version
	date, err := topScalar(val, "Date")
	if err != nil {
		return err
	}
	c.Date = date
	c.Features, c.Groups, c.Users = nil, nil, nil
	c.BackOffices, c.PMBackOffices, c.Firmwares = nil, nil, nil

	bos, err := val.array("Backoffices", "config")
	if err != nil {
		return err
	}
	for i, raw := range bos {
		if _, err := backOfficeHeader(raw); err != nil {
			return fmt.Errorf("config: Backoffices[%d]: %w", i, err)
		}
		c.BackOffices = append(c.BackOffices, raw)
	}
	if val.has("PMBackOffices") {
		pms, err := val.array("PMBackOffices", "config")
		if err != nil {
			return err
		}
		for i, raw := range pms {
			if err := checkPMBackOffice(raw); err != nil {
				return fmt.Errorf("config: PMBackOffices[%d]: %w", i, err)
			}
			c.PMBackOffices = append(c.PMBackOffices, raw)
		}
	}
	if val.has("Features") {
		feats, err := val.array("Features", "config")
		if err != nil {
			return err
		}
		for i, raw := range feats {
			f, err := featureFromJSON(raw, fmt.Sprintf("Features[%d]", i))
			if err != nil {
				return err
			}
			c.Features = append(c.Features, f)
		}
	}
	c.addManualFeatures()
	c.addBackofficesToFeatures()

	if val.has("Groups") {
		groups, err := val.array("Groups", "config")
		if err != nil {
			return err
		}
		for i, raw := range groups {
			g, err := groupFromJSON(c.Features, raw, fmt.Sprintf("Groups[%d]", i))
			if err != nil {
				return err
			}
			c.Groups = append(c.Groups, g)
		}
	}
	users, err := val.array("Users", "config")
	if err != nil {
		return err
	}
	if len(c.Groups) == 0 {
		var companies []string
		for i, raw := range users {
			o, err := decodeObject(raw, fmt.Sprintf("Users[%d]", i))
			if err != nil {
				return err
			}
			if o.has("Company") {
				s, err := o.str("Company", fmt.Sprintf("Users[%d]", i))
				if err != nil {
					return err
				}
				companies = append(companies, strings.TrimSpace(s))
			}
		}
		if err := c.addDefaultGroups(companies); err != nil {
			return err
		}
	}
	for i, raw := range users {
		u, err := userFromJSON(c.Groups, raw, fmt.Sprintf("Users[%d]", i))
		if err != nil {
			return err
		}
		c.Users = append(c.Users, u)
	}
	fws, err := val.array("Firmwares", "config")
	if err != nil {
		return err
	}
	for i, raw := range fws {
		f, err := firmwareFromJSON(raw, fmt.Sprintf("Firmwares[%d]", i))
		if err != nil {
			return err
		}
		c.Firmwares = append(c.Firmwares, f)
	}
	for _, u := range c.Users {
		if u.Group != nil {
			u.Group.NumberOfUsers++
		}
	}
	return nil
}

func topScalar(val jsonObject, key string) (string, error) {
	raw, ok := val[key]
	if !ok {
		return "", fmt.Errorf("config: the given key %q was not present in the dictionary", key)
	}
	s, err := scalarString(raw, false)
	if err != nil {
		return "", fmt.Errorf("config: %q: %w", key, err)
	}
	return s, nil
}

// featureFromJSON ports ICUFeature(dynamic obj). Type defaults to Normal and
// Default to Full (field initialisers) when absent; ID is trimmed but not
// upper-cased.
func featureFromJSON(raw json.RawMessage, what string) (*Feature, error) {
	o, err := decodeObject(raw, what)
	if err != nil {
		return nil, err
	}
	f := &Feature{Type: FeatureTypeNormal, Default: RightsFull}
	if f.Name, err = o.str("Name", what); err != nil {
		return nil, err
	}
	f.Name = strings.TrimSpace(f.Name)
	if f.ID, err = o.str("ID", what); err != nil {
		return nil, err
	}
	f.ID = strings.TrimSpace(f.ID)
	if o.has("Type") {
		s, err := o.str("Type", what)
		if err != nil {
			return nil, err
		}
		if f.Type, err = ParseFeatureType(strings.TrimSpace(s)); err != nil {
			return nil, fmt.Errorf("%s: %w", what, err)
		}
	}
	if o.has("Default") {
		s, err := o.str("Default", what)
		if err != nil {
			return nil, err
		}
		if f.Default, err = ParseRights(strings.TrimSpace(s)); err != nil {
			return nil, fmt.Errorf("%s: %w", what, err)
		}
	}
	s, err := o.str("Comment", what)
	if err != nil {
		return nil, err
	}
	f.Comment = fromBR(s)
	return f, nil
}

// groupFromJSON ports ICUGroup(IList<ICUFeature>, dynamic obj). Note that
// InitializeFeatures runs before Name is assigned, so every right starts at the
// feature default (even for "Admin"); "FeatureRights" entries then override
// the feature whose ID matches exactly. Unknown IDs are ignored.
func groupFromJSON(features []*Feature, raw json.RawMessage, what string) (*Group, error) {
	o, err := decodeObject(raw, what)
	if err != nil {
		return nil, err
	}
	g := &Group{}
	g.initializeFeatures(features, false)
	if g.Name, err = o.str("Name", what); err != nil {
		return nil, err
	}
	g.Name = strings.TrimSpace(g.Name)
	s, err := o.str("Comment", what)
	if err != nil {
		return nil, err
	}
	g.Comment = fromBR(s)
	if o.has("HTTPUser") {
		if s, err = o.str("HTTPUser", what); err != nil {
			return nil, err
		}
		g.HTTPUser = strings.TrimSpace(s)
	}
	if o.has("HTTPPassword") {
		if s, err = o.str("HTTPPassword", what); err != nil {
			return nil, err
		}
		g.HTTPPassword = strings.TrimSpace(s)
	}
	if o.has("FeatureRights") {
		frs, err := o.array("FeatureRights", what)
		if err != nil {
			return nil, err
		}
		for i, fraw := range frs {
			fwhat := fmt.Sprintf("%s.FeatureRights[%d]", what, i)
			fo, err := decodeObject(fraw, fwhat)
			if err != nil {
				return nil, err
			}
			if len(g.Features) == 0 {
				continue // the FirstOrDefault predicate (and fr["ID"]) never runs
			}
			id, err := fo.str("ID", fwhat)
			if err != nil {
				return nil, err
			}
			var target *FeatureRight
			for _, fr := range g.Features {
				if fr.Feature.ID == id {
					target = fr
					break
				}
			}
			if target == nil {
				continue
			}
			rs, err := fo.str("Rights", fwhat)
			if err != nil {
				return nil, err
			}
			if target.Rights, err = ParseRights(strings.TrimSpace(rs)); err != nil {
				return nil, fmt.Errorf("%s: %w", fwhat, err)
			}
		}
	}
	return g, nil
}

// resolveGroup ports the group lookup of the ICUUser constructors: exact name,
// then "Extern_{company}", then "Extern_ICU"; nil when none exists.
func resolveGroup(groups []*Group, groupName, company string) *Group {
	for _, name := range []string{groupName, "Extern_" + company, "Extern_ICU"} {
		for _, g := range groups {
			if g.Name == name {
				return g
			}
		}
	}
	return nil
}

// userFromJSON ports ICUUser(List<ICUGroup>, dynamic obj): User, Password,
// Fullname and Group are required; Company defaults to the group name;
// Comment is optional.
func userFromJSON(groups []*Group, raw json.RawMessage, what string) (*User, error) {
	o, err := decodeObject(raw, what)
	if err != nil {
		return nil, err
	}
	u := &User{}
	for _, kv := range []struct {
		key string
		dst *string
	}{{"User", &u.User}, {"Password", &u.Password}, {"Fullname", &u.Fullname}} {
		s, err := o.str(kv.key, what)
		if err != nil {
			return nil, err
		}
		*kv.dst = strings.TrimSpace(s)
	}
	groupName, err := o.str("Group", what)
	if err != nil {
		return nil, err
	}
	groupName = strings.TrimSpace(groupName)
	if o.has("Company") {
		s, err := o.str("Company", what)
		if err != nil {
			return nil, err
		}
		u.Company = strings.TrimSpace(s)
	} else {
		u.Company = groupName
	}
	u.Group = resolveGroup(groups, groupName, u.Company)
	if o.has("Comment") {
		s, err := o.str("Comment", what)
		if err != nil {
			return nil, err
		}
		u.Comment = fromBR(s)
	}
	return u, nil
}

// firmwareFromJSON ports ICUFirmware(dynamic obj); all four keys are required.
func firmwareFromJSON(raw json.RawMessage, what string) (*Firmware, error) {
	o, err := decodeObject(raw, what)
	if err != nil {
		return nil, err
	}
	f := &Firmware{}
	for _, kv := range []struct {
		key string
		dst *string
	}{{"Filename", &f.Filename}, {"Version", &f.Version}, {"Comments", &f.Comments}, {"Date", &f.Date}} {
		s, err := o.str(kv.key, what)
		if err != nil {
			return nil, err
		}
		*kv.dst = strings.TrimSpace(s)
	}
	f.Comments = strings.ReplaceAll(f.Comments, "<br>", "\n")
	return f, nil
}

// addFeature ports ICUConfig.AddFeature: append unless the ID already exists.
func (c *Config) addFeature(f *Feature) {
	for _, e := range c.Features {
		if e.ID == f.ID {
			return
		}
	}
	c.Features = append(c.Features, f)
}

// addManualFeatures ports ICUConfig.AddManualFeatures verbatim (including the
// duplicate entries, which addFeature drops).
func (c *Config) addManualFeatures() {
	page := func(id, name string, def Rights) { c.addFeature(NewPageFeature(id, name, def)) }
	prop := func(id uint16, name string, def Rights) { c.addFeature(NewIDFeature(id, name, def)) }
	page("INFORMATION", "Page Information", RightsReadOnly)
	page("BACKOFFICE", "Page Backoffice", RightsFull)
	page("POWER", "Page Power", RightsReadOnly)
	page("NETWORK", "Page Network", RightsReadOnly)
	page("STATES", "Page States", RightsReadOnly)
	page("SOCKET", "Page Socket", RightsFull)
	page("LOG", "Page Log", RightsFull)
	page("METERVALUES", "Page MeterValue", RightsReadOnly)
	page("WHITELIST", "Page Whitelist", RightsReadOnly)
	page("UI", "Page UserInterface", RightsReadOnly)
	page("UPLOAD", "Page Upload", RightsReadOnly)
	page("PRODUCTION", "Page Production", RightsNone)
	page("ALLPROPERTIES", "All settings page", RightsFull)
	page("TRANSACTIONS", "Page Transactions", RightsFull)
	page("FAT", "Factory Acceptance Test", RightsNone)
	page("SAT", "Service Acceptance Test", RightsFull)
	c.addFeature(NewNormalFeature("CREATEFWU", "Create an FWU file", RightsFull))
	prop(8272, "Charge Box Model", RightsReadOnly)
	prop(8273, "Charge Box Serial Number", RightsReadOnly)
	prop(8275, "Charge Box Identity", RightsReadOnly)
	prop(8271, "Charge Box Configuration", RightsReadOnly)
	prop(8290, "Max Station Current", RightsFull)
	prop(8488, "Start Max Current", RightsFull)
	prop(8489, "Normal Max Current", RightsFull)
	prop(8496, "Simplified Max Current", RightsFull)
	prop(8292, "Load Balancing Mode", RightsReadOnly)
	prop(8295, "P1 Max Installation Current", RightsFull)
	prop(8296, "P1 Balancing Safe Current", RightsFull)
	prop(8310, "BackOffice short name", RightsFull)
	prop(8311, "Connection method", RightsFull)
	prop(8312, "BackOffice Server Domain and Port", RightsFull)
	prop(8305, "BackOffice Server Domain and Port Wired", RightsFull)
	prop(8487, "Main Offline NFC Authorization", RightsFull)
	prop(8503, "Main EV Disconnect Action", RightsFull)
	prop(8448, "GPRS APN Name", RightsFull)
	prop(8449, "GPRS APN User", RightsFull)
	prop(8450, "GPRS APN Password", RightsFull)
	prop(8313, "Communication DNS 1", RightsFull)
	prop(8320, "Communication DNS 2", RightsFull)
	prop(8339, "Send Station Status", RightsFull)
	prop(8321, "Protocol Name", RightsFull)
	prop(8322, "Protocol Version", RightsFull)
	prop(8502, "EV Disconnect Timeout", RightsFull)
	prop(8289, "LED/Display AutoDim & Intensity", RightsFull)
	prop(8485, "Main Socket Type (Socket 1)", RightsFull)
	prop(12581, "Main Socket Type (Socket 2)", RightsFull)
	prop(8728, "Energy Meter Type (Socket 1)", RightsFull)
	prop(12824, "Energy Meter Type (Socket 2)", RightsFull)
	prop(8309, "IP1 Address", RightsFull)
	prop(8307, "IP1 Netmask", RightsFull)
	prop(8308, "IP1 Gateway Address", RightsFull)
	prop(8313, "IP1 DNS 1", RightsFull)
	prop(8320, "IP1 DNS 2", RightsFull)
	prop(8317, "IP2 Address", RightsFull)
	prop(8316, "IP2 Netmask", RightsFull)
	prop(8315, "IP2 Gateway Address", RightsFull)
	prop(8318, "IP2 DNS 1", RightsFull)
	prop(8319, "IP2 DNS 2", RightsFull)
	prop(8485, "Main Socket Type (Socket 1)", RightsFull)
	prop(12581, "Main Socket Type (Socket 2)", RightsFull)
	prop(8728, "Energy Meter Type (Socket 1)", RightsFull)
	prop(12824, "Energy Meter Type (Socket 2)", RightsFull)
}

// addBackofficesToFeatures ports ICUConfig.AddBackofficesToFeatures: drop all
// Backoffice-type features, then add one ReadOnly "BO_*" feature per backoffice
// with a non-empty Title, named "BackOffice '<Title>'".
func (c *Config) addBackofficesToFeatures() {
	c.Features = slices.DeleteFunc(slices.Clone(c.Features), func(f *Feature) bool {
		return f.Type == FeatureTypeBackoffice
	})
	for _, raw := range c.BackOffices {
		h, err := backOfficeHeader(raw)
		if err != nil || h.Title == "" {
			continue
		}
		c.addFeature(NewBackOfficeFeature(h.Title, fmt.Sprintf("BackOffice '%s'", h.Title), RightsReadOnly))
	}
}

// addDefaultGroups ports ICUConfig.AddDefaultGroups (used when the file has no
// groups): Admin, Production, Service, Customer, then "Extern_<company>" per
// distinct non-empty company whose backoffice features are ReadOnly when the
// backoffice's Groups string contains the upper-cased company, else None.
func (c *Config) addDefaultGroups(allCompanies []string) error {
	var companies []string
	for _, co := range allCompanies {
		if !slices.Contains(companies, co) {
			companies = append(companies, co)
		}
	}
	c.Groups = append(c.Groups,
		NewGroup(c.Features, "Admin", "Administrators", "", ""),
		NewGroup(c.Features, "Production", "Production", "", ""),
		NewGroup(c.Features, "Service", "Our own service engineers", "", ""),
		NewGroup(c.Features, "Customer", "Customers", "", ""))
	for _, company := range companies {
		if company == "" {
			continue
		}
		g := NewGroup(c.Features, "Extern_"+company, company, "", "")
		var allowed []string
		for _, raw := range c.BackOffices {
			h, err := backOfficeHeader(raw)
			if err != nil {
				return err
			}
			if strings.Contains(h.Groups, strings.ToUpper(company)) {
				allowed = append(allowed, BackOfficeFeatureID(h.Title))
			}
		}
		for _, fr := range g.Features {
			if fr.Feature.Type == FeatureTypeBackoffice {
				if slices.Contains(allowed, fr.Feature.ID) {
					fr.Rights = RightsReadOnly
				} else {
					fr.Rights = RightsNone
				}
			}
		}
		c.Groups = append(c.Groups, g)
	}
	return nil
}

// FormatDate renders t the way DateTime.Now.ToString() does under the en-US
// culture ("M/d/yyyy h:mm:ss tt"). The C# output is culture dependent (the
// shipped files contain both en-US and nl-NL dates); readers treat it as an
// opaque string.
func FormatDate(t time.Time) string {
	return t.Format("1/2/2006 3:04:05 PM")
}

// Marshal ports the text WriteInstallerSettings produces (before it is written
// to disk): the hand-built JSON (byte-exact), then Base64 or
// EncryptDecrypt.Encrypt(ConfigPassPhrase) according to enc. date is written
// verbatim into "Date" (the C# uses DateTime.Now.ToString(); see FormatDate).
//
// EncryptRijndaelHashed additionally drops the group HTTP credentials and
// replaces Users by CreateHashedUser(user.User, user.Password, user.Group,
// isah.Password, group.HTTPUser+":"+group.HTTPPassword) for every user except
// "isah" (case-insensitive), ordered by the resulting Password. That requires
// an "isah" user and a group on every user (the C# throws otherwise).
// Backoffices / PMBackOffices are emitted only when addBackoffices is set.
func (c *Config) Marshal(enc EncryptionType, addBackoffices bool, date string) (string, error) {
	return c.marshal(rand.Reader, enc, addBackoffices, date)
}

func (c *Config) marshal(rng io.Reader, enc EncryptionType, addBackoffices bool, date string) (string, error) {
	var feats []string
	for _, f := range c.Features {
		if f.Type != FeatureTypeBackoffice {
			feats = append(feats, f.JSON())
		}
	}
	var groups []string
	for _, g := range c.Groups {
		groups = append(groups, g.JSON(enc != EncryptRijndaelHashed))
	}
	var users []string
	if enc == EncryptRijndaelHashed {
		var isah *User
		for _, u := range c.Users {
			if strings.ToLower(u.User) == "isah" {
				isah = u
				break
			}
		}
		var hashed []*User
		for _, u := range c.Users {
			if strings.ToLower(u.User) == "isah" {
				continue
			}
			if u.Group == nil {
				return "", fmt.Errorf("config: user without group cannot be hashed")
			}
			if isah == nil {
				return "", errors.New("config: hashed configs need an \"isah\" user")
			}
			h, err := createHashedUser(rng, u.User, u.Password, u.Group, isah.Password, u.Group.HTTPUser+":"+u.Group.HTTPPassword)
			if err != nil {
				return "", err
			}
			hashed = append(hashed, h)
		}
		// list.OrderBy(a => a.Password): stable. The culture-aware string
		// comparer orders the "NN:HEXSALT:" prefix (digits, upper-case A-F and
		// ':' at fixed positions) exactly like an ordinal comparison.
		slices.SortStableFunc(hashed, func(a, b *User) int { return strings.Compare(a.Password, b.Password) })
		for _, h := range hashed {
			users = append(users, h.JSON())
		}
	} else {
		for _, u := range c.Users {
			users = append(users, u.JSON())
		}
	}
	var fws []string
	for _, f := range c.Firmwares {
		fws = append(fws, f.JSON())
	}
	var bos, pms []string
	if addBackoffices {
		for _, raw := range c.BackOffices {
			bos = append(bos, "\n\t"+string(raw))
		}
		for _, raw := range c.PMBackOffices {
			pms = append(pms, "\n\t"+string(raw))
		}
	}
	text := fmt.Sprintf("{\n\"Type\":\"%s\",\n\"Version\":\"%s\",\n\"Date\":\"%s\",\n\"Features\":[%s\n],\n\"Groups\":[%s\n],\n\"Users\":[%s\n],\n\"Backoffices\":[%s\n],\n\"PMBackOffices\":[%s\n],\n\"Firmwares\":[%s\n]}",
		FileID, c.Version, date,
		strings.Join(feats, ","), strings.Join(groups, ","), strings.Join(users, ","),
		strings.Join(bos, ","), strings.Join(pms, ","), strings.Join(fws, ","))
	switch enc {
	case EncryptBase64:
		text = base64.StdEncoding.EncodeToString([]byte(text))
	case EncryptRijndael, EncryptRijndaelHashed:
		text = Encrypt(text, ConfigPassPhrase)
	}
	return text, nil
}

// Save ports ICUConfig.WriteInstallerSettings(filename, encType,
// addBackoffices): an existing file is first copied to
// Path.ChangeExtension(filename, "bak"), then Marshal(enc, addBackoffices,
// FormatDate(time.Now())) is written as UTF-8 without BOM (StreamWriter).
func (c *Config) Save(filename string, enc EncryptionType, addBackoffices bool) error {
	if old, err := os.ReadFile(filename); err == nil {
		bak := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".bak"
		if err := os.WriteFile(bak, old, 0o600); err != nil {
			return fmt.Errorf("config: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("config: %w", err)
	}
	text, err := c.Marshal(enc, addBackoffices, FormatDate(time.Now()))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, []byte(text), 0o600); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

// FindUser ports ICUConfig.FindUser: case-insensitive match on User (for
// plain-text configs; hashed configs are searched by Logon).
func (c *Config) FindUser(username string) *User {
	lower := strings.ToLower(username)
	for _, u := range c.Users {
		if strings.ToLower(u.User) == lower {
			return u
		}
	}
	return nil
}

// FindFeature returns the feature with exactly this ID (the
// FirstOrDefault(a => a.ID == id) lookup of AddFeature / ICUGroup), or nil.
func (c *Config) FindFeature(id string) *Feature {
	for _, f := range c.Features {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// FindGroup returns the group with exactly this name (the lookup the ICUUser
// constructors use), or nil.
func (c *Config) FindGroup(name string) *Group {
	for _, g := range c.Groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// FindFirmware ports ICUConfig.FindFirmware: case-insensitive match of
// Path.GetFileName(fileName) (Windows separators '\\', '/' and ':') against
// Filename. Used by DlgUpload.
func (c *Config) FindFirmware(fileName string) *Firmware {
	name := fileName
	if i := strings.LastIndexAny(name, "\\/:"); i >= 0 {
		name = name[i+1:]
	}
	name = strings.ToLower(name)
	for _, f := range c.Firmwares {
		if strings.ToLower(f.Filename) == name {
			return f
		}
	}
	return nil
}

// Rights returns the rights the named group has on featureID
// (ICUGroup.GetRights: RightsFull for an unknown feature). A group name that
// does not resolve behaves like a user without a group (ICUUser.GetRights:
// RightsReadOnly).
func (c *Config) Rights(group, featureID string) Rights {
	if g := c.FindGroup(group); g != nil {
		return g.GetRights(featureID)
	}
	return RightsReadOnly
}

// AddGroup ports ICUConfig.AddGroup: a new unnamed group with default rights.
func (c *Config) AddGroup() *Group {
	g := NewGroup(c.Features, "", "", "", "")
	c.Groups = append(c.Groups, g)
	return g
}

// RemoveGroup ports ICUConfig.RemoveGroup.
func (c *Config) RemoveGroup(g *Group) bool {
	c.Groups = slices.DeleteFunc(c.Groups, func(e *Group) bool { return e == g })
	return true
}

// AddUser ports ICUConfig.AddUser (NewUser: no group, random password).
func (c *Config) AddUser() *User {
	u := NewUser()
	c.Users = append(c.Users, u)
	return u
}

// RemoveUser ports ICUConfig.RemoveUser.
func (c *Config) RemoveUser(u *User) bool {
	c.Users = slices.DeleteFunc(c.Users, func(e *User) bool { return e == u })
	return true
}

// AddFirmware ports ICUConfig.AddFirmware (NewFirmware: version "0.0.0").
func (c *Config) AddFirmware() *Firmware {
	f := NewFirmware()
	c.Firmwares = append(c.Firmwares, f)
	return f
}

// RemoveFirmware ports ICUConfig.RemoveFirmware.
func (c *Config) RemoveFirmware(f *Firmware) bool {
	c.Firmwares = slices.DeleteFunc(c.Firmwares, func(e *Firmware) bool { return e == f })
	return true
}
