using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.Linq;
using System.Reflection;
using ICUNetwork;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelSCNBase : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelSCNBase>();

	protected SCNNetwork m_currentNetwork;

	protected LANConnection m_lanConnection;

	protected string m_sSCNName = string.Empty;

	protected ListView m_lstSockets;

	protected ListStore m_lsSockets;

	protected DataField<string> m_dfId = new DataField<string>();

	protected DataField<string> m_dfIP = new DataField<string>();

	protected DataField<string> m_dfUniqueID = new DataField<string>();

	protected DataField<string> m_dfTotalNumberOfSockets = new DataField<string>();

	protected DataField<string> m_dfName = new DataField<string>();

	protected DataField<string> m_dfFirmware = new DataField<string>();

	protected DataField<string> m_dfSock = new DataField<string>();

	protected DataField<string> m_dfState = new DataField<string>();

	protected DataField<string> m_dfCurrentL1 = new DataField<string>();

	protected DataField<string> m_dfCurrentL2 = new DataField<string>();

	protected DataField<string> m_dfCurrentL3 = new DataField<string>();

	protected DataField<SCNSocket> m_dfSocket = new DataField<SCNSocket>();

	protected DataField<string> m_dfClock = new DataField<string>();

	protected DataField<ICUDevice> m_dfItem = new DataField<ICUDevice>();

	protected DataField<int> m_dfOptions = new DataField<int>();

	protected DataField<string> m_dfInfo = new DataField<string>();

	protected DataField<string> m_dfPhaseMapping = new DataField<string>();

	protected DataField<string> m_dfFeederCables = new DataField<string>();

	protected DataField<string> m_dfScnLibVersion = new DataField<string>();

	protected Button m_btnInitialize;

	protected UIPropertyLabel m_lblInfo;

	protected int m_nIdsChangingCounter = 10;

	protected bool m_fAskFirewallQuestion = true;

	private int m_nCheckFirewallInterval = 10;

	protected double m_dLastSafeCurrent;

	public PanelSCNBase(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		m_lsSockets = new ListStore(m_dfId, m_dfName, m_dfScnLibVersion, m_dfFirmware, m_dfUniqueID, m_dfTotalNumberOfSockets, m_dfIP, m_dfSock, m_dfState, m_dfCurrentL1, m_dfCurrentL2, m_dfCurrentL3, m_dfSocket, m_dfClock, m_dfItem, m_dfInfo, m_dfOptions, m_dfPhaseMapping, m_dfFeederCables);
	}

	protected void AddLeftCol(ListView lv, ListViewColumn col)
	{
		col.Alignment = Alignment.Start;
		lv.Columns.Add(col);
	}

	protected void TestUDPConnection()
	{
		m_nCheckFirewallInterval--;
		if (m_nCheckFirewallInterval != 0)
		{
			return;
		}
		if (!CheckFirewallRules(out var info))
		{
			m_lblInfo.SetValue($"No sockets are detected due to a firewall problem:\n{info}");
			m_lblInfo.SetType(EUILabelType.LargeError);
			m_lblInfo.SetVisible(fVisible: true);
			m_lstSockets.Sensitive = false;
			if (m_fAskFirewallQuestion)
			{
				if (MessageDialog.AskQuestion(string.Format("An inbound rule seems to be missing from the Firewall, do you want to add the inbound rule?", Array.Empty<object>()), Command.Yes, Command.No, Command.Cancel) == Command.Yes)
				{
					AddFirewallRule();
				}
				m_fAskFirewallQuestion = false;
			}
		}
		else
		{
			m_lblInfo.SetVisible(fVisible: false);
			m_lstSockets.Sensitive = true;
		}
		m_nCheckFirewallInterval = 10;
	}

	protected void CheckForSafecurrent()
	{
		if (m_currentNetwork == null)
		{
			return;
		}
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		SCNSocket sCNSocket = copyOfSockets.FirstOrDefault();
		if (sCNSocket == null)
		{
			return;
		}
		double num = sCNSocket.PropSocketSafeCurrent;
		double num2 = sCNSocket.PropTotalSafeCurrent;
		double num3 = sCNSocket.PropMaximumStaticCurrent;
		if (num * (double)copyOfSockets.Count > num3)
		{
			m_lblInfo.SetType(EUILabelType.LargeInfo);
			int num4 = 0;
			if (num != 0.0)
			{
				num4 = (int)(num3 / num);
			}
			if (num4 <= 0)
			{
				m_lblInfo.SetValue("When sockets are disconnected from the network,\nno socket is able to reach the safe current!");
			}
			else if (num4 == 1)
			{
				m_lblInfo.SetValue($"When sockets are disconnected from the network,\nonly the first socket (Id 0) is able to charge with the safe current of {num:0}A");
			}
			else
			{
				m_lblInfo.SetValue($"When sockets are disconnected from the network,\nonly the first {num4} sockets are able to reach the safe current of {num:0}A");
			}
			m_lblInfo.SetVisible(fVisible: true);
		}
		else if (num2 > num3)
		{
			m_lblInfo.SetType(EUILabelType.LargeInfo);
			m_lblInfo.SetValue($"The Total SCN Safe current of {num2} cannot be higher than the Maximum SCN current of {num2}");
			m_lblInfo.SetVisible(fVisible: true);
		}
		else
		{
			m_lblInfo.SetVisible(fVisible: false);
		}
	}

	protected void CheckForValidSCN()
	{
		if (m_nIdsChangingCounter > 0)
		{
			m_nIdsChangingCounter--;
		}
		if (m_nIdsChangingCounter == 0)
		{
			Logger.Debug("Checking for valid SCN");
			m_lblInfo.SetType(EUILabelType.LargeError);
			IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
			int num = copyOfSockets.Count();
			int num2 = copyOfSockets.FirstOrDefault()?.TotalNumberOfSockets ?? 0;
			if (num != num2)
			{
				m_lblInfo.SetValue($"Not all sockets are detected! received {num} vs {num2} registered");
			}
			else if (copyOfSockets.Select((SCNSocket a) => a.ScnLibVersion).Distinct().Count() != 1)
			{
				m_lblInfo.SetValue("The chargers in the SCN are not aligned with the same SCN library version! Read more about the compatibility ");
				m_lblInfo.SetHyperlink("here", "https://knowledge.alfen.com/categories/CAT-00913/KA-01634");
			}
			else
			{
				m_lblInfo.SetValue("Your SCN is not configured correctly!\nWhen all Charging Stations are present in the list, please press the 'Initialize' button to (re-)initialize this SCN.");
			}
			m_lblInfo.SetVisible(fVisible: true);
		}
	}

	protected void CheckForErrors()
	{
		Application.Invoke(() =>
		{
			if (!m_currentNetwork.GetCopyOfSockets(m_sSCNName).Any())
			{
				TestUDPConnection();
			}
			else if (!IsSCNValid())
			{
				CheckForValidSCN();
			}
			else
			{
				CheckForSafecurrent();
			}
		});
	}

	protected bool IsSCNValid()
	{
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		if (copyOfSockets.Count <= 0)
		{
			return true;
		}
		if (copyOfSockets.Count((SCNSocket a) => a.State != EChargingState.Unconnected && a.State != EChargingState.Empty) != copyOfSockets.Count())
		{
			return true;
		}
		int numberOfSockets = copyOfSockets[0].TotalNumberOfSockets;
		if (copyOfSockets.Count() != numberOfSockets)
		{
			Logger.Debug("Error IsSCNValid: Not all sockets are detected!");
			return false;
		}
		if (copyOfSockets.Count((SCNSocket a) => a.TotalNumberOfSockets != numberOfSockets) > 0)
		{
			Logger.Debug("Error IsSCNValid: Not all sockets have the same total-number-of-sockets");
			return false;
		}
		int n;
		for (n = 0; n < numberOfSockets; n++)
		{
			if (copyOfSockets.Count((SCNSocket a) => a.Id == n) != 1)
			{
				Logger.Debug("Error IsSCNValid: Gap detected in socket Ids");
				return false;
			}
		}
		if (copyOfSockets.Select((SCNSocket a) => a.ScnLibVersion).Distinct().Count() != 1)
		{
			Logger.Debug("Error The version of the SCN lib version is not the same for all chargers");
			return false;
		}
		return true;
	}

	protected List<dynamic> GetFirewallRules(string progID, string applicationTarget = "")
	{
		dynamic val = Activator.CreateInstance(Type.GetTypeFromProgID("HNetCfg.FwPolicy2"));
		List<object> list = new List<object>();
		foreach (dynamic item in val.Rules)
		{
			if (!((item.ApplicationName == null) ? true : false))
			{
				bool flag = string.IsNullOrEmpty(applicationTarget);
				if (flag || ((flag | (string.Compare(applicationTarget, Path.GetFullPath(item.ApplicationName).TrimEnd('\\'), StringComparison.InvariantCultureIgnoreCase) == 0)) ? true : false))
				{
					list.Add(item);
				}
			}
		}
		return list;
	}

	protected bool CheckFirewallRules(out string info)
	{
		bool result = false;
		try
		{
			Activator.CreateInstance(Type.GetTypeFromProgID("HNetCfg.FwPolicy2"));
			string applicationTarget = Path.GetFullPath(Process.GetCurrentProcess().MainModule.FileName).TrimEnd(new char[1] { '\\' });
			info = $"Inbound UDP port not found for application '{((AssemblyTitleAttribute)Attribute.GetCustomAttribute(Assembly.GetExecutingAssembly(), typeof(AssemblyTitleAttribute), inherit: false))?.Title}' in firewall";
			foreach (dynamic firewallRule in GetFirewallRules("HNetCfg.FwPolicy2", applicationTarget))
			{
				if (!((firewallRule.Protocol == 17 && firewallRule.Direction == 1) ? true : false))
				{
					continue;
				}
				string text = firewallRule.LocalPorts.ToString().Trim();
				if (text == "*" || text.Contains(SCNNetwork.UDPPORT.ToString()))
				{
					if (firewallRule.Enabled)
					{
						if (!((firewallRule.Action == 1) ? true : false))
						{
							result = false;
							info = "Inbound UDP port in firewall is being blocked!";
							break;
						}
						result = true;
					}
					else
					{
						info = "Inbound UDP port in firewall is disabled!";
					}
				}
				else
				{
					info = "Incorrect Inbound UDP port number in firewall!";
				}
			}
		}
		catch (Exception exception)
		{
			info = "";
			Logger.Error(exception, "Error while checking firewall");
		}
		return result;
	}

	protected void DisableBlockingFirewallRules()
	{
		string applicationTarget = Path.GetFullPath(Process.GetCurrentProcess().MainModule.FileName).TrimEnd(new char[1] { '\\' });
		foreach (dynamic firewallRule in GetFirewallRules("HNetCfg.FwPolicy2", applicationTarget))
		{
			if (firewallRule.Protocol == 17 && firewallRule.Direction == 1)
			{
				string text = firewallRule.LocalPorts.ToString().Trim();
				if ((text == "*" || text.Contains(SCNNetwork.UDPPORT.ToString())) && ((firewallRule.Enabled && firewallRule.Action == 0) ? true : false))
				{
					firewallRule.Enabled = false;
				}
			}
		}
	}

	protected void AddFirewallRule()
	{
		try
		{
			DisableBlockingFirewallRules();
			dynamic val = Activator.CreateInstance(Type.GetTypeFromProgID("HNetCfg.FwPolicy2"));
			dynamic val2 = Activator.CreateInstance(Type.GetTypeFromProgID("HNetCfg.FWRule"));
			AssemblyTitleAttribute assemblyTitleAttribute = (AssemblyTitleAttribute)Attribute.GetCustomAttribute(Assembly.GetExecutingAssembly(), typeof(AssemblyTitleAttribute), inherit: false);
			string text = Path.GetFullPath(Process.GetCurrentProcess().MainModule.FileName).TrimEnd(new char[1] { '\\' });
			dynamic val3 = val.CurrentProfileTypes;
			dynamic val4 = val.Rules;
			val2.Name = assemblyTitleAttribute?.Title;
			val2.Description = "Allows monitoring of the Smart Charging Network";
			val2.ApplicationName = text;
			val2.Protocol = 17;
			val2.LocalPorts = "*";
			val2.Enabled = true;
			val2.Direction = 1;
			val2.Grouping = "";
			val2.Profiles = val3;
			val2.Action = 1;
			val4.Add(val2);
		}
		catch (Exception ex)
		{
			string text2 = $"There was an error adding the inbound rule to the firewall.\nPleas contact your system administrator.\nError: {ex.Message}";
			Logger.Error(ex, text2);
			MessageDialog.ShowError(ParentWindow, text2);
		}
	}

	protected string DoubleToString(double? d)
	{
		return d?.ToString("0.0", CultureInfo.InvariantCulture) ?? "";
	}

	protected void SetSocketData(string deviceName1, uint newSocketID, int? numberOfSockets = null)
	{
		ICUDevice iCUDevice = m_lanConnection.Devices.FirstOrDefault((ICUDevice a) => a.Identification == deviceName1);
		if (iCUDevice == null)
		{
			return;
		}
		if (iCUDevice is ICULanDevice iCULanDevice && iCULanDevice.Login().IsLoggedIn)
		{
			iCULanDevice.UpdateCategories("scn");
			ICUProperty property = iCULanDevice.GetProperty(8576, 2);
			property.Value = newSocketID;
			if (numberOfSockets.HasValue)
			{
				ICUProperty property2 = iCULanDevice.GetProperty(8576, 3);
				property2.Value = numberOfSockets;
				iCULanDevice.StoreProperties(property, property2);
			}
			else
			{
				iCULanDevice.StoreProperties(property);
			}
			iCULanDevice.Logout();
		}
		else if (iCUDevice.Connection is LANConnection lANConnection)
		{
			lANConnection.CallErrorHandler("Unable to communicate with the device '" + iCUDevice.Identification + "'. Please check SCN connections.", iCUDevice as ICULanDevice);
		}
	}

	protected void OnInitializeClicked(object sender, EventArgs e)
	{
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		if (copyOfSockets.Count <= 0 || MessageDialog.AskQuestion($"Are you sure you want to (re-)initialize the SCN '{m_sSCNName}'?\nOnly perform this action when all sockets are present in the list!\nWe currently detected {copyOfSockets.Count()} sockets.", Command.Yes, Command.No, Command.Cancel) != Command.Yes)
		{
			return;
		}
		int value = copyOfSockets.Count();
		uint num = 0u;
		foreach (SCNSocket item in copyOfSockets)
		{
			if (item.SocketIndex == 0)
			{
				SetSocketData(item.Name, num, value);
				num += item.SocketCount;
			}
		}
		MessageDialog.ShowMessage(ParentWindow, "The SCN '" + m_sSCNName + "' has been initialized.");
		m_nIdsChangingCounter = 10;
	}
}
