using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgDeviceTempPassword : Dialog
{
	private readonly PasswordEntry m_txtNewPassword1;

	private readonly PasswordEntry m_txtNewPassword2;

	private readonly ComboBox m_cmbExpiration = new ComboBox();

	public string NewPassword { get; set; }

	public uint ExpirationTime => Convert.ToUInt32(m_cmbExpiration.SelectedItem);

	public DlgDeviceTempPassword(ICULanDevice lanDevice)
	{
		Title = "Set temporary password";
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
		Label label = new Label("Set an expiration time for the temporary password. \nAfter the set time the password will be reset automatically.\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		m_cmbExpiration.Items.Add(1, "1 hour");
		m_cmbExpiration.Items.Add(2, "2 hours");
		m_cmbExpiration.Items.Add(3, "3 hours");
		m_cmbExpiration.Items.Add(4, "4 hours");
		m_cmbExpiration.Items.Add(8, "8 hours");
		m_cmbExpiration.Items.Add(12, "12 hours");
		m_cmbExpiration.Items.Add(24, "24 hours");
		m_cmbExpiration.Items.Add(48, "48 hours");
		m_cmbExpiration.Items.Add(72, "72 hours");
		m_cmbExpiration.SelectedItem = 24;
		table.Add(label, 0, 1, 1, 2);
		table.Add(new Label("New temporary password:"), 0, 2);
		table.Add(m_txtNewPassword1, 1, 2, 1, 1, hexpand: true);
		table.Add(new Label("Verify new temporary password:"), 0, 3);
		table.Add(m_txtNewPassword2, 1, 3, 1, 1, hexpand: true);
		table.Add(new Label("Expiration time:"), 0, 4);
		table.Add(m_cmbExpiration, 1, 4, 1, 1, hexpand: true);
		table.Add(new Label("The password must be between 10 and 40 characters long.\nCharacters '\\', '\"' and ',' are not allowed.\n")
		{
			TextColor = Colors.Gray
		}, 1, 5, 1, 1, hexpand: true);
		Content = content;
		m_txtNewPassword1.SetFocus();
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
}
