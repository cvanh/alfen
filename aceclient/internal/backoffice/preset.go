package backoffice

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

// Pseudo-preset keys and titles of the "Backoffice preset" select
// (PanelConnectivity m_keyManual/m_keyStandAlone).
const (
	KeyManual       = "_MAN"
	KeyStandAlone   = "_SA"
	TitleManual     = "<Manually enter backend settings>"
	TitleStandAlone = "<Standalone>"
)

// PresetsFolderName ports AppProperties.LocalBackofficePresetsFolderName (and
// FTPBackofficePresetsFolder): the folder under
// %AppData%\ACE Service Installer that holds the preset files.
const PresetsFolderName = "BackofficePresets"

// InstallerFolderName ports AppProperties.LocalInstallerFolder.
const InstallerFolderName = "ACE Service Installer"

// Object-dictionary indices used by the backoffice section of
// PanelConnectivity (decimal, as in the C#).
const (
	PropFirmwareVersion        uint16 = 4106 // "6.6.2-4227" (FirmwareVersionNumber)
	PropBackOfficeURLWired     uint16 = 8305 // sub 1 domain+port, sub 2 path
	PropShortName              uint16 = 8310 // "BackOffice short name": "<preset>[,<meter>]"
	PropConnectMethod          uint16 = 8311 // 0 none, 1 wired, 2 mobile, 3 auto (99 = auto)
	PropBackOfficeURL          uint16 = 8312 // mobile: sub 1 domain+port, sub 2 path
	PropProtocolName           uint16 = 8321 // "ocpp/json"
	PropProtocolVersion        uint16 = 8322
	PropNetworkProfile         uint16 = 8432 // first of 4 network profiles (8432..8435)
	PropNetworkProfileAttempts uint16 = 8436
	PropAPNName                uint16 = 8448
	PropAPNUser                uint16 = 8449
	PropAPNPassword            uint16 = 8450
	PropSIMPin                 uint16 = 8451
	PropProxyEnabled           uint16 = 8471
)

// NetworkProfileCount is m_catNetworkProfile.Length.
const NetworkProfileCount = 4

// Network-profile subindices: PanelConnectivity.ENPSubIds.
const (
	NPSubProtocolVersion       byte = 1
	NPSubURL                   byte = 3
	NPSubMessageTimeout        byte = 4
	NPSubSecurityProfile       byte = 5
	NPSubInterface             byte = 6
	NPSubAPNName               byte = 7
	NPSubAPNUsernameDeprecated byte = 8
	NPSubAPNPassword           byte = 9
	NPSubSimPinDeprecated      byte = 10
	NPSubPriority              byte = 14
	NPSubAPNUsername           byte = 15
	NPSubSimPin                byte = 16
)

// NPVersion ports PanelConnectivity.ENPVersion (network profile sub 1).
type NPVersion int

// PanelConnectivity.ENPVersion members.
const (
	NPVersionNone    NPVersion = 0
	NPVersionUnknown NPVersion = 1
	NPVersionOCPP15  NPVersion = 2
	NPVersionOCPP16  NPVersion = 3
	NPVersionOCPP201 NPVersion = 4
)

// Connect-method values (PanelConnectivity connMethod* constants) and their
// m_dicConnectionMethod titles.
const (
	ConnectMethodNone  = "0"
	ConnectMethodWired = "1"
	ConnectMethodGPRS  = "2"
	ConnectMethodAuto  = "3"
)

// ConnectMethodTitles ports m_dicConnectionMethod (insertion order).
var ConnectMethodTitles = []Option{
	{ConnectMethodNone, "None"},
	{ConnectMethodWired, "Wired"},
	{ConnectMethodGPRS, "Mobile"},
	{ConnectMethodAuto, "Autodetect"},
}

// Option is one key/title entry of a select (a Dictionary<string,string>
// entry in the C#).
type Option struct {
	Key   string
	Title string
}

// UpdateFileTypes ports ICULanDevice.getUpdateFileTypes: the extensions a
// preset (or firmware) file must have for the connected device family.
func UpdateFileTypes(isAHP bool) []string {
	if isAHP {
		return []string{".tfw", ".tcf"}
	}
	return []string{".fwi", ".fwu"}
}

// RemoveTrailingEncryptionKey ports PanelConnectivity.RemoveTrailingEncryptionKey:
// strips a "-a"/"-b"/"-c" key-variant suffix (case-insensitive).
func RemoveTrailingEncryptionKey(fileName string) string {
	l := strings.ToLower(fileName)
	if strings.HasSuffix(l, "-a") || strings.HasSuffix(l, "-b") || strings.HasSuffix(l, "-c") {
		return fileName[:len(fileName)-2]
	}
	return fileName
}

// Catalog ports PanelConnectivity.m_allBackoffices: preset name (file name
// without extension, case preserved) -> full path. Lookups are case-sensitive
// like the C# Dictionary. (The installer's FTP sync stores new files
// lowercased, so on a real install every key is lowercase.)
type Catalog map[string]string

// csExtension ports Path.GetExtension (a trailing '.' yields "").
func csExtension(p string) string {
	e := filepath.Ext(p)
	if e == "." {
		return ""
	}
	return e
}

// NewCatalog ports the m_allBackoffices query of PanelConnectivity.OnChangeDevice
// over a list of file paths: files whose lowercased extension is one of
// UpdateFileTypes(isAHP), keyed by name without extension. A duplicate key
// fails, like ToDictionary's ArgumentException.
func NewCatalog(paths []string, isAHP bool) (Catalog, error) {
	types := UpdateFileTypes(isAHP)
	c := Catalog{}
	for _, p := range paths {
		ext := strings.ToLower(csExtension(p))
		match := false
		for _, t := range types {
			if ext == t {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		base := filepath.Base(p)
		key := strings.TrimSuffix(base, csExtension(base))
		if _, dup := c[key]; dup {
			return nil, fmt.Errorf("backoffice presets: duplicate preset name %q", key)
		}
		full, err := filepath.Abs(p)
		if err != nil {
			full = p
		}
		c[key] = full
	}
	return c, nil
}

// ScanPresetFolder ports the Directory.Exists/Directory.GetFiles part of
// PanelConnectivity.OnChangeDevice. A missing folder yields an empty catalog.
func ScanPresetFolder(dir string, isAHP bool) (Catalog, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return Catalog{}, nil
		}
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return NewCatalog(paths, isAHP)
}

// Options ports m_dicBackOffices: the distinct display names (encryption-key
// suffix removed) plus "_MAN"/"_SA", ordered by key. The C# OrderBy uses the
// culture-sensitive default string comparer; cultureCompare approximates it.
func (c Catalog) Options() ([]Option, error) {
	seen := map[string]bool{}
	var keys []string
	for k := range c {
		d := RemoveTrailingEncryptionKey(k)
		if !seen[d] {
			seen[d] = true
			keys = append(keys, d)
		}
	}
	for _, k := range []string{KeyManual, KeyStandAlone} {
		if seen[k] { // dictionary.Add throws on an existing key
			return nil, fmt.Errorf("backoffice presets: preset named %q clashes with a built-in entry", k)
		}
	}
	opts := make([]Option, 0, len(keys)+2)
	for _, k := range keys {
		opts = append(opts, Option{k, k})
	}
	opts = append(opts, Option{KeyManual, TitleManual}, Option{KeyStandAlone, TitleStandAlone})
	sort.SliceStable(opts, func(i, j int) bool { return cultureCompare(opts[i].Key, opts[j].Key) < 0 })
	return opts, nil
}

// Resolve ports the preset-file selection in PanelConnectivity.OnSaveChanges:
// firmware >= 4.12.0 takes "<name>-b", older firmware "<name>-a"; otherwise
// the exact name. "" when nothing matches (nothing is uploaded).
func (c Catalog) Resolve(name string, fw Version) string {
	path := ""
	if fw.AtLeast(V(4, 12, 0)) {
		if p, ok := c[name+"-b"]; ok {
			path = p
		}
	} else if p, ok := c[name+"-a"]; ok {
		path = p
	}
	if path == "" {
		if p, ok := c[name]; ok {
			path = p
		}
	}
	return path
}

// ParseShortName ports the 8310 split in PanelConnectivity.OnChangeDevice
// (also PanelLoadbalancing/DlgModbusRegisterMap): "<preset>[,<meter>]".
func ParseShortName(value string) (preset, meter string) {
	parts := strings.Split(value, ",")
	preset = parts[0]
	if len(parts) > 1 {
		meter = parts[1]
	}
	return preset, meter
}

// CombineShortName ports PanelConnectivity.CombineBopresetMeterName: appends
// ",<meter>" unless that makes the value longer than 50 (UTF-16) characters.
func CombineShortName(preset, meter string) string {
	if meter != "" {
		text := preset + "," + meter
		if len(utf16.Encode([]rune(text))) > 50 {
			return preset
		}
		return text
	}
	return preset
}

// NormalizeConnectMethod ports the `GetPropertyInt(8311).ToString()` with
// "99" -> "3" mapping PanelConnectivity uses for the connect-method select.
func NormalizeConnectMethod(v int) string {
	s := strconv.Itoa(v)
	if s == "99" {
		return ConnectMethodAuto
	}
	return s
}

// LevenshteinDistance ports PanelConnectivity.LevenshteinDistance (over UTF-16
// code units, like the C#).
func LevenshteinDistance(s, t string) int {
	a, b := utf16.Encode([]rune(s)), utf16.Encode([]rune(t))
	n, m := len(a), len(b)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}
	d := make([][]int, n+1)
	for i := range d {
		d[i] = make([]int, m+1)
		d[i][0] = i
	}
	for j := 0; j <= m; j++ {
		d[0][j] = j
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if b[j-1] == a[i-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
		}
	}
	return d[n][m]
}

// SelectionInput is what PanelConnectivity.LoadCustomBackofficeSettings reads
// from the device (after refreshing 8311 resp. 8432..8435 sub 14).
type SelectionInput struct {
	ShortName          string // 8310 value
	HasNetworkProfiles bool   // HasNetworkProfiles(device)
	ConnectMethod      int    // GetPropertyInt(8311, 0) (no network profiles)
	// ProfilePriorities are GetPropertyInt(8432+i, 14) (network profiles).
	ProfilePriorities [NetworkProfileCount]int
	// PartManagementMode is PanelConnectivity.m_isPartManagementMode; it is a
	// readonly field that is never assigned in v4.4.1, i.e. always false.
	PartManagementMode bool
	// Options are the m_dicBackOffices keys (needed for part-management mode).
	Options []Option
}

// Selection is the outcome of SelectionFromDevice.
type Selection struct {
	// Preset is the new value of the "Backoffice preset" select
	// (m_cmbBackOffices.CustomValue); "" leaves it unchanged.
	Preset string
	// ConnectMode is the new value of the connect-method select; "" leaves it
	// unchanged.
	ConnectMode string
}

// SelectionFromDevice ports PanelConnectivity.LoadCustomBackofficeSettings:
// which preset/connect method the panel shows for the device's current state.
func SelectionFromDevice(in SelectionInput) Selection {
	preset, _ := ParseShortName(in.ShortName)
	text := strings.ToLower(csTrim(preset))
	if text == "" || strings.Contains(text, "standalone") || strings.Contains(text, "stand-alone") {
		if !in.HasNetworkProfiles {
			cm := NormalizeConnectMethod(in.ConnectMethod)
			sel := KeyManual
			if cm == ConnectMethodNone {
				sel = KeyStandAlone
			}
			return Selection{Preset: sel, ConnectMode: cm}
		}
		standalone := true
		for _, p := range in.ProfilePriorities {
			if p != 0 {
				standalone = false
				break
			}
		}
		if standalone {
			return Selection{Preset: KeyStandAlone}
		}
		return Selection{Preset: KeyManual}
	}
	if in.PartManagementMode {
		return Selection{Preset: matchPartManagement(text, in.Options)}
	}
	return Selection{Preset: text}
}

// matchPartManagement ports the m_isPartManagementMode branch of
// LoadCustomBackofficeSettings: exact (case-insensitive) match after removing
// the "- production gprs|wired|auto" suffix, else the closest key by
// Levenshtein distance.
func matchPartManagement(text string, options []Option) string {
	t := strings.ToLower(csTrim(text))
	t = strings.ReplaceAll(t, "- production gprs", "")
	t = strings.ReplaceAll(t, "- production wired", "")
	t = strings.ReplaceAll(t, "- production auto", "")
	t = csTrim(t)
	if len(utf16.Encode([]rune(t))) > 1 {
		for _, o := range options {
			if t == strings.ToLower(csTrim(o.Key)) {
				return o.Key
			}
		}
	}
	best, bestD := "", int(^uint(0)>>1)
	for _, o := range options {
		if d := LevenshteinDistance(strings.ToLower(o.Key), t); d < bestD {
			bestD, best = d, o.Key
		}
	}
	return best
}

// cultureCompare approximates the culture-sensitive string comparison
// (Comparer<string>.Default) that orders the preset list in the C#: hyphens
// and apostrophes are ignored at the first level, symbols sort before digits
// and digits before letters, letters compare case-insensitively; ties are
// broken lowercase-first and finally ordinally.
func cultureCompare(a, b string) int {
	key := func(s string) []rune {
		var out []rune
		for _, r := range s {
			if r == '-' || r == '\'' {
				continue
			}
			out = append(out, r)
		}
		return out
	}
	class := func(r rune) int {
		switch {
		case unicode.IsLetter(r):
			return 2
		case unicode.IsDigit(r):
			return 1
		}
		return 0
	}
	ka, kb := key(a), key(b)
	for i := 0; i < len(ka) && i < len(kb); i++ {
		ca, cb := class(ka[i]), class(kb[i])
		if ca != cb {
			return ca - cb
		}
		la, lb := unicode.ToLower(ka[i]), unicode.ToLower(kb[i])
		if la != lb {
			if la < lb {
				return -1
			}
			return 1
		}
	}
	if len(ka) != len(kb) {
		if len(ka) < len(kb) {
			return -1
		}
		return 1
	}
	for i := range ka {
		if ka[i] != kb[i] {
			if unicode.IsLower(ka[i]) && unicode.IsUpper(kb[i]) {
				return -1
			}
			if unicode.IsUpper(ka[i]) && unicode.IsLower(kb[i]) {
				return 1
			}
		}
	}
	return strings.Compare(a, b)
}
