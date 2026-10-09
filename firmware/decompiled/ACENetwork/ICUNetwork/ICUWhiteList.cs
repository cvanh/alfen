using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net;
using System.Threading;
using System.Web.Script.Serialization;
using Serilog;

namespace ICUNetwork;

public class ICUWhiteList
{
	public delegate void WhiteListHandler(object myObject, WhiteListArgs args);

	private readonly ILogger Logger = Log.ForContext<ICUWhiteList>();

	private readonly ICULanDevice m_device;

	private readonly BackgroundWorker m_bgwReadWhitelist;

	private readonly object readWhitelistLock = new object();

	private bool m_restartWhitelistUpdate;

	private const int s_nRequestLongTimeOut = 10000;

	private const string noExpiryDate = "1970-01-01";

	public ObservableCollection<ICUWhitelistItem> Whitelist { get; private set; } = new ObservableCollection<ICUWhitelistItem>();

	public bool IsUpdating { get; private set; }

	public int MaxTags { get; set; }

	public event WhiteListHandler WhitelistUpdated;

	public event WhiteListHandler WhitelistCompleted;

	public ICUWhiteList(ICULanDevice device, int maxTags = 10000)
	{
		m_device = device;
		MaxTags = maxTags;
		Whitelist.Clear();
		m_bgwReadWhitelist = new BackgroundWorker();
		m_bgwReadWhitelist.DoWork += DoWorkRead;
		m_bgwReadWhitelist.RunWorkerCompleted += CompletedReading;
		m_bgwReadWhitelist.WorkerSupportsCancellation = true;
	}

	public bool Remove(string tagId)
	{
		if (m_device == null)
		{
			return false;
		}
		return m_device.ExecuteWebRequest("whitelist", "remove=" + tagId).RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public bool Clear()
	{
		if (m_device == null)
		{
			return false;
		}
		bool flag = m_device.ExecuteWebRequest("whitelist", "clear", null, 10000).RequestState == EWebRequestState.VALID_RESPONSE;
		if (flag)
		{
			Whitelist.Clear();
		}
		return flag;
	}

	public bool Add(string tagId)
	{
		if (m_device == null)
		{
			return false;
		}
		Logger.Debug("Adding tag: '{TagId}'", tagId);
		return m_device.ExecuteWebRequest("whitelist", "add=" + tagId).RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public bool UpdateOrAdd(string tagId, string parentId, ICUTagStatus status, string dateTime)
	{
		if (m_device == null)
		{
			return false;
		}
		string content = string.Format("{{\"tagid\":\"{0}\",\"parentid\":\"{1}\",\"status\":{2},\"expire\":\"{3}\"}}", new object[4]
		{
			tagId,
			parentId,
			(int)status,
			dateTime
		});
		return m_device.ExecuteWebRequest("addtag", "", content).RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public bool ContainsTag(string tagId)
	{
		Read();
		return Whitelist.Any((ICUWhitelistItem a) => a.Tag == tagId);
	}

	public bool StartAutoAddMode()
	{
		if (m_device == null)
		{
			return false;
		}
		Logger.ForContext("ChargerID", $"{m_device.SerialNumber?.GetHashCode():X}").Information("Auto add tag mode started");
		return m_device.ExecuteWebRequest("whitelist", "starttagaddmode").RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public bool Read(bool forceRestart = false)
	{
		if (m_bgwReadWhitelist == null)
		{
			return false;
		}
		if (m_bgwReadWhitelist.IsBusy & forceRestart)
		{
			m_restartWhitelistUpdate = true;
			IsUpdating = true;
			StopReading();
			return true;
		}
		if (!m_bgwReadWhitelist.IsBusy)
		{
			m_bgwReadWhitelist.RunWorkerAsync();
			IsUpdating = true;
			return true;
		}
		return false;
	}

	public void WaitUntilUpdateCompleted(int timeout = 10000)
	{
		Stopwatch stopwatch = new Stopwatch();
		stopwatch.Start();
		while (IsUpdating && (timeout == 0 || stopwatch.Elapsed.TotalMilliseconds < (double)timeout))
		{
			Thread.Sleep(500);
		}
		stopwatch.Stop();
	}

	public void StopReading()
	{
		m_bgwReadWhitelist.CancelAsync();
	}

	private void CompletedReading(object sender, RunWorkerCompletedEventArgs e)
	{
		if (m_restartWhitelistUpdate)
		{
			m_bgwReadWhitelist.RunWorkerAsync();
		}
		else
		{
			IsUpdating = false;
			WhitelistCompleted?.Invoke(this, new WhiteListArgs(m_device, Whitelist, m_bgwReadWhitelist.CancellationPending));
		}
		m_restartWhitelistUpdate = false;
	}

	private void DoWorkRead(object sender, DoWorkEventArgs e)
	{
		//IL_008d: Unknown result type (might be due to invalid IL or missing references)
		if (m_device == null)
		{
			return;
		}
		bool flag = false;
		int num = 0;
		lock (readWhitelistLock)
		{
			Whitelist = new ObservableCollection<ICUWhitelistItem>();
			while (!flag && !m_bgwReadWhitelist.CancellationPending)
			{
				try
				{
					int num2 = Whitelist.Count() + num;
					Whitelist.Count();
					(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("whitelist", $"index={num2}");
					if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
					{
						break;
					}
					dynamic val = new JavaScriptSerializer().Deserialize<object>(tuple.Item3);
					ObservableCollection<ICUWhitelistItem> observableCollection = new ObservableCollection<ICUWhitelistItem>();
					foreach (dynamic item in val)
					{
						string text = item.Key;
						if (text == "version")
						{
							_ = (int)item.Value;
						}
						else
						{
							if (!(text == "whitelist"))
							{
								continue;
							}
							dynamic val2 = item.Value;
							foreach (dynamic item2 in val2)
							{
								if (item2 != null)
								{
									ICUWhitelistItem newItem = new ICUWhitelistItem(item2);
									if (!Whitelist.Any((ICUWhitelistItem a) => a.Tag == newItem.Tag))
									{
										Whitelist.Add(newItem);
										observableCollection.Add(newItem);
									}
								}
							}
						}
					}
					if (observableCollection.Count() == 0)
					{
						if (num <= 128 && num2 + num < MaxTags)
						{
							num += ((m_device.FirmwareVersionNumber < new Version("3.4.0")) ? 15 : 16);
						}
						else
						{
							flag = true;
						}
					}
					else
					{
						WhitelistUpdated?.Invoke(this, new WhiteListArgs(m_device, observableCollection, m_bgwReadWhitelist.CancellationPending));
					}
				}
				catch (Exception ex)
				{
					Logger.Debug(ex, ex.Message);
					break;
				}
				if (Whitelist.Count > MaxTags)
				{
					break;
				}
			}
		}
	}

	public bool SaveToCSV(string fileName)
	{
		if (m_device == null)
		{
			return false;
		}
		List<string> list = new List<string>
		{
			"# Device, " + m_device.Identification,
			$"# Generated, {DateTime.Now}"
		};
		if (!string.IsNullOrEmpty(m_device.MasterTag.Tag))
		{
			list.Add(string.Format("{0},{1},{2},{3}", new object[4]
			{
				m_device.MasterTag.Tag,
				string.Empty,
				99,
				"1970-01-01"
			}));
		}
		foreach (ICUWhitelistItem item in Whitelist)
		{
			list.Add(string.Format("{0},{1},{2},{3}", new object[4]
			{
				item.Tag,
				item.ParentTag,
				(int)item.Status,
				item.ExpiryDate
			}));
		}
		File.WriteAllLines(fileName, list.ToArray());
		return true;
	}

	public void ClearAllEvents()
	{
		WhitelistUpdated = null;
		WhitelistCompleted = null;
	}

	public bool LoadCSV(string fileName, BackgroundWorker wd)
	{
		try
		{
			int num = 0;
			string[] array = File.ReadAllLines(fileName);
			string[] array2 = array;
			foreach (string text in array2)
			{
				wd?.ReportProgress((int)((double)num * 100.0 / (double)array.Count()));
				if (!text.StartsWith("#"))
				{
					string[] array3 = text.Split(new char[1] { ',' });
					if (array3.Count() > 3)
					{
						ICUTagStatus iCUTagStatus = (ICUTagStatus)Enum.Parse(typeof(ICUTagStatus), array3[2]);
						DateTime dateTime = DateTime.Parse(array3[3]);
						bool flag = dateTime.Year > 1970;
						if (iCUTagStatus == ICUTagStatus.MasterCard)
						{
							m_device.MasterTag.Set(array3[0]);
						}
						else
						{
							UpdateOrAdd(array3[0], array3[1], iCUTagStatus, flag ? dateTime.ToString("yyyy-MM-dd") : "1970-01-01");
						}
					}
				}
				num++;
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Debug("Error during csv file import (Error: {Message})", ex.Message);
			return false;
		}
	}
}
