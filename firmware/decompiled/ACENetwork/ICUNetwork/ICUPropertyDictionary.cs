using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using ICUSettings;

namespace ICUNetwork;

public class ICUPropertyDictionary : List<ICUProperty>
{
	public ICUPropertyDictionary()
	{
		foreach (EDSParameter parameter in DataSheet.Parameters)
		{
			Add(new ICUProperty(parameter));
		}
	}

	public ICUProperty GetProperty(ushort propId, byte subId = 0)
	{
		return this.FirstOrDefault((ICUProperty a) => a.Id == propId && a.SubId == subId);
	}

	public ICUProperty GetProperty(string id_sub)
	{
		string[] array = id_sub.Split(new char[1] { '_' });
		if (array.Length > 1)
		{
			ushort propId = 0;
			byte subId = 0;
			int result = 0;
			if (int.TryParse(array[0], NumberStyles.HexNumber, null, out result))
			{
				propId = Convert.ToUInt16(result);
			}
			if (int.TryParse(array[1], NumberStyles.HexNumber, null, out result))
			{
				subId = Convert.ToByte(result);
			}
			return GetProperty(propId, subId);
		}
		return null;
	}

	public ICUProperty AddProperty(SDT dataType, ushort propId, byte subId, string title, string description, bool readOnly)
	{
		if (GetProperty(propId, subId) == null)
		{
			ICUProperty iCUProperty = new ICUProperty(dataType, propId, subId);
			iCUProperty.ICUName = description;
			iCUProperty.Title = title;
			iCUProperty.ReadOnly = readOnly;
			Add(iCUProperty);
			return iCUProperty;
		}
		return null;
	}

	public string GetText(ushort propId)
	{
		ICUProperty iCUProperty = this.FirstOrDefault((ICUProperty a) => a.Id == propId);
		if (iCUProperty != null && iCUProperty.Value != null)
		{
			return iCUProperty.Value.ToString();
		}
		return "<unknown>";
	}
}
