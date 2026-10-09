package presets

import (
	"strconv"
	"strings"
)

// DeviceModel ports enum ICUNetwork.ICUDeviceModel
// (ACENetwork/ICUNetwork/ICUDeviceModel.cs). The constants live in
// model_enum.go (generated from the C# enum, same order and values).
//
// PropertyStorage.LoadProperties compares the model stored in a settings file
// against the connected device's ICULanDevice.ModelType, so the enum and its
// two extension methods are needed here.
type DeviceModel int

// String returns the C# enum member name (Enum.ToString), e.g. "Twin_4_0".
// Undefined values print as their number, like an undefined C# enum value.
func (m DeviceModel) String() string {
	if m >= 0 && int(m) < len(deviceModelNames) {
		return deviceModelNames[m]
	}
	return strconv.Itoa(int(m))
}

// normalizeHostName is the shared prefix of ICUDeviceModelExtension.From and
// ToString: lower-case, trim, drop an "icu" prefix (then trim again), drop one
// leading '-', and map ' ' and '_' to '-'.
func normalizeHostName(hostName string) string {
	text := strings.TrimSpace(strings.ToLower(hostName))
	if strings.HasPrefix(text, "icu") {
		text = strings.TrimSpace(text[3:])
	}
	if strings.HasPrefix(text, "-") {
		text = text[1:]
	}
	text = strings.ReplaceAll(text, " ", "-")
	text = strings.ReplaceAll(text, "_", "-")
	return text
}

// ParseDeviceModel ports ICUDeviceModelExtension.From(hostName,
// numberOfSockets = 1, fixedCable = false)
// (ACENetwork/ICUNetwork/ICUDeviceModelExtension.cs). It makes two passes;
// after the first pass the text is cut at its last '-' (e.g. a serial-number
// suffix) and matched again. Unrecognised names return ModelUnknown (the C#
// also logs "Model name {0} is not recognized as a valid model").
func ParseDeviceModel(hostName string, numberOfSockets int, fixedCable bool) DeviceModel {
	text := normalizeHostName(hostName)
	for pass := 0; pass < 2; pass++ {
		switch text {
		case "twin-4-xl":
			return ModelTwin_4_XL
		case "twin-4.0", "twin-4-0":
			return ModelTwin_4_0
		case "twin-4.0-single", "twin-4-0-single":
			return ModelTwin_4_0_Single
		case "twin-4.1", "twin-4-1":
			return ModelTwin_4_1
		case "twin-4.2", "twin-4-2":
			return ModelTwin_4_2
		case "twin-3.0", "twin-3-0":
			return ModelTwin_3_0
		case "twin-5", "twin-5.0", "twin-5-0":
			return ModelTwin_5_0
		case "twin":
			return ModelTwin_3_0
		}
		if strings.HasPrefix(text, "eve-dual") {
			return ModelEve_Dual
		}
		if strings.HasPrefix(text, "eve-single") {
			return ModelEve_Single
		}
		if strings.HasPrefix(text, "compact") {
			if fixedCable {
				return ModelCompact_FC
			}
			return ModelCompact
		}
		if strings.HasPrefix(text, "lolo3") {
			if fixedCable {
				return ModelLolo3_FC
			}
			return ModelLolo3
		}
		if strings.HasPrefix(text, "tube") {
			if numberOfSockets == 1 {
				return ModelTube_1
			}
			return ModelTube_2
		}
		if strings.HasPrefix(text, "eve-mini") {
			if fixedCable {
				return ModelEve_Mini_FC
			}
			return ModelEve_Mini
		}
		if strings.HasPrefix(text, "ng9") || strings.HasPrefix(text, "ahwp") || strings.HasPrefix(text, "ahp") {
			n := 11
			if strings.HasPrefix(text, "ahwp") {
				n = 12
			}
			if len(text) < n {
				n = len(text)
			}
			if m, ok := enumTryParseModel(strings.ReplaceAll(text[:n], "-", "_")); ok {
				return m
			}
		}
		if i := strings.LastIndex(text, "-"); i >= 0 {
			text = text[:i]
		}
	}
	return ModelUnknown
}

// enumTryParseModel ports Enum.TryParse<ICUDeviceModel>(s, ignoreCase: true)
// for the member-name case (the only one reachable from From: the text always
// starts with "ng9"/"ahwp"/"ahp", so it is never numeric).
func enumTryParseModel(s string) (DeviceModel, bool) {
	s = strings.TrimSpace(s)
	for i, n := range deviceModelNames {
		if strings.EqualFold(n, s) {
			return DeviceModel(i), true
		}
	}
	return ModelUnknown, false
}

// Display ports ICUDeviceModelExtension.ToString(modelType, hostName): the
// human readable model name stored in settings files and shown in the UI.
// For ModelUnknown it derives a name from hostName (or "Unknown").
func (m DeviceModel) Display(hostName string) string {
	switch m {
	case ModelTwin_3_0:
		return "Twin 3.0"
	case ModelTwin_4_0:
		return "Twin 4.0"
	case ModelTwin_4_0_Single:
		return "Twin 4.0 Single"
	case ModelTwin_4_1:
		return "Twin 4.1"
	case ModelTwin_4_2:
		return "Twin 4.2"
	case ModelTwin_4_XL:
		return "Twin 4 XL"
	case ModelTwin_5_0:
		return "Twin 5.0"
	case ModelLolo3, ModelLolo3_FC:
		return "LOLO3"
	case ModelEve_Dual:
		return "EVe-dual"
	case ModelEve_Single:
		return "EVe-single"
	case ModelCompact, ModelCompact_FC:
		return "Compact"
	case ModelEve_Mini, ModelEve_Mini_FC:
		return "ICU Eve Mini"
	case ModelTube, ModelTube_1, ModelTube_2:
		return "TUBE"
	case ModelUnknown:
		if hostName == "" {
			return "Unknown"
		}
		text := normalizeHostName(hostName)
		if i := strings.LastIndex(text, "-"); i > 8 {
			return text[:i]
		}
		return text
	default:
		return strings.ReplaceAll(m.String(), "_", "-")
	}
}

// DeviceInfo is the subset of ICULanDevice that PropertyStorage reads
// (ACENetwork/ICUNetwork/ICULanDevice.cs, ICUDevice.cs).
//
// Identification, IPAddress and HostName may be empty, which is treated like
// the C# null (the element is written as <X />).
type DeviceInfo struct {
	ModelType       DeviceModel // ICULanDevice.ModelType
	Model           string      // ICULanDevice.Model = ModelType.Display(HostName)
	NumberOfSockets int         // ICUDevice.NumberOfSockets
	Identification  string      // ICULanDevice.Identification (= Identity)
	IPAddress       string      // ICULanDevice.IPAddress.ToString()
	Port            int         // ICULanDevice.Port
	HostName        string      // ICULanDevice.HostName
}

// NewDeviceInfo builds a DeviceInfo the way ICULanDevice derives it: ModelType
// ports the ICULanDevice.ModelType getter
// (From(HostName, NumberOfSockets, SocketTypes[0] == 0)) and Model ports
// ICULanDevice.Model (ToString(ModelType, HostName)). socketType0 is
// SocketTypes[0] (from the SCN "type" TXT field or property 8485_0).
func NewDeviceInfo(hostName string, numberOfSockets, socketType0 int, identity, ipAddress string, port int) DeviceInfo {
	mt := ParseDeviceModel(hostName, numberOfSockets, socketType0 == 0)
	return DeviceInfo{
		ModelType:       mt,
		Model:           mt.Display(hostName),
		NumberOfSockets: numberOfSockets,
		Identification:  identity,
		IPAddress:       ipAddress,
		Port:            port,
		HostName:        hostName,
	}
}
