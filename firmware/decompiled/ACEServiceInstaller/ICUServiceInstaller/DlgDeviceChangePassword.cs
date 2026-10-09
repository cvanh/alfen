using System;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgDeviceChangePassword : Dialog
{
	private readonly ICULanDevice m_lanDevice;

	private readonly PasswordEntry m_txtNewPassword1;

	private readonly PasswordEntry m_txtNewPassword2;

	public string NewPassword { get; set; }

	public ICUUser User { get; set; }

	public DlgDeviceChangePassword(ICULanDevice lanDevice)
	{
		m_lanDevice = lanDevice;
		Title = "Change current password";
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
		m_txtNewPassword1 = new PasswordEntry();
		m_txtNewPassword1.MinWidth = 290.0;
		m_txtNewPassword2 = new PasswordEntry();
		m_txtNewPassword2.MinWidth = 290.0;
		Label label = new Label("Enter the password to change the owner password\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		label.MinWidth = 300.0;
		table.Add(label, 0, 0, 1, 2);
		table.Add(new Label("Charging Station identity: " + m_lanDevice.Identity + " (Serial number: " + m_lanDevice.SerialNumber + ")\n")
		{
			TextColor = Colors.Gray
		}, 0, 1, 1, 2);
		table.Add(new Label("Tip: By using the same password for all Charging Stations in a Smart Charging Network,\nAlfen Service Installer will automatically login to connected Charging Stations.\nThis comes with a degree of security risk.\n")
		{
			TextColor = Colors.Gray
		}, 0, 2, 1, 2);
		table.Add(new Label("New password:"), 0, 3);
		table.Add(m_txtNewPassword1, 1, 3, 1, 1, hexpand: true);
		table.Add(new Label("Verify new password:"), 0, 4);
		table.Add(m_txtNewPassword2, 1, 4, 1, 1, hexpand: true);
		table.Add(new Label("The password must be between 10 and 40 characters long.\nCharacters '\\', '\"' and ',' are not allowed.\n")
		{
			TextColor = Colors.Gray
		}, 1, 5, 1, 1, hexpand: true);
		Content = content;
		m_txtNewPassword1.SetFocus();
		m_txtNewPassword2.KeyPressed += OnTxtKeyPressed;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		DialogButton dialogButton = new DialogButton("Ok");
		dialogButton.Clicked += OnOkClicked;
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(dialogButton);
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		string password = m_txtNewPassword1.Password;
		string password2 = m_txtNewPassword2.Password;
		if (password.Length < 10)
		{
			MessageDialog.ShowWarning(this, "The new password is too short, it needs to be at least 10 characters. Please enter another password.");
			m_txtNewPassword1.SetFocus();
		}
		else if (password.Length > 40)
		{
			MessageDialog.ShowWarning(this, "The new password is too long, the maximum length is 40 characters. Please enter another password.");
			m_txtNewPassword1.SetFocus();
		}
		else if (password.IndexOfAny(new char[3] { '\\', '"', ',' }) != -1)
		{
			MessageDialog.ShowWarning(this, "The new password contains an invalid character, the characters '\\', '\"' and ',' are not allowed. Please enter another password.");
			m_txtNewPassword1.SetFocus();
		}
		else if (password != password2)
		{
			MessageDialog.ShowWarning(this, "The new passwords do not match! Please ensure that the new passwords are the same.");
			m_txtNewPassword2.SetFocus();
		}
		else
		{
			NewPassword = password;
			Respond(Command.Ok);
		}
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return)
		{
			e.Handled = true;
			OnOkClicked(null, null);
		}
	}
}
