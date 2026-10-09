// Package i18n ports the ACE Service Installer's tooltip and label-text
// machinery and its Properties.Resources table:
//
//   - ICUServiceInstaller.ELanguage                 -> Language
//   - ICUServiceInstaller.Tip                       -> Tip
//   - ICUServiceInstaller.TooltipCollection         -> TooltipCollection
//   - ICUServiceInstaller.Tooltips (static class)   -> Tooltips + package funcs
//   - UIPropertyBase.MakeToolTipID / LabelText and the shared
//     "overwriteLabelText" rule of the UIProperty* controls -> ids.go
//   - UIPropertyBase.MakeToolTip and the info-button visibility rule ->
//     Tooltips.MakeTooltip / Tooltips.HasTooltip (package funcs use Default)
//   - ICUServiceInstaller.Properties.Resources      -> LogoAlfenSmall
//
// (all under firmware/decompiled/ACEServiceInstaller/).
//
// Tooltip files are "tooltip_<ll>_<CC>.csv", one Tip per line:
//
//	<ID>, <OD name>, <label text>, <tooltip text, may contain commas>
//
// ID is "XXXX_YY" (object-dictionary index and subindex, uppercase hex). The
// installer ships only tooltip_en_GB.csv (in its "Tooltips" folder); this
// package go:embeds a byte-identical copy and uses it as the default source.
//
// The CSV is NOT RFC 4180: the C# just splits on ',' — there is no quoting,
// double quotes are literal, and lines starting with "//" are only skipped
// because they have fewer than four fields.
//
// ELanguage here is the installer UI language for tooltips. The charger's
// own display languages (Lang_Eve_Mini_*.csv, DisplayLanguage) are a
// different concept and live elsewhere.
package i18n

import (
	"path"
	"strconv"
	"strings"
)

// Language ports ICUServiceInstaller.ELanguage (ELanguage.cs). Values match
// the C# enum ordinals.
type Language int

// ports ELanguage (ACEServiceInstaller/ICUServiceInstaller/ELanguage.cs)
const (
	LanguageUnknown   Language = 0
	LanguageEnglish   Language = 1
	LanguageDutch     Language = 2
	LanguageGerman    Language = 3
	LanguageFrench    Language = 4
	LanguageItalian   Language = 5
	LanguageNorwegian Language = 6
	LanguagePortugese Language = 7 // "Portugese" (sic) in the source
	LanguageSpanish   Language = 8
	LanguageSwedish   Language = 9
	LanguageFinnish   Language = 10
	LanguagePolish    Language = 11
	LanguageDanish    Language = 12
	LanguageRomanian  Language = 13
	LanguageCzech     Language = 14
	LanguageHungarian Language = 15
	LanguageIcelandic Language = 16
)

var languageNames = [...]string{
	LanguageUnknown:   "Unknown",
	LanguageEnglish:   "English",
	LanguageDutch:     "Dutch",
	LanguageGerman:    "German",
	LanguageFrench:    "French",
	LanguageItalian:   "Italian",
	LanguageNorwegian: "Norwegian",
	LanguagePortugese: "Portugese",
	LanguageSpanish:   "Spanish",
	LanguageSwedish:   "Swedish",
	LanguageFinnish:   "Finnish",
	LanguagePolish:    "Polish",
	LanguageDanish:    "Danish",
	LanguageRomanian:  "Romanian",
	LanguageCzech:     "Czech",
	LanguageHungarian: "Hungarian",
	LanguageIcelandic: "Icelandic",
}

// String returns the C# enum member name (ELanguage.ToString()); values
// outside the enum format as their number, as .NET does.
func (l Language) String() string {
	if l >= 0 && int(l) < len(languageNames) {
		return languageNames[l]
	}
	return strconv.Itoa(int(l))
}

// ParseLanguage ports Tooltips.ParseLanguage (Tooltips.cs): the
// lower-cased, trimmed locale code selects the language; anything else is
// LanguageUnknown. Note the source's codes: Norwegian is "nn_no" and Czech is
// "cz_cz" (not "cs_cz").
func ParseLanguage(lang string) Language {
	switch strings.TrimSpace(strings.ToLower(lang)) {
	case "en_gb":
		return LanguageEnglish
	case "nl_nl":
		return LanguageDutch
	case "de_de":
		return LanguageGerman
	case "fr_fr":
		return LanguageFrench
	case "it_it":
		return LanguageItalian
	case "nn_no":
		return LanguageNorwegian
	case "pt_pt":
		return LanguagePortugese
	case "es_es":
		return LanguageSpanish
	case "sv_se":
		return LanguageSwedish
	case "fi_fi":
		return LanguageFinnish
	case "pl_pl":
		return LanguagePolish
	case "da_dk":
		return LanguageDanish
	case "ro_ro":
		return LanguageRomanian
	case "cz_cz":
		return LanguageCzech
	case "hu_hu":
		return LanguageHungarian
	case "is_is":
		return LanguageIcelandic
	default:
		return LanguageUnknown
	}
}

// LanguageFromFileName ports the language derivation at the top of
// Tooltips.ParseFile (Tooltips.cs):
//
//	ParseLanguage(Path.GetFileName(path).ToLower()
//	    .Replace("tooltip_", "").Replace(".csv", ""))
//
// Replace removes every occurrence, exactly like the C#. Both '/' and '\'
// are treated as directory separators, as Path.GetFileName does on Windows.
func LanguageFromFileName(name string) Language {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, "tooltip_", "")
	s = strings.ReplaceAll(s, ".csv", "")
	return ParseLanguage(s)
}

// tooltipFilePattern is the Directory.GetFiles search pattern used by
// Tooltips.ReinitToolTip.
const tooltipFilePattern = "tooltip_*_*.csv"

// matchTooltipFile reports whether a file name matches tooltipFilePattern,
// case-insensitively as Windows' Directory.GetFiles does. (The Win32
// three-character-extension quirk that also matches e.g. ".csvx" cannot
// produce a recognised language, so it is not reproduced.)
func matchTooltipFile(name string) bool {
	ok, _ := path.Match(tooltipFilePattern, strings.ToLower(name))
	return ok
}
