using System;
using System.Collections.Specialized;
using System.Configuration;
using System.Diagnostics;
using System.Linq;
using System.Reflection;
using System.Security.Cryptography;
using System.Text;
using System.Web.Script.Serialization;
using ICUServiceInstaller.Properties;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgLogon : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgLogon>();

	private readonly TextEntry m_txtUsername;

	private readonly PasswordEntry m_txtPassword;

	private readonly Label m_lblNewPassword;

	private readonly Label m_lblNewPasswordConfirm;

	private readonly PasswordEntry m_txtNewPassword;

	private readonly PasswordEntry m_txtNewPasswordConfirm;

	private readonly ICUConfig m_icuConfig;

	private readonly Button m_btnChange;

	public ICUUser User { get; set; }

	public DlgLogon(ICUConfig icuConfig)
	{
		m_icuConfig = icuConfig;
		Title = AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		Table table = new Table
		{
			BackgroundColor = Colors.White
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
		m_txtUsername = new TextEntry();
		m_txtUsername.MinWidth = 150.0;
		m_txtPassword = new PasswordEntry();
		m_txtPassword.MinWidth = 150.0;
		m_txtNewPassword = new PasswordEntry();
		m_txtNewPasswordConfirm = new PasswordEntry();
		Label label = new Label("Welcome to the ACE Service Installer.\nPlease enter your user name and password below.\n\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, 0, 1, 2);
		Label widget2 = new Label("User name:");
		Label widget3 = new Label("Password:");
		table.Add(widget2, 0, 1);
		table.Add(m_txtUsername, 1, 1, 1, 1, hexpand: true);
		table.Add(widget3, 0, 2);
		table.Add(m_txtPassword, 1, 2, 1, 1, hexpand: true);
		m_lblNewPassword = new Label("New password:");
		m_lblNewPasswordConfirm = new Label("Confirm new password:");
		table.Add(m_lblNewPassword, 0, 3);
		table.Add(m_txtNewPassword, 1, 3);
		table.Add(m_lblNewPasswordConfirm, 0, 4);
		table.Add(m_txtNewPasswordConfirm, 1, 4);
		m_btnChange = new Button("Change password");
		m_btnChange.Style = ButtonStyle.Flat;
		m_btnChange.BackgroundColor = Colors.Transparent;
		m_btnChange.Clicked += OnChangePassword;
		table.Add(m_btnChange, 1, 5);
		string text = FileVersionInfo.GetVersionInfo(Assembly.GetExecutingAssembly().Location).FileVersion;
		string[] array = text.Split(new char[1] { '.' });
		if (array.Count() >= 4)
		{
			text = string.Format("{0}.{1}.{2}-{3}", new object[4]
			{
				array[0],
				array[1],
				array[2],
				array[3]
			});
		}
		table.Add(new Label($"Settings version: {m_icuConfig.Version}\nApplication version: {text}")
		{
			TextColor = Colors.Gray,
			MinHeight = 32.0
		}, 0, 6, 1, 2, hexpand: false, vexpand: true, WidgetPlacement.Start, WidgetPlacement.End);
		ShowChangePassword(fShow: false);
		Button button = new Button(Image.FromResource(typeof(App), AppProperties.ResourcePath("padlock.png")));
		button.Style = ButtonStyle.Borderless;
		button.BackgroundColor = Colors.Transparent;
		button.MinHeight = 32.0;
		button.TooltipText = "Change password";
		button.Clicked += OnBtnChangePassClicked;
		table.Add(button, 1, 6, 1, 1, hexpand: false, vexpand: true, WidgetPlacement.End, WidgetPlacement.End);
		HBox hBox = new HBox();
		ImageView widget4 = new ImageView(Image.FromResource(typeof(App), AppProperties.ResourcePath("Banner.png")));
		hBox.PackStart(widget4);
		hBox.PackEnd(widget, expand: true);
		Content = hBox;
		m_txtUsername.KeyPressed += OnTxtKeyPressed;
		m_txtPassword.KeyPressed += OnTxtKeyPressed;
		m_txtUsername.Text = Settings.Default.LastUserName;
		if (string.IsNullOrEmpty(m_txtUsername.Text))
		{
			m_txtUsername.Text = AppProperties.DefaultUserName;
		}
		m_txtPassword.SetFocus();
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		DialogButton dialogButton = new DialogButton("Ok");
		dialogButton.Clicked += OnOkClicked;
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(dialogButton);
	}

	private void OnChangePassword(object sender, EventArgs e)
	{
		ICUUser iCUUser = ValidatePassword();
		if (iCUUser == null)
		{
			return;
		}
		using SHA256 sHA = SHA256.Create();
		if (m_txtNewPassword.Password != m_txtNewPasswordConfirm.Password)
		{
			MessageDialog.ShowError(this, "The new password and the confirm-new-passwords do not match!");
			return;
		}
		if (Settings.Default.LocalPasswords == null)
		{
			Settings.Default.LocalPasswords = new StringCollection();
		}
		string text = m_txtUsername.Text;
		string text2 = Convert.ToBase64String(sHA.ComputeHash(Encoding.UTF8.GetBytes(text.ToLowerInvariant())));
		for (int i = 0; i < Settings.Default.LocalPasswords.Count; i++)
		{
			string text3 = Settings.Default.LocalPasswords[i];
			string[] array = text3.Split(new char[1] { ';' });
			if (array.Count() > 2 && array[0] == text2)
			{
				Settings.Default.LocalPasswords.Remove(text3);
				break;
			}
		}
		string password = m_txtNewPassword.Password;
		RNGCryptoServiceProvider rng = new RNGCryptoServiceProvider();
		string httpData = iCUUser.Group.HTTPUser + ":" + iCUUser.Group.HTTPPassword;
		ICUUser iCUUser2 = ICUUser.CreateHashedUser(sHA, rng, text.ToLowerInvariant(), password, iCUUser.Group, iCUUser.Company, httpData);
		Settings.Default.LocalPasswords.Add(iCUUser2.User + ";" + iCUUser2.Json);
		((SettingsBase)Settings.Default).Save();
		MessageDialog.ShowMessage(this, "Your password has been changed!", "Note that it is only changed on this PC\nAnd that your original password still works.");
		ShowChangePassword(fShow: false);
		m_txtPassword.Password = m_txtNewPassword.Password;
		m_txtPassword.SetFocus();
	}

	private void ShowChangePassword(bool fShow)
	{
		m_lblNewPassword.Visible = fShow;
		m_txtNewPassword.Visible = fShow;
		m_lblNewPasswordConfirm.Visible = fShow;
		m_txtNewPasswordConfirm.Visible = fShow;
		m_btnChange.Visible = fShow;
	}

	private void OnBtnChangePassClicked(object sender, EventArgs e)
	{
		ShowChangePassword(!m_txtNewPassword.Visible);
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return)
		{
			OnOkClicked(null, null);
		}
	}

	private ICUUser ValidatePassword()
	{
		//IL_00ff: Unknown result type (might be due to invalid IL or missing references)
		using (SHA256 sHA = SHA256.Create())
		{
			string text = m_txtUsername.Text;
			string userNameHashed = Convert.ToBase64String(sHA.ComputeHash(Encoding.UTF8.GetBytes(text.ToLowerInvariant())));
			ICUUser iCUUser = m_icuConfig.Users.FirstOrDefault((ICUUser a) => a.User == userNameHashed);
			if (iCUUser == null)
			{
				MessageDialog.ShowWarning(this, "Unknown user name.\nPlease enter a valid user name.");
				m_txtUsername.SetFocus();
			}
			else
			{
				string password = m_txtPassword.Password;
				if (iCUUser.ValidateHashedPassword(sHA, password))
				{
					return CreateNewUser(iCUUser, password);
				}
				if (Settings.Default.LocalPasswords != null)
				{
					StringEnumerator enumerator = Settings.Default.LocalPasswords.GetEnumerator();
					try
					{
						while (enumerator.MoveNext())
						{
							string current = enumerator.Current;
							try
							{
								string[] array = current.Split(new char[1] { ';' });
								if (array.Count() > 1 && array[0] == userNameHashed)
								{
									dynamic val = new JavaScriptSerializer().DeserializeObject(array[1]);
									ICUUser iCUUser2 = new ICUUser(m_icuConfig.Groups.ToList(), val);
									if (iCUUser2.ValidateHashedPassword(sHA, password))
									{
										return CreateNewUser(iCUUser2, password);
									}
								}
							}
							catch (Exception ex)
							{
								Logger.Error(ex, ex.Message);
								return null;
							}
						}
					}
					finally
					{
						if (enumerator is IDisposable disposable)
						{
							disposable.Dispose();
						}
					}
				}
				MessageDialog.ShowWarning(this, "Incorrect password.\nPlease enter the correct password.");
				m_txtPassword.SetFocus();
			}
		}
		User = null;
		return null;
	}

	private ICUUser CreateNewUser(ICUUser originalUser, string userPassword)
	{
		ICUGroup iCUGroup = new ICUGroup(originalUser.Group.Features.Select((ICUFeatureRight a) => a.Feature).ToList(), originalUser.Group.Name);
		foreach (ICUFeatureRight feature in iCUGroup.Features)
		{
			feature.Rights = originalUser.GetRights(feature.Feature.ID);
		}
		string[] array = originalUser.Password.Split(new char[1] { ':' });
		if (array.Length == 3)
		{
			string passPhrase = array[1] + userPassword;
			ICUUser iCUUser = new ICUUser();
			iCUUser.Group = iCUGroup;
			iCUUser.User = originalUser.User;
			iCUUser.Password = originalUser.Password;
			iCUUser.Fullname = new EncryptDecrypt().Decrypt(originalUser.Fullname, passPhrase);
			string text = new EncryptDecrypt().Decrypt(originalUser.Comment, passPhrase);
			iCUUser.Company = new EncryptDecrypt().Decrypt(originalUser.Company, passPhrase);
			string[] array2 = text.Split(new char[1] { ':' });
			if (array2.Length == 2)
			{
				iCUUser.Group.HTTPUser = array2[0];
				iCUUser.Group.HTTPPassword = array2[1];
			}
			User = iCUUser;
			return iCUUser;
		}
		return null;
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		ICUUser iCUUser = ValidatePassword();
		if (iCUUser != null)
		{
			Logger.Information("User logged in, group: {Group}, generic: {GenericUser}", iCUUser.Group.Name, m_txtUsername.Text.ToLowerInvariant().Equals("post"));
			Settings.Default.LastUserName = m_txtUsername.Text;
			((SettingsBase)Settings.Default).Save();
			Respond(Command.Ok);
		}
	}
}
