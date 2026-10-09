using System.Linq;
using System.Xml.Linq;

namespace ICUSettings;

public class EDSParameterOption
{
	public int Index { get; set; }

	public string Value { get; set; }

	public string Title { get; set; }

	public XElement XElement { get; set; }

	public EDSParameterOption(int index, XElement option)
	{
		XElement = option;
		Index = index;
		Value = option.Attribute("Value").Value;
		if (option.Descendants("Title").Any())
		{
			Title = option.Descendants("Title").First().Value;
		}
	}
}
