using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using Serilog;

namespace ICUSettings;

public static class DataSheet
{
	public static XDocument EDS { get; set; }

	public static List<EDSParameter> Parameters { get; set; }

	static DataSheet()
	{
		Parse("eds.xml");
	}

	public static bool Parse(string fileName)
	{
		try
		{
			Parameters = new List<EDSParameter>();
			EDS = XDocument.Load(fileName);
			foreach (XElement item in EDS.Root.Descendants("Object"))
			{
				Parameters.Add(new EDSParameter(item));
			}
			return true;
		}
		catch (Exception exception)
		{
			Log.Logger.Error(exception, "");
			return false;
		}
	}

	public static EDSParameter FindParameter(int Id, int subId = 0)
	{
		if (Parameters == null)
		{
			return null;
		}
		return Parameters.FirstOrDefault((EDSParameter a) => a.Id == Id && a.SubId == subId);
	}

	public static int SelectParameterOption(int Id, int subId, string description)
	{
		int result = 0;
		EDSParameter eDSParameter = FindParameter(Id, subId);
		if (eDSParameter != null)
		{
			int.TryParse(eDSParameter.Options.FirstOrDefault((EDSParameterOption b) => b.Title == description).Value, out result);
		}
		return result;
	}
}
