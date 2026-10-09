using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Dialogs;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelConnectivity : PanelBase
{
	public enum ENPAPNAuthentication
	{
		Unknown,
		CHAP,
		None,
		PAP,
		Automatic
	}

	public enum ENPPriority
	{
		Off,
		Priority1,
		Priority2,
		Priority3,
		Priority4
	}

	public enum ENPInterface
	{
		None,
		Wired0,
		Wired1,
		Wired2,
		Wired3,
		Wireless0,
		Wireless1,
		Wireless2,
		Wireless3
	}

	public enum ENPVersion
	{
		None,
		Unknown,
		OCPP15,
		OCPP16,
		OCPP201
	}

	public enum ENPSubIds : byte
	{
		ProtocolVersion = 1,
		URL = 3,
		MessageTimeout = 4,
		SecurityProfile = 5,
		Interface = 6,
		APNName = 7,
		APNUsernameDeprecated = 8,
		APNPassword = 9,
		SimPinDeprecated = 10,
		Priority = 14,
		APNUsername = 15,
		SimPin = 16
	}

	private readonly ILogger Logger = Log.ForContext<PanelConnectivity>();

	protected Dictionary<string, string> m_allBackoffices = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicBackOffices = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicConnectionMethod = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicOCPP = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicStatusNotification = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMeterValueTransmissionMode = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMeterValueAlignment = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMeterMeasurandData = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSecurityProfile = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMobileNetworkMode = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicNP_APNAuthentication = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicNP_ConnectionMethod = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicNP_Versions = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicNP_Priority = new Dictionary<string, string>();

	protected UIPropertyExpander m_expMeterValueSampledData;

	protected UIPropertyExpander m_expMeterValueAlignedData;

	protected UIPropertySelect m_cmbDisconnectAction;

	protected UISelect m_cmbBackOffices;

	protected UISelect m_cmbConnectMode;

	protected UIPropertySelect m_cmbProtocol;

	protected UIPropertySelect m_cmbSmartCharging;

	protected UIPropertySelect m_cmbNetworkTechnology;

	protected UIPropertySelect m_cmbNetworkMode;

	protected UIPropertyLabel m_lblConnectionMethod;

	protected UIPropertyLabel m_lblInfoMobileTechnology;

	protected UIPropertyNumber m_nmPingPongInterval;

	protected UIPropertyCheckbox m_chkEichrechtEnabled;

	protected UIPropertyString m_txtBackofficeURL;

	protected UIPropertyString m_txtBackofficePath;

	protected UIPropertyString m_txtBackofficeURLGPRS;

	protected UIPropertyString m_txtBackofficePathGPRS;

	protected UIPropertyString m_txtAPNName;

	protected UIPropertyString m_txtAPNUser;

	protected UIPropertyString m_txtAPNPassword;

	protected UIPropertyString m_txtSimPin;

	protected UIPropertyBase m_chkNetworkFixedGprs;

	protected UIPropertyBase m_chkNetworkFixedWired;

	protected UIPropertyBase m_chkNetworkEnableWifi;

	protected UIPropertyBase m_chkNetworkFixedWifi;

	protected UIPropertyBase m_chkProxyEnabled;

	protected List<UIPropertyBase> m_lstNetworkGprs = new List<UIPropertyBase>();

	protected List<UIPropertyBase> m_lstNetworkWired = new List<UIPropertyBase>();

	protected List<UIPropertyBase> m_lstNetworkWifi = new List<UIPropertyBase>();

	protected List<UIPropertyBase> m_lstProxy = new List<UIPropertyBase>();

	protected List<UIPropertyBase> m_lstWifi = new List<UIPropertyBase>();

	protected UIPropertyBase m_chkApEnabled;

	protected UIPropertyBase m_chkApStart;

	protected Button m_btnScanwifi;

	protected List<OccpMeasurand> m_lsOccpMeasurands;

	protected List<OccpVersions> m_lsOccpVersions;

	protected List<OccpPhases> m_lsOccpPhases;

	protected UIPassword m_txtProxyPassword;

	protected UIPassword m_txtAuthorizationKey;

	protected UIPassword m_txtWifiPsk;

	protected UIPropertyString m_txtWifiSsid;

	protected UIPropertySelect m_cmbWifiSecurityType;

	protected const string m_keyManual = "_MAN";

	protected const string m_keyStandAlone = "_SA";

	private readonly bool m_isPartManagementMode;

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catWired;

	protected UIConfigCategory m_catGprs;

	protected UIConfigCategory m_catWifi;

	protected UIConfigCategory[] m_catNetworkProfile = new UIConfigCategory[4];

	protected UIPropertySelect[] m_cmbNP_Priority = new UIPropertySelect[4];

	protected UIPropertySelect[] m_cmbNP_Interface = new UIPropertySelect[4];

	protected UIPropertySelect[] m_cmbNP_SecurityProfile = new UIPropertySelect[4];

	protected UIPropertySelect[] m_cmbNP_OcppVersion = new UIPropertySelect[4];

	protected UIPropertyNumber[] m_numNP_WebsocketTimeout = new UIPropertyNumber[4];

	protected UIConfigCategory m_catProxy;

	protected UIConfigCategory m_catWebsocket;

	protected UIConfigCategory m_catHeartbeat;

	protected UIConfigCategory m_catStatusNotification;

	protected UIConfigCategory m_catTansactionData;

	protected UIConfigCategory m_catMeterValue;

	protected UIConfigCategory m_catCentralMeter;

	protected UIConfigCategory m_catSCProfiles;

	protected UIConfigCategory m_catNuvve;

	protected UIConfigCategory m_catEichrecht;

	protected UIConfigCategory m_catOCPP16SecurityExtensions;

	private bool? prevSelectionPreset;

	private bool m_bHasNetworkProfiles;

	private bool m_hasExtendedFieldLengths;

	private string m_currentProtocolVersion;

	private string m_meterName = string.Empty;

	private string m_bopresetName = string.Empty;

	private const string connMethodNone = "0";

	private const string connMethodWired = "1";

	private const string connMethodGPRS = "2";

	private const string connMethodAuto = "3";

	public PanelConnectivity(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Connectivity";
		Tooltip = "Connectivity";
		IconName = "cloud-computing.png";
		m_dicConnectionMethod.Add("0", "None");
		m_dicConnectionMethod.Add("1", "Wired");
		m_dicConnectionMethod.Add("2", "Mobile");
		m_dicConnectionMethod.Add("3", "Autodetect");
		m_dicStatusNotification.Add("0", "Immediate");
		m_dicStatusNotification.Add("1", "Immediate timestamp");
		m_dicStatusNotification.Add("2", "Queued");
		m_dicMeterValueTransmissionMode.Add("0", "End");
		m_dicMeterValueTransmissionMode.Add("1", "During");
		m_dicMeterValueTransmissionMode.Add("2", "Always");
		m_dicMeterValueAlignment.Add("0", "Boot");
		m_dicMeterValueAlignment.Add("1", "Clock");
		m_dicSecurityProfile.Add("0", "0: Default");
		m_dicSecurityProfile.Add("1", "1: Unsecure transport with Basic Authentication");
		m_dicSecurityProfile.Add("2", "2: TLS with Basic Authentication");
		m_dicSecurityProfile.Add("3", "3: TLS with Client Side Certificates");
		m_lsOccpMeasurands = new List<OccpMeasurand>
		{
			new OccpMeasurand(EOcppMeasurand.MEAS_NONE, EOccpVersion.VERSION_15_16_20, "None"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_ACTIVE_EXPORT, EOccpVersion.VERSION_15_16_20, "Energy.Active.Export.Register"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_ACTIVE_IMPORT, EOccpVersion.VERSION_15_16_20, "Energy.Active.Import.Register"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_REACTIVE_EXPORT, EOccpVersion.VERSION_15_16_20, "Energy.Reactive.Export.Register"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_REACTIVE_IMPORT, EOccpVersion.VERSION_15_16_20, "Energy.Reactive.Import.Register"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_ACTIVE_EXPORT_INTERVAL, EOccpVersion.VERSION_15_16_20, "Energy.Active.Export.Interval"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_ACTIVE_IMPORT_INTERVAL, EOccpVersion.VERSION_15_16_20, "Energy.Active.Import.Interval"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_REACTIVE_EXPORT_INTERVAL, EOccpVersion.VERSION_15_16_20, "Energy.Reactive.Export.Interval"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_REACTIVE_IMPORT_INTERVAL, EOccpVersion.VERSION_15_16_20, "Energy.Reactive.Import.Interval"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_ACTIVE_EXPORT, EOccpVersion.VERSION_15_16_20, "Power.Active.Export"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_ACTIVE_IMPORT, EOccpVersion.VERSION_15_16_20, "Power.Active.Import"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_REACTIVE_EXPORT, EOccpVersion.VERSION_15_16_20, "Power.Reactive.Export"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_REACTIVE_IMPORT, EOccpVersion.VERSION_15_16_20, "Power.Reactive.Import"),
			new OccpMeasurand(EOcppMeasurand.MEAS_CURRENT_EXPORT, EOccpVersion.VERSION_15_16_20, "Current.Export"),
			new OccpMeasurand(EOcppMeasurand.MEAS_CURRENT_IMPORT, EOccpVersion.VERSION_15_16_20, "Current.Import"),
			new OccpMeasurand(EOcppMeasurand.MEAS_VOLTAGE, EOccpVersion.VERSION_15_16_20, "Voltage"),
			new OccpMeasurand(EOcppMeasurand.MEAS_TEMP, EOccpVersion.VERSION_15_16, "Temperature"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ALFEN_CURRENT_L1, EOccpVersion.VERSION_15, "Current.L1"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ALFEN_CURRENT_L2, EOccpVersion.VERSION_15, "Current.L2"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ALFEN_CURRENT_L3, EOccpVersion.VERSION_15, "Current.L3"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ALFEN_CURRENT_MAX, EOccpVersion.VERSION_15, "Current.Maximum"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_FACTOR, EOccpVersion.VERSION_15_16_20, "Power.Factor"),
			new OccpMeasurand(EOcppMeasurand.MEAS_CURRENT_OFFERED, EOccpVersion.VERSION_16_20, "Current.Offered"),
			new OccpMeasurand(EOcppMeasurand.MEAS_POWER_OFFERED, EOccpVersion.VERSION_16_20, "Power.Offered"),
			new OccpMeasurand(EOcppMeasurand.MEAS_FREQUENCY, EOccpVersion.VERSION_16_20, "Frequency"),
			new OccpMeasurand(EOcppMeasurand.MEAS_RPM, EOccpVersion.VERSION_16, "RPM"),
			new OccpMeasurand(EOcppMeasurand.MEAS_SOC, EOccpVersion.VERSION_16_20, "SoC"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_ACTIVE_NET, EOccpVersion.VERSION_20, "Energy.Active.Net"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_REACTIVE_NET, EOccpVersion.VERSION_20, "Energy.Reactive.Net"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_APPARENT_NET, EOccpVersion.VERSION_20, "Energy.Apparent.Net"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_APPARENT_IMPORT, EOccpVersion.VERSION_20, "Energy.Apparent.Import"),
			new OccpMeasurand(EOcppMeasurand.MEAS_ENERGY_APPARENT_EXPORT, EOccpVersion.VERSION_20, "Energy.Apparent.Export")
		};
		m_lsOccpPhases = new List<OccpPhases>
		{
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L1N, "L1-N"),
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L2N, "L2-N"),
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L3N, "L3-N"),
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L1L2, "L1-L2"),
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L2L3, "L2-L3"),
			new OccpPhases(EOcppMeasurand.MEAS_VOLTAGE, EOccpPhase.PHASE_L3L1, "L3-L1"),
			new OccpPhases(EOcppMeasurand.MEAS_CURRENT_IMPORT, EOccpPhase.PHASE_N, "N"),
			new OccpPhases(EOcppMeasurand.MEAS_CURRENT_IMPORT, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_CURRENT_IMPORT, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_CURRENT_IMPORT, EOccpPhase.PHASE_L3, "L3"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_FACTOR, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_FACTOR, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_FACTOR, EOccpPhase.PHASE_L3, "L3"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_ACTIVE_IMPORT, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_ACTIVE_IMPORT, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_ACTIVE_IMPORT, EOccpPhase.PHASE_L3, "L3"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_REACTIVE_IMPORT, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_REACTIVE_IMPORT, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_POWER_REACTIVE_IMPORT, EOccpPhase.PHASE_L3, "L3"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_ACTIVE_IMPORT, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_ACTIVE_IMPORT, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_ACTIVE_IMPORT, EOccpPhase.PHASE_L3, "L3"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_REACTIVE_IMPORT, EOccpPhase.PHASE_L1, "L1"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_REACTIVE_IMPORT, EOccpPhase.PHASE_L2, "L2"),
			new OccpPhases(EOcppMeasurand.MEAS_ENERGY_REACTIVE_IMPORT, EOccpPhase.PHASE_L3, "L3")
		};
		m_lsOccpVersions = new List<OccpVersions>
		{
			new OccpVersions("1.5", "OCPP 1.5", EOccpVersion.VERSION_15),
			new OccpVersions("1.6", "OCPP 1.6", EOccpVersion.VERSION_16),
			new OccpVersions("2.0.1", "OCPP 2.0.1", EOccpVersion.VERSION_20)
		};
		OccpVersions occpVersions = m_lsOccpVersions.Find((OccpVersions x) => (x.Version & EOccpVersion.VERSION_15) != 0);
		OccpVersions occpVersions2 = m_lsOccpVersions.Find((OccpVersions x) => (x.Version & EOccpVersion.VERSION_16) != 0);
		m_dicOCPP.Add(occpVersions.Key, occpVersions.Name);
		m_dicOCPP.Add(occpVersions2.Key, occpVersions2.Name);
		m_dicMobileNetworkMode.Add("0", "Automatic");
		m_dicMobileNetworkMode.Add("1", "Manual");
		AddEnumToDictionary(m_dicNP_APNAuthentication, ENPAPNAuthentication.CHAP);
		AddEnumToDictionary(m_dicNP_APNAuthentication, ENPAPNAuthentication.None);
		AddEnumToDictionary(m_dicNP_APNAuthentication, ENPAPNAuthentication.PAP);
		AddEnumToDictionary(m_dicNP_APNAuthentication, ENPAPNAuthentication.Automatic);
		AddEnumToDictionary(m_dicNP_Priority, ENPPriority.Off, "Not used");
		AddEnumToDictionary(m_dicNP_Priority, ENPPriority.Priority1, "1");
		AddEnumToDictionary(m_dicNP_Priority, ENPPriority.Priority2, "2");
		AddEnumToDictionary(m_dicNP_Priority, ENPPriority.Priority3, "3");
		AddEnumToDictionary(m_dicNP_Priority, ENPPriority.Priority4, "4");
		AddEnumToDictionary(m_dicNP_ConnectionMethod, ENPInterface.None, "None");
		AddEnumToDictionary(m_dicNP_ConnectionMethod, ENPInterface.Wired0, "Wired (Ethernet)");
		AddEnumToDictionary(m_dicNP_ConnectionMethod, ENPInterface.Wireless0, "Mobile");
		AddEnumToDictionary(m_dicNP_Versions, ENPVersion.Unknown, "Unknown");
		AddEnumToDictionary(m_dicNP_Versions, ENPVersion.OCPP15, "OCPP 1.5");
		AddEnumToDictionary(m_dicNP_Versions, ENPVersion.OCPP16, "OCPP 1.6");
		AddEnumToDictionary(m_dicNP_Versions, ENPVersion.OCPP201, "OCPP 2.0.1");
		StartUpdateTimer(1000);
	}

	private void AddEnumToDictionary(Dictionary<string, string> dict, object value, string description = "")
	{
		if (!dict.ContainsKey(((int)value).ToString()))
		{
			if (string.IsNullOrEmpty(description))
			{
				dict.Add(((int)value).ToString(), value.ToString());
			}
			else
			{
				dict.Add(((int)value).ToString(), description);
			}
		}
	}

	private string RemoveTrailingEncryptionKey(string fileName)
	{
		if (fileName.EndsWith("-a", StringComparison.OrdinalIgnoreCase) || fileName.EndsWith("-b", StringComparison.OrdinalIgnoreCase) || fileName.EndsWith("-c", StringComparison.OrdinalIgnoreCase))
		{
			return fileName.Substring(0, fileName.Length - 2);
		}
		return fileName;
	}

	private string CombineBopresetMeterName(string bopresetname)
	{
		if (!string.IsNullOrEmpty(m_meterName))
		{
			string text = bopresetname + "," + m_meterName;
			if (text.Length > 50)
			{
				return bopresetname;
			}
			return text;
		}
		return bopresetname;
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		newDevice.UpdateCategories("generic", "generic2", "ocpp", "comm");
		if (m_currentDevice.HasProperty(8310))
		{
			string[] array = m_currentDevice.GetPropertyString(8310, 0, 0).Split(new char[1] { ',' });
			m_bopresetName = array[0];
			if (array.Length > 1)
			{
				m_meterName = array[1];
			}
		}
		m_bHasNetworkProfiles = iCULanDevice.HasProperty(8432, 14);
		m_hasExtendedFieldLengths = m_bHasNetworkProfiles && iCULanDevice.HasProperty(8432, 15);
		Dictionary<string, string> dictionary = new Dictionary<string, string>();
		if (Directory.Exists(AppProperties.LocalBackofficePresetsFolder))
		{
			string[] updateFileTypes = iCULanDevice.getUpdateFileTypes;
			m_allBackoffices = (from a in Directory.GetFiles(AppProperties.LocalBackofficePresetsFolder)
				where Enumerable.Contains(updateFileTypes, Path.GetExtension(a).ToLowerInvariant())
				select a).ToDictionary((string a) => Path.GetFileNameWithoutExtension(a), (string path) => Path.GetFullPath(path));
			dictionary = m_allBackoffices.Keys.Select((string a) => RemoveTrailingEncryptionKey(a)).Distinct().ToDictionary((string a) => a, (string a) => a);
		}
		dictionary.Add("_MAN", "<Manually enter backend settings>");
		dictionary.Add("_SA", "<Standalone>");
		m_dicBackOffices = dictionary.OrderBy((KeyValuePair<string, string> a) => a.Key).ToDictionary((KeyValuePair<string, string> a) => a.Key, (KeyValuePair<string, string> a) => a.Value);
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			UIConfigCategory uIConfigCategory = m_configPanel.AddCategory("General");
			m_cmbBackOffices = (UISelect)uIConfigCategory.Add(AddPlainSelect("Backoffice preset", m_dicBackOffices, "", null, 2127360u));
			m_cmbBackOffices.Changed += OnBackOfficeSelectChanged;
			if (!m_bHasNetworkProfiles)
			{
				m_cmbConnectMode = (UISelect)uIConfigCategory.Add(AddPlainSelect("Connect method", m_dicConnectionMethod, "", null, 2127616u));
				string text = newDevice.GetPropertyInt(8311, 0).ToString();
				if (text == "99")
				{
					text = "3";
				}
				m_cmbConnectMode.CustomValue = text;
			}
			else
			{
				m_cmbConnectMode = null;
			}
			if (iCULanDevice.HasProperty(8436))
			{
				uIConfigCategory.Add(AddCustomNumber(8436, 0, 0, "Network profile connection attempts", 1.0, null, 2159616u));
			}
			UpdateOcppDictionairies(iCULanDevice);
			if (iCULanDevice.HasWifiSupport)
			{
				AddEnumToDictionary(m_dicNP_ConnectionMethod, ENPInterface.Wireless1, "WiFi");
			}
			else
			{
				m_dicNP_ConnectionMethod.Remove(6.ToString());
			}
			if (m_bHasNetworkProfiles)
			{
				for (int num = 0; num < m_catNetworkProfile.Length; num++)
				{
					AddNetworkProfileCategory(num, iCULanDevice);
				}
			}
			else
			{
				m_cmbProtocol = (UIPropertySelect)uIConfigCategory.Add(AddCustomSelect(8322, 0, "Protocol", m_dicOCPP));
				m_cmbProtocol.Changed += OnProtocol_Changed;
				m_lblConnectionMethod = (UIPropertyLabel)uIConfigCategory.Add(AddWarningText("Back office preset has only GPRS connection. But device supports only wired connections.", "", 0, null, 2));
				m_lblConnectionMethod.Hide = true;
			}
			m_currentProtocolVersion = GetOcppVersion(iCULanDevice);
			m_catWired = m_configPanel.AddCategory("Wired");
			if (!m_bHasNetworkProfiles)
			{
				m_txtBackofficeURL = (UIPropertyString)m_catWired.Add(AddText(8305, 1, "Back office URL"));
				m_txtBackofficePath = (UIPropertyString)m_catWired.Add(AddText(8305, 2, "Back office path"));
				m_lstNetworkWired.Add(m_txtBackofficeURL);
				m_lstNetworkWired.Add(m_txtBackofficePath);
			}
			m_catWired.Add(m_chkNetworkFixedWired = AddCheckBox(8317, 2, 0, "Fixed IP address"));
			m_lstNetworkWired.Clear();
			m_lstNetworkWired.Add(m_catWired.Add(AddIPAddress(8317, 1, "IP address")));
			m_lstNetworkWired.Add(m_catWired.Add(AddIPAddress(8315, 1, "Netmask")));
			m_lstNetworkWired.Add(m_catWired.Add(AddIPAddress(8316, 1, "Gateway address")));
			m_lstNetworkWired.Add(m_catWired.Add(AddIPAddress(8318, 1, "DNS 1")));
			m_lstNetworkWired.Add(m_catWired.Add(AddIPAddress(8319, 1, "DNS 2")));
			m_catWired.Add(AddText(8274, 1, "Ethernet MAC address"), advancedProperty: true);
			m_catGprs = m_configPanel.AddCategory("Mobile");
			AddNetworkModeAndTechnologyToMobile(iCULanDevice);
			if (!m_bHasNetworkProfiles)
			{
				m_txtBackofficeURLGPRS = (UIPropertyString)m_catGprs.Add(AddText(8312, 1, "Back office URL"));
				m_txtBackofficePathGPRS = (UIPropertyString)m_catGprs.Add(AddText(8312, 2, "Back office path"));
				m_txtAPNName = (UIPropertyString)m_catGprs.Add(AddWriteOnlyText(8448, 0, "APN name"));
				m_txtAPNUser = (UIPropertyString)m_catGprs.Add(AddWriteOnlyText(8449, 0, "APN user"));
				m_txtAPNPassword = (UIPropertyString)m_catGprs.Add(AddWriteOnlyText(8450, 0, "APN password"));
				m_txtSimPin = (UIPropertyString)m_catGprs.Add(AddWriteOnlyText(8451, 0, "SIM pin"), advancedProperty: true);
			}
			AddGPRSInfoToMobile();
			m_catGprs.Add(m_chkNetworkFixedGprs = AddCheckBox(8309, 2));
			m_lstNetworkGprs.Clear();
			m_lstNetworkGprs.Add(m_catGprs.Add(AddIPAddress(8309, 1)));
			m_lstNetworkGprs.Add(m_catGprs.Add(AddIPAddress(8307, 1)));
			m_lstNetworkGprs.Add(m_catGprs.Add(AddIPAddress(8308, 1)));
			m_lstNetworkGprs.Add(m_catGprs.Add(AddIPAddress(8313, 1)));
			m_lstNetworkGprs.Add(m_catGprs.Add(AddIPAddress(8320, 1)));
			if (iCULanDevice.HasWifiSupport)
			{
				m_catWifi = m_configPanel.AddCategory("WiFi");
				m_catWifi.Add(m_chkNetworkEnableWifi = AddCheckBox(12932, 0));
				m_lstWifi.Clear();
				m_lstWifi.Add(m_catWifi.Add(m_txtWifiSsid = (UIPropertyString)AddText(12938, 0)));
				m_lstWifi.Add(m_catWifi.Add(m_txtWifiPsk = (UIPassword)AddPlainPassword("Password", "", null, 3312384u)));
				m_lstWifi.Add(m_catWifi.Add(m_cmbWifiSecurityType = (UIPropertySelect)AddSelect(12940, 0, 0, 0)));
				m_lstWifi.Add(m_catWifi.Add(AddReadOnlySelect(12947, 0, 0, 0)));
				m_lstWifi.Add(m_catWifi.Add(m_chkApEnabled = AddCheckBox(12945, 0)));
				m_lstWifi.Add(m_catWifi.Add(m_chkApStart = AddCheckBox(12946, 0)));
				m_chkNetworkFixedWifi = AddCheckBox(12933, 2);
				m_lstWifi.Add(m_catWifi.Add(m_chkNetworkFixedWifi, advancedProperty: true));
				m_lstNetworkWifi.Clear();
				m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12933, 1), advancedProperty: true));
				m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12934, 1), advancedProperty: true));
				m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12935, 1), advancedProperty: true));
				m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12936, 1), advancedProperty: true));
				m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12937, 1), advancedProperty: true));
				if (iCULanDevice.HasProperty(12950, 1))
				{
					m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12950, 1), advancedProperty: true));
					m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12951, 1), advancedProperty: true));
					m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12952, 1), advancedProperty: true));
					m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(12953, 1), advancedProperty: true));
					m_lstNetworkWifi.Add(m_catWifi.Add(AddIPAddress(13056, 1), advancedProperty: true));
				}
				m_catWifi.AddWidget(AddToolBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
				m_btnScanwifi = AddCustomButton("Scan Wi-Fi networks", "Scan", OnScanClicked, true);
				m_catWifi.AddWidget(m_btnScanwifi);
				m_btnScanwifi.Sensitive = false;
			}
			m_catOCPP16SecurityExtensions = m_configPanel.AddCategory("Back office security", "Back office security");
			if (iCULanDevice.HasProperty(10018))
			{
				m_catOCPP16SecurityExtensions.Add(AddText(10018, 0, "CPO name (certificate)"));
			}
			if (!m_bHasNetworkProfiles && iCULanDevice.HasProperty(10019))
			{
				m_catOCPP16SecurityExtensions.Add(AddCustomSelect(10019, 0, "Security Profile", m_dicSecurityProfile));
			}
			if (iCULanDevice.HasProperty(10019))
			{
				m_txtAuthorizationKey = (UIPassword)AddPlainPassword("Back office authorization Key", "", null, 2564865u);
				m_catOCPP16SecurityExtensions.Add(m_txtAuthorizationKey);
			}
			m_catWebsocket = m_configPanel.AddCategory("Websocket");
			m_nmPingPongInterval = (UIPropertyNumber)m_catWebsocket.Add(AddCustomNumber(8330, 0, 0, "Ping pong interval (s)"), advancedProperty: true);
			m_nmPingPongInterval.Changed += OnPingPongInterval_Changed;
			if (!m_bHasNetworkProfiles)
			{
				m_catWebsocket.Add(AddCustomNumber(8331, 1, 0, "Wired websocket timeout (s)"), advancedProperty: true);
				m_catWebsocket.Add(AddCustomNumber(8331, 2, 0, "Mobile websocket timeout (s)"), advancedProperty: true);
			}
			m_catWebsocket.Add(AddCustomNumber(8333, 1, 0, "Wired OCPP send timeout (s)"), advancedProperty: true);
			m_catWebsocket.Add(AddCustomNumber(8333, 2, 0, "Mobile OCPP send timeout (s)"), advancedProperty: true);
			m_catWebsocket.Add(AddCustomNumber(8334, 1, 0, "Wired OCPP reply timeout (s)"), advancedProperty: true);
			m_catWebsocket.Add(AddCustomNumber(8334, 2, 0, "Mobile OCPP reply timeout (s)"), advancedProperty: true);
			m_catProxy = m_configPanel.AddCategory("Proxy");
			if (iCULanDevice.HasProperty(8471))
			{
				m_catProxy.Add(m_chkProxyEnabled = AddCheckBox(8471, 0, 0, "Proxy enabled"));
			}
			if (iCULanDevice.HasProperty(8469))
			{
				m_lstProxy.Add(m_catProxy.Add(AddText(8469, 0, "Proxy address and port")));
			}
			if (iCULanDevice.HasProperty(8470))
			{
				m_lstProxy.Add(m_catProxy.Add(AddText(8470, 0, "Proxy user name")));
			}
			if (iCULanDevice.HasProperty(8469))
			{
				m_txtProxyPassword = (UIPassword)AddPlainPassword("Proxy password", "", null, 2168321u);
				m_lstProxy.Add(m_catProxy.Add(m_txtProxyPassword));
			}
			m_catHeartbeat = m_configPanel.AddCategory("Heartbeat");
			m_catHeartbeat.Add(AddCheckBox(8349, 0, 0, "Send always"), advancedProperty: true);
			m_catHeartbeat.Add(AddCustomNumber(8326, 0, 0, "Interval (s)"), advancedProperty: true);
			m_catStatusNotification = m_configPanel.AddCategory("Status notification");
			if (iCULanDevice.HasProperty(8339))
			{
				m_catStatusNotification.Add(AddCheckBox(8339, 0, 0, "Send station status"), advancedProperty: true);
			}
			if (iCULanDevice.HasProperty(8484))
			{
				m_catStatusNotification.Add(AddCheckBox(8484, 0, 0, "Send informational notifications"), advancedProperty: true);
			}
			if (iCULanDevice.HasProperty(12888))
			{
				m_catStatusNotification.Add(AddCustomNumber(12888, 0, 0, "Minimum Status Duration"), advancedProperty: true);
			}
			m_catStatusNotification.Add(AddCustomSelect(8348, 0, "Mode", m_dicStatusNotification), advancedProperty: true);
			m_catTansactionData = m_configPanel.AddCategory("Transaction data");
			m_catTansactionData.Add(AddCustomNumber(8342, 0, 0, "Message attemps"), advancedProperty: true);
			m_catTansactionData.Add(AddCustomNumber(8343, 0, 0, "Message retry interval (s)"), advancedProperty: true);
			m_catTansactionData.Add(AddCustomNumber(8578, 0, 0, "Random mv clock aligned msg (s)"), advancedProperty: true);
			m_cmbDisconnectAction = (UIPropertySelect)m_catTansactionData.Add(AddSelect(8503, 0, 0, 0, "Disconnect action"), advancedProperty: true);
			m_cmbDisconnectAction.SetTooltip("Action that should be executed when the charging cable is removed from the EV");
			m_catMeterValue = m_configPanel.AddCategory("Meter value");
			m_catMeterValue.Add(AddCustomNumber(8327, 0, 0, "Meter value sample interval (s)"), advancedProperty: true);
			m_catMeterValue.Add(AddCustomNumber(8346, 0, 0, "Clock aligned data interval (s)"), advancedProperty: true);
			if ((newDevice as ICULanDevice).HasProperty(21040))
			{
				m_catMeterValue.Add(AddCheckBox(21040, 0, 0, "Register meter values incl. phases"), advancedProperty: true);
			}
			LoadMeasurandDataDictionairy(m_currentProtocolVersion);
			m_expMeterValueSampledData = AddExpander("Meter values sampled data");
			for (byte b = 1; b <= 9; b++)
			{
				UIPropertySelect propBase = (UIPropertySelect)AddCustomSelect(8344, b, $"Sampled data {b}", m_dicMeterMeasurandData, 65535);
				m_expMeterValueSampledData.Add(propBase);
			}
			m_catMeterValue.Add(m_expMeterValueSampledData, advancedProperty: true);
			m_expMeterValueAlignedData = AddExpander("Meter values aligned data");
			for (byte b2 = 1; b2 <= 9; b2++)
			{
				UIPropertySelect propBase2 = (UIPropertySelect)AddCustomSelect(8345, b2, $"Aligned data {b2}", m_dicMeterMeasurandData, 65535);
				m_expMeterValueAlignedData.Add(propBase2);
			}
			m_catMeterValue.Add(m_expMeterValueAlignedData, advancedProperty: true);
			m_catCentralMeter = m_configPanel.AddCategory("Central meter");
			m_catCentralMeter.Add(AddCustomSelect(8328, 0, "Meter value transmission mode", m_dicMeterValueTransmissionMode), advancedProperty: true);
			m_catCentralMeter.Add(AddCustomSelect(8329, 0, "Meter value alignment", m_dicMeterValueAlignment), advancedProperty: true);
			m_catCentralMeter.Add(AddCustomNumber(8335, 0, 0, "Meter value sample interval (s)"), advancedProperty: true);
			if (iCULanDevice.HasProperty(8300, 1))
			{
				m_catSCProfiles = m_configPanel.AddCategory("SC profiles", "Smart charging profiles");
				m_cmbSmartCharging = (UIPropertySelect)m_catSCProfiles.Add(AddSelect(8300, 1, 0, 0, "OCPP15 SC type"), advancedProperty: true);
				m_cmbSmartCharging.SetTooltip("OCPP 1.5 smart charging type");
			}
			m_catNuvve = m_configPanel.AddCategory("Nuvve");
			m_catNuvve.Add(AddCheckBox(8299, 1, 0, "Enabled"), advancedProperty: true);
			m_catNuvve.Add(AddCustomNumber(8299, 2, 0, "Interval (s)"), advancedProperty: true);
			m_catNuvve.Add(AddCustomNumber(8299, 3, 0, "Threshold (W)"), advancedProperty: true);
			m_catEichrecht = m_configPanel.AddCategory("Eichrecht");
			if (iCULanDevice.HasProperty(8554))
			{
				m_chkEichrechtEnabled = (UIPropertyCheckbox)m_catEichrecht.Add(AddCheckBox(8554, 0, 0, "Eichrecht enabled"));
				m_chkEichrechtEnabled.SetEnable(!iCULanDevice.IsEichrechtEnabled);
			}
			if (iCULanDevice.HasProperty(8555))
			{
				m_catEichrecht.Add(AddCheckBox(8555, 0, 0, "Signed meter values at Interval"));
			}
			if (iCULanDevice.HasProperty(9537))
			{
				m_catEichrecht.Add(AddReadOnlyText(9537, 0, "Public Key Socket 1", UIPropertyStringType.PublicKey));
			}
			if (iCULanDevice.HasProperty(9538) && newDevice.NumberOfSockets > 1)
			{
				m_catEichrecht.Add(AddReadOnlyText(9538, 0, "Public Key Socket 2", UIPropertyStringType.PublicKey));
			}
			if (iCULanDevice.HasProperty(9760))
			{
				m_catEichrecht.Add(AddSelect(9760, 0, 0, 0));
			}
			if (iCULanDevice.HasProperty(9761))
			{
				m_catEichrecht.Add(AddCheckBox(9761, 0));
			}
			if (iCULanDevice.HasProperty(8558))
			{
				m_catEichrecht.Add(AddText(8558, 0, "QR code base URL"), advancedProperty: true);
			}
		}
		LoadCustomBackofficeSettings(newDevice);
		return true;
	}

	private string GetOcppVersion(ICULanDevice lanDevice)
	{
		if (lanDevice.HasProperty(8322))
		{
			return lanDevice.GetPropertyString(8322, 0, 0);
		}
		string ordinalVersion = lanDevice.GetPrioritizedOcppVersion();
		if (ordinalVersion == "1")
		{
			return m_lsOccpVersions.Find((OccpVersions x) => (x.Version & EOccpVersion.VERSION_16) != 0).Key;
		}
		return m_lsOccpVersions.Find((OccpVersions x) => x.Name == m_dicNP_Versions[ordinalVersion]).Key;
	}

	private void AddNetworkModeAndTechnologyToMobile(ICULanDevice lanDevice)
	{
		if ((lanDevice.FirmwareVersionNumber >= new Version("4.10.0") || lanDevice.isAHP) && lanDevice.SupportsModem())
		{
			m_catGprs.Add(m_cmbNetworkMode = (UIPropertySelect)AddCustomSelect(8467, 0, "Network Mode", m_dicMobileNetworkMode));
			m_catGprs.Add(m_cmbNetworkTechnology = (UIPropertySelect)AddCustomSelect(8468, 0, "Network Technology", m_dicMobileNetworkTechnology));
			m_lblInfoMobileTechnology = (UIPropertyLabel)m_catGprs.Add(AddLabelText("", EUILabelType.LargeInfo));
			LoadSupportedNetworkTechnologies(lanDevice);
		}
	}

	private void AddGPRSInfoToMobile()
	{
		m_catGprs.Add(AddReadOnlyText(8464, 0, "Mobile signal strength"), advancedProperty: true);
		m_catGprs.Add(AddReadOnlyText(8452, 0, "SIM IMSI"), advancedProperty: true);
		m_catGprs.Add(AddReadOnlyText(8453, 0, "SIM ICCID"), advancedProperty: true);
	}

	private void AddNetworkProfileCategory(int index, ICULanDevice lanDev)
	{
		if (index >= 0 && index < m_catNetworkProfile.Length)
		{
			ushort id = (ushort)(8432 + index);
			m_catNetworkProfile[index] = m_configPanel.AddCategory($"Network Profile {index + 1}");
			m_cmbNP_Priority[index] = (UIPropertySelect)m_catNetworkProfile[index].Add(AddCustomSelect(id, 14, "Priority", m_dicNP_Priority, 0, null, 2158606u));
			m_cmbNP_Priority[index].Changed += OnNetworkProfileSelectionChanged;
			m_cmbNP_Interface[index] = (UIPropertySelect)m_catNetworkProfile[index].Add(AddCustomSelect(id, 6, "Connect method", m_dicNP_ConnectionMethod, 0, null, 2158598u));
			m_cmbNP_Interface[index].Changed += OnNetworkProfileSelectionChanged;
			m_cmbNP_OcppVersion[index] = (UIPropertySelect)m_catNetworkProfile[index].Add(AddCustomSelect(id, 1, "Protocol", m_dicNP_Versions, 0, null, 2158593u));
			m_cmbNP_OcppVersion[index].Changed += OnProtocol_Changed;
			m_catNetworkProfile[index].Add(AddText(id, 3, "CSMS URL", null, 2158595u));
			m_cmbNP_SecurityProfile[index] = (UIPropertySelect)m_catNetworkProfile[index].Add(AddCustomSelect(id, 5, "Security Profile", m_dicSecurityProfile, 0, null, 2158597u));
			m_catNetworkProfile[index].Add(AddWriteOnlyText(id, 7, "APN name", null, 2162688u));
			m_catNetworkProfile[index].Add(AddWriteOnlyText(id, (byte)(m_hasExtendedFieldLengths ? 15 : 8), "APN user", null, 2162944u));
			m_catNetworkProfile[index].Add(AddWriteOnlyText(id, 9, "APN password", null, 2163200u));
			m_catNetworkProfile[index].Add(AddWriteOnlyText(id, (byte)(m_hasExtendedFieldLengths ? 16 : 10), "SIM pin", null, 2163456u));
			m_numNP_WebsocketTimeout[index] = (UIPropertyNumber)m_catNetworkProfile[index].Add(AddCustomNumber(id, 4, 0, "Websocket timeout (s)", 1.0, null, 2158596u));
			m_numNP_WebsocketTimeout[index].SetValueMinMax(10.0, 3600.0);
		}
	}

	private void OnPingPongInterval_Changed(object sender, EventArgs e)
	{
		int num = Convert.ToInt32(m_nmPingPongInterval.GetValue());
		if (num > 0 && num < 30)
		{
			num = 30;
		}
		m_nmPingPongInterval.SetValue(num);
	}

	private void OnProtocol_Changed(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		string ocppVersion = GetOcppVersion(m_currentDevice);
		bool flag = ocppVersion != m_currentProtocolVersion;
		m_currentProtocolVersion = ocppVersion;
		if (ocppVersion != m_lsOccpVersions.Find((OccpVersions x) => x.Version == EOccpVersion.VERSION_15).Key)
		{
			ICUProperty property = m_currentDevice.GetProperty(8300, 1);
			if (property != null)
			{
				property.Value = 0;
			}
			m_configPanel.ShowCategory(m_catSCProfiles, show: false);
		}
		else
		{
			m_configPanel.ShowCategory(m_catSCProfiles);
		}
		if ((!BusyChangingDevice & flag) && m_currentDevice != null)
		{
			LoadMeasurandDataDictionairy(ocppVersion);
			UpdateMeasurandData(m_expMeterValueSampledData, m_dicMeterMeasurandData);
			UpdateMeasurandData(m_expMeterValueAlignedData, m_dicMeterMeasurandData);
			m_expMeterValueSampledData.Childs().ForEach((UIPropertyBase a) =>
			{
				((UIPropertySelect)a).SetCustomList(m_dicMeterMeasurandData);
			});
			m_expMeterValueAlignedData.Childs().ForEach((UIPropertyBase a) =>
			{
				((UIPropertySelect)a).SetCustomList(m_dicMeterMeasurandData);
			});
		}
	}

	private void UpdateMeasurandData(UIPropertyExpander expMeasurands, Dictionary<string, string> meterValueOptions)
	{
		List<Tuple<string, string>> list = new List<Tuple<string, string>>();
		foreach (UIPropertySelect item2 in expMeasurands.Childs())
		{
			if (int.TryParse(item2.GetValue().ToString(), out var value))
			{
				value &= 65535;
				if (!meterValueOptions.ContainsKey(value.ToString()))
				{
					OccpMeasurand occpMeasurand = m_lsOccpMeasurands.Find((OccpMeasurand a) => a.Measurand == (EOcppMeasurand)value);
					string item = ((occpMeasurand != null) ? occpMeasurand.Key : item2.GetValue().ToString());
					list.Add(new Tuple<string, string>(item2.Name, item));
					item2.SetValue("0");
				}
			}
			else
			{
				list.Add(new Tuple<string, string>(item2.Name, item2.GetValue().ToString()));
				item2.SetValue("0");
			}
		}
		if (list.Any())
		{
			string dialogText = "The following " + expMeasurands.Name + " are incompatible with OCPP " + m_cmbProtocol?.GetValue().ToString() + " and will be resetted to 'None':";
			list.ForEach((Tuple<string, string> a) =>
			{
				dialogText = dialogText + "\n - " + a.Item1 + "    Value: " + a.Item2;
			});
			new DlgInfo(dialogText, showInTaskbar: true).Run();
		}
	}

	private void LoadCustomBackofficeSettings(ICUDevice newDevice)
	{
		if (newDevice == null)
		{
			return;
		}
		string text = newDevice.GetPropertyString(8310, 0, 0).Split(new char[1] { ',' })[0].Trim().ToLowerInvariant();
		if (string.IsNullOrEmpty(text) || text.Contains("standalone") || text.Contains("stand-alone"))
		{
			if (!m_bHasNetworkProfiles)
			{
				newDevice.UpdateProperties(newDevice.GetProperty(8311, 0));
				string text2 = newDevice.GetPropertyInt(8311, 0).ToString();
				if (text2 == "99")
				{
					text2 = "3";
				}
				m_cmbConnectMode.CustomValue = text2;
				m_cmbConnectMode.SetValue(text2);
				if (text2 == "0")
				{
					m_cmbBackOffices.CustomValue = "_SA";
				}
				else
				{
					m_cmbBackOffices.CustomValue = "_MAN";
				}
				return;
			}
			bool flag = true;
			for (int i = 0; i < m_catNetworkProfile.Length; i++)
			{
				ushort propId = (ushort)(8432 + i);
				newDevice.UpdateProperties(newDevice.GetProperty(propId, 14));
				if (newDevice.GetPropertyInt(propId, 14) != 0)
				{
					flag = false;
					break;
				}
			}
			m_cmbBackOffices.CustomValue = (flag ? "_SA" : "_MAN");
			return;
		}
		if (m_isPartManagementMode)
		{
			string text3 = string.Empty;
			int num = int.MaxValue;
			string text4 = text.ToLowerInvariant().Trim();
			text4 = text4.Replace("- production gprs", "");
			text4 = text4.Replace("- production wired", "");
			text4 = text4.Replace("- production auto", "");
			text4 = text4.Trim();
			if (text4.Length > 1)
			{
				foreach (KeyValuePair<string, string> dicBackOffice in m_dicBackOffices)
				{
					if (text4 == dicBackOffice.Key.ToLowerInvariant().Trim())
					{
						text3 = dicBackOffice.Key;
						break;
					}
				}
			}
			if (string.IsNullOrEmpty(text3))
			{
				foreach (KeyValuePair<string, string> dicBackOffice2 in m_dicBackOffices)
				{
					int num2 = LevenshteinDistance(dicBackOffice2.Key.ToLowerInvariant(), text4);
					if (num2 < num)
					{
						num = num2;
						text3 = dicBackOffice2.Key;
					}
				}
			}
			if (!string.IsNullOrEmpty(text3))
			{
				m_cmbBackOffices.CustomValue = text3;
			}
		}
		else
		{
			m_cmbBackOffices.CustomValue = text;
		}
		if (m_cmbConnectMode != null)
		{
			m_cmbConnectMode.SetValue(m_cmbConnectMode.CustomValue);
		}
	}

	private void LoadSupportedNetworkTechnologies(ICULanDevice lanDevice)
	{
		bool flag = lanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.Mobile3G4G);
		Dictionary<string, string> dictionary = new Dictionary<string, string>(m_dicMobileNetworkTechnology);
		lanDevice.UpdateProperties(2137600u);
		if (!lanDevice.SupportsRats(ERadioAccessTechnology.GPRS))
		{
			dictionary.Remove("0");
		}
		if (!lanDevice.SupportsRats(ERadioAccessTechnology.UMTS) || !flag)
		{
			dictionary.Remove("1");
		}
		if ((!lanDevice.SupportsRats(ERadioAccessTechnology.LTE) || !flag) && !lanDevice.isAHP)
		{
			dictionary.Remove("2");
		}
		m_cmbNetworkTechnology.SetCustomList(dictionary);
		m_dicMobileNetworkMode.TryGetValue(m_cmbNetworkMode.GetValue().ToString(), out var value);
		m_cmbNetworkTechnology.SetEnable(string.Compare(value, "Manual") == 0 && dictionary.Any());
		if (dictionary.Any() && m_cmbNetworkTechnology.GetSelectedOption() == string.Empty)
		{
			m_cmbNetworkTechnology.SetValue(dictionary.Keys.FirstOrDefault());
		}
		if (!dictionary.Any())
		{
			m_lblInfoMobileTechnology.SetValue("None of the Network technologies is currently available.");
			m_lblInfoMobileTechnology.SetType(EUILabelType.LargeWarning);
		}
		else if (!lanDevice.isAHP && !flag && (lanDevice.SupportsRats(ERadioAccessTechnology.LTE) || lanDevice.SupportsRats(ERadioAccessTechnology.UMTS)))
		{
			m_lblInfoMobileTechnology.SetValue("Obtain a license to unlock the 3G & 4G network technologies.");
			m_lblInfoMobileTechnology.SetType(EUILabelType.LargeInfo);
		}
		else
		{
			m_lblInfoMobileTechnology.SetValue("");
			m_lblInfoMobileTechnology.SetType(EUILabelType.LargeInfo);
		}
	}

	public static int LevenshteinDistance(string s, string t)
	{
		int length = s.Length;
		int length2 = t.Length;
		int[,] array = new int[length + 1, length2 + 1];
		if (length == 0)
		{
			return length2;
		}
		if (length2 == 0)
		{
			return length;
		}
		int num = 0;
		while (num <= length)
		{
			array[num, 0] = num++;
		}
		int num2 = 0;
		while (num2 <= length2)
		{
			array[0, num2] = num2++;
		}
		for (int i = 1; i <= length; i++)
		{
			for (int j = 1; j <= length2; j++)
			{
				int num3 = ((t[j - 1] != s[i - 1]) ? 1 : 0);
				array[i, j] = Math.Min(Math.Min(array[i - 1, j] + 1, array[i, j - 1] + 1), array[i - 1, j - 1] + num3);
			}
		}
		return array[length, length2];
	}

	protected void SetProperty(ref List<ICUProperty> propList, ushort propId, byte subId, object newValue)
	{
		if (m_currentDevice != null)
		{
			ICUProperty property = m_currentDevice.GetProperty(propId, subId);
			if (property != null)
			{
				property.Value = newValue;
				propList.Add(property);
			}
		}
	}

	private void ClearNetworkProfileCategory(ref List<ICUProperty> propList, int index)
	{
		if (index >= 0 && index < m_catNetworkProfile.Length)
		{
			ushort propId = (ushort)(8432 + index);
			SetProperty(ref propList, propId, 14, (byte)0);
			SetProperty(ref propList, propId, 6, (byte)0);
			SetProperty(ref propList, propId, 1, (byte)1);
			SetProperty(ref propList, propId, 3, "");
			SetProperty(ref propList, propId, 5, 0);
			SetProperty(ref propList, propId, 7, "");
			SetProperty(ref propList, propId, (byte)(m_hasExtendedFieldLengths ? 15 : 8), "");
			SetProperty(ref propList, propId, 9, "");
			SetProperty(ref propList, propId, (byte)(m_hasExtendedFieldLengths ? 16 : 10), "");
			SetProperty(ref propList, propId, 4, 10);
		}
	}

	private void ClearAllBackOfficeSettings(ref List<ICUProperty> lstProperties)
	{
		SetProperty(ref lstProperties, 8305, 1, "");
		SetProperty(ref lstProperties, 8305, 2, "");
		SetProperty(ref lstProperties, 8312, 1, "");
		SetProperty(ref lstProperties, 8312, 2, "");
		SetProperty(ref lstProperties, 8448, 0, "");
		SetProperty(ref lstProperties, 8449, 0, "");
		SetProperty(ref lstProperties, 8450, 0, "");
		for (int i = 0; i < m_catNetworkProfile.Length; i++)
		{
			ClearNetworkProfileCategory(ref lstProperties, i);
		}
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		if (m_currentDevice == null)
		{
			return true;
		}
		bool flag = false;
		DateTime uploadEnded = DateTime.MinValue;
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			bool flag2 = false;
			string text = m_cmbBackOffices.GetValue().ToString().Trim();
			bool flag3 = text == "_MAN";
			bool flag4 = text == "_SA";
			List<ICUProperty> propList = new List<ICUProperty>();
			if (flag3 | flag4)
			{
				SetProperty(ref propList, 8310, 0, CombineBopresetMeterName(""));
				SetProperty(ref propList, 8321, 0, "ocpp/json");
				if (text == "_SA")
				{
					ClearAllBackOfficeSettings(ref propList);
				}
			}
			else if (m_cmbBackOffices.IsChanged || text != m_bopresetName)
			{
				string text2 = string.Empty;
				if (currentDevice.FirmwareVersionNumber >= new Version("4.12.0"))
				{
					if (m_allBackoffices.ContainsKey(text + "-b"))
					{
						text2 = m_allBackoffices[text + "-b"];
					}
				}
				else if (m_allBackoffices.ContainsKey(text + "-a"))
				{
					text2 = m_allBackoffices[text + "-a"];
				}
				if (string.IsNullOrEmpty(text2) && m_allBackoffices.ContainsKey(text))
				{
					text2 = m_allBackoffices[text];
				}
				if (!string.IsNullOrEmpty(text2))
				{
					ClearAllBackOfficeSettings(ref propList);
					if (propList.Count > 0)
					{
						Logger.Debug("Clearing all backoffice related properties");
						m_currentDevice.StoreProperties(propList.ToArray());
					}
					if (currentDevice.UploadFirmware(null, text2))
					{
						SetProperty(ref propList, 8310, 0, CombineBopresetMeterName(text));
						flag2 = true;
						uploadEnded = DateTime.Now;
					}
				}
			}
			else if (m_bHasNetworkProfiles)
			{
				bool flag5 = false;
				for (int i = 0; i < m_catNetworkProfile.Length; i++)
				{
					if (Convert.ToByte(m_cmbNP_Priority[i].GetValue().ToString()) != 0)
					{
						flag5 = true;
						break;
					}
				}
				if (!flag5)
				{
					m_cmbBackOffices.CustomValue = "_SA";
					SetProperty(ref propList, 8310, 0, CombineBopresetMeterName(""));
					SetProperty(ref propList, 8321, 0, "ocpp/json");
					ClearAllBackOfficeSettings(ref propList);
				}
			}
			if (!m_bHasNetworkProfiles)
			{
				string text3 = m_cmbConnectMode.GetValue().ToString();
				string strB = m_currentDevice.GetProperty(8311, 0).Value.ToString();
				if (text3 != "1")
				{
					SetProperty(ref propList, 8471, 0, false);
					m_chkProxyEnabled?.SetValue(false);
				}
				if (m_cmbConnectMode.IsChanged || string.Compare(text3, strB, ignoreCase: true) != 0)
				{
					SetProperty(ref propList, 8311, 0, text3);
					m_cmbConnectMode.CustomValue = text3;
				}
			}
			else if (flag3)
			{
				byte[] array = new byte[m_catNetworkProfile.Length];
				for (int j = 0; j < m_catNetworkProfile.Length; j++)
				{
					array[j] = byte.Parse(m_cmbNP_Priority[j].GetSelectedOption());
					flag = flag || m_cmbNP_Priority[j].IsChanged;
				}
				if (array.Distinct().Count((byte p) => p > 0) != array.Count((byte p) => p > 0))
				{
					MessageDialog.ShowError("Network profiles must have unique priorities.");
					return false;
				}
				if (currentDevice.isAHP)
				{
					for (int num = 0; num < m_catNetworkProfile.Length; num++)
					{
						if (int.Parse(m_cmbNP_OcppVersion[num].GetSelectedOption()) >= 4 && m_cmbNP_SecurityProfile[num].GetSelectedOption() == "0")
						{
							MessageDialog.ShowError($"You cannot use \"0: Default\" as security profile in Network Profile {num + 1} when using OCPP 2.0.1 or higher");
							return false;
						}
					}
				}
			}
			if (m_txtProxyPassword != null && m_txtProxyPassword.IsChanged && currentDevice.Domain.AddOrUpdateItem(EDomainItemType.KEY_PROXY_AUTHORIZATION, m_txtProxyPassword.GetValue().ToString()))
			{
				m_txtProxyPassword.ChangesAreSaved();
			}
			if (m_txtAuthorizationKey != null && m_txtAuthorizationKey.IsChanged && currentDevice.Domain.AddOrUpdateItem(EDomainItemType.KEY_BO_AUTHORIZATION, m_txtAuthorizationKey.GetValue().ToString()))
			{
				m_txtAuthorizationKey.ChangesAreSaved();
			}
			if (m_txtWifiPsk != null && m_txtWifiPsk.IsChanged)
			{
				SetProperty(ref propList, 12939, 0, m_txtWifiPsk.GetValue().ToString());
				m_txtWifiPsk.ChangesAreSaved();
			}
			if (m_chkEichrechtEnabled != null)
			{
				bool flag6 = (byte)m_chkEichrechtEnabled.GetValue() == 1;
				if (m_chkEichrechtEnabled.IsChanged || m_currentDevice.GetPropertyBool(8559, 0) != flag6)
				{
					SetProperty(ref propList, 8559, 0, flag6);
				}
				if ((m_chkEichrechtEnabled.IsChanged & flag6) && currentDevice.FirmwareVersionNumber >= new Version("6.6.0"))
				{
					SetProperty(ref propList, 8486, 0, 2);
					currentDevice.UpdateProperties(3301888u);
					SetProperty(ref propList, 12898, 1, "EUR");
				}
				if (m_chkEichrechtEnabled.IsChanged & flag6)
				{
					m_chkEichrechtEnabled.SetEnable(fEnable: false);
				}
			}
			m_currentDevice.StoreChangedProperties();
			m_currentDevice.ResetBackOfficeConfigured();
			LoadCustomBackofficeSettings(m_currentDevice);
			if ((flag2 | flag) && MessageDialog.AskQuestion("You have changed the Backoffice settings. The new settings will only be active after a restart. Do you want to restart " + currentDevice.Identity + " now?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
			{
				MainWindow mainWindow = (MainWindow)ParentWindow;
				mainWindow?.SuspendUpdateTimers(fSuspend: true);
				try
				{
					new DlgReboot(currentDevice, fAutoStartReboot: true, hardReboot: true, uploadEnded).Run();
				}
				finally
				{
					mainWindow?.SuspendUpdateTimers(fSuspend: false);
				}
			}
			m_currentDevice.UpdateCategories("ocpp", "comm", "generic", "generic2");
		}
		return true;
	}

	public override void OnRevertChanges()
	{
		LoadCustomBackofficeSettings(m_currentDevice);
		m_txtProxyPassword?.ClearValue();
		m_txtAuthorizationKey?.ClearValue();
	}

	protected override void OnUpdateTick()
	{
		if (m_currentDevice == null || !m_currentDevice.IsConnected || m_currentDevice.LastHttpStatusCode != HttpStatusCode.OK)
		{
			return;
		}
		if (m_currentDevice.HasWifiSupport)
		{
			m_currentDevice.UpdateProperties(3314176u, 3314432u);
		}
		if (m_currentDevice.LastHttpStatusCode == HttpStatusCode.OK && (m_cmbNetworkTechnology != null || m_lblInfoMobileTechnology != null))
		{
			Application.Invoke(() =>
			{
				LoadSupportedNetworkTechnologies(m_currentDevice);
			});
		}
	}

	private void ClearConfidentialAPNFields()
	{
		m_txtAPNName?.SetValue(string.Empty);
		m_txtAPNUser?.SetValue(string.Empty);
		m_txtAPNPassword?.SetValue(string.Empty);
		m_txtSimPin?.SetValue(string.Empty);
	}

	private void OnBackOfficeSelectChanged(object sender, EventArgs e)
	{
		if (m_cmbBackOffices == null || BusyChangingDevice)
		{
			return;
		}
		string text = m_cmbBackOffices.GetValue().ToString().Trim();
		if (string.Compare(text, m_bopresetName, ignoreCase: true) == 0)
		{
			return;
		}
		if (!prevSelectionPreset.HasValue)
		{
			prevSelectionPreset = text != "_MAN" && text != "_SA";
		}
		if (text == "_MAN")
		{
			if (m_bHasNetworkProfiles)
			{
				bool flag = false;
				for (int i = 0; i < m_catNetworkProfile.Length; i++)
				{
					if (Convert.ToByte(m_cmbNP_Priority[i].GetValue().ToString()) != 0)
					{
						flag = true;
						break;
					}
				}
				if (!flag && m_catNetworkProfile.Length != 0)
				{
					m_cmbNP_Priority[0].SetValue(m_dicNP_Priority[1.ToString()]);
				}
			}
			if (prevSelectionPreset == true)
			{
				ClearConfidentialAPNFields();
			}
		}
		else if (text == "_SA")
		{
			if (m_cmbConnectMode != null)
			{
				m_cmbConnectMode.SetValue("0");
			}
			else
			{
				for (int j = 0; j < m_catNetworkProfile.Length; j++)
				{
					m_cmbNP_Priority[j].SetValue(0);
				}
			}
			if (prevSelectionPreset == true)
			{
				ClearConfidentialAPNFields();
			}
		}
		prevSelectionPreset = text != "_MAN" && text != "_SA";
		OnUpdateControls(PageID);
	}

	private void UpdateOcppDictionairies(ICULanDevice device)
	{
		if (device == null)
		{
			return;
		}
		OccpVersions occpVersions = m_lsOccpVersions.Find((OccpVersions x) => (x.Version & EOccpVersion.VERSION_15) != 0);
		OccpVersions occpVersions2 = m_lsOccpVersions.Find((OccpVersions x) => (x.Version & EOccpVersion.VERSION_20) != 0);
		if (device.IsOCPPVersionSupported(occpVersions.Version))
		{
			if (!m_dicOCPP.ContainsKey(occpVersions.Key))
			{
				m_dicOCPP.Add(occpVersions.Key, occpVersions.Name);
			}
			if (!m_dicNP_Versions.ContainsKey(2.ToString()))
			{
				AddEnumToDictionary(m_dicNP_Versions, ENPVersion.OCPP15, occpVersions.Name);
			}
		}
		else
		{
			if (m_dicOCPP.ContainsKey(occpVersions.Key))
			{
				m_dicOCPP.Remove(occpVersions.Key);
			}
			if (m_dicNP_Versions.ContainsKey(2.ToString()))
			{
				m_dicNP_Versions.Remove(2.ToString());
			}
		}
		if (device.IsOCPPVersionSupported(occpVersions2.Version))
		{
			if (!m_dicOCPP.ContainsKey(occpVersions2.Key))
			{
				m_dicOCPP.Add(occpVersions2.Key, occpVersions2.Name);
			}
			if (!m_dicNP_Versions.ContainsKey(4.ToString()))
			{
				AddEnumToDictionary(m_dicNP_Versions, ENPVersion.OCPP201, occpVersions2.Name);
			}
		}
		else
		{
			if (m_dicOCPP.ContainsKey(occpVersions2.Key))
			{
				m_dicOCPP.Remove(occpVersions2.Key);
			}
			if (m_dicNP_Versions.ContainsKey(4.ToString()))
			{
				m_dicNP_Versions.Remove(4.ToString());
			}
		}
	}

	private void LoadMeasurandDataDictionairy(string ocpp)
	{
		m_dicMeterMeasurandData.Clear();
		EOccpVersion version = m_lsOccpVersions.Find((OccpVersions x) => x.Key == ocpp).Version;
		foreach (OccpMeasurand occpMeasurand in m_lsOccpMeasurands)
		{
			if ((occpMeasurand.Version & version) == 0)
			{
				continue;
			}
			List<OccpPhases> list = m_lsOccpPhases.FindAll((OccpPhases x) => x.Measurand == occpMeasurand.Measurand);
			m_dicMeterMeasurandData.Add(occpMeasurand.Measurand.ToString("d"), occpMeasurand.Key);
			if (list.Count <= 0)
			{
				continue;
			}
			int measurand = (int)occpMeasurand.Measurand;
			string text = occpMeasurand.Key + ".";
			foreach (OccpPhases item in list)
			{
				m_dicMeterMeasurandData.Add((measurand + ((int)item.Phase << 8)).ToString("d"), text + item.Key);
			}
		}
	}

	private void OnScanClicked(object sender, EventArgs e)
	{
		MainWindow mainWindow = (MainWindow)ParentWindow;
		mainWindow?.SuspendUpdateTimers(fSuspend: true);
		DlgScanWifiNetworks dlgScanWifiNetworks = new DlgScanWifiNetworks(m_currentDevice);
		try
		{
			Command command = dlgScanWifiNetworks.Run();
			if (command != null && command == Command.Ok)
			{
				m_txtWifiSsid.SetValue(dlgScanWifiNetworks.Ssid);
				m_txtWifiPsk.SetValue(dlgScanWifiNetworks.Password);
				m_cmbWifiSecurityType.SetValue((uint)dlgScanWifiNetworks.SecurityType);
			}
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "Failed to load the Wi-Fi networks");
		}
		finally
		{
			mainWindow?.SuspendUpdateTimers(fSuspend: false);
		}
	}

	private void OnNetworkProfileSelectionChanged(object sender, EventArgs e)
	{
		NetworkProfileUpdateControls();
	}

	private void NetworkProfileUpdateControls()
	{
		string text = m_cmbBackOffices.GetValue().ToString().Trim();
		bool flag = text == "_MAN";
		bool flag2 = text == "_SA";
		for (int i = 0; i < m_cmbNP_Interface.Length; i++)
		{
			if (m_cmbNP_Priority[i] != null)
			{
				bool flag3 = (Convert.ToByte(m_cmbNP_Priority[i].GetValue().ToString()) == 0) | flag2;
				foreach (UIPropertyBase item in m_catNetworkProfile[i].m_lstProperties.Where((UIPropertyBase a) => a.SubId != 14).ToList())
				{
					item.SetHide(flag3);
				}
				if (m_cmbNP_Interface[i] != null && !flag3)
				{
					byte b = Convert.ToByte(m_cmbNP_Interface[i].GetValue().ToString());
					bool flag4 = b == 0;
					bool flag5 = (b == 5) & flag;
					foreach (UIPropertyBase item2 in m_catNetworkProfile[i].m_lstProperties.Where((UIPropertyBase a) => a.SubId == 1 || a.SubId == 3 || a.SubId == 5 || a.SubId == 4).ToList())
					{
						item2.SetHide(flag4);
					}
					foreach (UIPropertyBase item3 in m_catNetworkProfile[i].m_lstProperties.Where((UIPropertyBase a) => a.SubId >= 7 && a.SubId <= 16 && a.SubId != 14).ToList())
					{
						item3.SetHide(flag4 || !flag5);
					}
				}
			}
			if (m_catNetworkProfile[i] == null || m_cmbNP_Priority[i] == null)
			{
				continue;
			}
			foreach (UIPropertyBase lstProperty in m_catNetworkProfile[i].m_lstProperties)
			{
				if (lstProperty.SubId != m_cmbNP_Priority[i].SubId && lstProperty.SubId != m_cmbNP_SecurityProfile[i].SubId)
				{
					lstProperty.SetEnable(flag);
				}
			}
			m_cmbNP_Priority[i].SetEnable(!flag2);
		}
	}

	public override bool OnUpdateControls(string pageID = "")
	{
		if (!string.IsNullOrEmpty(pageID) && string.Compare(pageID, PageID) != 0)
		{
			return true;
		}
		if (m_cmbBackOffices == null)
		{
			return false;
		}
		string text = m_cmbBackOffices.GetValue().ToString().Trim();
		bool flag = text == "_MAN";
		bool flag2 = text == "_SA";
		Dictionary<string, string> dictionary = new Dictionary<string, string>(m_dicConnectionMethod);
		if (flag2)
		{
			dictionary.Remove("1");
			dictionary.Remove("2");
			dictionary.Remove("3");
			m_cmbConnectMode?.SetValue(0);
		}
		else if (!flag)
		{
			m_lstNetworkGprs.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(fEnable: false);
			});
		}
		NetworkProfileUpdateControls();
		ICULanDevice currentDevice = m_currentDevice;
		if (m_lblConnectionMethod != null)
		{
			m_lblConnectionMethod.Hide = true;
		}
		if (currentDevice != null && !currentDevice.SupportsModem())
		{
			dictionary.Remove("2");
			dictionary.Remove("3");
		}
		if (dictionary.Count() == 0)
		{
			dictionary.Add("0", "None");
			if (m_lblConnectionMethod != null)
			{
				m_lblConnectionMethod.Hide = false;
			}
		}
		m_cmbConnectMode?.SetCustomList(dictionary);
		m_cmbConnectMode?.SetEnable(dictionary.Count() > 1);
		m_lstNetworkGprs.ForEach((UIPropertyBase a) =>
		{
			a.SetEnable(Convert.ToInt32(m_chkNetworkFixedGprs.GetValue()) != 0);
		});
		m_lstNetworkWired.ForEach((UIPropertyBase a) =>
		{
			a.SetEnable(Convert.ToInt32(m_chkNetworkFixedWired.GetValue()) != 0);
		});
		if (currentDevice.HasWifiSupport)
		{
			bool wifiEnabled = Convert.ToInt32(m_chkNetworkEnableWifi.GetValue()) != 0;
			m_lstNetworkWifi.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(wifiEnabled);
			});
			m_lstWifi.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(wifiEnabled);
			});
			m_btnScanwifi.Sensitive = wifiEnabled;
		}
		if (m_cmbProtocol != null)
		{
			m_configPanel.ShowCategory(m_catSCProfiles, !flag2 && m_cmbProtocol.GetValue().ToString() == m_lsOccpVersions.Find((OccpVersions x) => x.Version == EOccpVersion.VERSION_15).Key);
			m_configPanel.ShowCategory(m_catOCPP16SecurityExtensions, !flag2 && m_cmbProtocol.GetValue().ToString() != m_lsOccpVersions.Find((OccpVersions x) => x.Version == EOccpVersion.VERSION_15).Key);
		}
		else if (m_bHasNetworkProfiles)
		{
			m_configPanel.ShowCategory(m_catSCProfiles, !flag2);
			m_configPanel.ShowCategory(m_catOCPP16SecurityExtensions, !flag2);
			m_configPanel.ShowCategory(m_catGprs);
			m_configPanel.ShowCategory(m_catProxy);
		}
		if (!m_bHasNetworkProfiles)
		{
			m_dicConnectionMethod.TryGetValue(m_cmbConnectMode?.GetValue().ToString(), out var value);
			m_configPanel.ShowCategory(m_catGprs, value == "Mobile" || value == "Autodetect");
			m_configPanel.ShowCategory(m_catProxy, value == "Wired");
		}
		m_txtAPNName?.SetHide(!flag);
		m_txtAPNUser?.SetHide(!flag);
		m_txtAPNPassword?.SetHide(!flag);
		m_txtSimPin?.SetHide(flag);
		m_cmbProtocol?.SetHide(hide: false);
		m_lstProxy.ForEach((UIPropertyBase a) =>
		{
			a.SetEnable(Convert.ToInt32(m_chkProxyEnabled.GetValue()) != 0);
		});
		m_lstNetworkWired.ForEach((UIPropertyBase a) =>
		{
			a.SetEnable(Convert.ToInt32(m_chkNetworkFixedWired?.GetValue()) != 0);
		});
		bool enable = (currentDevice.FirmwareVersionNumber.Major < 5) | flag;
		m_txtBackofficeURL?.SetEnable(enable);
		m_txtBackofficePath?.SetEnable(enable);
		m_txtBackofficeURLGPRS?.SetEnable(enable);
		m_txtBackofficePathGPRS?.SetEnable(enable);
		m_txtBackofficeURL?.SetHide(flag2);
		m_txtBackofficePath?.SetHide(flag2);
		return true;
	}
}
