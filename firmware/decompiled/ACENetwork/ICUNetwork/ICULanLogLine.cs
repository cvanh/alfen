using System;
using System.Globalization;
using System.Text.RegularExpressions;
using Serilog;

namespace ICUNetwork;

public class ICULanLogLine : IComparable<ICULanLogLine>
{
	private readonly ILogger Logger = Log.ForContext<ICULanLogLine>();

	public long ID { get; set; }

	public DateTime Time { get; set; }

	public ICULanLogType Type { get; set; }

	public string SourceFileName { get; set; }

	public int SourceLineNumber { get; set; }

	public string Text { get; set; }

	public int LineNumber { get; set; }

	public string Socket1 { get; set; }

	public string Socket2 { get; set; }

	public ICUMode3States State1 { get; set; }

	public ICUMode3States State2 { get; set; }

	public bool IsSocketMsg { get; set; }

	public (long, long) Key => (ID, Time.Ticks);

	public int CompareTo(ICULanLogLine other)
	{
		if (other == null)
		{
			return 1;
		}
		int num = ID.CompareTo(other.ID);
		if (num != 0)
		{
			return num;
		}
		return Time.CompareTo(other.Time);
	}

	public override bool Equals(object obj)
	{
		if (obj is ICULanLogLine iCULanLogLine)
		{
			if (ID == iCULanLogLine.ID)
			{
				return Time == iCULanLogLine.Time;
			}
			return false;
		}
		return false;
	}

	public override int GetHashCode()
	{
		return (ID, Time).GetHashCode();
	}

	public ICULanLogLine(string line)
	{
		Time = DateTime.UtcNow;
		ID = -1L;
		LineNumber = -1;
		IsSocketMsg = false;
		Socket1 = string.Empty;
		Socket2 = string.Empty;
		State1 = ICUMode3States.Unknown;
		State2 = ICUMode3States.Unknown;
		int num = line.IndexOf('_');
		if (num >= 0 && num < 20)
		{
			long.TryParse(line.Substring(0, num), out var result);
			ID = result;
			if (result < 0)
			{
				Logger.Debug("ID < 0");
			}
			string text = line.Substring(num + 1);
			string[] array = text.Split(new char[1] { ':' });
			if (array.Length >= 7)
			{
				string text2 = array[0].Replace("\0", "") + ":" + array[1] + ":" + array[2];
				if (DateTime.TryParseExact(text2, "yyyy-MM-ddTHH:mm:ss.fffZ", CultureInfo.InvariantCulture, DateTimeStyles.AdjustToUniversal | DateTimeStyles.AssumeUniversal, out var result2))
				{
					Time = result2;
				}
				else
				{
					Time = DateTime.UtcNow;
					Logger.Error("Error while parsing datetime {dateTimeStr} for log line", text2);
				}
				Type = ICULanLogType.UNKNOWN;
				switch (array[3])
				{
				case "INFO":
					Type = ICULanLogType.INFO;
					break;
				case "WARNING":
					Type = ICULanLogType.WARNING;
					break;
				case "ERROR":
					Type = ICULanLogType.ERROR;
					break;
				case "USER":
					Type = ICULanLogType.USER;
					break;
				case "CONSOLE":
					Type = ICULanLogType.CONSOLE;
					break;
				case "SEC":
					Type = ICULanLogType.SECURITY;
					break;
				case "COMM":
				case "COM":
					Type = ICULanLogType.COM;
					break;
				}
				SourceFileName = array[4];
				int.TryParse(array[5], out var result3);
				SourceLineNumber = result3;
				int startIndex = 6 + array[0].Length + array[1].Length + array[2].Length + array[3].Length + array[4].Length + array[5].Length;
				Text = text.Substring(startIndex).Trim(new char[1] { '\r' }).Trim(new char[1] { '\n' });
				Text = Regex.Replace(Text, "\\x1B\\[(\\d*;)?(\\d*)m", "");
				if (Text.StartsWith("Socket #1", StringComparison.OrdinalIgnoreCase))
				{
					Socket1 = Text.Substring(9).TrimStart(new char[2] { ':', ' ' });
					IsSocketMsg = true;
					State1 = ParseState(Socket1);
				}
				else if (Text.StartsWith("Socket #2", StringComparison.OrdinalIgnoreCase))
				{
					Socket2 = Text.Substring(9).TrimStart(new char[2] { ':', ' ' });
					IsSocketMsg = true;
					State2 = ParseState(Socket2);
				}
				else if (Text == "==========================================")
				{
					Type = ICULanLogType.RESET;
					State1 = ICUMode3States.Booting;
					State2 = ICUMode3States.Booting;
				}
			}
			else
			{
				Text = line.Substring(num);
			}
		}
		else
		{
			Text = line;
		}
	}

	public static string GetStateName(ICUMode3States state)
	{
		if (state == ICUMode3States.Unknown || state == ICUMode3States.Booting)
		{
			return string.Empty;
		}
		return Enum.GetName(typeof(ICUMode3States), state);
	}

	private ICUMode3States ParseState(string socketText)
	{
		if (socketText.Contains("Mode-3: STATE"))
		{
			switch (socketText.Substring(socketText.LastIndexOf(':')).Trim(new char[2] { ':', ' ' }))
			{
			case "STATE_B1":
				return ICUMode3States.STATE_B1;
			case "STATE_B2":
				return ICUMode3States.STATE_B2;
			case "STATE_C1":
				return ICUMode3States.STATE_C1;
			case "STATE_C2":
				return ICUMode3States.STATE_C2;
			case "STATE_E":
				return ICUMode3States.STATE_E;
			case "STATE_F":
				return ICUMode3States.STATE_F;
			}
		}
		return ICUMode3States.Unknown;
	}

	public override string ToString()
	{
		return Text;
	}
}
