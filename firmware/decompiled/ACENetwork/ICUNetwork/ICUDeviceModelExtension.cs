using System;
using Serilog;

namespace ICUNetwork;

public static class ICUDeviceModelExtension
{
	public static ICUDeviceModel From(string hostName, int numberOfSockets = 1, bool fixedCable = false)
	{
		string text = hostName.ToLowerInvariant().Trim();
		if (text.StartsWith("icu"))
		{
			text = text.Remove(0, 3).Trim();
		}
		if (text.StartsWith("-"))
		{
			text = text.Substring(1);
		}
		text = text.Replace(" ", "-");
		text = text.Replace("_", "-");
		for (uint num = 0u; num < 2; num++)
		{
			switch (text)
			{
			case "twin-4-xl":
				return ICUDeviceModel.Twin_4_XL;
			case "twin-4.0":
			case "twin-4-0":
				return ICUDeviceModel.Twin_4_0;
			case "twin-4.0-single":
			case "twin-4-0-single":
				return ICUDeviceModel.Twin_4_0_Single;
			case "twin-4.1":
			case "twin-4-1":
				return ICUDeviceModel.Twin_4_1;
			case "twin-4.2":
			case "twin-4-2":
				return ICUDeviceModel.Twin_4_2;
			case "twin-3.0":
			case "twin-3-0":
				return ICUDeviceModel.Twin_3_0;
			case "twin-5":
			case "twin-5.0":
			case "twin-5-0":
				return ICUDeviceModel.Twin_5_0;
			case "twin":
				return ICUDeviceModel.Twin_3_0;
			}
			if (text.StartsWith("eve-dual"))
			{
				return ICUDeviceModel.Eve_Dual;
			}
			if (text.StartsWith("eve-single"))
			{
				return ICUDeviceModel.Eve_Single;
			}
			if (text.StartsWith("compact"))
			{
				if (fixedCable)
				{
					return ICUDeviceModel.Compact_FC;
				}
				return ICUDeviceModel.Compact;
			}
			if (text.StartsWith("lolo3"))
			{
				if (fixedCable)
				{
					return ICUDeviceModel.Lolo3_FC;
				}
				return ICUDeviceModel.Lolo3;
			}
			if (text.StartsWith("tube"))
			{
				if (numberOfSockets == 1)
				{
					return ICUDeviceModel.Tube_1;
				}
				return ICUDeviceModel.Tube_2;
			}
			if (text.StartsWith("eve-mini"))
			{
				if (fixedCable)
				{
					return ICUDeviceModel.Eve_Mini_FC;
				}
				return ICUDeviceModel.Eve_Mini;
			}
			if (text.StartsWith("ng9") || text.StartsWith("ahwp") || text.StartsWith("ahp"))
			{
				int num2 = (text.StartsWith("ahwp") ? 12 : 11);
				if (text.Length < num2)
				{
					num2 = text.Length;
				}
				if (Enum.TryParse<ICUDeviceModel>(text.Substring(0, num2).Replace('-', '_'), ignoreCase: true, out var result))
				{
					return result;
				}
			}
			if (text.LastIndexOf('-') >= 0)
			{
				text = text.Substring(0, text.LastIndexOf('-'));
			}
		}
		Log.Logger.Error("Model name {0} is not recognized as a valid model", hostName);
		return ICUDeviceModel.Unknown;
	}

	public static string ToString(ICUDeviceModel modelType, string hostName)
	{
		switch (modelType)
		{
		case ICUDeviceModel.Twin_3_0:
			return "Twin 3.0";
		case ICUDeviceModel.Twin_4_0:
			return "Twin 4.0";
		case ICUDeviceModel.Twin_4_0_Single:
			return "Twin 4.0 Single";
		case ICUDeviceModel.Twin_4_1:
			return "Twin 4.1";
		case ICUDeviceModel.Twin_4_2:
			return "Twin 4.2";
		case ICUDeviceModel.Twin_4_XL:
			return "Twin 4 XL";
		case ICUDeviceModel.Twin_5_0:
			return "Twin 5.0";
		case ICUDeviceModel.Lolo3:
		case ICUDeviceModel.Lolo3_FC:
			return "LOLO3";
		case ICUDeviceModel.Eve_Dual:
			return "EVe-dual";
		case ICUDeviceModel.Eve_Single:
			return "EVe-single";
		case ICUDeviceModel.Compact:
		case ICUDeviceModel.Compact_FC:
			return "Compact";
		case ICUDeviceModel.Eve_Mini:
		case ICUDeviceModel.Eve_Mini_FC:
			return "ICU Eve Mini";
		case ICUDeviceModel.Tube:
		case ICUDeviceModel.Tube_1:
		case ICUDeviceModel.Tube_2:
			return "TUBE";
		case ICUDeviceModel.Unknown:
		{
			if (string.IsNullOrEmpty(hostName))
			{
				return "Unknown";
			}
			string text = hostName.ToLowerInvariant().Trim();
			if (text.StartsWith("icu"))
			{
				text = text.Remove(0, 3).Trim();
			}
			if (text.StartsWith("-"))
			{
				text = text.Substring(1);
			}
			text = text.Replace(" ", "-");
			text = text.Replace("_", "-");
			int num = text.LastIndexOf('-');
			if (num > 8)
			{
				return text.Substring(0, num);
			}
			return text;
		}
		default:
			return modelType.ToString().Replace('_', '-');
		}
	}
}
