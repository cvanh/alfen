using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Text;
using System.Text.RegularExpressions;
using System.Xml.Linq;
using Serilog;

namespace ICUSettings;

public class ICUBackOffice : ICUBaseObject
{
	private readonly ILogger Logger = Log.ForContext<ICUBackOffice>();

	public List<ICUBackOfficeProperty> m_lstProperties = new List<ICUBackOfficeProperty>();

	private readonly EDSParameterOption m_edsOption;

	public string Title
	{
		get
		{
			return getValue("Title");
		}
		set
		{
			setValue(value, "Title");
		}
	}

	public string TitleNL
	{
		get
		{
			return getValue("TitleNL");
		}
		set
		{
			setValue(value, "TitleNL");
		}
	}

	public string TitleDE
	{
		get
		{
			return getValue("TitleDE");
		}
		set
		{
			setValue(value, "TitleDE");
		}
	}

	public string TitleFR
	{
		get
		{
			return getValue("TitleFR");
		}
		set
		{
			setValue(value, "TitleFR");
		}
	}

	public string Groups
	{
		get
		{
			return getValue("Groups");
		}
		set
		{
			setValue(value, "Groups");
		}
	}

	public int ConnectMethod
	{
		get
		{
			return Convert.ToInt32(getValue("ConnectMethod"));
		}
		set
		{
			setValue(value, "ConnectMethod");
			FireChangedEvent("IsGPRS");
			FireChangedEvent("IsAuto");
		}
	}

	public string APNName
	{
		get
		{
			return getValue("APNName");
		}
		set
		{
			setValue(value, "APNName");
		}
	}

	public string APNUser
	{
		get
		{
			return getValue("APNUser");
		}
		set
		{
			setValue(value, "APNUser");
		}
	}

	public string APNPassword
	{
		get
		{
			return getValue("APNPassword");
		}
		set
		{
			setValue(value, "APNPassword");
		}
	}

	public string SimPin
	{
		get
		{
			return getValue("SimPin");
		}
		set
		{
			setValue(value, "SimPin");
		}
	}

	public string DNS1_1
	{
		get
		{
			return getValue("DNS1_1");
		}
		set
		{
			setValue(value, "DNS1_1");
		}
	}

	public string DNS1_2
	{
		get
		{
			return getValue("DNS1_2");
		}
		set
		{
			setValue(value, "DNS1_2");
		}
	}

	public string DNS2_1
	{
		get
		{
			return getValue("DNS2_1");
		}
		set
		{
			setValue(value, "DNS2_1");
		}
	}

	public string DNS2_2
	{
		get
		{
			return getValue("DNS2_2");
		}
		set
		{
			setValue(value, "DNS2_2");
		}
	}

	public string BackOfficeURL_Domain
	{
		get
		{
			return getValue("BackOfficeURL_Domain");
		}
		set
		{
			setValue(value, "BackOfficeURL_Domain");
		}
	}

	public string BackOfficeURL_Path
	{
		get
		{
			return getValue("BackOfficeURL_Path");
		}
		set
		{
			setValue(value, "BackOfficeURL_Path");
		}
	}

	public string BackOfficeURLwired_Domain
	{
		get
		{
			return getValue("BackOfficeURLwired_Domain");
		}
		set
		{
			setValue(value, "BackOfficeURLwired_Domain");
		}
	}

	public string BackOfficeURLwired_Path
	{
		get
		{
			return getValue("BackOfficeURLwired_Path");
		}
		set
		{
			setValue(value, "BackOfficeURLwired_Path");
		}
	}

	public bool SendStationStatus
	{
		get
		{
			return Convert.ToBoolean(getValue("SendStationStatus"));
		}
		set
		{
			setValue(value, "SendStationStatus");
		}
	}

	public int OfflineNFCAuthorization
	{
		get
		{
			return getValueInt("OfflineNFCAuthorization");
		}
		set
		{
			setValue(value, "OfflineNFCAuthorization");
		}
	}

	public string ProtocolName
	{
		get
		{
			return getValue("ProtocolName");
		}
		set
		{
			setValue(value, "ProtocolName");
		}
	}

	public string ProtocolVersion
	{
		get
		{
			return getValue("ProtocolVersion");
		}
		set
		{
			setValue(value, "ProtocolVersion");
		}
	}

	public int EVDisconnectTimeout
	{
		get
		{
			return getValueInt("EVDisconnectTimeout");
		}
		set
		{
			setValue(value, "EVDisconnectTimeout");
		}
	}

	public int EVDisconnectAction
	{
		get
		{
			return getValueInt("EVDisconnectAction");
		}
		set
		{
			setValue(value, "EVDisconnectAction");
		}
	}

	public bool IntensityAuto
	{
		get
		{
			return Convert.ToBoolean(getValue("IntensityAuto"));
		}
		set
		{
			setValue(value, "IntensityAuto");
		}
	}

	public int IntensityIntensity
	{
		get
		{
			return getValueInt("IntensityIntensity");
		}
		set
		{
			setValue(value, "IntensityIntensity");
		}
	}

	public int TimezoneMinutes
	{
		get
		{
			return getValueInt("TimezoneMinutes");
		}
		set
		{
			setValue(value, "TimezoneMinutes");
		}
	}

	public string Language
	{
		get
		{
			return getValue("Language");
		}
		set
		{
			setValue(value, "Language");
		}
	}

	public int PingPongInterval
	{
		get
		{
			return getValueInt("PingPongInterval");
		}
		set
		{
			setValue(value, "PingPongInterval");
		}
	}

	public bool OnlineNFCAuthorization
	{
		get
		{
			return Convert.ToBoolean(getValue("OnlineNFCAuthorization"));
		}
		set
		{
			setValue(value, "OnlineNFCAuthorization");
		}
	}

	public int OCPP15SmartChargingType
	{
		get
		{
			return getValueInt("OCPP15SmartChargingType");
		}
		set
		{
			setValue(value, "OCPP15SmartChargingType");
		}
	}

	public bool IsGPRS => ConnectMethod >= 2;

	public bool IsAuto
	{
		get
		{
			if (ConnectMethod != 1)
			{
				return ConnectMethod == 3;
			}
			return true;
		}
	}

	public int CentralMeterValueAlignment
	{
		get
		{
			return getValueInt("CentralMeterValueAlignment");
		}
		set
		{
			setValue(value, "CentralMeterValueAlignment");
		}
	}

	public int TransactionMessageAttempts
	{
		get
		{
			return getValueInt("TransactionMessageAttempts");
		}
		set
		{
			setValue(value, "TransactionMessageAttempts");
		}
	}

	public int TransactionMessageRetryInterval
	{
		get
		{
			return getValueInt("TransactionMessageRetryInterval");
		}
		set
		{
			setValue(value, "TransactionMessageRetryInterval");
		}
	}

	public ICUBackOffice OriginalValues { get; set; }

	public string Json
	{
		get
		{
			StringBuilder sbData = new StringBuilder();
			foreach (ICUBackOfficeProperty item in m_lstProperties.Where((ICUBackOfficeProperty a) => a.Type == ICUBackOfficePropetyTypes.Normal))
			{
				AddVariable(ref sbData, item.Name, item.Value);
			}
			StringBuilder sbData2 = new StringBuilder();
			foreach (ICUBackOfficeProperty item2 in m_lstProperties.Where((ICUBackOfficeProperty a) => a.Type == ICUBackOfficePropetyTypes.Extended))
			{
				AddVariable(ref sbData2, item2.Name, item2.Value);
			}
			return string.Format("\n\t{{\"Title\":\"{0}\",\"TitleNL\":\"{1}\",\"TitleDE\":\"{2}\",\"TitleFR\":\"{3}\",\"Groups\":\"{4}\",\"Values\":[{5}\n\t], \"ValuesEx\":[{6}\n\t]}}", new object[7]
			{
				Title,
				TitleNL,
				TitleDE,
				TitleFR,
				Groups,
				sbData.ToString(),
				sbData2.ToString()
			});
		}
	}

	private void InitializeProperties()
	{
		AddPropertyHidden("Title", "", 0u, "Title");
		AddPropertyHidden("TitleNL", "", 0u, "TitleNL");
		AddPropertyHidden("TitleDE", "", 0u, "TitleDE");
		AddPropertyHidden("TitleFR", "", 0u, "TitleFR");
		AddPropertyHidden("Groups", "", 0u, "Groups");
		AddProperty("OD_commConnectMethod", 0, 2127616u, "ConnectMethod");
		AddProperty("OD_gprsAPNname", "", 2162688u, "APNName");
		AddProperty("OD_gprsAPNuser", "", 2162944u, "APNUser");
		AddProperty("OD_gprsAPNpassword", "", 2163200u, "APNPassword");
		AddProperty("OD_commDNS1_1_value", "", 2128129u, "DNS1_1");
		AddProperty("OD_commDNS1_2_value", "", 2129921u, "DNS1_2");
		AddProperty("OD_commDNS2_1_value", "", 2129409u, "DNS2_1");
		AddProperty("OD_commDNS2_2_value", "", 2129665u, "DNS2_2");
		AddProperty("OD_commBackOfficeURL_serverDomainAndPort", "", 2127873u, "BackOfficeURL_Domain");
		AddProperty("OD_commBackOfficeURL_serverPath", "", 2127874u, "BackOfficeURL_Path");
		AddProperty("OD_commBackOfficeURLwired_serverDomainAndPort", "", 2126081u, "BackOfficeURLwired_Domain");
		AddProperty("OD_commBackOfficeURLwired_serverPath", "", 2126082u, "BackOfficeURLwired_Path");
		AddProperty("OD_commSendStationStatus", false, 2134784u, "SendStationStatus");
		AddProperty("OD_mainOfflineNFCAuthorization", 0, 2172672u, "OfflineNFCAuthorization");
		AddProperty("OD_commProtocolName", "", 2130176u, "ProtocolName");
		AddProperty("OD_commProtocolVersion", "", 2130432u, "ProtocolVersion");
		AddProperty("OD_mainEVDisconnectTimeout", 0, 2176512u, "EVDisconnectTimeout");
		AddProperty("OD_mainEVDisconnectAction", 0, 2176768u, "EVDisconnectAction");
		AddPropertyEx("OD_sysIntensity_auto", true, 2121985u, "IntensityAuto");
		AddPropertyEx("OD_sysIntensity_intensity", 100, 2121986u, "IntensityIntensity");
		AddPropertyEx("OD_sysTimeZoneMinutes", 60, 2125312u, "TimezoneMinutes");
		AddPropertyEx("OD_sysLanguage", "nl_NL", 2120960u, "Language");
		AddPropertyEx("OD_commPingPongInterval", 120, 2132480u, "PingPongInterval");
		AddPropertyEx("OD_mainOnlineNFCAuthorization", true, 2178048u, "OnlineNFCAuthorization");
		AddPropertyEx("OD_sysOCPP15SmartCharging_smartChargingType", 0, 2124801u, "OCPP15SmartChargingType");
		AddPropertyEx("OD_gprsSIMpin", "", 2163456u, "SimPin");
		AddPropertyEx("OD_commMeteringAlignment", 1, 2132224u, "CentralMeterValueAlignment");
		AddPropertyEx("OD_commTransactionMessageAttempts", 0, 2135552u, "TransactionMessageAttempts");
		AddPropertyEx("OD_commTransactionMessageRetryInterval", 60, 2135808u, "TransactionMessageRetryInterval");
	}

	public static List<uint> GetPropertyIds()
	{
		return new List<uint>
		{
			2127616u, 2162688u, 2162944u, 2163200u, 2163456u, 2128129u, 2129921u, 2129409u, 2129665u, 2127873u,
			2127874u, 2126081u, 2126082u, 2134784u, 2172672u, 2130176u, 2130432u, 2176512u, 2176768u, 2121985u,
			2121986u, 2125312u, 2120960u, 2132480u, 2178048u, 2124801u, 2132224u, 2135552u, 2135808u
		};
	}

	private bool AddPropertyInternal(ICUBackOfficePropetyTypes type, string name, object value, uint propnumber, string propname)
	{
		try
		{
			ICUBackOfficeProperty iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.Name == name);
			if (iCUBackOfficeProperty == null)
			{
				iCUBackOfficeProperty = new ICUBackOfficeProperty
				{
					Type = type,
					Name = name,
					Value = value,
					PropertyName = propname,
					PropertyNumber = propnumber
				};
				m_lstProperties.Add(iCUBackOfficeProperty);
			}
		}
		catch (Exception ex)
		{
			Logger.Error("SetProperty '{Name}', failed to set property! Error: {Message}", name, ex.Message);
		}
		return true;
	}

	private bool AddProperty(string name, object value, uint propnumber, string propname)
	{
		return AddPropertyInternal(ICUBackOfficePropetyTypes.Normal, name, value, propnumber, propname);
	}

	private bool AddPropertyEx(string name, object value, uint propnumber, string propname)
	{
		return AddPropertyInternal(ICUBackOfficePropetyTypes.Extended, name, value, propnumber, propname);
	}

	private bool AddPropertyHidden(string name, object value, uint propnumber, string propname)
	{
		return AddPropertyInternal(ICUBackOfficePropetyTypes.Hidden, name, value, propnumber, propname);
	}

	private bool SetProperty(string name, object value)
	{
		try
		{
			ICUBackOfficeProperty iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.Name == name);
			if (iCUBackOfficeProperty == null)
			{
				return false;
			}
			Type type = iCUBackOfficeProperty.Value.GetType();
			if (type.Equals(typeof(string)))
			{
				iCUBackOfficeProperty.Value = Convert.ToString(value).Trim();
			}
			else if (type.Equals(typeof(int)))
			{
				iCUBackOfficeProperty.Value = Convert.ToInt32(value);
			}
			else if (type.Equals(typeof(bool)))
			{
				if (value.ToString() == "0")
				{
					iCUBackOfficeProperty.Value = false;
				}
				else if (value.ToString() == "1")
				{
					iCUBackOfficeProperty.Value = true;
				}
				else
				{
					iCUBackOfficeProperty.Value = Convert.ToBoolean(value);
				}
			}
			else
			{
				Logger.Error("SetProperty '{Name}' called with unexpected type, please add the type ('{Type}')!", name, type.ToString());
			}
		}
		catch (Exception ex)
		{
			Logger.Error("SetProperty '{Name}', failed to set property! Error: {Error}", name, ex.Message);
		}
		return true;
	}

	public ICUBackOfficeProperty GetProperty(string name)
	{
		return m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.Name == name);
	}

	public ICUBackOfficeProperty GetProperty(uint propnumber)
	{
		return m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyNumber == propnumber);
	}

	public string getValue([CallerMemberName] string caller = null)
	{
		return m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyName == caller)?.Value.ToString();
	}

	public int getValueInt([CallerMemberName] string caller = null)
	{
		ICUBackOfficeProperty iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyName == caller);
		if (iCUBackOfficeProperty == null)
		{
			return 0;
		}
		return Convert.ToInt32(iCUBackOfficeProperty.Value);
	}

	private void setValue(object value, [CallerMemberName] string caller = null)
	{
		ICUBackOfficeProperty iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyName == caller);
		if (iCUBackOfficeProperty != null)
		{
			object value2 = iCUBackOfficeProperty.Value;
			iCUBackOfficeProperty.Value = value;
			if (value2 != value)
			{
				FireChangedEvent(caller);
			}
		}
	}

	public ICUBackOffice()
		: base(fDirty: true)
	{
		InitializeProperties();
	}

	public ICUBackOffice(dynamic obj)
	{
		InitializeProperties();
		LoadFromDynamic(obj);
	}

	public ICUBackOffice(EDSParameterOption option)
	{
		InitializeProperties();
		m_edsOption = option;
		ParseElement(option.XElement);
	}

	private void LoadFromDynamic(dynamic obj)
	{
		SetProperty("Title", obj["Title"]);
		SetProperty("TitleNL", obj["TitleNL"]);
		SetProperty("TitleDE", obj["TitleDE"]);
		SetProperty("TitleFR", obj["TitleFR"]);
		SetProperty("Groups", obj["Groups"]);
		foreach (dynamic item in obj["Values"])
		{
			SetProperty(item["Key"], item["Value"]);
		}
		if (obj.ContainsKey("ValuesEx"))
		{
			foreach (dynamic item2 in obj["ValuesEx"])
			{
				SetProperty(item2["Key"], item2["Value"]);
			}
		}
		ICUBackOffice iCUBackOffice = new ICUBackOffice();
		iCUBackOffice.CopyFrom(this);
		OriginalValues = iCUBackOffice;
	}

	public void ParseElement(XElement xelem)
	{
		SetProperty("Title", getAttribute(xelem, "Value"));
		List<XElement> list = xelem.Descendants("Title").ToList();
		if (list.Count > 0)
		{
			SetProperty("Title", getTitle(list, ""));
			SetProperty("TitleNL", getTitle(list, "nl"));
			SetProperty("TitleDE", getTitle(list, "de"));
			SetProperty("TitleFR", getTitle(list, "fr"));
			List<XElement> list2 = xelem.Descendants("Connection").ToList();
			if (list2.Count > 0)
			{
				foreach (XElement item in list2)
				{
					Match match = Regex.Match(item.Attribute("Id").Value.ToString().Trim().ToLowerInvariant(), "([0-9]).([0-9a-fA-F]+)(sub([0-9a-fA-F]+))?");
					int result = 0;
					uint fullId = 0u;
					if (match.Groups.Count > 4)
					{
						if (int.TryParse(match.Groups[2].Value, NumberStyles.HexNumber, null, out result))
						{
							fullId |= (uint)(result << 8);
						}
						if (int.TryParse(match.Groups[4].Value, NumberStyles.HexNumber, null, out result))
						{
							fullId |= (uint)(result & 0xFF);
						}
					}
					ICUBackOfficeProperty iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyNumber == fullId);
					if (iCUBackOfficeProperty != null)
					{
						string text = "";
						if (item.Descendants("Property") != null && item.Descendants("Property").Attributes("Value") != null)
						{
							text = item.Descendants("Property").Attributes("Value").FirstOrDefault()
								.Value.ToString();
						}
						if (fullId == 2127616 && text == "99")
						{
							text = "3";
						}
						SetProperty(iCUBackOfficeProperty.Name, text);
					}
					else if (fullId == 2120192)
					{
						int num = 0;
						if (item.Descendants("Property") != null && item.Descendants("Property").Attributes("Value") != null)
						{
							num = Convert.ToInt32(item.Descendants("Property").Attributes("Value").FirstOrDefault()
								.Value.ToString());
						}
						iCUBackOfficeProperty = m_lstProperties.FirstOrDefault((ICUBackOfficeProperty a) => a.PropertyNumber == 2125312);
						if (iCUBackOfficeProperty != null)
						{
							SetProperty(iCUBackOfficeProperty.Name, num * 6);
						}
					}
					else if (fullId != 2128128 && fullId != 2129920 && fullId != 2119936 && fullId != 2127104 && fullId != 2127105)
					{
						Logger.Debug("Unknown connected property: {Id}", $"{fullId:X6}");
					}
				}
			}
		}
		SetProperty("Groups", getAttribute(xelem, "Group"));
		ICUBackOffice iCUBackOffice = new ICUBackOffice();
		iCUBackOffice.CopyFrom(this);
		OriginalValues = iCUBackOffice;
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		foreach (ICUBackOfficeProperty lstProperty in m_lstProperties)
		{
			ICUBackOfficeProperty property = OriginalValues.GetProperty(lstProperty.Name);
			if (property != null && property.Value != null && lstProperty.Value.ToString() != property.Value.ToString())
			{
				Logger.Debug("Difference detected for '{Name}': Old '{OldValue}' => New '{NewValue}'", lstProperty.Name, property.Value.ToString(), lstProperty.Value.ToString());
				return true;
			}
		}
		return false;
	}

	public void CopyFrom(ICUBackOffice other, bool allProperties = false)
	{
		if (other == null)
		{
			return;
		}
		m_lstProperties.Clear();
		foreach (ICUBackOfficeProperty lstProperty in other.m_lstProperties)
		{
			if (allProperties)
			{
				AddPropertyInternal(lstProperty.Type, lstProperty.Name, lstProperty.Value, lstProperty.PropertyNumber, lstProperty.PropertyName);
			}
			else if (lstProperty.Type == ICUBackOfficePropetyTypes.Normal)
			{
				AddProperty(lstProperty.Name, lstProperty.Value, lstProperty.PropertyNumber, lstProperty.PropertyName);
			}
			else if (lstProperty.Type == ICUBackOfficePropetyTypes.Extended)
			{
				AddPropertyEx(lstProperty.Name, lstProperty.Value, lstProperty.PropertyNumber, lstProperty.PropertyName);
			}
			else
			{
				_ = lstProperty.Type;
			}
			FireChangedEvent(lstProperty.PropertyName);
		}
		CheckDirty();
	}

	private string getTitle(List<XElement> xelemTitles, string lang)
	{
		XElement xElement = xelemTitles.FirstOrDefault((XElement a) => a.Attribute("Lang").Value.ToString().Trim().ToLowerInvariant() == lang);
		if (xElement != null)
		{
			return xElement.Value.ToString();
		}
		return string.Empty;
	}

	private void AddVariable(ref StringBuilder sbData, string odName, object value)
	{
		if (sbData.Length > 0)
		{
			sbData.Append(',');
		}
		string arg = Regex.Replace(value.ToString(), "\\r\\n?|\\n", ", ");
		sbData.AppendFormat("\n\t\t{{\"Key\":\"{0}\",\"Value\":\"{1}\"}}", odName, arg);
	}

	public void Commit()
	{
		if (OriginalValues != null)
		{
			OriginalValues.CopyFrom(this);
		}
		CheckDirty();
	}

	public void Rollback()
	{
		CopyFrom(OriginalValues);
	}

	public object GetValue(uint combinedPropId)
	{
		ushort usId = (ushort)(combinedPropId >> 8);
		byte bSubId = (byte)(combinedPropId & 0xFF);
		return GetValue(usId, bSubId);
	}

	public object GetValue(ushort usId, byte bSubId)
	{
		uint propnumber = (uint)((usId << 8) + bSubId);
		return GetProperty(propnumber)?.Value;
	}
}
