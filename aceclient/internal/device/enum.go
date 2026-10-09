package device

import (
	"sort"
	"strconv"
	"strings"
)

// enumMember is one C# enum member: its name, value and [Description]
// attribute ("" when the member has none).
type enumMember struct {
	name  string
	value int64
	desc  string
}

// enumInfo holds what .NET Framework's Enum.GetCachedValuesAndNames gives
// Enum.ToString / Enum.TryParse: the members sorted by their value converted
// to ulong (a negative value sorts after every positive one), plus whether
// the type carries [Flags].
type enumInfo struct {
	flags   bool
	members []enumMember
}

func newEnumInfo(flags bool, members []enumMember) *enumInfo {
	sorted := append([]enumMember(nil), members...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return uint64(sorted[i].value) < uint64(sorted[j].value)
	})
	return &enumInfo{flags: flags, members: sorted}
}

// name ports Enum.GetName: the member whose value equals v.
func (e *enumInfo) name(v int64) (string, bool) {
	for _, m := range e.members {
		if m.value == v {
			return m.name, true
		}
	}
	return "", false
}

// format ports Enum.ToString() (Enum.InternalFormat, .NET Framework 4.8):
//   - without [Flags]: the member name, else the decimal value;
//   - with [Flags] (InternalFlagsFormat): walking the members from the
//     highest value down, every member whose bits are all still set is
//     prepended and its bits removed; names are joined with ", ". A value
//     that cannot be fully represented gives the decimal value; 0 gives the
//     0-valued member's name, else "0".
func (e *enumInfo) format(v int64) string {
	if !e.flags {
		if n, ok := e.name(v); ok {
			return n
		}
		return strconv.FormatInt(v, 10)
	}
	result := uint64(v)
	save := result
	var names []string
	for i := len(e.members) - 1; i >= 0; i-- {
		mv := uint64(e.members[i].value)
		if i == 0 && mv == 0 {
			break
		}
		if result&mv == mv {
			result -= mv
			names = append([]string{e.members[i].name}, names...)
		}
	}
	if result != 0 {
		return strconv.FormatInt(v, 10)
	}
	if save == 0 {
		if len(e.members) > 0 && e.members[0].value == 0 {
			return e.members[0].name
		}
		return "0"
	}
	return strings.Join(names, ", ")
}

// description ports EnumExtensions.GetEnumDescription
// (ACENetwork/ICUNetwork/EnumExtensions.cs): the [Description] attribute of
// the member named ToString(), else ToString(). Combined flags and undefined
// values have no member of that name and fall back to ToString().
func (e *enumInfo) description(v int64) string {
	s := e.format(v)
	for _, m := range e.members {
		if m.name == s && m.desc != "" {
			return m.desc
		}
	}
	return s
}

// parse ports Enum.TryParse<T>(value, ignoreCase, out result) (.NET Framework
// 4.8 Enum.TryParseEnum for an int-based enum):
//   - value is trimmed; empty fails;
//   - when it starts with a digit, '-' or '+' it is first tried as
//     Convert.ChangeType(value, typeof(int)) (Int32.Parse, NumberStyles.Integer,
//     invariant): success returns that number (defined or not), an overflow
//     fails, a format error falls through to the name path;
//   - otherwise it is split on ',' and every trimmed part must equal a member
//     name (ordinal, or OrdinalIgnoreCase); the values are OR-ed together.
func (e *enumInfo) parse(value string, ignoreCase bool) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if c := value[0]; (c >= '0' && c <= '9') || c == '-' || c == '+' {
		n, err := strconv.ParseInt(value, 10, 32)
		if err == nil {
			return n, true
		}
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return 0, false
		}
	}
	var result uint64
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		found := false
		for _, m := range e.members {
			if m.name == part || (ignoreCase && strings.EqualFold(m.name, part)) {
				result |= uint64(m.value)
				found = true
				break
			}
		}
		if !found {
			return 0, false
		}
	}
	return int64(int32(uint32(result))), true
}
