using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.IO.Ports;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Net.NetworkInformation;
using System.Net.Security;
using System.Net.Sockets;
using System.Reflection;
using System.Runtime.CompilerServices;
using System.Runtime.Versioning;
using System.Security.Authentication;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using System.Web.Script.Serialization;
using System.Windows.Forms;
using System.Xml.Linq;
using ICUIWSConnection;
using ICUNetwork.Helpers;
using ICUSettings;
using Newtonsoft.Json;
using Newtonsoft.Json.Converters;
using Newtonsoft.Json.Serialization;
using Serilog;
using Serilog.Context;
using Serilog.Events;
using Tmds.MDns;

[assembly: CompilationRelaxations(8)]
[assembly: RuntimeCompatibility(WrapNonExceptionThrows = true)]
[assembly: Debuggable(DebuggableAttribute.DebuggingModes.IgnoreSymbolStoreSequencePoints)]
[assembly: TargetFramework(".NETFramework,Version=v4.8.1", FrameworkDisplayName = ".NET Framework 4.8.1")]
[assembly: AssemblyCompany("Alfen N.V.")]
[assembly: AssemblyConfiguration("Release")]
[assembly: AssemblyCopyright("Copyright © 2025")]
[assembly: AssemblyFileVersion("4.4.1.434")]
[assembly: AssemblyInformationalVersion("4.4.1.434+13315c34f5816fff71821aafad09646ceb3cfbf5")]
[assembly: AssemblyProduct("ACENetwork")]
[assembly: AssemblyTitle("ACENetwork")]
[assembly: AssemblyVersion("4.4.1.434")]
namespace ICUNetwork
{
	public enum UploadStatus
	{
		usIdle,
		usStarting,
		usPending,
		usDone
	}
	public class BaseConnection
	{
		protected string m_sName = "<base>";

		public ObservableCollection<ICUDevice> Devices { get; set; }

		public ObservableCollection<NetworkInterface> NetworkInterfaces { get; set; } = new ObservableCollection<NetworkInterface>();

		public BaseConnection(ObservableCollection<ICUDevice> lstDevices)
		{
			Devices = lstDevices;
		}

		public virtual void Initialize()
		{
		}

		public virtual void StopBrowsing()
		{
		}

		public virtual void StartBrowsing()
		{
		}
	}
	public class CertificateMap
	{
		public CertificateStore.CertificateType type;

		public string certLocation;

		public CertificateMap(CertificateStore.CertificateType type, string location)
		{
			this.type = type;
			certLocation = location;
		}
	}
	public static class CertificateStore
	{
		public enum CertificateType
		{
			ACE_CA_ROOT
		}

		private static readonly List<CertificateMap> certificateMapping = new List<CertificateMap>
		{
			new CertificateMap(CertificateType.ACE_CA_ROOT, "ICUNetwork.Certificates.webserverrootcert.pem")
		};

		public static X509Certificate2 GetCertificate(CertificateType type)
		{
			X509Certificate2 result = null;
			try
			{
				string text = certificateMapping.FirstOrDefault((CertificateMap a) => a.type == type)?.certLocation;
				if (!string.IsNullOrEmpty(text))
				{
					using StreamReader streamReader = new StreamReader(Assembly.GetExecutingAssembly().GetManifestResourceStream(text));
					result = new X509Certificate2(Encoding.ASCII.GetBytes(streamReader.ReadToEnd()));
				}
			}
			catch (Exception exception)
			{
				Log.Logger.Debug(exception, "Failed to retreive certificate");
			}
			return result;
		}
	}
	public enum ICULogType
	{
		UNKNOWN,
		INFO,
		WARNING,
		ERROR,
		COM,
		USER
	}
	public class ICULogLine
	{
		public DateTime Time { get; set; }

		public string Filename { get; set; }

		public int LineNumber { get; set; }

		public string FieldType { get; set; }

		public ICULogType Type { get; set; }

		public string Message { get; set; }

		public ICULogLine(string text)
		{
			string[] array = text.Split(new char[1] { ':' });
			Time = DateTime.Now;
			FieldType = "INFO";
			string text2 = text;
			if (array.Length >= 7)
			{
				string s = (array[0] + ":" + array[1] + ":" + array[2]).Trim(new char[1] { ' ' }).Trim(new char[1] { '$' });
				DateTime result = DateTime.Now;
				DateTime.TryParse(s, out result);
				Time = result;
				FieldType = array[3];
				Filename = array[4];
				int result2 = 0;
				int.TryParse(array[5], out result2);
				LineNumber = result2;
				int startIndex = 6 + array[0].Length + array[1].Length + array[2].Length + array[3].Length + array[4].Length + array[5].Length;
				text2 = text.Substring(startIndex).Trim(new char[1] { '\r' }).Trim(new char[1] { '\n' });
			}
			MatchCollection matchCollection = Regex.Matches(text2, "\\x1B\\[(\\d*;)?(\\d*)m");
			if (matchCollection.Count > 0 && matchCollection[0].Groups.Count > 2)
			{
				int result3 = 0;
				int result4 = 0;
				int.TryParse(matchCollection[0].Groups[1].Value, out result3);
				int.TryParse(matchCollection[0].Groups[2].Value, out result4);
				for (int num = matchCollection.Count - 1; num >= 0; num--)
				{
					text2 = text2.Remove(matchCollection[num].Index, matchCollection[num].Length);
				}
			}
			Type = ICULogType.UNKNOWN;
			switch (FieldType)
			{
			case "INFO":
				Type = ICULogType.INFO;
				break;
			case "WARNING":
				Type = ICULogType.WARNING;
				break;
			case "ERROR":
				Type = ICULogType.ERROR;
				break;
			case "USER":
				Type = ICULogType.USER;
				break;
			case "COM":
				Type = ICULogType.COM;
				break;
			}
			Message = text2;
		}
	}
	public class COMPortInterface
	{
		public delegate void updateVirtualListSizeDelegate();

		public delegate void ensureVisibleDelegate(int index);

		private readonly ILogger Logger = Log.ForContext<COMPortInterface>();

		private readonly SerialPort m_port;

		private readonly byte[] m_buffer = new byte[16384];

		private readonly byte[] m_LineBuffer = new byte[16384];

		private int m_lineBufferPos;

		private readonly List<ICULogLine> m_logLines = new List<ICULogLine>();

		private Timer m_timer;

		public bool EnsureNewLineVisible { get; set; }

		~COMPortInterface()
		{
			if (m_timer != null)
			{
				m_timer.Dispose();
				m_timer = null;
			}
			if (m_port != null && m_port.IsOpen)
			{
				m_port.Close();
			}
		}

		public bool IsValid()
		{
			if (m_port != null)
			{
				return m_port.IsOpen;
			}
			return false;
		}

		private void onDataReceived(byte[] received)
		{
			Buffer.BlockCopy(received, 0, m_LineBuffer, m_lineBufferPos, received.Length);
			m_lineBufferPos += received.Length;
			int num = 0;
			bool flag = false;
			do
			{
				flag = false;
				for (int i = 0; i < m_lineBufferPos; i++)
				{
					if (m_LineBuffer[i] == 10)
					{
						byte[] array = new byte[i];
						Buffer.BlockCopy(m_LineBuffer, 0, array, 0, i);
						string text = Encoding.Default.GetString(array).Trim(new char[3] { '\r', '\n', ' ' });
						if (!string.IsNullOrEmpty(text))
						{
							ICULogLine item = new ICULogLine(text);
							m_logLines.Add(item);
						}
						m_lineBufferPos -= i + 1;
						if (m_lineBufferPos > 0)
						{
							Buffer.BlockCopy(m_LineBuffer, i + 1, m_LineBuffer, 0, m_lineBufferPos);
						}
						flag = true;
						num++;
						break;
					}
				}
			}
			while (flag);
			_ = 0;
		}

		private void startRead()
		{
			Action kickoffRead = null;
			kickoffRead = () =>
			{
				m_port.BaseStream.BeginRead(m_buffer, 0, m_buffer.Length, (IAsyncResult ar) =>
				{
					try
					{
						int num = m_port.BaseStream.EndRead(ar);
						byte[] array = new byte[num];
						Buffer.BlockCopy(m_buffer, 0, array, 0, num);
						onDataReceived(array);
					}
					catch (Exception ex)
					{
						Logger.Error(ex, "Error during reading: {Message}", ex.Message);
					}
					if (m_port.IsOpen)
					{
						kickoffRead();
					}
				}, null);
			};
			kickoffRead();
		}

		private void ensureVisible(int index)
		{
		}

		public static bool IsPortAvailable()
		{
			return SerialPort.GetPortNames().Count() > 0;
		}

		public void Reset()
		{
			if (m_port.IsOpen)
			{
				m_port.WriteLine("\u0012");
			}
		}

		public void SendCommand(string command)
		{
			if (m_port.IsOpen)
			{
				m_port.WriteLine(command);
			}
		}

		public void SendKey(string key)
		{
			if (m_port.IsOpen)
			{
				m_port.Write(key);
			}
		}

		public void Clear()
		{
			m_logLines.Clear();
		}
	}
	public enum ICUChargingProfilePurposeType
	{
		ExternalConstraints,
		MaxProfile,
		TxDefaultProfile,
		TxProfile
	}
	public enum ICURecurrencyKind
	{
		Daily = 0,
		Weekly = 1,
		NotSpecified = 3
	}
	public enum ICUChargingProfileKind
	{
		Absolute,
		Recurring,
		Relative
	}
	public enum ICUChargingRateUnit
	{
		Amperes,
		Watts
	}
	public class ICUChargingProfile
	{
		private readonly ILogger Logger = Log.ForContext<ICUChargingProfile>();

		public int ConnectorId { get; set; }

		public int ChargingProfileId { get; set; }

		public ICUChargingProfileKind ChargingProfileKind { get; set; }

		public ICURecurrencyKind RecurrencyKind { get; set; }

		public ICUChargingProfilePurposeType ChargingProfilePurpose { get; set; }

		public DateTime? StartSchedule { get; set; }

		public DateTime? ValidFrom { get; set; }

		public DateTime? ValidTo { get; set; } = DateTime.MinValue;

		public int StackLevel { get; set; }

		public int TransactionId { get; set; }

		public bool UseLocalTime { get; set; }

		public bool UseRandomisedDelay { get; set; }

		public ICUChargingRateUnit ChargingRateUnit { get; set; }

		public ICUChargingProfile()
		{
		}

		public ICUChargingProfile(int connectorId, dynamic chargingProfileJson)
		{
			ConnectorId = connectorId;
			try
			{
				ChargingProfileId = Convert.ToInt32(chargingProfileJson["chargingProfileId"]);
				ChargingProfileKind = Enum.Parse(typeof(ICUChargingProfileKind), chargingProfileJson["chargingProfileKind"], true);
				RecurrencyKind = Enum.Parse(typeof(ICURecurrencyKind), chargingProfileJson["recurrencyKind"], true);
				ChargingProfilePurpose = Enum.Parse(typeof(ICUChargingProfilePurposeType), chargingProfileJson["chargingProfilePurpose"], true);
				StartSchedule = Convert.ToDateTime(chargingProfileJson["startSchedule"]);
				ValidFrom = ((chargingProfileJson["validFrom"] != null) ? Convert.ToDateTime(chargingProfileJson["validFrom"]) : null);
				ValidTo = ((chargingProfileJson["validTo"] != null) ? Convert.ToDateTime(chargingProfileJson["validTo"]) : null);
				StackLevel = Convert.ToInt32(chargingProfileJson["stackLevel"]);
				TransactionId = Convert.ToInt32(chargingProfileJson["transactionId"]);
				UseLocalTime = Convert.ToBoolean(chargingProfileJson["useLocalTime"]);
				UseRandomisedDelay = Convert.ToBoolean(chargingProfileJson["useRandomisedDelay"]);
				ChargingRateUnit = Enum.Parse(typeof(ICUChargingRateUnit), chargingProfileJson["chargingRateUnit"], true);
			}
			catch (Exception exception)
			{
				Logger.Error(exception, "");
			}
		}

		public override string ToString()
		{
			return $"Profile {ChargingProfileId}";
		}
	}
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
	public enum ICUDeviceModel
	{
		Unknown,
		Twin_3_0,
		Eve_Dual,
		Eve_Single,
		Compact,
		Compact_FC,
		Lolo3,
		Lolo3_FC,
		Eve_Mini,
		Eve_Mini_FC,
		Tube,
		Tube_1,
		Tube_2,
		Twin,
		Twin_4_0,
		Twin_4_0_Single,
		Twin_4_1,
		Twin_4_2,
		Twin_5_0,
		Twin_4_XL,
		NG900_60503,
		NG900_60505,
		NG900_60507,
		NG910_60001,
		NG910_60002,
		NG910_60003,
		NG910_60004,
		NG910_60005,
		NG910_60006,
		NG910_60007,
		NG910_60011,
		NG910_60012,
		NG910_60013,
		NG910_60014,
		NG910_60016,
		NG910_60021,
		NG910_60022,
		NG910_60023,
		NG910_60024,
		NG910_60025,
		NG910_60026,
		NG910_60027,
		NG910_60031,
		NG910_60032,
		NG910_60033,
		NG910_60034,
		NG910_60035,
		NG910_60036,
		NG910_60103,
		NG910_60107,
		NG910_60123,
		NG910_60127,
		NG910_60503,
		NG910_60505,
		NG910_60507,
		NG910_60523,
		NG910_60525,
		NG910_60527,
		NG910_60553,
		NG910_60555,
		NG910_60557,
		NG910_60573,
		NG910_60575,
		NG910_60577,
		NG910_60583,
		NG910_60585,
		NG910_60587,
		NG910_60593,
		NG910_60595,
		NG910_60597,
		NG910_60603,
		NG910_60605,
		NG910_60607,
		NG910_60623,
		NG910_60625,
		NG910_60627,
		NG920_52001,
		NG920_52002,
		NG920_52501,
		NG920_52502,
		NG920_52503,
		NG920_52504,
		NG920_52505,
		NG920_52506,
		NG920_52507,
		NG920_52511,
		NG920_52512,
		NG920_52513,
		NG920_52514,
		NG920_52551,
		NG920_52570,
		NG920_52571,
		NG920_52893,
		NG920_61001,
		NG920_61002,
		NG920_61007,
		NG920_61008,
		NG920_61011,
		NG920_61012,
		NG920_61017,
		NG920_61018,
		NG920_61021,
		NG920_61022,
		NG920_61027,
		NG920_61028,
		NG920_61031,
		NG920_61032,
		NG920_61037,
		NG920_61038,
		NG920_61101,
		NG920_61102,
		NG920_61127,
		NG920_61128,
		NG920_61205,
		NG920_61206,
		NG920_61215,
		NG920_61216,
		NG920_62001,
		NG920_62002,
		NG920_62003,
		NG920_62004,
		NG920_62005,
		AHWP01_52599,
		AHWP01_52651,
		AHWP01_52652,
		AHWP01_52653,
		AHWP01_52654,
		AHWP01_52655,
		AHWP01_52660,
		AHWP01_52661,
		AHWP01_52699,
		AHP01_52750,
		AHP01_52752,
		AHP01_52760,
		AHP02_60223,
		AHP02_60225,
		AHP02_60227,
		AHP02_60323,
		AHP02_60327,
		AHP02_61323,
		AHP02_61423,
		AHP02_63021,
		AHP02_63022,
		AHP02_63023,
		AHP02_63024,
		AHP02_63025,
		AHP02_63026,
		AHP02_63041,
		AHP02_63042,
		AHP02_63027,
		AHP02_63028,
		AHP02_63031,
		AHP02_63012,
		AHP02_63121,
		AHP02_63122,
		AHP02_63125,
		AHP02_63126,
		AHP02_63141,
		AHP02_63142,
		AHP02_63127,
		AHP02_63128,
		AHP02_63145,
		AHP02_63146,
		AHP02_63147,
		AHP02_63148,
		AHP02_63225,
		AHP02_63226,
		AHP02_63235,
		AHP02_63236,
		AHP02_65025,
		AHP02_65027,
		AHPDC_30001
	}
	public enum EBoardRevision
	{
		BOARDREVISION_UNKNOWN = -1,
		BOARDREVISION_DEFAULT,
		BOARDREVISION_A,
		BOARDREVISION_B,
		BOARDREVISION_C,
		BOARDREVISION_D,
		BOARDREVISION_E,
		BOARDREVISION_F,
		BOARDREVISION_G,
		BOARDREVISION_H,
		BOARDREVISION_J,
		BOARDREVISION_K,
		BOARDREVISION_L,
		BOARDREVISION_M,
		BOARDREVISION_N,
		BOARDREVISION_P,
		BOARDREVISION_Q,
		BOARDREVISION_R
	}
	public enum EBoardAssy
	{
		BOARDASSEMBLY_UNKNOWN = -1,
		BOARDASSEMBLY_DEFAULT,
		BOARDASSEMBLY_00,
		BOARDASSEMBLY_01,
		BOARDASSEMBLY_02,
		BOARDASSEMBLY_03,
		BOARDASSEMBLY_04,
		BOARDASSEMBLY_05,
		BOARDASSEMBLY_06,
		BOARDASSEMBLY_07,
		BOARDASSEMBLY_08,
		BOARDASSEMBLY_09,
		BOARDASSEMBLY_10,
		BOARDASSEMBLY_11,
		BOARDASSEMBLY_12,
		BOARDASSEMBLY_13,
		BOARDASSEMBLY_14,
		BOARDASSEMBLY_15
	}
	public static class ICUDeviceModelExtension
	{
		public static ICUDeviceModel From(string hostName, int numberOfSockets = 1, bool fixedCable = false)
		{
			string text = hostName.ToLowerInvariant().Trim();
			if (text.StartsWith("icu"))
			{
				text = text.Remove(0, 3).Trim();
			}
			if (text.StartsWith("-"))
			{
				text = text.Substring(1);
			}
			text = text.Replace(" ", "-");
			text = text.Replace("_", "-");
			for (uint num = 0u; num < 2; num++)
			{
				switch (text)
				{
				case "twin-4-xl":
					return ICUDeviceModel.Twin_4_XL;
				case "twin-4.0":
				case "twin-4-0":
					return ICUDeviceModel.Twin_4_0;
				case "twin-4.0-single":
				case "twin-4-0-single":
					return ICUDeviceModel.Twin_4_0_Single;
				case "twin-4.1":
				case "twin-4-1":
					return ICUDeviceModel.Twin_4_1;
				case "twin-4.2":
				case "twin-4-2":
					return ICUDeviceModel.Twin_4_2;
				case "twin-3.0":
				case "twin-3-0":
					return ICUDeviceModel.Twin_3_0;
				case "twin-5":
				case "twin-5.0":
				case "twin-5-0":
					return ICUDeviceModel.Twin_5_0;
				case "twin":
					return ICUDeviceModel.Twin_3_0;
				}
				if (text.StartsWith("eve-dual"))
				{
					return ICUDeviceModel.Eve_Dual;
				}
				if (text.StartsWith("eve-single"))
				{
					return ICUDeviceModel.Eve_Single;
				}
				if (text.StartsWith("compact"))
				{
					if (fixedCable)
					{
						return ICUDeviceModel.Compact_FC;
					}
					return ICUDeviceModel.Compact;
				}
				if (text.StartsWith("lolo3"))
				{
					if (fixedCable)
					{
						return ICUDeviceModel.Lolo3_FC;
					}
					return ICUDeviceModel.Lolo3;
				}
				if (text.StartsWith("tube"))
				{
					if (numberOfSockets == 1)
					{
						return ICUDeviceModel.Tube_1;
					}
					return ICUDeviceModel.Tube_2;
				}
				if (text.StartsWith("eve-mini"))
				{
					if (fixedCable)
					{
						return ICUDeviceModel.Eve_Mini_FC;
					}
					return ICUDeviceModel.Eve_Mini;
				}
				if (text.StartsWith("ng9") || text.StartsWith("ahwp") || text.StartsWith("ahp"))
				{
					int num2 = (text.StartsWith("ahwp") ? 12 : 11);
					if (text.Length < num2)
					{
						num2 = text.Length;
					}
					if (Enum.TryParse<ICUDeviceModel>(text.Substring(0, num2).Replace('-', '_'), ignoreCase: true, out var result))
					{
						return result;
					}
				}
				if (text.LastIndexOf('-') >= 0)
				{
					text = text.Substring(0, text.LastIndexOf('-'));
				}
			}
			Log.Logger.Error("Model name {0} is not recognized as a valid model", hostName);
			return ICUDeviceModel.Unknown;
		}

		public static string ToString(ICUDeviceModel modelType, string hostName)
		{
			switch (modelType)
			{
			case ICUDeviceModel.Twin_3_0:
				return "Twin 3.0";
			case ICUDeviceModel.Twin_4_0:
				return "Twin 4.0";
			case ICUDeviceModel.Twin_4_0_Single:
				return "Twin 4.0 Single";
			case ICUDeviceModel.Twin_4_1:
				return "Twin 4.1";
			case ICUDeviceModel.Twin_4_2:
				return "Twin 4.2";
			case ICUDeviceModel.Twin_4_XL:
				return "Twin 4 XL";
			case ICUDeviceModel.Twin_5_0:
				return "Twin 5.0";
			case ICUDeviceModel.Lolo3:
			case ICUDeviceModel.Lolo3_FC:
				return "LOLO3";
			case ICUDeviceModel.Eve_Dual:
				return "EVe-dual";
			case ICUDeviceModel.Eve_Single:
				return "EVe-single";
			case ICUDeviceModel.Compact:
			case ICUDeviceModel.Compact_FC:
				return "Compact";
			case ICUDeviceModel.Eve_Mini:
			case ICUDeviceModel.Eve_Mini_FC:
				return "ICU Eve Mini";
			case ICUDeviceModel.Tube:
			case ICUDeviceModel.Tube_1:
			case ICUDeviceModel.Tube_2:
				return "TUBE";
			case ICUDeviceModel.Unknown:
			{
				if (string.IsNullOrEmpty(hostName))
				{
					return "Unknown";
				}
				string text = hostName.ToLowerInvariant().Trim();
				if (text.StartsWith("icu"))
				{
					text = text.Remove(0, 3).Trim();
				}
				if (text.StartsWith("-"))
				{
					text = text.Substring(1);
				}
				text = text.Replace(" ", "-");
				text = text.Replace("_", "-");
				int num = text.LastIndexOf('-');
				if (num > 8)
				{
					return text.Substring(0, num);
				}
				return text;
			}
			default:
				return modelType.ToString().Replace('_', '-');
			}
		}
	}
	public class ICUDevice
	{
		public static readonly DateTime UnixEpoch = new DateTime(1970, 1, 1, 0, 0, 0, DateTimeKind.Utc);

		private readonly ILogger Logger = Log.ForContext<ICUDevice>();

		private const string PropertyConvertErrorFormat = "{0} type: {1} {2} message: {3}";

		public BaseConnection Connection { get; set; }

		public string Name { get; set; }

		public string SerialNumber { get; set; }

		public ICUPropertyDictionary PropertyDictionary { get; set; }

		public int NumberOfSockets { get; set; }

		public int NumberOfFeederCables { get; set; }

		public int[] SocketTypes { get; set; }

		public EndUserAccessType EndUserAccessType { get; set; }

		public string DisplayUserName { get; set; }

		public bool IsConnected
		{
			get
			{
				if (Connection != null)
				{
					return Connection.Devices.Any((ICUDevice a) => a == this);
				}
				return false;
			}
		}

		public virtual string Identification => "<unknown>";

		public virtual string DisplayNameLine2 => "";

		public virtual string Address => "";

		public virtual ICUDeviceModel ModelType => ICUDeviceModel.Unknown;

		public virtual string Model => "Unknown";

		public ICUDevice(BaseConnection connection, string connectionAddress = "")
		{
			Connection = connection;
			PropertyDictionary = new ICUPropertyDictionary();
			Name = string.Empty;
			SerialNumber = string.Empty;
		}

		public ICUProperty GetProperty(uint combinedPropId)
		{
			ushort num = (ushort)(combinedPropId >> 8);
			byte b = (byte)(combinedPropId & 0xFF);
			if (combinedPropId <= 65535)
			{
				num = (ushort)combinedPropId;
				b = 0;
			}
			if (num == 0 && b == 0)
			{
				return null;
			}
			return PropertyDictionary.GetProperty(num, b);
		}

		public ICUProperty GetProperty(ushort propId, byte subId = 0)
		{
			if (propId == 0 && subId == 0)
			{
				return null;
			}
			return PropertyDictionary.GetProperty(propId, subId);
		}

		public string GetPropertyString(ushort propId, byte subId = 0, ushort parentPropId = 0)
		{
			ICUProperty prop = GetProperty(propId, subId);
			ICUProperty iCUProperty = prop;
			if (prop != null && prop.Value != null)
			{
				if (parentPropId != 0)
				{
					iCUProperty = GetProperty(parentPropId, subId);
				}
				if (iCUProperty != null && iCUProperty.Parameter != null && iCUProperty.Parameter.Options != null)
				{
					EDSParameterOption eDSParameterOption = iCUProperty.Parameter.Options.FirstOrDefault((EDSParameterOption a) => a.Value == prop.Value.ToString());
					if (eDSParameterOption != null)
					{
						return eDSParameterOption.Title;
					}
				}
				return prop.Value.ToString();
			}
			return string.Empty;
		}

		public string GetDeviceValueString(ushort propId, byte subId = 0, ushort parentPropId = 0)
		{
			ICUProperty prop = GetProperty(propId, subId);
			ICUProperty iCUProperty = prop;
			if (prop != null && prop.DeviceValue != null)
			{
				if (parentPropId != 0)
				{
					iCUProperty = GetProperty(parentPropId, subId);
				}
				if (iCUProperty != null && iCUProperty.Parameter != null && iCUProperty.Parameter.Options != null)
				{
					EDSParameterOption eDSParameterOption = iCUProperty.Parameter.Options.FirstOrDefault((EDSParameterOption a) => a.Value == prop.DeviceValue.ToString());
					if (eDSParameterOption != null)
					{
						return eDSParameterOption.Title;
					}
				}
				if (prop.DataType == SDT.REAL32 || prop.DataType == SDT.REAL64)
				{
					return Convert.ToDouble(prop.DeviceValue).ToString("0.000", CultureInfo.InvariantCulture);
				}
				return prop.DeviceValue.ToString();
			}
			return string.Empty;
		}

		public int GetPropertyInt(ushort propId, byte subId = 0, int nValueMask = 0)
		{
			ICUProperty property = GetProperty(propId, subId);
			int num = 0;
			if (property != null && property.Value != null)
			{
				try
				{
					num = Convert.ToInt32(property.Value);
					if (nValueMask != 0)
					{
						num &= nValueMask;
					}
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return num;
		}

		public uint GetPropertyUInt(ushort propId, byte subId = 0, uint nValueMask = 0u)
		{
			ICUProperty property = GetProperty(propId, subId);
			uint num = 0u;
			if (property != null && property.Value != null)
			{
				try
				{
					num = Convert.ToUInt32(property.Value);
					if (nValueMask != 0)
					{
						num &= nValueMask;
					}
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return num;
		}

		public bool GetPropertyBool(ushort propId, byte subId = 0, bool defaultValue = false)
		{
			ICUProperty property = GetProperty(propId, subId);
			if (property != null && property.Value != null)
			{
				try
				{
					return Convert.ToBoolean(property.Value);
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return defaultValue;
		}

		public int GetDeviceValueInt(ushort propId, byte subId = 0, int nValueMask = 0)
		{
			ICUProperty property = GetProperty(propId, subId);
			int num = 0;
			if (property != null && property.Value != null)
			{
				try
				{
					num = Convert.ToInt32(property.DeviceValue);
					if (nValueMask != 0)
					{
						num &= nValueMask;
					}
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return num;
		}

		public ulong GetPropertyUInt64(ushort propId, byte subId = 0, ulong defaultValue = 0uL)
		{
			ICUProperty property = GetProperty(propId, subId);
			if (property != null && property.Value != null)
			{
				try
				{
					return Convert.ToUInt64(property.Value);
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return defaultValue;
		}

		public double GetPropertyDouble(ushort propId, byte subId = 0)
		{
			ICUProperty property = GetProperty(propId, subId);
			double result = 0.0;
			if (property != null && property.Value != null)
			{
				try
				{
					result = Convert.ToDouble(property.Value);
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
				}
			}
			return result;
		}

		public virtual bool OnDeviceRemoved()
		{
			return true;
		}

		public virtual bool UpdateProperties(params ICUProperty[] properties)
		{
			return false;
		}

		public virtual bool UpdateProperties(string sIds, bool notifyChanges = true)
		{
			return false;
		}

		public virtual bool UpdateProperties(params uint[] properties)
		{
			string[] value = properties.Select((uint a) => $"{a >> 8:X}_{a & 0xFF:X}").ToArray();
			return UpdateProperties(string.Join(",", value));
		}

		public virtual bool UpdateCategories(params string[] categories)
		{
			return false;
		}

		public virtual List<string> RequestCategories(Action dispatchUiEvents = null)
		{
			return null;
		}

		public virtual bool StoreChangedProperties()
		{
			List<ICUProperty> list = PropertyDictionary.Where((ICUProperty a) => a.IsChanged).ToList();
			if (list != null)
			{
				return StoreProperties(list.ToArray());
			}
			return false;
		}

		public virtual void RevertChanges()
		{
			foreach (ICUProperty item in PropertyDictionary.Where((ICUProperty a) => a.IsChanged).ToList())
			{
				item.Rollback();
			}
		}

		public virtual bool StoreProperties(params ICUProperty[] properties)
		{
			return false;
		}

		public virtual bool UploadFirmware(BackgroundWorker bgw, string fileName, string newPassword = "")
		{
			return false;
		}

		public virtual bool SaveProperties(string fileName)
		{
			return false;
		}
	}
	public enum EDomainItemType
	{
		KEY_BO_AUTHORIZATION = 0,
		KEY_FW_VALIDATION = 1,
		KEY_PROXY_AUTHORIZATION = 2,
		KEY_GETDIAGNOSTICS = 3,
		KEY_FW_ENCRYPTION = 4,
		KEY_DS_ENCRYPTION = 5,
		KEY_PRIVATE_CSR = 6,
		CERT_MANUFACTURER_ROOT = 16,
		CERT_CENTRALSYSTEM_ROOT = 17,
		CERTANDKEY_CLIENTCHAIN = 48,
		PASSWORD_RESET_CODE = 57,
		DOM_UNKNOWN = 255
	}
	public class ICUDomain
	{
		private readonly ILogger Logger = Log.ForContext<ICUDomain>();

		private readonly ICULanDevice m_device;

		public ICUDomain(ICULanDevice device)
		{
			m_device = device;
		}

		public bool AddOrUpdateItem(EDomainItemType type, string data)
		{
			if (m_device == null)
			{
				return false;
			}
			string arg = BitConverter.ToString(Encoding.Default.GetBytes(data)).Replace("-", "");
			Logger.Debug("Adding/updating domain item '{Type}'", type);
			string content = $"{{\"cmd\":\"add\",\"type\":{(int)type},\"data\":\"{arg}\"}}";
			return m_device.ExecuteWebRequest("domain", "", content).RequestState == EWebRequestState.VALID_RESPONSE;
		}
	}
	public enum CustomHttpStatusCode
	{
		LoginLockout = 429
	}
	public enum ScnLibraryVersions
	{
		[Description("1.1")]
		SCN_LIB_VERSION_1_1 = 1,
		[Description("1.2")]
		SCN_LIB_VERSION_1_2,
		[Description("1.3")]
		SCN_LIB_VERSION_1_3,
		[Description("1.4")]
		SCN_LIB_VERSION_1_4,
		[Description("1.5")]
		SCN_LIB_VERSION_1_5,
		[Description("1.6")]
		SCN_LIB_VERSION_1_6
	}
	public enum EMainStates
	{
		STATE_ILLEGAL = -1,
		STATE_UNKNOWN,
		STATE_BOOTING,
		STATE_AVAILABLE,
		STATE_CABLE_CONNECTED,
		STATE_CABLE_CONNECTED_TIMEOUT,
		STATE_EV_CONNECTED,
		STATE_BUTTON_ACTIVATED,
		STATE_NFC_AVAILABLE,
		STATE_NFC_AUTHORISED,
		STATE_WAIT_FOR_EVCONNECT,
		STATE_CHARGING_TEST_RELAYS,
		STATE_CHARGING_POWER_OFF,
		STATE_CHARGING_POWER_OFF_LOW_MAXCURRENT,
		STATE_CHARGING_POWER_STARTING,
		STATE_CHARGING_POWER_ON,
		STATE_CHARGING_POWER_ON_SIMPLIFIED,
		STATE_CHARGING_WAIT_FOR_EV_RECONNECT,
		STATE_CHARGING_TERMINATING,
		STATE_CHARGING_WAKEUP,
		STATE_WAIT_FOR_DISCONNECT,
		STATE_WAIT_FOR_RELEASE_AUTHORISATION,
		STATE_CHARGING_RECOVER_FROM_OUTAGE,
		STATE_ERROR,
		STATE_ERROR_MESSAGE,
		STATE_ERROR_MESSAGE_CABLE_NOT_SUPPORTED,
		STATE_ERROR_ILLEGAL_MODE_3,
		STATE_ERROR_TOO_MANY_RESTARTS,
		STATE_ERROR_CHARGING,
		STATE_ERROR_CHARGING_OVERCURRENT,
		STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING,
		STATE_ERROR_S2_NOT_OPENED,
		STATE_ERROR_PROTECTIVE_EARTH,
		STATE_ERROR_RELAYS,
		STATE_ERROR_LOW_SUPPLY_VOLTAGE,
		STATE_ERROR_INTERNAL_VOLTAGE,
		STATE_ERROR_POWERMETER,
		STATE_ERROR_TEMPERATURE,
		STATE_SUSPENDED,
		STATE_INOPERATIVE,
		STATE_RESERVED,
		STATE_ERROR_CHARGING_RCD_SIGNALED,
		STATE_CHARGING_POWER_OFF_VENTILATING,
		STATE_CHARGING_POWER_OFF_SUSPENDED,
		STATE_CHARGING_POWER_OFF_PHASE_CHANGE,
		STATE_WAIT_FOR_START_METERVALUE,
		STATE_WAIT_FOR_STOP_METERVALUE,
		STATE_ERROR_SOCKET_MOTOR,
		STATE_CABLE_CONNECTED_TYPE_E,
		STATE_CABLE_CONNECTED_TIMEOUT_TYPE_E,
		STATE_CHARGING_TYPE_E,
		STATE_WAIT_FOR_DISCONNECT_TYPE_E,
		STATE_CHARGING_SUSPENDED_TYPE_E,
		STATE_CHARGING_LOW_MAXCURRENT_TYPE_E,
		STATE_INVALID_CARD,
		STATE_EV_CONNECTED_UNAUTHORIZED,
		STATE_WAIT_FOR_DISCONNECT_PP
	}
	[Flags]
	public enum TariffDisplayOptionsType
	{
		NONE = 0,
		disclaimer = 1,
		perKwh = 2,
		perMinute = 4,
		perSession = 8,
		perOther = 0x10,
		adhocOnlyDisclaimer = 0x20
	}
	public enum ELEDStates
	{
		LED_UNKNOWN,
		LED_OFF,
		LED_BOOTING,
		LED_BOOTING_CHECK_MAINS,
		LED_AVAILABLE,
		LED_PREP_AUTHORIZING,
		LED_PREP_AUTHORIZED,
		LED_PREP_CABLE_CONNECTED,
		LED_PREP_EV_CONNECTED,
		LED_CHARGING_PREPARING,
		LED_CHARGING_WAIT_VEHICLE,
		LED_CHARGING_ACTIVE_NORMAL,
		LED_CHARGING_ACTIVE_SIMPLIFIED,
		LED_CHARGING_SUSPENDED_OVERCURRENT,
		LED_CHARGING_SUSPENDED_HF_SWITCHING,
		LED_CHARGING_SUSPENDED_EV_DISCONNECTED,
		LED_FINISH_WAIT_VEHICLE,
		LED_FINISH_WAIT_FOR_DISCONNECT,
		LED_ERROR_PROTECTIVE_EARTH,
		LED_ERROR_POWERLINE_FAULT,
		LED_ERROR_CONTACTOR_FAULT,
		LED_ERROR_CHARGING,
		LED_ERROR_POWERFAILURE,
		LED_ERROR_TEMPERATURE,
		LED_ERROR_ILLEGAL_CP_VALUE,
		LED_ERROR_ILLEGAL_PP_VALUE,
		LED_ERROR,
		LED_ERROR_TOO_MANY_RESTARTS,
		LED_ERRORMESSAGE,
		LED_ERRORMESSAGE_NOT_AUTHORIZED,
		LED_ERRORMESSAGE_CABLE_NOT_SUPPORTED,
		LED_ERRORMESSAGE_S2_NOT_OPENED,
		LED_ERRORMESSAGE_TIMEOUT,
		LED_RESERVED,
		LED_INOPERATIVE,
		LED_LOADBALANCING_LIMITED,
		LED_LOADBALANCING_FORCED_OFF,
		LED_TAG_MODE,
		LED_TAG_MODE_ADDED,
		LED_TAG_MODE_REMOVED,
		LED_CHARGING_NON_CHARGING
	}
	public enum ESocketStates
	{
		POWER_RELAYS_UNKNOWN = -1,
		POWER_MAIN_OFF_BYPASS_OFF,
		POWER_MAIN_ON_BYPASS_OFF,
		POWER_MAIN_ON_BYPASS_ON,
		POWER_MAIN_ON_TRIAC_ON
	}
	public enum ESocketPowerStates
	{
		OFF,
		ON
	}
	public enum EUserInterfaceStates
	{
		UI_STATE_UNKNOWN,
		UI_STATE_BOOTING,
		UI_STATE_AVAILABLE,
		UI_STATE_CABLE_CONNECTED,
		UI_STATE_EV_CONNECTED,
		UI_STATE_CABLE_AUTHORISED,
		UI_STATE_AUTHORISED,
		UI_STATE_COMMUNICATING,
		UI_STATE_POWER_OFF_LOW_MAX_CURRENT,
		UI_STATE_POWER_OFF_SUSPENDED,
		UI_STATE_CHARGING,
		UI_STATE_CHARGING_FULL_LOCKED,
		UI_STATE_CHARGING_FULL_UNLOCKED,
		UI_STATE_WAIT_FOR_EV_RECONNECT,
		UI_STATE_TRANSACTION_INFO,
		UI_STATE_CARD_REJECTED,
		UI_STATE_ERROR,
		UI_STATE_PLEASEWAIT,
		UI_STATE_RESERVED,
		UI_STATE_QRCODE,
		UI_STATE_WARNING,
		UI_STATE_WAIT_FOR_RELEASE,
		UI_STATE_PLEASE_REMOVE_CABLE,
		UI_STATE_PLEASEWAIT_EV_COMM,
		UI_STATE_WAIT_FOR_RELEASE_PC,
		UI_STATE_INVALID_CARD
	}
	public enum EUserInterfaceError
	{
		UI_ERROR_NONE = 0,
		UI_ERROR_GENERIC = 1,
		UI_ERROR_CHARGING_RCD = 101,
		UI_ERROR_RELAYS = 102,
		UI_ERROR_INTERNAL_VOLTAGE = 104,
		UI_ERROR_POWERMETER = 105,
		UI_ERROR_RCD = 106,
		UI_ERROR_SOCKET_MOTOR_STARTUP_OLD = 107,
		UI_ERROR_MISSINGPCID = 108,
		UI_ERROR_NFCREADER = 109,
		UI_ERROR_PROTECTIVE_EARTH = 201,
		UI_ERROR_LOW_SUPPLY_VOLTAGE = 202,
		UI_ERROR_INOPERATIVE = 206,
		UI_ERROR_HIGH_SUPPLY_VOLTAGE = 208,
		UI_ERROR_P1PPORT = 209,
		UI_ERROR_MODBUSTCPIP = 210,
		UI_ERROR_SOCKET_MOTOR_STARTUP = 211,
		UI_ERROR_MISSINGPHASE = 212,
		UI_ERROR_TICPORT = 213,
		UI_ERROR_CHARGING = 301,
		UI_ERROR_CHARGING_OVERCURRENT = 302,
		UI_ERROR_CHARGING_HF_SWITCHING = 303,
		UI_ERROR_CABLE_CONNECTED_TIMEOUT = 304,
		UI_ERROR_TEMPERATURE_HIGH = 401,
		UI_ERROR_TEMPERATURE_LOW = 402,
		UI_ERROR_MESSAGE = 403,
		UI_ERROR_SOCKET_MOTOR = 404,
		UI_ERROR_ILLEGAL_MODE_3_PP = 405,
		UI_ERROR_ILLEGAL_MODE_3_CP = 406,
		UI_ERROR_TILT = 407
	}
	public enum EBootNoticationStates
	{
		NOT_SENT,
		AWAITING_REPLY,
		REJECTED,
		ACCEPTED,
		PENDING
	}
	public enum EMode3States
	{
		STATE_A = 160,
		STATE_A1 = 161,
		STATE_A2 = 162,
		STATE_B1 = 177,
		STATE_B2 = 178,
		STATE_C1 = 193,
		STATE_C2 = 194,
		STATE_D1 = 209,
		STATE_D2 = 210,
		STATE_E = 224,
		STATE_F = 240
	}
	public enum EModbusTCPIPConnectionStates
	{
		COMMUNICATION_IDLE,
		COMMUNICATION_INITIALIZING,
		COMMUNICATION_NORMAL,
		COMMUNICATION_WARNING,
		COMMUNICATION_ERROR
	}
	public enum EStatusIcon
	{
		STATUS_ICON_VALID,
		STATUS_ICON_INFORMATION,
		STATUS_ICON_WARNING,
		STATUS_ICON_ERROR
	}
	public enum EMeterTypes
	{
		ENERGYMETER_NONE = -1,
		ENERGYMETER_MODBUS_CENTRAL = 0,
		ENERGYMETER_FKN_METER = 2,
		ENERGYMETER_TCPIP_CENTRAL = 3,
		ENERGYMETER_TCPIP_SMART = 4,
		ENERGYMETER_P1 = 5,
		ENERGYMETER_RTU_SMART = 6,
		ENERGYMETER_TIC = 7
	}
	public enum ETCPIPSlaveOptions
	{
		TCPIPSLAVE_NONE,
		TCPIPSLAVE_READ,
		TCPIPSLAVE_WRITE,
		TCPIPSLAVE_ALL
	}
	public enum ETCPIPSlaveMode
	{
		BALANCEMODE_NONE,
		BALANCEMODE_SCN,
		BALANCEMODE_SOCKET,
		BALANCEMODE_BOTH
	}
	public enum EAuthorisationMethod
	{
		AUTHORIZE_PLUG_AND_CHARGE,
		AUTHORIZE_NFCREADER,
		AUTHORIZE_CANBUS,
		AUTHORIZE_BUTTON
	}
	public enum EOfflineAuthorisationMethod
	{
		OFFLINE_REFUSE_ALL = 0,
		OFFLINE_ACCEPT_KNOWN = 1,
		OFFLINE_ACCEPT_ALL = 3
	}
	[Flags]
	public enum ERadioAccessTechnology
	{
		GPRS = 1,
		UMTS = 2,
		LTE = 4
	}
	public enum EOcppMeasurand
	{
		MEAS_NONE = 0,
		MEAS_ENERGY_ACTIVE_EXPORT = 1,
		MEAS_ENERGY_ACTIVE_IMPORT = 2,
		MEAS_ENERGY_REACTIVE_EXPORT = 3,
		MEAS_ENERGY_REACTIVE_IMPORT = 4,
		MEAS_ENERGY_ACTIVE_EXPORT_INTERVAL = 5,
		MEAS_ENERGY_ACTIVE_IMPORT_INTERVAL = 6,
		MEAS_ENERGY_REACTIVE_EXPORT_INTERVAL = 7,
		MEAS_ENERGY_REACTIVE_IMPORT_INTERVAL = 8,
		MEAS_POWER_ACTIVE_EXPORT = 9,
		MEAS_POWER_ACTIVE_IMPORT = 10,
		MEAS_POWER_REACTIVE_EXPORT = 11,
		MEAS_POWER_REACTIVE_IMPORT = 12,
		MEAS_CURRENT_EXPORT = 13,
		MEAS_CURRENT_IMPORT = 14,
		MEAS_VOLTAGE = 15,
		MEAS_TEMP = 16,
		MEAS_ALFEN_CURRENT_L1 = 17,
		MEAS_ALFEN_CURRENT_L2 = 18,
		MEAS_ALFEN_CURRENT_L3 = 19,
		MEAS_ALFEN_CURRENT_MAX = 20,
		MEAS_POWER_FACTOR = 21,
		MEAS_CURRENT_OFFERED = 22,
		MEAS_POWER_OFFERED = 23,
		MEAS_FREQUENCY = 24,
		MEAS_RPM = 25,
		MEAS_SOC = 26,
		MEAS_ENERGY_ACTIVE_NET = 29,
		MEAS_ENERGY_REACTIVE_NET = 30,
		MEAS_ENERGY_APPARENT_NET = 31,
		MEAS_ENERGY_APPARENT_IMPORT = 32,
		MEAS_ENERGY_APPARENT_EXPORT = 33
	}
	public enum EOccpPhase
	{
		PHASE_NONE,
		PHASE_L1,
		PHASE_L2,
		PHASE_L3,
		PHASE_N,
		PHASE_L1N,
		PHASE_L2N,
		PHASE_L3N,
		PHASE_L1L2,
		PHASE_L2L3,
		PHASE_L3L1
	}
	public enum EOccpUnit
	{
		UNIT_NONE,
		UNIT_WH,
		UNIT_KWH,
		UNIT_VARH,
		UNIT_KVARH,
		UNIT_W,
		UNIT_KW,
		UNIT_VAR,
		UNIT_KVAR,
		UNIT_AMP,
		UNIT_VOLT,
		UNIT_CELSIUS,
		UNIT_VA,
		UNIT_KVA,
		UNIT_KELVIN,
		UNIT_FAHRENH,
		UNIT_PERCENT
	}
	[Flags]
	public enum EOccpVersion
	{
		VERSION_NONE = 0,
		VERSION_15 = 1,
		VERSION_16 = 2,
		VERSION_20 = 4,
		VERSION_15_16 = VERSION_15 | VERSION_16,
		VERSION_16_20 = VERSION_16 | VERSION_20,
		VERSION_15_16_20 = VERSION_15_16 | VERSION_20
	}
	public enum ECPConfigurationFlags
	{
		FLAG_32A = 1,
		FLAG_3PHASE = 2,
		FLAG_CABLE = 112,
		FLAG_TYPE_E = 128
	}
	public enum ESocketType
	{
		FIXED_CABLE_UNKNOWN = 0,
		MENNEKES = 1,
		FCT = 2,
		SCHUKO = 3,
		FIXED_CABLE_TYPE_1 = 4,
		FIXED_CABLE_TYPE_2 = 5,
		UNKNOWN = 99
	}
	public enum EWebRequestState
	{
		VALID_RESPONSE,
		INVALID_RESPONSE,
		RETRY_REQUEST,
		UNSUCCESSFUL
	}
	public enum EGiroEState
	{
		DISABLED,
		ENABLED,
		OFFLINE
	}
	public enum ESolarChargingModes
	{
		[Description("Off")]
		SOLAR_CHARGING_OFF,
		[Description("Comfort")]
		SOLAR_CHARGING_COMFORT,
		[Description("Green")]
		SOLAR_CHARGING_GREEN
	}
	public enum EFirmwareUpdateStatus
	{
		NO_ACTIVE_UPDATE = 0,
		ERASING_BUFFER = 1,
		BUFFER_ERASED = 2,
		READY_FOR_DOWNLOAD = 3,
		DOWNLOADING_FIRMWARE = 4,
		DOWNLOAD_DONE = 5,
		DOWNLOAD_CHECKED = 6,
		READY_FOR_UPDATE = 7,
		UPDATE_IN_PROGRESS = 8,
		UPDATE_READY_TO_ROLL = 9,
		UPDATE_DONE = 10,
		CRC_CALCULATING = 11,
		CRC_CALCULATED = 12,
		[Description("Error during download")]
		ERROR_DURING_DOWNLOAD = -1,
		[Description("Error during update")]
		ERROR_DURING_UPDATE = -2,
		[Description("Executing rollback")]
		EXECUTING_ROLLBACK = -3,
		[Description("Rolled back")]
		ROLLED_BACK = -4
	}
	public enum EAHWPCSMMainStates
	{
		Unknown = 0,
		Booting = 15,
		Available = 1,
		Authorising = 2,
		Authorised = 4,
		Rejected = 8,
		CableConnected = 16,
		CableConnectedAuthorising = 18,
		CableConnectedAuthorised = 20,
		CableConnectedRejected = 24,
		EVConnected = 48,
		EVConnectedAuthorising = 50,
		EVConnectedAuthorised = 52,
		EVConnectedRejected = 56,
		CableLocked = 65,
		ChargingStarting = 66,
		Charging = 67,
		ChargingFinishing = 68,
		ChargingFinished = 69,
		CableUnlock = 70,
		SuspendedEV = 71,
		SuspendedEVSE = 72,
		ChargingEVFull = 73,
		ChargeParameterDiscovery = 74,
		CableCheck = 75,
		PreCharge = 76,
		WaitForCableDisconnect = 79,
		TimeoutWaitingForCable = 128,
		TimeoutWaitingForEVConnect = 129,
		TimeoutWaitingForAuthorisation = 130,
		TimeoutWaitingForS2 = 131,
		TimeoutWaitingForCableRemoval = 132,
		Offline = 159,
		Inoperative = 160,
		Reserved = 161,
		TariffAndOrTimeChanged = 162,
		ErrorMask = 192,
		ErrorRelay = 193,
		ErrorTemperatureHigh = 194,
		ErrorOvercurrent = 195,
		ErrorSocketMotor = 196,
		ErrorIllegalMode3CP = 197,
		ErrorEnergyMeter = 198,
		ErrorPhase = 199,
		ErrorInternalRCDTripped = 200,
		ErrorHFSwitching = 201,
		ErrorLowSupplyVoltage = 202,
		ErrorExternalRCD = 203,
		ErrorGeneric = 204,
		ErrorTamper = 205,
		ErrorComponent = 206,
		ErrorInternalVoltage = 207,
		ErrorIllegalMode3PP = 208,
		ErrorRelayOrRCD = 209,
		ErrorSocketMotorBoot = 210,
		ErrorTemperatureLow = 211,
		ErrorInternalRCDFailure = 212,
		ErrorSCBMask = 224,
		ErrorStationError = 225,
		ErrorMissingRFID = 226,
		ErrorMissingPnCID = 227,
		ErrorMissingSignedMeter = 228,
		ErrorMissingTariffs = 229
	}
	public enum EAHWPCCStates
	{
		unknown,
		init,
		idle,
		waitForStartMV,
		preparing,
		waitForEVConnect,
		chargingPowerOn,
		chargingPowerOff,
		chargingWaitForEVReconnect,
		switchPhases,
		waitForStopMV,
		finished,
		lowSupplyVoltageError,
		recoverFromOutage,
		relayError,
		chargingSuspendedEVSE,
		temperatureError,
		overcurrentError,
		socketMotorError,
		illegalMode3Error,
		energyMeterError,
		phaseError,
		internalRCDError,
		hfSwitchingError,
		stopRequested,
		stationError,
		chargeParameterDiscovery,
		cableCheck,
		preCharge,
		genericError,
		tamperError,
		componentError,
		InternalVoltageError,
		IllegalMode3PPError,
		SocketMotorBootError,
		TemperatureLowError,
		InternalRCDTrippedError,
		stateCount
	}
	public enum EAHWPCPROStates
	{
		CPRO_STATE_UNKNOWN,
		CPRO_STATE_INACTIVE,
		CPRO_STATE_CONNECTED_ISO15118_PWM,
		CPRO_STATE_WAIT_FOR_EV_CONNECT,
		CPRO_STATE_EV_CONNECTED,
		CPRO_STATE_ACTIVE,
		CPRO_STATE_WAIT_FOR_S2_CLOSE,
		CPRO_STATE_WAIT_FOR_S2_OPEN,
		CPRO_STATE_SUSPENDED,
		CPRO_STATE_VENTILATING,
		CPRO_STATE_WAKEUP_STATE_E,
		CPRO_STATE_WAKEUP_STATE_B1,
		CPRO_STATE_ERROR,
		CPRO_STATE_ERROR_EV_DETECT,
		CPRO_STATE_WAIT_FOR_EV_DISCONNECT,
		CPRO_STATE_PREPARED,
		CPRO_STATE_CONNECTED_ISO15118_ERROR,
		CPRO_STATE_CONNECTED_ISO15118_X1,
		CPRO_STATE_CHECK_RELAYS,
		CPRO_STATE_COUNT
	}
	public enum EAUXBoardStates
	{
		BOARD_NOT_CONNECTED,
		BOARD_CONNECTED,
		BOARD_COMMUNICATION
	}
	public enum EAlbConfiguration
	{
		NoLicense,
		NotConfigered,
		EMS,
		SM_Tcp_Custom,
		SM_Tcp_Socomec,
		SM_TicLinky,
		SM_Rtu,
		SM_P1_Serial,
		SM_P1_Telnet,
		SM_P1_HomeWizard,
		SM_P1_Unknown
	}
	[Flags]
	public enum EDirectPaymentOptions
	{
		None = 0,
		OTS = 1,
		QRCode = 2,
		GiroE = 4
	}
	public enum EEVSEType
	{
		AC,
		EcogDC
	}
	public enum EndUserAccessType
	{
		NotAvailable,
		[Description("Disabled")]
		Disabled,
		[Description("Enabled (without PIN)")]
		Enabled,
		[Description("Enabled (with PIN)")]
		Configured
	}
	public static class EnumExtensions
	{
		public static string GetEnumDescription(this Enum genericEnum)
		{
			string text = genericEnum.GetDescriptionFromAttribute();
			if (string.IsNullOrEmpty(text))
			{
				text = genericEnum.ToString();
			}
			return text;
		}

		private static string GetDescriptionFromAttribute(this Enum genericEnum)
		{
			MemberInfo[] member = genericEnum.GetType().GetMember(genericEnum.ToString());
			if (member != null && member.Length != 0)
			{
				object[] customAttributes = member[0].GetCustomAttributes(typeof(DescriptionAttribute), inherit: false);
				if (customAttributes != null && customAttributes.Length != 0)
				{
					return ((DescriptionAttribute)customAttributes[0]).Description;
				}
			}
			return null;
		}
	}
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
	public class NetworkProfileDataElement
	{
		public int Priority { get; set; }

		public object DataElement { get; set; }
	}
	public class ACEWebLoginData
	{
		public const string AdminUser = "admin";

		public const string ServiceUser = "service";

		public const string TempUser = "temp";

		public string Username { get; set; }

		public string Password { get; set; }

		public string DisplayName { get; set; }

		public HttpStatusCode LastHttpStatusCode { get; set; }

		public bool IsLoggedIn { get; set; }

		public bool IsLoginCancelled { get; set; }

		public string LoginError { get; set; }

		public bool IsTempUser => Username == "temp";

		public bool IsAdminUser => Username == "admin";

		public bool IsServiceUser => Username == "service";
	}
	internal class ReentrantAsyncLock
	{
		private readonly AsyncLocal<SemaphoreSlim> _semaphore = new AsyncLocal<SemaphoreSlim>();

		public async Task<T> WithLock<T>(Func<Task<T>> func)
		{
			SemaphoreSlim currentSemaphore = _semaphore.Value ?? new SemaphoreSlim(1);
			await currentSemaphore.WaitAsync();
			SemaphoreSlim nextSemaphore = new SemaphoreSlim(1);
			_semaphore.Value = nextSemaphore;
			T result;
			try
			{
				result = await func();
			}
			finally
			{
				await nextSemaphore.WaitAsync();
				_semaphore.Value = currentSemaphore;
				currentSemaphore.Release();
			}
			return result;
		}
	}
	internal class LogingResponse
	{
		[JsonProperty("access")]
		internal string AccessToken;

		[JsonProperty("refresh")]
		internal string RefreshToken;
	}
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
	public enum ICULanLogType
	{
		UNKNOWN,
		INFO,
		WARNING,
		ERROR,
		COM,
		USER,
		RESET,
		CONSOLE,
		SECURITY
	}
	public enum ICUMode3States
	{
		Unknown,
		Booting,
		STATE_B1,
		STATE_B2,
		STATE_C1,
		STATE_C2,
		STATE_E,
		STATE_F
	}
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
	public class ICUMasterTag
	{
		protected ICUDevice m_device;

		private bool m_masterTagEnabled;

		private string m_masterTagId = string.Empty;

		public string Tag
		{
			get
			{
				m_masterTagId = "";
				if (m_device == null)
				{
					return m_masterTagId;
				}
				if (IsFeatureSupported())
				{
					ICUProperty property = m_device.GetProperty(9216, 2);
					if (property != null)
					{
						m_device.UpdateProperties(null, property);
						m_masterTagId = m_device.GetPropertyString(9216, 2, 0);
					}
				}
				return m_masterTagId;
			}
		}

		public bool Enabled
		{
			get
			{
				if (m_device == null)
				{
					return false;
				}
				m_masterTagEnabled = false;
				if (IsFeatureSupported())
				{
					ICUProperty property = m_device.GetProperty(9216, 1);
					if (property != null)
					{
						m_device.UpdateProperties(null, property);
						m_masterTagEnabled = m_device.GetPropertyBool(9216, 1);
					}
				}
				return m_masterTagEnabled;
			}
		}

		public ICUMasterTag(ICUDevice device)
		{
			ReInitialize(device);
		}

		public void ReInitialize(ICUDevice device)
		{
			m_device = device;
		}

		public bool Set(string tag)
		{
			if (m_device == null)
			{
				return false;
			}
			ICUProperty property = m_device.GetProperty(9216, 2);
			if (property != null)
			{
				property.Value = tag;
				return m_device.StoreProperties(property);
			}
			return false;
		}

		public bool Clear()
		{
			return Set("");
		}

		public bool IsFeatureSupported()
		{
			if (m_device == null)
			{
				return false;
			}
			if (m_device.GetProperty(9216, 1) != null)
			{
				return m_device.GetProperty(9216, 2) != null;
			}
			return false;
		}
	}
	public enum EnergyMeterMeasurand
	{
		VOLTAGE_L1N,
		VOLTAGE_L2N,
		VOLTAGE_L3N,
		VOLTAGE_L1L2,
		VOLTAGE_L2L3,
		VOLTAGE_L3L1,
		CURRENT_N,
		CURRENT_L1,
		CURRENT_L2,
		CURRENT_L3,
		CURRENT_SUM,
		COSPHI_L1,
		COSPHI_L2,
		COSPHI_L3,
		COSPHI_SUM,
		FREQUENCY,
		POWER_REAL_L1,
		POWER_REAL_L2,
		POWER_REAL_L3,
		POWER_REAL_SUM,
		POWER_APPARENT_L1,
		POWER_APPARENT_L2,
		POWER_APPARENT_L3,
		POWER_APPARENT_SUM,
		POWER_REACTIVE_L1,
		POWER_REACTIVE_L2,
		POWER_REACTIVE_L3,
		POWER_REACTIVE_SUM,
		ENERGY_REAL_DELIVERED_L1,
		ENERGY_REAL_DELIVERED_L2,
		ENERGY_REAL_DELIVERED_L3,
		ENERGY_REAL_DELIVERED_SUM,
		ENERGY_REAL_CONSUMED_L1,
		ENERGY_REAL_CONSUMED_L2,
		ENERGY_REAL_CONSUMED_L3,
		ENERGY_REAL_CONSUMED_SUM,
		ENERGY_APPARENT_L1,
		ENERGY_APPARENT_L2,
		ENERGY_APPARENT_L3,
		ENERGY_APPARENT_SUM,
		ENERGY_REACTIVE_L1,
		ENERGY_REACTIVE_L2,
		ENERGY_REACTIVE_L3,
		ENERGY_REACTIVE_SUM,
		MAX_MEASURAND_COUNT
	}
	public enum ModbusDataType
	{
		SIGNED16,
		UNSIGNED16,
		SIGNED32,
		UNSIGNED32,
		SIGNED64,
		UNSIGNED64,
		FLOAT32,
		FLOAT64
	}
	public class ModbusRegmapEntry
	{
		[JsonConverter(typeof(StringEnumConverter))]
		public EnergyMeterMeasurand Key { get; set; }

		public ushort RegNum { get; set; }

		[JsonConverter(typeof(StringEnumConverter))]
		public ModbusDataType DataType { get; set; }

		public sbyte ScaleE { get; set; }
	}
	public class ModbusRegmapData
	{
		public string Name { get; set; }

		public string Version { get; set; }

		public byte? Address { get; set; }

		public string Parity { get; set; }

		public ushort? Baudrate { get; set; }

		public string WordOrder { get; set; }

		public uint? UpdateTime { get; set; }

		public uint? ReadTimeout { get; set; }

		public string FunctionCode { get; set; }

		public ushort SampleIntervalMs { get; set; }

		public List<ModbusRegmapEntry> Regmap { get; set; }
	}
	public class ICUModbusRegmap
	{
		private readonly ILogger Logger = Log.ForContext<ModbusRegmapEntry>();

		private readonly ICUDevice m_device;

		public ModbusRegmapData RegmapData { get; set; }

		public List<string> LoadFromJSONErrorList { get; set; }

		public List<string> WriteToDeviceErrorList { get; set; }

		public ICUModbusRegmap(ICUDevice m_device)
		{
			this.m_device = m_device;
			LoadFromJSONErrorList = new List<string>();
			WriteToDeviceErrorList = new List<string>();
		}

		public bool LoadFromDevice(EMeterTypes energyMeterType)
		{
			if (!m_device.UpdateCategories("MbusTCP"))
			{
				Logger.Error("Failed updating category MbusTCP");
				return false;
			}
			bool flag = energyMeterType == EMeterTypes.ENERGYMETER_MODBUS_CENTRAL;
			ICUProperty property = m_device.GetProperty(flag ? 9568u : 9584u);
			ICUProperty property2 = m_device.GetProperty(flag ? 9569u : 9585u);
			ICUProperty property3 = m_device.GetProperty(flag ? 9570u : 9586u);
			ICUProperty property4 = m_device.GetProperty(flag ? 9571u : 9587u);
			if (property == null || property2 == null || property3 == null || property4 == null)
			{
				Logger.Error("Missing a modbusTCP/IP register mapping property");
				return false;
			}
			byte[] array = (byte[])property.Value;
			ushort[] array2 = (ushort[])property2.Value;
			byte[] array3 = (byte[])property3.Value;
			sbyte[] array4 = (sbyte[])property4.Value;
			RegmapData = new ModbusRegmapData
			{
				Name = "",
				Version = "",
				Address = null,
				Parity = "",
				Baudrate = null,
				WordOrder = "",
				UpdateTime = null,
				ReadTimeout = null,
				FunctionCode = "",
				SampleIntervalMs = 2000,
				Regmap = new List<ModbusRegmapEntry>()
			};
			bool flag2 = false;
			for (int i = 0; i < array.Length; i++)
			{
				if (array[i] != byte.MaxValue)
				{
					try
					{
						RegmapData.Regmap.Add(new ModbusRegmapEntry
						{
							Key = (EnergyMeterMeasurand)Enum.ToObject(typeof(EnergyMeterMeasurand), array[i]),
							RegNum = array2[i],
							DataType = (ModbusDataType)Enum.ToObject(typeof(ModbusDataType), array3[i]),
							ScaleE = array4[i]
						});
					}
					catch (Exception ex)
					{
						Logger.Error(ex, "Can't parse at key index:{Index} msg:{Message}", i, ex.Message);
						flag2 = true;
					}
				}
			}
			return !flag2;
		}

		public bool WriteToDevice(EMeterTypes energyMeterType)
		{
			WriteToDeviceErrorList.Clear();
			if (!m_device.UpdateCategories("MbusTCP"))
			{
				WriteToDeviceErrorList.Add("Failed updating category MbusTCP.");
				return false;
			}
			bool flag = energyMeterType == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL;
			ICUProperty property = m_device.GetProperty(flag ? 9568u : 9584u);
			ICUProperty property2 = m_device.GetProperty(flag ? 9569u : 9585u);
			ICUProperty property3 = m_device.GetProperty(flag ? 9570u : 9586u);
			ICUProperty property4 = m_device.GetProperty(flag ? 9571u : 9587u);
			if (property == null || property2 == null || property3 == null || property4 == null)
			{
				WriteToDeviceErrorList.Add("Missing a modbusTCP/IP register mapping property.");
				return false;
			}
			if (RegmapData.Regmap.Count > ((byte[])property.Value).Length)
			{
				WriteToDeviceErrorList.Add($"Maximum number of Modbus TCP/IP register mapping entries is {((byte[])property.Value).Length}.");
				return false;
			}
			List<byte> list = new List<byte>();
			List<ushort> list2 = new List<ushort>();
			List<byte> list3 = new List<byte>();
			List<sbyte> list4 = new List<sbyte>();
			foreach (ModbusRegmapEntry item in RegmapData.Regmap)
			{
				list.Add((byte)item.Key);
				list2.Add(item.RegNum);
				list3.Add((byte)item.DataType);
				list4.Add(item.ScaleE);
			}
			while (list.Count < ((byte[])property.Value).Length)
			{
				list.Add(byte.MaxValue);
				list2.Add(0);
				list3.Add(0);
				list4.Add(0);
			}
			property.Value = list.ToArray();
			property2.Value = list2.ToArray();
			property3.Value = list3.ToArray();
			property4.Value = list4.ToArray();
			List<ICUProperty> list5 = new List<ICUProperty>();
			if (property.IsChanged)
			{
				list5.Add(property);
			}
			if (property2.IsChanged)
			{
				list5.Add(property2);
			}
			if (property3.IsChanged)
			{
				list5.Add(property3);
			}
			if (property4.IsChanged)
			{
				list5.Add(property4);
			}
			if (list5.Count != 0)
			{
				if ((m_device as ICULanDevice).HasProperty(9507, 2))
				{
					(m_device as ICULanDevice).storeProperty(9507, 2, 2);
				}
				(m_device as ICULanDevice).storeProperty(8292, 0, 0);
				if (!m_device.StoreProperties(list5.ToArray()))
				{
					WriteToDeviceErrorList.Add("Storing properties failed.");
					return false;
				}
				(m_device as ICULanDevice).storeProperty(8292, 0, 3);
			}
			return true;
		}

		public void LoadFromJSON(string json)
		{
			LoadFromJSONErrorList.Clear();
			if (string.IsNullOrWhiteSpace(json))
			{
				LoadFromJSONErrorList.Add("json is empty");
				return;
			}
			JsonSerializerSettings settings = new JsonSerializerSettings
			{
				Error = HandleDeserializationError,
				MissingMemberHandling = MissingMemberHandling.Error
			};
			RegmapData = JsonConvert.DeserializeObject<ModbusRegmapData>(json, settings);
		}

		public string WriteToJSON()
		{
			return JsonConvert.SerializeObject(RegmapData, Formatting.Indented);
		}

		private void HandleDeserializationError(object sender, Newtonsoft.Json.Serialization.ErrorEventArgs errorArgs)
		{
			string message = errorArgs.ErrorContext.Error.Message;
			Logger.Error(message);
			LoadFromJSONErrorList.Add(message);
			errorArgs.ErrorContext.Handled = true;
		}

		public ModbusRegmapEntry GetEntry(EnergyMeterMeasurand key)
		{
			return RegmapData.Regmap.FirstOrDefault((ModbusRegmapEntry a) => a.Key == key);
		}
	}
	internal static class ICUNetworkConfig
	{
		public static string HTTPUsernameOld = "cpadmin";

		public static string HTTPPasswordOld = "L@0Pa$$";
	}
	public enum SDT
	{
		LOW_LIMIT = 0,
		BOOLEAN = 1,
		INTEGER8 = 2,
		INTEGER16 = 3,
		INTEGER24 = 16,
		INTEGER32 = 4,
		INTEGER40 = 18,
		INTEGER48 = 19,
		INTEGER56 = 20,
		INTEGER64 = 21,
		UNSIGNED8 = 5,
		UNSIGNED16 = 6,
		UNSIGNED24 = 22,
		UNSIGNED32 = 7,
		UNSIGNED40 = 24,
		UNSIGNED48 = 25,
		UNSIGNED56 = 26,
		UNSIGNED64 = 27,
		REAL32 = 8,
		REAL64 = 17,
		VISIBLE_STRING = 9,
		OCTET_STRING = 10,
		UNICODE_STRING = 11,
		TIME_OF_DAY = 12,
		TIME_DIFFERENCE = 13,
		DOAMIN = 15,
		BYTEARRAY = 64,
		ARRAY_16 = 65,
		HIGH_LIMIT = 65535
	}
	[DebuggerDisplay("{DebuggerDisplay,nq}")]
	public class ICUProperty
	{
		private readonly ILogger Logger = Log.ForContext<ICUProperty>();

		private object m_objValue;

		public ushort Id { get; set; }

		public byte SubId { get; set; }

		public string ICUName { get; set; }

		public string Title { get; set; }

		public string Units { get; set; }

		public SDT DataType { get; set; }

		public bool ReadOnly { get; set; }

		public bool IsChanged { get; private set; }

		public object DeviceValue { get; private set; }

		public string Category { get; set; }

		public ulong MaxLength { get; set; }

		private string DebuggerDisplay => $"{ID_SUB} ({DataType}) = {Value}";

		public object Value
		{
			get
			{
				return m_objValue;
			}
			set
			{
				SetValue(value);
			}
		}

		public EDSParameter Parameter { get; set; }

		public string ID_SUB => $"{Id:X4}_{SubId:X2}";

		public string ODIndex => $"{Id:X4}_{SubId:X}";

		public XElement Element
		{
			get
			{
				if (DataType == SDT.BYTEARRAY)
				{
					string value = "";
					byte[] array = (byte[])Value;
					if (array != null)
					{
						value = string.Join(",", array.Select((byte a) => a.ToString("X2")));
					}
					return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", value));
				}
				if (DataType == SDT.ARRAY_16)
				{
					string value2 = "";
					ushort[] array2 = (ushort[])Value;
					if (array2 != null)
					{
						value2 = string.Join(",", array2.Select((ushort a) => a.ToString("X4")));
					}
					return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", value2));
				}
				return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", Value));
			}
		}

		public event EventHandler ValueChanged;

		public event EventHandler<ValueExceptionEventArgs> ExceptionOccured;

		internal ICUProperty()
		{
		}

		protected void SetValue(object newValue, bool invokeChange = true)
		{
			if (newValue == null)
			{
				Logger.Error("Setting a null value for {Name} Index={Index} Category={Category}", ICUName, ID_SUB, Category);
				return;
			}
			string text = (newValue.GetType().IsArray ? string.Empty : newValue.ToString());
			object objValue = m_objValue;
			try
			{
				switch (DataType)
				{
				case SDT.VISIBLE_STRING:
					m_objValue = newValue;
					break;
				case SDT.UNSIGNED8:
				{
					byte result12;
					if (text.Equals("false", StringComparison.OrdinalIgnoreCase))
					{
						m_objValue = (byte)0;
					}
					else if (text.Equals("true", StringComparison.OrdinalIgnoreCase))
					{
						m_objValue = (byte)1;
					}
					else if (byte.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out result12))
					{
						m_objValue = result12;
					}
					else
					{
						m_objValue = (byte)0;
					}
					break;
				}
				case SDT.INTEGER8:
				{
					sbyte result6;
					if (text.Equals("false", StringComparison.OrdinalIgnoreCase))
					{
						m_objValue = (sbyte)0;
					}
					else if (text.Equals("true", StringComparison.OrdinalIgnoreCase))
					{
						m_objValue = (sbyte)1;
					}
					else if (sbyte.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out result6))
					{
						m_objValue = result6;
					}
					else
					{
						m_objValue = (sbyte)0;
					}
					break;
				}
				case SDT.UNSIGNED16:
				{
					if (ushort.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result9))
					{
						m_objValue = result9;
					}
					else
					{
						m_objValue = (ushort)0;
					}
					break;
				}
				case SDT.UNSIGNED32:
				{
					if (uint.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result2))
					{
						m_objValue = result2;
					}
					else
					{
						m_objValue = 0u;
					}
					break;
				}
				case SDT.UNSIGNED64:
				{
					if (ulong.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result8))
					{
						m_objValue = result8;
					}
					else
					{
						m_objValue = 0uL;
					}
					break;
				}
				case SDT.INTEGER16:
				{
					if (short.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result3))
					{
						m_objValue = result3;
					}
					else
					{
						m_objValue = (short)0;
					}
					break;
				}
				case SDT.INTEGER32:
				{
					if (int.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result11))
					{
						m_objValue = result11;
					}
					else
					{
						m_objValue = 0;
					}
					break;
				}
				case SDT.INTEGER64:
				{
					if (long.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result7))
					{
						m_objValue = result7;
					}
					else
					{
						m_objValue = 0L;
					}
					break;
				}
				case SDT.REAL32:
				{
					if (float.TryParse(text.Replace(',', '.'), NumberStyles.Float, CultureInfo.InvariantCulture, out var result4))
					{
						m_objValue = result4;
					}
					else
					{
						m_objValue = 0f;
					}
					break;
				}
				case SDT.REAL64:
				{
					if (double.TryParse(text.Replace(',', '.'), NumberStyles.Float, CultureInfo.InvariantCulture, out var result13))
					{
						m_objValue = result13;
					}
					else
					{
						m_objValue = 0.0;
					}
					break;
				}
				case SDT.BOOLEAN:
				{
					if (bool.TryParse(text, out var result10))
					{
						m_objValue = result10;
					}
					else
					{
						m_objValue = (byte)0;
					}
					break;
				}
				case SDT.BYTEARRAY:
				{
					if (newValue.GetType().IsArray)
					{
						m_objValue = newValue;
						break;
					}
					List<byte> list2 = new List<byte>();
					string[] array = text.Split(new char[1] { ',' });
					for (int i = 0; i < array.Length; i++)
					{
						if (byte.TryParse(array[i], NumberStyles.HexNumber, CultureInfo.InvariantCulture, out var result5))
						{
							list2.Add(result5);
						}
						else
						{
							list2.Add(0);
						}
					}
					m_objValue = list2.ToArray();
					break;
				}
				case SDT.ARRAY_16:
				{
					if (newValue.GetType().IsArray)
					{
						m_objValue = newValue;
						break;
					}
					List<ushort> list = new List<ushort>();
					string[] array = text.Split(new char[1] { ',' });
					for (int i = 0; i < array.Length; i++)
					{
						if (ushort.TryParse(array[i], NumberStyles.HexNumber, CultureInfo.InvariantCulture, out var result))
						{
							list.Add(result);
						}
						else
						{
							list.Add(0);
						}
					}
					m_objValue = list.ToArray();
					break;
				}
				case SDT.LOW_LIMIT:
					m_objValue = string.Empty;
					ReadOnly = true;
					break;
				default:
					Logger.Debug("SDT {Name} has unknown datatype {DataType:X}. Id={Id:X} SubId={SubId} Category={Category}", ICUName, DataType, Id, SubId, Category);
					break;
				}
			}
			catch (Exception ex)
			{
				Logger.Verbose(ex, ex.Message);
				ValueExceptionEventArgs e = new ValueExceptionEventArgs
				{
					Exception = ex,
					InvalidValue = text
				};
				ExceptionOccured?.Invoke(this, e);
				return;
			}
			if (m_objValue != null)
			{
				IsChanged = !m_objValue.Equals(DeviceValue);
				if (!m_objValue.Equals(objValue) & invokeChange)
				{
					ValueChanged?.Invoke(this, EventArgs.Empty);
				}
			}
			else
			{
				IsChanged = false;
			}
		}

		public void SetDirty()
		{
			IsChanged = true;
		}

		public ICUProperty(EDSParameter param)
		{
			Initialize(param);
		}

		public ICUProperty(SDT dataType, ushort id, byte subId = 0)
		{
			DataType = dataType;
			Id = id;
			SubId = subId;
			IsChanged = false;
		}

		private void Initialize(EDSParameter param)
		{
			Parameter = param;
			DataType = (SDT)param.DataType;
			Id = (ushort)param.Id;
			SubId = (byte)param.SubId;
			ICUName = "OD_" + param.Name;
			Title = param.Title;
			Units = param.Units;
			IsChanged = false;
		}

		public override string ToString()
		{
			if (Value != null)
			{
				return Value.ToString();
			}
			return string.Empty;
		}

		public void SetInitialValue(object objInitialValue, bool notifyChanges = true)
		{
			SetValue(objInitialValue, invokeChange: false);
			DeviceValue = m_objValue;
			IsChanged = false;
			if (notifyChanges)
			{
				ValueChanged?.Invoke(this, EventArgs.Empty);
			}
		}

		public void CommitChange()
		{
			DeviceValue = m_objValue;
			IsChanged = false;
			ValueChanged?.Invoke(this, EventArgs.Empty);
		}

		public void Rollback()
		{
			SetValue(DeviceValue);
		}

		public void FireChanged()
		{
			ValueChanged?.Invoke(this, EventArgs.Empty);
		}
	}
	public class ValueExceptionEventArgs : EventArgs
	{
		public Exception Exception { get; set; }

		public string InvalidValue { get; set; }
	}
	public class ICUPropertyDictionary : List<ICUProperty>
	{
		public ICUPropertyDictionary()
		{
			foreach (EDSParameter parameter in DataSheet.Parameters)
			{
				Add(new ICUProperty(parameter));
			}
		}

		public ICUProperty GetProperty(ushort propId, byte subId = 0)
		{
			return this.FirstOrDefault((ICUProperty a) => a.Id == propId && a.SubId == subId);
		}

		public ICUProperty GetProperty(string id_sub)
		{
			string[] array = id_sub.Split(new char[1] { '_' });
			if (array.Length > 1)
			{
				ushort propId = 0;
				byte subId = 0;
				int result = 0;
				if (int.TryParse(array[0], NumberStyles.HexNumber, null, out result))
				{
					propId = Convert.ToUInt16(result);
				}
				if (int.TryParse(array[1], NumberStyles.HexNumber, null, out result))
				{
					subId = Convert.ToByte(result);
				}
				return GetProperty(propId, subId);
			}
			return null;
		}

		public ICUProperty AddProperty(SDT dataType, ushort propId, byte subId, string title, string description, bool readOnly)
		{
			if (GetProperty(propId, subId) == null)
			{
				ICUProperty iCUProperty = new ICUProperty(dataType, propId, subId);
				iCUProperty.ICUName = description;
				iCUProperty.Title = title;
				iCUProperty.ReadOnly = readOnly;
				Add(iCUProperty);
				return iCUProperty;
			}
			return null;
		}

		public string GetText(ushort propId)
		{
			ICUProperty iCUProperty = this.FirstOrDefault((ICUProperty a) => a.Id == propId);
			if (iCUProperty != null && iCUProperty.Value != null)
			{
				return iCUProperty.Value.ToString();
			}
			return "<unknown>";
		}
	}
	public enum ICUTransactionType
	{
		Unknown,
		Transaction,
		Reservation,
		MeterValue,
		StatusNotification,
		StartTransaction,
		StopTransaction,
		DateTimeOffset,
		ReservationStatus,
		SecurityEvent
	}
	public enum ICUTransactionStopReason
	{
		Other = 0,
		Local = 1,
		EmergencyStop = 2,
		EVDisconnected = 3,
		Hardreset = 4,
		Powerloss = 5,
		Reboot = 6,
		Remote = 7,
		Softreset = 8,
		UnlockCommand = 9,
		Deauthorized = 10,
		None = 255
	}
	public enum ICUTransactionSecurityEvent
	{
		None,
		FirmwareUpdated,
		AuthenticationFailedAtCSMS,
		CSMSFailedToAuthenticate,
		SetSystemTime,
		StartUpDevice,
		ResetOrReboot,
		SecurityLogCleared,
		ReconfigOfParameter,
		MemoryExhaustion,
		InvalidMessages,
		ReplayAttack,
		TamperDetection,
		FirmwareSignature,
		FirmwareSignCertificate,
		CSMSCertificate,
		CPCertificate,
		TLSVersion,
		TLSCipherSuite
	}
	public enum ICUTransactionReservationStatus
	{
		Unknown = -1,
		Expired,
		Removed
	}
	public enum ICUTransactionTriggerReason
	{
		Autohorized = 0,
		CablePluggedIn = 1,
		ChargingRateChanged = 2,
		ChargingStateChanged = 3,
		Deauthorized = 4,
		EnergyLimitReached = 5,
		EVCommunicationLost = 6,
		EVConnectTimeout = 7,
		MeterValueClock = 8,
		MeterValuePeriodic = 9,
		TimeLimitReached = 10,
		Trigger = 11,
		UnlockCommand = 12,
		StopAuthorized = 13,
		EVDeparted = 14,
		EVDetected = 15,
		RemoteStop = 16,
		RemoteStart = 17,
		AbnormalCondition = 18,
		SignedDataReceived = 19,
		ResetCommand = 20,
		None = 31
	}
	public enum ICUTransactionChargingState
	{
		None,
		Charging,
		EVConnected,
		SuspendedEV,
		SuspendedEVSE,
		Idle
	}
	public class ICUTransactionItem
	{
		private readonly ILogger Logger = Log.ForContext<ICUTransactionItem>();

		public long Offset { get; set; }

		public string FullText { get; set; }

		public uint Socket { get; set; }

		public ulong TransactionId { get; set; }

		public DateTime? StartTime { get; set; }

		public DateTime? StopTime { get; set; } = DateTime.MinValue;

		public string StartTag { get; set; }

		public string StopTag { get; set; }

		public ICUTransactionStopReason StopReason { get; set; } = ICUTransactionStopReason.None;

		public ICUTransactionTriggerReason TriggerReason { get; set; } = ICUTransactionTriggerReason.None;

		public ICUTransactionChargingState ChargingState { get; set; }

		public double? StartMeterValue { get; set; }

		public double? StopMeterValue { get; set; }

		public bool StartToSend { get; set; }

		public bool StopToSend { get; set; }

		public ICUTransactionType Type { get; set; }

		public string ExtraInfo { get; set; } = string.Empty;

		public ICUTransactionSecurityEvent SecurityEvent { get; set; }

		public ICUTransactionReservationStatus ReservationStatus { get; set; } = ICUTransactionReservationStatus.Unknown;

		public ICUTransactionItem()
		{
		}

		public ICUTransactionItem(string tx)
		{
			try
			{
				string[] array = tx.Split(new char[1] { '_' });
				if (array.Count() > 1)
				{
					if (long.TryParse(array[0], out var result))
					{
						Offset = result;
					}
					FullText = array[1];
					string text = array[1];
					if (text.StartsWith("tx:"))
					{
						Type = ICUTransactionType.Transaction;
						string[] array2 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
						if (array2.Count() < 3)
						{
							return;
						}
						string text2 = array2[0].Trim();
						if (text2.StartsWith("id"))
						{
							TransactionId = ulong.Parse(text2.Substring(5), NumberStyles.HexNumber);
						}
						string text3 = array2[1].Trim();
						if (text3.StartsWith("socket"))
						{
							Socket = uint.Parse(text3.Substring(6));
						}
						string text4 = array2[2].Trim();
						string s = text4.Substring(0, 19);
						string[] array3 = text4.Split(new char[1] { ' ' });
						if (array3.Count() == 5)
						{
							StartTime = DateTime.ParseExact(s, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StartMeterValue = double.Parse(array3[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StartTag = array3[3].Trim();
							StartToSend = array3[4].Trim() == "Y";
						}
						if (array2.Count() >= 4)
						{
							string text5 = array2[3].Trim();
							string s2 = text5.Substring(0, 19);
							string[] array4 = text5.Split(new char[1] { ' ' });
							if (array4.Count() == 5)
							{
								StopTime = DateTime.ParseExact(s2, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
								StopMeterValue = double.Parse(array4[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
								StopTag = array4[3].Trim();
								StopToSend = array4[4].Trim() == "Y";
							}
							else if (array4.Count() == 6)
							{
								StopTime = DateTime.ParseExact(s2, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
								StopMeterValue = double.Parse(array4[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
								StopTag = array4[3].Trim();
								StopReason = (ICUTransactionStopReason)int.Parse(array4[4].Trim(), CultureInfo.InvariantCulture);
								StopToSend = array4[5].Trim() == "Y";
							}
						}
					}
					else if (text.StartsWith("rs:"))
					{
						Type = ICUTransactionType.Reservation;
						string[] array5 = text.Substring(3).Trim().Split(new char[1] { ' ' });
						if (array5.Count() >= 8)
						{
							if (array5[0].StartsWith("#"))
							{
								TransactionId = ulong.Parse(array5[0].Substring(1));
							}
							StartTag = array5[2].Trim();
							Socket = uint.Parse(array5[5].Trim(new char[1] { ',' }));
							StartTime = DateTime.ParseExact(array5[7] + " " + array5[8], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						}
					}
					else if (text.StartsWith("sn:") || text.StartsWith("sn3:"))
					{
						Type = ICUTransactionType.StatusNotification;
						string[] array6 = text.Substring(text.IndexOf(":") + 1).Trim().Split(new char[1] { ' ' });
						if (string.Compare(array6[0], ":") == 0)
						{
							array6 = array6.Skip(1).ToArray();
						}
						if (array6.Count() >= 8)
						{
							Socket = uint.Parse(array6[1].Trim(new char[1] { ',' }));
							StartTime = DateTime.ParseExact(array6[2] + " " + array6[3], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							string text6 = "";
							for (int i = 6; i < array6.Count() - 3; i++)
							{
								text6 = text6 + array6[i].Trim(new char[1] { ',' }) + " ";
							}
							ExtraInfo = (array6[4].Trim(new char[1] { ',' }) + " " + array6[5].Trim(new char[1] { ',' }) + " " + text6.Trim()).Trim();
							if (array6.Count() >= 10)
							{
								TriggerReason = (ICUTransactionTriggerReason)int.Parse(array6[array6.Count() - 3].Trim(new char[1] { ',' }).Trim(), CultureInfo.InvariantCulture);
								ChargingState = (ICUTransactionChargingState)int.Parse(array6[array6.Count() - 2].Trim(new char[1] { ',' }).Trim(), CultureInfo.InvariantCulture);
							}
						}
					}
					else if (text.StartsWith("mv:"))
					{
						Type = ICUTransactionType.MeterValue;
						string[] array7 = text.Substring(3).Trim().Split(new char[1] { ' ' });
						if (array7.Count() >= 6)
						{
							Socket = uint.Parse(array7[1].Trim(new char[1] { ',' }));
							StartTime = DateTime.ParseExact(array7[2] + " " + array7[3], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StartMeterValue = double.Parse(array7[4].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StartToSend = array7[array7.Count() - 1].Trim() == "Y";
							if (array7[array7.Count() - 2].ToLower() == "start" || array7[array7.Count() - 2].ToLower() == "stop" || array7[array7.Count() - 2].ToLower() == "regular")
							{
								ExtraInfo = array7[array7.Count() - 2];
							}
						}
					}
					else if (text.StartsWith("txstart:") || text.StartsWith("txstart2:"))
					{
						Type = ICUTransactionType.StartTransaction;
						string[] array8 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
						if (array8.Count() < 3)
						{
							return;
						}
						string text7 = array8[0].Trim();
						if (text7.StartsWith("id"))
						{
							TransactionId = ulong.Parse(text7.Substring(5), NumberStyles.HexNumber);
						}
						string text8 = array8[1].Trim();
						if (text8.StartsWith("socket"))
						{
							Socket = uint.Parse(text8.Substring(6));
						}
						string text9 = array8[2].Trim();
						string s3 = text9.Substring(0, 19);
						string[] array9 = text9.Split(new char[1] { ' ' });
						int num = array9.Count();
						if (num == 6 || num == 7)
						{
							StartTime = DateTime.ParseExact(s3, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StartMeterValue = double.Parse(array9[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StartTag = array9[3].Trim();
							if (num == 6)
							{
								StopReason = (ICUTransactionStopReason)int.Parse(array9[4].Trim(), CultureInfo.InvariantCulture);
								StartToSend = array9[5].Trim() == "Y";
							}
							if (num == 7)
							{
								TriggerReason = (ICUTransactionTriggerReason)int.Parse(array9[4].Trim(), CultureInfo.InvariantCulture);
								ChargingState = (ICUTransactionChargingState)int.Parse(array9[5].Trim(), CultureInfo.InvariantCulture);
								StartToSend = array9[6].Trim() == "Y";
							}
						}
					}
					else if (text.StartsWith("txstop:") || text.StartsWith("txstop2:"))
					{
						Type = ICUTransactionType.StopTransaction;
						string[] array10 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
						if (array10.Count() >= 3)
						{
							string text10 = array10[0].Trim();
							if (text10.StartsWith("id"))
							{
								TransactionId = ulong.Parse(text10.Substring(5), NumberStyles.HexNumber);
							}
							string text11 = array10[1].Trim();
							if (text11.StartsWith("socket"))
							{
								Socket = uint.Parse(text11.Substring(6));
							}
							string text12 = array10[2].Trim();
							string s4 = text12.Substring(0, 19);
							string[] array11 = text12.Split(new char[1] { ' ' });
							if (array11.Count() == 5)
							{
								StopTime = DateTime.ParseExact(s4, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
								StopMeterValue = double.Parse(array11[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
								StopTag = ((array11[3].Trim() != "(null)" && array11[3].Trim() != "") ? array11[3].Trim() : null);
								StopToSend = array11[4].Trim() == "Y";
							}
							if (array11.Count() == 7)
							{
								StopTime = DateTime.ParseExact(s4, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
								StopMeterValue = double.Parse(array11[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
								StopTag = ((array11[3].Trim() != "(null)" && array11[3].Trim() != "") ? array11[3].Trim() : null);
								TriggerReason = (ICUTransactionTriggerReason)int.Parse(array11[4].Trim(), CultureInfo.InvariantCulture);
								ChargingState = (ICUTransactionChargingState)int.Parse(array11[5].Trim(), CultureInfo.InvariantCulture);
								StopToSend = array11[6].Trim() == "Y";
							}
						}
					}
					else if (text.StartsWith("dto:"))
					{
						Type = ICUTransactionType.DateTimeOffset;
						if (int.TryParse(text.Replace("dto:", ""), out var result2))
						{
							TimeSpan timeSpan = TimeSpan.FromSeconds((double)result2);
							ExtraInfo = string.Format("Days: {0} Hours: {1} Minutes: {2} Seconds: {3}", new object[4] { timeSpan.Days, timeSpan.Hours, timeSpan.Minutes, timeSpan.Seconds });
						}
					}
					else if (text.StartsWith("se:"))
					{
						Type = ICUTransactionType.SecurityEvent;
						string[] array12 = text.Substring(4).Split(new char[1] { ' ' });
						if (array12.Count() >= 4)
						{
							StartTime = DateTime.ParseExact(array12[0] + " " + array12[1].Trim(new char[2] { ' ', ':' }), "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							if (int.TryParse(array12[2], out var result3))
							{
								SecurityEvent = (ICUTransactionSecurityEvent)result3;
							}
							StartToSend = array12[3].Trim(new char[3] { ' ', '(', ')' }) == "Y";
						}
					}
					else if (text.StartsWith("rss:"))
					{
						Type = ICUTransactionType.ReservationStatus;
						string[] array13 = text.Substring(5).Split(new char[1] { ' ' });
						if (array13.Count() >= 3)
						{
							if (ulong.TryParse(array13[0].Trim(new char[2] { ' ', ':' }), out var result4))
							{
								TransactionId = result4;
							}
							if (int.TryParse(array13[1], out var result5))
							{
								ReservationStatus = (ICUTransactionReservationStatus)result5;
							}
							StartToSend = array13[2].Trim(new char[3] { ' ', '(', ')' }) == "Y";
						}
					}
					else
					{
						Logger.Debug("Transaction part \"{Transaction}\" is not recognized", text);
					}
				}
				else
				{
					Logger.Debug("Line: \"{Tx}\" cannot be parsed", tx);
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "Error parsing line \"{Tx}\" with message \"{Message}\"", tx, ex.Message);
			}
		}

		public override string ToString()
		{
			return FullText;
		}
	}
	public class ICUTransactions
	{
		private readonly ILogger Logger = Log.ForContext<ICUTransactions>();

		private ICULanDevice m_device;

		private const int s_nRequestLongTimeOut = 10000;

		public ObservableCollection<ICUTransactionItem> Transactions { get; private set; } = new ObservableCollection<ICUTransactionItem>();

		public ICUTransactions(ICULanDevice device)
		{
			m_device = device;
			Transactions.Clear();
		}

		~ICUTransactions()
		{
			m_device = null;
			Transactions.Clear();
		}

		public bool Clear()
		{
			if (m_device == null)
			{
				return false;
			}
			bool result = m_device.EraseTransactionDatabase(10000);
			Transactions.Clear();
			return result;
		}

		private string[] unwrapTransActions(string rawTransactions)
		{
			if (string.IsNullOrEmpty(rawTransactions))
			{
				return new string[0];
			}
			rawTransactions = Regex.Replace(rawTransactions, "{\"version\":\\d+,", "");
			rawTransactions = Regex.Replace(rawTransactions, "AP;.*?; ", "");
			rawTransactions = Regex.Replace(rawTransactions, "\\(\\s*null\\s*\\) ", "");
			rawTransactions = rawTransactions.Replace("}", "");
			return rawTransactions.Split(new char[1] { '\n' });
		}

		public bool Read()
		{
			if (m_device == null)
			{
				return false;
			}
			Transactions = new ObservableCollection<ICUTransactionItem>();
			bool flag = false;
			uint num = uint.MaxValue;
			int num2;
			do
			{
				num2 = 0;
				try
				{
					(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("transactions", $"offset={num}", null, 10000, 1);
					if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
					{
						Logger.Debug("Transaction API web request failed {RequestOffset}", num);
						flag = true;
						break;
					}
					string[] array = unwrapTransActions(tuple.Item3);
					if (array.Count() == 0 || array[0].ToLower().Contains("empty transaction database"))
					{
						break;
					}
					string[] array2 = array;
					foreach (string text in array2)
					{
						if (!string.IsNullOrEmpty(text.Trim()))
						{
							ICUTransactionItem newItem = new ICUTransactionItem(text);
							if (Transactions.Count((ICUTransactionItem tx) => tx.Offset == newItem.Offset) <= 0)
							{
								num2++;
								Transactions.Add(newItem);
								num = (uint)newItem.Offset;
							}
						}
					}
					continue;
				}
				catch (Exception ex)
				{
					Logger.Debug(ex, ex.Message);
					flag = true;
				}
				break;
			}
			while (num2 > 0);
			return !flag;
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
			foreach (ICUTransactionItem transaction in Transactions)
			{
				list.Add(transaction.FullText);
			}
			File.WriteAllLines(fileName, list.ToArray());
			return true;
		}
	}
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
	public class WhiteListArgs : EventArgs
	{
		public ICULanDevice Device { get; private set; }

		public ObservableCollection<ICUWhitelistItem> Whitelist { get; private set; }

		public bool CancellationPending { get; private set; }

		public WhiteListArgs(ICULanDevice lanDevice, ObservableCollection<ICUWhitelistItem> whitelist, bool cancellationPending = false)
		{
			Device = lanDevice;
			Whitelist = new ObservableCollection<ICUWhitelistItem>(whitelist);
			CancellationPending = cancellationPending;
		}
	}
	public enum ICUTagStatus
	{
		Unknown = 0,
		Active = 1,
		Blocked = 2,
		Deleted = 3,
		MasterCard = 99
	}
	public class ICUWhitelistItem
	{
		private readonly ILogger Logger = Log.ForContext<ICUWhitelistItem>();

		public string Tag { get; set; }

		public string ParentTag { get; set; }

		public ICUTagStatus Status { get; set; }

		public bool HasExpiryDate { get; set; }

		public DateTime ExpiryDate { get; set; }

		public string Expiry
		{
			get
			{
				if (HasExpiryDate)
				{
					return ExpiryDate.ToString();
				}
				return "<no expiry date>";
			}
		}

		public string ExpireDate
		{
			get
			{
				if (HasExpiryDate)
				{
					return ExpiryDate.ToShortDateString();
				}
				return "<no expiry date>";
			}
		}

		public ICUWhitelistItem()
		{
			Tag = "";
			ParentTag = "";
			Status = ICUTagStatus.Unknown;
		}

		public ICUWhitelistItem(dynamic item)
		{
			Tag = "";
			ParentTag = "";
			Status = ICUTagStatus.Unknown;
			try
			{
				Tag = Convert.ToString(item["tag"]);
				ParentTag = Convert.ToString(item["parent"]);
				int num = Convert.ToInt32(item["status"]);
				if (num >= 0 && num <= 3)
				{
					Status = (ICUTagStatus)num;
				}
				string s = Convert.ToString(item["expiryDate"]);
				HasExpiryDate = false;
				if (long.TryParse(s, out var result) && result != 0L)
				{
					DateTime unixEpoch = ICUDevice.UnixEpoch;
					ExpiryDate = unixEpoch.AddSeconds(result).ToLocalTime();
					HasExpiryDate = true;
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
		}

		public override string ToString()
		{
			return Tag;
		}
	}
	public class DeviceEventArgs : EventArgs
	{
		public ICUDevice Device { get; set; }

		public DeviceEventArgs(ICUDevice device)
		{
			Device = device;
		}
	}
	public class LANConnection : BaseConnection
	{
		public delegate void DeviceEventHandler(object sender, DeviceEventArgs e);

		public delegate void LoginEventHandler(ICULanDevice lanDevice, ACEWebLoginData loginData);

		private readonly ILogger Logger = Log.Logger.ForContext<LANConnection>();

		private readonly int DEFAULT_LOCK_TIMEOUT = 5000;

		private readonly Lazy<ServiceBrowser> m_serviceBrowserLazy;

		private List<ICULanDevice> m_doNotRemoveDevices = new List<ICULanDevice>();

		private readonly List<string> serviceTypes = new List<string> { "_lolo3._http._tcp", "_alfen._tcp" };

		private ServiceBrowser m_serviceBrowser => m_serviceBrowserLazy.Value;

		public bool DeviceFound { get; }

		public event DeviceEventHandler DeviceRegistered;

		public event DeviceEventHandler DeviceReRegistered;

		public event DeviceEventHandler DeviceUnregistered;

		public event LoginEventHandler LoginRequest;

		public event Action<string> ErrorHandler;

		public LANConnection(ObservableCollection<ICUDevice> lstDevices)
			: base(lstDevices)
		{
			m_sName = "Ethernet";
			m_serviceBrowserLazy = new Lazy<ServiceBrowser>(() => new ServiceBrowser());
		}

		public void StartSearch()
		{
			Logger.Verbose("Start browsing for types: {ServiceTypes}", serviceTypes);
			lock (m_serviceBrowser)
			{
				m_serviceBrowser.QueryParameters.Robustness = 10;
				m_serviceBrowser.QueryParameters.ResponseTime = 2000;
				m_serviceBrowser.ServiceAdded += OnServiceAdded;
				m_serviceBrowser.ServiceChanged += OnServiceAdded;
				m_serviceBrowser.ServiceRemoved += OnServiceRemoved;
				m_serviceBrowser.NetworkInterfaceAdded += OnNetworkInterfaceAdded;
				m_serviceBrowser.NetworkInterfaceRemoved += OnNetworkInterfaceRemoved;
				if (!m_serviceBrowser.IsBrowsing)
				{
					m_serviceBrowser.StartBrowse(serviceTypes);
				}
			}
		}

		public void StopSearch()
		{
			Logger.Verbose("Stop browsing for types: {ServiceTypes}", serviceTypes);
			lock (m_serviceBrowser)
			{
				m_serviceBrowser.ServiceAdded -= OnServiceAdded;
				m_serviceBrowser.ServiceChanged -= OnServiceAdded;
				m_serviceBrowser.ServiceRemoved -= OnServiceRemoved;
				m_serviceBrowser.NetworkInterfaceAdded -= OnNetworkInterfaceAdded;
				m_serviceBrowser.NetworkInterfaceRemoved -= OnNetworkInterfaceRemoved;
				if (m_serviceBrowser.IsBrowsing)
				{
					m_serviceBrowser.StopBrowse();
				}
			}
			Devices.Clear();
			lock (NetworkInterfaces)
			{
				NetworkInterfaces.Clear();
			}
		}

		public void CallErrorHandler(string msg, ICULanDevice device)
		{
			if (Devices.OfType<ICULanDevice>().ToList().Exists((ICULanDevice a) => a.HostName != null && a.HostName == device.HostName))
			{
				Logger.Error("Error: {Message} (HostName={HostName}, IP={IPAddress})", msg, device.HostName, device.IPAddress);
				ErrorHandler?.Invoke(msg);
			}
		}

		public void DeviceLoginCallback(ICULanDevice lanDevice, ACEWebLoginData loginData)
		{
			LoginRequest?.Invoke(lanDevice, loginData);
		}

		private void OnServiceRemoved(object sender, ServiceAnnouncementEventArgs e)
		{
			if (!(sender is ServiceBrowser) || !e.Announcement.Addresses.Any() || !Monitor.TryEnter(Devices, DEFAULT_LOCK_TIMEOUT))
			{
				return;
			}
			try
			{
				string propertyValue = e.Announcement.Addresses.First().ToString();
				if (m_doNotRemoveDevices.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == e.Announcement.Hostname) == null)
				{
					ICULanDevice iCULanDevice = Devices.Cast<ICULanDevice>().ToList().FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == e.Announcement.Hostname);
					if (iCULanDevice != null && iCULanDevice.OnDeviceRemoved())
					{
						Logger.Debug("Device removed: {DeviceName} } at {IpAddress}, SCN: {SCNNetwork}", e.Announcement.Hostname.Split(new char[1] { '-' }).Last(), propertyValue, iCULanDevice.SCNNetwork);
						DeviceUnregistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
						Devices.Remove(iCULanDevice);
					}
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
			finally
			{
				Monitor.Exit(Devices);
			}
		}

		private void OnServiceAdded(object sender, ServiceAnnouncementEventArgs e)
		{
			if (!(sender is ServiceBrowser) || !e.Announcement.Addresses.Any() || !Monitor.TryEnter(Devices, DEFAULT_LOCK_TIMEOUT))
			{
				return;
			}
			try
			{
				string text = e.Announcement.Addresses.First().ToString();
				List<ICULanDevice> source = Devices.Cast<ICULanDevice>().ToList();
				string newHostName = e.Announcement.Hostname;
				ICULanDevice iCULanDevice = source.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == newHostName);
				if (iCULanDevice == null)
				{
					string objectID = newHostName.Split(new char[1] { '-' }).Last();
					iCULanDevice = source.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName.Split(new char[1] { '-' }).Last() == objectID);
				}
				if (iCULanDevice == null)
				{
					iCULanDevice = new ICULanDevice(this, e.Announcement.Addresses.First(), e.Announcement.Port, e.Announcement);
					iCULanDevice.RegisterLoginCallback(DeviceLoginCallback);
					Devices.Add(iCULanDevice);
					string text2 = e.Announcement.Hostname.Split(new char[1] { '-' })[^1];
					Logger.Debug("Added Device: {SerialNumber} at {IpAddress} SCN: '{SCNNetwork}' using: {Protocol}", text2, text, iCULanDevice.SCNNetwork, iCULanDevice.Protocol);
					Logger.Information("Discovered device: {SerialNumber:X} in SCN: {SCN:X}", text2.GetHashCode(), (!string.IsNullOrEmpty(iCULanDevice.SCNNetwork)) ? iCULanDevice.SCNNetwork.GetHashCode() : 0);
					DeviceRegistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
				}
				else
				{
					iCULanDevice.ReInitialize(e.Announcement.Addresses.First(), e.Announcement.Port, e.Announcement, newDevice: false);
					iCULanDevice.RegisterLoginCallback(DeviceLoginCallback);
					Logger.Debug("Reinitialized Device: {DeviceName} at {IpAddress} SCN: '{SCNNetwork}' using: {Protocol}", e.Announcement.Hostname.Split(new char[1] { '-' }).Last(), text, iCULanDevice.SCNNetwork, iCULanDevice.Protocol);
					DeviceReRegistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
			finally
			{
				Monitor.Exit(Devices);
			}
		}

		public override void StopBrowsing()
		{
			lock (m_serviceBrowser)
			{
				m_serviceBrowser.StopBrowse();
			}
		}

		public override void StartBrowsing()
		{
			lock (m_serviceBrowser)
			{
				try
				{
					m_doNotRemoveDevices = Devices.Cast<ICULanDevice>().ToList();
					if (m_serviceBrowser.IsBrowsing)
					{
						m_serviceBrowser.StopBrowse();
					}
					m_serviceBrowser.StartBrowse(serviceTypes);
				}
				catch (Exception ex)
				{
					Logger.Error(ex, ex.Message);
				}
			}
		}

		public ICULanDevice AddManualDevice(IPAddress address, int port, string hostName = "", int numberOfSockets = 2, bool LoginRequired = false)
		{
			foreach (ICUDevice device in Devices)
			{
				if (device is ICULanDevice iCULanDevice && iCULanDevice.IPAddress == address && iCULanDevice.Port == port)
				{
					return iCULanDevice;
				}
			}
			ICULanDevice iCULanDevice2 = new ICULanDevice(this, address, port, null, isManuallyAdded: true);
			if (!string.IsNullOrEmpty(hostName))
			{
				iCULanDevice2.SetHostInfo(hostName, numberOfSockets);
			}
			if (LoginRequired)
			{
				iCULanDevice2.IsUniquePasswordRequired = true;
			}
			iCULanDevice2.RegisterLoginCallback(DeviceLoginCallback);
			if (iCULanDevice2.Login().IsLoggedIn)
			{
				iCULanDevice2.UpdateCategories("generic", "generic2");
				iCULanDevice2.NumberOfSockets = iCULanDevice2.GetPropertyInt(8286, 0);
				iCULanDevice2.SocketTypes[0] = iCULanDevice2.GetPropertyInt(8485, 0);
				iCULanDevice2.SocketTypes[1] = ((iCULanDevice2.NumberOfSockets > 1) ? iCULanDevice2.GetPropertyInt(12581, 0) : 0);
				EMeterTypes propertyInt = (EMeterTypes)iCULanDevice2.GetPropertyInt(16919, 0);
				EMeterTypes propertyInt2 = (EMeterTypes)iCULanDevice2.GetPropertyInt(21015, 0);
				iCULanDevice2.HasCentralMeter = propertyInt == EMeterTypes.ENERGYMETER_MODBUS_CENTRAL || propertyInt == EMeterTypes.ENERGYMETER_P1 || propertyInt == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL || propertyInt == EMeterTypes.ENERGYMETER_FKN_METER;
				iCULanDevice2.HasSmartMeter = propertyInt2 == EMeterTypes.ENERGYMETER_P1 || propertyInt2 == EMeterTypes.ENERGYMETER_TCPIP_SMART;
				Logger.Debug("New device firmware: {FirmwareVersionNumber}", iCULanDevice2.FirmwareVersionNumber);
				Devices.Add(iCULanDevice2);
				return iCULanDevice2;
			}
			return null;
		}

		public void RemoveManualDevice(ICUDevice device)
		{
			if (device != null)
			{
				if (device is ICULanDevice iCULanDevice)
				{
					iCULanDevice.Deallocate();
				}
				Devices.Remove(device);
			}
		}

		public void RemoveDevice(ICULanDevice device)
		{
			if (device.OnDeviceRemoved())
			{
				Logger.Debug("Device removed: {DeviceName} at {IpAddress}, SCN: {SCNNetwork}", device.Identification, device.Address, device.SCNNetwork);
				DeviceUnregistered(this, new DeviceEventArgs(device));
				device.Deallocate();
				Devices.Remove(device);
			}
		}

		public ICULanDevice FindLanDevice(IPAddress ipAddress)
		{
			return (Devices?.Cast<ICULanDevice>()).FirstOrDefault((ICULanDevice device) => device.Address.ToLowerInvariant() == ipAddress.ToString().ToLowerInvariant());
		}

		public List<ICULanDevice> FindDevicesInSCN(string scnName)
		{
			return Devices?.Cast<ICULanDevice>().Where((ICULanDevice a) => a.SCNNetwork?.ToLowerInvariant() == scnName.ToString().ToLowerInvariant()).ToList();
		}

		private void OnNetworkInterfaceAdded(object sender, NetworkInterfaceEventArgs args)
		{
			if (!(sender is ServiceBrowser) || NetworkInterfaces == null)
			{
				return;
			}
			lock (NetworkInterfaces)
			{
				if (NetworkInterfaces.Count > 0)
				{
					NetworkInterface networkInterface = NetworkInterfaces.ToList().FirstOrDefault((NetworkInterface a) => a.Id == args.NetworkInterface.Id);
					if (networkInterface != null)
					{
						NetworkInterfaces.Remove(networkInterface);
					}
				}
				NetworkInterfaces.Add(args.NetworkInterface);
			}
		}

		private void OnNetworkInterfaceRemoved(object sender, NetworkInterfaceEventArgs args)
		{
			if (!(sender is ServiceBrowser) || NetworkInterfaces == null)
			{
				return;
			}
			lock (NetworkInterfaces)
			{
				NetworkInterfaces.Remove(args.NetworkInterface);
			}
		}
	}
	public class LogHttpClientHandler(HttpMessageHandler innerHandler) : DelegatingHandler(innerHandler)
	{
		private readonly ILogger Logger = Log.ForContext<LogHttpClientHandler>();

		protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
		{
			string value = Guid.NewGuid().ToString().Substring(0, 5);
			using (LogContext.PushProperty("CorrelationId", value))
			{
				using (LogContext.PushProperty("IpAddress", request.RequestUri.Host))
				{
					if (request.Content is MultipartFormDataContent)
					{
						return await base.SendAsync(request, cancellationToken);
					}
					if ((!Logger.IsEnabled(LogEventLevel.Verbose) && request.RequestUri.ToString().Contains("/prop?ids=2059_0,2060_0")) || request.RequestUri.ToString().Contains("/prop?ids=2191_2,2191_3") || request.RequestUri.ToString().Contains("/prop?ids=2210_0,2211_0,2212_0"))
					{
						return await base.SendAsync(request, cancellationToken);
					}
					Stopwatch stopwatch = Stopwatch.StartNew();
					string requestHeaders = string.Join(", ", request.Headers.Select((KeyValuePair<string, IEnumerable<string>> h) => "\"" + h.Key + "\": \"" + string.Join(", ", h.Value) + "\""));
					string text = ((request.Content == null) ? "{}" : (await request.Content.ReadAsStringAsync()));
					string requestContent = text;
					string command = string.Empty;
					if (request.RequestUri.ToString().Contains("/cmd"))
					{
						command = JsonConvert.DeserializeAnonymousType(requestContent, new
						{
							command = string.Empty
						}).command;
					}
					HttpResponseMessage response = null;
					try
					{
						requestContent = ((!request.RequestUri.ToString().Contains("/login")) ? requestContent : "<masked>");
						if (!Logger.IsEnabled(LogEventLevel.Verbose))
						{
							requestHeaders = $"Length={requestHeaders.Length}";
							requestContent = $"Length={requestContent.Length}";
						}
						Logger.Debug("-> {Uri} {Command} {Request}", request.RequestUri, (!string.IsNullOrEmpty(command)) ? (command ?? "") : "", new
						{
							Method = request.Method,
							RequestUri = request.RequestUri,
							Headers = requestHeaders,
							Content = requestContent
						});
						_ = string.Empty;
						_ = string.Empty;
						response = await base.SendAsync(request, cancellationToken);
						stopwatch.Stop();
						text = ((response.Content == null) ? string.Empty : (await response.Content.ReadAsStringAsync()));
						string text2 = text;
						int length = text2.Length;
						string text3 = string.Join(", ", response.Headers.Select((KeyValuePair<string, IEnumerable<string>> h) => "\"" + h.Key + "\": \"" + string.Join(", ", h.Value) + "\""));
						if (!Logger.IsEnabled(LogEventLevel.Verbose) || request.RequestUri.ToString().Contains("/api/log"))
						{
							text3 = $"Length={text3.Length}";
							text2 = "...";
						}
						Logger.Debug("<- {StatusCode} {Uri} {Length:0.0}kB {ElapsedMilliseconds}ms {Command} {Response}", response.StatusCode, request.RequestUri, (double)length / 1024.0, stopwatch.ElapsedMilliseconds, (!string.IsNullOrEmpty(command)) ? (command + " ") : "", new
						{
							HttpStatus = (int)response.StatusCode + ": " + response.ReasonPhrase.Trim(),
							Headers = text3,
							Content = text2
						});
						return response;
					}
					catch (Exception ex)
					{
						stopwatch.Stop();
						Logger.Debug("<- {StatusCode} {Uri} {ElapsedMilliseconds}ms {Command} {Request} {ExceptionDetails}", response?.StatusCode.ToString() ?? "NOK", request.RequestUri, stopwatch.ElapsedMilliseconds, (!string.IsNullOrEmpty(command)) ? (command + " ") : "", new
						{
							Method = request.Method,
							RequestUri = request.RequestUri,
							Headers = requestHeaders,
							Content = requestContent
						}, new { ex.Message, ex.StackTrace });
						throw;
					}
				}
			}
		}
	}
	public class SCNNetwork
	{
		private readonly ILogger Logger = Log.ForContext<SCNNetwork>();

		public static int UDPPORT = 36549;

		private readonly TimeSpan m_16SecondsTimeSpan = new TimeSpan(0, 0, 16);

		private const int udpMaxBufferLength = 1024;

		private const int udpMinPacketLength = 96;

		private static readonly byte[] s_aesKey = new byte[16]
		{
			222, 11, 77, 232, 113, 88, 244, 239, 138, 139,
			108, 36, 96, 130, 116, 112
		};

		private ICryptoTransform m_decryptor;

		private readonly ConcurrentDictionary<string, SCNSocket> Sockets = new ConcurrentDictionary<string, SCNSocket>();

		private static readonly Lazy<SCNNetwork> lazySCNNetwork = new Lazy<SCNNetwork>(() => new SCNNetwork());

		private readonly BlockingCollection<UdpReceiveResult> _udpPackets = new BlockingCollection<UdpReceiveResult>();

		private static readonly object _lock = new object();

		private CancellationTokenSource m_cancellationTokenSource;

		private long totalUdpPacketsReceived;

		private long totalUdpPacketsProcessed;

		public static SCNNetwork Instance => lazySCNNetwork.Value;

		private SCNNetwork()
		{
		}

		private static string GetSocketKey(SCNSocket socket)
		{
			return $"{socket.UniqueID}_{socket.SocketIndex}_{socket.IPAddress}";
		}

		private static string GenerateSocketKey(ulong uniqueId, int socketIndex, IPAddress ipAddress)
		{
			return $"{uniqueId}_{socketIndex}_{ipAddress}";
		}

		private void InitializeAESDecryptor()
		{
			byte[] rgbIV = new byte[16];
			using RijndaelManaged rijndaelManaged = new RijndaelManaged
			{
				Padding = PaddingMode.None
			};
			m_decryptor = rijndaelManaged.CreateDecryptor(s_aesKey, rgbIV);
		}

		private void AESDecrypt(ref byte[] binDataDecrypted, byte[] dataToDecrypt)
		{
			using MemoryStream stream = new MemoryStream(dataToDecrypt);
			using CryptoStream cryptoStream = new CryptoStream(stream, m_decryptor, CryptoStreamMode.Read);
			cryptoStream.Read(binDataDecrypted, 0, binDataDecrypted.Length);
		}

		private void StartUdpReceiveTask(CancellationToken cancellationToken)
		{
			Logger.Debug("Start SCN UDP packet listening task");
			Task.Run(async () =>
			{
				using (UdpClient udpClient = new UdpClient
				{
					ExclusiveAddressUse = false
				})
				{
					udpClient.Client.SetSocketOption(SocketOptionLevel.Socket, SocketOptionName.ReuseAddress, optionValue: true);
					IPEndPoint localEP = new IPEndPoint(IPAddress.Any, UDPPORT);
					udpClient.Client.Bind(localEP);
					InitializeAESDecryptor();
					Stopwatch stopwatchUdpPacketsReceived = new Stopwatch();
					stopwatchUdpPacketsReceived.Start();
					int udpPacketsReceived = 0;
					while (!cancellationToken.IsCancellationRequested)
					{
						try
						{
							UdpReceiveResult item = await udpClient.ReceiveAsync();
							_udpPackets.Add(item);
							udpPacketsReceived++;
							totalUdpPacketsReceived++;
						}
						catch (Exception ex)
						{
							Logger.Debug(ex, ex.Message);
						}
						if (totalUdpPacketsReceived % 250 == 0L)
						{
							double totalSeconds = stopwatchUdpPacketsReceived.Elapsed.TotalSeconds;
							Logger.Debug("Received {Packets} packets in {TimeInSeconds:F3}s, {PacketsReceivedPerSecond:F0} packets/s, total={TotalPackets}", udpPacketsReceived, totalSeconds, (double)udpPacketsReceived / totalSeconds, totalUdpPacketsReceived);
							stopwatchUdpPacketsReceived.Reset();
							udpPacketsReceived = 0;
							stopwatchUdpPacketsReceived.Start();
						}
					}
				}
				Logger.Debug("Finished SCN UDP listening task");
			}, m_cancellationTokenSource.Token);
		}

		public void StartUdpProcessTask(CancellationToken cancellationToken)
		{
			Logger.Debug("Starting SCN UDP packet processing task");
			Task.Run(() =>
			{
				try
				{
					Stopwatch stopwatch = new Stopwatch();
					stopwatch.Start();
					int num = 0;
					foreach (UdpReceiveResult item in _udpPackets.GetConsumingEnumerable(cancellationToken))
					{
						try
						{
							ProcessUdpPacket(item);
							num++;
							totalUdpPacketsProcessed++;
						}
						catch (Exception ex)
						{
							Logger.Debug(ex, ex.Message);
						}
						if (totalUdpPacketsProcessed % 250 == 0L)
						{
							double totalSeconds = stopwatch.Elapsed.TotalSeconds;
							Logger.Debug("Processed {Packets} packets in {TimeInSeconds:F3}s, {PacketsProcessedPerSecond:F0} packets/s, total={TotalPackets}", num, totalSeconds, (double)num / totalSeconds, totalUdpPacketsProcessed);
							stopwatch.Reset();
							num = 0;
							stopwatch.Start();
						}
					}
				}
				catch (OperationCanceledException)
				{
					Logger.Debug("Finished SCN UDP packet processing task");
				}
			});
		}

		private void ProcessUdpPacket(UdpReceiveResult udpPacket)
		{
			DateTime now = DateTime.Now;
			Stopwatch.StartNew();
			if (udpPacket.Buffer.Length >= 1024)
			{
				return;
			}
			byte[] binDataDecrypted = new byte[1024];
			AESDecrypt(ref binDataDecrypted, udpPacket.Buffer);
			if (binDataDecrypted.Length < 96)
			{
				return;
			}
			int item = binDataDecrypted[0];
			if (!SCNSocket.ValidLibraryVersions.Contains(item) || binDataDecrypted.Length < 96)
			{
				return;
			}
			int socketIndex = binDataDecrypted[23];
			ulong uniqueId = BitConverter.ToUInt64(binDataDecrypted, 48);
			try
			{
				string socketKey = GenerateSocketKey(uniqueId, socketIndex, udpPacket.RemoteEndPoint.Address);
				Sockets.GetOrAdd(socketKey, (string key) =>
				{
					Logger.Debug("Socket added: {SocketKey}", socketKey);
					return new SCNSocket();
				}).ParseData(binDataDecrypted, now, udpPacket.RemoteEndPoint.Address);
			}
			catch (Exception ex)
			{
				Logger.Debug(ex, ex.Message);
			}
		}

		public void Start()
		{
			lock (_lock)
			{
				m_cancellationTokenSource?.Dispose();
				m_cancellationTokenSource = new CancellationTokenSource();
				StartUdpReceiveTask(m_cancellationTokenSource.Token);
				StartUdpProcessTask(m_cancellationTokenSource.Token);
			}
		}

		public void Stop()
		{
			lock (_lock)
			{
				m_cancellationTokenSource?.Cancel();
			}
		}

		public void Clear()
		{
			Sockets.Clear();
		}

		public void SetPhasemapping(SCNSocket socket, string phasemapping)
		{
			if (Sockets.TryGetValue(GetSocketKey(socket), out var value))
			{
				value.PhaseMapping = phasemapping;
			}
			else
			{
				Logger.Warning("SetPhasemapping {PhaseMapping}: Socket not found for key {UniqueID}_{SocketIndex}", phasemapping, socket.UniqueID, socket.SocketIndex);
			}
		}

		public void Remove(SCNSocket socket)
		{
			if (!Sockets.TryRemove(GetSocketKey(socket), out var _))
			{
				Logger.Debug("Failed removing {SocketId}/{SocketIp} from the socket list", socket.Id, socket.IPAddress);
			}
			Logger.Debug("Removed {SocketId}/{SocketIp} from the socket list", socket.Id, socket.IPAddress);
		}

		public IReadOnlyList<SCNSocket> GetCopyOfSockets(string scnNetworkName)
		{
			return (from a in Sockets.Values
				where a.NetworkName?.ToLowerInvariant() == scnNetworkName.ToLowerInvariant() && DateTime.UtcNow - a.LastUpdate <= m_16SecondsTimeSpan
				orderby a.Id, a.PropChangedCounter descending
				select a).ToList();
		}

		public SCNSocket GetFirstSocket(string scnNetworkName)
		{
			return GetCopyOfSockets(scnNetworkName).FirstOrDefault();
		}

		public ICULanDevice GetLanDeviceForFirstSocket(string scnName, LANConnection lanConnection)
		{
			SCNSocket firstSocket = GetFirstSocket(scnName);
			if (firstSocket == null)
			{
				return null;
			}
			return lanConnection.FindLanDevice(firstSocket.IPAddress);
		}
	}
	public enum EChargingState
	{
		Empty,
		Idle,
		ChargingInitializing,
		ChargingProbing,
		ChargingIncreaseCurrent,
		Charging,
		Alternating,
		Unconnected
	}
	public class SCNSocket
	{
		public int ScnLibVersion { get; set; }

		public string Name { get; set; }

		public uint Id { get; set; }

		public byte Mode3State { get; set; }

		public byte SocketIndex { get; set; }

		public byte SocketCount { get; set; }

		public EChargingState State { get; set; }

		public DateTime LastUpdate { get; set; }

		public string NetworkName { get; set; }

		public uint Timestamp { get; set; }

		public int TotalNumberOfSockets { get; set; }

		public uint PhaseMask { get; set; }

		public double ActiveCurrentL1 { get; set; }

		public double ActiveCurrentL2 { get; set; }

		public double ActiveCurrentL3 { get; set; }

		public ulong UniqueID { get; set; }

		public int MaximumGroupID { get; set; }

		public ulong Clock { get; set; }

		public DateTime LastClockUpdate { get; set; }

		public uint WaitingSince { get; set; }

		public int AlterningSince { get; set; }

		public double MinimumCurrent { get; set; }

		public double MaximumCurrent { get; set; }

		public double AvailableCurrentL1 { get; set; }

		public double AvailableCurrentL2 { get; set; }

		public double AvailableCurrentL3 { get; set; }

		public double SetPointCurrent { get; set; }

		public uint ActiveChargingTime { get; set; }

		public uint AlternatingCountDown { get; set; }

		public IPAddress IPAddress { get; set; }

		public double PropSocketSafeCurrent { get; set; }

		public double PropMaximumStaticCurrent { get; set; }

		public int PropAlternatingPeriod { get; set; }

		public int PropChangedCounter { get; set; }

		public byte OptionByte { get; set; }

		public double ExtraCurrentL1 { get; set; }

		public double ExtraCurrentL2 { get; set; }

		public double ExtraCurrentL3 { get; set; }

		public double MaximumGroupCurrent { get; set; }

		public double PropTotalSafeCurrent { get; set; }

		public string PhaseMapping { get; set; }

		public static HashSet<int> ValidLibraryVersions { get; private set; }

		static SCNSocket()
		{
			ValidLibraryVersions = Enum.GetValues(typeof(ScnLibraryVersions)).Cast<int>().ToHashSet();
		}

		public SCNSocket()
		{
			NetworkName = string.Empty;
			Id = 0u;
			Name = string.Empty;
			State = EChargingState.Idle;
			LastClockUpdate = DateTime.MinValue;
		}

		public void ParseData(byte[] data, DateTime dtReceived, IPAddress ipAddress)
		{
			int num = data[0];
			if (!ValidLibraryVersions.Contains(num) || data.Length < 96)
			{
				return;
			}
			IPAddress = ipAddress;
			LastClockUpdate = dtReceived;
			LastUpdate = DateTime.UtcNow;
			ScnLibVersion = num;
			using MemoryStream input = new MemoryStream(data);
			using BinaryReader binaryReader = new BinaryReader(input);
			binaryReader.ReadByte();
			binaryReader.ReadByte();
			binaryReader.ReadByte();
			binaryReader.ReadByte();
			uint timestamp = binaryReader.ReadUInt32();
			byte[] bytes = binaryReader.ReadBytes(8);
			NetworkName = Encoding.UTF8.GetString(bytes).Trim(new char[1]).Trim();
			if (NetworkName.Length == 0)
			{
				return;
			}
			Id = binaryReader.ReadUInt16();
			binaryReader.ReadByte();
			Mode3State = binaryReader.ReadByte();
			State = (EChargingState)binaryReader.ReadByte();
			PhaseMask = binaryReader.ReadByte();
			TotalNumberOfSockets = binaryReader.ReadByte();
			SocketIndex = binaryReader.ReadByte();
			Timestamp = timestamp;
			byte[] bytes2 = binaryReader.ReadBytes(21);
			Name = Encoding.UTF8.GetString(bytes2).Trim(new char[1]).Trim();
			SocketCount = binaryReader.ReadByte();
			MaximumGroupID = binaryReader.ReadUInt16();
			UniqueID = binaryReader.ReadUInt64();
			Clock = binaryReader.ReadUInt64();
			WaitingSince = binaryReader.ReadUInt32();
			AlterningSince = binaryReader.ReadInt32();
			MinimumCurrent = binaryReader.ReadSingle();
			MaximumCurrent = binaryReader.ReadSingle();
			ActiveCurrentL1 = binaryReader.ReadSingle();
			ActiveCurrentL2 = binaryReader.ReadSingle();
			ActiveCurrentL3 = binaryReader.ReadSingle();
			AvailableCurrentL1 = binaryReader.ReadSingle();
			AvailableCurrentL2 = binaryReader.ReadSingle();
			AvailableCurrentL3 = binaryReader.ReadSingle();
			SetPointCurrent = binaryReader.ReadSingle();
			ActiveChargingTime = binaryReader.ReadUInt32();
			AlternatingCountDown = binaryReader.ReadUInt32();
			if (num >= 3)
			{
				PropSocketSafeCurrent = binaryReader.ReadSingle();
				PropMaximumStaticCurrent = binaryReader.ReadSingle();
			}
			else
			{
				PropSocketSafeCurrent = (double)(int)binaryReader.ReadUInt16() / 10.0;
				PropMaximumStaticCurrent = (double)(int)binaryReader.ReadUInt16() / 10.0;
			}
			PropAlternatingPeriod = binaryReader.ReadUInt16();
			PropChangedCounter = binaryReader.ReadByte();
			OptionByte = binaryReader.ReadByte();
			if (num >= 2)
			{
				ExtraCurrentL1 = (int)binaryReader.ReadUInt16();
				ExtraCurrentL2 = (int)binaryReader.ReadUInt16();
				ExtraCurrentL3 = (int)binaryReader.ReadUInt16();
				binaryReader.ReadInt16();
				MaximumGroupCurrent = binaryReader.ReadSingle();
				if (num >= 3)
				{
					PropTotalSafeCurrent = binaryReader.ReadSingle();
				}
			}
		}
	}
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
}
namespace ICUNetwork.Helpers
{
	internal class ProgressHelper
	{
		protected double[] _steps;

		protected double[] _tresholds;

		protected int _currtentStage;

		internal ProgressHelper(bool isAhp)
		{
			if (isAhp)
			{
				_steps = new double[4] { 0.1, 0.1, 0.25, 0.3 };
				_tresholds = new double[4] { 4.0, 8.0, 97.0, 100.0 };
			}
			else
			{
				_steps = new double[3] { 0.25, 0.5, 0.5 };
				_tresholds = new double[3] { 50.0, 97.0, 100.0 };
			}
		}

		internal double GetNextStage()
		{
			if (_currtentStage < _steps.Length - 1)
			{
				return _tresholds[_currtentStage++];
			}
			return 0.0;
		}

		internal double GetProgress(double progress)
		{
			double num = progress + _steps[_currtentStage];
			if (num >= _tresholds[_currtentStage])
			{
				return progress;
			}
			return num;
		}
	}
}
