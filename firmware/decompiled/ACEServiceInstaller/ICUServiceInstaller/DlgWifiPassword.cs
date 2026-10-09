using System;
using System.Text.RegularExpressions;
using ICUNetwork;
using ICUServiceInstaller.Enums;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgWifiPassword : Dialog
{
	protected Label m_ssid = new Label();

	protected Label m_message = new Label();

	protected PasswordEntry m_password = new PasswordEntry();

	protected ICULanDevice m_lanDevice;

	private readonly Regex m_passwordValidatorRegex = new Regex("^.{8,64}$");

	public string SelectedSsid => m_ssid.Text;

	public SupportedWifiSecurityType SecurityType { get; private set; }

	public string Password => m_password.Password;

	public DlgWifiPassword(string ssid, SupportedWifiSecurityType securityType, ICULanDevice device)
	{
		m_ssid.Text = ssid;
		SecurityType = securityType;
		m_lanDevice = device;
		Table table = new Table
		{
			BackgroundColor = Colors.White
		};
		FrameBox content = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		Label label = new Label("Please enter the password for SSID " + m_ssid.Text + "\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		label.MinWidth = 370.0;
		table.Add(label, 0, 0, 1, 2);
		Label widget = new Label("Password:");
		table.Add(widget, 0, 3);
		table.Add(m_password, 1, 3, 1, 1, hexpand: true);
		table.Add(m_message, 0, 4, 1, 2, hexpand: true);
		m_message.TextColor = Colors.Red;
		Content = content;
		m_password.KeyPressed += OnTxtKeyPressed;
		DialogButton dialogButton = new DialogButton("Ok");
		dialogButton.Clicked += OnOkClicked;
		Buttons.Add(dialogButton);
		Buttons.Add(new DialogButton(Command.Cancel));
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		if (m_passwordValidatorRegex.IsMatch(m_password.Password))
		{
			m_message.Text = string.Empty;
			Respond(Command.Ok);
		}
		m_message.Text = "Password has to be between 8 to 64 characters";
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return)
		{
			OnOkClicked(null, null);
		}
	}
}
