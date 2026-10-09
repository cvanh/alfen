using System;
using System.Text.RegularExpressions;

namespace ICUNetwork;

public class ICULogLine
{
	public DateTime Time { get; set; }

	public string Filename { get; set; }

	public int LineNumber { get; set; }

	public string FieldType { get; set; }

	public ICULogType Type { get; set; }

	public string Message { get; set; }

	public ICULogLine(string text)
	{
		string[] array = text.Split(new char[1] { ':' });
		Time = DateTime.Now;
		FieldType = "INFO";
		string text2 = text;
		if (array.Length >= 7)
		{
			string s = (array[0] + ":" + array[1] + ":" + array[2]).Trim(new char[1] { ' ' }).Trim(new char[1] { '$' });
			DateTime result = DateTime.Now;
			DateTime.TryParse(s, out result);
			Time = result;
			FieldType = array[3];
			Filename = array[4];
			int result2 = 0;
			int.TryParse(array[5], out result2);
			LineNumber = result2;
			int startIndex = 6 + array[0].Length + array[1].Length + array[2].Length + array[3].Length + array[4].Length + array[5].Length;
			text2 = text.Substring(startIndex).Trim(new char[1] { '\r' }).Trim(new char[1] { '\n' });
		}
		MatchCollection matchCollection = Regex.Matches(text2, "\\x1B\\[(\\d*;)?(\\d*)m");
		if (matchCollection.Count > 0 && matchCollection[0].Groups.Count > 2)
		{
			int result3 = 0;
			int result4 = 0;
			int.TryParse(matchCollection[0].Groups[1].Value, out result3);
			int.TryParse(matchCollection[0].Groups[2].Value, out result4);
			for (int num = matchCollection.Count - 1; num >= 0; num--)
			{
				text2 = text2.Remove(matchCollection[num].Index, matchCollection[num].Length);
			}
		}
		Type = ICULogType.UNKNOWN;
		switch (FieldType)
		{
		case "INFO":
			Type = ICULogType.INFO;
			break;
		case "WARNING":
			Type = ICULogType.WARNING;
			break;
		case "ERROR":
			Type = ICULogType.ERROR;
			break;
		case "USER":
			Type = ICULogType.USER;
			break;
		case "COM":
			Type = ICULogType.COM;
			break;
		}
		Message = text2;
	}
}
