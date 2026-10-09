using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;

namespace ICUServiceInstaller;

internal static class Tooltips
{
	private static readonly string m_tooltipFolder = "Tooltips";

	private static List<TooltipCollection> m_lsTooltip;

	private static ELanguage m_currentLanguage = ELanguage.English;

	public static ELanguage Language
	{
		get
		{
			return m_currentLanguage;
		}
		set
		{
			m_currentLanguage = value;
		}
	}

	public static void InitToolTip()
	{
		if (m_lsTooltip == null)
		{
			ReinitToolTip();
		}
	}

	public static void ReinitToolTip()
	{
		m_lsTooltip = new List<TooltipCollection>();
		if (!Directory.Exists(m_tooltipFolder))
		{
			return;
		}
		string[] files = Directory.GetFiles(m_tooltipFolder, "tooltip_*_*.csv", SearchOption.TopDirectoryOnly);
		if (!files.Any())
		{
			return;
		}
		string[] array = files;
		for (int i = 0; i < array.Length; i++)
		{
			TooltipCollection tooltipCollection = ParseFile(array[i]);
			if (tooltipCollection != null)
			{
				m_lsTooltip.Add(tooltipCollection);
			}
		}
	}

	private static TooltipCollection ParseFile(string path)
	{
		ELanguage eLanguage = ParseLanguage(Path.GetFileName(path).ToLower().Replace("tooltip_", "")
			.Replace(".csv", ""));
		if (eLanguage == ELanguage.Unknown)
		{
			return null;
		}
		TooltipCollection tooltipCollection = new TooltipCollection(eLanguage);
		try
		{
			using StreamReader streamReader = new StreamReader(path);
			string text;
			while ((text = streamReader.ReadLine()) != null)
			{
				string[] array = text.Split(new char[1] { ',' });
				if (array.Length >= 4)
				{
					Tip item = new Tip(array[0].Trim(), array[1].Trim(), array[2].Trim(), string.Join(",", array, 3, array.Count() - 3).Trim());
					tooltipCollection.Add(item);
				}
				else if (!string.IsNullOrEmpty(text))
				{
					text.StartsWith("//");
				}
			}
			return tooltipCollection;
		}
		catch (Exception)
		{
			return null;
		}
	}

	private static ELanguage ParseLanguage(string lang)
	{
		return lang.ToLowerInvariant().Trim() switch
		{
			"en_gb" => ELanguage.English, 
			"nl_nl" => ELanguage.Dutch, 
			"de_de" => ELanguage.German, 
			"fr_fr" => ELanguage.French, 
			"it_it" => ELanguage.Italian, 
			"nn_no" => ELanguage.Norwegian, 
			"pt_pt" => ELanguage.Portugese, 
			"es_es" => ELanguage.Spanish, 
			"sv_se" => ELanguage.Swedish, 
			"fi_fi" => ELanguage.Finnish, 
			"pl_pl" => ELanguage.Polish, 
			"da_dk" => ELanguage.Danish, 
			"ro_ro" => ELanguage.Romanian, 
			"cz_cz" => ELanguage.Czech, 
			"hu_hu" => ELanguage.Hungarian, 
			"is_is" => ELanguage.Icelandic, 
			_ => ELanguage.Unknown, 
		};
	}

	public static string GetTooltip(string tooltipID, ELanguage language = ELanguage.Unknown)
	{
		TooltipCollection collection = GetCollection(language);
		if (collection == null)
		{
			return string.Empty;
		}
		Tip tip = collection.Get(tooltipID);
		if (tip == null)
		{
			return string.Empty;
		}
		return tip.Tooltip.Replace("\\n", "\n");
	}

	public static string GetLabelText(string tooltipID, ELanguage language = ELanguage.Unknown)
	{
		TooltipCollection collection = GetCollection(language);
		if (collection == null)
		{
			return string.Empty;
		}
		Tip tip = collection.Get(tooltipID);
		if (tip == null)
		{
			return string.Empty;
		}
		return tip.LabelText;
	}

	public static TooltipCollection GetCollection(ELanguage language = ELanguage.Unknown)
	{
		InitToolTip();
		if (language == ELanguage.Unknown)
		{
			language = m_currentLanguage;
		}
		return m_lsTooltip.FirstOrDefault((TooltipCollection a) => a.Language == language);
	}
}
