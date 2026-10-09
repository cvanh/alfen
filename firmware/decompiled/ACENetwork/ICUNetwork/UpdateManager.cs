using System;
using System.Collections.Generic;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Net;
using System.Text.RegularExpressions;
using System.Windows.Forms;
using Serilog;

namespace ICUNetwork;

public class UpdateManager
{
	private static readonly ILogger Logger = Log.ForContext<UpdateManager>();

	protected static string FTPSite;

	protected static string FTPUsername;

	protected static string FTPPassword;

	public static event Action<bool> CanConnect;

	public static void InitUpdateManager(string ftpSite, string ftpUsername, string ftpPassword)
	{
		FTPSite = ftpSite;
		FTPUsername = ftpUsername;
		FTPPassword = ftpPassword;
	}

	private static DateTime? GetFTPFileModificationDate(string fileName, string remotePath = "", int timeout = 1000)
	{
		try
		{
			string relativeUri = Path.Combine(remotePath, Path.GetFileName(fileName));
			Uri uri = new Uri(new Uri(FTPSite), relativeUri);
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "MDTM";
			ftpWebRequest.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			DateTime lastModified = ftpWebResponse.LastModified;
			ftpWebResponse.Close();
			Logger.Debug("GetFTPFileModificationDate {Uri} {LastModified}", uri, lastModified);
			return lastModified;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return null;
		}
	}

	private static bool CanConnectToFTP(int timeout = 500)
	{
		bool flag = false;
		string text = "FileForDownloadTestDoNotDelete.txt";
		try
		{
			Uri uri = new Uri(FTPSite);
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "NLST";
			ftpWebRequest.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			string text2 = new StreamReader(ftpWebResponse.GetResponseStream()).ReadToEnd();
			ftpWebResponse.Close();
			if (text2.IndexOf(text, StringComparison.OrdinalIgnoreCase) >= 0)
			{
				FtpWebRequest ftpWebRequest2 = (FtpWebRequest)WebRequest.Create(new Uri(uri, text));
				ftpWebRequest2.Method = "RETR";
				ftpWebRequest2.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
				ftpWebRequest2.Timeout = timeout;
				FtpWebResponse ftpWebResponse2 = (FtpWebResponse)ftpWebRequest2.GetResponse();
				new StreamReader(ftpWebResponse2.GetResponseStream());
				ftpWebResponse2.Close();
				flag = true;
			}
		}
		catch (Exception)
		{
			flag = false;
		}
		finally
		{
			CanConnect?.Invoke(flag);
		}
		return flag;
	}

	public static Version GetNewestVersion(string fileNameStart, ref string fileName, int timeout = 1000)
	{
		//IL_001e: Unknown result type (might be due to invalid IL or missing references)
		if (!CanConnectToFTP())
		{
			MessageBox.Show("Unable to connect to FTP.\nPlease check your firewall and router settings");
			return null;
		}
		try
		{
			fileName = string.Empty;
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(new Uri(FTPSite));
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "NLST";
			ftpWebRequest.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			string text = new StreamReader(ftpWebResponse.GetResponseStream()).ReadToEnd();
			ftpWebResponse.Close();
			string[] array = (from a in text.Split(new char[1] { '\n' })
				where a.StartsWith(fileNameStart)
				select a.Trim(new char[3] { ' ', '\r', '\n' })).ToArray();
			Dictionary<Version, string> dictionary = new Dictionary<Version, string>();
			string[] array2 = array;
			foreach (string text2 in array2)
			{
				string text3 = text2.Substring(fileNameStart.Length).Trim();
				if (text3.Length > 0)
				{
					Match match = Regex.Match(text3, "v(\\d*).(\\d*).(\\d*).(\\d*)", RegexOptions.IgnoreCase);
					if (match.Success && match.Groups.Count > 4)
					{
						Version version = new Version(Convert.ToInt32(match.Groups[1].Value), Convert.ToInt32(match.Groups[2].Value), Convert.ToInt32(match.Groups[3].Value), Convert.ToInt32(match.Groups[4].Value));
						dictionary.Add(version, text2);
						Logger.Debug("FTP found setup: {FileName}, version: {Version}", text2, version);
					}
				}
			}
			if (dictionary.Count() > 0)
			{
				KeyValuePair<Version, string> keyValuePair = dictionary.OrderByDescending((KeyValuePair<Version, string> a) => a.Key).FirstOrDefault();
				fileName = keyValuePair.Value;
				return keyValuePair.Key;
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return null;
	}

	public static bool IsFTPNewer(string fileName)
	{
		if (!CanConnectToFTP())
		{
			return false;
		}
		try
		{
			DateTime? fTPFileModificationDate = GetFTPFileModificationDate(fileName);
			if (fTPFileModificationDate.HasValue)
			{
				FileInfo fileInfo = new FileInfo(fileName);
				return fTPFileModificationDate > fileInfo.LastWriteTime;
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public static bool DownloadFile(string fileName, string remotePath = "", string localPath = "", Action<string, string> onShowError = null)
	{
		try
		{
			if (!Directory.Exists(localPath))
			{
				Directory.CreateDirectory(localPath);
			}
			string path = Path.Combine(localPath, Path.GetFileName(fileName));
			string relativeUri = Path.Combine(remotePath, Path.GetFileName(fileName));
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(new Uri(new Uri(FTPSite), relativeUri));
			ftpWebRequest.Method = "RETR";
			ftpWebRequest.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
			Stream responseStream = ((FtpWebResponse)ftpWebRequest.GetResponse()).GetResponseStream();
			using (FileStream destination = File.Create(path))
			{
				responseStream.CopyTo(destination);
			}
			DateTime? fTPFileModificationDate = GetFTPFileModificationDate(fileName, remotePath);
			if (fTPFileModificationDate.HasValue)
			{
				File.SetLastWriteTime(path, fTPFileModificationDate.Value);
			}
			return true;
		}
		catch (Exception ex)
		{
			onShowError?.Invoke("Error during file download!", ex.Message);
			return false;
		}
	}

	public static List<string> CheckAdditionalFTPFiles(string RemoteFolder, string LocalFolder, out long totalDownloadSize, bool checkFileDates = true, bool removeFilesNotOnFTP = false, int timeout = 1000)
	{
		List<string> list = new List<string>();
		totalDownloadSize = 0L;
		if (!CanConnectToFTP())
		{
			return list;
		}
		try
		{
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(new Uri(new Uri(FTPSite), RemoteFolder));
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "LIST";
			ftpWebRequest.Credentials = new NetworkCredential(FTPUsername, FTPPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			string text = new StreamReader(ftpWebResponse.GetResponseStream()).ReadToEnd();
			ftpWebResponse.Close();
			string[] array = (from a in text.Split(new char[1] { '\n' })
				select a.Trim(new char[3] { ' ', '\r', '\n' }) into a
				where !string.IsNullOrEmpty(a)
				select a).ToArray();
			if (!Directory.Exists(LocalFolder))
			{
				Directory.CreateDirectory(LocalFolder);
			}
			Dictionary<string, bool> dictionary = Directory.GetFiles(LocalFolder).ToDictionary((string a) => Path.GetFileName(a), (string f) => false);
			Regex regex = new Regex("^([\\-ld])([\\-rwxs]{9})\\s+(\\d+)\\s+(\\w+)\\s+(\\w+)\\s+(\\d+)\\s+(\\w{3}\\s+\\d{1,2}\\s+(?:\\d{1,2}:\\d{1,2}|\\d{4}))\\s+(.+)$");
			string[] array2 = array;
			foreach (string input in array2)
			{
				Match match = regex.Match(input);
				string value = match.Groups[8].Value;
				string ftpFileNameOnly = Path.GetFileName(value).ToLowerInvariant();
				KeyValuePair<string, bool> keyValuePair = dictionary.FirstOrDefault((KeyValuePair<string, bool> a) => a.Key.ToLowerInvariant() == ftpFileNameOnly);
				int result = 0;
				if (!int.TryParse(match.Groups[6].Value, out result))
				{
					result = 0;
				}
				if (string.IsNullOrEmpty(keyValuePair.Key))
				{
					list.Add(Path.Combine(RemoteFolder, ftpFileNameOnly));
					totalDownloadSize += result;
					continue;
				}
				dictionary[keyValuePair.Key] = true;
				if (!checkFileDates)
				{
					continue;
				}
				FileInfo fileInfo = new FileInfo(Path.Combine(LocalFolder, ftpFileNameOnly));
				DateTime dateTime;
				if (match.Groups[7].Value.Contains(":"))
				{
					dateTime = Convert.ToDateTime(DateTime.Now.Year.ToString(CultureInfo.InvariantCulture) + " " + match.Groups[7].Value);
					if (dateTime.DayOfYear > DateTime.Now.DayOfYear)
					{
						dateTime = Convert.ToDateTime((DateTime.Now.Year - 1).ToString(CultureInfo.InvariantCulture) + " " + match.Groups[7].Value);
					}
				}
				else
				{
					dateTime = Convert.ToDateTime(match.Groups[7].Value);
				}
				if (dateTime > fileInfo.LastWriteTime)
				{
					list.Add(Path.Combine(RemoteFolder, value));
					totalDownloadSize += result;
				}
			}
			if (removeFilesNotOnFTP)
			{
				foreach (KeyValuePair<string, bool> item in dictionary.Where((KeyValuePair<string, bool> a) => !a.Value))
				{
					try
					{
						char directorySeparatorChar = Path.DirectorySeparatorChar;
						File.Delete(LocalFolder + directorySeparatorChar + item.Key);
					}
					catch (IOException ex)
					{
						Logger.Error(ex, ex.Message);
					}
				}
			}
		}
		catch (Exception ex2)
		{
			Logger.Error(ex2, ex2.Message);
		}
		return list;
	}
}
