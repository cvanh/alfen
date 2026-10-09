using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelSCNSettings : PanelSCNBase
{
	private readonly ILogger Logger = Log.ForContext<PanelSCNSettings>();

	private TimeSpan m_8sTimeSpan = new TimeSpan(0, 0, 8);

	private Button m_btnSwap;

	private UIPropertyNumber m_numTotalCurrent;

	private UIPropertyNumber m_numSocketSafeCurrent;

	private UIPropertyNumber m_numTotalSafeCurrent;

	private UIPropertyNumber m_numAltPeriod;

	private int m_nLastPropertyChangedCounter = -1;

	private CancellationTokenSource m_phasemappingTaskCancellationTokenSource;

	private Task<bool> m_phasemappingTask;

	public PanelSCNSettings(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, showBorder)
	{
		Title = "SCN Settings";
		Tooltip = "Smart Charging Network - Settings Screen";
		IconName = "networking-settings.png";
	}

	protected override void OnShowPanelUser()
	{
		BuildScreen(refreshAll: false);
	}

	public override bool OnChangeSCN(SCNNetwork network, string SCNName, LANConnection lanCon)
	{
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		m_fAskFirewallQuestion = true;
		m_currentNetwork = network;
		m_sSCNName = SCNName;
		m_lanConnection = lanCon;
		BuildScreen(refreshAll: true);
		return true;
	}

	private async Task<bool> RefreshPhaseMapping(bool refreshAll, CancellationToken cancellationToken)
	{
		if (!IsPanelVisible)
		{
			return false;
		}
		var socketsThatNeedPhaseMappingUpdate = from socket in m_currentNetwork.GetCopyOfSockets(m_sSCNName)
			where string.IsNullOrEmpty(socket.PhaseMapping) | refreshAll
			select new
			{
				Socket = socket,
				LanDevice = m_lanConnection.FindLanDevice(socket.IPAddress)
			};
		if (socketsThatNeedPhaseMappingUpdate.Count() == 0)
		{
			return true;
		}
		bool onlyUpdateIfAllChargersLoggedInSuccessfully = true;
		var chargersToLogin = from s in socketsThatNeedPhaseMappingUpdate
			group s by s.Socket.UniqueID into g
			select g.OrderBy(s => s.Socket.SocketIndex).First();
		foreach (var charger in chargersToLogin)
		{
			if (cancellationToken.IsCancellationRequested)
			{
				return false;
			}
			if (!(await Application.InvokeAsync(() => charger.LanDevice.Login())).Item1)
			{
				onlyUpdateIfAllChargersLoggedInSuccessfully = false;
				break;
			}
		}
		if (onlyUpdateIfAllChargersLoggedInSuccessfully)
		{
			foreach (var sock in socketsThatNeedPhaseMappingUpdate)
			{
				if (cancellationToken.IsCancellationRequested)
				{
					return false;
				}
				sock.LanDevice.UpdateProperties(2195463u, 2195465u);
				if (sock.Socket.SocketIndex != 0)
				{
				}
				m_currentNetwork.SetPhasemapping(sock.Socket, sock.LanDevice.GetPropertyString(8576, 7, 0));
				var anon = socketsThatNeedPhaseMappingUpdate.FirstOrDefault(a => a.Socket.UniqueID == sock.Socket.UniqueID && a.Socket.Id != sock.Socket.Id);
				if (anon != null)
				{
					int num = ((anon.Socket.SocketIndex == 0) ? 7 : 9);
					m_currentNetwork.SetPhasemapping(anon.Socket, anon.LanDevice.GetPropertyString(8576, (byte)num, 0));
				}
				Logger.Warning("Updated phasemapping of device {DeviceName} {DeviceIp}", sock.LanDevice.Name, sock.LanDevice.IPAddress.ToString());
			}
		}
		foreach (var item in chargersToLogin)
		{
			item.LanDevice.Logout();
		}
		return onlyUpdateIfAllChargersLoggedInSuccessfully;
	}

	private void BuildScreen(bool refreshAll)
	{
		ClearPanel();
		AddHeader($"{Tooltip}: {m_sSCNName}");
		m_lstSockets = null;
		m_lstSockets = new ListView
		{
			DataSource = m_lsSockets,
			SelectionMode = SelectionMode.Single
		};
		m_btnSwap = AddToolImageButton("swap24.png", "Swap the phasemapping(s)", OnSwapClicked, rightSide: false);
		AddToolImageButton("refresh24.png", "Refresh (login to all devices to fill column phase mapping)", async (object s, EventArgs a) =>
		{
			if (m_phasemappingTask == null || m_phasemappingTask.IsCompleted)
			{
				m_phasemappingTaskCancellationTokenSource = new CancellationTokenSource();
				m_phasemappingTask = RefreshPhaseMapping(refreshAll: false, m_phasemappingTaskCancellationTokenSource.Token);
				await m_phasemappingTask;
			}
			else
			{
				m_phasemappingTaskCancellationTokenSource.Cancel();
				m_phasemappingTaskCancellationTokenSource.Dispose();
			}
		}, rightSide: false, enabled: true);
		m_lstSockets.Columns.Clear();
		AddLeftCol(m_lstSockets, new ListViewColumn("Id", new UICustomCell(m_dfId, 16, m_dfOptions, Colors.DarkGray)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Name", new UICustomCell(m_dfName, 140, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Socket", new UICustomCell(m_dfSock, 40, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("State", new UICustomCell(m_dfState, 90, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Phase\n Mapping", new UICustomCell(m_dfPhaseMapping, 60, m_dfOptions)));
		AddLeftCol(m_lstSockets, new ListViewColumn("Feeder\n Cables", new UICustomCell(m_dfFeederCables, 40, m_dfOptions)));
		m_lstSockets.GridLinesVisible = GridLines.Horizontal;
		m_lstSockets.MinHeight = 220.0;
		m_lstSockets.MinWidth = 100.0;
		m_lstSockets.WidthRequest = 100.0;
		m_lstSockets.HeightRequest = 220.0;
		m_lstSockets.SelectionChanged += OnDeviceSelectionChanged;
		m_lstSockets.Margin = 3.0;
		Add(m_lstSockets, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
		m_tableRowCounter++;
		m_lstSockets.QueueForReallocate();
		AddSmallHeader("Settings");
		using (DoubleColumnMode doubleColumnMode = new DoubleColumnMode(this))
		{
			m_numTotalCurrent = (UIPropertyNumber)AddCustomNumber(0, "Total Max. current (A)", "", "", null, 2195461u);
			m_numAltPeriod = (UIPropertyNumber)AddCustomNumber(0, "Alternating period (s)", "", "", null, 2195460u);
			m_numTotalCurrent.SetValueMinMax(1.0, 3.4028234663852886E+38);
			m_numAltPeriod.SetValueMinMax(900.0, 36000.0);
			doubleColumnMode.NextColumn();
			m_numSocketSafeCurrent = (UIPropertyNumber)AddCustomNumber(0, "Socket Safe current (A)", "", "", null, 2195462u);
			m_numTotalSafeCurrent = (UIPropertyNumber)AddCustomNumber(0, "Total Safe current (A)", "", "", null, 2195466u);
			m_numSocketSafeCurrent.SetValueMinMax(6.0, 64.0);
			m_numTotalSafeCurrent.SetValueMinMax(0.0, 3.4028234663852886E+38);
		}
		if (IsPanelVisible)
		{
			m_numAltPeriod.SetProperty(new ICUProperty(SDT.INTEGER16, 8576, 4));
			m_numTotalCurrent.SetProperty(new ICUProperty(SDT.REAL32, 8576, 5));
			m_numSocketSafeCurrent.SetProperty(new ICUProperty(SDT.REAL32, 8576, 6));
			m_numTotalSafeCurrent.SetProperty(new ICUProperty(SDT.REAL32, 8576, 10));
			m_nLastPropertyChangedCounter = -1;
			m_lblInfo = null;
			m_lblInfo = (UIPropertyLabel)AddLabelText("\n", EUILabelType.LargeInfo, "", m_nLeftMargin);
			m_lblInfo.SetVisible(fVisible: false);
			m_nIdsChangingCounter = 5;
			if (m_currentNetwork != null)
			{
				UpdateInfo(refreshAll);
			}
			m_btnInitialize = null;
			m_btnInitialize = AddCustomButton("Initialize", "Initialize this SCN", OnInitializeClicked, rightSide: false);
			m_btnInitialize.Sensitive = true;
			StartUpdateTimer(1000);
		}
	}

	private bool UpdateInfo(bool refreshAll)
	{
		if (m_currentNetwork == null || string.IsNullOrEmpty(m_sSCNName))
		{
			m_lblInfo.SetVisible(fVisible: false);
			return false;
		}
		SCNSocket sCNSocket = null;
		if (m_lstSockets.SelectedRow >= 0)
		{
			sCNSocket = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
		}
		m_lsSockets.Clear();
		int num = 0;
		int row = 0;
		IReadOnlyList<SCNSocket> copyOfSockets = m_currentNetwork.GetCopyOfSockets(m_sSCNName);
		foreach (SCNSocket sock in copyOfSockets)
		{
			ICULanDevice lanDev = m_lanConnection.FindLanDevice(sock.IPAddress);
			if (lanDev == null)
			{
				return false;
			}
			if (m_nLastPropertyChangedCounter != sock.PropChangedCounter)
			{
				long num2 = ((long?)m_currentNetwork.GetFirstSocket(m_sSCNName)?.Id) ?? (-1L);
				if (sock.Id == num2)
				{
					m_numTotalCurrent.SetValue(sock.PropMaximumStaticCurrent);
					m_numSocketSafeCurrent.SetValue(sock.PropSocketSafeCurrent);
					m_numTotalSafeCurrent.SetValue(sock.PropTotalSafeCurrent);
					m_numAltPeriod.SetValue(sock.PropAlternatingPeriod);
					m_numTotalCurrent.GetProperty()?.CommitChange();
					m_numSocketSafeCurrent.GetProperty()?.CommitChange();
					m_numTotalSafeCurrent.GetProperty()?.CommitChange();
					m_numAltPeriod.GetProperty()?.CommitChange();
					m_nLastPropertyChangedCounter = sock.PropChangedCounter;
					if (lanDev.IsLoggedIn)
					{
						lanDev.UpdateCategories("scn");
						m_numTotalSafeCurrent.SetEnable(lanDev.HasProperty(8576, 10));
					}
					else
					{
						EventHandler value = (object s, EventArgs e) =>
						{
							if (Math.Abs(sock.PropTotalSafeCurrent - (double)m_numTotalSafeCurrent.GetValue()) > 0.001 && !lanDev.IsLoggedIn && lanDev.Login().IsLoggedIn)
							{
								lanDev.UpdateCategories("scn");
								bool flag2 = lanDev.HasProperty(8576, 10);
								m_numTotalSafeCurrent.SetEnable(flag2);
								if (!flag2)
								{
									m_numTotalSafeCurrent.SetValue(sock.PropTotalSafeCurrent);
								}
							}
							if (!lanDev.IsLoggedIn)
							{
								m_numTotalSafeCurrent.SetValue(sock.PropTotalSafeCurrent);
							}
						};
						m_numTotalSafeCurrent.Changed += value;
					}
				}
			}
			string text = "";
			switch (sock.State)
			{
			case EChargingState.Alternating:
				text = "Alternating";
				break;
			case EChargingState.Charging:
				text = "Charging";
				break;
			case EChargingState.ChargingIncreaseCurrent:
				text = "Increasing";
				break;
			case EChargingState.ChargingInitializing:
				text = "Initializing";
				break;
			case EChargingState.ChargingProbing:
				text = "Probing";
				break;
			case EChargingState.Empty:
				text = "Empty";
				break;
			case EChargingState.Idle:
				text = "Idle";
				break;
			case EChargingState.Unconnected:
				text = "Unconnected";
				break;
			}
			if (sock.Mode3State != 0)
			{
				text = $"{text} ({sock.Mode3State:X2})";
			}
			if (DateTime.UtcNow - sock.LastUpdate > m_8sTimeSpan)
			{
				text = "[Unconnected]";
			}
			string value2 = "─ " + sock.Name;
			bool flag = false;
			if (copyOfSockets.Count((SCNSocket a) => a.UniqueID == sock.UniqueID && a.Id != sock.Id) > 0)
			{
				flag = true;
				value2 = ((sock.SocketIndex != 0) ? ("└ " + sock.Name) : ("┌ " + sock.Name));
			}
			string value3 = "";
			if (flag)
			{
				value3 = (sock.SocketIndex + 1).ToString();
			}
			num = m_lsSockets.AddRow();
			m_lsSockets.SetValues(num, m_dfId, sock.Id.ToString(), m_dfName, value2, m_dfSock, value3, m_dfState, text, m_dfSocket, sock, m_dfPhaseMapping, sock.PhaseMapping ?? "", m_dfFeederCables, (sock.SocketIndex != 0) ? "" : lanDev?.NumberOfFeederCables.ToString());
			if (sock == sCNSocket)
			{
				row = num;
			}
		}
		if (copyOfSockets.Count > 0)
		{
			m_lstSockets.SelectRow(row);
			return true;
		}
		return false;
	}

	protected override void OnUpdateTick()
	{
		if (m_currentNetwork == null || m_currentUser == null)
		{
			return;
		}
		Application.Invoke(() =>
		{
			if (!m_numAltPeriod.m_isFocused && !m_numTotalCurrent.m_isFocused && !m_numSocketSafeCurrent.m_isFocused && !m_numTotalSafeCurrent.m_isFocused)
			{
				UpdateInfo(refreshAll: false);
			}
			CheckForErrors();
		});
	}

	private void OnDeviceSelectionChanged(object sender, EventArgs e)
	{
		if (m_btnSwap != null && m_lstSockets.SelectedRow >= 0 && m_lstSockets.SelectedRow < m_lsSockets.RowCount)
		{
			SCNSocket value = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
			m_btnSwap.Sensitive = value != null && value.SocketIndex == 0;
		}
	}

	private void OnSwapClicked(object sender, EventArgs e)
	{
		try
		{
			if (m_lstSockets.SelectedRow < 0)
			{
				return;
			}
			SCNSocket value = m_lsSockets.GetValue(m_lstSockets.SelectedRow, m_dfSocket);
			using DlgSCNPhaseMapping dlgSCNPhaseMapping = new DlgSCNPhaseMapping(m_lanConnection.FindLanDevice(value.IPAddress), m_currentNetwork, m_lanConnection);
			if (dlgSCNPhaseMapping.Run(ParentWindow) == Command.Ok)
			{
				UpdateInfo(refreshAll: true);
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		ICULanDevice lanDeviceForFirstSocket = m_currentNetwork.GetLanDeviceForFirstSocket(m_sSCNName, m_lanConnection);
		if (lanDeviceForFirstSocket != null && (lanDeviceForFirstSocket.IsLoggedIn || lanDeviceForFirstSocket.Login().IsLoggedIn))
		{
			lanDeviceForFirstSocket.UpdateCategories("scn");
			lanDeviceForFirstSocket.GetProperty(8576, 4).Value = m_numAltPeriod.GetProperty().Value;
			lanDeviceForFirstSocket.GetProperty(8576, 5).Value = m_numTotalCurrent.GetProperty().Value;
			lanDeviceForFirstSocket.GetProperty(8576, 6).Value = m_numSocketSafeCurrent.GetProperty().Value;
			lanDeviceForFirstSocket.GetProperty(8576, 10).Value = m_numTotalSafeCurrent.GetProperty().Value;
			lanDeviceForFirstSocket.StoreChangedProperties();
		}
		else if (lanDeviceForFirstSocket?.Connection is LANConnection lANConnection)
		{
			lANConnection.CallErrorHandler("Unable to communicate with the device '" + lanDeviceForFirstSocket.Identification + "'. Please check SCN connections.", lanDeviceForFirstSocket);
		}
		UpdateInfo(refreshAll: true);
		return true;
	}

	public override void OnRevertChanges()
	{
		m_numAltPeriod?.GetProperty().Rollback();
		m_numTotalCurrent?.GetProperty().Rollback();
		m_numSocketSafeCurrent?.GetProperty().Rollback();
		m_numTotalSafeCurrent?.GetProperty().Rollback();
		OnUpdateControls(PageID);
		base.OnRevertChanges();
	}
}
