package isah

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version ports System.Version, which IWSFirmwareFeatures.IsFeatureUnlocked
// compares against. A component that is not given is -1, as in .NET:
// new Version("3.4") is {3, 4, -1, -1}, which is less than 3.4.0.
type Version struct {
	Major, Minor, Build, Revision int
}

// NewVersion ports the System.Version(major, minor[, build[, revision]])
// constructors. It panics on a negative component or on more than two
// optional components, where .NET throws ArgumentOutOfRangeException.
func NewVersion(major, minor int, buildRevision ...int) Version {
	if len(buildRevision) > 2 {
		panic("isah.NewVersion: at most build and revision")
	}
	v := Version{Major: major, Minor: minor, Build: -1, Revision: -1}
	if len(buildRevision) > 0 {
		v.Build = buildRevision[0]
	}
	if len(buildRevision) > 1 {
		v.Revision = buildRevision[1]
	}
	if v.Major < 0 || v.Minor < 0 || (len(buildRevision) > 0 && v.Build < 0) || (len(buildRevision) > 1 && v.Revision < 0) {
		panic("isah.NewVersion: version components must be >= 0")
	}
	return v
}

// ZeroVersion is new Version() (0.0, Build and Revision undefined).
var ZeroVersion = Version{Major: 0, Minor: 0, Build: -1, Revision: -1}

// ParseVersion ports new Version(string) / Version.Parse (.NET Framework
// Version.TryParseVersion): 2 to 4 components separated by '.'. Each component
// is parsed as Int32.TryParse(c, NumberStyles.Integer, InvariantCulture)
// (surrounding white space and a leading sign are allowed) and must be >= 0.
func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return Version{}, errors.New("Version string portion was too short or too long.")
	}
	comps := make([]int, len(parts))
	for i, p := range parts {
		n, ok, overflow := parseDotNetInt(p, 32)
		if !ok {
			if overflow {
				return Version{}, errors.New("Value was either too large or too small for an Int32.")
			}
			return Version{}, errors.New("Input string was not in a correct format.")
		}
		if n < 0 {
			return Version{}, errors.New("Version's parameters must be greater than or equal to zero.")
		}
		comps[i] = int(n)
	}
	return NewVersion(comps[0], comps[1], comps[2:]...), nil
}

// Compare ports Version.CompareTo: Major, then Minor, Build, Revision
// (an undefined -1 sorts before 0).
func (v Version) Compare(o Version) int {
	for _, d := range [4][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Build, o.Build}, {v.Revision, o.Revision}} {
		if d[0] != d[1] {
			if d[0] > d[1] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// Less ports Version.op_LessThan (v < o).
func (v Version) Less(o Version) bool { return v.Compare(o) < 0 }

// String ports Version.ToString(): "Major.Minor[.Build[.Revision]]".
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.Build >= 0 {
		s += "." + strconv.Itoa(v.Build)
		if v.Revision >= 0 {
			s += "." + strconv.Itoa(v.Revision)
		}
	}
	return s
}

// FirmwareVersionNumber ports the ICULanDevice.FirmwareVersionNumber getter
// (ACENetwork/ICUNetwork/ICULanDevice.cs), without the m_vFirmwareVersion
// cache. swVersion is GetPropertyString(4106, 0, 0) (PropSoftwareVersion,
// e.g. "6.4.0-4166"), and isHTTPS is ICULanDevice.IsHTTPS:
//   - fewer than 2 '-' separated parts: 4.10.0 for HTTPS devices, else 4.9.0;
//   - else the first part with "X" replaced by "99" is parsed by
//     new Version(...); a parse error gives new Version() (0.0).
func FirmwareVersionNumber(swVersion string, isHTTPS bool) Version {
	parts := strings.Split(swVersion, "-")
	if len(parts) < 2 {
		if !isHTTPS {
			return NewVersion(4, 9, 0)
		}
		return NewVersion(4, 10, 0)
	}
	v, err := ParseVersion(strings.ReplaceAll(parts[0], "X", "99"))
	if err != nil {
		return ZeroVersion
	}
	return v
}

// isDotNetNumberWhite reports the white space NumberStyles.AllowLeadingWhite
// and AllowTrailingWhite accept (U+0009 to U+000D and U+0020).
func isDotNetNumberWhite(c byte) bool { return c == ' ' || (c >= 0x09 && c <= 0x0d) }

// parseDotNetDigits ports the NumberStyles.Integer grammar of .NET Framework
// Number.ParseNumber with the invariant culture: [ws][+|-]digits[ws], and
// trailing NUL characters are ignored. It returns the sign and the digits.
func parseDotNetDigits(s string) (neg bool, digits string, ok bool) {
	for len(s) > 0 && s[len(s)-1] == 0 {
		s = s[:len(s)-1]
	}
	i, j := 0, len(s)
	for i < j && isDotNetNumberWhite(s[i]) {
		i++
	}
	for j > i && isDotNetNumberWhite(s[j-1]) {
		j--
	}
	s = s[i:j]
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if s == "" {
		return false, "", false
	}
	for k := 0; k < len(s); k++ {
		if s[k] < '0' || s[k] > '9' {
			return false, "", false
		}
	}
	return neg, s, true
}

// parseDotNetInt ports IntNN.TryParse(s, NumberStyles.Integer,
// InvariantCulture) for a signed type of the given bit size. overflow
// separates OverflowException from FormatException for the Parse variants.
func parseDotNetInt(s string, bits int) (n int64, ok, overflow bool) {
	neg, digits, ok := parseDotNetDigits(s)
	if !ok {
		return 0, false, false
	}
	if neg {
		digits = "-" + digits
	}
	n, err := strconv.ParseInt(digits, 10, bits)
	if err != nil {
		return 0, false, true // only ErrRange is possible after the grammar check
	}
	return n, true, false
}

// parseDotNetUint ports UIntNN.TryParse(s, NumberStyles.Integer,
// InvariantCulture) for an unsigned type of the given bit size. "-0" is 0 and
// another negative value is an overflow, as in .NET.
func parseDotNetUint(s string, bits int) (n uint64, ok, overflow bool) {
	neg, digits, ok := parseDotNetDigits(s)
	if !ok {
		return 0, false, false
	}
	n, err := strconv.ParseUint(digits, 10, bits)
	if err != nil {
		return 0, false, true
	}
	if neg && n != 0 {
		return 0, false, true
	}
	return n, true, false
}
