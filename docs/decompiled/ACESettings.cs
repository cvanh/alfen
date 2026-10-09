using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Net;
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.Versioning;
using System.Security.Cryptography;
using System.Text;
using System.Text.RegularExpressions;
using System.Web.Script.Serialization;
using System.Xml;
using System.Xml.Linq;
using Serilog;

[assembly: CompilationRelaxations(8)]
[assembly: RuntimeCompatibility(WrapNonExceptionThrows = true)]
[assembly: Debuggable(DebuggableAttribute.DebuggingModes.IgnoreSymbolStoreSequencePoints)]
[assembly: TargetFramework(".NETFramework,Version=v4.8.1", FrameworkDisplayName = ".NET Framework 4.8.1")]
[assembly: AssemblyCompany("Alfen N.V.")]
[assembly: AssemblyConfiguration("Release")]
[assembly: AssemblyCopyright("Copyright © 2025")]
[assembly: AssemblyDescription("Alfen Charge Equipment Settings")]
[assembly: AssemblyFileVersion("4.4.1.434")]
[assembly: AssemblyInformationalVersion("4.4.1.434+13315c34f5816fff71821aafad09646ceb3cfbf5")]
[assembly: AssemblyProduct("ACESettings")]
[assembly: AssemblyTitle("ACESettings")]
[assembly: AssemblyVersion("4.4.1.434")]
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
public class EDSParameter
{
	private static readonly string[] seperator = new string[1] { "sub" };

	public int Id { get; set; }

	public int SubId { get; set; }

	public string Name { get; set; }

	public string Title { get; set; }

	public int DataType { get; set; }

	public string Units { get; set; }

	public bool ReadWrite { get; set; }

	public ulong MaxLength { get; set; }

	public List<EDSParameterOption> Options { get; set; }

	public EDSParameter(XElement parameter)
	{
		int result = 0;
		string[] array = parameter.Attribute("Id").Value.Split(seperator, StringSplitOptions.None);
		if (array.Length != 0)
		{
			if (int.TryParse(array[0], NumberStyles.HexNumber, null, out result))
			{
				Id = result;
			}
			if (array.Length > 1 && int.TryParse(array[1], NumberStyles.HexNumber, null, out result))
			{
				SubId = result;
			}
		}
		Name = parameter.Attribute("ParameterName").Value;
		string text = parameter.Attribute("DataType").Value;
		if (text.StartsWith("0x", StringComparison.OrdinalIgnoreCase))
		{
			text = text.Substring(2);
		}
		if (int.TryParse(text, NumberStyles.HexNumber, null, out result))
		{
			DataType = result;
		}
		string value = parameter.Attribute("AccessType").Value;
		ReadWrite = value.Equals("rw", StringComparison.OrdinalIgnoreCase);
		if (parameter.Descendants("Title").Any())
		{
			Title = parameter.Descendants("Title").First().Value;
		}
		if (parameter.Descendants("Units").Any())
		{
			Units = parameter.Descendants("Units").First().Value;
		}
		if (parameter.Attribute("Length") != null)
		{
			MaxLength = Convert.ToUInt64(parameter.Attribute("Length").Value, CultureInfo.InvariantCulture);
		}
		else
		{
			MaxLength = 256uL;
		}
		IEnumerable<XElement> enumerable = parameter.Descendants("Option");
		if (!enumerable.Any())
		{
			return;
		}
		Options = new List<EDSParameterOption>();
		int num = 0;
		foreach (XElement item in enumerable)
		{
			Options.Add(new EDSParameterOption(num++, item));
		}
	}
}
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
public class EncryptDecrypt
{
	private readonly string saltValue = "s@1tVaLue";

	private readonly string hashAlgorithm = "SHA1";

	private readonly int passwordIterations = 2;

	private readonly string initVector = "@1B2c3D4e5F6g7H8";

	private readonly int keySize = 256;

	public string Encrypt(string plainText, string passPhrase)
	{
		byte[] bytes = Encoding.ASCII.GetBytes(initVector);
		byte[] bytes2 = Encoding.ASCII.GetBytes(saltValue);
		byte[] bytes3 = Encoding.UTF8.GetBytes(plainText);
		byte[] bytes4 = new PasswordDeriveBytes(passPhrase, bytes2, hashAlgorithm, passwordIterations).GetBytes(keySize / 8);
		ICryptoTransform transform = new RijndaelManaged
		{
			Mode = CipherMode.CBC
		}.CreateEncryptor(bytes4, bytes);
		MemoryStream memoryStream = new MemoryStream();
		CryptoStream cryptoStream = new CryptoStream(memoryStream, transform, CryptoStreamMode.Write);
		cryptoStream.Write(bytes3, 0, bytes3.Length);
		cryptoStream.FlushFinalBlock();
		byte[] inArray = memoryStream.ToArray();
		memoryStream.Close();
		cryptoStream.Close();
		return Convert.ToBase64String(inArray);
	}

	public string Decrypt(string cipherText, string passPhrase)
	{
		if (string.IsNullOrEmpty(cipherText))
		{
			return string.Empty;
		}
		byte[] bytes = Encoding.ASCII.GetBytes(initVector);
		byte[] bytes2 = Encoding.ASCII.GetBytes(saltValue);
		byte[] array = Convert.FromBase64String(cipherText);
		byte[] bytes3 = new PasswordDeriveBytes(passPhrase, bytes2, hashAlgorithm, passwordIterations).GetBytes(keySize / 8);
		ICryptoTransform transform = new RijndaelManaged
		{
			Mode = CipherMode.CBC
		}.CreateDecryptor(bytes3, bytes);
		MemoryStream memoryStream = new MemoryStream(array);
		CryptoStream cryptoStream = new CryptoStream(memoryStream, transform, CryptoStreamMode.Read);
		byte[] array2 = new byte[array.Length];
		int count = cryptoStream.Read(array2, 0, array2.Length);
		memoryStream.Close();
		cryptoStream.Close();
		return Encoding.UTF8.GetString(array2, 0, count);
	}
}
public enum ICUBackOfficePropetyTypes
{
	Hidden,
	Normal,
	Extended
}
public class ICUBackOfficeProperty
{
	public string Name { get; set; }

	public string PropertyName { get; set; }

	public object Value { get; set; }

	public uint PropertyNumber { get; set; }

	public ICUBackOfficePropetyTypes Type { get; set; }
}
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
public class ICUBaseObject : INotifyPropertyChanged
{
	private bool m_fDirty;

	public bool Dirty
	{
		get
		{
			return m_fDirty;
		}
		set
		{
			m_fDirty = value;
		}
	}

	public event PropertyChangedEventHandler PropertyChanged;

	public ICUBaseObject(bool fDirty = false)
	{
		m_fDirty = fDirty;
	}

	protected virtual bool OnCheckDirty()
	{
		return false;
	}

	public void CheckDirty()
	{
		m_fDirty = OnCheckDirty();
		PropertyChanged?.Invoke(this, new PropertyChangedEventArgs("Dirty"));
	}

	protected void FireChangedEvent(string propertyName)
	{
		PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
		CheckDirty();
	}

	protected void SetPropertyField<T>(ref T field, T newValue, [CallerMemberName] string caller = null)
	{
		if (!EqualityComparer<T>.Default.Equals(field, newValue))
		{
			field = newValue;
			FireChangedEvent(caller);
		}
	}

	protected string getAttribute(XElement xelem, string attrName)
	{
		XAttribute xAttribute = xelem.Attribute(attrName);
		if (xAttribute != null)
		{
			return xAttribute.Value.ToString();
		}
		return string.Empty;
	}

	protected string setAttribute(XElement xelem, string attrName, string newValue)
	{
		XAttribute xAttribute = xelem.Attribute(attrName);
		if (xAttribute != null)
		{
			return xAttribute.Value = newValue;
		}
		return string.Empty;
	}

	protected string ValidString(string value)
	{
		value = value ?? string.Empty;
		return string.Format("\"{0}\"", Regex.Replace(value, "\\r\\n?|\\n", "<br>"));
	}
}
public enum ICUEncryptionType
{
	encryptNone,
	encryptBase64,
	encryptRijndael,
	encryptRijndaelHashed
}
public class ICUConfig : INotifyPropertyChanged
{
	private readonly ILogger Logger = Log.ForContext<ICUConfig>();

	protected static string s_fileID = "ICUConfigFile";

	protected string m_sVersion = string.Empty;

	protected string m_sDateTime = string.Empty;

	protected string m_sOriginalVersion = string.Empty;

	protected ObservableCollection<ICUFeature> m_lstFeatures = new ObservableCollection<ICUFeature>();

	protected ObservableCollection<ICUGroup> m_lstGroups = new ObservableCollection<ICUGroup>();

	protected ObservableCollection<ICUUser> m_lstUsers = new ObservableCollection<ICUUser>();

	protected ObservableCollection<ICUBackOffice> m_lstBackOffices = new ObservableCollection<ICUBackOffice>();

	protected ObservableCollection<ICUPMBackOffice> m_lstPMBackOffices = new ObservableCollection<ICUPMBackOffice>();

	protected ObservableCollection<ICUFirmware> m_lstFirmwares = new ObservableCollection<ICUFirmware>();

	private bool m_fThisDirty;

	private bool m_fGroupCollectionDirty;

	private bool m_fUserCollectionDirty;

	private bool m_fBackofficeCollectionDirty;

	private bool m_fPMBackofficeCollectionDirty;

	private bool m_fFirmwareCollectionDirty;

	public ObservableCollection<ICUFeature> Features => m_lstFeatures;

	public ObservableCollection<ICUGroup> Groups => m_lstGroups;

	public ObservableCollection<ICUUser> Users => m_lstUsers;

	public ObservableCollection<ICUBackOffice> BackOffices => m_lstBackOffices;

	public ObservableCollection<ICUPMBackOffice> PMBackOffices => m_lstPMBackOffices;

	public ObservableCollection<ICUFirmware> Firmwares => m_lstFirmwares;

	public string Version
	{
		get
		{
			return m_sVersion;
		}
		set
		{
			m_sVersion = value;
			m_fThisDirty = m_sVersion != m_sOriginalVersion;
			PropertyChanged(this, new PropertyChangedEventArgs("Version"));
		}
	}

	public string Date => m_sDateTime;

	public bool IsGroupsDirty
	{
		get
		{
			if (m_lstGroups == null)
			{
				return false;
			}
			if (!m_fGroupCollectionDirty)
			{
				return m_lstGroups.FirstOrDefault((ICUGroup a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsUsersDirty
	{
		get
		{
			if (m_lstUsers == null)
			{
				return false;
			}
			if (!m_fUserCollectionDirty)
			{
				return m_lstUsers.FirstOrDefault((ICUUser a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsBackofficesDirty
	{
		get
		{
			if (m_lstBackOffices == null)
			{
				return false;
			}
			if (!m_fBackofficeCollectionDirty)
			{
				return m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsPMBackofficesDirty
	{
		get
		{
			if (m_lstPMBackOffices == null)
			{
				return false;
			}
			if (!m_fPMBackofficeCollectionDirty)
			{
				return m_lstPMBackOffices.FirstOrDefault((ICUPMBackOffice a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsFirmwareDirty
	{
		get
		{
			if (m_lstFirmwares == null)
			{
				return false;
			}
			if (!m_fFirmwareCollectionDirty)
			{
				return m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsDirty
	{
		get
		{
			if (!m_fThisDirty && !IsGroupsDirty && !IsUsersDirty && !IsBackofficesDirty && !IsPMBackofficesDirty)
			{
				return IsFirmwareDirty;
			}
			return true;
		}
	}

	public event PropertyChangedEventHandler PropertyChanged;

	public bool UpdateUsersFromConfig(string configFilename)
	{
		try
		{
			XDocument xDocument = XDocument.Parse(File.ReadAllText(configFilename));
			List<XElement> list = xDocument.Descendants("Field").ToList();
			if (list.Count == 0)
			{
				list = xDocument.Descendants("User").ToList();
			}
			List<ICUUser> list2 = new List<ICUUser>();
			foreach (XElement item in list)
			{
				ICUUser iCUUser = new ICUUser(m_lstGroups.ToList(), item);
				iCUUser.PropertyChanged += OnUserChanged;
				list2.Add(iCUUser);
			}
			foreach (ICUUser newUser in list2)
			{
				ICUUser iCUUser2 = m_lstUsers.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == newUser.User.ToLowerInvariant());
				if (iCUUser2 == null)
				{
					newUser.Dirty = true;
					m_lstUsers.Add(newUser);
				}
				else
				{
					iCUUser2.CopyFrom(newUser);
				}
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return false;
		}
	}

	private void OnFeatureChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Features"));
	}

	private void OnGroupChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Groups"));
	}

	private void OnUserChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Users"));
	}

	private void OnBackOfficeChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("BackOffices"));
		AddBackofficesToFeatures();
	}

	private void OnFirmwareChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Firmwares"));
	}

	public bool WriteUsersXML(string filename = "Users2.xml", bool saveInBase64 = true)
	{
		try
		{
			XDocument xDocument = new XDocument(new XElement("Config", new XAttribute("Version", Version), new XElement("Users", m_lstUsers.Select((ICUUser x) => x.Element))));
			StringBuilder stringBuilder = new StringBuilder();
			using (TextWriter textWriter = new StringWriter(stringBuilder))
			{
				xDocument.Save(textWriter);
			}
			if (saveInBase64)
			{
				stringBuilder = new StringBuilder(Convert.ToBase64String(Encoding.UTF8.GetBytes(stringBuilder.ToString())));
			}
			using (StreamWriter streamWriter = new StreamWriter(filename))
			{
				streamWriter.Write((object?)stringBuilder);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool WritePMBackOfficeSettingsXML(string directory)
	{
		string empty = string.Empty;
		try
		{
			foreach (ICUPMBackOffice lstPMBackOffice in m_lstPMBackOffices)
			{
				XDocument xDocument = new XDocument(new XElement("Config", new XElement("Setting", new XElement("Product", new XAttribute("Model", "NG9xx"), new XAttribute("Device", "NG9xx"), from x in lstPMBackOffice.m_lstProperties
					where x.Element != null
					select x.Element))));
				empty = Path.Combine(directory, lstPMBackOffice.Title.Trim().Replace(' ', '-') + ".xml");
				xDocument.Save(empty);
				string[] source = File.ReadAllLines(empty);
				File.WriteAllLines(empty, source.Skip(1).ToArray());
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool ReadInstallerSettings(string filename = "InstallerSettings.dat", ICUEncryptionType encType = ICUEncryptionType.encryptBase64)
	{
		//IL_003d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0042: Unknown result type (might be due to invalid IL or missing references)
		try
		{
			string text = File.ReadAllText(filename);
			switch (encType)
			{
			case ICUEncryptionType.encryptBase64:
			{
				byte[] array = Convert.FromBase64String(text);
				text = Encoding.UTF8.GetString(array, 0, array.Length);
				break;
			}
			case ICUEncryptionType.encryptRijndael:
			case ICUEncryptionType.encryptRijndaelHashed:
				text = new EncryptDecrypt().Decrypt(text, "Pas5pR@sE");
				break;
			}
			dynamic val = new JavaScriptSerializer
			{
				MaxJsonLength = 52428800
			}.DeserializeObject(text);
			if (val["Type"].ToString() != s_fileID)
			{
				throw new ArgumentException("Incorrect file type!");
			}
			m_sVersion = val["Version"].ToString();
			m_sOriginalVersion = m_sVersion;
			m_sDateTime = val["Date"].ToString();
			m_lstFeatures = new ObservableCollection<ICUFeature>();
			m_lstGroups = new ObservableCollection<ICUGroup>();
			m_lstUsers = new ObservableCollection<ICUUser>();
			m_lstBackOffices = new ObservableCollection<ICUBackOffice>();
			m_lstPMBackOffices = new ObservableCollection<ICUPMBackOffice>();
			m_lstFirmwares = new ObservableCollection<ICUFirmware>();
			foreach (dynamic item in val["Backoffices"])
			{
				ICUBackOffice iCUBackOffice = new ICUBackOffice(item);
				iCUBackOffice.PropertyChanged += OnBackOfficeChanged;
				m_lstBackOffices.Add(iCUBackOffice);
			}
			if (((IDictionary<string, object>)val).ContainsKey("PMBackOffices"))
			{
				foreach (dynamic item2 in val["PMBackOffices"])
				{
					ICUPMBackOffice iCUPMBackOffice = new ICUPMBackOffice(item2);
					iCUPMBackOffice.PropertyChanged += OnBackOfficeChanged;
					m_lstPMBackOffices.Add(iCUPMBackOffice);
				}
			}
			if (((IDictionary<string, object>)val).ContainsKey("Features"))
			{
				foreach (dynamic item3 in val["Features"])
				{
					ICUFeature iCUFeature = new ICUFeature(item3);
					iCUFeature.PropertyChanged += OnFeatureChanged;
					m_lstFeatures.Add(iCUFeature);
				}
			}
			AddManualFeatures();
			AddBackofficesToFeatures();
			if (((IDictionary<string, object>)val).ContainsKey("Groups"))
			{
				foreach (dynamic item4 in val["Groups"])
				{
					ICUGroup iCUGroup = new ICUGroup(m_lstFeatures, item4);
					iCUGroup.PropertyChanged += OnGroupChanged;
					m_lstGroups.Add(iCUGroup);
				}
			}
			if (m_lstGroups.Count == 0)
			{
				List<string> list = new List<string>();
				foreach (dynamic item5 in val["Users"])
				{
					if (((IDictionary<string, object>)item5).ContainsKey("Company"))
					{
						list.Add(item5["Company"].Trim());
					}
				}
				AddDefaultGroups(list);
			}
			foreach (dynamic item6 in val["Users"])
			{
				ICUUser iCUUser = new ICUUser(m_lstGroups.ToList(), item6);
				iCUUser.PropertyChanged += OnUserChanged;
				m_lstUsers.Add(iCUUser);
			}
			foreach (dynamic item7 in val["Firmwares"])
			{
				ICUFirmware iCUFirmware = new ICUFirmware(item7);
				iCUFirmware.PropertyChanged += OnFirmwareChanged;
				m_lstFirmwares.Add(iCUFirmware);
			}
			foreach (ICUUser lstUser in m_lstUsers)
			{
				if (lstUser.Group != null)
				{
					lstUser.Group.NumberOfUsers++;
				}
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool WriteInstallerSettings(string filename = "InstallerSettings.dat", ICUEncryptionType encType = ICUEncryptionType.encryptBase64, bool addBackoffices = true)
	{
		try
		{
			if (File.Exists(filename))
			{
				File.Copy(filename, Path.ChangeExtension(filename, "bak"), overwrite: true);
			}
			List<ICUFeature> source = Features.Where((ICUFeature a) => a.Type != ICUFeatureType.Backoffice).ToList();
			StringBuilder stringBuilder = new StringBuilder();
			string text = string.Join(",", source.Select((ICUFeature a) => a.Json).ToArray());
			string text2 = string.Join(",", Groups.Select((ICUGroup a) => a.Json(encType != ICUEncryptionType.encryptRijndaelHashed)).ToArray());
			string text3 = string.Join(",", Users.Select((ICUUser a) => a.Json).ToArray());
			if (encType == ICUEncryptionType.encryptRijndaelHashed)
			{
				ICUUser iCUUser = Users.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == "isah");
				List<ICUUser> list = new List<ICUUser>();
				using (SHA256 sha256Hasher = SHA256.Create())
				{
					RNGCryptoServiceProvider rng = new RNGCryptoServiceProvider();
					foreach (ICUUser user in Users)
					{
						if (user.User.ToLowerInvariant() != "isah")
						{
							string httpData = user.Group.HTTPUser + ":" + user.Group.HTTPPassword;
							string password = iCUUser.Password;
							list.Add(ICUUser.CreateHashedUser(sha256Hasher, rng, user.User, user.Password, user.Group, password, httpData));
						}
					}
				}
				list = list.OrderBy((ICUUser a) => a.Password).ToList();
				text3 = string.Join(",", list.Select((ICUUser a) => a.Json).ToArray());
			}
			string text4 = string.Join(",", Firmwares.Select((ICUFirmware a) => a.Json).ToArray());
			string text5 = string.Empty;
			string text6 = string.Empty;
			if (addBackoffices)
			{
				text5 = string.Join(",", BackOffices.Select((ICUBackOffice a) => a.Json).ToArray());
				text6 = string.Join(",", PMBackOffices.Select((ICUPMBackOffice a) => a.Json).ToArray());
			}
			stringBuilder.AppendFormat("{{\n\"Type\":\"{0}\",\n\"Version\":\"{1}\",\n\"Date\":\"{2}\",\n\"Features\":[{3}\n],\n\"Groups\":[{4}\n],\n\"Users\":[{5}\n],\n\"Backoffices\":[{6}\n],\n\"PMBackOffices\":[{7}\n],\n\"Firmwares\":[{8}\n]}}", new object[9]
			{
				s_fileID,
				Version,
				DateTime.Now.ToString(),
				text,
				text2,
				text3,
				text5,
				text6,
				text4
			});
			if (encType == ICUEncryptionType.encryptBase64)
			{
				stringBuilder = new StringBuilder(Convert.ToBase64String(Encoding.UTF8.GetBytes(stringBuilder.ToString())));
			}
			else if (encType == ICUEncryptionType.encryptRijndael || encType == ICUEncryptionType.encryptRijndaelHashed)
			{
				stringBuilder = new StringBuilder(new EncryptDecrypt().Encrypt(stringBuilder.ToString(), "Pas5pR@sE"));
			}
			using (StreamWriter streamWriter = new StreamWriter(filename))
			{
				streamWriter.Write((object?)stringBuilder);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool UploadFileToFTP(string ftpSite, string fileName, string fullFilePath, string userName, string password)
	{
		try
		{
			Uri uri = new Uri(new Uri(ftpSite), fileName);
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Method = "STOR";
			ftpWebRequest.Credentials = new NetworkCredential(userName, password);
			StreamReader streamReader = new StreamReader(fullFilePath);
			byte[] bytes = Encoding.UTF8.GetBytes(streamReader.ReadToEnd());
			streamReader.Close();
			ftpWebRequest.ContentLength = bytes.Length;
			Stream requestStream = ftpWebRequest.GetRequestStream();
			requestStream.Write(bytes, 0, bytes.Length);
			requestStream.Close();
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			Logger.Debug("Upload file complete to file {Uri}, status {Description}", uri.ToString(), ftpWebResponse.StatusDescription);
			ftpWebResponse.Close();
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "Upload failure");
			return false;
		}
		return true;
	}

	public bool UpdateFTPFirmwareList(string ftpSite, string ftpUserName, string ftpPassword, int timeout = 1000)
	{
		try
		{
			Uri uri = new Uri(new Uri(ftpSite), "Firmware");
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "NLST";
			ftpWebRequest.Credentials = new NetworkCredential(ftpUserName, ftpPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			string text = new StreamReader(ftpWebResponse.GetResponseStream()).ReadToEnd();
			ftpWebResponse.Close();
			string[] array = (from a in text.Split(new char[1] { '\n' })
				select a.Trim(new char[3] { ' ', '\r', '\n' }) into a
				where !string.IsNullOrEmpty(a)
				select a).ToArray();
			foreach (string fiFile in array)
			{
				ICUFirmware iCUFirmware = m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Filename.ToLowerInvariant() == fiFile.ToLowerInvariant());
				if (iCUFirmware == null)
				{
					DateTime dateTime = DateTime.Now;
					try
					{
						FtpWebRequest ftpWebRequest2 = (FtpWebRequest)WebRequest.Create(new Uri(uri, fiFile));
						ftpWebRequest2.Timeout = timeout;
						ftpWebRequest2.Method = "MDTM";
						ftpWebRequest2.Credentials = new NetworkCredential(ftpUserName, ftpPassword);
						FtpWebResponse ftpWebResponse2 = (FtpWebResponse)ftpWebRequest2.GetResponse();
						dateTime = ftpWebResponse.LastModified;
						ftpWebResponse2.Close();
					}
					catch (Exception ex)
					{
						Logger.Error(ex, "Error while requesting the filedate for {File}: {Message}", fiFile, ex.Message);
					}
					ICUFirmware iCUFirmware2 = new ICUFirmware(fiFile, "0.0.0", "", dateTime.ToString());
					iCUFirmware2.OnFTP = true;
					m_lstFirmwares.Add(iCUFirmware2);
				}
				else
				{
					iCUFirmware.OnFTP = true;
				}
			}
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "Upload failure!");
			return false;
		}
		return true;
	}

	public ICUGroup AddGroup()
	{
		ICUGroup iCUGroup = new ICUGroup(m_lstFeatures);
		iCUGroup.PropertyChanged += OnGroupChanged;
		m_lstGroups.Add(iCUGroup);
		m_fGroupCollectionDirty = true;
		OnGroupChanged(this, new PropertyChangedEventArgs("Group"));
		return iCUGroup;
	}

	public bool RemoveGroup(ICUGroup oldGroup)
	{
		m_lstGroups.Remove(oldGroup);
		m_fGroupCollectionDirty = true;
		return true;
	}

	public void ResetGroup(ICUGroup group)
	{
		group?.Rollback();
	}

	public ICUUser AddUser()
	{
		ICUUser iCUUser = new ICUUser();
		iCUUser.PropertyChanged += OnUserChanged;
		m_lstUsers.Add(iCUUser);
		m_fUserCollectionDirty = true;
		OnUserChanged(this, new PropertyChangedEventArgs("User"));
		return iCUUser;
	}

	public bool RemoveUser(ICUUser oldUser)
	{
		m_lstUsers.Remove(oldUser);
		m_fUserCollectionDirty = true;
		return true;
	}

	public void ResetUser(ICUUser user)
	{
		user?.Rollback();
	}

	public ICUBackOffice AddBackoffice(ICUBackOffice copyFrom)
	{
		ICUBackOffice iCUBackOffice = new ICUBackOffice();
		iCUBackOffice.PropertyChanged += OnBackOfficeChanged;
		iCUBackOffice.CopyFrom(copyFrom, allProperties: true);
		iCUBackOffice.Title += "(copy)";
		iCUBackOffice.TitleNL += "(copy)";
		iCUBackOffice.TitleDE += "(copy)";
		iCUBackOffice.TitleFR += "(copy)";
		m_lstBackOffices.Add(iCUBackOffice);
		m_fBackofficeCollectionDirty = true;
		OnBackOfficeChanged(this, new PropertyChangedEventArgs("Title"));
		return iCUBackOffice;
	}

	public bool RemoveBackoffice(ICUBackOffice oldBO)
	{
		m_lstBackOffices.Remove(oldBO);
		m_fBackofficeCollectionDirty = true;
		return true;
	}

	public void ResetBackoffice(ICUBackOffice backoffice)
	{
		backoffice?.Rollback();
	}

	public ICUPMBackOffice AddPMBackoffice(ICUPMBackOffice copyFrom)
	{
		ICUPMBackOffice iCUPMBackOffice = new ICUPMBackOffice();
		iCUPMBackOffice.PropertyChanged += OnBackOfficeChanged;
		iCUPMBackOffice.CopyFrom(copyFrom, allProperties: true);
		iCUPMBackOffice.Title += "(copy)";
		m_lstPMBackOffices.Add(iCUPMBackOffice);
		m_fPMBackofficeCollectionDirty = true;
		OnBackOfficeChanged(this, new PropertyChangedEventArgs("Title"));
		return iCUPMBackOffice;
	}

	public bool RemovePMBackoffice(ICUPMBackOffice oldBO)
	{
		m_lstPMBackOffices.Remove(oldBO);
		m_fPMBackofficeCollectionDirty = true;
		return true;
	}

	public void ResetPMBackoffice(ICUPMBackOffice backoffice)
	{
		backoffice?.Rollback();
	}

	public ICUFirmware AddFirmware()
	{
		ICUFirmware iCUFirmware = new ICUFirmware();
		iCUFirmware.PropertyChanged += OnFirmwareChanged;
		m_lstFirmwares.Add(iCUFirmware);
		m_fFirmwareCollectionDirty = true;
		OnFirmwareChanged(this, new PropertyChangedEventArgs("Firmware"));
		return iCUFirmware;
	}

	public ICUFirmware FindFirmware(string fileName)
	{
		string filenameNoPath = Path.GetFileName(fileName).ToLowerInvariant();
		return m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Filename.ToLowerInvariant() == filenameNoPath);
	}

	public bool RemoveFirmware(ICUFirmware oldFirmware)
	{
		m_lstFirmwares.Remove(oldFirmware);
		m_fFirmwareCollectionDirty = true;
		return true;
	}

	public void ResetFirmware(ICUFirmware firmware)
	{
		firmware?.Rollback();
	}

	public void CommitChanges()
	{
		m_fThisDirty = false;
		m_fGroupCollectionDirty = false;
		m_fUserCollectionDirty = false;
		m_fBackofficeCollectionDirty = false;
		m_fFirmwareCollectionDirty = false;
		foreach (ICUFeature lstFeature in m_lstFeatures)
		{
			lstFeature.Commit();
		}
		foreach (ICUGroup lstGroup in m_lstGroups)
		{
			lstGroup.Commit();
		}
		foreach (ICUUser lstUser in m_lstUsers)
		{
			lstUser.Commit();
		}
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			lstBackOffice.Commit();
		}
		foreach (ICUPMBackOffice lstPMBackOffice in m_lstPMBackOffices)
		{
			lstPMBackOffice.Commit();
		}
		foreach (ICUFirmware lstFirmware in m_lstFirmwares)
		{
			lstFirmware.Commit();
		}
	}

	protected void AddBackofficesToFeatures()
	{
		m_lstFeatures = new ObservableCollection<ICUFeature>(m_lstFeatures.Where((ICUFeature a) => a.Type != ICUFeatureType.Backoffice));
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			if (!string.IsNullOrEmpty(lstBackOffice.Title))
			{
				AddFeature(ICUFeature.CreateBackOfficeFeature(lstBackOffice.Title, $"BackOffice '{lstBackOffice.Title}'"));
			}
		}
	}

	protected void AddDefaultGroups(List<string> allCompanies)
	{
		List<string> list = allCompanies.Distinct().ToList();
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Admin", "Administrators"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Production", "Production"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Service", "Our own service engineers"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Customer", "Customers"));
		foreach (string company in list)
		{
			if (!(company != ""))
			{
				continue;
			}
			ICUGroup iCUGroup = new ICUGroup(m_lstFeatures, $"Extern_{company}", company);
			if (iCUGroup == null)
			{
				continue;
			}
			List<string> list2 = (from a in m_lstBackOffices
				where a.Groups.Contains(company.ToUpperInvariant())
				select $"BO_{a.Title.Replace(' ', '_').Trim().ToUpperInvariant()}").ToList();
			foreach (ICUFeatureRight feature in iCUGroup.Features)
			{
				if (feature.Feature.Type == ICUFeatureType.Backoffice)
				{
					if (list2.Contains(feature.Feature.ID))
					{
						feature.Rights = ICURights.ReadOnly;
					}
					else
					{
						feature.Rights = ICURights.None;
					}
				}
			}
			m_lstGroups.Add(iCUGroup);
		}
	}

	protected void AddFeature(ICUFeature newFeature)
	{
		if (newFeature != null && !m_lstFeatures.Any((ICUFeature a) => a.ID == newFeature.ID))
		{
			m_lstFeatures.Add(newFeature);
		}
	}

	protected void AddManualFeatures()
	{
		AddFeature(ICUFeature.CreatePageFeature("INFORMATION", "Page Information", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("BACKOFFICE", "Page Backoffice"));
		AddFeature(ICUFeature.CreatePageFeature("POWER", "Page Power", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("NETWORK", "Page Network", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("STATES", "Page States", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("SOCKET", "Page Socket"));
		AddFeature(ICUFeature.CreatePageFeature("LOG", "Page Log"));
		AddFeature(ICUFeature.CreatePageFeature("METERVALUES", "Page MeterValue", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("WHITELIST", "Page Whitelist", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("UI", "Page UserInterface", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("UPLOAD", "Page Upload", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("PRODUCTION", "Page Production", ICURights.None));
		AddFeature(ICUFeature.CreatePageFeature("ALLPROPERTIES", "All settings page"));
		AddFeature(ICUFeature.CreatePageFeature("TRANSACTIONS", "Page Transactions"));
		AddFeature(ICUFeature.CreatePageFeature("FAT", "Factory Acceptance Test", ICURights.None));
		AddFeature(ICUFeature.CreatePageFeature("SAT", "Service Acceptance Test"));
		AddFeature(ICUFeature.CreateFeature("CREATEFWU", "Create an FWU file"));
		AddFeature(ICUFeature.CreateIDFeature(8272, "Charge Box Model", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8273, "Charge Box Serial Number", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8275, "Charge Box Identity", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8271, "Charge Box Configuration", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8290, "Max Station Current"));
		AddFeature(ICUFeature.CreateIDFeature(8488, "Start Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8489, "Normal Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8496, "Simplified Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8292, "Load Balancing Mode", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8295, "P1 Max Installation Current"));
		AddFeature(ICUFeature.CreateIDFeature(8296, "P1 Balancing Safe Current"));
		AddFeature(ICUFeature.CreateIDFeature(8310, "BackOffice short name"));
		AddFeature(ICUFeature.CreateIDFeature(8311, "Connection method"));
		AddFeature(ICUFeature.CreateIDFeature(8312, "BackOffice Server Domain and Port"));
		AddFeature(ICUFeature.CreateIDFeature(8305, "BackOffice Server Domain and Port Wired"));
		AddFeature(ICUFeature.CreateIDFeature(8487, "Main Offline NFC Authorization"));
		AddFeature(ICUFeature.CreateIDFeature(8503, "Main EV Disconnect Action"));
		AddFeature(ICUFeature.CreateIDFeature(8448, "GPRS APN Name"));
		AddFeature(ICUFeature.CreateIDFeature(8449, "GPRS APN User"));
		AddFeature(ICUFeature.CreateIDFeature(8450, "GPRS APN Password"));
		AddFeature(ICUFeature.CreateIDFeature(8313, "Communication DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8320, "Communication DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8339, "Send Station Status"));
		AddFeature(ICUFeature.CreateIDFeature(8321, "Protocol Name"));
		AddFeature(ICUFeature.CreateIDFeature(8322, "Protocol Version"));
		AddFeature(ICUFeature.CreateIDFeature(8502, "EV Disconnect Timeout"));
		AddFeature(ICUFeature.CreateIDFeature(8289, "LED/Display AutoDim & Intensity"));
		AddFeature(ICUFeature.CreateIDFeature(8485, "Main Socket Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12581, "Main Socket Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8728, "Energy Meter Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12824, "Energy Meter Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8309, "IP1 Address"));
		AddFeature(ICUFeature.CreateIDFeature(8307, "IP1 Netmask"));
		AddFeature(ICUFeature.CreateIDFeature(8308, "IP1 Gateway Address"));
		AddFeature(ICUFeature.CreateIDFeature(8313, "IP1 DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8320, "IP1 DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8317, "IP2 Address"));
		AddFeature(ICUFeature.CreateIDFeature(8316, "IP2 Netmask"));
		AddFeature(ICUFeature.CreateIDFeature(8315, "IP2 Gateway Address"));
		AddFeature(ICUFeature.CreateIDFeature(8318, "IP2 DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8319, "IP2 DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8485, "Main Socket Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12581, "Main Socket Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8728, "Energy Meter Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12824, "Energy Meter Type (Socket 2)"));
	}

	public bool UpdatePMBackOffices()
	{
		m_lstPMBackOffices.Clear();
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			string text = lstBackOffice.Title.ToLowerInvariant().Trim();
			string text2 = lstBackOffice.Title.Substring(0, lstBackOffice.Title.LastIndexOf('-')).Trim();
			if (text.EndsWith("production auto") || text.EndsWith("productionauto"))
			{
				AddPMBackOffice(text2, lstBackOffice, lstBackOffice);
			}
			else if (text.EndsWith("production gprs") || text.EndsWith("productiongprs"))
			{
				string autoName = lstBackOffice.Title.Replace("gprs", "auto");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName) == null)
				{
					string wiredName = lstBackOffice.Title.Replace("gprs", "wired");
					ICUBackOffice boWired = m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == wiredName);
					AddPMBackOffice(text2, lstBackOffice, boWired);
				}
			}
			else if (text.EndsWith("production wired") || text.EndsWith("productionwired"))
			{
				string autoName2 = lstBackOffice.Title.Replace("wired", "auto");
				string gprsName = lstBackOffice.Title.Replace("wired", "gprs");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName2) == null && m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == gprsName) == null)
				{
					AddPMBackOffice(text2, null, lstBackOffice);
				}
			}
			else if (text.EndsWith("sandbox auto"))
			{
				text2 = $"{text2} sandbox";
				AddPMBackOffice(text2, lstBackOffice, lstBackOffice);
			}
			else if (text.EndsWith("sandbox gprs"))
			{
				string autoName3 = lstBackOffice.Title.Replace("gprs", "auto");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName3) == null)
				{
					string wiredName2 = lstBackOffice.Title.Replace("gprs", "wired");
					ICUBackOffice boWired2 = m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == wiredName2);
					text2 = $"{text2} sandbox";
					AddPMBackOffice(text2, lstBackOffice, boWired2);
				}
			}
			else if (text.EndsWith("sandbox wired"))
			{
				string autoName4 = lstBackOffice.Title.Replace("wired", "auto");
				string gprsName2 = lstBackOffice.Title.Replace("wired", "gprs");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName4) == null && m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == gprsName2) == null)
				{
					text2 = $"{text2} sandbox";
					AddPMBackOffice(text2, null, lstBackOffice);
				}
			}
		}
		return true;
	}

	private bool AddPMBackOffice(string newtitle, ICUBackOffice boGPRS, ICUBackOffice boWired)
	{
		bool isLANEnabled = true;
		bool isGPRSEnabled = true;
		if (boWired == null)
		{
			isLANEnabled = false;
			boWired = boGPRS;
		}
		if (boGPRS == null)
		{
			isGPRSEnabled = false;
			boGPRS = boWired;
		}
		if (boWired == null && boGPRS == null)
		{
			return false;
		}
		ICUPMBackOffice item = new ICUPMBackOffice
		{
			Title = newtitle,
			APNName = boGPRS.APNName,
			APNUser = boGPRS.APNUser,
			APNPassword = boGPRS.APNPassword,
			BackOfficeURLwired_Domain = boWired.BackOfficeURLwired_Domain,
			BackOfficeURLwired_Path = boWired.BackOfficeURLwired_Path,
			BackOfficeURL_Domain = boGPRS.BackOfficeURL_Domain,
			BackOfficeURL_Path = boGPRS.BackOfficeURL_Path,
			CentralMeterValueAlignment = boGPRS.CentralMeterValueAlignment,
			DNS1_1 = boGPRS.DNS1_1,
			DNS1_2 = boGPRS.DNS1_2,
			DNS2_1 = boWired.DNS2_1,
			DNS2_2 = boWired.DNS2_2,
			EVDisconnectAction = boGPRS.EVDisconnectAction,
			EVDisconnectTimeout = boGPRS.EVDisconnectTimeout,
			IntensityAuto = boGPRS.IntensityAuto,
			IntensityIntensity = boGPRS.IntensityIntensity,
			Language = boGPRS.Language,
			OCPP15SmartChargingType = boGPRS.OCPP15SmartChargingType,
			OfflineNFCAuthorization = boGPRS.OfflineNFCAuthorization,
			OnlineNFCAuthorization = boGPRS.OnlineNFCAuthorization,
			PingPongInterval = boGPRS.PingPongInterval,
			ProtocolName = boGPRS.ProtocolName,
			ProtocolVersion = boGPRS.ProtocolVersion,
			SendStationStatus = boGPRS.SendStationStatus,
			SimPin = boGPRS.SimPin,
			TimezoneMinutes = boGPRS.TimezoneMinutes,
			TransactionMessageAttempts = boGPRS.TransactionMessageAttempts,
			TransactionMessageRetryInterval = boGPRS.TransactionMessageRetryInterval,
			IsGPRSEnabled = isGPRSEnabled,
			IsLANEnabled = isLANEnabled
		};
		m_lstPMBackOffices.Add(item);
		return true;
	}

	public ICUUser FindUser(string username)
	{
		string loweruser = username.ToLowerInvariant();
		return Users.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == loweruser);
	}
}
public enum ICURights
{
	None,
	ReadOnly,
	Full
}
public enum ICUFeatureType
{
	Normal,
	Page,
	Property,
	Backoffice
}
public class ICUFeature : ICUBaseObject
{
	private string m_sName;

	private string m_sID;

	private ICURights m_eDefault = ICURights.Full;

	private ICUFeatureType m_eType;

	private string m_sComment;

	public ICUFeature OriginalValues { get; set; }

	public string Name
	{
		get
		{
			return m_sName;
		}
		set
		{
			m_sName = value.Trim();
			FireChangedEvent("Name");
			CheckDirty();
		}
	}

	public string ID
	{
		get
		{
			return m_sID;
		}
		set
		{
			m_sID = value.Trim().ToUpperInvariant();
			FireChangedEvent("ID");
			CheckDirty();
		}
	}

	public ICURights Default
	{
		get
		{
			return m_eDefault;
		}
		set
		{
			m_eDefault = value;
			FireChangedEvent("Default");
			CheckDirty();
		}
	}

	public ICUFeatureType Type
	{
		get
		{
			return m_eType;
		}
		set
		{
			m_eType = value;
			FireChangedEvent("Type");
			CheckDirty();
		}
	}

	public string Comment
	{
		get
		{
			return m_sComment;
		}
		set
		{
			m_sComment = value.Trim();
			FireChangedEvent("Comment");
			CheckDirty();
		}
	}

	public XElement Element => new XElement("Feature", new XAttribute("Name", Name), new XAttribute("ID", ID), new XAttribute("Type", Type), new XAttribute("Default", Default), new XAttribute("Comment", Comment));

	public string Json => string.Format("\n\t{{\"Name\":{0},\"ID\":{1},\"Type\":{2},\"Default\":{3},\"Comment\":{4}}}", new object[5]
	{
		ValidString(Name),
		ValidString(ID),
		ValidString(Type.ToString()),
		ValidString(Default.ToString()),
		ValidString(Comment)
	});

	public ICUFeature()
		: base(fDirty: true)
	{
		m_sName = "";
		m_sID = "";
		m_eType = ICUFeatureType.Normal;
		m_eDefault = ICURights.None;
		m_sComment = "";
	}

	public ICUFeature(List<ICUGroup> allGroups, XElement xelem)
	{
		m_sName = getAttribute(xelem, "Name").Trim();
		m_sID = getAttribute(xelem, "ID").Trim();
		m_eType = (ICUFeatureType)Enum.Parse(typeof(ICUFeatureType), getAttribute(xelem, "Type").Trim());
		m_eDefault = (ICURights)Enum.Parse(typeof(ICURights), getAttribute(xelem, "Default").Trim());
		if (xelem.NextNode != null && xelem.NextNode.NodeType == XmlNodeType.Comment)
		{
			m_sComment = ((XComment)xelem.NextNode).Value.Trim();
		}
		OriginalValues = new ICUFeature(Name, ID, Type, Default, Comment);
	}

	public ICUFeature(dynamic obj)
	{
		m_sName = obj["Name"].Trim();
		m_sID = obj["ID"].Trim();
		if (((IDictionary<string, object>)obj).ContainsKey("Type"))
		{
			m_eType = (ICUFeatureType)Enum.Parse(typeof(ICUFeatureType), obj["Type"].Trim());
		}
		if (((IDictionary<string, object>)obj).ContainsKey("Default"))
		{
			m_eDefault = (ICURights)Enum.Parse(typeof(ICURights), obj["Default"].Trim());
		}
		m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		OriginalValues = new ICUFeature(Name, ID, Type, Default, Comment);
	}

	public ICUFeature(string name, string id, ICUFeatureType eType, ICURights eDefaultRights = ICURights.Full, string comment = "")
	{
		m_sName = name.Trim();
		m_sID = id.Trim();
		m_eType = eType;
		m_eDefault = eDefaultRights;
		m_sComment = comment.Trim();
		OriginalValues = new ICUFeature();
		OriginalValues.Name = name;
		OriginalValues.ID = id;
		OriginalValues.Type = eType;
		OriginalValues.Default = eDefaultRights;
		OriginalValues.Comment = comment;
	}

	public static ICUFeature CreatePageFeature(string id, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"PAGE_{id}", ICUFeatureType.Page, eDefaultRights);
	}

	public static ICUFeature CreateFeature(string id, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"FEATURE_{id}", ICUFeatureType.Normal, eDefaultRights);
	}

	public static ICUFeature CreateIDFeature(ushort usId, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"ID_{usId:X4}", ICUFeatureType.Property, eDefaultRights);
	}

	public static ICUFeature CreateBackOfficeFeature(string title, string name, ICURights eDefaultRights = ICURights.ReadOnly)
	{
		return new ICUFeature(name, $"BO_{title.Replace(' ', '_').Trim().ToUpperInvariant()}", ICUFeatureType.Backoffice, eDefaultRights);
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sName != OriginalValues.Name) && !(m_sID != OriginalValues.ID) && m_eType == OriginalValues.Type)
		{
			return m_sComment != OriginalValues.Comment;
		}
		return true;
	}

	public void CopyFrom(ICUFeature other)
	{
		if (other != null)
		{
			Name = other.Name;
			ID = other.ID;
			Type = other.Type;
			Default = other.Default;
			Comment = other.Comment;
			CheckDirty();
		}
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
		if (OriginalValues != null)
		{
			CopyFrom(OriginalValues);
		}
		CheckDirty();
	}
}
public class ICUFeatureRight : ICUBaseObject
{
	private ICURights m_eCurrentRights;

	private ICURights m_eOriginalRights;

	public ICUFeature Feature { get; }

	public ICURights Rights
	{
		get
		{
			return m_eCurrentRights;
		}
		set
		{
			m_eCurrentRights = value;
			FireChangedEvent("Rights");
		}
	}

	public ICUFeatureRight(ICUFeature feature, ICURights rights)
	{
		Feature = feature;
		m_eCurrentRights = rights;
		m_eOriginalRights = rights;
	}

	protected override bool OnCheckDirty()
	{
		return m_eOriginalRights != Rights;
	}

	public void Commit()
	{
		m_eOriginalRights = m_eCurrentRights;
		CheckDirty();
		FireChangedEvent("Rights");
	}

	public void Rollback()
	{
		m_eCurrentRights = m_eOriginalRights;
		FireChangedEvent("Rights");
	}
}
public class ICUFirmware : ICUBaseObject
{
	private string m_sFilename;

	private string m_sVersion;

	private string m_sComments;

	private string m_sDate;

	private bool m_fFoundOnFtp;

	public ICUFirmware OriginalValues { get; set; }

	public string Filename
	{
		get
		{
			return m_sFilename;
		}
		set
		{
			m_sFilename = value.Trim();
			FireChangedEvent("Filename");
		}
	}

	public string Version
	{
		get
		{
			return m_sVersion;
		}
		set
		{
			m_sVersion = value.Trim();
			FireChangedEvent("Version");
		}
	}

	public string Comments
	{
		get
		{
			return m_sComments;
		}
		set
		{
			m_sComments = value;
			FireChangedEvent("Comments");
		}
	}

	public string Date
	{
		get
		{
			return m_sDate;
		}
		set
		{
			m_sDate = value;
			FireChangedEvent("Date");
		}
	}

	public bool OnFTP
	{
		get
		{
			return m_fFoundOnFtp;
		}
		set
		{
			m_fFoundOnFtp = value;
			FireChangedEvent("OnFTP");
		}
	}

	public XElement Element => new XElement("Firmware", new XAttribute("Filename", Filename), new XAttribute("Version", Version), new XAttribute("Date", Date), new XAttribute("Comments", Comments));

	public string Json => string.Format("\n\t{{\"Filename\":{0},\"Version\":{1},\"Date\":{2},\"Comments\":{3}}}", new object[4]
	{
		ValidString(Filename),
		ValidString(Version),
		ValidString(Date.ToString()),
		ValidString(Comments)
	});

	public ICUFirmware()
		: base(fDirty: true)
	{
		m_sFilename = "";
		m_sVersion = "0.0.0";
		m_sComments = "";
		m_sDate = "";
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(XElement xelem)
	{
		m_sFilename = getAttribute(xelem, "Filename").Trim();
		m_sVersion = getAttribute(xelem, "Version").Trim();
		m_sComments = getAttribute(xelem, "Comments").Trim();
		m_sDate = getAttribute(xelem, "Date").Trim();
		OriginalValues = new ICUFirmware(m_sFilename, m_sVersion, m_sComments, m_sDate);
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(dynamic obj)
	{
		m_sFilename = obj["Filename"].Trim();
		m_sVersion = obj["Version"].Trim();
		m_sComments = obj["Comments"].Trim().Replace("<br>", "\n");
		m_sDate = obj["Date"].Trim();
		OriginalValues = new ICUFirmware(m_sFilename, m_sVersion, m_sComments, m_sDate);
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(string fileName, string version, string comments, string date)
	{
		m_sFilename = fileName.Trim();
		m_sVersion = version.Trim();
		m_sComments = comments.Trim();
		m_sDate = date.Trim();
		m_fFoundOnFtp = false;
		OriginalValues = null;
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sFilename != OriginalValues.Filename) && !(m_sVersion != OriginalValues.Version) && !(m_sComments != OriginalValues.Comments))
		{
			return m_sDate != OriginalValues.Date;
		}
		return true;
	}

	public void CopyFrom(ICUFirmware other)
	{
		if (other != null)
		{
			Filename = other.Filename;
			Version = other.Version;
			Comments = other.Comments;
			Date = other.Date;
			CheckDirty();
		}
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
}
public class ICUGroup : ICUBaseObject
{
	private string m_sName;

	private string m_sComment;

	private string m_sHTTPUser = string.Empty;

	private string m_sHTTPPassword = string.Empty;

	private readonly ObservableCollection<ICUFeatureRight> m_lstFeatures = new ObservableCollection<ICUFeatureRight>();

	public ICUGroup OriginalValues { get; set; }

	public int NumberOfUsers { get; set; }

	public string Name
	{
		get
		{
			return m_sName;
		}
		set
		{
			m_sName = value.Trim();
			FireChangedEvent("Name");
		}
	}

	public string Comment
	{
		get
		{
			return m_sComment;
		}
		set
		{
			m_sComment = value.Trim();
			FireChangedEvent("Comment");
		}
	}

	public string HTTPUser
	{
		get
		{
			return m_sHTTPUser;
		}
		set
		{
			m_sHTTPUser = value.Trim();
			FireChangedEvent("HTTPUser");
		}
	}

	public string HTTPPassword
	{
		get
		{
			return m_sHTTPPassword;
		}
		set
		{
			m_sHTTPPassword = value.Trim();
			FireChangedEvent("HTTPPassword");
		}
	}

	public ObservableCollection<ICUFeatureRight> Features
	{
		get
		{
			return m_lstFeatures;
		}
		set
		{
			FireChangedEvent("Features");
		}
	}

	public XElement Element
	{
		get
		{
			XElement xElement = new XElement("Group", new XAttribute("Name", Name), new XAttribute("Comment", Comment), new XAttribute("HTTPUser", HTTPUser), new XAttribute("HTTPPassword", HTTPPassword));
			foreach (ICUFeatureRight lstFeature in m_lstFeatures)
			{
				if (lstFeature.Rights != lstFeature.Feature.Default)
				{
					xElement.Add(new XElement("FeatureRight", new XAttribute("ID", lstFeature.Feature.ID), new XAttribute("Rights", lstFeature.Rights.ToString())));
				}
			}
			return xElement;
		}
	}

	public ICUGroup(IList<ICUFeature> lstFeatures)
		: base(fDirty: true)
	{
		m_sName = "";
		m_sComment = "";
		m_sHTTPUser = "";
		m_sHTTPPassword = "";
		InitializeFeatures(lstFeatures);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, string name, string comment = "", string httpuser = "", string httppassword = "")
		: base(fDirty: true)
	{
		m_sName = name;
		m_sComment = comment;
		m_sHTTPUser = httpuser;
		m_sHTTPPassword = httppassword;
		InitializeFeatures(lstFeatures);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, XElement xelem)
	{
		InitializeFeatures(lstFeatures);
		m_sName = getAttribute(xelem, "Name").Trim();
		m_sComment = getAttribute(xelem, "Comment").Trim();
		m_sHTTPUser = getAttribute(xelem, "HTTPUser").Trim();
		m_sHTTPPassword = getAttribute(xelem, "HTTPPassword").Trim();
		OriginalValues = new ICUGroup(Name, Comment, HTTPUser, HTTPPassword);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, dynamic obj)
	{
		InitializeFeatures(lstFeatures);
		m_sName = obj["Name"].Trim();
		m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		if (((IDictionary<string, object>)obj).ContainsKey("HTTPUser"))
		{
			m_sHTTPUser = obj["HTTPUser"].Trim();
		}
		if (((IDictionary<string, object>)obj).ContainsKey("HTTPPassword"))
		{
			m_sHTTPPassword = obj["HTTPPassword"].Trim();
		}
		if (((IDictionary<string, object>)obj).ContainsKey("FeatureRights"))
		{
			foreach (dynamic fr in obj["FeatureRights"])
			{
				ICUFeatureRight iCUFeatureRight = m_lstFeatures.FirstOrDefault((ICUFeatureRight a) => a.Feature.ID == fr["ID"]);
				if (iCUFeatureRight != null)
				{
					iCUFeatureRight.Rights = (ICURights)Enum.Parse(typeof(ICURights), fr["Rights"].Trim());
					iCUFeatureRight.Commit();
				}
			}
		}
		OriginalValues = new ICUGroup(Name, Comment, HTTPUser, HTTPPassword);
		CheckDirty();
	}

	private ICUGroup(string name, string comment, string httpuser, string httppassword)
	{
		m_sName = name.Trim();
		m_sComment = comment.Trim();
		m_sHTTPUser = httpuser.Trim();
		m_sHTTPPassword = httppassword.Trim();
		OriginalValues = null;
	}

	private void InitializeFeatures(IList<ICUFeature> lstFeatures)
	{
		m_lstFeatures.Clear();
		foreach (ICUFeature lstFeature in lstFeatures)
		{
			ICUFeatureRight iCUFeatureRight = null;
			iCUFeatureRight = ((Name == null || !(Name.ToLowerInvariant() == "admin")) ? new ICUFeatureRight(lstFeature, lstFeature.Default) : new ICUFeatureRight(lstFeature, ICURights.Full));
			if (iCUFeatureRight != null)
			{
				iCUFeatureRight.PropertyChanged += OnFeatureRightChanged;
				m_lstFeatures.Add(iCUFeatureRight);
			}
		}
	}

	private void OnFeatureRightChanged(object sender, PropertyChangedEventArgs e)
	{
		FireChangedEvent("Features");
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		bool flag = m_lstFeatures.Any((ICUFeatureRight a) => a.Dirty);
		return (m_sName != OriginalValues.Name || m_sComment != OriginalValues.Comment || m_sHTTPUser != OriginalValues.HTTPUser || m_sHTTPPassword != OriginalValues.HTTPPassword) | flag;
	}

	public void CopyFrom(ICUGroup other)
	{
		if (other != null)
		{
			Name = other.Name;
			Comment = other.Comment;
			HTTPUser = other.HTTPUser;
			HTTPPassword = other.HTTPPassword;
			CheckDirty();
		}
	}

	public string Json(bool IncludeCredentials)
	{
		List<string> list = new List<string>();
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			if (lstFeature.Rights != lstFeature.Feature.Default)
			{
				list.Add($"\n\t\t{{\"ID\":{ValidString(lstFeature.Feature.ID)},\"Rights\":{ValidString(lstFeature.Rights.ToString())}}}");
			}
		}
		string text = string.Join(",", list);
		return string.Format("\n\t{{\"Name\":{0},\"Comment\":{1},\"HTTPUser\":{2},\"HTTPPassword\":{3},\"FeatureRights\":[{4}]}}", new object[5]
		{
			ValidString(Name),
			ValidString(Comment),
			ValidString(IncludeCredentials ? HTTPUser : string.Empty),
			ValidString(IncludeCredentials ? HTTPPassword : string.Empty),
			text
		});
	}

	public void Commit()
	{
		if (OriginalValues != null)
		{
			OriginalValues.CopyFrom(this);
		}
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			lstFeature.Commit();
		}
		CheckDirty();
	}

	public void Rollback()
	{
		CopyFrom(OriginalValues);
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			lstFeature.Rollback();
		}
		CheckDirty();
	}

	public ICURights GetRights(string id)
	{
		return m_lstFeatures.FirstOrDefault((ICUFeatureRight a) => a.Feature.ID == id)?.Rights ?? ICURights.Full;
	}
}
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
public class ICUPMProperty : ICloneable
{
	public string Name { get; set; }

	public string PropertyName { get; set; }

	public object Value { get; set; }

	public uint PropertyNumber { get; set; }

	public XElement Element
	{
		get
		{
			if (PropertyNumber == 0 || string.IsNullOrEmpty(Value.ToString()))
			{
				return null;
			}
			string text = Convert.ToString(PropertyNumber >> 8, 16);
			byte b = (byte)(PropertyNumber & 0xFF);
			string text2 = ((b != 0) ? ("sub" + Convert.ToString(b, 16)) : string.Empty);
			return new XElement("Object", new XAttribute("Id", "1." + text + text2), new XAttribute("Value", (Value is bool) ? ((object)(((bool)Value) ? 1 : 0)) : Value), PropertyName);
		}
	}

	public object Clone()
	{
		return new ICUPMProperty
		{
			Name = Name,
			PropertyName = PropertyName,
			Value = Value,
			PropertyNumber = PropertyNumber
		};
	}
}
public class ICUUser : ICUBaseObject
{
	public static int s_passwordLength = 6;

	private string m_sUser;

	private string m_sPassword;

	private string m_sFullname;

	private ICUGroup m_oGroup;

	private string m_sCompany;

	private string m_sComment = string.Empty;

	public ICUUser OriginalValues { get; set; }

	public string User
	{
		get
		{
			return m_sUser;
		}
		set
		{
			m_sUser = value.Trim();
			FireChangedEvent("User");
		}
	}

	public string Password
	{
		get
		{
			return m_sPassword;
		}
		set
		{
			m_sPassword = value.Trim();
			FireChangedEvent("Password");
		}
	}

	public ICUGroup Group
	{
		get
		{
			return m_oGroup;
		}
		set
		{
			m_oGroup = value;
			FireChangedEvent("Group");
		}
	}

	public string Fullname
	{
		get
		{
			return m_sFullname;
		}
		set
		{
			m_sFullname = value.Trim();
			FireChangedEvent("Fullname");
		}
	}

	public string Comment
	{
		get
		{
			return m_sComment;
		}
		set
		{
			m_sComment = value.Trim();
			FireChangedEvent("Comment");
		}
	}

	public string Company
	{
		get
		{
			return m_sCompany;
		}
		set
		{
			m_sCompany = value.Trim();
			FireChangedEvent("Company");
		}
	}

	public XElement Element
	{
		get
		{
			string value = ((Group != null) ? Group.Name : "");
			return new XElement("User", new XAttribute("User", User), new XAttribute("Password", Password), new XAttribute("Group", value), new XAttribute("Fullname", Fullname), new XAttribute("Comment", Comment), new XAttribute("Company", Company));
		}
	}

	public string Json
	{
		get
		{
			string value = ((Group != null) ? Group.Name : "");
			return string.Format("\n\t{{\"User\":{0},\"Password\":{1},\"Group\":{2},\"Fullname\":{3},\"Comment\":{4},\"Company\":{5}}}", new object[6]
			{
				ValidString(User),
				ValidString(Password),
				ValidString(value),
				ValidString(Fullname),
				ValidString(Comment),
				ValidString(Company)
			});
		}
	}

	public string CSVLine => $"\"{Fullname}\", \"{User}\", \"{Password}\"";

	public ICUUser()
		: base(fDirty: true)
	{
		m_oGroup = null;
		m_sPassword = CreatePassword(s_passwordLength);
		m_sComment = "";
	}

	public ICUUser(List<ICUGroup> allGroups, XElement xelem)
	{
		m_sUser = getAttribute(xelem, "User").Trim();
		m_sPassword = getAttribute(xelem, "Password").Trim();
		string rightsName = getAttribute(xelem, "Rights").Trim();
		m_sCompany = getAttribute(xelem, "Group").Trim();
		m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == rightsName);
		if (m_oGroup == null)
		{
			m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == $"Extern_{m_sCompany}");
			if (m_oGroup == null)
			{
				m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == "Extern_ICU");
			}
		}
		m_sFullname = getAttribute(xelem, "Fullname").Trim();
		if (xelem.NextNode != null && xelem.NextNode.NodeType == XmlNodeType.Comment)
		{
			m_sComment = ((XComment)xelem.NextNode).Value.Trim();
		}
		OriginalValues = new ICUUser(User, Password, Group, Fullname, Comment, Company);
	}

	public ICUUser(List<ICUGroup> allGroups, dynamic obj)
	{
		m_sUser = obj["User"].Trim();
		m_sPassword = obj["Password"].Trim();
		m_sFullname = obj["Fullname"].Trim();
		string groupName = obj["Group"].Trim();
		if (((IDictionary<string, object>)obj).ContainsKey("Company"))
		{
			m_sCompany = obj["Company"].Trim();
		}
		else
		{
			m_sCompany = groupName;
		}
		m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == groupName);
		if (m_oGroup == null)
		{
			m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == $"Extern_{m_sCompany}");
			if (m_oGroup == null)
			{
				m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == "Extern_ICU");
			}
		}
		if (((IDictionary<string, object>)obj).ContainsKey("Comment"))
		{
			m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		}
		OriginalValues = new ICUUser(User, Password, Group, Fullname, Comment, Company);
	}

	private ICUUser(string name, string password, ICUGroup group, string fullName, string comment, string company)
	{
		m_sUser = name.Trim();
		m_sPassword = password.Trim();
		m_oGroup = group;
		m_sFullname = fullName.Trim();
		m_sComment = comment.Trim();
		m_sCompany = company.Trim();
		OriginalValues = null;
	}

	public static ICUUser CreateHashedUser(SHA256 sha256Hasher, RNGCryptoServiceProvider rng, string username, string password, ICUGroup group, string isahData, string httpData)
	{
		ICUUser iCUUser = new ICUUser();
		byte[] array = new byte[4];
		byte[] array2 = new byte[1];
		rng.GetBytes(array);
		rng.GetNonZeroBytes(array2);
		int num = array2[0];
		iCUUser.User = Convert.ToBase64String(sha256Hasher.ComputeHash(Encoding.UTF8.GetBytes(username.ToLowerInvariant())));
		string text = BitConverter.ToString(array).Replace("-", "");
		string text2 = text + password;
		byte[] array3 = Encoding.UTF8.GetBytes(text2);
		for (int i = 0; i < num; i++)
		{
			array3 = sha256Hasher.ComputeHash(array3);
		}
		string arg = Convert.ToBase64String(array3);
		iCUUser.Password = $"{num:X2}:{text}:{arg}";
		iCUUser.Group = group;
		iCUUser.Fullname = string.Empty;
		iCUUser.Comment = new EncryptDecrypt().Encrypt(httpData, text2);
		iCUUser.Company = new EncryptDecrypt().Encrypt(isahData, text2);
		return iCUUser;
	}

	public bool ValidateHashedPassword(SHA256 sha256Hash, string otherPassword)
	{
		string[] array = Password.Split(new char[1] { ':' });
		if (array.Length == 3)
		{
			ushort num = Convert.ToUInt16(array[0], 16);
			string s = array[1] + otherPassword;
			byte[] array2 = Encoding.UTF8.GetBytes(s);
			for (int i = 0; i < num; i++)
			{
				array2 = sha256Hash.ComputeHash(array2);
			}
			return array[2] == Convert.ToBase64String(array2);
		}
		return false;
	}

	public static string CreatePassword(int length)
	{
		StringBuilder stringBuilder = new StringBuilder();
		Random random = new Random();
		while (0 < length--)
		{
			stringBuilder.Append("aeiouyaeiouyaeiouybcdfghjklmnpqrstvwxz"[random.Next("aeiouyaeiouyaeiouybcdfghjklmnpqrstvwxz".Length)]);
		}
		stringBuilder[random.Next(stringBuilder.Length)] = "1234567890"[random.Next("1234567890".Length)];
		return stringBuilder.ToString();
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sUser != OriginalValues.User) && !(m_sPassword != OriginalValues.Password) && !(m_sFullname != OriginalValues.Fullname) && m_oGroup == OriginalValues.Group && !(m_sComment != OriginalValues.Comment))
		{
			return m_sCompany != OriginalValues.Company;
		}
		return true;
	}

	public void CopyFrom(ICUUser other)
	{
		if (other != null)
		{
			User = other.User;
			Password = other.Password;
			if (!string.IsNullOrEmpty(other.Fullname.Trim()))
			{
				Fullname = other.Fullname;
			}
			Group = other.Group;
			if (!string.IsNullOrEmpty(other.Comment.Trim()))
			{
				Comment = other.Comment;
			}
			if (!string.IsNullOrEmpty(other.Company.Trim()))
			{
				Company = other.Company;
			}
			CheckDirty();
		}
	}

	public void AddOldElement(XElement xuser)
	{
		string value = ((Group != null) ? Group.Name : "");
		XElement content = new XElement("Field", new XAttribute("User", User), new XAttribute("Password", Password), new XAttribute("Group", value), new XAttribute("Fullname", Fullname));
		xuser.Add(content);
		xuser.Add(new XComment(Comment));
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

	public ICURights GetRights(string id)
	{
		if (m_oGroup != null)
		{
			return m_oGroup.GetRights(id);
		}
		return ICURights.ReadOnly;
	}
}
