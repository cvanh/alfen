package device

import (
	"strconv"
	"strings"
)

// Model ports the enum ICUNetwork.ICUDeviceModel
// (ACENetwork/ICUNetwork/ICUDeviceModel.cs). The constants live in
// models_gen.go; String returns the C# member name.
type Model int

// String ports ICUDeviceModel.ToString(): the member name, or the decimal
// value for an undefined model.
func (m Model) String() string {
	if m >= 0 && int(m) < len(modelNames) {
		return modelNames[m]
	}
	return strconv.Itoa(int(m))
}

// parseModelName ports Enum.TryParse<ICUDeviceModel>(s, ignoreCase: true, out
// result) as ICUDeviceModelExtension.From uses it (see enumInfo.parse).
func parseModelName(s string) (Model, bool) {
	v, ok := modelEnum.parse(s, true)
	return Model(v), ok
}

// modelEnum is the enumInfo view of ICUDeviceModel (built from modelNames).
var modelEnum = func() *enumInfo {
	members := make([]enumMember, len(modelNames))
	for i, n := range modelNames {
		members[i] = enumMember{name: n, value: int64(i)}
	}
	return newEnumInfo(false, members)
}()

// normalizeHostName is the host-name clean-up shared by
// ICUDeviceModelExtension.From and ToString: ToLowerInvariant().Trim(), drop a
// leading "icu" (then Trim()) and a leading '-', and turn ' ' and '_' into '-'.
func normalizeHostName(hostName string) string {
	text := strings.TrimSpace(strings.ToLower(hostName))
	if strings.HasPrefix(text, "icu") {
		text = strings.TrimSpace(text[3:])
	}
	text = strings.TrimPrefix(text, "-")
	text = strings.ReplaceAll(text, " ", "-")
	return strings.ReplaceAll(text, "_", "-")
}

// ModelFrom ports ICUDeviceModelExtension.From(hostName, numberOfSockets,
// fixedCable) (ACENetwork/ICUNetwork/ICUDeviceModelExtension.cs, lines 8-105).
// The normalized host name is matched twice: as is, then with its last
// "-<part>" removed (the serial). Exact names select the Twin variants,
// prefixes the Eve/Compact/LOLO3/TUBE families (fixedCable and
// numberOfSockets pick the variant), and "ng9…", "ahwp…" and "ahp…" names
// parse their first 11 (12 for "ahwp") characters, '-' -> '_', as an
// ICUDeviceModel member name, ignoring case. Unrecognized names give
// ModelUnknown (the C# also logs an error).
func ModelFrom(hostName string, numberOfSockets int, fixedCable bool) Model {
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
		switch {
		case strings.HasPrefix(text, "eve-dual"):
			return ModelEve_Dual
		case strings.HasPrefix(text, "eve-single"):
			return ModelEve_Single
		case strings.HasPrefix(text, "compact"):
			if fixedCable {
				return ModelCompact_FC
			}
			return ModelCompact
		case strings.HasPrefix(text, "lolo3"):
			if fixedCable {
				return ModelLolo3_FC
			}
			return ModelLolo3
		case strings.HasPrefix(text, "tube"):
			if numberOfSockets == 1 {
				return ModelTube_1
			}
			return ModelTube_2
		case strings.HasPrefix(text, "eve-mini"):
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
			// text.Substring(0, n) counts UTF-16 code units; host names are ASCII.
			u := []rune(text)
			if len(u) < n {
				n = len(u)
			}
			if m, ok := parseModelName(strings.ReplaceAll(string(u[:n]), "-", "_")); ok {
				return m
			}
		}
		if i := strings.LastIndexByte(text, '-'); i >= 0 {
			text = text[:i]
		}
	}
	return ModelUnknown
}

// ModelString ports ICUDeviceModelExtension.ToString(modelType, hostName)
// (ICUDeviceModelExtension.cs, lines 107-172): the display name of a model.
// Known families have fixed names ("Twin 4.0", "EVe-dual", "ICU Eve Mini",
// "TUBE", ...); ModelUnknown is derived from the host name ("Unknown" when it
// is empty; otherwise the normalized name, cut at its last '-' when that '-'
// is past index 8); every other model is its member name with '_' -> '-'
// (e.g. "NG910-60023").
func ModelString(model Model, hostName string) string {
	switch model {
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
		if i := strings.LastIndexByte(text, '-'); i > 8 {
			return text[:i]
		}
		return text
	}
	return strings.ReplaceAll(model.String(), "_", "-")
}

// ModelOptions ports PanelInformation's m_dicModel (PanelInformation.cs,
// lines 69-76): ModelString(m, "") for every ICUDeviceModel value in order,
// duplicates dropped. The "Model" row (property 8272) is a read-only select
// over these strings, keyed and titled by the same text.
func ModelOptions() []string {
	var out []string
	seen := map[string]bool{}
	for m := range Model(len(modelNames)) {
		s := ModelString(m, "")
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// modelsWithoutIcon are the ICUDeviceModel members without an embedded
// "ICUServiceInstaller.Resources.<member>.png" in ACEServiceInstaller.exe
// (firmware/msi_work/files3); UIListLanDevice.Icon falls back to
// "Unknown.png" for them.
var modelsWithoutIcon = map[Model]bool{
	ModelTwin:            true,
	ModelTwin_4_0_Single: true,
	ModelNG920_52503:     true,
}

// IconName ports the resource name UIListLanDevice.Icon loads for a model:
// Enum.GetName(typeof(ICUDeviceModel), model) + ".png". ok is false when the
// installer ships no such image (it then shows "Unknown.png", without the
// lock overlay).
func (m Model) IconName() (name string, ok bool) {
	if m < 0 || int(m) >= len(modelNames) || modelsWithoutIcon[m] {
		return "Unknown.png", false
	}
	return modelNames[m] + ".png", true
}
