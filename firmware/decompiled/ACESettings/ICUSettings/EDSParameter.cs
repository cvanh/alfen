using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Xml.Linq;

namespace ICUSettings;

public class EDSParameter
{
	private static readonly string[] seperator = new string[1] { "sub" };

	public int Id { get; set; }

	public int SubId { get; set; }

	public string Name { get; set; }

	public string Title { get; set; }

	public int DataType { get; set; }

	public string Units { get; set; }

	public bool ReadWrite { get; set; }

	public ulong MaxLength { get; set; }

	public List<EDSParameterOption> Options { get; set; }

	public EDSParameter(XElement parameter)
	{
		int result = 0;
		string[] array = parameter.Attribute("Id").Value.Split(seperator, StringSplitOptions.None);
		if (array.Length != 0)
		{
			if (int.TryParse(array[0], NumberStyles.HexNumber, null, out result))
			{
				Id = result;
			}
			if (array.Length > 1 && int.TryParse(array[1], NumberStyles.HexNumber, null, out result))
			{
				SubId = result;
			}
		}
		Name = parameter.Attribute("ParameterName").Value;
		string text = parameter.Attribute("DataType").Value;
		if (text.StartsWith("0x", StringComparison.OrdinalIgnoreCase))
		{
			text = text.Substring(2);
		}
		if (int.TryParse(text, NumberStyles.HexNumber, null, out result))
		{
			DataType = result;
		}
		string value = parameter.Attribute("AccessType").Value;
		ReadWrite = value.Equals("rw", StringComparison.OrdinalIgnoreCase);
		if (parameter.Descendants("Title").Any())
		{
			Title = parameter.Descendants("Title").First().Value;
		}
		if (parameter.Descendants("Units").Any())
		{
			Units = parameter.Descendants("Units").First().Value;
		}
		if (parameter.Attribute("Length") != null)
		{
			MaxLength = Convert.ToUInt64(parameter.Attribute("Length").Value, CultureInfo.InvariantCulture);
		}
		else
		{
			MaxLength = 256uL;
		}
		IEnumerable<XElement> enumerable = parameter.Descendants("Option");
		if (!enumerable.Any())
		{
			return;
		}
		Options = new List<EDSParameterOption>();
		int num = 0;
		foreach (XElement item in enumerable)
		{
			Options.Add(new EDSParameterOption(num++, item));
		}
	}
}
