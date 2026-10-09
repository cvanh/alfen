using System;
using System.Text.RegularExpressions;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

internal class DlgEndUserPin : Dialog
{
	private ICULanDevice m_lanDevice;

	private PasswordEntry m_txtNewEndUserPin;

	private PasswordEntry m_txtVerifyEndUserPin;

	private CheckBox m_chkAllowEmptyPin;

	private readonly Label m_txtErrorMessage = new Label();

	public string Pin { get; set; }

	public DlgEndUserPin(ICULanDevice lanDevice)
	{
		m_txtErrorMessage.Opacity = 0.0;
		m_txtErrorMessage.TextColor = Colors.Red;
		m_lanDevice = lanDevice;
		Title = "Eve Connect app access";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
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
		table.Margin = 8.0;
		m_txtNewEndUserPin = new PasswordEntry();
		m_txtNewEndUserPin.MinWidth = 50.0;
		m_txtVerifyEndUserPin = new PasswordEntry();
		m_txtVerifyEndUserPin.MinWidth = 50.0;
		Label label = new Label("Configure Eve Connect app access\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		label.MinWidth = 300.0;
		table.Add(label, 0, 0, 1, 2);
		Label widget = new Label("Charging Station identity: " + m_lanDevice.Identity + " (Serial number: " + m_lanDevice.SerialNumber + ")\n")
		{
			TextColor = Colors.Gray
		};
		table.Add(widget, 0, 1, 1, 2);
		Label widget2 = new Label("The PIN gives access to the charging station from the Eve Connect app.\nThis is not recommended for publicly accessible chargers.\n");
		table.Add(widget2, 0, 2, 1, 2);
		table.Add(new Label("New PIN:"), 0, 3);
		table.Add(m_txtNewEndUserPin, 1, 3, 1, 1, hexpand: true);
		table.Add(new Label("Verify PIN:"), 0, 4);
		table.Add(m_txtVerifyEndUserPin, 1, 4, 1, 1, hexpand: true);
		table.Add(new Label("Allow empty PIN"), 0, 5);
		table.Add(m_chkAllowEmptyPin = new CheckBox(), 1, 5);
		m_chkAllowEmptyPin.Clicked += OnChkAllowEmptyPin_Clicked;
		table.Add(m_txtErrorMessage, 1, 6);
		table.Add(new Label("The PIN has to be between 4 and 6 digits long.\n")
		{
			TextColor = Colors.Gray
		}, 1, 7, 1, 1, hexpand: true);
		Content = content;
		m_txtNewEndUserPin.SetFocus();
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		DialogButton dialogButton = new DialogButton("Ok");
		dialogButton.Clicked += OnOkClicked;
		DialogButton dialogButton2 = new DialogButton("Disable");
		dialogButton2.Clicked += OnDisableCLicked;
		Buttons.Add(dialogButton2);
		Buttons.Add(dialogButton);
		Buttons.Add(new DialogButton(Command.Cancel));
	}

	private void OnDisableCLicked(object sender, EventArgs e)
	{
		Respond(Command.Remove);
	}

	private void OnChkAllowEmptyPin_Clicked(object sender, EventArgs e)
	{
		if (m_chkAllowEmptyPin.State == CheckBoxState.On)
		{
			m_txtNewEndUserPin.Password = string.Empty;
			m_txtVerifyEndUserPin.Password = string.Empty;
		}
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		if (m_chkAllowEmptyPin.State == CheckBoxState.On)
		{
			Pin = string.Empty;
			Respond(Command.Ok);
			return;
		}
		string password = m_txtNewEndUserPin.Password;
		string password2 = m_txtVerifyEndUserPin.Password;
		if (!new Regex("^[0-9]{4,6}$").Match(password).Success)
		{
			m_txtErrorMessage.Opacity = 1.0;
			m_txtErrorMessage.Text = "Use 4 to 6 digits!";
			m_txtNewEndUserPin.SetFocus();
		}
		else if (password.Length < 4)
		{
			m_txtErrorMessage.Opacity = 1.0;
			m_txtErrorMessage.Text = "The PIN is too short, it needs to be at least 4 digits!";
			m_txtNewEndUserPin.SetFocus();
		}
		else if (password.Length > 6)
		{
			m_txtErrorMessage.Opacity = 1.0;
			m_txtErrorMessage.Text = "The PIN is too long, the maximum length is 6 digits!";
			m_txtNewEndUserPin.SetFocus();
		}
		else if (password != password2)
		{
			m_txtErrorMessage.Opacity = 1.0;
			m_txtErrorMessage.Text = "The PINs do not match!";
			m_txtVerifyEndUserPin.SetFocus();
		}
		else
		{
			Pin = password;
			Respond(Command.Ok);
		}
	}
}
