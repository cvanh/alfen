using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Threading;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelSCNOverview : PanelSCNBase
{
	private readonly ILogger Logger = Log.ForContext<PanelSCNOverview>();

	private TimeSpan m_8sTimeSpan = new TimeSpan(0, 0, 8);

	private Button m_btnAdd;

	private Button m_btnRemove;

	private Button m_btnUp;

	private Button m_btnDown;

	public PanelSCNOverview(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, showBorder)
	{
		Title = "SCN Overview";
		Tooltip = "Smart Charging Network - Overview Screen";
		IconName = "networking-overview.png";
	}

	protected override void OnShowPanelUser()
	{
		BuildScreen();
	}

	public override bool OnChangeSCN(SCNNetwork network, string SCNName, LANConnection lanCon)
	{
		m_fAskFirewallQuestion = true;
		m_currentNetwork = network;
		m_sSCNName = SCNName;
		m_lanConnection = lanCon;
		BuildScreen();
		return true;
	}

	private void BuildScreen()
	{
		ClearPanel();
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		AddHeader($"{Tooltip}: {m_sSCNName}");
		m_lstSockets = null;
		m_lstSockets = new ListView
		{
			DataSource = m_lsSockets,
			SelectionMode = SelectionMode.Single
		};
		m_btnAdd = AddToolImageButton("add24.png", "Add a new Charging Station", OnAddClicked, rightSide: false);
		m_btnRemove = AddToolImageButton("remove24.png", "Remove an Charging Station from this Smart Charging Network", OnRemoveClicked, rightSide: false);
		m_btnUp = AddToolImageButton("arrow-up.png", "Increase the Charging Station Priority", OnIncreasePrio, rightSide: false);
		m_btnDown = AddToolImageButton("arrow-down.png", "Lower the Charging Station Priority", OnDecreasePrio, rightSide: false);
		AddToolImageButton("refresh24.png", "Refresh", (object s, EventArgs a) =>
		{
			m_currentNetwork.Clear();
		}, rightSide: false, enabled: true);
		m_lstSockets.Columns.Clear();
		AddLeftCol(m_lstSockets, new ListViewColumn("Id", new UICustomCell(m_dfId, 16, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("V", new UICustomCell(m_dfScnLibVersion, 18, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Name", new UICustomCell(m_dfName, 140, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Socket", new UICustomCell(m_dfSock, 30, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("State", new UICustomCell(m_dfState, 90, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Current\n     L1", new UICustomCell(m_dfCurrentL1, 60, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Current\n     L2", new UICustomCell(m_dfCurrentL2, 60, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Current\n     L3", new UICustomCell(m_dfCurrentL3, 60, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Clock", new UICustomCell(m_dfClock, 40, m_dfOptions, Colors.DarkGray)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Info", new UICustomCell(m_dfInfo, 300, m_dfOptions, Colors.DarkGray)));
		AddLeftCol(m_lstSockets, new ListViewColumn("FW", new UICustomCell(m_dfFirmware, 40, m_dfOptions, Colors.DarkGray)));
		AddLeftCol(m_lstSockets, new ListViewColumn("IP-address", new UICustomCell(m_dfIP, 60, m_dfOptions, Colors.DarkGray)));
		m_lstSockets.GridLinesVisible = GridLines.Horizontal;
		m_lstSockets.MinHeight = 220.0;
		m_lstSockets.MinWidth = 100.0;
		m_lstSockets.WidthRequest = 100.0;
		m_lstSockets.HeightRequest = 300.0;
		m_lstSockets.SelectionChanged += OnDeviceSelectionChanged;
		Add(m_lstSockets, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
		m_tableRowCounter++;
		m_lstSockets.QueueForReallocate();
		m_lblInfo = (UIPropertyLabel)AddLabelText("\n\n\n", EUILabelType.LargeInfo);
		m_lblInfo.SetVisible(fVisible: false);
		m_nIdsChangingCounter = 5;
		if (m_currentNetwork != null)
		{
			UpdateInfo();
		}
		m_btnInitialize = null;
		m_btnInitialize = AddCustomButton("Initialize", "Initialize this SCN", OnInitializeClicked, rightSide: false);
		m_btnInitialize.Sensitive = true;
		StartUpdateTimer(1000);
	}

	private void UpdateInfo()
	{
		if (m_currentNetwork == null || string.IsNullOrEmpty(m_sSCNName))
		{
			m_btnRemove.Sensitive = false;
			m_btnAdd.Sensitive = false;
			return;
		}
		m_btnAdd.Sensitive = true;
		SCNSocket sCNSocket = null;
		if (m_lstSockets.SelectedRow >= 0)
		{
			sCNSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
		}
		m_lsSockets.Clear();
		int num = 0;
		int row = 0;
		double num2 = 0.0;
		double num3 = 0.0;
		double num4 = 0.0;
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		foreach (SCNSocket sock in copyOfSockets)
		{
			string text = ((sock.MaximumGroupID != 0) ? $"GrpMax {sock.MaximumGroupCurrent}" : string.Empty);
			int num5 = (int)Math.Max(sock.ExtraCurrentL1, Math.Max(sock.ExtraCurrentL2, sock.ExtraCurrentL3));
			if (sock.ExtraCurrentL1 > 0.0 && sock.ExtraCurrentL1 < (double)num5)
			{
				num5 = (int)sock.ExtraCurrentL1;
			}
			if (sock.ExtraCurrentL2 > 0.0 && sock.ExtraCurrentL2 < (double)num5)
			{
				num5 = (int)sock.ExtraCurrentL2;
			}
			if (sock.ExtraCurrentL3 > 0.0 && sock.ExtraCurrentL3 < (double)num5)
			{
				num5 = (int)sock.ExtraCurrentL3;
			}
			string value = "Min " + sock.MinimumCurrent.ToString("0", CultureInfo.InvariantCulture) + ", Max " + sock.MaximumCurrent.ToString("0", CultureInfo.InvariantCulture) + ", SP " + sock.SetPointCurrent.ToString("F1", CultureInfo.InvariantCulture) + ", " + $"Ex {num5}, " + text;
			string text2 = string.Empty;
			switch (sock.State)
			{
			case EChargingState.Alternating:
				text2 = "Alternating";
				break;
			case EChargingState.Charging:
				text2 = "Charging";
				break;
			case EChargingState.ChargingIncreaseCurrent:
				text2 = "Increasing";
				break;
			case EChargingState.ChargingInitializing:
				text2 = "Initializing";
				break;
			case EChargingState.ChargingProbing:
				text2 = "Probing";
				break;
			case EChargingState.Empty:
				text2 = "Empty";
				break;
			case EChargingState.Idle:
				text2 = "Idle";
				break;
			case EChargingState.Unconnected:
				text2 = "Unconnected";
				break;
			}
			if (sock.Mode3State != 0)
			{
				text2 = $"{text2} ({sock.Mode3State:X2})";
			}
			string text3 = (((sock.PhaseMask & 1) != 0) ? DoubleToString(sock.ActiveCurrentL1) : "-");
			string value2 = (((sock.PhaseMask & 2) != 0) ? DoubleToString(sock.ActiveCurrentL2) : "-");
			string value3 = (((sock.PhaseMask & 4) != 0) ? DoubleToString(sock.ActiveCurrentL3) : "-");
			if (DateTime.UtcNow - sock.LastUpdate > m_8sTimeSpan)
			{
				text2 = "[Unconnected]";
				text3 = "[" + m_dLastSafeCurrent.ToString("0.0", CultureInfo.InvariantCulture) + "]";
				value3 = (value2 = text3);
			}
			string value4 = "─ " + sock.Name;
			if (copyOfSockets.Count((SCNSocket a) => a.UniqueID == sock.UniqueID && a.Id != sock.Id) > 0)
			{
				value4 = ((sock.SocketIndex != 0) ? ("└ " + sock.Name) : ("┌ " + sock.Name));
			}
			string value5 = (sock.SocketIndex + 1).ToString();
			ICULanDevice iCULanDevice = m_lanConnection.FindLanDevice(sock.IPAddress);
			num = m_lsSockets.AddRow();
			m_lsSockets.SetValues(num, m_dfId, sock.Id.ToString(), m_dfName, value4, m_dfFirmware, $"{iCULanDevice?.FirmwareVersionNumber}", m_dfSock, value5, m_dfState, text2, m_dfCurrentL1, text3, m_dfCurrentL2, value2, m_dfCurrentL3, value3, m_dfSocket, sock, m_dfClock, (sock.Clock & 0xFFFF).ToString());
			m_lsSockets.SetValues(num, m_dfInfo, value, m_dfUniqueID, sock.UniqueID.ToString(), m_dfIP, sock.IPAddress.ToString(), m_dfScnLibVersion, ((ScnLibraryVersions)sock.ScnLibVersion).GetEnumDescription(), m_dfTotalNumberOfSockets, sock.TotalNumberOfSockets.ToString());
			num2 += sock.ActiveCurrentL1;
			num3 += sock.ActiveCurrentL2;
			num4 += sock.ActiveCurrentL3;
			if (sock == sCNSocket)
			{
				row = num;
			}
		}
		if (copyOfSockets.Count > 0)
		{
			m_lstSockets.SelectRow(row);
			num = m_lsSockets.AddRow();
			m_lsSockets.SetValues(num, m_dfName, "", m_dfState, "", m_dfOptions, 2);
			num = m_lsSockets.AddRow();
			m_lsSockets.SetValues(num, m_dfName, "Total SCN Chargers Usage", m_dfState, "", m_dfCurrentL1, DoubleToString(num2) ?? "", m_dfCurrentL2, DoubleToString(num3) ?? "", m_dfCurrentL3, DoubleToString(num4) ?? "", m_dfSocket, null, m_dfClock, "", m_dfItem, null, m_dfOptions, 1);
			num = m_lsSockets.AddRow();
			string text4 = DoubleToString(copyOfSockets[0].PropMaximumStaticCurrent);
			m_lsSockets.SetValues(num, m_dfName, "SCN Currents (Available/Total)", m_dfState, "", m_dfCurrentL1, DoubleToString(copyOfSockets.Min((SCNSocket a) => a.AvailableCurrentL1)) + "/" + text4, m_dfCurrentL2, DoubleToString(copyOfSockets.Min((SCNSocket a) => a.AvailableCurrentL2)) + "/" + text4, m_dfCurrentL3, DoubleToString(copyOfSockets.Min((SCNSocket a) => a.AvailableCurrentL3)) + "/" + text4, m_dfSocket, null, m_dfClock, "", m_dfItem, null, m_dfOptions, 1);
			m_btnRemove.Sensitive = true;
			m_btnAdd.Sensitive = true;
		}
	}

	protected override void OnUpdateTick()
	{
		if (m_currentNetwork != null && m_currentUser != null)
		{
			Application.Invoke(() =>
			{
				UpdateInfo();
				CheckForErrors();
			});
		}
	}

	private void OnDeviceSelectionChanged(object sender, EventArgs e)
	{
		SCNSocket sCNSocket = null;
		if (m_btnUp != null && m_btnDown != null)
		{
			if (m_lstSockets.SelectedRow >= 0)
			{
				sCNSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
			}
			int num = 0;
			if (m_currentNetwork != null)
			{
				num = m_currentNetwork.GetCopyOfSockets(m_sSCNName).Count;
			}
			m_btnUp.Sensitive = m_lsSockets.RowCount > 1 && sCNSocket != null && sCNSocket.Id != 0 && sCNSocket.SocketIndex == 0;
			m_btnDown.Sensitive = sCNSocket != null && sCNSocket.Id < num - sCNSocket.SocketCount && sCNSocket.SocketIndex == 0;
		}
	}

	private void OnAddClicked(object sender, EventArgs e)
	{
		MainWindow mainWindow = (MainWindow)ParentWindow;
		mainWindow.SuspendUpdateTimers(fSuspend: true);
		try
		{
			using DlgAddToSCN dlgAddToSCN = new DlgAddToSCN(m_sSCNName, m_currentNetwork, m_lanConnection);
			if (dlgAddToSCN.Run(ParentWindow) == Command.Ok)
			{
				Logger.AddChargerContext(m_currentDevice).Information("Device added to SCN");
				UpdateInfo();
				Thread.Sleep(1000);
				m_parent.RefreshDeviceList();
			}
		}
		finally
		{
			mainWindow.SuspendUpdateTimers(fSuspend: false);
		}
	}

	private void OnRemoveClicked(object sender, EventArgs e)
	{
		if (m_currentNetwork == null)
		{
			return;
		}
		SCNSocket selectedSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
		if (selectedSocket == null || m_lanConnection == null)
		{
			Log.Warning("no socket selected");
			return;
		}
		ICULanDevice selectedDevice = m_lanConnection.Devices.OfType<ICULanDevice>().FirstOrDefault((ICULanDevice a) => a.Identification == selectedSocket.Name);
		if (selectedDevice == null)
		{
			Log.Warning("no device selected");
		}
		else
		{
			if (MessageDialog.AskQuestion($"Are you sure you want to remove Charging Station '{selectedDevice.Identification}' from network '{m_sSCNName}'?", Command.Yes, Command.No, Command.Cancel) != Command.Yes)
			{
				return;
			}
			if (!selectedDevice.Login().IsLoggedIn)
			{
				Log.Warning("login failed or cancelled");
				return;
			}
			if (!selectedDevice.UpdateCategories("scn"))
			{
				Log.Warning("update scn categories failed");
				return;
			}
			bool flag = false;
			IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
			if (copyOfSockets.Count() == 0)
			{
				Log.Warning("no sockets in SCN");
				return;
			}
			foreach (SCNSocket item in copyOfSockets.Where((SCNSocket s) => s.IPAddress.ToString() != selectedDevice.IPAddress.ToString()))
			{
				if (item.SocketIndex == 0)
				{
					ICULanDevice iCULanDevice = m_lanConnection.FindLanDevice(item.IPAddress);
					if (iCULanDevice != null && !iCULanDevice.Login().IsLoggedIn)
					{
						flag = true;
						Log.Warning("{IpAddress} login failed", iCULanDevice.IPAddress);
						break;
					}
				}
			}
			if (!flag)
			{
				Log.Information("Updating SCN");
				ICUProperty property = selectedDevice.GetProperty(8576, 1);
				if (property == null)
				{
					return;
				}
				property.Value = string.Empty;
				selectedDevice.StoreChangedProperties();
				selectedDevice.SCNNetwork = string.Empty;
				List<SCNSocket> list = copyOfSockets.ToList();
				SCNSocket sCNSocket = copyOfSockets.FirstOrDefault((SCNSocket a) => a.UniqueID == selectedSocket.UniqueID && a.Id != selectedSocket.Id);
				if (sCNSocket != null)
				{
					m_currentNetwork.Remove(sCNSocket);
				}
				list.RemoveAll((SCNSocket a) => a.UniqueID == selectedSocket.UniqueID);
				m_currentNetwork.Remove(selectedSocket);
				Log.Verbose("Updating sockets {SocketIds}", list.Select((SCNSocket s) => s.Id));
				ICUProperty property2 = selectedDevice.GetProperty(8576, 2);
				ICUProperty property3 = selectedDevice.GetProperty(8576, 3);
				ICUProperty property4 = selectedDevice.GetProperty(8576, 4);
				ICUProperty property5 = selectedDevice.GetProperty(8576, 5);
				ICUProperty property6 = selectedDevice.GetProperty(8576, 6);
				ICUProperty property7 = selectedDevice.GetProperty(8576, 10);
				int num = Convert.ToInt32(property4.Value);
				double num2 = Convert.ToDouble(property6.Value);
				double num3 = Convert.ToDouble(property7.Value);
				double num4 = Convert.ToDouble(property5.Value);
				ushort num5 = (ushort)list.Count;
				uint num6 = 0u;
				foreach (SCNSocket item2 in list)
				{
					Log.Information("Updating socket {SocketName}/{SocketIp}/{SocketId}", "", item2.Name, item2.IPAddress, item2.Id);
					if (item2.SocketIndex == 0)
					{
						ICULanDevice iCULanDevice2 = m_lanConnection.FindLanDevice(item2.IPAddress);
						if (iCULanDevice2 != null && iCULanDevice2.IsLoggedIn)
						{
							iCULanDevice2.UpdateCategories("scn");
							property2.Value = num6;
							property3.Value = num5;
							property4.Value = num;
							property5.Value = num4;
							property6.Value = num2;
							property7.Value = num3;
							iCULanDevice2.StoreProperties(property2, property3, property6, property7, property5, property4);
							num6 += item2.SocketCount;
						}
					}
				}
				Log.Information("Validating CS configuration");
				m_parent.ValidateCSConfiguration(m_currentNetwork.GetLanDeviceForFirstSocket(m_sSCNName, m_lanConnection));
				Log.Information("Refresh SCN screen");
				UpdateInfo();
				Log.Information("Refresh device list");
				m_parent.RefreshDeviceList();
			}
			Log.Information("Logout of all devices");
			selectedDevice.Logout();
			foreach (SCNSocket item3 in copyOfSockets)
			{
				if (item3.SocketIndex == 0)
				{
					m_lanConnection.FindLanDevice(item3.IPAddress)?.Logout();
				}
			}
			Logger.AddChargerContext(m_currentDevice).Information("Device removed from SCN");
		}
	}

	private void OnIncreasePrio(object sender, EventArgs e)
	{
		if (m_lstSockets.SelectedRow <= 0)
		{
			return;
		}
		SCNSocket previousSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow - 1, m_dfSocket);
		SCNSocket selectedSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
		if (previousSocket == null || selectedSocket == null || m_lanConnection == null)
		{
			return;
		}
		uint num = previousSocket.Id - previousSocket.SocketIndex;
		uint num2 = selectedSocket.Id - previousSocket.SocketIndex + selectedSocket.SocketCount - 1;
		if (num >= num2)
		{
			return;
		}
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		if (copyOfSockets.Any((SCNSocket s) => s.UniqueID == previousSocket.UniqueID && s.State != EChargingState.Idle) || copyOfSockets.Any((SCNSocket s) => s.UniqueID == selectedSocket.UniqueID && s.State != EChargingState.Idle))
		{
			MessageDialog.ShowError("All sockets on the selected CS and the CS above have to be idle");
		}
		else if ((!(m_lanConnection.Devices.FirstOrDefault((ICUDevice a) => a.Identification == selectedSocket.Name) is ICULanDevice iCULanDevice) || iCULanDevice.Login().IsLoggedIn) && (!(m_lanConnection.Devices.FirstOrDefault((ICUDevice a) => a.Identification == previousSocket.Name) is ICULanDevice iCULanDevice2) || iCULanDevice2.Login().IsLoggedIn))
		{
			SetSocketData(selectedSocket.Name, num);
			SetSocketData(previousSocket.Name, num2);
			Logger.AddChargerContext(m_currentDevice).Information("Increased socket priority");
			if (m_currentNetwork != null)
			{
				m_currentNetwork.Clear();
			}
			m_nIdsChangingCounter = 5;
		}
	}

	private void OnDecreasePrio(object sender, EventArgs e)
	{
		int selectedRow = m_lstSockets.SelectedRow;
		byte b = 1;
		if (selectedRow < 0 || selectedRow + 1 >= m_lsSockets.RowCount)
		{
			return;
		}
		SCNSocket selectedSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
		SCNSocket nextSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow + b, m_dfSocket);
		if (nextSocket.Name == selectedSocket.Name)
		{
			b = 2;
			nextSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow + b, m_dfSocket);
		}
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		if (copyOfSockets.Any((SCNSocket s) => s.UniqueID == nextSocket.UniqueID && s.State != EChargingState.Idle) || copyOfSockets.Any((SCNSocket s) => s.UniqueID == selectedSocket.UniqueID && s.State != EChargingState.Idle))
		{
			MessageDialog.ShowError("All sockets on the selected CS and the CS below have to be idle");
		}
		else
		{
			if (nextSocket == null || selectedSocket == null || m_lanConnection == null)
			{
				return;
			}
			uint id = selectedSocket.Id;
			uint num = nextSocket.Id + (uint)(nextSocket.SocketCount - b);
			if (num > id && (!(m_lanConnection.Devices.FirstOrDefault((ICUDevice a) => a.Identification == selectedSocket.Name) is ICULanDevice iCULanDevice) || iCULanDevice.Login().IsLoggedIn) && (!(m_lanConnection.Devices.FirstOrDefault((ICUDevice a) => a.Identification == nextSocket.Name) is ICULanDevice iCULanDevice2) || iCULanDevice2.Login().IsLoggedIn))
			{
				SetSocketData(selectedSocket.Name, num);
				SetSocketData(nextSocket.Name, id);
				Logger.AddChargerContext(m_currentDevice).Information("Decreased socket priority");
				if (m_currentNetwork != null)
				{
					m_currentNetwork.Clear();
				}
				m_nIdsChangingCounter = 5;
			}
		}
	}
}
