package i18n

import "testing"

func TestLanguageValuesAndNames(t *testing.T) {
	// ELanguage ordinals and member names (ELanguage.cs).
	want := []string{"Unknown", "English", "Dutch", "German", "French", "Italian",
		"Norwegian", "Portugese", "Spanish", "Swedish", "Finnish", "Polish",
		"Danish", "Romanian", "Czech", "Hungarian", "Icelandic"}
	for i, name := range want {
		if got := Language(i).String(); got != name {
			t.Errorf("Language(%d).String() = %q; want %q", i, got, name)
		}
	}
	if LanguageIcelandic != 16 || LanguagePortugese != 7 {
		t.Errorf("enum values drifted: Icelandic=%d Portugese=%d", LanguageIcelandic, LanguagePortugese)
	}
	if got := Language(17).String(); got != "17" {
		t.Errorf("out-of-range String() = %q", got)
	}
	if got := Language(-1).String(); got != "-1" {
		t.Errorf("negative String() = %q", got)
	}
}

func TestParseLanguage(t *testing.T) {
	tests := []struct {
		in   string
		want Language
	}{
		{"en_gb", LanguageEnglish},
		{"nl_nl", LanguageDutch},
		{"de_de", LanguageGerman},
		{"fr_fr", LanguageFrench},
		{"it_it", LanguageItalian},
		{"nn_no", LanguageNorwegian},
		{"pt_pt", LanguagePortugese},
		{"es_es", LanguageSpanish},
		{"sv_se", LanguageSwedish},
		{"fi_fi", LanguageFinnish},
		{"pl_pl", LanguagePolish},
		{"da_dk", LanguageDanish},
		{"ro_ro", LanguageRomanian},
		{"cz_cz", LanguageCzech},
		{"hu_hu", LanguageHungarian},
		{"is_is", LanguageIcelandic},
		{"EN_GB", LanguageEnglish},
		{"  En_Gb\t", LanguageEnglish},
		{"nb_no", LanguageUnknown}, // only "nn_no" is Norwegian
		{"cs_cz", LanguageUnknown}, // Czech is "cz_cz" in the source
		{"en_us", LanguageUnknown},
		{"en", LanguageUnknown},
		{"", LanguageUnknown},
	}
	for _, tc := range tests {
		if got := ParseLanguage(tc.in); got != tc.want {
			t.Errorf("ParseLanguage(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestLanguageFromFileName(t *testing.T) {
	tests := []struct {
		in   string
		want Language
	}{
		{"tooltip_en_GB.csv", LanguageEnglish},
		{"Tooltips/tooltip_nl_NL.csv", LanguageDutch},
		{`Tooltips\TOOLTIP_DE_DE.CSV`, LanguageGerman},
		{"tooltip_en_GB.csv.csv", LanguageEnglish}, // Replace removes every ".csv"
		{"tooltip_tooltip_fr_FR.csv", LanguageFrench},
		{"tooltip_en_GB.txt", LanguageUnknown},
		{"tooltip_xx_YY.csv", LanguageUnknown},
		{"en_GB.csv", LanguageEnglish}, // no prefix check in ParseFile itself
	}
	for _, tc := range tests {
		if got := LanguageFromFileName(tc.in); got != tc.want {
			t.Errorf("LanguageFromFileName(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestMatchTooltipFile(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"tooltip_en_GB.csv", true},
		{"TOOLTIP_NL_NL.CSV", true},
		{"tooltip_en.csv", false},
		{"tooltip_en_GB.txt", false},
		{"xtooltip_en_GB.csv", false},
		{"Lang_Eve_Mini_EN.csv", false},
	}
	for _, tc := range tests {
		if got := matchTooltipFile(tc.in); got != tc.want {
			t.Errorf("matchTooltipFile(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}
