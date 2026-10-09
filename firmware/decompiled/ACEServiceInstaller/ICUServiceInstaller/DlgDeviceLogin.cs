using System;
using System.Configuration;
using System.Diagnostics;
using System.Net;
using System.Security.Authentication;
using System.Threading;
using System.Threading.Tasks;
using System.Windows.Threading;
using ICUNetwork;
using ICUServiceInstaller.Properties;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgDeviceLogin : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgDeviceLogin>();

	private bool m_isPasswordChanged;

	private readonly ComboBox m_cmbUserName = new ComboBox();

	private readonly ICULanDevice m_lanDevice;

	private readonly PasswordEntry m_txtPassword;

	private readonly DialogButton m_btnOK;

	private readonly DialogButton m_btnCancel;

	private readonly DialogButton m_btnResetPassword;

	private readonly CheckBox m_chkStorePassword;

	protected ProgressBar m_prbLogin = new ProgressBar();

	public ICUUser User { get; set; }

	public string Username => m_cmbUserName.SelectedItem.ToString();

	public string Password => m_txtPassword.Password;

	public DlgDeviceLogin(ICULanDevice lanDevice)
	{
		ValidateStoredPassword();
		m_lanDevice = lanDevice;
		Title = AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
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
		m_cmbUserName.Items.Add("admin", "Owner");
		m_cmbUserName.Items.Add("temp", "Temporary");
		m_cmbUserName.Items.Add("service", "Secure Service Access");
		m_cmbUserName.SelectedItem = "admin";
		m_cmbUserName.SelectionChanged += (object o, EventArgs e) =>
		{
			if (m_cmbUserName.SelectedItem.ToString() == "service")
			{
				m_chkStorePassword.State = CheckBoxState.Off;
				m_chkStorePassword.Visible = false;
			}
			else
			{
				m_chkStorePassword.Visible = true;
			}
		};
		m_cmbUserName.MinWidth = 290.0;
		m_txtPassword = new PasswordEntry();
		m_txtPassword.MinWidth = 290.0;
		Label label = new Label("Please select the user level and enter the password to login\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		label.MinWidth = 300.0;
		table.Add(label, 0, 0, 1, 2);
		table.Add(new Label("Charging Station identity: " + m_lanDevice.Identity + " (Serial number: " + m_lanDevice.SerialNumber + ")\n")
		{
			TextColor = Colors.Gray
		}, 0, 1, 1, 2);
		Label widget = new Label("User level:");
		Label widget2 = new Label("Password:");
		table.Add(widget, 0, 2);
		table.Add(m_cmbUserName, 1, 2, 1, 1, hexpand: true);
		table.Add(widget2, 0, 3);
		table.Add(m_txtPassword, 1, 3, 1, 1, hexpand: true);
		m_chkStorePassword = new CheckBox("Apply if all chargers have the same password.\r\nThe last used user level and password will be used for the next device you log into.");
		table.Add(m_chkStorePassword, 0, 7, 1, 2);
		m_prbLogin.Hide();
		m_prbLogin.MinHeight = 16.0;
		table.Add(m_prbLogin, 0, 8, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, -1.0, UIPropertyBase.s_marginHorMax);
		Content = content;
		m_cmbUserName.KeyPressed += OnTxtKeyPressed;
		m_txtPassword.KeyPressed += OnTxtKeyPressed;
		m_txtPassword.MouseEntered += (object sender, EventArgs e) =>
		{
			if (m_isPasswordChanged)
			{
				m_txtPassword.TooltipText = m_txtPassword.Password;
			}
		};
		m_txtPassword.MouseExited += (object sender, EventArgs e) =>
		{
			m_txtPassword.TooltipText = null;
		};
		m_chkStorePassword.KeyPressed += OnTxtKeyPressed;
		m_cmbUserName.SetFocus();
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		m_btnOK = new DialogButton("Ok");
		m_btnOK.Clicked += OnOkClicked;
		if (Settings.Default.StorePasswords && m_cmbUserName.SelectedItem.ToString() != "service")
		{
			m_chkStorePassword.State = CheckBoxState.On;
			if (!string.IsNullOrEmpty(Settings.Default.LastDeviceUsername))
			{
				m_cmbUserName.SelectedItem = Settings.Default.LastDeviceUsername;
			}
			m_txtPassword.Password = Settings.Default.LastDevicePassword;
		}
		else
		{
			m_chkStorePassword.State = CheckBoxState.Off;
		}
		m_btnResetPassword = new DialogButton("Forgot password");
		m_btnResetPassword.Clicked += OnResetPassword;
		Buttons.Add(m_btnResetPassword);
		m_btnCancel = new DialogButton(Command.Cancel);
		Buttons.Add(m_btnCancel);
		Buttons.Add(m_btnOK);
		if (!string.IsNullOrEmpty(lanDevice.LoginData.Username) && !string.IsNullOrEmpty(lanDevice.LoginData.Password))
		{
			Logger.Debug("Login automatically using {UserName} {Password}", lanDevice.LoginData.Username, lanDevice.LoginData.Password);
			m_cmbUserName.SelectedItem = lanDevice.LoginData.Username;
			m_txtPassword.Password = lanDevice.LoginData.Password;
			Shown += (object a, EventArgs b) =>
			{
				OnOkClicked(null, null);
			};
		}
		m_txtPassword.Changed += (object sender, EventArgs e) =>
		{
			m_isPasswordChanged = true;
		};
		m_txtPassword.KeyPressed += (object sender, KeyEventArgs e) =>
		{
			if (!m_isPasswordChanged)
			{
				m_txtPassword.Password = string.Empty;
			}
		};
	}

	public static bool ValidateStoredPassword()
	{
		if ((Settings.Default.LastPasswordStoreTime - DateTime.Now).TotalHours > 24.0)
		{
			Settings.Default.LastDeviceUsername = "admin";
			Settings.Default.LastDevicePassword = "";
			((SettingsBase)Settings.Default).Save();
			return false;
		}
		return Settings.Default.StorePasswords;
	}

	private void OnResetPassword(object sender, EventArgs e)
	{
		DlgResetPassword dlgResetPassword = new DlgResetPassword(m_lanDevice);
		if (dlgResetPassword.Run(this) == Command.Ok)
		{
			(bool, string) tuple = m_lanDevice.ResetPassword(dlgResetPassword.PRC);
			if (tuple.Item1)
			{
				MessageDialog.ShowMessage(this, tuple.Item2);
			}
			else
			{
				MessageDialog.ShowError(this, tuple.Item2);
			}
		}
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return)
		{
			OnOkClicked(null, null);
		}
	}

	private async Task TimeOutProgressBarTask(CancellationToken token, int timeoutInSeconds)
	{
		double totalSteps = timeoutInSeconds * 10;
		int i;
		for (i = 0; (double)i <= totalSteps; i++)
		{
			if (token.IsCancellationRequested)
			{
				break;
			}
			await Dispatcher.CurrentDispatcher.InvokeAsync<double>((Func<double>)(() => m_prbLogin.Fraction = (double)i / totalSteps));
			await Task.Delay(100);
		}
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		CancellationTokenSource progressBarCancellationTokenSource = new CancellationTokenSource(TimeSpan.FromSeconds((double)m_lanDevice.LoginTimeoutInSeconds));
		try
		{
			ManualResetEvent loginTaskFinishedEvent = new ManualResetEvent(initialState: false);
			m_prbLogin.Show();
			m_txtPassword.Sensitive = false;
			m_cmbUserName.Sensitive = false;
			m_btnResetPassword.Sensitive = false;
			m_btnOK.Sensitive = false;
			m_btnCancel.Sensitive = false;
			m_lanDevice.LoginData.Username = Username;
			m_lanDevice.LoginData.Password = Password;
			TimeOutProgressBarTask(progressBarCancellationTokenSource.Token, m_lanDevice.LoginTimeoutInSeconds);
			Task<(bool, HttpStatusCode, string)> task = Task.Run(async () =>
			{
				_ = 1;
				try
				{
					Stopwatch sw = Stopwatch.StartNew();
					(bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content) response = await m_lanDevice.LoginRequest();
					sw.Stop();
					if (response.HttpStatusCode != HttpStatusCode.OK)
					{
						await m_lanDevice.HandleUnsuccessfulLoginRequest(response.HttpStatusCode, response.Content, DateTime.UtcNow, m_lanDevice.LoginTimeoutInSeconds * 1000, suppressPopups: false, isUniquePasswordRequired: true, (string message, ICULanDevice device) =>
						{
							(m_lanDevice.Connection as LANConnection).CallErrorHandler(message, device);
						});
					}
					else
					{
						LogSuccessfulLogin(sw.ElapsedMilliseconds);
					}
					return response;
				}
				catch (Exception innerException)
				{
					throw new AuthenticationException("Login failed", innerException);
				}
				finally
				{
					progressBarCancellationTokenSource.Cancel();
					loginTaskFinishedEvent.Set();
				}
			});
			while (!loginTaskFinishedEvent.WaitOne(100))
			{
				try
				{
					Application.MainLoop.DispatchPendingEvents();
				}
				catch (Exception ex)
				{
					Logger.Debug(ex, ex.Message);
				}
			}
			(bool, HttpStatusCode, string) result = task.GetAwaiter().GetResult();
			if (!result.Item1)
			{
				m_prbLogin.Hide();
				m_prbLogin.Fraction = 0.0;
				m_btnResetPassword.Sensitive = true;
				m_txtPassword.Sensitive = true;
				m_cmbUserName.Sensitive = true;
				m_btnOK.Sensitive = true;
				m_btnCancel.Sensitive = true;
				m_lanDevice.LoginData.IsLoggedIn = false;
				m_lanDevice.LoginData.LoginError = result.Item3;
				return;
			}
			if (m_chkStorePassword.State == CheckBoxState.On)
			{
				Settings.Default.LastDeviceUsername = Username;
				Settings.Default.LastDevicePassword = Password;
			}
			else
			{
				Settings.Default.LastDeviceUsername = "admin";
				Settings.Default.LastDevicePassword = "";
			}
			Settings.Default.LastPasswordStoreTime = DateTime.Now;
			Settings.Default.StorePasswords = m_chkStorePassword.State == CheckBoxState.On;
			((SettingsBase)Settings.Default).Save();
			m_lanDevice.LoginData.LastHttpStatusCode = result.Item2;
			m_lanDevice.LoginData.LoginError = string.Empty;
			m_lanDevice.LoginData.IsLoggedIn = true;
			Respond(Command.Ok);
		}
		catch (Exception ex2)
		{
			Logger.Debug(ex2, "Login failed");
			m_lanDevice.LoginData.IsLoggedIn = false;
			m_lanDevice.LoginData.LoginError = ex2.Message;
			Respond(Command.Cancel);
		}
		finally
		{
			progressBarCancellationTokenSource.Dispose();
		}
	}

	private void LogSuccessfulLogin(long milliSeconds)
	{
		if (!m_lanDevice.HasSuccessfulLogin)
		{
			m_lanDevice.UpdateAnalyticsProperties();
			Logger.AddChargerContext(m_lanDevice).Information("Login successfull, time: {Millis}", milliSeconds);
			m_lanDevice.HasSuccessfulLogin = true;
		}
	}
}
