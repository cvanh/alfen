using System;
using System.ComponentModel;
using System.Reflection;

namespace ICUNetwork;

public static class EnumExtensions
{
	public static string GetEnumDescription(this Enum genericEnum)
	{
		string text = genericEnum.GetDescriptionFromAttribute();
		if (string.IsNullOrEmpty(text))
		{
			text = genericEnum.ToString();
		}
		return text;
	}

	private static string GetDescriptionFromAttribute(this Enum genericEnum)
	{
		MemberInfo[] member = genericEnum.GetType().GetMember(genericEnum.ToString());
		if (member != null && member.Length != 0)
		{
			object[] customAttributes = member[0].GetCustomAttributes(typeof(DescriptionAttribute), inherit: false);
			if (customAttributes != null && customAttributes.Length != 0)
			{
				return ((DescriptionAttribute)customAttributes[0]).Description;
			}
		}
		return null;
	}
}
