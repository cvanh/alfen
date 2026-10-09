using System;
using System.ComponentModel;
using System.Linq;
using System.Reflection;

namespace ICUServiceInstaller.Enums;

public static class EnumExtensions
{
	public static string GetDescription(this Enum genericEnum)
	{
		string text = genericEnum.GetDescriptionFromAttribute();
		if (string.IsNullOrEmpty(text))
		{
			text = genericEnum.ToString();
		}
		return text;
	}

	public static string GetDescription(this SupportedWifiSecurityType wifiSecurityType)
	{
		string text = wifiSecurityType.GetDescriptionFromAttribute();
		if (string.IsNullOrEmpty(text))
		{
			text = $"Not supported ({wifiSecurityType})";
		}
		return text;
	}

	private static string GetDescriptionFromAttribute(this Enum genericEnum)
	{
		MemberInfo[] member = genericEnum.GetType().GetMember(genericEnum.ToString());
		if (member != null && member.Any())
		{
			object[] customAttributes = member[0].GetCustomAttributes(typeof(DescriptionAttribute), inherit: false);
			if (customAttributes != null && customAttributes.Any())
			{
				return ((DescriptionAttribute)customAttributes[0]).Description;
			}
		}
		return null;
	}
}
