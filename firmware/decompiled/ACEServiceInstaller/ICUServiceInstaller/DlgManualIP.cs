using System;
using System.Collections.Generic;
using System.Configuration;
using System.Net;
using System.Net.NetworkInformation;
using ICUNetwork;
using ICUServiceInstaller.Properties;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgManualIP : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgManualIP>();

	protected TextEntry m_txtIPAddress = new TextEntry();

	protected SpinButton m_spbPort = new SpinButton();

	protected ComboBox m_cmbType = new ComboBox();

	protected SpinButton m_spbNumberOfSockets = new SpinButton();

	protected CheckBox m_chkLogin = new CheckBox();

	protected Button m_btnSearchLinkLocal = new Button();

	public IPAddress IPAddress { get; set; }

	public int Port { get; set; }

	public string Hostname { get; set; }

	public int NumberOfSockets { get; set; }

	public bool LoginRequired { get; set; }

	public DlgManualIP(string ipAddress = "")
	{
		Title = "Manual IP address " + AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinWidth = 300.0
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
		Label label = new Label("Please enter a IP address.");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, 0, 1, 2);
		m_txtIPAddress.PlaceholderText = "xxx.xxx.xxx.xxx";
		m_spbPort.Digits = 0;
		m_spbPort.IncrementValue = 1.0;
		m_spbPort.MinimumValue = 0.0;
		m_spbPort.MaximumValue = 65535.0;
		m_spbPort.Value = 443.0;
		m_spbNumberOfSockets.Digits = 0;
		m_spbNumberOfSockets.IncrementValue = 1.0;
		m_spbNumberOfSockets.MinimumValue = 1.0;
		m_spbNumberOfSockets.MaximumValue = 2.0;
		m_spbNumberOfSockets.Value = 1.0;
		m_cmbType.Items.Clear();
		foreach (ICUDeviceModel value in Enum.GetValues(typeof(ICUDeviceModel)))
		{
			m_cmbType.Items.Add(value, value.ToString());
		}
		m_btnSearchLinkLocal = new Button("Search");
		m_btnSearchLinkLocal.Clicked += OnSearchLinkLocalClicked;
		table.Add(new Label("IP address:"), 0, 1);
		table.Add(m_txtIPAddress, 1, 1, 1, 1, hexpand: true);
		table.Add(m_btnSearchLinkLocal, 2, 1, 1, 1, hexpand: true);
		table.Add(new Label("Port:"), 0, 2);
		table.Add(m_spbPort, 1, 2, 1, 2, hexpand: true);
		table.Add(new Label("Model type:"), 0, 3);
		table.Add(m_cmbType, 1, 3, 1, 2, hexpand: true);
		table.Add(new Label("Number of sockets:"), 0, 4);
		table.Add(m_spbNumberOfSockets, 1, 4, 1, 2, hexpand: true);
		table.Add(new Label("Login required:"), 0, 5);
		table.Add(m_chkLogin, 1, 5, 1, 2, hexpand: true);
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		if (!string.IsNullOrEmpty(Settings.Default.LastManualIPAddress))
		{
			m_txtIPAddress.Text = Settings.Default.LastManualIPAddress;
			m_spbPort.Value = Settings.Default.LastManualIPPort;
			m_spbNumberOfSockets.Value = Settings.Default.LastManualNumberOfSockets;
			m_cmbType.SelectedText = Settings.Default.LastManualModelType;
		}
		m_chkLogin.State = CheckBoxState.On;
		m_txtIPAddress.SetFocus();
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(new DialogButton(Command.Ok));
		m_btnSearchLinkLocal.Sensitive = true;
	}

	private void OnSearchLinkLocalClicked(object sender, EventArgs e)
	{
		Dictionary<IPAddress, PhysicalAddress> allDevicesOnLAN = ArpList.GetAllDevicesOnLAN();
		bool flag = false;
		foreach (KeyValuePair<IPAddress, PhysicalAddress> item in allDevicesOnLAN)
		{
			if (item.Key.GetAddressBytes()[0] == 169 && item.Key.GetAddressBytes()[1] == 254)
			{
				Logger.Debug("LinkLocal IP : {Key} --> MAC {Mac}", item.Key, item.Value);
				if (MessageDialog.AskQuestion($"A new device is found on ip: {item.Key} do you want to add that manually?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
				{
					m_txtIPAddress.Text = item.Key.ToString();
					flag = true;
					break;
				}
			}
		}
		if (!flag)
		{
			MessageDialog.ShowMessage("No device found on the local subnet");
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		NumberOfSockets = 0;
		Hostname = "";
		if (cmd == Command.Ok)
		{
			string text = m_txtIPAddress.Text;
			int num = (int)m_spbPort.Value;
			int num2 = (int)m_spbNumberOfSockets.Value;
			if (!IPAddress.TryParse(text, out IPAddress address))
			{
				MessageDialog.ShowError(this, "Invalid IP addres!", "Please enter an ip address in the following form: 192.168.1.10");
				return;
			}
			IPAddress = address;
			Port = num;
			Hostname = m_cmbType.SelectedItem.ToString();
			NumberOfSockets = num2;
			LoginRequired = m_chkLogin.State == CheckBoxState.On;
			Settings.Default.LastManualIPAddress = address.ToString();
			Settings.Default.LastManualIPPort = num;
			Settings.Default.LastManualNumberOfSockets = num2;
			Settings.Default.LastManualLoginRequired = LoginRequired;
			Settings.Default.LastManualModelType = m_cmbType.SelectedText;
			((SettingsBase)Settings.Default).Save();
			Logger.Information("IP-address entered");
			Respond(Command.Ok);
			Close();
		}
		else
		{
			Respond(Command.Cancel);
			Close();
		}
	}
}
