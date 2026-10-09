using System;
using System.Globalization;
using System.Text.RegularExpressions;
using ICUIWSConnection;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgUnlockFeature : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgUnlockFeature>();

	protected ComboBox m_cmbDevices = new ComboBox();

	protected bool m_fUnlocked;

	protected ICULanDevice m_lanDevice;

	protected Label m_lblFeatures = new Label();

	protected TextEntry m_txtID = new TextEntry();

	protected TextEntry m_txtName = new TextEntry();

	protected TextEntry m_txtOrder = new TextEntry();

	protected TextEntry m_txtUnlock = new TextEntry();

	public DlgUnlockFeature(ICULanDevice lanDevice)
	{
		m_lanDevice = lanDevice;
		Title = string.Format("Install Charging Station Features", Array.Empty<object>());
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
		lanDevice.UpdateProperties(2203648u, 2203904u, 2204160u);
		int num = 0;
		bool flag = true;
		ICUProperty property = lanDevice.GetProperty(8609, 0);
		if (property != null && property.Value == null)
		{
			Label widget2 = new Label($"The current firmware of '{lanDevice.Identification}' does not support license keys.\nPlease update your firmware.")
			{
				TextColor = Colors.Orange
			};
			table.Add(widget2, 0, num, 1, 2, hexpand: true);
			flag = false;
		}
		else
		{
			Label widget3 = new Label($"In this dialog you can install new features for Charging Station '{lanDevice.Identification}'.\nInstalling new features requires the correct license key.\nPlease contact your vendor for a license key.")
			{
				TextColor = Colors.SteelBlue
			};
			table.Add(widget3, 0, num, 1, 2, hexpand: true);
			num++;
			if (lanDevice.FirmwareVersionNumber < new Version("3.4.0") && !lanDevice.isAHP)
			{
				ulong propertyUInt = lanDevice.GetPropertyUInt64(8608, 0, 0uL);
				table.Add(new Label("Your unique ID:"), 0, num);
				m_txtID.Text = CodeToString(propertyUInt);
				m_txtID.ReadOnly = true;
				table.Add(m_txtID, 1, num, 1, 1, hexpand: true);
				num++;
			}
			string propertyString = lanDevice.GetPropertyString(8273, 0, 0);
			table.Add(new Label("Serial number:"), 0, num);
			m_txtOrder.Text = propertyString;
			m_txtOrder.ReadOnly = true;
			table.Add(m_txtOrder, 1, num, 1, 1, hexpand: true);
			num++;
			string text = string.Empty;
			ICUProperty property2 = lanDevice.GetProperty(8610, 0);
			if (property2 != null && property2.Value != null)
			{
				text = IWSFirmwareFeatures.GetFeatureTextLong((IWSFirmwareFeatures.Features)Convert.ToUInt32(property2.Value), lanDevice.isAHP, lanDevice.isDC, "\n");
			}
			table.Add(new Label("Current installed features:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start);
			if (string.IsNullOrEmpty(text))
			{
				m_lblFeatures.Text = "<none>";
			}
			else
			{
				m_lblFeatures.Text = text;
			}
			table.Add(m_lblFeatures, 1, num, 1, 1, hexpand: true);
			num++;
			table.Add(new Label("License key:"), 0, num);
			m_txtUnlock.Text = lanDevice.GetPropertyString(8609, 0, 0);
			m_txtUnlock.MinWidth = 250.0;
			table.Add(m_txtUnlock, 1, num, 1, 1, hexpand: true);
			m_txtUnlock.SetFocus();
			num++;
		}
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Cancel));
		if (flag)
		{
			Buttons.Add(new DialogButton(Command.Ok));
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			if (m_lanDevice != null)
			{
				try
				{
					int num = 0;
					string value = "";
					Match match = Regex.Match(m_txtUnlock.Text, "([0-9a-fA-F]+)\\W+([0-9a-fA-F]+)\\W+([0-9a-fA-F]+)\\W+([0-9a-fA-F]+)\\W+([0-9a-fA-F]+)\\W+([0-9a-fA-F]+)");
					if (match.Groups.Count > 6)
					{
						ulong[] array = new ulong[6];
						for (int i = 0; i < 6; i++)
						{
							if (ulong.TryParse(match.Groups[i + 1].Value, NumberStyles.HexNumber, null, out var result))
							{
								array[i] = result;
							}
							else
							{
								num++;
							}
						}
						value = CodeToKey(array);
					}
					else
					{
						num = 1;
					}
					if (num != 0)
					{
						MessageDialog.ShowError(this, "Invalid license key!\nPlease enter a valid license key.");
						return;
					}
					ICUProperty property = m_lanDevice.GetProperty(8609, 0);
					property.Value = value;
					m_lanDevice.StoreProperties(property);
					MessageDialog.ShowMessage(this, "Your new license is activated.\nThe charging station will be rebooted to install any new feature(s)");
				}
				catch (Exception ex)
				{
					Logger.Error(ex, ex.Message);
				}
			}
			Respond(Command.Ok);
			Close();
		}
		else
		{
			Respond(Command.Cancel);
			Close();
		}
	}

	private string CodeToKey(ulong[] codes)
	{
		return string.Format("{0:X4}.{1:X4}.{2:X4}.{3:X4}.{4:X4}.{5:X4}", new object[6]
		{
			codes[0],
			codes[1],
			codes[2],
			codes[3],
			codes[4],
			codes[5]
		});
	}

	private string CodeToString(ulong code)
	{
		return string.Format("{0:X4}.{1:X4}.{2:X4}.{3:X4}", new object[4]
		{
			(code & 0xFFFF000000000000uL) >> 48,
			(code & 0xFFFF00000000L) >> 32,
			(code & 0xFFFF0000u) >> 16,
			code & 0xFFFF
		});
	}
}
