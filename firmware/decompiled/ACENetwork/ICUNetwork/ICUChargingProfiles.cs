using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Net;
using Newtonsoft.Json;
using Serilog;

namespace ICUNetwork;

public class ICUChargingProfiles
{
	private readonly ILogger Logger = Log.ForContext<ICUChargingProfiles>();

	private ICULanDevice m_device;

	private const int s_nRequestLongTimeOut = 10000;

	public static int s_idUKSmartCharging = -19061964;

	private const int s_smartChargingBlock1Start = 8;

	private const int s_smartChargingBlock1End = 11;

	private const int s_smartChargingBlock2Start = 16;

	private const int s_smartChargingBlock2End = 22;

	private const int s_secondsPerHour = 3600;

	private const int s_secondsPerDay = 86400;

	public ObservableCollection<ICUChargingProfile> ChargingProfiles { get; private set; } = new ObservableCollection<ICUChargingProfile>();

	public bool IsChargingProfileSupported { get; set; }

	public bool IsUKSmartChargingProfileInstalled { get; set; }

	public ICUChargingProfiles(ICULanDevice device)
	{
		m_device = device;
		IsChargingProfileSupported = false;
		IsUKSmartChargingProfileInstalled = false;
		ChargingProfiles.Clear();
	}

	public void Initialize()
	{
		if (m_device == null)
		{
			return;
		}
		(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("chargingprofiles", "id_list", null, 10000, 1);
		if (tuple.Item2 != HttpStatusCode.OK)
		{
			return;
		}
		IsChargingProfileSupported = true;
		string item = tuple.Item3;
		try
		{
			dynamic val = JsonConvert.DeserializeObject(item);
			foreach (dynamic item2 in val.ChargingProfileIDs)
			{
				if (Convert.ToInt32(item2.Value) == s_idUKSmartCharging)
				{
					IsUKSmartChargingProfileInstalled = true;
				}
			}
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "");
		}
	}

	~ICUChargingProfiles()
	{
		m_device = null;
		ChargingProfiles.Clear();
	}

	public bool SupportsChargingProfiles()
	{
		if (m_device != null)
		{
			return m_device.ExecuteWebRequest("chargingprofiles", "id_list", null, 10000, 1).HttpStatusCode == HttpStatusCode.OK;
		}
		return false;
	}

	public bool ClearAll()
	{
		if (m_device == null)
		{
			return false;
		}
		bool result = false;
		if (m_device.ExecuteWebRequest("chargingprofiles", "clear=all", "", 10000, 1).HttpStatusCode == HttpStatusCode.OK)
		{
			Logger.Debug("Cleared all charging profiles");
			result = true;
		}
		ChargingProfiles.Clear();
		return result;
	}

	public bool Clear(int profileId)
	{
		if (m_device == null)
		{
			return false;
		}
		bool result = false;
		if (m_device.ExecuteWebRequest("chargingprofiles", $"clear={profileId}", "", 10000, 1).HttpStatusCode == HttpStatusCode.OK)
		{
			Logger.Debug("Cleared charging profile with profileId: {ProfileId}", profileId);
			result = true;
		}
		ChargingProfiles = new ObservableCollection<ICUChargingProfile>(ChargingProfiles.Where((ICUChargingProfile a) => a.ChargingProfileId != profileId));
		return result;
	}

	public bool ClearUKSmartChargingProfile()
	{
		bool flag = Clear(s_idUKSmartCharging);
		if (flag)
		{
			IsUKSmartChargingProfileInstalled = false;
		}
		return flag;
	}

	public bool GetChargingProfile(int profileId)
	{
		if (m_device == null)
		{
			return false;
		}
		bool flag = false;
		try
		{
			(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("chargingprofiles", $"cpid={profileId}", null, 10000, 1);
			if (tuple.Item2 == HttpStatusCode.NotFound)
			{
				Logger.Debug("No charging profile for cpid={ProfileId}", profileId);
				return false;
			}
			if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
			{
				Logger.Debug("GetChargingProfile API web request failed for cpid={ProfileId}", profileId);
				return false;
			}
			dynamic val = JsonConvert.DeserializeObject(tuple.Item3);
			if (val.version == 2)
			{
				int num = 0;
				foreach (dynamic item in val.Profile)
				{
					num = (int)item.connectorId;
					foreach (dynamic item2 in item.csChargingProfiles)
					{
						ICUChargingProfile newProfile = new ICUChargingProfile(num, item2);
						ICUChargingProfile iCUChargingProfile = ChargingProfiles.FirstOrDefault((ICUChargingProfile tx) => tx.ChargingProfileId == newProfile.ChargingProfileId);
						if (iCUChargingProfile != null)
						{
							ChargingProfiles.Remove(iCUChargingProfile);
						}
						ChargingProfiles.Add(newProfile);
					}
				}
			}
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "");
			flag = true;
		}
		return !flag;
	}

	public List<int> GetAllChargingProfileIds()
	{
		List<int> list = new List<int>();
		if (m_device != null)
		{
			(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("chargingprofiles", "id_list", null, 10000, 1);
			if (tuple.Item2 == HttpStatusCode.OK)
			{
				string item = tuple.Item3;
				try
				{
					dynamic val = JsonConvert.DeserializeObject(item);
					foreach (dynamic item2 in val.ChargingProfileIDs)
					{
						list.Add(Convert.ToInt32(item2.Value));
					}
				}
				catch (Exception exception)
				{
					Logger.Error(exception, "");
				}
			}
		}
		return list;
	}

	public bool AddUkSmartChargingProfile()
	{
		if (m_device != null)
		{
			DateTime utcNow = DateTime.UtcNow;
			int num = (int)(7 + (utcNow.DayOfWeek - 1)) % 7;
			string text = utcNow.AddDays(-1 * num).Date.ToString("yyyy-MM-ddT00:00:00Z");
			string text2 = "{\"connectorId\":0,\"csChargingProfiles\":{";
			text2 += $"\"chargingProfileId\":{s_idUKSmartCharging},\"chargingProfileKind\":\"Recurring\",";
			text2 += "\"recurrencyKind\":\"Weekly\",\"chargingProfilePurpose\":\"ChargingStationExternalConstraints\",";
			text2 += "\"useLocalTime\":true, \"useRandomisedDelay\":true,\"stackLevel\":1,\"chargingSchedule\":{";
			text2 = text2 + "\"startSchedule\":\"" + text + "\",\"chargingRateUnit\":\"A\",";
			int num2 = 28800;
			int num3 = 39600;
			int num4 = 57600;
			int num5 = 79200;
			int num6 = 86400;
			int num7 = 0;
			int num8 = 32;
			int num9 = 0;
			List<string> list = new List<string>();
			list.Add($"{{\"startPeriod\":0,\"limit\":{num8}}}");
			for (int i = 0; i < 5; i++)
			{
				list.Add($"{{\"startPeriod\":{num7 + num2},\"limit\":{num9}}}");
				list.Add($"{{\"startPeriod\":{num7 + num3},\"limit\":{num8}}}");
				list.Add($"{{\"startPeriod\":{num7 + num4},\"limit\":{num9}}}");
				list.Add($"{{\"startPeriod\":{num7 + num5},\"limit\":{num8}}}");
				num7 += num6;
			}
			string text3 = string.Join(",", list);
			text2 = text2 + "\"chargingSchedulePeriod\":[" + text3 + "]";
			text2 += "}}}";
			(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("chargingprofiles", "add=", text2, 10000, 3);
			if (tuple.Item2 != HttpStatusCode.OK)
			{
				Logger.Error("Failed to add a new GetChargingProfile. Error: {ResponseData}", tuple.Item3);
				return false;
			}
			Logger.Debug("GetChargingProfile added to CS. Response: {ResponseData}", tuple.Item3);
			IsUKSmartChargingProfileInstalled = true;
		}
		return true;
	}
}
