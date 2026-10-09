package isah

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Features ports the [Flags] enum ICUIWSConnection.IWSFirmwareFeatures.Features
// (ACEISAHConnection/ICUIWSConnection/IWSFirmwareFeatures.cs). The underlying
// type is uint (uint32), and the values are the original ones. On the device,
// the unlocked set is property 8610 (0x21A2) sub 0, see PropFeatureFlags.
type Features uint32

// Feature flag values, 1:1 with IWSFirmwareFeatures.Features.
const (
	FeatureNone                 Features = 0x00000000 // None
	FeatureLoadBalancingSCN     Features = 0x00000001 // LoadBalancing_SCN
	FeatureLoadBalancingStatic  Features = 0x00000002 // LoadBalancing_Static
	FeatureLoadBalancingActive  Features = 0x00000004 // LoadBalancing_Active
	FeatureHighPowerSockets     Features = 0x00000010 // HighPowerSockets
	FeatureRFIDReader           Features = 0x00000100 // RFIDReader
	FeatureISO15118             Features = 0x00000200 // ISO15118
	FeaturePersonalizedDisplay  Features = 0x00001000 // PersonalizedDisplay
	FeatureMobile3G4G           Features = 0x00010000 // Mobile3G4G
	FeaturePaymentOptions       Features = 0x00100000 // Payment_Options
	FeatureExposeSmartMeterData Features = 0x01000000 // Expose_SmartMeterData
	FeatureObjectID             Features = 0x80000000 // ObjectID
)

// FeaturesAll is (IWSFirmwareFeatures.Features)uint.MaxValue, which
// PanelInformation passes to GetFeatureTextLongList to list every feature.
const FeaturesAll Features = 0xFFFFFFFF

// featureEntry is one enum member, as Enum.GetNames/GetValues give them
// (sorted by unsigned value).
type featureEntry struct {
	name  string
	value Features
}

// featureEntries holds the C# member names, sorted ascending by value (the
// order .NET's enum name/value cache uses for formatting and parsing).
var featureEntries = []featureEntry{
	{"None", FeatureNone},
	{"LoadBalancing_SCN", FeatureLoadBalancingSCN},
	{"LoadBalancing_Static", FeatureLoadBalancingStatic},
	{"LoadBalancing_Active", FeatureLoadBalancingActive},
	{"HighPowerSockets", FeatureHighPowerSockets},
	{"RFIDReader", FeatureRFIDReader},
	{"ISO15118", FeatureISO15118},
	{"PersonalizedDisplay", FeaturePersonalizedDisplay},
	{"Mobile3G4G", FeatureMobile3G4G},
	{"Payment_Options", FeaturePaymentOptions},
	{"Expose_SmartMeterData", FeatureExposeSmartMeterData},
	{"ObjectID", FeatureObjectID},
}

// String ports Enum.ToString() for this [Flags] enum (.NET Framework
// InternalFlagsFormat): "None" for 0, a single member name, or member names in
// ascending value order joined by ", ". If any set bit has no member name, the
// result is the decimal number, for example "4294967295" for FeaturesAll.
func (f Features) String() string {
	if f == 0 {
		return featureEntries[0].name
	}
	rest := f
	var names []string
	for i := len(featureEntries) - 1; i >= 0; i-- {
		e := featureEntries[i]
		if i == 0 && e.value == 0 {
			break
		}
		if rest&e.value == e.value {
			rest -= e.value
			names = append([]string{e.name}, names...)
		}
	}
	if rest != 0 {
		return strconv.FormatUint(uint64(f), 10)
	}
	return strings.Join(names, ", ")
}

// ParseFeatures ports Enum.Parse(typeof(IWSFirmwareFeatures.Features), value,
// ignoreCase) from .NET Framework (Enum.TryParseEnum):
//   - the value is trimmed; an empty value is an error;
//   - if it starts with a digit, '-' or '+', it is parsed as a UInt32
//     (invariant culture). An overflow is an error. A format error falls
//     through to name parsing;
//   - else it is split on ',' and each trimmed part must be a member name
//     (ordinal, or ordinal-ignore-case) and the values are OR-ed together.
func ParseFeatures(value string, ignoreCase bool) (Features, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("Must specify valid information for parsing in the string.")
	}
	if c := value[0]; (c >= '0' && c <= '9') || c == '-' || c == '+' {
		n, ok, overflow := parseDotNetUint(value, 32)
		if ok {
			return Features(n), nil
		}
		if overflow {
			return 0, errors.New("Value was either too large or too small for a UInt32.")
		}
	}
	var result Features
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		found := false
		for _, e := range featureEntries {
			if e.name == part || (ignoreCase && strings.EqualFold(e.name, part)) {
				result |= e.value
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("Requested value '%s' was not found.", value)
		}
	}
	return result, nil
}

// GetFeatureTextLongList ports IWSFirmwareFeatures.GetFeatureTextLongList:
// the user-visible feature names that requestedFeatures unlocks. SCN also
// implies active+static load balancing, and active also implies static.
// "32A output per socket" is hidden on DC chargers. "Mobile Technology 3G & 4G"
// is hidden on AHP chargers. Expose_SmartMeterData and ObjectID have no text.
func GetFeatureTextLongList(requestedFeatures Features, isAhp, isDC bool) []string {
	if requestedFeatures == FeatureNone {
		return []string{"None"}
	}
	list := []string{}
	if requestedFeatures&FeatureLoadBalancingSCN != 0 {
		list = append(list, "Smart Charging Network")
	}
	if requestedFeatures&(FeatureLoadBalancingSCN|FeatureLoadBalancingActive) != 0 {
		list = append(list, "Active load balancing")
	}
	if requestedFeatures&(FeatureLoadBalancingSCN|FeatureLoadBalancingStatic|FeatureLoadBalancingActive) != 0 {
		list = append(list, "Static Load balancing")
	}
	if requestedFeatures&FeatureHighPowerSockets != 0 && !isDC {
		list = append(list, "32A output per socket")
	}
	if requestedFeatures&FeatureRFIDReader != 0 {
		list = append(list, "RFID reader")
	}
	if requestedFeatures&FeatureISO15118 != 0 {
		list = append(list, "ISO15118")
	}
	if requestedFeatures&FeaturePersonalizedDisplay != 0 {
		list = append(list, "Personalized display")
	}
	if requestedFeatures&FeatureMobile3G4G != 0 && !isAhp {
		list = append(list, "Mobile Technology 3G & 4G")
	}
	if requestedFeatures&FeaturePaymentOptions != 0 {
		list = append(list, "Direct Payment Solutions")
	}
	return list
}

// DefaultFeatureTextJoin is the default joinText of GetFeatureTextLong.
const DefaultFeatureTextJoin = ", "

// GetFeatureTextLong ports IWSFirmwareFeatures.GetFeatureTextLong
// (string.Join(joinText, GetFeatureTextLongList(...))). The C# default
// joinText is ", " (DefaultFeatureTextJoin). DlgUnlockFeature passes "\n".
func GetFeatureTextLong(requestedFeatures Features, isAhp, isDC bool, joinText string) string {
	return strings.Join(GetFeatureTextLongList(requestedFeatures, isAhp, isDC), joinText)
}

// Version thresholds used by IsFeatureUnlocked (the literals new
// Version("3.4.0") and new Version("1.4.0")).
var (
	licensingMinVersionNG9xx = Version{Major: 3, Minor: 4, Build: 0, Revision: -1}
	licensingMinVersionAHP   = Version{Major: 1, Minor: 4, Build: 0, Revision: -1}
)

// IsFeatureUnlocked ports IWSFirmwareFeatures.IsFeatureUnlocked:
//   - non-AHP firmware below 3.4.0 has no licensing: everything is unlocked,
//     except LoadBalancing_SCN, which still needs its flag;
//   - AHP firmware below 1.4.0: everything is unlocked;
//   - otherwise: (featureEnabledFlags & feature) != 0. FeatureNone is
//     therefore never unlocked on licensed firmware.
func IsFeatureUnlocked(firmwareVersion Version, featureEnabledFlags uint32, feature Features, isAHP bool) bool {
	if !isAHP && firmwareVersion.Less(licensingMinVersionNG9xx) {
		if feature == FeatureLoadBalancingSCN && featureEnabledFlags&uint32(feature) == 0 {
			return false
		}
		return true
	}
	if isAHP && firmwareVersion.Less(licensingMinVersionAHP) {
		return true
	}
	return featureEnabledFlags&uint32(feature) != 0
}

// DeviceFeatures carries the three ICULanDevice values that
// ICULanDevice.IsFeatureUnlocked(feature) reads.
type DeviceFeatures struct {
	FirmwareVersion Version // ICULanDevice.FirmwareVersionNumber (see FirmwareVersionNumber)
	FeatureFlags    uint32  // ICUDevice.GetPropertyUInt(8610, 0) (PropFeatureFlags); 0 if absent
	IsAHP           bool    // ICULanDevice.isAHP
}

// IsFeatureUnlocked ports ICULanDevice.IsFeatureUnlocked
// (ACENetwork/ICUNetwork/ICULanDevice.cs):
// IWSFirmwareFeatures.IsFeatureUnlocked(FirmwareVersionNumber,
// GetPropertyUInt(8610, 0), feature, isAHP).
func (d DeviceFeatures) IsFeatureUnlocked(feature Features) bool {
	return IsFeatureUnlocked(d.FirmwareVersion, d.FeatureFlags, feature, d.IsAHP)
}
