package config

import (
	"fmt"
	"strconv"
	"strings"
)

// EncryptionType ports ICUSettings.ICUEncryptionType
// (ACESettings/ICUSettings/ICUEncryptionType.cs).
type EncryptionType int

const (
	EncryptNone           EncryptionType = 0 // encryptNone: plain JSON text
	EncryptBase64         EncryptionType = 1 // encryptBase64: Base64(UTF-8 JSON)
	EncryptRijndael       EncryptionType = 2 // encryptRijndael: EncryptDecrypt(ConfigPassPhrase)
	EncryptRijndaelHashed EncryptionType = 3 // encryptRijndaelHashed: as Rijndael, users hashed, group credentials blanked
)

var encryptionTypeNames = []string{"encryptNone", "encryptBase64", "encryptRijndael", "encryptRijndaelHashed"}

// String returns the C# enum member name (Enum.ToString()).
func (e EncryptionType) String() string { return enumName(int(e), encryptionTypeNames) }

// Rights ports ICUSettings.ICURights (ACESettings/ICUSettings/ICURights.cs).
type Rights int

const (
	RightsNone     Rights = 0 // ICURights.None
	RightsReadOnly Rights = 1 // ICURights.ReadOnly
	RightsFull     Rights = 2 // ICURights.Full
)

var rightsNames = []string{"None", "ReadOnly", "Full"}

// String returns the C# enum member name as written into the config JSON
// (Rights.ToString()); undefined values render as their number, like .NET.
func (r Rights) String() string { return enumName(int(r), rightsNames) }

// ParseRights ports Enum.Parse(typeof(ICURights), s) as used by ICUFeature and
// ICUGroup when reading the config.
func ParseRights(s string) (Rights, error) {
	v, err := parseEnum(s, rightsNames, "ICURights")
	return Rights(v), err
}

// FeatureType ports ICUSettings.ICUFeatureType (ACESettings/ICUSettings/ICUFeatureType.cs).
type FeatureType int

const (
	FeatureTypeNormal     FeatureType = 0 // ICUFeatureType.Normal  ("FEATURE_*")
	FeatureTypePage       FeatureType = 1 // ICUFeatureType.Page    ("PAGE_*")
	FeatureTypeProperty   FeatureType = 2 // ICUFeatureType.Property ("ID_XXXX")
	FeatureTypeBackoffice FeatureType = 3 // ICUFeatureType.Backoffice ("BO_*")
)

var featureTypeNames = []string{"Normal", "Page", "Property", "Backoffice"}

// String returns the C# enum member name (Enum.ToString()).
func (t FeatureType) String() string { return enumName(int(t), featureTypeNames) }

// ParseFeatureType ports Enum.Parse(typeof(ICUFeatureType), s).
func ParseFeatureType(s string) (FeatureType, error) {
	v, err := parseEnum(s, featureTypeNames, "ICUFeatureType")
	return FeatureType(v), err
}

func enumName(v int, names []string) string {
	if v >= 0 && v < len(names) {
		return names[v]
	}
	return strconv.Itoa(v)
}

// parseEnum mirrors System.Enum.Parse(Type, string) (case-sensitive) for an
// int-backed, non-[Flags] enum: surrounding white space is ignored, a value
// starting with a digit/sign is parsed as a number (any int accepted), and a
// comma-separated list of member names is OR-ed together.
func parseEnum(s string, names []string, typeName string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("config: must specify valid information for parsing in the string (%s)", typeName)
	}
	if c := s[0]; (c >= '0' && c <= '9') || c == '-' || c == '+' {
		v, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("config: requested value '%s' was not found (%s)", s, typeName)
		}
		return int(v), nil
	}
	result := 0
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		found := false
		for i, n := range names {
			if n == part {
				result |= i
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("config: requested value '%s' was not found (%s)", part, typeName)
		}
	}
	return result, nil
}
