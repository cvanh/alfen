using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Net.NetworkInformation;
using System.Net.Security;
using System.Net.Sockets;
using System.Security.Authentication;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using System.Web.Script.Serialization;
using ICUIWSConnection;
using ICUNetwork.Helpers;
using ICUSettings;
using Newtonsoft.Json;
using Serilog;
using Serilog.Context;
using Tmds.MDns;

namespace ICUNetwork;

public class ICULanDevice : ICUDevice
{
	private ILogger Logger = Serilog.Log.ForContext<ICULanDevice>();

	private static HttpClient m_httpClient = null;

	private static readonly object m_httpClientLock = new object();

	private CookieContainer m_cookieContainer = new CookieContainer();

	public const int s_nRequestTimeout = 5000;

	public const int s_nLoginTimeoutInMs = 15000;

	protected const int s_maxRequestRetries = 4;

	protected const int s_nFirmwareUploadTimeout = 900000;

	protected const int s_nFirmwareUploadResetTimeout = 170000;

	protected const int s_nRequestLongTimeOut = 10000;

	protected const int s_nUpdateInterval = 500;

	protected const int s_nRebootTimerInterval = 250;

	protected const int s_nRebootMaxTime = 120000;

	protected const int s_nDefaultLockTimeOut = 5000;

	protected static int s_nMaxLogoWidth = 320;

	protected static int s_nMaxLogoHeight = 160;

	protected static int s_nMaxLogoWidthLarge = 800;

	protected static int s_nMaxLogoHeightLarge = 350;

	protected static int s_nMaxLogLines = 10000;

	private Version m_vFirmwareVersion;

	protected Action<ICULanDevice, ACEWebLoginData> m_cbRequestLoginCredentials;

	protected DateTime m_lastSuccessfulFileUpload = DateTime.MinValue;

	protected Action<ICULanDevice, Exception> m_cbRebootCallback;

	protected Action<ICULanDevice, double> m_cbRebootProgress;

	protected double m_dRebootProgress;

	protected Timer m_timReboot;

	private readonly object m_lockRebootProgress = new object();

	private readonly ReentrantAsyncLock m_connectionLock = new ReentrantAsyncLock();

	private readonly ReaderWriterLockSlim _propertyLock = new ReaderWriterLockSlim();

	private bool m_fUniquePasswordGracePeriod;

	private bool _isRebooting;

	private bool? _hasBackOfficeConfigured;

	private bool _isUploading;

	private HttpStatusCode _lastHttpStatusCode;

	private ICUDeviceModel m_modelType;

	public string AccessToken { get; set; }

	public string RefreshToken { get; set; }

	public string RawFirmwareVersion { get; private set; }

	public IPAddress IPAddress { get; private set; }

	public int Port { get; private set; }

	public string HostName { get; set; }

	public string Identity { get; set; }

	public string SCNNetwork { get; set; }

	public bool HasSCNNetwork => !string.IsNullOrEmpty(SCNNetwork);

	public ServiceAnnouncement Annoucement { get; set; }

	public ICULanLog Log { get; set; }

	public bool Discovered { get; set; }

	public bool IsRebooting
	{
		get
		{
			return ReadLocked(() => _isRebooting);
		}
		private set
		{
			WriteLocked(() =>
			{
				_isRebooting = value;
			});
		}
	}

	public bool HasFixedLanIpAddress
	{
		get
		{
			if (!HasProperty(8317, 2) || GetPropertyInt(8317, 2) != 1)
			{
				if (HasProperty(12932) && GetPropertyInt(12932, 0) == 1)
				{
					return GetPropertyInt(12933, 2) == 1;
				}
				return false;
			}
			return true;
		}
	}

	public bool HasBackOfficeConfigured
	{
		get
		{
			if (_hasBackOfficeConfigured.HasValue)
			{
				return _hasBackOfficeConfigured.Value;
			}
			if (HasProperty(8432, 14))
			{
				_hasBackOfficeConfigured = HasBackOfficeConfiguredInProfiles;
				return _hasBackOfficeConfigured.Value;
			}
			_hasBackOfficeConfigured = (HasProperty(8432, 14) ? HasBackOfficeConfiguredInProfiles : (HasProperty(8311) && GetPropertyInt(8311, 0) != 0));
			return _hasBackOfficeConfigured.Value;
		}
	}

	private bool HasBackOfficeConfiguredInProfiles
	{
		get
		{
			if (!HasProperty(8432, 14))
			{
				return false;
			}
			ushort[] array = new ushort[4] { 8432, 8433, 8434, 8435 };
			foreach (ushort propId in array)
			{
				if (GetPropertyInt(propId, 14) > 0 && GetPropertyInt(propId, 6) > 0)
				{
					return true;
				}
			}
			return false;
		}
	}

	public bool IsUploading
	{
		get
		{
			return ReadLocked(() => _isUploading);
		}
		private set
		{
			WriteLocked(() =>
			{
				_isUploading = value;
			});
		}
	}

	public bool IsLoggedIn
	{
		get
		{
			return ReadLocked(() => LoginData.IsLoggedIn);
		}
		private set
		{
			WriteLocked(() =>
			{
				LoginData.IsLoggedIn = value;
			});
		}
	}

	public bool HasSuccessfulLogin { get; set; }

	public bool IsRequestingLoginData { get; set; }

	public bool IsDefaultCategoryCollected { get; set; }

	public ICUWhiteList Whitelist { get; set; }

	public ICUTransactions Transactions { get; set; }

	public ICUChargingProfiles ChargingProfiles { get; set; }

	public HttpStatusCode LastHttpStatusCode
	{
		get
		{
			return ReadLocked(() => _lastHttpStatusCode);
		}
		private set
		{
			WriteLocked(() =>
			{
				_lastHttpStatusCode = value;
			});
		}
	}

	public DateTime LastValidResponse { get; private set; }

	private string LastCommand { get; set; }

	public string LastUploadError { get; set; }

	public bool HasCentralMeter { get; set; }

	public bool HasSmartMeter { get; set; }

	public ICUMasterTag MasterTag { get; set; }

	public ICUModbusRegmap ModbusTcpIpRegmap { get; set; }

	public bool AllowObjectIDUpdate { get; set; }

	public ICUDomain Domain { get; set; }

	public bool IsHTTPS => Port == 443;

	public string Protocol
	{
		get
		{
			if (!IsHTTPS)
			{
				return "http";
			}
			return "https";
		}
	}

	public ACEWebLoginData LoginData { get; private set; } = new ACEWebLoginData();

	public object UserData { get; set; }

	public bool IsManuallyAdded { get; }

	public DateTime LastUpdate
	{
		get
		{
			if (HasProperty(8583))
			{
				DateTime unixEpoch = ICUDevice.UnixEpoch;
				return unixEpoch.AddMilliseconds(GetPropertyDouble(8583, 0));
			}
			return DateTime.UtcNow;
		}
	}

	public Version FirmwareVersionNumber
	{
		get
		{
			if (m_vFirmwareVersion == null)
			{
				string[] array = GetPropertyString(4106, 0, 0).Split(new char[1] { '-' });
				if (array.Count() < 2)
				{
					if (!IsHTTPS)
					{
						return new Version(4, 9, 0);
					}
					return new Version(4, 10, 0);
				}
				try
				{
					string version = array[0].Replace("X", "99");
					m_vFirmwareVersion = new Version(version);
				}
				catch (Exception ex)
				{
					Logger.Debug(ex, ex.Message);
					m_vFirmwareVersion = new Version();
				}
			}
			return m_vFirmwareVersion;
		}
	}

	public bool isAHP
	{
		get
		{
			if (!Model.StartsWith("AHWP", ignoreCase: true, CultureInfo.InvariantCulture))
			{
				return Model.StartsWith("AHP", ignoreCase: true, CultureInfo.InvariantCulture);
			}
			return true;
		}
	}

	public bool isAHPV2
	{
		get
		{
			if (!Model.StartsWith("AHP02", ignoreCase: true, CultureInfo.InvariantCulture))
			{
				return Model.StartsWith("AHPDC", ignoreCase: true, CultureInfo.InvariantCulture);
			}
			return true;
		}
	}

	public bool isDC => isEcogDC;

	public bool isEcogDC
	{
		get
		{
			if (isAHPV2 && HasProperty(33034, 1) && (GetPropertyUInt(33034, 1) == 1 || GetPropertyUInt(33034, 2) == 1))
			{
				return true;
			}
			return false;
		}
	}

	public string[] getUpdateFileTypes
	{
		get
		{
			if (isAHP)
			{
				return new string[2] { ".tfw", ".tcf" };
			}
			return new string[2] { ".fwi", ".fwu" };
		}
	}

	public bool IsUniquePasswordRequired
	{
		get
		{
			Version vFirmwareVersion = m_vFirmwareVersion;
			if ((object)vFirmwareVersion == null || vFirmwareVersion.Major < 5)
			{
				return isAHP;
			}
			return true;
		}
		set
		{
			if (value)
			{
				m_vFirmwareVersion = new Version(5, 0, 0);
			}
		}
	}

	public bool ShouldLoginUsingUniquePassword
	{
		get
		{
			if (m_fUniquePasswordGracePeriod)
			{
				return false;
			}
			return IsUniquePasswordRequired;
		}
	}

	public override ICUDeviceModel ModelType
	{
		get
		{
			if (m_modelType == ICUDeviceModel.Unknown)
			{
				m_modelType = ICUDeviceModelExtension.From(HostName, NumberOfSockets, SocketTypes[0] == 0);
			}
			return m_modelType;
		}
	}

	public bool IsOldModelWithDisplay
	{
		get
		{
			if (ModelType != ICUDeviceModel.Eve_Mini && ModelType != ICUDeviceModel.NG910_60014)
			{
				return ModelType == ICUDeviceModel.NG910_60034;
			}
			return true;
		}
	}

	public override string Model => ICUDeviceModelExtension.ToString(ModelType, HostName);

	public bool HasLargeDisplay => MaxLogoWidth > 320;

	public bool HasDisplay
	{
		get
		{
			if (MaxLogoWidth <= 0 && !IsOldModelWithDisplay)
			{
				return IsDualPG;
			}
			return true;
		}
	}

	public int MaxLogoWidth
	{
		get
		{
			int result = 0;
			if (GetProperty(3301377u) != null)
			{
				int propertyInt = GetPropertyInt(12896, 3);
				int propertyInt2 = GetPropertyInt(12896, 4);
				if (propertyInt > 0 && propertyInt2 > 0)
				{
					result = propertyInt;
				}
			}
			else if (IsOldModelWithDisplay)
			{
				result = s_nMaxLogoWidth;
			}
			return result;
		}
	}

	public int MaxLogoHeight
	{
		get
		{
			int result = 0;
			if (GetProperty(3301377u) != null)
			{
				int propertyInt = GetPropertyInt(12896, 3);
				int propertyInt2 = GetPropertyInt(12896, 4);
				if (propertyInt > 0 && propertyInt2 > 0)
				{
					result = propertyInt2;
				}
			}
			else if (IsOldModelWithDisplay)
			{
				result = s_nMaxLogoHeight;
			}
			return result;
		}
	}

	public override string Identification => Identity;

	public override string DisplayNameLine2 => HostName;

	public override string Address => $"{IPAddress}";

	public bool IsTwin
	{
		get
		{
			if (ModelType >= ICUDeviceModel.NG920_52001)
			{
				return ModelType <= ICUDeviceModel.NG920_52571;
			}
			return false;
		}
	}

	public bool IsDualPG
	{
		get
		{
			if (ModelType >= ICUDeviceModel.NG920_62001)
			{
				return ModelType <= ICUDeviceModel.NG920_62005;
			}
			return false;
		}
	}

	public bool IsEichrechtEnabled
	{
		get
		{
			if (HasProperty(8554))
			{
				return GetPropertyBool(8554, 0);
			}
			return false;
		}
	}

	public int LoginTimeoutInSeconds => 15;

	public bool HasWifiSupport
	{
		get
		{
			if (HasProperty(12943))
			{
				return GetPropertyBool(12943, 0);
			}
			return false;
		}
	}

	private void ClearTokens()
	{
		AccessToken = string.Empty;
		RefreshToken = string.Empty;
	}

	private EAlbConfiguration GetTcpAlbConfiguration()
	{
		if (HasProperty(9520, 1) && GetPropertyInt(9520, 1) == 3)
		{
			return EAlbConfiguration.EMS;
		}
		if (HasProperty(9507, 2) && GetPropertyInt(9507, 2) == 1)
		{
			return EAlbConfiguration.SM_Tcp_Socomec;
		}
		return EAlbConfiguration.SM_Tcp_Custom;
	}

	private EAlbConfiguration GetP1AlbConfiguration()
	{
		return (HasProperty(8593, 1) ? GetPropertyInt(8593, 1) : (-1)) switch
		{
			0 => EAlbConfiguration.SM_P1_Serial, 
			1 => EAlbConfiguration.SM_P1_Telnet, 
			2 => EAlbConfiguration.SM_P1_HomeWizard, 
			_ => EAlbConfiguration.SM_P1_Unknown, 
		};
	}

	public void UpdateAnalyticsProperties()
	{
		UpdateProperties(2122752u, 2123520u, 2123776u, 5379840u, 2433794u, 2199809u, 2204160u, 2117888u, 2118400u, 2119168u, 3309569u, 3309570u, 3309571u, 2129154u, 3310592u, 3310850u, 2158598u, 2158854u, 2159110u, 2159366u, 2158606u, 2158862u, 2159118u, 2159374u, 2127616u);
	}

	public EAlbConfiguration GetAlbConfiguration()
	{
		if (!IsFeatureUnlocked(IWSFirmwareFeatures.Features.LoadBalancing_Active))
		{
			return EAlbConfiguration.NoLicense;
		}
		if (HasProperty(8292) && GetPropertyInt(8292, 0, 2) == 0)
		{
			return EAlbConfiguration.NotConfigered;
		}
		return (HasProperty(21015) ? GetPropertyInt(21015, 0) : 0) switch
		{
			4 => GetTcpAlbConfiguration(), 
			5 => GetP1AlbConfiguration(), 
			6 => EAlbConfiguration.SM_Rtu, 
			7 => EAlbConfiguration.SM_TicLinky, 
			_ => EAlbConfiguration.NotConfigered, 
		};
	}

	public ESolarChargingModes GetSolarChargingMode()
	{
		if (!HasProperty(12928, 1))
		{
			return ESolarChargingModes.SOLAR_CHARGING_OFF;
		}
		return (ESolarChargingModes)GetPropertyInt(12928, 1);
	}

	public void ResetBackOfficeConfigured()
	{
		_hasBackOfficeConfigured = null;
	}

	public bool IsFeatureUnlocked(IWSFirmwareFeatures.Features feature)
	{
		return IWSFirmwareFeatures.IsFeatureUnlocked(FirmwareVersionNumber, GetPropertyUInt(8610, 0), feature, isAHP);
	}

	public ICULanDevice(LANConnection connection, IPAddress address, int portNumber, ServiceAnnouncement announcement = null, bool isManuallyAdded = false)
		: base(connection, address.ToString())
	{
		ServicePointManager.ServerCertificateValidationCallback = ValidateServerCertificate;
		ServicePointManager.SecurityProtocol = SecurityProtocolType.Tls12 | SecurityProtocolType.Tls13;
		ServicePointManager.MaxServicePointIdleTime = 600000;
		AllowObjectIDUpdate = true;
		ReInitialize(address, portNumber, announcement, newDevice: true);
		LastValidResponse = DateTime.UtcNow;
		IsManuallyAdded = isManuallyAdded;
	}

	public void ReInitialize(IPAddress address, int portNumber, ServiceAnnouncement announcement, bool newDevice)
	{
		if (newDevice || !IPAddress.Equals(address) || Port != portNumber)
		{
			IPAddress = address;
			Port = portNumber;
			AllowObjectIDUpdate = newDevice;
			IsLoggedIn = false;
			IsDefaultCategoryCollected = false;
			Log = new ICULanLog(this);
		}
		Annoucement = announcement;
		NumberOfSockets = 1;
		NumberOfFeederCables = 1;
		SocketTypes = new int[2];
		SocketTypes[0] = 0;
		SocketTypes[1] = 0;
		if (announcement != null && announcement.Txt.Count > 0)
		{
			Logger.Debug("TXT received: {0}", string.Join("; ", announcement.Txt));
			foreach (string item in announcement.Txt)
			{
				string[] array = item.Split(new char[1] { '=' });
				if (array.Count() <= 1)
				{
					continue;
				}
				switch (array[0].ToLowerInvariant())
				{
				case "identity":
					Identity = string.Join("=", array.Skip(1));
					break;
				case "scnnetwork":
					SCNNetwork = array[1].Trim();
					break;
				case "type":
				{
					string[] array3 = array[1].Split(new char[1] { '.' });
					if (array3.Count() > 2)
					{
						if (int.TryParse(array3[0], out var result))
						{
							NumberOfSockets = result;
						}
						if (int.TryParse(array3[1], out var result2))
						{
							SocketTypes[0] = result2;
						}
						if (int.TryParse(array3[2], out var result3))
						{
							SocketTypes[1] = result3;
						}
					}
					break;
				}
				case "fwversion":
				{
					string[] array2 = array[1].Trim().Split(new char[1] { '-' });
					RawFirmwareVersion = array[1];
					if (array2.Length != 0)
					{
						string text = array2[0].ToLowerInvariant().Replace("x", "99");
						if (!Version.TryParse(text, out m_vFirmwareVersion))
						{
							Logger.Debug("Error: Failed to parse firmware version! Firwaretext= {Version}", text);
						}
					}
					break;
				}
				case "euaenabled":
					if (array[1] == "1" && EndUserAccessType != EndUserAccessType.Configured)
					{
						EndUserAccessType = EndUserAccessType.Enabled;
					}
					else
					{
						EndUserAccessType = EndUserAccessType.Disabled;
					}
					break;
				case "euaconfigured":
					if (array[1] == "1")
					{
						EndUserAccessType = EndUserAccessType.Configured;
					}
					break;
				}
			}
			string[] array4 = announcement.Hostname.Split(new char[1] { '-' });
			string empty = string.Empty;
			if (array4.Length > 1)
			{
				empty = array4.Last();
				SerialNumber = empty;
			}
			Name = Identity;
			Discovered = true;
			HostName = announcement.Hostname;
			NumberOfFeederCables = GetNumberOfFeederCables();
		}
		else
		{
			Discovered = false;
			HostName = "";
		}
		if (string.IsNullOrEmpty(Identity))
		{
			Identity = HostName;
		}
		if (!IsRebooting)
		{
			return;
		}
		DumpActiveConnections(IPAddress);
		IsLoggedIn = false;
		IsDefaultCategoryCollected = false;
		Log = new ICULanLog(this);
		Whitelist = new ICUWhiteList(this);
		Transactions = new ICUTransactions(this);
		ChargingProfiles = new ICUChargingProfiles(this);
		MasterTag = new ICUMasterTag(this);
		ModbusTcpIpRegmap = new ICUModbusRegmap(this);
		Domain = new ICUDomain(this);
		if (IsUploading)
		{
			return;
		}
		ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
		Task<(bool, HttpStatusCode, string)> task = Task.Run(async () =>
		{
			try
			{
				return ((bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content))(await LoginRequest());
			}
			finally
			{
				finishedEvent.Set();
			}
		});
		while (!finishedEvent.WaitOne(10))
		{
		}
		(bool, HttpStatusCode, string) result4 = task.GetAwaiter().GetResult();
		if (result4.Item2 != HttpStatusCode.RequestTimeout && result4.Item2 != HttpStatusCode.ServiceUnavailable)
		{
			FinishedRebooting();
			Logger.Debug("* Device finished rebooting: {HostName} at {IpAddress}", announcement?.Hostname, address);
		}
	}

	public void Allocate()
	{
		Whitelist = new ICUWhiteList(this);
		Transactions = new ICUTransactions(this);
		ChargingProfiles = new ICUChargingProfiles(this);
		MasterTag = new ICUMasterTag(this);
		ModbusTcpIpRegmap = new ICUModbusRegmap(this);
		Domain = new ICUDomain(this);
		m_cookieContainer = new CookieContainer();
		LastValidResponse = DateTime.UtcNow;
		LastHttpStatusCode = HttpStatusCode.OK;
	}

	public void Deallocate()
	{
		try
		{
			Whitelist = null;
			Transactions = null;
			ChargingProfiles = null;
			MasterTag = null;
			ModbusTcpIpRegmap = null;
			Domain = null;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
	}

	public void FinishedRebooting()
	{
		if (m_cbRebootCallback != null)
		{
			m_cbRebootProgress?.Invoke(this, 1.0);
			m_cbRebootCallback(this, null);
			m_cbRebootProgress = null;
			m_cbRebootCallback = null;
			m_timReboot.Dispose();
		}
		IsRebooting = false;
		LastHttpStatusCode = HttpStatusCode.OK;
	}

	private async Task<(EWebRequestState, string responseData)> HandleIncomingResponse(HttpResponseMessage result)
	{
		if (result.StatusCode == HttpStatusCode.NotFound)
		{
			return (EWebRequestState.VALID_RESPONSE, responseData: string.Empty);
		}
		if (result.Content != null)
		{
			string item = (await result.Content.ReadAsStringAsync()).Replace(":nan", ":null");
			return ((!result.IsSuccessStatusCode) ? EWebRequestState.INVALID_RESPONSE : EWebRequestState.VALID_RESPONSE, responseData: item);
		}
		return (EWebRequestState.INVALID_RESPONSE, responseData: result.ReasonPhrase);
	}

	private bool ValidateServerCertificate(object sender, X509Certificate certificate, X509Chain chain, SslPolicyErrors sslPolicyErrors)
	{
		try
		{
			chain.ChainPolicy.ExtraStore.Add(CertificateStore.GetCertificate(CertificateStore.CertificateType.ACE_CA_ROOT));
			chain.Build(new X509Certificate2(certificate));
			X509ChainStatus[] chainStatus = chain.ChainStatus;
			for (int i = 0; i < chainStatus.Length; i++)
			{
				X509ChainStatus x509ChainStatus = chainStatus[i];
				X509ChainStatusFlags status = x509ChainStatus.Status;
				if ((uint)(status - 1) > 1u && status != X509ChainStatusFlags.UntrustedRoot && status != X509ChainStatusFlags.CtlNotTimeValid)
				{
					Logger.Debug("Server certificate is invalid. {Status}: {StatusInformation}", x509ChainStatus.Status, x509ChainStatus.StatusInformation);
					return false;
				}
			}
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, "Failed to validate server certificate: {Message}", ex.Message);
			return false;
		}
		return true;
	}

	private void CreateHttpClient(Uri uri)
	{
		lock (m_httpClientLock)
		{
			if (m_httpClient == null)
			{
				HttpClientHandler httpClientHandler = new HttpClientHandler
				{
					MaxConnectionsPerServer = 1
				};
				if (!IsHTTPS)
				{
					if (m_cookieContainer == null)
					{
						m_cookieContainer = new CookieContainer();
					}
					httpClientHandler.CookieContainer = m_cookieContainer;
				}
				m_httpClient = new HttpClient(new LogHttpClientHandler(httpClientHandler))
				{
					BaseAddress = new Uri(uri.GetLeftPart(UriPartial.Authority)),
					Timeout = TimeSpan.FromMinutes(15.0)
				};
				m_httpClient.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
				Logger.Debug("HttpClient created");
				LastValidResponse = DateTime.UtcNow;
			}
		}
		if (UseBearerAuthentication())
		{
			m_httpClient.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Bearer", AccessToken);
		}
		else if (UseBasicAuthentication())
		{
			byte[] bytes = Encoding.ASCII.GetBytes(LoginData.Username + ":" + LoginData.Password);
			m_httpClient.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Basic", Convert.ToBase64String(bytes));
		}
		else
		{
			m_httpClient.DefaultRequestHeaders.Authorization = null;
		}
	}

	private bool UseBearerAuthentication()
	{
		if (IsHTTPS)
		{
			return !string.IsNullOrEmpty(AccessToken);
		}
		return false;
	}

	private bool UseBasicAuthentication()
	{
		if (!IsHTTPS)
		{
			return HasCredentials();
		}
		return false;
	}

	private bool HasCredentials()
	{
		if (!string.IsNullOrEmpty(LoginData.Username))
		{
			return !string.IsNullOrEmpty(LoginData.Password);
		}
		return false;
	}

	public (EWebRequestState RequestState, HttpStatusCode HttpStatusCode, string ResponseData) ExecuteWebRequest(string currentCommand, string parameters = "", object content = null, int requestTimeoutInMs = 5000, int maxRetries = 4, bool suppressPopups = false, CancellationToken cancellationToken = default(CancellationToken))
	{
		using (LogContext.PushProperty("SerialNumber", SerialNumber?.GetHashCode()))
		{
			using (LogContext.PushProperty("SCNNetwork", SCNNetwork?.GetHashCode()))
			{
				using (LogContext.PushProperty("Firmware", FirmwareVersionNumber))
				{
					(EWebRequestState, HttpStatusCode, string) tuple = ExecuteAndWait(ExecuteWebRequestInternal(currentCommand, parameters, content, requestTimeoutInMs, maxRetries, suppressPopups, cancellationToken));
					LastCommand = currentCommand;
					LastHttpStatusCode = tuple.Item2;
					if (tuple.Item1 == EWebRequestState.VALID_RESPONSE)
					{
						LastValidResponse = DateTime.UtcNow;
					}
					return (RequestState: tuple.Item1, HttpStatusCode: tuple.Item2, ResponseData: tuple.Item3);
				}
			}
		}
	}

	private Func<Task<(EWebRequestState currentState, HttpStatusCode currentHttpStatusCode, string currentResponseData)>> ExecuteWebRequestInternal(string currentCommand, string parameters, object content, int requestTimeoutInMs, int maxRetries, bool suppressPopups, CancellationToken cancellationToken)
	{
		return async () =>
		{
			EWebRequestState currentState = EWebRequestState.UNSUCCESSFUL;
			HttpStatusCode currentHttpStatusCode = HttpStatusCode.InternalServerError;
			string currentResponseData = string.Empty;
			Uri uri = GetUri(currentCommand, parameters);
			if (requestTimeoutInMs < 5000)
			{
				requestTimeoutInMs = 5000;
			}
			SocketException ex3 = default;
			for (int retry = 0; retry < maxRetries; retry++)
			{
				CancellationTokenSource ctsTimeout = new CancellationTokenSource(requestTimeoutInMs);
				CancellationTokenSource linkedCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken, ctsTimeout.Token);
				try
				{
					_ = 1;
					try
					{
						HttpResponseMessage httpResponseMessage = await SendHttpRequest(uri, content, fireAndForget: false, linkedCts.Token);
						currentHttpStatusCode = httpResponseMessage.StatusCode;
						if (httpResponseMessage.Content != null)
						{
							(currentState, currentResponseData) = await HandleIncomingResponse(httpResponseMessage);
						}
						else
						{
							currentState = ((httpResponseMessage.StatusCode != HttpStatusCode.OK) ? EWebRequestState.INVALID_RESPONSE : EWebRequestState.VALID_RESPONSE);
							currentResponseData = string.Empty;
						}
					}
					catch (Exception ex) when ((ex is TaskCanceledException || ex is AggregateException || ex is OperationCanceledException) && !(ex.InnerException is HttpRequestException))
					{
						if (ex.InnerException?.InnerException?.InnerException is AuthenticationException)
						{
							Logger.Debug(ex, "'{Url}': invalid Certificate HttpStatusCode={HttpStatusCode} => NotAcceptable", uri, currentHttpStatusCode);
							currentHttpStatusCode = HttpStatusCode.NotAcceptable;
						}
						else if (ex is OperationCanceledException)
						{
							Logger.Debug(ex, "'{Url}': request timeout after {Timeout}ms, could be network cable problem or CS webserver is not yet running because of a reboot HttpStatusCode={HttpStatusCode} => RequestTimeout", uri, requestTimeoutInMs, currentHttpStatusCode);
							currentHttpStatusCode = HttpStatusCode.RequestTimeout;
						}
						else
						{
							Logger.Debug(ex, "'{Url}': unhandled exception HttpStatusCode={HttpStatusCode} => Unused, State={State}", uri, currentHttpStatusCode, currentState);
							currentHttpStatusCode = HttpStatusCode.Unused;
						}
					}
					catch (HttpRequestException ex2) when (((Func<bool>)delegate
					{
						// Could not convert BlockContainer to single expression
						ex3 = ex2.InnerException as SocketException;
						return ex3 != null;
					}).Invoke())
					{
						Logger.Debug(ex3, ex3.Message);
						if (ex3.Message.CompareTo("A socket operation was attempted to an unreachable host") != 0)
						{
							currentHttpStatusCode = HttpStatusCode.RequestTimeout;
						}
					}
					catch (Exception ex4)
					{
						Logger.Error(ex4, ex4.Message);
					}
				}
				finally
				{
					linkedCts.Dispose();
					ctsTimeout.Dispose();
					if (currentState != EWebRequestState.VALID_RESPONSE)
					{
						currentState = await HandleUnsuccessfulRequest(LastHttpStatusCode, currentCommand, currentHttpStatusCode, retry >= maxRetries - 1, suppressPopups);
					}
				}
				if (currentState != EWebRequestState.RETRY_REQUEST)
				{
					break;
				}
				Logger.Warning("'{Url}': retrying request ({Retry} / {MaxRetries})", uri, retry, maxRetries - 1);
			}
			return (currentState: currentState, currentHttpStatusCode: currentHttpStatusCode, currentResponseData: currentResponseData);
		};
	}

	private static bool DisposeHttpClient()
	{
		if (m_httpClient != null)
		{
			m_httpClient.Dispose();
			m_httpClient = null;
			return true;
		}
		return false;
	}

	private T ExecuteAndWait<T>(Func<Task<T>> func)
	{
		ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
		Task<T> task = Task.Run(async () =>
		{
			try
			{
				return await m_connectionLock.WithLock(async () => await func());
			}
			finally
			{
				finishedEvent.Set();
			}
		});
		while (!finishedEvent.WaitOne(10))
		{
		}
		return task.GetAwaiter().GetResult();
	}

	private Uri GetUri(string command, string parameters = null)
	{
		TryGetUri(command, parameters, out var uri);
		return uri;
	}

	private bool TryGetUri(string command, string parameters, out Uri uri)
	{
		string text = string.Format("{0}://{1}:{2}/api/{3}", new object[4] { Protocol, IPAddress, Port, command });
		if (!string.IsNullOrEmpty(parameters))
		{
			text = text + "?" + parameters;
		}
		if (!Uri.IsWellFormedUriString(text, UriKind.Absolute))
		{
			Logger.Error("Incorrect uri: {Url}", text);
			uri = null;
			return false;
		}
		uri = new Uri(text);
		return true;
	}

	private async Task<HttpResponseMessage> SendHttpRequest(Uri uri, object body = null, bool fireAndForget = false, CancellationToken cancellationToken = default(CancellationToken))
	{
		CreateHttpClient(uri);
		if (body == null)
		{
			return await m_httpClient.GetAsync(uri, cancellationToken);
		}
		if (body is string)
		{
			return await SendStandardPost(uri, body, fireAndForget, cancellationToken);
		}
		if (body is byte[])
		{
			return await SendMultipartFormDataContent(uri, body, cancellationToken);
		}
		throw new NotSupportedException("HttpRequest not supported");
	}

	private static async Task<HttpResponseMessage> SendStandardPost(Uri uri, object body, bool fireAndForget, CancellationToken cancellationToken)
	{
		StringContent content = new StringContent(body.ToString(), Encoding.UTF8, "application/json");
		HttpRequestMessage httpRequestMessage = new HttpRequestMessage(HttpMethod.Post, uri)
		{
			Content = content
		};
		if (fireAndForget)
		{
			httpRequestMessage.Headers.ConnectionClose = true;
		}
		return await m_httpClient.SendAsync(httpRequestMessage, cancellationToken);
	}

	private async Task<HttpResponseMessage> SendMultipartFormDataContent(Uri uri, object body, CancellationToken cancellationToken)
	{
		byte[] array = body as byte[];
		MultipartFormDataContent multipartFormDataContent = new MultipartFormDataContent();
		if (IsHTTPS)
		{
			int num = 4096;
			int num2 = array.Length;
			int num3 = 0;
			while (num2 > 0)
			{
				ByteArrayContent content = new ByteArrayContent(array, num3, (num2 > num) ? num : num2);
				multipartFormDataContent.Add(content, "\"firmwarefile\"", "\"filename\"");
				num3 += num;
				num2 -= num;
			}
		}
		else
		{
			ByteArrayContent content2 = new ByteArrayContent(array);
			multipartFormDataContent.Add(content2, "\"firmwarefile\"", "\"filename\"");
		}
		return await m_httpClient.PostAsync(uri, multipartFormDataContent, cancellationToken);
	}

	private async Task<EWebRequestState> HandleUnsuccessfulRequest(HttpStatusCode previousStatusCode, string currentCommand, HttpStatusCode currentStatusCode, bool isLastRetry, bool suppressPopups = false)
	{
		Logger.Debug("HandleUnsuccessfulRequest Command={Command}, StatusCode={StatusCode}, IsLastRetry={IsLastRetry}", currentCommand, currentStatusCode, isLastRetry);
		BaseConnection connection = Connection;
		if (!(connection is LANConnection lanCon) || (IsRebooting && previousStatusCode == HttpStatusCode.RequestTimeout))
		{
			return EWebRequestState.UNSUCCESSFUL;
		}
		switch (currentStatusCode)
		{
		case HttpStatusCode.Unauthorized:
		case HttpStatusCode.Forbidden:
		case HttpStatusCode.TooManyRequests:
			IsDefaultCategoryCollected = false;
			IsLoggedIn = false;
			AccessToken = null;
			if (currentCommand.CompareTo("login") != 0 && currentCommand.CompareTo("logout") != 0 && currentCommand.CompareTo("prc") != 0 && currentStatusCode != HttpStatusCode.TooManyRequests)
			{
				ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
				Task<(bool, HttpStatusCode, string)> task = Task.Run(async () =>
				{
					try
					{
						(bool, HttpStatusCode, string) result = await LoginRequest();
						_ = result.Item2;
						_ = 403;
						return ((bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content))result;
					}
					finally
					{
						finishedEvent.Set();
					}
				});
				while (!finishedEvent.WaitOne(10))
				{
				}
				if (task.GetAwaiter().GetResult().Item1)
				{
					Logger.Debug("Login success, retrying command {Command}", currentCommand);
					return EWebRequestState.RETRY_REQUEST;
				}
			}
			return EWebRequestState.UNSUCCESSFUL;
		case HttpStatusCode.BadRequest:
		case HttpStatusCode.NotImplemented:
			if (isLastRetry && !suppressPopups)
			{
				lanCon.CallErrorHandler("Device '" + Identification + "' does not support the request, please contact support.", this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		case HttpStatusCode.NotAcceptable:
			if (isLastRetry && !suppressPopups)
			{
				lanCon.CallErrorHandler("Device '" + Identification + "' has an invalid or expired certificate, please contact support.", this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		case HttpStatusCode.MultipleChoices:
		case HttpStatusCode.Conflict:
		case HttpStatusCode.InternalServerError:
		case HttpStatusCode.ServiceUnavailable:
			IsDefaultCategoryCollected = false;
			IsLoggedIn = false;
			if (isLastRetry && !suppressPopups)
			{
				lanCon.CallErrorHandler($"Device '{Identification}'/{IPAddress} connection lost.", this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		case HttpStatusCode.Unused:
		case HttpStatusCode.RequestTimeout:
		case HttpStatusCode.GatewayTimeout:
		{
			IsDefaultCategoryCollected = false;
			IsLoggedIn = false;
			if (IsUploading || (IsRebooting && currentCommand.CompareTo("reboot") == 0))
			{
				Logger.Debug("Skip charger availability checks for {DeviceName} because it is currently rebooting/updating", Identification);
				return EWebRequestState.UNSUCCESSFUL;
			}
			(bool, string) tuple = await CheckNetworkAndPing(IPAddress);
			if (!tuple.Item1)
			{
				string item = tuple.Item2;
				if (isLastRetry && !suppressPopups)
				{
					lanCon.CallErrorHandler(item, this);
				}
			}
			if (!isLastRetry)
			{
				return EWebRequestState.RETRY_REQUEST;
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		default:
			return EWebRequestState.RETRY_REQUEST;
		}
	}

	public async Task<EWebRequestState> HandleUnsuccessfulLoginRequest(HttpStatusCode currentStatusCode, string currentResponseData, DateTime previousValidResponse, int timeoutInMilliSeconds, bool suppressPopups, bool isUniquePasswordRequired, Action<string, ICULanDevice> showErrorCallback)
	{
		Logger.Debug("HandleUnsuccessfulLoginRequest Identification={Identification} StatusCode={StatusCode}, TimeOut={TimeOut}, SuppressPopups={SuppressPopups}, IsUniquePasswordRequired={IsUniquePasswordRequired}, LastValidResponse={LastValidResponse}", Identification, currentStatusCode, timeoutInMilliSeconds, suppressPopups, isUniquePasswordRequired, previousValidResponse);
		if (!isUniquePasswordRequired)
		{
			if (!IsRebooting && (DateTime.UtcNow - previousValidResponse).TotalMilliseconds > (double)(timeoutInMilliSeconds * 3))
			{
				string text = $"Login failed: Device '{Identification}' has given no valid response since {previousValidResponse.ToLocalTime()}, please reboot the device.";
				Logger.Debug(text);
				if (!suppressPopups)
				{
					showErrorCallback(text, this);
				}
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		switch (currentStatusCode)
		{
		case HttpStatusCode.Unauthorized:
		{
			string text5 = "Secure Service Access is disabled for device '" + Identification + "'";
			Logger.Debug("Login failed (Unauthorized): {Message}", text5);
			if (!suppressPopups)
			{
				showErrorCallback(text5, this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		case HttpStatusCode.Forbidden:
		{
			string text4 = "The password for device '" + Identification + "' is incorrect! Please provide a valid password.";
			Logger.Debug("Login failed (Forbidden), {Message}", text4);
			if (!suppressPopups)
			{
				showErrorCallback(text4, this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		case HttpStatusCode.TooManyRequests:
		{
			double? num = (double?)JsonConvert.DeserializeAnonymousType(currentResponseData, new
			{
				lockout_remaining_seconds = 0
			})?.lockout_remaining_seconds / 60.0;
			string arg = ((num == 1.0) ? "minute" : "minutes");
			string text2 = ((num > 0.0) ? $"for {num:F1} {arg} " : string.Empty);
			string text3 = "Locked out of device '" + Identification + "' " + text2 + "due to multiple incorrect password attempts.";
			Logger.Debug("Login failed (LoginLockout), {Message}", text3);
			if (!suppressPopups)
			{
				showErrorCallback(text3, this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		case HttpStatusCode.RequestTimeout:
		{
			string message = $"Login request timeout for {Identification} and IP {IPAddress}";
			(bool, string) tuple = await CheckNetworkAndPing(IPAddress);
			if (!tuple.Item1)
			{
				message = tuple.Item2;
			}
			if (!suppressPopups)
			{
				showErrorCallback(message, this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		case HttpStatusCode.ServiceUnavailable:
		{
			string arg2 = $"Device {Identification}/{IPAddress} is not reachable";
			if (!suppressPopups)
			{
				showErrorCallback(arg2, this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
		default:
			if (!suppressPopups)
			{
				showErrorCallback("Unknown error", this);
			}
			return EWebRequestState.UNSUCCESSFUL;
		}
	}

	private static async Task<(bool IsSuccess, string Message)> CheckNetworkAndPing(IPAddress chargerIpAddress)
	{
		_ = 1;
		try
		{
			if (!NetworkInterface.GetIsNetworkAvailable())
			{
				return (IsSuccess: false, Message: "No active network available, please make sure you are connected.");
			}
			using Ping pinger = new Ping();
			if ((await pinger.SendPingAsync(chargerIpAddress, 500)).Status == IPStatus.Success)
			{
				return (IsSuccess: true, Message: string.Empty);
			}
			var source = (from n in NetworkInterface.GetAllNetworkInterfaces()
				where n.NetworkInterfaceType == NetworkInterfaceType.Ethernet
				select n).SelectMany((NetworkInterface n) => from ip in n.GetIPProperties().UnicastAddresses
				where ip.Address.AddressFamily == AddressFamily.InterNetwork
				select new
				{
					NetworkName = n.Name,
					Address = ip.Address,
					SubnetMask = ip.IPv4Mask
				}).Take(10).ToList();
			string networkDetails = string.Join("\n", source.Select(n => FormatIpAddress(n.Address) + "\t" + FormatIpAddress(n.SubnetMask) + "\t" + n.NetworkName));
			string text = $"Charger IP address: {chargerIpAddress}\n\n";
			string text2 = await GetLocalIpAddress();
			string item = "Failed to communicate with the charger.\n\n" + text + "Your local IP address: " + text2 + "\n\nIs the charger on the right subnet?\n\nAvailable networks:\n" + networkDetails;
			return (IsSuccess: false, Message: item);
		}
		catch
		{
			return (IsSuccess: false, Message: $"Charger with IP address {chargerIpAddress} is not reachable.");
		}
	}

	private static string FormatIpAddress(IPAddress ipAddress)
	{
		return string.Join(".", from b in ipAddress.GetAddressBytes()
			select b.ToString("D3"));
	}

	private static async Task<string> GetLocalIpAddress()
	{
		using Socket socket = new Socket(AddressFamily.InterNetwork, SocketType.Dgram, ProtocolType.IP);
		await SocketTaskExtensions.ConnectAsync(socket, "8.8.8.8", 65530);
		return (socket.LocalEndPoint as IPEndPoint)?.Address.ToString();
	}

	private void DumpActiveConnections(IPAddress ipAddress)
	{
		try
		{
			Process.GetCurrentProcess();
			IEnumerable<TcpConnectionInformation> enumerable = from c in IPGlobalProperties.GetIPGlobalProperties().GetActiveTcpConnections()
				where c.RemoteEndPoint.Address.ToString() == ipAddress.ToString() && c.RemoteEndPoint.Port == 443 && c.State == TcpState.Established
				select c;
			if (!enumerable.Any())
			{
				return;
			}
			StringBuilder stringBuilder = new StringBuilder();
			stringBuilder.AppendLine("Unexpected active TCP connection found that could cause deadlocks");
			stringBuilder.AppendFormat("{0,-20} {1,-20} {2,-10}", "Local Endpoint", "Remote Endpoint", "State");
			foreach (TcpConnectionInformation item in enumerable)
			{
				stringBuilder.AppendFormat("{0,-20} {1,-20} {2,-10}", item.LocalEndPoint, item.RemoteEndPoint, item.State);
			}
			Logger.Warning(stringBuilder.ToString());
		}
		catch
		{
		}
	}

	public void SetHostInfo(string hostName, int numberOfSockets)
	{
		HostName = hostName;
		NumberOfSockets = numberOfSockets;
	}

	public override bool UpdateProperties(params ICUProperty[] properties)
	{
		if (properties.Length != 0)
		{
			string[] value = (from a in properties
				where a != null
				select a.ODIndex).ToArray();
			return UpdateProperties(string.Join(",", value));
		}
		return false;
	}

	public override bool UpdateCategories(params string[] categories)
	{
		if (IPAddress == null)
		{
			return false;
		}
		bool flag = isAHP && FirmwareVersionNumber >= new Version(2, 2);
		if ((categories.Length == 0) & flag)
		{
			return UpdatePropertiesInternal("limit=500");
		}
		foreach (string item in (categories.Length != 0) ? categories.ToList() : RequestCategories())
		{
			if (!UpdatePropertiesInternal("cat=" + item + (flag ? "&limit=500" : string.Empty)))
			{
				return false;
			}
		}
		return true;
	}

	public override List<string> RequestCategories(Action dispatchUiEvents = null)
	{
		//IL_0039: Unknown result type (might be due to invalid IL or missing references)
		//IL_003f: Expected Obj, but got Unknown
		List<string> list = new List<string>();
		try
		{
			(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("categories");
			if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
			{
				return list;
			}
			JavaScriptSerializer val = new JavaScriptSerializer();
			if (string.IsNullOrEmpty(tuple.Item3))
			{
				return list;
			}
			dynamic val2 = val.Deserialize<object>(tuple.Item3);
			foreach (dynamic item in val2)
			{
				if (!((item.Key == "categories") ? true : false))
				{
					continue;
				}
				dynamic val3 = item.Value;
				foreach (object item2 in val3)
				{
					if (item2 != null)
					{
						string text = Convert.ToString(item2);
						list.Add(text.Trim(new char[2] { '"', ' ' }));
					}
				}
			}
			return list;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return list;
		}
	}

	public override bool OnDeviceRemoved()
	{
		return !IsRebooting;
	}

	public bool ChangePassword(string newpassword)
	{
		if (!IsUniquePasswordRequired)
		{
			return false;
		}
		if (IsLoggedIn)
		{
			string content = "{\"password\":\"" + newpassword + "\"}";
			if (ExecuteWebRequest("password", "", content, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE)
			{
				Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Password changed");
				return true;
			}
		}
		return false;
	}

	public bool SetEndUserPin(string endUserPin)
	{
		if (IsLoggedIn)
		{
			string content = "{\"username\":\"end user\",\"password\":\"" + endUserPin + "\"}";
			if (ExecuteWebRequest("password", "", content, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE)
			{
				EndUserAccessType = ((endUserPin == string.Empty) ? EndUserAccessType.Enabled : EndUserAccessType.Configured);
				Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("End user access enabled {Has} PIN", (endUserPin == string.Empty) ? "without" : "with");
				return true;
			}
		}
		return false;
	}

	public bool DisableEndUserAccess()
	{
		if (IsLoggedIn)
		{
			string content = "{\"username\":\"end user\",\"reset\":true}";
			if (ExecuteWebRequest("password", "", content, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE)
			{
				Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("End user access disabled");
				EndUserAccessType = EndUserAccessType.Disabled;
				return true;
			}
		}
		return false;
	}

	public (bool success, string message) ResetPassword(string passwordresetcode)
	{
		if (!IsUniquePasswordRequired)
		{
			return (success: false, message: "Not supported");
		}
		LoginData.Password = string.Empty;
		string content = "{\"code\":\"" + passwordresetcode + "\"}";
		(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("prc", "", content, 5000, 1, suppressPopups: true);
		if (tuple.Item1 == EWebRequestState.VALID_RESPONSE)
		{
			Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Password reset");
			return (success: true, message: "The password of this Charging Station (" + Identification + ") has been successfully reset to the default password.");
		}
		if (tuple.Item2 == HttpStatusCode.Forbidden)
		{
			return (success: false, message: "The password reset code is incorrect.");
		}
		if (tuple.Item2 == HttpStatusCode.TooManyRequests)
		{
			string value = tuple.Item3;
			if (tuple.Item3.Contains("version"))
			{
				value = Regex.Replace(Regex.Replace(tuple.Item3, "{\"version\":1,", string.Empty), "}}", "}");
			}
			double valueOrDefault = ((double?)JsonConvert.DeserializeAnonymousType(value, new
			{
				lockout_remaining_seconds = 0
			})?.lockout_remaining_seconds / 60.0).GetValueOrDefault();
			string arg = (($"{valueOrDefault:F1}" == "1.0") ? "minute" : "minutes");
			string text = ((valueOrDefault > 0.0) ? $"for {valueOrDefault:F1} {arg} " : string.Empty);
			string item = "Locked out of device '" + Identification + "' " + text + " due to multiple incorrect reset attempts.";
			return (success: false, message: item);
		}
		if (tuple.Item2 == HttpStatusCode.ServiceUnavailable)
		{
			return (success: false, message: "Password recovery not available for this charging station");
		}
		return (success: false, message: "Unknown error.");
	}

	public bool CreateTempPassword(string newpassword, uint expirationTime)
	{
		if (!IsUniquePasswordRequired)
		{
			return false;
		}
		if (IsLoggedIn)
		{
			string content = $"{{\"password\":\"{newpassword}\",\"expiration\":{expirationTime}}}";
			if (ExecuteWebRequest("temporarypassword", "", content, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE)
			{
				Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Temporary password set");
				return true;
			}
		}
		return false;
	}

	public bool IsNg9xxFirmwareEqualOrHigherThan(Version version)
	{
		if (FirmwareVersionNumber != null && !isAHP)
		{
			return FirmwareVersionNumber >= version;
		}
		return false;
	}

	public bool IsAhpFirmwareEqualOrHigherThan(Version version)
	{
		if (FirmwareVersionNumber != null && isAHP)
		{
			return FirmwareVersionNumber >= version;
		}
		return false;
	}

	public void RegisterLoginCallback(Action<ICULanDevice, ACEWebLoginData> callback)
	{
		m_cbRequestLoginCredentials = callback;
	}

	public (bool IsLoggedIn, HttpStatusCode HttpStatusCode, string LoginError) Login(bool suppressLoginDialog = false)
	{
		Logger.Debug("Login UserName={UserName} Password={Password} IsLoggedIn={IsLoggedIn} SupressLoginDialog={SuppressLoginDialog} ", LoginData.Username, LoginData.Password, LoginData.IsLoggedIn, suppressLoginDialog);
		Logger.Debug("Login shows a login dialog if its not already shown");
		IsRequestingLoginData = true;
		m_cbRequestLoginCredentials?.Invoke(this, LoginData);
		IsRequestingLoginData = false;
		Logger.Debug("Login IsLoggedIn={IsLoggedIn}", IsLoggedIn);
		return (IsLoggedIn: LoginData.IsLoggedIn, HttpStatusCode: LoginData.LastHttpStatusCode, LoginError: LoginData.LoginError);
	}

	public void LogInformation(string messageTemplate, params object[] list)
	{
		using (LogContext.PushProperty("SerialNumber", $"{SerialNumber?.GetHashCode():X}"))
		{
			using (LogContext.PushProperty("SCNNetwork", $"{SCNNetwork?.GetHashCode():X}"))
			{
			}
		}
	}

	private async Task<(bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content)> ClassicLogin(int loginTimeoutInMs = 5000, CancellationToken cancellationToken = default(CancellationToken))
	{
		HttpStatusCode lastStatusCode = HttpStatusCode.ServiceUnavailable;
		for (int retry = 0; retry < 3; retry++)
		{
			if (retry == 2)
			{
				LoginData.Username = ICUNetworkConfig.HTTPUsernameOld;
				LoginData.Password = ICUNetworkConfig.HTTPPasswordOld;
			}
			using CancellationTokenSource ctsTimeout = new CancellationTokenSource(loginTimeoutInMs);
			using CancellationTokenSource linkedCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken, ctsTimeout.Token);
			HttpResponseMessage httpResponseMessage = await SendHttpRequest(GetUri("login"), "{\"username\":\"" + LoginData.Username + "\",\"password\":\"" + LoginData.Password + "\",\"displayname\":\"" + LoginData.DisplayName + "\"}", fireAndForget: false, linkedCts.Token).ConfigureAwait(continueOnCapturedContext: false);
			lastStatusCode = httpResponseMessage.StatusCode;
			if (httpResponseMessage.IsSuccessStatusCode)
			{
				return (IsLoggedIn: httpResponseMessage.IsSuccessStatusCode, HttpStatusCode: httpResponseMessage.StatusCode, Content: httpResponseMessage.StatusCode.ToString());
			}
			await Task.Delay(1000);
		}
		return (IsLoggedIn: false, HttpStatusCode: lastStatusCode, Content: string.Empty);
	}

	public async Task<(bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content)> LoginRequest(int loginTimeoutInMs = 15000, CancellationToken cancellationToken = default(CancellationToken))
	{
		_ = 3;
		try
		{
			if (IsHTTPS)
			{
				using (CancellationTokenSource ctsTimeout = new CancellationTokenSource(loginTimeoutInMs))
				{
					using CancellationTokenSource linkedCts = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken, ctsTimeout.Token);
					if (await IsAccessTokenAvailableAsync(cancellationToken))
					{
						IsLoggedIn = true;
						return (IsLoggedIn: true, HttpStatusCode: LoginData.LastHttpStatusCode, Content: string.Empty);
					}
					HttpResponseMessage loginResponse = await SendHttpRequest(GetUri("login"), "{\"username\":\"" + LoginData.Username + "\",\"password\":\"" + LoginData.Password.Trim(new char[1] { '"' }) + "\",\"displayname\":\"" + LoginData.DisplayName + "\"}", fireAndForget: false, linkedCts.Token).ConfigureAwait(continueOnCapturedContext: false);
					string text = ((loginResponse.Content == null) ? string.Empty : (await loginResponse.Content.ReadAsStringAsync().ConfigureAwait(continueOnCapturedContext: false)));
					string text2 = text;
					if (!loginResponse.IsSuccessStatusCode)
					{
						Logger.Warning("Login request failed with status code {StatusCode} and content: {Content}", loginResponse.StatusCode, text2);
						return (IsLoggedIn: false, HttpStatusCode: loginResponse.StatusCode, Content: text2);
					}
					UpdateTokenInfo(text2);
					IsLoggedIn = true;
					return (IsLoggedIn: true, HttpStatusCode: loginResponse.StatusCode, Content: text2);
				}
			}
			return await ClassicLogin(5000, cancellationToken).ConfigureAwait(continueOnCapturedContext: false);
		}
		catch (TaskCanceledException)
		{
			return (IsLoggedIn: false, HttpStatusCode: HttpStatusCode.RequestTimeout, Content: HttpStatusCode.RequestTimeout.ToString());
		}
		catch (HttpRequestException ex2)
		{
			if (ex2.InnerException is WebException ex3 && ex3.InnerException is SocketException)
			{
				Logger.Verbose("SocketException occurred during login request. Happens after a reboot caused by e.g. a firmware update");
				return (IsLoggedIn: false, HttpStatusCode: HttpStatusCode.RequestTimeout, Content: string.Empty);
			}
			return (IsLoggedIn: false, HttpStatusCode: HttpStatusCode.ServiceUnavailable, Content: ex2.Message);
		}
		catch (Exception ex4)
		{
			Logger.Error(ex4, "Unexpected error occurred during login request");
			return (IsLoggedIn: false, HttpStatusCode: HttpStatusCode.InternalServerError, Content: ex4.Message);
		}
	}

	private void UpdateTokenInfo(string httpBody)
	{
		ClearTokens();
		if (!string.IsNullOrEmpty(httpBody))
		{
			try
			{
				LogingResponse logingResponse = JsonConvert.DeserializeObject<LogingResponse>(httpBody);
				AccessToken = logingResponse.AccessToken;
				RefreshToken = logingResponse.RefreshToken;
			}
			catch (Exception exception)
			{
				Logger.Error(exception, "Failed to deserialize login response");
			}
		}
	}

	private async Task<bool> IsAccessTokenAvailableAsync(CancellationToken cancellationToken)
	{
		if (string.IsNullOrEmpty(AccessToken) && !string.IsNullOrEmpty(RefreshToken))
		{
			HttpResponseMessage httpResponseMessage = await SendHttpRequest(GetUri("token/refresh"), "{\"refresh\":\"" + RefreshToken + "\"}", fireAndForget: false, cancellationToken).ConfigureAwait(continueOnCapturedContext: false);
			if (httpResponseMessage.IsSuccessStatusCode && httpResponseMessage.Content != null)
			{
				LoginData.LastHttpStatusCode = httpResponseMessage.StatusCode;
				string text = ((httpResponseMessage.Content == null) ? string.Empty : (await httpResponseMessage.Content.ReadAsStringAsync().ConfigureAwait(continueOnCapturedContext: false)));
				string httpBody = text;
				UpdateTokenInfo(httpBody);
			}
		}
		return !string.IsNullOrEmpty(AccessToken);
	}

	public bool Logout(bool forceClose = false)
	{
		try
		{
			IsDefaultCategoryCollected = false;
			bool flag = false;
			Logger.Debug("Logout from charger ForceClose={ForceClose}", forceClose);
			if (IsHTTPS && m_httpClient != null && LastCommand?.ToLowerInvariant() != "logout" && !IsUploading)
			{
				flag = ExecuteWebRequest("logout", "", string.Empty, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE;
			}
			if (flag | forceClose)
			{
				ClearTokens();
				DisposeHttpClient();
			}
			IsLoggedIn = false;
			return flag;
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "Could not logout");
			return true;
		}
	}

	public override bool UpdateProperties(string sIds, bool notifyChanges = true)
	{
		if (IPAddress == null)
		{
			return false;
		}
		return UpdatePropertiesInternal("ids=" + sIds, notifyChanges);
	}

	private void ParseProperty(string propName, dynamic item, bool notifyChanges)
	{
		object obj = item["value"];
		if (obj == null)
		{
			obj = "0";
		}
		try
		{
			SDT dataType = (SDT)Convert.ToInt32(item["type"]);
			string text = Convert.ToString(item["id"]);
			int num = Convert.ToInt32(item["access"]);
			string category = "";
			if (item.ContainsKey("cat"))
			{
				category = Convert.ToString(item["cat"]);
			}
			ulong maxLength = 0uL;
			if (item.ContainsKey("len"))
			{
				maxLength = Convert.ToUInt64(item["len"]);
			}
			string[] array = text.Split(new char[1] { '_' });
			if (array.Length != 2)
			{
				return;
			}
			ushort propId = (ushort)int.Parse(array[0], NumberStyles.HexNumber);
			byte subId = (byte)int.Parse(array[1], NumberStyles.HexNumber);
			ICUProperty iCUProperty = PropertyDictionary.GetProperty(propId, subId) ?? PropertyDictionary.AddProperty(dataType, propId, subId, propName, propName, num == 1);
			if (iCUProperty != null && (iCUProperty.Id != 8273 || AllowObjectIDUpdate || iCUProperty.Value == null))
			{
				iCUProperty.DataType = dataType;
				if (string.IsNullOrEmpty(iCUProperty.ICUName))
				{
					iCUProperty.ICUName = propName;
				}
				iCUProperty.ReadOnly = num == 1;
				if (string.IsNullOrEmpty(iCUProperty.Title))
				{
					iCUProperty.Title = propName;
				}
				iCUProperty.Category = category;
				iCUProperty.MaxLength = maxLength;
				string objInitialValue = obj.ToString();
				if (!iCUProperty.IsChanged)
				{
					iCUProperty.SetInitialValue(objInitialValue, notifyChanges);
				}
			}
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "{PropName}", propName);
		}
	}

	private bool UpdatePropertiesInternal(string parameters, bool notifyChanges = true)
	{
		//IL_0035: Unknown result type (might be due to invalid IL or missing references)
		try
		{
			bool flag = false;
			int num = 0;
			int num2 = 0;
			(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("prop", parameters, null, 5000, 2);
			if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
			{
				return false;
			}
			do
			{
				dynamic val = new JavaScriptSerializer().Deserialize<object>(tuple.Item3);
				if (val == null)
				{
					return false;
				}
				int num3 = 0;
				foreach (dynamic item in val)
				{
					switch ((string)(object)item.Key)
					{
					case "version":
						_ = (int)item.Value;
						break;
					case "count":
						num3 = (int)item.Value;
						break;
					case "total":
						num = (int)item.Value;
						break;
					case "offset":
						num2 = (int)item.Value;
						break;
					case "properties":
					{
						dynamic val2 = item.Value;
						foreach (dynamic item2 in val2)
						{
							ParseProperty(item2["id"].ToString(), item2, notifyChanges);
							num3++;
						}
						break;
					}
					default:
						if (item.Value.GetType() != typeof(bool))
						{
							ParseProperty(item.Key, item.Value, notifyChanges);
						}
						break;
					}
				}
				if (num <= num2 + num3)
				{
					break;
				}
				string parameters2 = $"{parameters}&offset={num2 + num3}";
				tuple = ExecuteWebRequest("prop", parameters2);
				if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
				{
					return false;
				}
			}
			while (true);
			if (string.IsNullOrEmpty(Identification))
			{
				Identity = GetPropertyString(8275, 0, 0);
			}
			if (ModelType == ICUDeviceModel.Unknown)
			{
				string propertyString = GetPropertyString(8272, 0, 0);
				string propertyString2 = GetPropertyString(8273, 0, 0);
				SocketTypes[0] = GetPropertyInt(8485, 0);
				if (NumberOfSockets > 1)
				{
					SocketTypes[1] = GetPropertyInt(12581, 0);
				}
				propertyString = propertyString.Replace(' ', '-').Replace('.', '-');
				HostName = (propertyString + "-" + propertyString2).ToLowerInvariant();
				Logger.Debug("Model: {ModelType}", ModelType);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return false;
		}
	}

	public override bool StoreProperties(params ICUProperty[] propertiesToStore)
	{
		try
		{
			List<ICUProperty> list = new List<ICUProperty>(propertiesToStore);
			while (list.Count > 0)
			{
				List<ICUProperty> list2 = list.Take(15).ToList();
				list.RemoveRange(0, list2.Count);
				StringBuilder stringBuilder = new StringBuilder();
				stringBuilder.Append("{");
				for (int i = 0; i < list2.Count; i++)
				{
					ICUProperty iCUProperty = list2[i];
					if (iCUProperty.DataType == SDT.BYTEARRAY)
					{
						byte[] array = (byte[])iCUProperty.Value;
						if (array != null)
						{
							string text = string.Join(",", array.Select((byte a) => a.ToString("X2")));
							stringBuilder.AppendFormat("\"{0}\":{{\"id\":\"{1:X}_{2:X}\",\"value\":\"{3}\"}}", new object[4] { iCUProperty.ICUName, iCUProperty.Id, iCUProperty.SubId, text });
						}
					}
					else if (iCUProperty.DataType == SDT.ARRAY_16)
					{
						ushort[] array2 = (ushort[])iCUProperty.Value;
						if (array2 != null)
						{
							string text2 = string.Join(",", array2.Select((ushort a) => a.ToString("X4")));
							stringBuilder.AppendFormat("\"{0}\":{{\"id\":\"{1:X}_{2:X}\",\"value\":\"{3}\"}}", new object[4] { iCUProperty.ICUName, iCUProperty.Id, iCUProperty.SubId, text2 });
						}
					}
					else if (iCUProperty.DataType == SDT.UNICODE_STRING || iCUProperty.DataType == SDT.VISIBLE_STRING || iCUProperty.DataType == SDT.DOAMIN || iCUProperty.DataType == SDT.BOOLEAN)
					{
						stringBuilder.AppendFormat("\"{0}\":{{\"id\":\"{1:X}_{2:X}\",\"value\":\"{3}\"}}", new object[4]
						{
							iCUProperty.ICUName,
							iCUProperty.Id,
							iCUProperty.SubId,
							iCUProperty.Value.ToString()
						});
					}
					else if (iCUProperty.DataType == SDT.REAL32 || iCUProperty.DataType == SDT.REAL64)
					{
						string text3 = Convert.ToString(iCUProperty.Value, new CultureInfo("en-US"));
						stringBuilder.AppendFormat("\"{0}\":{{\"id\":\"{1:X}_{2:X}\",\"value\":{3}}}", new object[4] { iCUProperty.ICUName, iCUProperty.Id, iCUProperty.SubId, text3 });
					}
					else
					{
						stringBuilder.AppendFormat("\"{0}\":{{\"id\":\"{1:X}_{2:X}\",\"value\":{3}}}", new object[4]
						{
							iCUProperty.ICUName,
							iCUProperty.Id,
							iCUProperty.SubId,
							iCUProperty.Value.ToString()
						});
					}
					if (i < list2.Count - 1)
					{
						stringBuilder.Append(",");
					}
				}
				stringBuilder.Append("}");
				if (ExecuteWebRequest("prop", "", stringBuilder.ToString()).RequestState == EWebRequestState.VALID_RESPONSE)
				{
					Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Stored properties: {PropertyList}", string.Join(",", propertiesToStore.Select((ICUProperty p) => p.ID_SUB).ToList()));
					list2.ForEach((ICUProperty a) =>
					{
						a.CommitChange();
					});
					continue;
				}
				return false;
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool storeProperty(ushort propId, byte subId, object newValue)
	{
		ICUProperty property = GetProperty(propId, subId);
		if (property != null && newValue != null)
		{
			property.Value = newValue;
			if (property.IsChanged)
			{
				StoreProperties(property);
				return true;
			}
		}
		return false;
	}

	public List<ICULanLogLine> GetLogLines(long offset)
	{
		List<ICULanLogLine> list = new List<ICULanLogLine>();
		bool flag = isAHP && FirmwareVersionNumber >= new Version(2, 4, 0);
		string text = $"offset={offset}";
		if ((offset != 0) & flag)
		{
			text += $"&lines={s_nMaxLogLines}";
		}
		(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("log", text, null, 10000, 2);
		if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
		{
			list.Add(new ICULanLogLine("Communication Error")
			{
				ID = long.MinValue
			});
			return list;
		}
		string[] array = tuple.Item3.Split(new char[1] { '\n' });
		foreach (string text2 in array)
		{
			if (text2.Trim().Length > 0)
			{
				ICULanLogLine iCULanLogLine = new ICULanLogLine(text2);
				if (iCULanLogLine.Type != ICULanLogType.UNKNOWN)
				{
					list.Add(iCULanLogLine);
				}
				else
				{
					Logger.Warning("Unknown log line: {LogLine}", text2);
				}
			}
		}
		return list;
	}

	private string StartUpload(BackgroundWorker bgw, byte[] fileData, bool isFirmwareFile = true, string newPassword = "")
	{
		ProgressHelper progressHelper = new ProgressHelper(isAHP);
		bool flag = false;
		double progress = progressHelper.GetProgress(0.9);
		UpdateProperties(3538945u);
		if (GetProperty(13824, 1) == null || GetPropertyInt(13824, 1) != 3)
		{
			SetDate(DateTime.UtcNow);
		}
		bool firmwareUploadStatus = GetFirmwareUploadStatus(out var inProgress, out var _);
		if ((IsUploading || !firmwareUploadStatus) | inProgress)
		{
			Logger.Debug("Firmware upload: there is already an upload in progress");
			return "There is still a firmware upload in progress";
		}
		CancellationTokenSource firmwareUploadCts = new CancellationTokenSource();
		IsUploading = true;
		try
		{
			LastUploadError = string.Empty;
			Logger.Debug("Firmware upload: uploading firmware to device");
			CancellationTokenSource fakeProgressCts = new CancellationTokenSource();
			if (bgw != null)
			{
				Task.Run(async () =>
				{
					int fakeCounter = 0;
					while (!fakeProgressCts.IsCancellationRequested)
					{
						await Task.Delay(1000);
						progress = progressHelper.GetProgress(progress);
						bgw.ReportProgress((int)progress);
						if (fakeCounter++ % 5 == 0)
						{
							Logger.Debug("Firmware upload: {Progress}%", progress);
						}
					}
				}, fakeProgressCts.Token);
			}
			try
			{
				int num = 3;
				while (num-- > 0)
				{
					(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("firmware", string.Empty, fileData, 900000, 1, suppressPopups: true, firmwareUploadCts.Token);
					if (tuple.Item2 == HttpStatusCode.Unauthorized || tuple.Item2 == HttpStatusCode.Forbidden)
					{
						if (Login().HttpStatusCode == HttpStatusCode.Unauthorized)
						{
							Thread.Sleep(1000);
							continue;
						}
					}
					else if (tuple.Item2 != HttpStatusCode.OK)
					{
						return "Couldn't communicate with the device, please reboot the device.";
					}
					m_lastSuccessfulFileUpload = DateTime.UtcNow;
					break;
				}
			}
			finally
			{
				fakeProgressCts.Cancel();
				fakeProgressCts.Dispose();
			}
			Logger.Debug("Firmware upload: File uploaded");
			if (bgw == null)
			{
				Logger.Debug("Firmware upload: No progress to be followed and/or reported");
				return string.Empty;
			}
			if (!isFirmwareFile)
			{
				Logger.Debug("Firmware upload: Not a firmware update, so no reboot to track");
				return string.Empty;
			}
			progress = progressHelper.GetNextStage();
			bool flag2 = !string.IsNullOrEmpty(newPassword);
			if (flag2)
			{
				m_fUniquePasswordGracePeriod = true;
			}
			bool flag3 = false;
			string text = "";
			Stopwatch stopwatch = null;
			bool flag4 = false;
			int num2 = 3;
			Version firmwareVersionNumber = FirmwareVersionNumber;
			int num3 = 0;
			while (!flag3 && !firmwareUploadCts.IsCancellationRequested && num2 > 0)
			{
				progress = progressHelper.GetProgress(progress);
				bgw.ReportProgress((int)progress);
				bool flag5 = num3++ % 10 == 0;
				if (flag5)
				{
					Logger.Debug("Firmware upload: {Progress}% {DeviceName} {DeviceIP} IsRebooting={IsRebooting} IsUploading={IsUploading} IsLoggedIn={IsLoggedIn} IsResetted={IsResetted} RetryLoginCounter={RetryLoginCounter} OldFirmwareVersion={OldFirmwareVersion}", (int)progress, Name, IPAddress, IsRebooting, IsUploading, IsLoggedIn, flag4, num2, firmwareVersionNumber);
				}
				Thread.Sleep(1000);
				if (CheckNetworkAndPing(IPAddress).GetAwaiter().GetResult().IsSuccess && !IsLoggedIn)
				{
					if (flag5)
					{
						Logger.Debug("Firmware upload: Trying to login to {DeviceName} {IpAddress}, check if the webservice is running again", Name, IPAddress);
					}
					ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
					Task<(bool, HttpStatusCode, string)> task = Task.Run(async () =>
					{
						_ = 1;
						try
						{
							(bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content) loginResponse = await LoginRequest(LoginTimeoutInSeconds * 1000, firmwareUploadCts.Token);
							if (loginResponse.HttpStatusCode == HttpStatusCode.RequestTimeout)
							{
								await Task.Delay(1000, firmwareUploadCts.Token);
							}
							else
							{
								_ = loginResponse.HttpStatusCode;
								_ = 403;
							}
							return loginResponse;
						}
						finally
						{
							finishedEvent.Set();
						}
					}, firmwareUploadCts.Token);
					while (!finishedEvent.WaitOne(1000))
					{
						Logger.Debug("Firmware upload: wait for finishedEvent");
						progress = progressHelper.GetProgress(progress);
						bgw.ReportProgress((int)progress);
					}
					Logger.Debug("Firmware upload: after finishedEvent");
					IsLoggedIn = task.GetAwaiter().GetResult().Item2 == HttpStatusCode.OK;
				}
				if (flag4 && !IsLoggedIn)
				{
					Logger.Debug("Firmware upload: login failed {DeviceName}/{IpAddress}/{Firmware}", Name, IPAddress, FirmwareVersionNumber);
					if (num2 > 0)
					{
						num2--;
					}
					else if (flag2)
					{
						m_fUniquePasswordGracePeriod = false;
						flag2 = false;
						text = "Failed to change the password. There is probably already a password present. Please use that to login to the device";
					}
					else
					{
						firmwareUploadCts.Cancel();
					}
					continue;
				}
				if (IsLoggedIn && GetFirmwareUploadStatus(out var inProgress2, out var updateStatus2))
				{
					LastValidResponse = DateTime.UtcNow;
					Logger.Debug("Firmware upload: Response received {DateTime} {Status}", LastValidResponse, updateStatus2);
					switch (updateStatus2)
					{
					case EFirmwareUpdateStatus.NO_ACTIVE_UPDATE:
						Logger.Debug("Firmware upload: no active update");
						flag3 = true;
						break;
					case EFirmwareUpdateStatus.UPDATE_IN_PROGRESS:
						Logger.Debug("Firmware upload: downloading firmware");
						if (isAHP && !flag)
						{
							progress = progressHelper.GetNextStage();
							flag = true;
						}
						flag3 = false;
						break;
					case EFirmwareUpdateStatus.READY_FOR_UPDATE:
						Logger.Debug("Firmware upload: ready for update");
						if (!inProgress2 && fileData.Length < 8192)
						{
							Logger.Debug("Firmware upload: special case, probably a updated settings file, goes directly to update state but is already finished");
							flag3 = true;
						}
						else
						{
							flag3 = false;
						}
						break;
					case EFirmwareUpdateStatus.UPDATE_DONE:
						Logger.Debug("Firmware upload: update done");
						flag3 = true;
						break;
					case EFirmwareUpdateStatus.ROLLED_BACK:
					case EFirmwareUpdateStatus.EXECUTING_ROLLBACK:
					case EFirmwareUpdateStatus.ERROR_DURING_UPDATE:
					case EFirmwareUpdateStatus.ERROR_DURING_DOWNLOAD:
						text = updateStatus2.GetEnumDescription();
						flag3 = true;
						break;
					}
					Thread.Sleep(1000);
					continue;
				}
				if (stopwatch == null)
				{
					stopwatch = Stopwatch.StartNew();
				}
				if (stopwatch.ElapsedMilliseconds > 900000)
				{
					text = "Error during upload, it took too long!";
					flag3 = true;
				}
				if (!IsRebooting && !flag4)
				{
					Logger.Debug("Firmware upload: Rebooting charger");
					IsRebooting = true;
					IsLoggedIn = false;
					Connection?.StartBrowsing();
					if (isAHP)
					{
						progress = progressHelper.GetNextStage();
					}
				}
				else if (m_vFirmwareVersion != firmwareVersionNumber)
				{
					Logger.Debug("Firmware upload: Continue because firmware version was changed");
					IsRebooting = false;
					flag4 = true;
				}
				else if (stopwatch.ElapsedMilliseconds > 170000)
				{
					Logger.Debug("Firmware upload: Reset timeout was reached");
					IsRebooting = false;
					flag4 = true;
				}
				else
				{
					Connection?.StartBrowsing();
				}
			}
			bgw.ReportProgress(98);
			if (string.IsNullOrEmpty(text))
			{
				if (isFirmwareFile)
				{
					Logger.Debug("Firmware upload: force firmware permanent");
					SendCommand("forcefirmwarepermanent");
				}
				if (flag2)
				{
					Logger.Debug("Firmware upload: per-charger-password");
					if (ChangePassword(newPassword))
					{
						Logger.Debug("Firmware upload: succeeded in setting a new password");
						m_fUniquePasswordGracePeriod = false;
					}
					else
					{
						text = "Failed to change the password. There is probably already a password present. Please use that to login to the device";
					}
				}
			}
			m_fUniquePasswordGracePeriod = false;
			return text;
		}
		catch (Exception)
		{
			IsRebooting = false;
			throw;
		}
		finally
		{
			IsUploading = false;
			firmwareUploadCts.Cancel();
			firmwareUploadCts.Dispose();
		}
	}

	public override bool UploadFirmware(BackgroundWorker bgw, string fileName, string newPassword = "")
	{
		if (string.IsNullOrEmpty(fileName))
		{
			return false;
		}
		Logger.Debug("Firmware upload: disable mDNS");
		Connection.StopBrowsing();
		Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Start file upload: {FileName}", fileName.Substring(fileName.LastIndexOf(Path.DirectorySeparatorChar) + 1));
		try
		{
			Logger.Debug("Firmware upload: read the entire file data into memory");
			byte[] fileData = File.ReadAllBytes(fileName);
			Logger.Debug("Firmware upload: starting upload");
			string text = StartUpload(bgw, fileData, isFirmwareFile: true, newPassword);
			if (!string.IsNullOrEmpty(text))
			{
				Logger.Error("Firmware upload: {Message}", text);
				LastUploadError = text;
				return false;
			}
			Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("File upload finished: {FileName}", fileName.Substring(fileName.LastIndexOf(Path.DirectorySeparatorChar) + 1));
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			LastUploadError = ex.Message;
			return false;
		}
		finally
		{
			Connection.StartBrowsing();
		}
	}

	public bool UploadResource(BackgroundWorker bgw, byte[] resourceData)
	{
		try
		{
			string text = StartUpload(bgw, resourceData, isFirmwareFile: false);
			if (!string.IsNullOrEmpty(text))
			{
				Logger.Debug("{Error}", text);
				LastUploadError = text;
				return false;
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, ex.Message);
			LastUploadError = ex.Message;
			return false;
		}
	}

	private bool GetFirmwareUploadStatus(out bool inProgress, out EFirmwareUpdateStatus updateStatus, CancellationToken cancellationToken = default(CancellationToken))
	{
		//IL_002a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0030: Expected Obj, but got Unknown
		updateStatus = EFirmwareUpdateStatus.NO_ACTIVE_UPDATE;
		inProgress = false;
		(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("firmware", "", null, 10000, 1, suppressPopups: true, cancellationToken);
		if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
		{
			return false;
		}
		JavaScriptSerializer val = new JavaScriptSerializer();
		if (tuple.Item3.EndsWith(",}"))
		{
			tuple.Item3 = tuple.Item3.Replace(",}", "}");
		}
		if (string.IsNullOrEmpty(tuple.Item3))
		{
			return false;
		}
		dynamic val2 = val.Deserialize<object>(tuple.Item3);
		foreach (dynamic item in val2)
		{
			if (item.Key == "uploadInProgress")
			{
				inProgress = item.Value.ToString() == "true";
			}
			if (item.Key == "OD_fileFirmwareUpdateStatus")
			{
				dynamic val3 = item.Value;
				int num = 0;
				if (!((val3.Count > 0 && int.TryParse(val3["value"].ToString(), out num)) ? true : false))
				{
					return false;
				}
				if (!Enum.IsDefined(typeof(EFirmwareUpdateStatus), num))
				{
					return false;
				}
				updateStatus = (EFirmwareUpdateStatus)num;
			}
		}
		Logger.Debug("Firmware upload: InProgress: {InProgress}, Status: {UpdateStatus}", inProgress, updateStatus);
		return true;
	}

	private int GetRebootDelayInMillis()
	{
		int num = 5000;
		DateTime utcNow = DateTime.UtcNow;
		if (isAHP && m_lastSuccessfulFileUpload > utcNow.AddSeconds(-20.0))
		{
			num = 20000 - (int)(utcNow - m_lastSuccessfulFileUpload).TotalMilliseconds;
		}
		else
		{
			DateTime dateTime = ((LastUpdate > m_lastSuccessfulFileUpload) ? LastUpdate : m_lastSuccessfulFileUpload);
			num -= (int)(utcNow - dateTime).TotalMilliseconds;
		}
		return Math.Max(num, 0);
	}

	public bool Reboot(Action<ICULanDevice, Exception> callback, Action<ICULanDevice, double> progress, bool hardReboot = true)
	{
		(Connection as LANConnection)?.StopBrowsing();
		bool result = true;
		if (hardReboot)
		{
			m_cbRebootCallback = callback;
			m_cbRebootProgress = progress;
			m_dRebootProgress = 0.0;
			m_timReboot = new Timer(OnRebootProgress, this, 250, 250);
			IsRebooting = SendReboot();
			if (IsRebooting)
			{
				IsLoggedIn = false;
			}
			result = true;
		}
		else if (isAHP)
		{
			if (ExecuteWebRequest("reset", "type=soft", string.Empty, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE)
			{
				progress?.Invoke(this, 1.0);
				callback?.Invoke(this, null);
			}
			else
			{
				callback?.Invoke(this, new TimeoutException());
			}
		}
		else
		{
			result = false;
		}
		return result;
	}

	public void OnRebootProgress(object stateInfo)
	{
		ICULanDevice lanDev = (ICULanDevice)stateInfo;
		m_dRebootProgress += 1.0 / 480.0;
		if (!Monitor.TryEnter(m_lockRebootProgress, 250))
		{
			return;
		}
		try
		{
			int num = (int)(m_dRebootProgress * 100.0);
			if (lanDev == null)
			{
				return;
			}
			if (!lanDev.IsManuallyAdded)
			{
				_ = Connection;
				double num2 = (isAHP ? 0.5 : 0.2);
				if ((m_dRebootProgress > num2 && num % 8 == 0) || (m_dRebootProgress > 0.9 && num % 2 == 0))
				{
					Connection.StartBrowsing();
				}
			}
			else if (m_dRebootProgress > 0.5)
			{
				Task.Run(async () =>
				{
					if ((await lanDev.LoginRequest()).Item1)
					{
						FinishedRebooting();
					}
				});
			}
			if (m_dRebootProgress > 0.7)
			{
				m_cookieContainer = new CookieContainer();
			}
			if (m_dRebootProgress > 1.0)
			{
				if (m_cbRebootCallback != null)
				{
					m_cbRebootCallback(this, new TimeoutException());
				}
				m_cbRebootProgress = null;
				m_cbRebootCallback = null;
				m_timReboot.Dispose();
				IsRebooting = false;
			}
			else
			{
				lanDev.m_cbRebootProgress?.Invoke(this, m_dRebootProgress);
			}
		}
		finally
		{
			Monitor.Exit(m_lockRebootProgress);
		}
	}

	public bool SetDate(DateTime dt)
	{
		ICUProperty property = GetProperty(8281, 0);
		if (property != null)
		{
			double num = Math.Round((TimeZoneInfo.ConvertTimeToUtc(dt) - ICUDevice.UnixEpoch).TotalMilliseconds);
			property.Value = num;
			StoreProperties(property);
		}
		return SendDatetime(dt);
	}

	public bool EraseTransactionDatabase(int timeout = 5000)
	{
		if (isAHP)
		{
			return ExecuteWebRequest("db/reset", "dbname=transactions", string.Empty, timeout, 1).RequestState == EWebRequestState.VALID_RESPONSE;
		}
		return SendCommand("txerase");
	}

	public bool SendCommand(string command, int timeout = 5000, bool suppressPopups = false)
	{
		return ExecuteWebRequest("cmd", string.Empty, "{\"command\":\"" + command + "\"}", timeout, 2, suppressPopups).RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public bool SendReboot(int timeout = 5000)
	{
		Logger.Debug("SendReboot");
		int delayInMillis = GetRebootDelayInMillis();
		Logger.Debug("Waiting {Millis} millis before reboot", delayInMillis);
		if (isAHP)
		{
			Task.Run(async () =>
			{
				await Task.Delay(delayInMillis);
				return ExecuteWebRequest("reset", "type=hard", string.Empty, timeout, 1, suppressPopups: true).RequestState == EWebRequestState.VALID_RESPONSE;
			});
			return true;
		}
		Task.Run(async () =>
		{
			await Task.Delay(delayInMillis);
			SendCommand("reboot", 5000, suppressPopups: true);
		});
		Logger.ForContext("ChargerID", $"{SerialNumber?.GetHashCode():X}").Information("Reboot initiated by user");
		return true;
	}

	public bool SendDatetime(DateTime dt, int timeout = 5000)
	{
		if (isAHP)
		{
			return ExecuteWebRequest("datetime", string.Empty, "\"" + dt.ToString("yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture) + "\"", timeout, 1).RequestState == EWebRequestState.VALID_RESPONSE;
		}
		return SendCommand("date " + dt.ToString("yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture));
	}

	public bool SendDiagCommand(string command, int seqId, params object[] param)
	{
		int num = 0;
		string text = $"{{\"command\":\"{command}\",\"sequenceID\":\"{seqId}\",\"parameters\":[";
		bool flag = true;
		foreach (object arg in param)
		{
			if (!flag)
			{
				text += ",";
			}
			text += $"{{\"param{num++}\":\"{arg}\"}}";
			flag = false;
		}
		text += "]}";
		return ExecuteWebRequest("diagtool", "", text).RequestState == EWebRequestState.VALID_RESPONSE;
	}

	public static string GetNFCVersion(ICUDevice device, int reader, bool software = true)
	{
		string result = "N/A";
		if (device is ICULanDevice iCULanDevice)
		{
			if (iCULanDevice.FirmwareVersionNumber < new Version("4.3.0") && reader == 1)
			{
				result = iCULanDevice.GetPropertyString(8276, 0, 0);
				if (result.Contains("#N:"))
				{
					int num = result.LastIndexOf("#N:") + 3;
					string text = "";
					if (num < result.Length)
					{
						text = result.Substring(num);
					}
					string[] array = text.Split(new char[1] { ';' });
					result = ((array.Count() > (software ? 1 : 0)) ? array[software ? 1u : 0u] : "N/A");
				}
				else
				{
					result = "N/A";
				}
			}
			else
			{
				result = ((reader == 1) ? iCULanDevice.GetPropertyString(12672, 0, 0) : iCULanDevice.GetPropertyString(12673, 0, 0));
				if (!string.IsNullOrEmpty(result))
				{
					string[] array2 = result.Split(new char[1] { ',' });
					result = ((array2.Count() > (software ? 1 : 0)) ? array2[software ? 1u : 0u].Split(new char[1] { ':' })[1] : "N/A");
				}
				else
				{
					result = "N/A";
				}
			}
		}
		return result;
	}

	public static string GetHWVersion(ICUDevice device, bool isControllerBoard)
	{
		string result = string.Empty;
		if (device is ICULanDevice iCULanDevice)
		{
			if (isControllerBoard)
			{
				if (iCULanDevice.GetProperty(2116865u) != null)
				{
					result = ((EBoardRevision)Enum.ToObject(typeof(EBoardRevision), iCULanDevice.GetPropertyInt(8269, 1))/*cast due to constrained. prefix*/).ToString().Split(new char[1] { '_' })[1] + "-" + ((EBoardAssy)Enum.ToObject(typeof(EBoardAssy), iCULanDevice.GetPropertyInt(8269, 2))/*cast due to constrained. prefix*/).ToString().Split(new char[1] { '_' })[1] + " ";
				}
				else
				{
					result = ((iCULanDevice.GetProperty(4105, 0) == null) ? "N/A" : iCULanDevice.GetPropertyString(4105, 0, 0));
				}
			}
			else
			{
				result = ((iCULanDevice.GetProperty(2116867u) == null) ? "N/A" : (((EBoardRevision)Enum.ToObject(typeof(EBoardRevision), iCULanDevice.GetPropertyInt(8269, 3))/*cast due to constrained. prefix*/).ToString().Split(new char[1] { '_' })[1] + "-" + ((EBoardAssy)Enum.ToObject(typeof(EBoardAssy), iCULanDevice.GetPropertyInt(8269, 4))/*cast due to constrained. prefix*/).ToString().Split(new char[1] { '_' })[1] + " "));
			}
		}
		return result;
	}

	public bool RequestDiagnosticResult(ref int version, ref string command, ref byte sequenceID, ref bool finished, ref List<string> data)
	{
		//IL_0031: Unknown result type (might be due to invalid IL or missing references)
		try
		{
			(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("diagtool", "result");
			if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
			{
				return false;
			}
			dynamic val = new JavaScriptSerializer().Deserialize<object>(tuple.Item3);
			data = new List<string>();
			foreach (dynamic item in val)
			{
				string text = ((string)item.Key).ToLower().Trim();
				if (!(text == "version"))
				{
					if (!(text == "diagnosticresult"))
					{
						continue;
					}
					foreach (dynamic item2 in item.Value)
					{
						switch ((string)item2.Key.ToLower().Trim())
						{
						case "command":
							command = item2.Value;
							break;
						case "finished":
							finished = item2.Value == "true";
							break;
						case "sequenceid":
							sequenceID = Convert.ToByte(item2.Value);
							break;
						case "result":
							foreach (dynamic item3 in item2.Value)
							{
								data.Add(item3);
							}
							break;
						}
					}
				}
				else
				{
					version = (int)item.Value;
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

	public bool ClearSettings()
	{
		if (isAHP)
		{
			return ExecuteWebRequest("db/reset", "dbname=configuration_item", string.Empty, 5000, 1).RequestState == EWebRequestState.VALID_RESPONSE;
		}
		return SendCommand("eepromx erase config");
	}

	public bool LoadSettingsFromIWS(ICUUser currentUser, string IsahSite, string IsahCredentials, ref IWSObject objData, bool includeLogo = true)
	{
		string text = GetPropertyString(8273, 0, 0).ToLowerInvariant().Trim();
		GetPropertyString(8611, 0, 0).ToLowerInvariant().Trim();
		try
		{
			IWSObject iwsobj = null;
			HttpResponseMessage objectData = IWSConnection.GetObjectData(ref iwsobj, IsahSite, IsahCredentials, text, NumberOfSockets, GetPropertyUInt64(8608, 0, 0uL).ToString(), currentUser.User ?? "");
			if (iwsobj != null && objectData.IsSuccessStatusCode)
			{
				objData = iwsobj;
				return true;
			}
			Logger.Error("The current {ObjectId} is unknown! Please contact Alfen!", text);
			return false;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return false;
		}
	}

	public bool HasProperty(int id, int subid = 0)
	{
		uint combinedPropId = (uint)((id << 8) | (subid & 0xFF));
		if (GetProperty(combinedPropId) != null)
		{
			return GetProperty(combinedPropId).Value != null;
		}
		return false;
	}

	public bool SupportsModem()
	{
		string text = GetPropertyString(4104, 0, 0)?.ToLowerInvariant().Trim();
		if (string.IsNullOrEmpty(text))
		{
			return false;
		}
		if (text.Contains("ng900"))
		{
			return false;
		}
		return true;
	}

	public bool IsOCPPVersionSupported(EOccpVersion version)
	{
		switch (version)
		{
		case EOccpVersion.VERSION_15:
			return !isAHP;
		case EOccpVersion.VERSION_16:
			return true;
		case EOccpVersion.VERSION_20:
			if (!IsAhpFirmwareEqualOrHigherThan(new Version("2.1.0")))
			{
				return IsNg9xxFirmwareEqualOrHigherThan(new Version("4.8.0"));
			}
			return true;
		default:
			return false;
		}
	}

	public bool SupportsNFC()
	{
		string text = GetPropertyString(4104, 0, 0)?.ToLowerInvariant().Trim();
		if (string.IsNullOrEmpty(text))
		{
			return false;
		}
		if (text.Contains("ng900"))
		{
			return false;
		}
		return true;
	}

	private int GetNumberOfFeederCables()
	{
		if (NumberOfSockets == 1 || IsTwinOrDualPG())
		{
			return 1;
		}
		switch (ModelType)
		{
		case ICUDeviceModel.NG920_61002:
		case ICUDeviceModel.NG920_61008:
		case ICUDeviceModel.NG920_61012:
		case ICUDeviceModel.NG920_61018:
		case ICUDeviceModel.NG920_61022:
		case ICUDeviceModel.NG920_61028:
		case ICUDeviceModel.NG920_61032:
		case ICUDeviceModel.NG920_61038:
		case ICUDeviceModel.NG920_61102:
		case ICUDeviceModel.NG920_61128:
		case ICUDeviceModel.NG920_61206:
		case ICUDeviceModel.NG920_61216:
		case ICUDeviceModel.AHP02_63022:
		case ICUDeviceModel.AHP02_63024:
		case ICUDeviceModel.AHP02_63026:
		case ICUDeviceModel.AHP02_63042:
		case ICUDeviceModel.AHP02_63028:
		case ICUDeviceModel.AHP02_63012:
		case ICUDeviceModel.AHP02_63122:
		case ICUDeviceModel.AHP02_63126:
		case ICUDeviceModel.AHP02_63142:
		case ICUDeviceModel.AHP02_63128:
		case ICUDeviceModel.AHP02_63146:
		case ICUDeviceModel.AHP02_63148:
		case ICUDeviceModel.AHP02_63226:
		case ICUDeviceModel.AHP02_63236:
			return 2;
		default:
			return 1;
		}
	}

	public bool IsTwinOrDualPG()
	{
		if (!IsTwin)
		{
			return IsDualPG;
		}
		return true;
	}

	public bool SupportsRats(ERadioAccessTechnology rat)
	{
		if (!HasProperty(8350))
		{
			return false;
		}
		return ((uint)GetPropertyInt(8350, 0) & (uint)rat) == (uint)rat;
	}

	private IList<NetworkProfileDataElement> GetNetworkProfileDataElementsWithPriority(byte networkProfileSubId)
	{
		byte b = 14;
		IList<NetworkProfileDataElement> list = new List<NetworkProfileDataElement>(4);
		for (ushort num = 8432; num <= 8435; num++)
		{
			if (HasProperty(num, b) && HasProperty(num, networkProfileSubId) && GetPropertyInt(num, b) > 0)
			{
				list.Add(new NetworkProfileDataElement
				{
					Priority = GetPropertyInt(num, b),
					DataElement = GetPropertyInt(num, networkProfileSubId)
				});
			}
		}
		return list;
	}

	public string GetPrioritizedOcppVersion()
	{
		NetworkProfileDataElement networkProfileDataElement = (from a in GetNetworkProfileDataElementsWithPriority(1)
			orderby a.Priority, a.DataElement descending
			select a).FirstOrDefault();
		if (networkProfileDataElement != null)
		{
			return networkProfileDataElement.DataElement.ToString();
		}
		return "1";
	}

	public void SetTariffDisplayOptions(TariffDisplayOptionsType tdo)
	{
		ICUProperty property = GetProperty(12898, 5);
		if (property != null)
		{
			if (property.DataType == SDT.UNSIGNED8)
			{
				property.Value = (int)tdo;
			}
			else
			{
				property.Value = ((tdo == TariffDisplayOptionsType.NONE) ? string.Empty : tdo.ToString());
			}
		}
	}

	public TariffDisplayOptionsType GetTariffDisplayOptions()
	{
		ICUProperty property = GetProperty(12898, 5);
		if (property != null)
		{
			if (property.DataType == SDT.UNSIGNED8)
			{
				return (TariffDisplayOptionsType)Convert.ToByte(property.Value);
			}
			Enum.TryParse<TariffDisplayOptionsType>(property.Value.ToString(), ignoreCase: true, out var result);
			return result;
		}
		return TariffDisplayOptionsType.NONE;
	}

	public (bool result, string responseData) ExecutedWifiScan()
	{
		(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("wifiscan", "", null, 11000, 1);
		return (result: tuple.Item1 == EWebRequestState.VALID_RESPONSE, responseData: tuple.Item3);
	}

	private T ReadLocked<T>(Func<T> readFunc)
	{
		_propertyLock.EnterReadLock();
		try
		{
			return readFunc();
		}
		finally
		{
			_propertyLock.ExitReadLock();
		}
	}

	private void WriteLocked(Action writeAction)
	{
		_propertyLock.EnterWriteLock();
		try
		{
			writeAction();
		}
		finally
		{
			_propertyLock.ExitWriteLock();
		}
	}

	public bool ClearPersonalData()
	{
		(EWebRequestState, HttpStatusCode, string) tuple = ExecuteWebRequest("clearpersonaldata", "", string.Empty, 5000, 2);
		if (tuple.Item1 == EWebRequestState.VALID_RESPONSE)
		{
			Logger.Information("Personal data is cleared");
			return true;
		}
		Logger.Error("Personal data not cleared, {ResponseData}", tuple.Item3);
		return false;
	}
}
