using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Net;
using System.Threading;
using ICUIWSConnection;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgAddToSCN : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgAddToSCN>();

	protected ComboBox m_cmbDevices = new ComboBox();

	protected TextEntry m_txtName = new TextEntry();

	protected Label m_lblPleasewait = new Label("Please wait.. Searching for Charging stations");

	protected Label m_lblNoDevice = new Label("There are no other Charging Stations found that can be added to this SCN");

	protected Label m_lblInfo = new Label();

	protected bool m_CreateSCN;

	private BackgroundWorker bgwAddDevices;

	private static readonly List<ICUDevice> FoundDevices = new List<ICUDevice>();

	protected ICULanDevice m_lanDevice;

	private const int DEFAULT_LOCK_TIMEOUT = 5000;

	private const int MAX_LENGTH_SCN_NAME = 7;

	private const double defaultTotalCurrent = 32.0;

	private const double defaultSocketSafeCurrent = 6.0;

	private const double defaultTotalSafeCurrent = 32.0;

	private const int defaultAlternatePeriod = 900;

	protected string SCNName { get; set; }

	protected SCNNetwork SCNNetwork { get; set; }

	public DlgAddToSCN(ICULanDevice lanDevice, SCNNetwork scnNetwork)
	{
		Initialize(createSCN: true, "", scnNetwork, lanDevice.Connection as LANConnection, lanDevice);
	}

	public DlgAddToSCN(string scnName, SCNNetwork scnNetwork, LANConnection lanCon)
	{
		Initialize(createSCN: false, scnName, scnNetwork, lanCon, null);
	}

	private void Initialize(bool createSCN, string scnName, SCNNetwork scnNetwork, LANConnection lanCon, ICULanDevice lanDevice)
	{
		m_CreateSCN = createSCN;
		SCNName = scnName;
		SCNNetwork = scnNetwork;
		m_lanDevice = lanDevice;
		if (createSCN && lanDevice != null)
		{
			Title = string.Format("Modify the Smart Charging Network of a device", Array.Empty<object>());
		}
		else
		{
			Title = $"Add Charging Station(s) to '{SCNName}'";
		}
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinWidth = 400.0
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		table.Margin = 8.0;
		bgwAddDevices = new BackgroundWorker();
		bgwAddDevices.DoWork += BgwAddDevices_DoWork;
		bgwAddDevices.RunWorkerCompleted += BgwAddDevices_RunWorkerCompleted;
		int num = 0;
		if (!m_CreateSCN && lanCon != null)
		{
			m_lblPleasewait.Font = m_lblInfo.Font.WithWeight(FontWeight.Semibold);
			m_lblPleasewait.TextColor = Colors.SteelBlue;
			table.Add(m_lblPleasewait, 0, num, 1, 2);
			num++;
			m_lblInfo.Text = "Choose the Charging Station that should be added to this SCN";
			m_lblInfo.Font = m_lblInfo.Font.WithWeight(FontWeight.Semibold);
			m_lblInfo.Visible = false;
			table.Add(m_lblInfo, 0, num, 1, 2);
			num++;
			table.Add(new Label("Charging Station:"), 0, num);
			table.Add(m_cmbDevices, 1, num, 1, 1, hexpand: true);
			m_cmbDevices.Items.Clear();
			m_cmbDevices.Sensitive = false;
			num++;
			m_lblNoDevice.TextColor = Colors.Red;
			m_lblNoDevice.Font = m_lblNoDevice.Font.WithWeight(FontWeight.Semibold);
			table.Add(m_lblNoDevice, 0, num, 1, 2);
			m_lblNoDevice.Visible = false;
			bgwAddDevices.RunWorkerAsync(lanCon);
		}
		if (m_CreateSCN && lanDevice != null)
		{
			lanDevice.UpdateProperties(2203648u, 2204160u);
			ICUProperty property = lanDevice.GetProperty(8608, 0);
			if (property != null && property.Value == null)
			{
				Label label = new Label($"The current firmware of '{lanDevice.Identification}' does not support the SCN feature.\nPlease update your firmware.");
				label.TextColor = Colors.Orange;
				table.Add(label, 0, num, 1, 2, hexpand: true);
				num++;
			}
			else if (!IWSFirmwareFeatures.IsFeatureUnlocked(lanDevice.FirmwareVersionNumber, lanDevice.GetPropertyUInt(8610, 0), IWSFirmwareFeatures.Features.LoadBalancing_SCN, lanDevice.isAHP))
			{
				Label label2 = new Label($"The Smart Charging Network feature is NOT enabled for device '{lanDevice.Identification}'!\nPlease contact your vendor for an upgrade key.");
				label2.TextColor = Colors.Orange;
				table.Add(label2, 0, num, 1, 2, hexpand: true);
				num++;
			}
			else
			{
				table.Add(new Label("Add Charging Station:"), 0, num);
				table.Add(new Label(lanDevice.Identification), 1, num, 1, 1, hexpand: true);
				num++;
				Label label3 = new Label("SCN feature is enabled for this device");
				label3.TextColor = Colors.SteelBlue;
				table.Add(label3, 0, num, 1, 2, hexpand: true);
				num++;
				m_lblInfo.Text = "To new Smart Charging Network:";
				table.Add(m_lblInfo, 0, num);
				m_txtName = new TextEntry();
				m_txtName.WidthRequest = 200.0;
				m_txtName.PlaceholderText = $"SCN network name (max {7} characters)";
				m_txtName.Changed += OnSCNNameChanged;
				m_txtName.KeyPressed += (object o, KeyEventArgs e) =>
				{
					if (e.Key == Key.Space)
					{
						e.Handled = true;
					}
					else if (e.Key == Key.Return)
					{
						OnCommandActivated(Command.Ok);
					}
				};
				m_txtName.SetFocus();
				if (!string.IsNullOrWhiteSpace(m_lanDevice?.SCNNetwork))
				{
					m_txtName.Text = m_lanDevice.SCNNetwork;
				}
				table.Add(m_txtName, 1, num, 1, 1, hexpand: true);
				num++;
				Buttons.Add(new DialogButton(Command.Ok));
			}
		}
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Cancel));
	}

	private bool CompareFirstTwoOctets(IPAddress IPAddr1, IPAddress IPAddr2)
	{
		byte[] addressBytes = IPAddr1.GetAddressBytes();
		byte[] addressBytes2 = IPAddr2.GetAddressBytes();
		if (addressBytes[0] == addressBytes2[0])
		{
			return addressBytes[1] == addressBytes2[1];
		}
		return false;
	}

	private void BgwAddDevices_RunWorkerCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		m_lblPleasewait.Visible = false;
		m_lblInfo.Visible = true;
		if (Monitor.TryEnter(FoundDevices, 5000))
		{
			SCNSocket firstSocket = SCNNetwork.GetFirstSocket(SCNName);
			foreach (ICULanDevice item in FoundDevices.Cast<ICULanDevice>())
			{
				bool flag = firstSocket == null || CompareFirstTwoOctets(firstSocket.IPAddress, item.IPAddress);
				m_cmbDevices.Items.Add(item, string.Format("{0} / {1} / {2} {3}", new object[4]
				{
					item.Identification,
					item.FirmwareVersionNumber,
					item.IPAddress,
					flag ? "" : "/ *"
				}));
			}
			Monitor.Exit(FoundDevices);
		}
		if (m_cmbDevices.Items.Any())
		{
			Buttons.Add(new DialogButton(Command.Ok));
			m_cmbDevices.SelectedIndex = 0;
			m_cmbDevices.Sensitive = true;
		}
		else
		{
			m_cmbDevices.Sensitive = false;
			m_lblNoDevice.Visible = true;
		}
	}

	private void BgwAddDevices_DoWork(object sender, DoWorkEventArgs e)
	{
		if (!(e.Argument is LANConnection lANConnection))
		{
			return;
		}
		List<ICUDevice> source = new List<ICUDevice>(lANConnection.Devices);
		FoundDevices.Clear();
		foreach (ICULanDevice lanDev in from ICULanDevice d in source
			where string.IsNullOrEmpty(d.SCNNetwork)
			select d)
		{
			try
			{
				if (FoundDevices.All((ICUDevice foundDevice) => foundDevice.SerialNumber != lanDev.SerialNumber))
				{
					FoundDevices.Add(lanDev);
				}
			}
			catch (Exception ex)
			{
				Logger.Debug(ex, ex.Message);
			}
		}
	}

	private void OnSCNNameChanged(object sender, EventArgs e)
	{
		if (m_txtName.Text.Trim().Length > 7)
		{
			m_txtName.Text = m_txtName.Text.Substring(0, 7);
			m_txtName.CursorPosition = 7;
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		Logger.Debug("OnCommandActivated");
		if (cmd != Command.Ok)
		{
			Respond(Command.Cancel);
			Close();
			return;
		}
		ICULanDevice iCULanDevice = m_cmbDevices.SelectedItem as ICULanDevice;
		if (m_lanDevice != null)
		{
			iCULanDevice = m_lanDevice;
		}
		if (m_CreateSCN)
		{
			if (string.IsNullOrWhiteSpace(m_txtName.Text))
			{
				MessageDialog.ShowError(this, "The SCN name cannot be empty!\nPlease enter a valid SCN name.");
				return;
			}
			if (m_txtName.Text.Trim().Length > 8)
			{
				MessageDialog.ShowError(this, "The SCN name cannot be longer then 8 characters!\nPlease enter a shorter SCN name.");
				return;
			}
			if (iCULanDevice.Connection.Devices.OfType<ICULanDevice>().Any((ICULanDevice d) => m_txtName.Text.Trim().ToLower() == d.SCNNetwork?.Trim().ToLower()))
			{
				MessageDialog.ShowError(this, "The SCN network name is already in use!\nPlease enter another name.");
				return;
			}
			SCNName = m_txtName.Text.Trim();
		}
		LANConnection lANConnection = (LANConnection)iCULanDevice.Connection;
		Sensitive = false;
		if (!m_CreateSCN && !iCULanDevice.Login().IsLoggedIn)
		{
			Logger.Debug("*** login failed or was cancelled");
			Sensitive = true;
			return;
		}
		iCULanDevice.UpdateCategories("scn");
		iCULanDevice.UpdateProperties(2204160u);
		uint propertyUInt = iCULanDevice.GetPropertyUInt(8610, 0);
		if (!IWSFirmwareFeatures.IsFeatureUnlocked(iCULanDevice.FirmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_SCN, iCULanDevice.isAHP))
		{
			lANConnection.CallErrorHandler("Device has no SCN license.", iCULanDevice);
			Respond(Command.Cancel);
			return;
		}
		string text = $"Are you sure you want to add device '{iCULanDevice.Identification}' to Smart Charging Network '{SCNName}'?";
		if (m_CreateSCN)
		{
			text += "\nThe device will be rebooted.";
		}
		if (MessageDialog.AskQuestion(text, Command.Yes, Command.No, Command.Cancel) != Command.Yes)
		{
			Sensitive = true;
			Respond(Command.Cancel);
			return;
		}
		MainWindow mainWindow = (MainWindow)TransientFor;
		mainWindow.SuspendUpdateTimers(fSuspend: true);
		try
		{
			bool flag = false;
			if ((!m_CreateSCN) ? AddCSToScn(lANConnection, SCNNetwork, iCULanDevice, SCNName) : AddCSToSCNViaMenu(iCULanDevice, SCNName))
			{
				Hide();
				using (DlgReboot dlgReboot = new DlgReboot(iCULanDevice, fAutoStartReboot: true))
				{
					dlgReboot.Run(this);
				}
				iCULanDevice.Logout();
				Sensitive = true;
				Respond(Command.Ok);
				Close();
			}
			else
			{
				Sensitive = true;
			}
		}
		finally
		{
			mainWindow.SuspendUpdateTimers(fSuspend: false);
		}
	}

	private static bool AddCSToScn(LANConnection lanConnection, SCNNetwork scnNetwork, ICULanDevice device, string scnName)
	{
		IReadOnlyList<SCNSocket> copyOfSockets = scnNetwork.GetCopyOfSockets(scnName);
		(uint nextSocketId, bool filledGap) tuple = FindAvailableSocketId(copyOfSockets, device.NumberOfSockets);
		uint item = tuple.nextSocketId;
		bool item2 = tuple.filledGap;
		int num = copyOfSockets.Count() + device.NumberOfSockets;
		if (item2)
		{
			device.UpdateProperties(2195457u, 2195458u, 2195459u);
			ICUProperty property = device.GetProperty(8576, 1);
			ICUProperty property2 = device.GetProperty(8576, 2);
			ICUProperty property3 = device.GetProperty(8576, 3);
			property2.Value = item;
			property.Value = scnName;
			property3.Value = num;
			device.StoreProperties(property, property2, property3);
			return true;
		}
		SCNSocket sCNSocket = copyOfSockets.FirstOrDefault();
		if (sCNSocket == null)
		{
			return false;
		}
		ICULanDevice iCULanDevice = lanConnection.FindLanDevice(sCNSocket.IPAddress);
		if (iCULanDevice == null)
		{
			return false;
		}
		(bool, HttpStatusCode, string) tuple2 = iCULanDevice.Login();
		if (!tuple2.Item1 || iCULanDevice.LoginData.IsLoginCancelled)
		{
			Log.Warning("Login failed for device {Device} with error {Error}", iCULanDevice.Name, tuple2.Item3);
			return false;
		}
		bool flag = false;
		foreach (SCNSocket item3 in copyOfSockets)
		{
			if (item3.SocketIndex == 0)
			{
				ICULanDevice iCULanDevice2 = lanConnection.FindLanDevice(item3.IPAddress);
				if (iCULanDevice2 != null && !iCULanDevice2.Login().IsLoggedIn)
				{
					flag = true;
					break;
				}
			}
		}
		if (!flag)
		{
			device.UpdateProperties(2195457u, 2195458u, 2195459u);
			ICUProperty property4 = device.GetProperty(8576, 1);
			ICUProperty property5 = device.GetProperty(8576, 2);
			ICUProperty property6 = device.GetProperty(8576, 3);
			property5.Value = item;
			property4.Value = scnName;
			property6.Value = num;
			device.StoreProperties(property4, property5, property6);
			iCULanDevice.UpdateCategories("scn");
			ICUProperty property7 = device.GetProperty(8576, 3);
			ICUProperty property8 = iCULanDevice.GetProperty(8576, 4);
			ICUProperty property9 = iCULanDevice.GetProperty(8576, 5);
			ICUProperty property10 = iCULanDevice.GetProperty(8576, 6);
			ICUProperty property11 = iCULanDevice.GetProperty(8576, 10);
			int num2 = Convert.ToInt32(property8.Value);
			double num3 = Convert.ToDouble(property9.Value);
			double num4 = Convert.ToDouble(property10.Value);
			double num5 = Convert.ToDouble(property11.Value);
			iCULanDevice.StoreProperties(property6, property7, property8, property9, property10, property11);
			uint num6 = 0u;
			foreach (SCNSocket item4 in copyOfSockets)
			{
				if (item4.SocketIndex == 0)
				{
					ICULanDevice iCULanDevice3 = lanConnection.FindLanDevice(item4.IPAddress);
					if (iCULanDevice3 != null && iCULanDevice3.IsLoggedIn)
					{
						iCULanDevice3.UpdateCategories("scn");
						ICUProperty property12 = iCULanDevice3.GetProperty(8576, 2);
						property12.Value = num6;
						property7.Value = property6.Value;
						property8.Value = num2;
						property9.Value = num3;
						property10.Value = num4;
						property11.Value = num5;
						iCULanDevice3.StoreProperties(property12, property7, property10, property11, property9, property8);
						num6 += item4.SocketCount;
					}
				}
			}
		}
		iCULanDevice.Logout();
		foreach (SCNSocket item5 in copyOfSockets)
		{
			if (item5.SocketIndex == 0)
			{
				ICULanDevice iCULanDevice4 = lanConnection.FindLanDevice(item5.IPAddress);
				if (iCULanDevice4.IsLoggedIn)
				{
					iCULanDevice4.Logout();
				}
			}
		}
		return !flag;
	}

	private static (uint nextSocketId, bool filledGap) FindAvailableSocketId(IReadOnlyList<SCNSocket> sockets, int numberOfSocketsNeeded)
	{
		int num = sockets.Count();
		if (num == 0)
		{
			return (nextSocketId: 0u, filledGap: false);
		}
		uint num2 = sockets.Select((SCNSocket s) => s.Id).Last();
		uint num3 = 0u;
		uint num4 = 0u;
		for (int num5 = 0; num5 < num; num5++)
		{
			if (num5 > 0)
			{
				num3 = sockets[num5 - 1].Id;
			}
			num4 = sockets[num5].Id;
			if (num4 != num5 && num4 > num5)
			{
				uint num6 = 0u;
				num6 = ((num5 != 0) ? (num4 - num3 - 1) : num4);
				if (numberOfSocketsNeeded <= num6)
				{
					return (nextSocketId: (uint)num5, filledGap: true);
				}
			}
		}
		return (nextSocketId: num2 + 1, filledGap: false);
	}

	private static bool AddCSToSCNViaMenu(ICULanDevice device, string scnName)
	{
		device.UpdateCategories("scn");
		ICUProperty property = device.GetProperty(8576, 1);
		ICUProperty property2 = device.GetProperty(8576, 2);
		ICUProperty property3 = device.GetProperty(8576, 3);
		ICUProperty property4 = device.GetProperty(8576, 4);
		ICUProperty property5 = device.GetProperty(8576, 5);
		ICUProperty property6 = device.GetProperty(8576, 6);
		ICUProperty property7 = device.GetProperty(8576, 10);
		property2.Value = 0;
		property.Value = scnName;
		property3.Value = device.NumberOfSockets;
		property5.Value = 32.0;
		property6.Value = 6.0;
		property7.Value = 32.0;
		property4.Value = 900;
		device.StoreProperties(property, property2, property3, property4, property5, property6, property7);
		return true;
	}
}
