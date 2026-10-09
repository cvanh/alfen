package fwucreator

import (
	"fmt"
	"strconv"
	"strings"
)

// enumName mirrors Enum.ToString(): the member name, or the decimal value for
// an undefined value.
func enumName[T ~int](v T, names []struct {
	Name  string
	Value T
}) string {
	for _, n := range names {
		if n.Value == v {
			return n.Name
		}
	}
	return strconv.Itoa(int(v))
}

func (v ObjectType) String() string         { return enumName(v, objectTypeNames) }
func (v ImageFormat) String() string        { return enumName(v, imageFormatNames) }
func (v DisplayString) String() string      { return enumName(v, displayStringNames) }
func (v StatusIcon) String() string         { return enumName(v, statusIconNames) }
func (v MainState) String() string          { return enumName(v, mainStateNames) }
func (v UserInterfaceError) String() string { return enumName(v, userInterfaceErrorNames) }
func (v UserInterfaceState) String() string { return enumName(v, userInterfaceStateNames) }
func (v FirmwareUpdateStatus) String() string {
	return enumName(v, firmwareUpdateStatusNames)
}

// IsDefined ports Enum.IsDefined(typeof(EFirmwareUpdateStatus), value).
func (v FirmwareUpdateStatus) IsDefined() bool {
	for _, n := range firmwareUpdateStatusNames {
		if n.Value == v {
			return true
		}
	}
	return false
}

// ParseDisplayString ports Enum.Parse(typeof(ICUDisplayStrings), value) as
// AddLanguage uses it (case-sensitive): the value is trimmed; a leading digit,
// '-' or '+' makes it an Int32 (any value, defined or not); otherwise it must
// be a member name. Comma-separated flag lists cannot occur because AddLanguage
// already split the line on ','.
func ParseDisplayString(value string) (DisplayString, error) {
	s := strings.TrimSpace(value)
	if s == "" {
		return 0, fmt.Errorf("Must specify valid information for parsing in the string.")
	}
	if c := s[0]; (c >= '0' && c <= '9') || c == '-' || c == '+' {
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
				return 0, fmt.Errorf("Value was either too large or too small for an Int32.")
			}
			return 0, fmt.Errorf("Input string was not in a correct format.")
		}
		return DisplayString(n), nil
	}
	for _, n := range displayStringNames {
		if n.Name == s {
			return n.Value, nil
		}
	}
	return 0, fmt.Errorf("Requested value '%s' was not found.", s)
}
