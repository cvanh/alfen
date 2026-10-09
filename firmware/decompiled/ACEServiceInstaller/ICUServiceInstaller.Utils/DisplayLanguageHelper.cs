using System;
using System.Collections.Generic;
using System.Linq;
using Serilog;

namespace ICUServiceInstaller.Utils;

internal static class DisplayLanguageHelper
{
	private static readonly ILogger Logger;

	private static readonly List<DisplayLanguage> VersionedLanguages;

	static DisplayLanguageHelper()
	{
		Logger = Log.ForContext("SourceContext", "DisplayLanguageHelper");
		VersionedLanguages = new List<DisplayLanguage>();
		VersionedLanguages.Add(new DisplayLanguage("en_GB", "English", "4.10", "2.0", "0.0"));
		VersionedLanguages.Add(new DisplayLanguage("nl_NL", "Dutch", "4.10", "2.0", "0.0"));
		VersionedLanguages.Add(new DisplayLanguage("fr_FR", "French", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("de_DE", "German", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("it_IT", "Italian", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("es_ES", "Spanish", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("pt_PT", "Portuguese", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("sv_SE", "Swedish", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("fi_FI", "Finnish", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("nn_NO", "Norwegian", "4.10", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("pl_PL", "Polish", "4.14", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("da_DK", "Danish", "5.0", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("ro_RO", "Romanian", "6.3", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("cz_CZ", "Czech", "5.0-6.5", "", ""));
		VersionedLanguages.Add(new DisplayLanguage("cs_CZ", "Czech", "6.6", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("hu_HU", "Hungarian", "6.4", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("is_IS", "Icelandic", "6.4", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("lv_LV", "Latvian", "6.6", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("sk_SK", "Slovac", "6.6", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("sl_SI", "Slovenian", "6.6", "2.0", ""));
		VersionedLanguages.Add(new DisplayLanguage("ca_ES", "Catalan", "7.1", "", ""));
		VersionedLanguages.Add(new DisplayLanguage("hr_HR", "Croatian", "7.1", "", ""));
	}

	internal static Dictionary<string, string> GetDisplayLanguages(bool isAhp, bool isDc, Version fwVersion, string currentLocale)
	{
		Logger.Debug("GetDisplayLanguages for version {FwVersion} and locale {CurrentLocale}", fwVersion, currentLocale);
		Version version = new Version(fwVersion.Major, fwVersion.Minor);
		Dictionary<string, string> dictionary;
		if (isDc)
		{
			dictionary = VersionedLanguages.Where((DisplayLanguage v) => v.FromDcVersion <= version).ToDictionary((DisplayLanguage v) => v.Locale, (DisplayLanguage v) => v.Name);
		}
		else if (isAhp)
		{
			dictionary = VersionedLanguages.Where((DisplayLanguage v) => version >= v.FromAhpVersion).ToDictionary((DisplayLanguage v) => v.Locale, (DisplayLanguage v) => v.Name);
		}
		else
		{
			dictionary = VersionedLanguages.Where((DisplayLanguage v) => version >= v.FromNG9Version && version <= v.ToNG9Version).ToDictionary((DisplayLanguage v) => v.Locale, (DisplayLanguage v) => v.Name);
			if (!dictionary.ContainsKey(currentLocale))
			{
				DisplayLanguage unexpectedLanguage = VersionedLanguages.FirstOrDefault((DisplayLanguage v) => v.Locale == currentLocale);
				if (unexpectedLanguage != null)
				{
					DisplayLanguage displayLanguage = VersionedLanguages.FirstOrDefault((DisplayLanguage v) => v.Name == unexpectedLanguage.Name && v.Locale != currentLocale);
					if (displayLanguage != null)
					{
						dictionary.Remove(displayLanguage.Locale);
						dictionary.Add(unexpectedLanguage.Locale, unexpectedLanguage.Name);
						Logger.Debug("Locale {ChargerLocale} is replaced by {CurrentLocale}", currentLocale, displayLanguage.Locale);
					}
				}
			}
		}
		if (!dictionary.ContainsKey(currentLocale))
		{
			dictionary.Add(currentLocale, currentLocale);
			Logger.Warning("Locale {CurrentLocale} is not in the list of supported languages for version {FwVersion}", currentLocale, fwVersion);
		}
		return dictionary;
	}
}
