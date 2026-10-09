package core

import (
	"strconv"
	"strings"
)

// Version mirrors System.Version as the installer uses it for firmware
// versions: two to four non-negative components, Build and Revision are -1
// when undefined (and an undefined component sorts before 0).
//
// Source: System.Version as used by firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:304-332
type Version struct {
	Major, Minor, Build, Revision int
}

// NewVersion builds a Version; omitted build/revision are -1 (undefined).
func NewVersion(major, minor int, buildRevision ...int) Version {
	v := Version{Major: major, Minor: minor, Build: -1, Revision: -1}
	if len(buildRevision) > 0 {
		v.Build = buildRevision[0]
	}
	if len(buildRevision) > 1 {
		v.Revision = buildRevision[1]
	}
	return v
}

// ParseVersion ports System.Version.TryParse: 2..4 dot-separated components,
// each an Int32 (NumberStyles.Integer: surrounding white space and a leading
// sign allowed) that must not be negative.
func ParseVersion(s string) (Version, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return Version{}, false
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, ok := parseDotNetInt(p, 32, true)
		if !ok || n < 0 {
			return Version{}, false
		}
		nums[i] = int(n)
	}
	return NewVersion(nums[0], nums[1], nums[2:]...), true
}

// MustVersion is ParseVersion for literals; it panics on malformed input.
func MustVersion(s string) Version {
	v, ok := ParseVersion(s)
	if !ok {
		panic("core: bad version literal " + s)
	}
	return v
}

// Compare orders versions like System.Version.CompareTo.
func (v Version) Compare(o Version) int {
	for _, d := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Build, o.Build}, {v.Revision, o.Revision}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Less reports v < o.
func (v Version) Less(o Version) bool { return v.Compare(o) < 0 }

// AtLeast reports v >= o.
func (v Version) AtLeast(o Version) bool { return v.Compare(o) >= 0 }

// String mirrors System.Version.ToString(): undefined components are omitted.
func (v Version) String() string {
	s := strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
	if v.Build >= 0 {
		s += "." + strconv.Itoa(v.Build)
		if v.Revision >= 0 {
			s += "." + strconv.Itoa(v.Revision)
		}
	}
	return s
}

// FirmwareVersionNumber ports ICULanDevice.FirmwareVersionNumber (getter):
//
//	string[] array = GetPropertyString(4106, 0, 0).Split('-');
//	if (array.Count() < 2) return IsHTTPS ? new Version(4,10,0) : new Version(4,9,0);
//	try { m_vFirmwareVersion = new Version(array[0].Replace("X", "99")); }
//	catch { m_vFirmwareVersion = new Version(); }
//
// sw4106 is the value of property 0x100A_0 ("SW Version Comm Unit").
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:304-332 (FirmwareVersionNumber)
func FirmwareVersionNumber(sw4106 string, isHTTPS bool) Version {
	parts := strings.Split(sw4106, "-")
	if len(parts) < 2 {
		if !isHTTPS {
			return NewVersion(4, 9, 0)
		}
		return NewVersion(4, 10, 0)
	}
	v, ok := ParseVersion(strings.ReplaceAll(parts[0], "X", "99"))
	if !ok {
		return NewVersion(0, 0) // new Version() after the caught FormatException
	}
	return v
}

// ---- .NET number parsing helpers (NumberStyles semantics) ----

// isDotNetWhite reports the characters NumberStyles.AllowLeading/TrailingWhite
// accept: U+0009..U+000D and U+0020.
func isDotNetWhite(r rune) bool { return r == ' ' || (r >= 0x09 && r <= 0x0d) }

func trimDotNetWhite(s string) string { return strings.TrimFunc(s, isDotNetWhite) }

// parseDotNetInt ports Int{8,16,32,64}.TryParse with surrounding white space
// allowed and, when allowSign, NumberStyles.AllowLeadingSign.
func parseDotNetInt(s string, bits int, allowSign bool) (int64, bool) {
	t := trimDotNetWhite(s)
	digits := t
	if t != "" && (t[0] == '+' || t[0] == '-') {
		if !allowSign {
			return 0, false
		}
		digits = t[1:]
	}
	if digits == "" {
		return 0, false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(t, 10, bits)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseDotNetUint ports UInt{8,16,32,64}.TryParse with
// NumberStyles.AllowLeadingWhite | AllowTrailingWhite (no sign).
func parseDotNetUint(s string, bits int) (uint64, bool) {
	t := trimDotNetWhite(s)
	if t == "" {
		return 0, false
	}
	for _, c := range t {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(t, 10, bits)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseDotNetHex ports Byte/UInt16.TryParse(s, NumberStyles.HexNumber):
// surrounding white space, hex digits only (no "0x" prefix).
func parseDotNetHex(s string, bits int) (uint64, bool) {
	t := trimDotNetWhite(s)
	if t == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(t, 16, bits)
	if err != nil {
		return 0, false
	}
	return n, true
}

// AnnouncedFirmwareVersion ports the "fwversion" TXT branch of
// ICULanDevice.ReInitialize: Trim().Split('-')[0].ToLowerInvariant()
// .Replace("x", "99") and Version.TryParse. ok is false when it does not
// parse (the C# then keeps m_vFirmwareVersion unset).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:642-760
func AnnouncedFirmwareVersion(fwversion string) (Version, bool) {
	parts := strings.Split(strings.TrimSpace(fwversion), "-")
	return ParseVersion(strings.ReplaceAll(strings.ToLower(parts[0]), "x", "99"))
}
