using System;
using System.Xml.Linq;

namespace ICUSettings;

public class ICUPMProperty : ICloneable
{
	public string Name { get; set; }

	public string PropertyName { get; set; }

	public object Value { get; set; }

	public uint PropertyNumber { get; set; }

	public XElement Element
	{
		get
		{
			if (PropertyNumber == 0 || string.IsNullOrEmpty(Value.ToString()))
			{
				return null;
			}
			string text = Convert.ToString(PropertyNumber >> 8, 16);
			byte b = (byte)(PropertyNumber & 0xFF);
			string text2 = ((b != 0) ? ("sub" + Convert.ToString(b, 16)) : string.Empty);
			return new XElement("Object", new XAttribute("Id", "1." + text + text2), new XAttribute("Value", (Value is bool) ? ((object)(((bool)Value) ? 1 : 0)) : Value), PropertyName);
		}
	}

	public object Clone()
	{
		return new ICUPMProperty
		{
			Name = Name,
			PropertyName = PropertyName,
			Value = Value,
			PropertyNumber = PropertyNumber
		};
	}
}
