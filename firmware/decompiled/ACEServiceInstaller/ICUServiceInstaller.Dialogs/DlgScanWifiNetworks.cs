using System;
using System.Collections.Generic;
using System.Linq;
using System.Web.Script.Serialization;
using System.Windows;
using ICUNetwork;
using ICUServiceInstaller.Enums;
using ICUServiceInstaller.UI;
using ICUServiceInstaller.Wifi;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller.Dialogs;

public class DlgScanWifiNetworks : Dialog
{
	private static readonly ILogger Logger = Log.ForContext<DlgScanWifiNetworks>();

	protected ICULanDevice m_currentDevice;

	protected TreeStore m_networkStore;

	protected TreeView m_treeNetworks = new TreeView();

	protected Spinner m_scanSpinner = new Spinner();

	protected DataField<Image> m_dfNetworkImage = new DataField<Image>();

	protected DataField<UIWifiProfile> m_dfNetworkData = new DataField<UIWifiProfile>();

	protected DataField<Image> m_dfNetworkSelection = new DataField<Image>();

	public string Ssid;

	public string Password;

	public SupportedWifiSecurityType SecurityType;

	protected Image m_imgEmpty = Image.FromResource(typeof(App), AppProperties.ResourcePath("empty16.png"));

	public DlgScanWifiNetworks(ICULanDevice currentDevice)
	{
		Width = 450.0;
		m_currentDevice = currentDevice;
		VBox vBox = new VBox
		{
			BackgroundColor = Colors.White
		};
		m_networkStore = new TreeStore(m_dfNetworkImage, m_dfNetworkData, m_dfNetworkSelection);
		m_treeNetworks.DataSource = m_networkStore;
		m_treeNetworks.SelectionMode = SelectionMode.Multiple;
		m_treeNetworks.VerticalPlacement = WidgetPlacement.Center;
		m_treeNetworks.Font = Font.SystemSansSerifFont.WithSize(12.0);
		m_treeNetworks.Columns.Add("1", new ImageCellView(m_dfNetworkImage));
		m_treeNetworks.Columns.Add("2", new TextCellView(m_dfNetworkData));
		m_treeNetworks.Columns.Add("3", new ImageCellView(m_dfNetworkSelection));
		m_treeNetworks.HeadersVisible = false;
		m_treeNetworks.SelectionChanged += OnConnectToWifi;
		m_treeNetworks.WidthRequest = (m_treeNetworks.HeightRequest = 350.0);
		m_treeNetworks.MarginRight = 8.0;
		m_treeNetworks.ExpandHorizontal = true;
		m_treeNetworks.VerticalScrollPolicy = ScrollPolicy.Automatic;
		vBox.PackStart(m_treeNetworks, expand: true);
		HBox hBox = new HBox
		{
			MarginRight = 8.0
		};
		hBox.PackStart(MainWindowBase.AddImageButton("update-arrow.png", "Refresh. Search for SSID's", OnRefreshWifi));
		vBox.PackStart(hBox);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Cancel));
		RefreshWifiList();
	}

	public void OnRefreshWifi(object sender, EventArgs e)
	{
		RefreshWifiList();
	}

	public void OnConnectToWifi(object sender, EventArgs e)
	{
		if (m_treeNetworks.SelectedRow == null || m_treeNetworks.SelectedRows.Length != 1)
		{
			return;
		}
		UIWifiProfile value = m_networkStore.GetNavigatorAt(m_treeNetworks.SelectedRow).GetValue(m_dfNetworkData);
		if (value == null)
		{
			return;
		}
		if (value.WifiSecurityType.GetDescription().StartsWith("Not supported"))
		{
			MessageDialog.ShowError("The security type of this network is not supported.");
			Logger.Debug("Security type with id {SecurityType} is not supported", (int)value.WifiSecurityType);
			return;
		}
		DlgWifiPassword dlgWifiPassword = new DlgWifiPassword(value.Ssid, value.WifiSecurityType, m_currentDevice);
		if (dlgWifiPassword.Run() == Command.Ok)
		{
			Ssid = dlgWifiPassword.SelectedSsid;
			Password = dlgWifiPassword.Password;
			SecurityType = dlgWifiPassword.SecurityType;
			Respond(Command.Ok);
		}
	}

	public void RefreshWifiList()
	{
		//IL_0038: Unknown result type (might be due to invalid IL or missing references)
		//IL_0050: Unknown result type (might be due to invalid IL or missing references)
		(bool, string) tuple = m_currentDevice.ExecutedWifiScan();
		if (tuple.Item1)
		{
			try
			{
				ProcessWifiScanResults(tuple.Item2);
				return;
			}
			catch (Exception exception)
			{
				Logger.Debug(exception, "Failed to process Wi-Fi scan results");
				MessageBox.Show("Failed to process Wi-Fi scan results");
				Respond(Command.Cancel);
				return;
			}
		}
		MessageBox.Show("There was an error executing the scan for Wi-Fi networks.");
		Respond(Command.Cancel);
	}

	private void ProcessWifiScanResults(string jsonData)
	{
		//IL_00ae: Unknown result type (might be due to invalid IL or missing references)
		WifiResult wifiResult = DeserializeWifiResult(jsonData);
		if (wifiResult != null && wifiResult.scan_results.Any())
		{
			List<UIWifiProfile> networks = (from item in wifiResult.scan_results
				orderby item.Ssid
				select new UIWifiProfile(item.SignalStrength)
				{
					WifiSecurityType = item.Security,
					Ssid = item.Ssid,
					SignalStrength = item.SignalStrength
				} into wn
				orderby wn.SignalStrength descending
				select wn).ToList();
			PopulateNetworkStore(networks);
		}
		else
		{
			PopulateNetworkStore(new List<UIWifiProfile>());
			MessageBox.Show("No Wi-Fi networks found.");
		}
	}

	private WifiResult DeserializeWifiResult(string jsonData)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		return new JavaScriptSerializer().Deserialize<WifiResult>(jsonData);
	}

	private void PopulateNetworkStore(List<UIWifiProfile> networks)
	{
		m_networkStore.Clear();
		foreach (UIWifiProfile network in networks)
		{
			m_networkStore.AddNode()?.SetValues(m_dfNetworkImage, network.Icon, m_dfNetworkData, network, m_dfNetworkSelection, m_imgEmpty);
		}
	}
}
