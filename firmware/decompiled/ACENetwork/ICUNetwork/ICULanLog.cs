using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Text;
using Serilog;

namespace ICUNetwork;

public class ICULanLog
{
	private readonly ILogger Logger = Log.ForContext<ICULanLog>();

	private const int HTTP_LINES_PER_REQUEST = 32;

	private const int HTTPS_LINES_PER_REQUEST = 16;

	private const int HTTPS_LINES_PER_REQUEST_AHWP = 256;

	private readonly ICULanDevice m_device;

	private (long, long) m_latestKey = (-1L, DateTime.MinValue.Ticks);

	private readonly object m_lineLock = new object();

	private readonly Dictionary<(long, long), ICULanLogLine> m_doubleUpdateLines = new Dictionary<(long, long), ICULanLogLine>();

	private readonly Dictionary<(long, long), ICULanLogLine> m_doubleSaveLines = new Dictionary<(long, long), ICULanLogLine>();

	protected bool m_fUpdateAllowed = true;

	protected ObservableCollection<ICULanLogLine> SaveLines { get; private set; } = new ObservableCollection<ICULanLogLine>();

	public ObservableCollection<ICULanLogLine> LogLines { get; private set; } = new ObservableCollection<ICULanLogLine>();

	private int MaxLines { get; set; }

	public ICULanLog(ICULanDevice device, int maxLines = 10000)
	{
		m_device = device;
		MaxLines = maxLines;
		LogLines.Clear();
		SaveLines.Clear();
	}

	public void Clear()
	{
		LogLines.Clear();
		m_doubleUpdateLines.Clear();
	}

	private int ReadNewLines(long offset, ObservableCollection<ICULanLogLine> lines, Dictionary<(long, long), ICULanLogLine> doubleLines, ref (long, long) latestKey)
	{
		if (m_device == null || lines == null || doubleLines == null)
		{
			return -1;
		}
		List<ICULanLogLine> logLines = m_device.GetLogLines(offset);
		if (logLines.Exists((ICULanLogLine x) => x.ID == long.MinValue))
		{
			Logger.Error("Error while retrieving log lines from device, communication error.");
			return -1;
		}
		lock (m_lineLock)
		{
			int num = logLines.RemoveAll((ICULanLogLine a) => doubleLines.ContainsKey((a.ID, a.Time.Ticks)));
			if (num > 0)
			{
				Logger.Debug("Removed {Removed} duplicate lines from newLogLines", num);
			}
			if (logLines.Count <= 0)
			{
				return 0;
			}
			latestKey = logLines.LastOrDefault().Key;
			if (latestKey.CompareTo(m_latestKey) > 0)
			{
				m_latestKey = latestKey;
			}
			foreach (ICULanLogLine item in logLines)
			{
				(long, long) currentKey = item.Key;
				if (currentKey.CompareTo(m_latestKey) > 0)
				{
					item.ID -= 100000000L;
				}
				try
				{
					doubleLines.Add(currentKey, item);
					if (lines.Count == 0 || currentKey.CompareTo(lines.LastOrDefault().Key) > 0)
					{
						lines.Add(item);
						continue;
					}
					if (currentKey.CompareTo(lines.FirstOrDefault().Key) < 0)
					{
						lines.Insert(0, item);
						continue;
					}
					ICULanLogLine iCULanLogLine = lines.FirstOrDefault((ICULanLogLine a) => a.Key.CompareTo(currentKey) > 0);
					if (iCULanLogLine != null)
					{
						int index = lines.IndexOf(iCULanLogLine);
						lines.Insert(index, item);
					}
					else
					{
						lines.Add(item);
					}
				}
				catch (Exception exception)
				{
					Logger.Debug(exception, "Error while parsing new loglines");
				}
			}
			for (int num2 = 1; num2 < lines.Count; num2++)
			{
				if (lines[num2].State1 == ICUMode3States.Unknown)
				{
					lines[num2].State1 = lines[num2 - 1].State1;
				}
				if (lines[num2].State2 == ICUMode3States.Unknown)
				{
					lines[num2].State2 = lines[num2 - 1].State2;
				}
			}
			return logLines.Count;
		}
	}

	private int ReadLinesForSaving(long offset, ref List<ICULanLogLine> lines, HashSet<(long, long)> doubleLines, ref (long, long) latestKey)
	{
		if (m_device == null || lines == null || doubleLines == null)
		{
			return -1;
		}
		List<ICULanLogLine> logLines = m_device.GetLogLines(offset);
		int count = logLines.Count;
		if (count == 0)
		{
			return 0;
		}
		if (logLines.Exists((ICULanLogLine x) => x.ID == long.MinValue))
		{
			Logger.Error("Error while retrieving log lines from device, communication error.");
			return -1;
		}
		lock (m_lineLock)
		{
			int num = logLines.RemoveAll((ICULanLogLine a) => doubleLines.Contains(a.Key));
			if (num > 0)
			{
				Logger.Debug("Removed {Removed} duplicate lines from newLogLines", num);
			}
			if (logLines.Count == 0)
			{
				return 0;
			}
			doubleLines.UnionWith(logLines.Select((ICULanLogLine x) => x.Key));
			latestKey = logLines.LastOrDefault().Key;
			logLines.AddRange(lines);
			lines = logLines;
			return count;
		}
	}

	private int getLinesPerRequest()
	{
		int result = (m_device.IsHTTPS ? 16 : 32);
		if (m_device.isAHP)
		{
			result = 256;
		}
		return result;
	}

	public int UpdateLog(int desiredLines = 250)
	{
		int linesPerRequest = getLinesPerRequest();
		if (!m_fUpdateAllowed)
		{
			return 0;
		}
		long item = -1L;
		DateTime minValue = DateTime.MinValue;
		(long, long) latestKey = (item, minValue.Ticks);
		int num = 0;
		for (int i = 0; i < (int)Math.Ceiling((double)desiredLines / (double)linesPerRequest); i++)
		{
			int num2 = ReadNewLines(i * linesPerRequest, LogLines, m_doubleUpdateLines, ref latestKey);
			if (num2 < 0)
			{
				return -1;
			}
			num += num2;
		}
		if (LogLines.Count > MaxLines)
		{
			lock (m_lineLock)
			{
				while (LogLines.Count > MaxLines)
				{
					ICULanLogLine iCULanLogLine = LogLines.First();
					m_doubleUpdateLines.Remove((iCULanLogLine.ID, iCULanLogLine.Time.Ticks));
					LogLines.RemoveAt(0);
				}
			}
		}
		return num;
	}

	private static void WriteLoglinesToFile(string path, IEnumerable<ICULanLogLine> logLines)
	{
		int num = 4194304;
		string[] names = Enum.GetNames(typeof(ICULanLogType));
		using FileStream stream = new FileStream(path, FileMode.OpenOrCreate, FileAccess.Write, FileShare.None, 65536);
		using StreamWriter streamWriter = new StreamWriter(stream, new UTF8Encoding(encoderShouldEmitUTF8Identifier: false));
		StringBuilder stringBuilder = new StringBuilder(65536);
		foreach (ICULanLogLine logLine in logLines)
		{
			stringBuilder.AppendFormat(CultureInfo.InvariantCulture, "{0:yyyy-MM-ddTHH:mm:ss.fffZ}:{1}:{2}:{3}:{4}", new object[5]
			{
				logLine.Time,
				names[(int)logLine.Type],
				logLine.SourceFileName,
				logLine.SourceLineNumber,
				logLine.Text
			});
			stringBuilder.AppendLine();
			if (stringBuilder.Length >= num)
			{
				streamWriter.Write(stringBuilder.ToString());
				stringBuilder.Clear();
			}
		}
		if (stringBuilder.Length > 0)
		{
			streamWriter.Write(stringBuilder.ToString());
		}
		streamWriter.Flush();
	}

	public string SaveToFile(TimeSpan span, string fileName, BackgroundWorker bgw = null)
	{
		try
		{
			m_fUpdateAllowed = false;
			Stopwatch stopwatch = Stopwatch.StartNew();
			long item = -2L;
			DateTime minValue = DateTime.MinValue;
			(long, long) other = (item, minValue.Ticks);
			long num = 0L;
			long item2 = -1L;
			minValue = DateTime.MinValue;
			(long, long) latestKey = (item2, minValue.Ticks);
			DateTime dateTime = DateTime.Now - span;
			HashSet<(long, long)> doubleLines = new HashSet<(long, long)>(1000000);
			List<ICULanLogLine> lines = new List<ICULanLogLine>(1000000);
			bool flag = false;
			while (true)
			{
				int num2 = ReadLinesForSaving(num, ref lines, doubleLines, ref latestKey);
				flag = num2 < 0;
				DateTime time = lines.FirstOrDefault().Time;
				if (lines.Count > 0)
				{
					bgw.ReportProgress(lines.Count, $"Downloading lines {time}, collected : {(DateTime.Now - time).TotalDays:0.0} days");
				}
				string messageTemplate = string.Format("SaveToFile, Index: [{0}], Date: {1}, Count {2}, Previous {3}-{4}", new object[5] { lines.Count, time, lines.Count, other.Item1, latestKey.Item1 });
				Logger.Debug(messageTemplate);
				if (flag || latestKey.CompareTo(other) == 0)
				{
					break;
				}
				other = latestKey;
				if (time < dateTime && !TimeSpan.Equals(span, TimeSpan.Zero))
				{
					Logger.Debug("Stopped downloading logfile (date < {Stop}). Downloaded {Count} loglines", dateTime, lines.Count);
					break;
				}
				num += num2;
			}
			Logger.Debug("Downloaded {Count} log lines in {Elapsed} seconds", lines.Count, stopwatch.Elapsed.TotalSeconds);
			stopwatch.Stop();
			WriteLoglinesToFile(fileName, lines);
			m_fUpdateAllowed = true;
			if (flag)
			{
				return $"Error: Not all requested loglines downloaded!\nSaved {lines.Count} loglines in {(int)stopwatch.Elapsed.TotalSeconds} seconds";
			}
			return $"All requested loglines downloaded.\nSaved {lines.Count} loglines in {(int)stopwatch.Elapsed.TotalSeconds} seconds";
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return ex.Message;
		}
	}
}
