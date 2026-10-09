using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Text;
using System.Text.RegularExpressions;
using Serilog;

namespace ICUSettings;

public class ICUPMBackOffice : ICUBaseObject
{
	private static readonly ILogger Logger = Logger.ForContext<ICUPMBackOffice>();

	public List<ICUPMProperty> m_lstProperties = new List<ICUPMProperty>();

	private List<ICUPMProperty> m_lstOriginalProperties = new List<ICUPMProperty>();

	private bool m_fLANEnabled = true;

	private bool m_fOriginalLANEnabled;

	private bool m_fGPRSEnabled = true;

	private bool m_fOriginalGPRSEnabled;

	public List<ICUPMProperty> Properties => m_lstProperties;

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

	public bool IsLANEnabled
	{
		get
		{
			return m_fLANEnabled;
		}
		set
		{
			SetPropertyField(ref m_fLANEnabled, value, "IsLANEnabled");
		}
	}

	public bool IsGPRSEnabled
	{
		get
		{
			return m_fGPRSEnabled;
		}
		set
		{
			SetPropertyField(ref m_fGPRSEnabled, value, "IsGPRSEnabled");
		}
	}

	public string Json
	{
		get
		{
			StringBuilder sbData = new StringBuilder();
			foreach (ICUPMProperty lstProperty in m_lstProperties)
			{
				AddVariable(ref sbData, lstProperty.Name, lstProperty.Value);
			}
			return string.Format("\n\t{{\"Title\":\"{0}\",\"LANEnabled\":{1},\"GPRSEnabled\":{2},\"Values\":[{3}\n\t] }}", new object[4]
			{
				Title,
				IsLANEnabled.ToString().ToLowerInvariant(),
				IsGPRSEnabled.ToString().ToLowerInvariant(),
				sbData.ToString()
			});
		}
	}

	public ICUPMBackOffice()
		: base(fDirty: true)
	{
		InitializeProperties();
	}

	public ICUPMBackOffice(dynamic obj)
	{
		InitializeProperties();
		LoadFromDynamic(obj);
	}

	private void InitializeProperties()
	{
		AddProperty("Title", "", 0u, "Title");
		AddProperty("OD_gprsAPNname", "", 2162688u, "APNName");
		AddProperty("OD_gprsAPNuser", "", 2162944u, "APNUser");
		AddProperty("OD_gprsAPNpassword", "", 2163200u, "APNPassword");
		AddProperty("OD_gprsSIMpin", "", 2163456u, "SimPin");
		AddProperty("OD_commDNS1_1_value", "", 2128129u, "DNS1_1");
		AddProperty("OD_commDNS1_2_value", "", 2129921u, "DNS1_2");
		AddProperty("OD_commDNS2_1_value", "", 2129409u, "DNS2_1");
		AddProperty("OD_commDNS2_2_value", "", 2129665u, "DNS2_2");
		AddProperty("OD_commBackOfficeURL_serverDomainAndPort", "", 2127873u, "BackOfficeURL_Domain");
		AddProperty("OD_commBackOfficeURL_serverPath", "", 2127874u, "BackOfficeURL_Path");
		AddProperty("OD_commBackOfficeURLwired_serverDomainAndPort", "", 2126081u, "BackOfficeURLwired_Domain");
		AddProperty("OD_commBackOfficeURLwired_serverPath", "", 2126082u, "BackOfficeURLwired_Path");
		AddProperty("OD_commProtocolName", "ocpp/json", 2130176u, "ProtocolName");
		AddProperty("OD_commProtocolVersion", "1.6", 2130432u, "ProtocolVersion");
		AddProperty("OD_mainEVDisconnectTimeout", 10, 2176512u, "EVDisconnectTimeout");
		AddProperty("OD_sysIntensity_auto", true, 2121985u, "IntensityAuto");
		AddProperty("OD_sysIntensity_intensity", 100, 2121986u, "IntensityIntensity");
		AddProperty("OD_sysTimeZoneMinutes", 60, 2125312u, "TimezoneMinutes");
		AddProperty("OD_sysOCPP15SmartCharging_smartChargingType", 0, 2124801u, "OCPP15SmartChargingType");
		AddProperty("OD_commSendStationStatus", true, 2134784u, "SendStationStatus");
		AddProperty("OD_commTransactionMessageAttempts", 0, 2135552u, "TransactionMessageAttempts");
		AddProperty("OD_commTransactionMessageRetryInterval", 60, 2135808u, "TransactionMessageRetryInterval");
	}

	private bool AddProperty(string name, object value, uint propnumber, string propname)
	{
		try
		{
			ICUPMProperty iCUPMProperty = m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.Name == name);
			if (iCUPMProperty == null)
			{
				iCUPMProperty = new ICUPMProperty();
				m_lstProperties.Add(iCUPMProperty);
			}
			if (iCUPMProperty != null)
			{
				iCUPMProperty.Name = name;
				iCUPMProperty.Value = value;
				iCUPMProperty.PropertyName = propname;
				iCUPMProperty.PropertyNumber = propnumber;
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, "SetProperty '{Name}', failed to set property! Error: {Message}", name, ex.Message);
		}
		return true;
	}

	private bool SetProperty(string name, object value)
	{
		try
		{
			ICUPMProperty iCUPMProperty = m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.Name == name);
			if (iCUPMProperty == null)
			{
				return false;
			}
			Type type = iCUPMProperty.Value.GetType();
			if (type.Equals(typeof(string)))
			{
				iCUPMProperty.Value = Convert.ToString(value).Trim();
			}
			else if (type.Equals(typeof(int)))
			{
				iCUPMProperty.Value = Convert.ToInt32(value);
			}
			else if (type.Equals(typeof(bool)))
			{
				if (value.ToString() == "0")
				{
					iCUPMProperty.Value = false;
				}
				else if (value.ToString() == "1")
				{
					iCUPMProperty.Value = true;
				}
				else
				{
					iCUPMProperty.Value = Convert.ToBoolean(value);
				}
			}
			else
			{
				Logger.Debug("SetProperty '{Name}' called with unexpected type, please add the type ('{Type}')!", name, type.ToString());
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, "SetProperty '{Name}', failed to set property! Error: {Message}", name, ex.Message);
		}
		return true;
	}

	public ICUPMProperty GetProperty(string name)
	{
		return m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.Name == name);
	}

	public ICUPMProperty GetProperty(uint propnumber)
	{
		return m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.PropertyNumber == propnumber);
	}

	public string getValue([CallerMemberName] string caller = null)
	{
		return m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.PropertyName == caller)?.Value.ToString();
	}

	public int getValueInt([CallerMemberName] string caller = null)
	{
		ICUPMProperty iCUPMProperty = m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.PropertyName == caller);
		if (iCUPMProperty == null)
		{
			return 0;
		}
		return Convert.ToInt32(iCUPMProperty.Value);
	}

	private void setValue(object value, [CallerMemberName] string caller = null)
	{
		ICUPMProperty iCUPMProperty = m_lstProperties.FirstOrDefault((ICUPMProperty a) => a.PropertyName == caller);
		if (iCUPMProperty != null)
		{
			object value2 = iCUPMProperty.Value;
			iCUPMProperty.Value = value;
			if (value2 != value)
			{
				FireChangedEvent(caller);
			}
		}
	}

	private void LoadFromDynamic(dynamic obj)
	{
		SetProperty("Title", obj["Title"]);
		if (((IDictionary<string, object>)obj).ContainsKey("GPRSEnabled"))
		{
			m_fGPRSEnabled = Convert.ToBoolean(obj["GPRSEnabled"]);
		}
		if (((IDictionary<string, object>)obj).ContainsKey("LANEnabled"))
		{
			m_fLANEnabled = Convert.ToBoolean(obj["LANEnabled"]);
		}
		m_fOriginalGPRSEnabled = m_fGPRSEnabled;
		m_fOriginalLANEnabled = m_fLANEnabled;
		foreach (dynamic item in obj["Values"])
		{
			SetProperty(item["Key"], item["Value"]);
		}
		m_lstOriginalProperties = m_lstProperties.Select((ICUPMProperty item) => (ICUPMProperty)item.Clone()).ToList();
	}

	protected override bool OnCheckDirty()
	{
		if (m_lstOriginalProperties == null)
		{
			return true;
		}
		foreach (ICUPMProperty bop in m_lstProperties)
		{
			ICUPMProperty iCUPMProperty = m_lstOriginalProperties.FirstOrDefault((ICUPMProperty a) => a.PropertyNumber == bop.PropertyNumber);
			if (iCUPMProperty != null && iCUPMProperty.Value != null && bop.Value.ToString() != iCUPMProperty.Value.ToString())
			{
				Logger.Debug("Difference detected for '{Name}': Old '{OldValue}' => New '{NewValue}'", bop.Name, iCUPMProperty.Value.ToString(), bop.Value.ToString());
				return true;
			}
		}
		if (m_fLANEnabled != m_fOriginalLANEnabled)
		{
			return true;
		}
		if (m_fGPRSEnabled != m_fOriginalGPRSEnabled)
		{
			return true;
		}
		return false;
	}

	public void CopyFrom(ICUPMBackOffice other, bool allProperties = false)
	{
		if (other == null)
		{
			return;
		}
		m_lstProperties.Clear();
		foreach (ICUPMProperty lstProperty in other.m_lstProperties)
		{
			AddProperty(lstProperty.Name, lstProperty.Value, lstProperty.PropertyNumber, lstProperty.PropertyName);
			FireChangedEvent(lstProperty.PropertyName);
		}
		m_fLANEnabled = other.IsLANEnabled;
		m_fGPRSEnabled = other.IsGPRSEnabled;
		CheckDirty();
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
		m_lstOriginalProperties = m_lstProperties.Select((ICUPMProperty item) => (ICUPMProperty)item.Clone()).ToList();
		m_fOriginalLANEnabled = m_fLANEnabled;
		m_fOriginalGPRSEnabled = m_fGPRSEnabled;
		CheckDirty();
	}

	public void Rollback()
	{
		m_lstProperties = m_lstOriginalProperties.Select((ICUPMProperty item) => (ICUPMProperty)item.Clone()).ToList();
		m_fLANEnabled = m_fOriginalLANEnabled;
		m_fGPRSEnabled = m_fOriginalGPRSEnabled;
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
