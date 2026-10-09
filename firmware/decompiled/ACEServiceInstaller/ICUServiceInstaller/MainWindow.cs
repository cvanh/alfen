using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Collections.Specialized;
using System.Configuration;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net;
using System.Reflection;
using System.Threading;
using System.Threading.Tasks;
using System.Timers;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Properties;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Serilog.Context;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class MainWindow : MainWindowBase
{
	private ILogger Logger = Log.ForContext<MainWindow>();

	protected AppOptions m_aoOptions;

	protected LANConnection m_lanConnection;

	protected SCNNetwork m_scnNetwork;

	protected ObservableCollection<ICUDevice> m_colDevices = new ObservableCollection<ICUDevice>();

	protected TreeView m_treeDevices = new TreeView();

	protected TreeStore m_treeDeviceStore;

	protected DataField<Image> m_dfDeviceImage = new DataField<Image>();

	protected DataField<UIListLanDevice> m_dfDeviceData = new DataField<UIListLanDevice>();

	protected DataField<Image> m_dfDeviceSelection = new DataField<Image>();

	protected TextEntry m_txtDeviceFilter;

	protected string m_currentFilter = string.Empty;

	protected bool m_fFTPAvailable;

	protected bool m_fPropertyChanges;

	protected ICULanDevice m_currentDevice;

	protected string m_currentSCNName;

	protected ICUUser m_currentUser;

	protected bool m_fIsShowingSCNTabPages;

	protected bool m_fAutomaticDeviceLoginAttempt = true;

	protected bool m_fForceShowDeviceLoginDialog = true;

	protected int m_fShowingDeviceLoginDialog;

	protected Image m_imgEmpty = Image.FromResource(typeof(App), AppProperties.ResourcePath("empty16.png"));

	protected List<PanelBase> m_allPanels = new List<PanelBase>();

	protected List<PanelBase> m_allSCNPanels = new List<PanelBase>();

	protected MenuItem m_mnuDevice;

	protected MenuItem m_mnuSCN;

	protected MenuItem m_mnuUploadLogo;

	protected MenuItem m_mnuReport;

	protected MenuItem m_mnuUnlockFeatures;

	protected MenuItem m_mnuAddToSCN;

	protected Button m_btnRemoveLan;

	protected Button m_btnReboot;

	protected ToggleButton m_btnTabs;

	protected Button m_btnSave;

	protected Button m_btnRevert;

	protected DlgUploadResources m_dlgUploadResources = new DlgUploadResources();

	protected DlgUpload m_dlgUpload = new DlgUpload();

	protected Dialog m_dlgSplash;

	protected const int s_nUpdateDevicesTimerMs = 1000;

	protected System.Timers.Timer m_timSearch = new System.Timers.Timer();

	protected System.Timers.Timer m_timRefreshDevices;

	public bool AllowStoreChanges { get; set; }

	public MainWindow(Dialog splashWindow, AppOptions aoOptions)
	{
		m_aoOptions = aoOptions;
		m_dlgSplash = splashWindow;
		Title = AppProperties.AppName;
		Decorated = true;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		AllowStoreChanges = true;
		m_timRefreshDevices = new System.Timers.Timer();
		m_timRefreshDevices.AutoReset = false;
		m_timRefreshDevices.Interval = 1000.0;
		m_timRefreshDevices.Elapsed += OnShouldUpdateDevices;
		UpdateManager.InitUpdateManager(AppProperties.FTPSite, AppProperties.FTPUsername, AppProperties.FTPPassword);
		UpdateManager.CanConnect += UpdateManager_CanConnect;
		CheckForUpdates_Installer();
		AppProperties.InitializeSettings();
		Shown += OnMainWindowShown;
		m_dlgUpload.Initialize(this);
		m_dlgUploadResources.Initialize();
		Menu menu = new Menu();
		MenuItem menuItem = new MenuItem("_File")
		{
			SubMenu = new Menu()
		};
		AddMenuItemHelper(menuItem, "_Logout...", OnLogout);
		menuItem.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(menuItem, "Check for _Updates...", OnCheckForUpdates);
		menuItem.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(menuItem, "Create Image Update file...", OnUploadLogo);
		menuItem.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(menuItem, "_Close", delegate
		{
			Close();
		});
		menu.Items.Add(menuItem);
		m_mnuDevice = new MenuItem("_Device");
		m_mnuDevice.SubMenu = new Menu();
		AddMenuItemHelper(m_mnuDevice, "_Save Settings As...", OnStoreSettings);
		AddMenuItemHelper(m_mnuDevice, "_Load Settings...", OnLoadSettings);
		m_mnuDevice.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(m_mnuDevice, "_Load Preset...", OnLoadPresets);
		m_mnuDevice.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(m_mnuDevice, "_Synchronise time...", OnSyncTime);
		AddMenuItemHelper(m_mnuDevice, "_Upload new firmware...", OnUploadFirmware);
		m_mnuUploadLogo = AddMenuItemHelper(m_mnuDevice, "Upload Image...", OnUploadLogo);
		m_mnuDevice.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(m_mnuDevice, "_Refresh\t\t\t\tF5", OnRefreshLAN);
		m_mnuDevice.SubMenu.Items.Add(new SeparatorMenuItem());
		m_mnuUnlockFeatures = AddMenuItemHelper(m_mnuDevice, "_Install feature(s)...", OnUnlockFeature);
		m_mnuAddToSCN = AddMenuItemHelper(m_mnuDevice, "_Add to a new SCN...", OnCreateNewSCN);
		m_mnuDevice.SubMenu.Items.Add(new SeparatorMenuItem());
		MenuItem menuItem2 = AddMenuItemHelper(m_mnuDevice, "Reset to _Factory Defaults...", OnResetToFactoryDefaults);
		menu.Items.Add(m_mnuDevice);
		m_mnuDevice.Sensitive = false;
		menuItem2.Sensitive = false;
		MenuItem menuItem3 = new MenuItem("_Help")
		{
			SubMenu = new Menu()
		};
		AddMenuItemHelper(menuItem3, "_About...", OnMenuAbout);
		menuItem3.SubMenu.Items.Add(new SeparatorMenuItem());
		AddMenuItemHelper(menuItem3, "_Settings...", OnMenuSettings);
		menu.Items.Add(menuItem3);
		MainMenu = menu;
		m_lanConnection = new LANConnection(m_colDevices);
		m_colDevices.CollectionChanged += OnDevicesCollectionChanged;
		m_lanConnection.LoginRequest += OnLoginRequest;
		m_lanConnection.DeviceReRegistered += OnDeviceReregistered;
		m_lanConnection.DeviceUnregistered += OnDeviceUnregistered;
		m_lanConnection.ErrorHandler += OnShowError;
		VBox vBox = new VBox();
		vBox.BackgroundColor = Colors.White;
		vBox.PackStart(CreateDeviceFilterUI());
		m_treeDeviceStore = new TreeStore(m_dfDeviceImage, m_dfDeviceData, m_dfDeviceSelection);
		m_treeDevices.DataSource = m_treeDeviceStore;
		m_treeDevices.SelectionMode = SelectionMode.Multiple;
		m_treeDevices.VerticalPlacement = WidgetPlacement.Center;
		m_treeDevices.Font = Font.SystemSansSerifFont.WithSize(12.0);
		m_treeDevices.Columns.Add("", new ImageCellView(m_dfDeviceImage));
		m_treeDevices.Columns.Add("", new TextCellView(m_dfDeviceData));
		m_treeDevices.Columns.Add("", new ImageCellView(m_dfDeviceSelection));
		m_treeDevices.HeadersVisible = false;
		m_treeDevices.MouseMoved += OnDeviceTreeMouseMoved;
		m_treeDevices.SelectionChanged += OnDeviceSelectionChanged;
		m_treeDevices.MinWidth = 280.0;
		m_treeDevices.WidthRequest = (m_treeDevices.HeightRequest = 280.0);
		m_treeDevices.MarginRight = 8.0;
		m_treeDevices.ExpandHorizontal = true;
		m_treeDevices.HorizontalScrollPolicy = ScrollPolicy.Never;
		vBox.PackStart(m_treeDevices, expand: true);
		HBox hBox = new HBox
		{
			MarginRight = 8.0
		};
		hBox.PackStart(MainWindowBase.AddImageButton("update-arrow.png", "Refresh. Search for attached devices.", OnRefreshLAN));
		hBox.PackStart(MainWindowBase.AddImageButton("add16.png", "Manually add a new device (enter IP address)", OnAddManualDeviceClicked));
		m_btnRemoveLan = MainWindowBase.AddImageButton("delete16.png", "Remove device from this list", OnRemoveManualDeviceClicked);
		hBox.PackStart(m_btnRemoveLan);
		m_btnReboot = MainWindowBase.AddImageButton("power-button-off.png", "Reboot the device", OnRebootClicked);
		hBox.PackStart(m_btnReboot);
		m_btnTabs = AddImageToggleButton("tabs.png", "Show all tab pages", OnShowTabs);
		hBox.PackEnd(m_btnTabs);
		vBox.PackStart(hBox);
		HBox hBox2 = new HBox();
		ImageView imageView = new ImageView(Image.FromResource(typeof(App), AppProperties.ResourcePath("logo_alfen_small.png")));
		imageView.ButtonPressed += OnImgLogoButtonPressed;
		hBox2.PackStart(imageView, expand: true, fill: true);
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			MarginRight = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = hBox2
		};
		vBox.PackEnd(widget);
		m_allPanels = new List<PanelBase>
		{
			new PanelSockets(this, "PAGE_SOCKET"),
			new PanelInformation(this, "PAGE_INFORMATION", showBorder: false),
			new PanelPower(this, "PAGE_POWER", showBorder: false),
			new PanelLoadbalancing(this, "PAGE_NETWORK", showBorder: false),
			new PanelAuthorization(this, "PAGE_WHITELIST", showBorder: false),
			new PanelTransactions(this, "PAGE_TRANSACTIONS"),
			new PanelConnectivity(this, "PAGE_BACKOFFICE", showBorder: false),
			new PanelInterface(this, "PAGE_UI", showBorder: false),
			new PanelAlerts(this, "PAGE_ALERTS", showBorder: false),
			new PanelLog(this, "PAGE_LOG"),
			new PanelMonitoring(this, "PAGE_STATES", showBorder: false),
			new PanelAllProperties(this, "PAGE_ALLPROPERTIES", showBorder: false)
		};
		m_allSCNPanels = new List<PanelBase>
		{
			new PanelSCNOverview(this, "PAGE_SCN_OVERVIEW"),
			new PanelSCNSettings(this, "PAGE_SCN_SETTINGS")
		};
		VBox vBox2 = new VBox();
		foreach (PanelBase allPanel in m_allPanels)
		{
			allPanel.Changed += OnPanelChanged;
		}
		foreach (PanelBase allSCNPanel in m_allSCNPanels)
		{
			allSCNPanel.Changed += OnPanelChanged;
		}
		if (Monitor.TryEnter(MainWindowBase.ActivePanels, -1))
		{
			try
			{
				MainWindowBase.ActivePanels.Add(new PanelNoDevice(this));
				vBox2 = CreateTabControl();
			}
			finally
			{
				Monitor.Exit(MainWindowBase.ActivePanels);
			}
		}
		HBox hBox3 = new HBox();
		Button button = new Button("Exit")
		{
			MinWidth = MainWindowBase.s_nButtonWidth,
			MinHeight = MainWindowBase.s_nButtonHeight
		};
		button.Clicked += delegate
		{
			Close();
		};
		m_btnSave = new Button("Save")
		{
			TooltipText = "Save all changes on this page to the charging station",
			MinWidth = MainWindowBase.s_nButtonWidth,
			MinHeight = MainWindowBase.s_nButtonHeight,
			Sensitive = false
		};
		m_btnSave.Clicked += OnSaveClicked;
		m_btnRevert = new Button("Revert")
		{
			TooltipText = "Revert all changes on this page back to their original values",
			MinWidth = MainWindowBase.s_nButtonWidth,
			MinHeight = MainWindowBase.s_nButtonHeight,
			Sensitive = false
		};
		m_btnRevert.Clicked += OnRevertClicked;
		hBox3.PackEnd(button);
		hBox3.PackEnd(m_btnSave);
		hBox3.PackEnd(m_btnRevert);
		FrameBox widget2 = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			PaddingTop = 8.0,
			PaddingBottom = 8.0,
			Margin = 0.0,
			BorderWidth = 0.0,
			BackgroundColor = Colors.White,
			Content = hBox3
		};
		vBox2.PackEnd(widget2);
		HPaned hPaned = new HPaned();
		hPaned.BackgroundColor = Colors.White;
		hPaned.Panel1.Content = vBox;
		hPaned.Panel2.Content = vBox2;
		hPaned.Panel1.Resize = false;
		hPaned.Panel2.Resize = true;
		FrameBox frameBox = new FrameBox
		{
			BackgroundColor = Colors.White,
			Padding = 16.0,
			Margin = 0.0,
			Content = hPaned
		};
		frameBox.KeyPressed += OnKeyPressed;
		Padding = 2.0;
		Content = frameBox;
		m_timSearch.Elapsed += OnSearchTimerElapsed;
		m_timSearch.AutoReset = true;
		m_timSearch.Interval = 2000.0;
		m_timSearch.Start();
		CloseRequested += HandleCloseRequested;
		ClearDeviceSelection();
	}

	private void OnDeviceTreeMouseMoved(object sender, MouseMovedEventArgs e)
	{
		try
		{
			TreePosition rowAtPosition = m_treeDevices.GetRowAtPosition(e.Position);
			if (rowAtPosition == null)
			{
				m_treeDevices.TooltipText = null;
				return;
			}
			TreeNavigator navigatorAt = m_treeDeviceStore.GetNavigatorAt(rowAtPosition);
			if (navigatorAt == null)
			{
				m_treeDevices.TooltipText = null;
				return;
			}
			UIListLanDevice value = navigatorAt.GetValue(m_dfDeviceData);
			if (value == null)
			{
				m_treeDevices.TooltipText = null;
				return;
			}
			if (value.IsSCN)
			{
				m_treeDevices.TooltipText = "SCN";
				return;
			}
			ICULanDevice device = value.GetDevice();
			if (device == null)
			{
				m_treeDevices.TooltipText = null;
			}
			else
			{
				m_treeDevices.TooltipText = "CPID: " + device.Identification + "\nIP: " + device.Address;
			}
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, "Error in OnDeviceTreeMouseMoved: {Message}", ex.Message);
			m_treeDevices.TooltipText = null;
		}
	}

	private async void OnLoginRequest(ICULanDevice lanDevice, ACEWebLoginData loginData)
	{
		Logger = Logger.ForContext("SerialNumber", lanDevice.SerialNumber).ForContext("Identity", lanDevice.HostName).ForContext("DeviceName", lanDevice.Name)
			.ForContext("DeviceIp", lanDevice.IPAddress)
			.ForContext("SCNNetwork", lanDevice.SCNNetwork)
			.ForContext("Protocol", lanDevice.Protocol)
			.ForContext("FirmwareVersion", lanDevice.FirmwareVersionNumber)
			.ForContext("NumberOfFeederCables", lanDevice.NumberOfFeederCables)
			.ForContext("NumberOfSockets", lanDevice.NumberOfSockets)
			.ForContext("IsRebooting", lanDevice.IsRebooting)
			.ForContext("IsLoggedIn", lanDevice.IsLoggedIn);
		if (!lanDevice.ShouldLoginUsingUniquePassword)
		{
			Logger.Debug("OnLoginRequest: Login to {DeviceName} without unique password", lanDevice.Name);
			loginData.Username = m_currentUser.Group.HTTPUser;
			loginData.Password = m_currentUser.Group.HTTPPassword;
			ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
			Task<(bool, HttpStatusCode, string)> task = Task.Run(async () =>
			{
				try
				{
					return ((bool IsLoggedIn, HttpStatusCode HttpStatusCode, string Content))(await lanDevice.LoginRequest());
				}
				finally
				{
					finishedEvent.Set();
				}
			});
			while (!finishedEvent.WaitOne(10))
			{
				try
				{
					Application.MainLoop.DispatchPendingEvents();
				}
				catch (Exception ex)
				{
					Logger.Error(ex, ex.Message);
				}
			}
			(bool, HttpStatusCode, string) result = task.GetAwaiter().GetResult();
			loginData.IsLoggedIn = result.Item1;
			loginData.DisplayName = Settings.Default.LastUserName;
			return;
		}
		loginData.LoginError = string.Empty;
		if (DlgDeviceLogin.ValidateStoredPassword())
		{
			Logger.Debug("OnLoginRequest: Stored password not valid for {DeviceName} ForceLoginDialog={ForceLoginDialog} AutomaticDeviceLoginAttempt={AutomaticDeviceLoginAttempt}", lanDevice.Name, m_fForceShowDeviceLoginDialog, m_fAutomaticDeviceLoginAttempt);
			if (!m_fForceShowDeviceLoginDialog && m_fAutomaticDeviceLoginAttempt)
			{
				if (lanDevice.UserData != null)
				{
					if (lanDevice.UserData is ACEWebLoginData aCEWebLoginData)
					{
						loginData.Username = aCEWebLoginData.Username;
						loginData.Password = aCEWebLoginData.Password;
						Logger.Debug("OnLoginRequest: Using LAN device `UserData` for {DeviceName}", lanDevice.Name);
					}
				}
				else
				{
					loginData.Username = Settings.Default.LastDeviceUsername;
					loginData.Password = Settings.Default.LastDevicePassword;
					Logger.Debug("OnLoginRequest: Using last used password for {DeviceName}", lanDevice.Name);
				}
			}
		}
		m_fAutomaticDeviceLoginAttempt = false;
		if (Interlocked.CompareExchange(ref m_fShowingDeviceLoginDialog, 1, 0) != 0)
		{
			return;
		}
		try
		{
			ManualResetEvent finishedEvent2 = new ManualResetEvent(initialState: false);
			Logger.Debug("OnLoginRequest: Show login dialog");
			await Application.InvokeAsync(() =>
			{
				DlgDeviceLogin dlgDeviceLogin = new DlgDeviceLogin(lanDevice);
				if (dlgDeviceLogin.Run(this) == Command.Ok)
				{
					loginData.Username = dlgDeviceLogin.Username;
					loginData.Password = dlgDeviceLogin.Password;
					loginData.IsLoggedIn = true;
				}
				else
				{
					loginData.Username = string.Empty;
					loginData.Password = string.Empty;
					loginData.IsLoginCancelled = true;
					loginData.LoginError = "User cancelled the login";
				}
				finishedEvent2.Set();
			});
			while (!finishedEvent2.WaitOne(10))
			{
				try
				{
					Application.MainLoop.DispatchPendingEvents();
				}
				catch (Exception ex2)
				{
					Logger.Error(ex2, ex2.Message);
				}
			}
			if (Settings.Default.StorePasswords)
			{
				Logger.Debug("OnLoginRequest: Store the login data");
				ACEWebLoginData aCEWebLoginData2 = new ACEWebLoginData();
				aCEWebLoginData2.Username = loginData.Username;
				aCEWebLoginData2.Password = loginData.Password;
				lanDevice.UserData = aCEWebLoginData2;
			}
			loginData.DisplayName = Settings.Default.LastUserName;
		}
		finally
		{
			Interlocked.Exchange(ref m_fShowingDeviceLoginDialog, 0);
		}
	}

	private void UpdateManager_CanConnect(bool connection)
	{
		m_fFTPAvailable = connection;
		if (connection || DateTime.Compare(Settings.Default.LastFTPActivity, new DateTime(2020, 1, 1)) == 0)
		{
			Settings.Default.LastFTPActivity = DateTime.Now;
			((SettingsBase)Settings.Default).Save();
		}
		else if ((DateTime.Now - Settings.Default.LastFTPActivity).TotalDays >= 7.0 && (DateTime.Now - Settings.Default.LastFTPConnectionWarning).TotalDays >= 7.0)
		{
			MessageDialog.ShowError(this, "Unable to reach the FTP Update server since: \n" + Settings.Default.LastFTPActivity.Date.ToLongDateString(), "Please check your network connection!");
			Settings.Default.LastFTPConnectionWarning = DateTime.Now;
			((SettingsBase)Settings.Default).Save();
		}
	}

	private void OnKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.F5)
		{
			RefreshDeviceList();
		}
	}

	private void OnImgLogoButtonPressed(object sender, ButtonEventArgs e)
	{
		if (e.Button == PointerButton.Left)
		{
			Process.Start("http://alfen.com");
		}
	}

	private bool CheckForUpdates_Installer()
	{
		bool result = false;
		string fileName = string.Empty;
		Version newestVersion = UpdateManager.GetNewestVersion("ACE Service Installer", ref fileName, AppProperties.FTPCommunicationTimeout);
		if (!m_fFTPAvailable)
		{
			return result;
		}
		if (newestVersion != null)
		{
			FileVersionInfo versionInfo = FileVersionInfo.GetVersionInfo(Assembly.GetExecutingAssembly().Location);
			GlobalLogContext.PushProperty("Internal", Environment.UserDomainName.Equals("VANALFEN"));
			string value = (Enumerable.Contains(versionInfo.ProductVersion.ToString(), '+') ? versionInfo.ProductVersion.ToString().Split(new char[1] { '+' })[0] : versionInfo.ProductVersion.ToString());
			GlobalLogContext.PushProperty("SiaVersion", value);
			GlobalLogContext.PushProperty("Environment", "Release");
			Version version = new Version(versionInfo.FileVersion);
			if (newestVersion > version)
			{
				if (MessageDialog.AskQuestion($"A new version of the '{AppProperties.AppName}' is found.", $"Your current version is {version}, the new version is {newestVersion}.\nDo you want to download and install the new version?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
				{
					Logger.AddChargerContext(m_currentDevice).Information("Application update accepted");
					ShowDownloadMessage($"Downloading new setup version {newestVersion}. Please wait....");
					Show();
					if (UpdateManager.DownloadFile(fileName, "", AppProperties.LocalSetupFolder, OnShowError))
					{
						Process.Start(Path.Combine(AppProperties.LocalSetupFolder, fileName));
					}
					Application.Exit();
					return true;
				}
				Logger.AddChargerContext(m_currentDevice).Information("Application update rejected");
			}
		}
		if (UpdateManager.IsFTPNewer(AppProperties.SettingsFilename))
		{
			if (MessageDialog.AskQuestion("A new version of the settingsfile is available online.\nDo you want to download the new settings", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
			{
				Logger.AddChargerContext(m_currentDevice).Information("Settings update accepted");
				UpdateManager.DownloadFile(AppProperties.SettingsFilename, "", Path.GetDirectoryName(AppProperties.SettingsFilename), OnShowError);
				result = true;
			}
			else
			{
				Logger.AddChargerContext(m_currentDevice).Information("Settings update rejected");
			}
		}
		return result;
	}

	private bool CheckForUpdates_Settings()
	{
		bool result = false;
		bool flag = false;
		long totalDownloadSize = 0L;
		List<string> list = UpdateManager.CheckAdditionalFTPFiles(AppProperties.FTPFirmwareFolder, AppProperties.LocalFirmwareFolder, out totalDownloadSize, checkFileDates: false, removeFilesNotOnFTP: true, AppProperties.FTPCommunicationTimeout);
		if (!m_fFTPAvailable)
		{
			return result;
		}
		if (list.Count > 0)
		{
			if (MessageDialog.AskQuestion($"There are {list.Count} new firmware versions available on the FTP site ({totalDownloadSize} bytes).", string.Format("Do you want to download the new firmware versions to your PC?", Array.Empty<object>()), Command.Yes, Command.No, Command.Cancel) == Command.Yes)
			{
				Logger.AddChargerContext(m_currentDevice).Information("Firmware files update accepted");
				int num = 0;
				Stopwatch stopwatch = Stopwatch.StartNew();
				foreach (string item in list)
				{
					if (UpdateManager.DownloadFile(Path.GetFileName(item), AppProperties.FTPFirmwareFolder, AppProperties.LocalFirmwareFolder, OnShowError))
					{
						num++;
						result = true;
					}
				}
				MessageDialog.ShowMessage(this, $"Downloaded {num} new firmware files (in {stopwatch.Elapsed.TotalSeconds} s).");
				flag = true;
			}
			else
			{
				Logger.AddChargerContext(m_currentDevice).Information("Firmware files update rejected");
			}
		}
		RemoveOldPresetFolder();
		List<string> list2 = UpdateManager.CheckAdditionalFTPFiles(AppProperties.FTPTCPPresetsFolder, AppProperties.LocalTCPPresetsFolder, out var totalDownloadSize2, checkFileDates: true, removeFilesNotOnFTP: true, AppProperties.FTPCommunicationTimeout);
		List<string> list3 = UpdateManager.CheckAdditionalFTPFiles(AppProperties.FTPRTUPresetsFolder, AppProperties.LocalRTUPresetsFolder, out var totalDownloadSize3, checkFileDates: true, removeFilesNotOnFTP: true, AppProperties.FTPCommunicationTimeout);
		totalDownloadSize = totalDownloadSize2 + totalDownloadSize3;
		int num2 = list2.Count + list3.Count;
		if (!m_fFTPAvailable)
		{
			return result;
		}
		if (num2 > 0)
		{
			if (MessageDialog.AskQuestion($"There are {num2} new modbus preset files found on the FTP site ({totalDownloadSize} bytes).", "Do you want to download the modbus preset files to your PC?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
			{
				Logger.AddChargerContext(m_currentDevice).Information("Modbus presets update accepted");
				int num3 = 0;
				foreach (string item2 in list2)
				{
					if (UpdateManager.DownloadFile(Path.GetFileName(item2), AppProperties.FTPTCPPresetsFolder, AppProperties.LocalTCPPresetsFolder, OnShowError))
					{
						num3++;
						result = true;
					}
				}
				foreach (string item3 in list3)
				{
					if (UpdateManager.DownloadFile(Path.GetFileName(item3), AppProperties.FTPRTUPresetsFolder, AppProperties.LocalRTUPresetsFolder, OnShowError))
					{
						num3++;
						result = true;
					}
				}
				MessageDialog.ShowMessage(this, $"Downloaded {num3} new preset files.");
				flag = true;
			}
			else
			{
				Logger.AddChargerContext(m_currentDevice).Information("Modbus presets update rejected");
			}
		}
		List<string> list4 = UpdateManager.CheckAdditionalFTPFiles(AppProperties.FTPBackofficePresetsFolder, AppProperties.LocalBackofficePresetsFolder, out totalDownloadSize, checkFileDates: true, removeFilesNotOnFTP: false, AppProperties.FTPCommunicationTimeout);
		string text = $"There are {list4.Count} new backoffice preset files found on the FTP site ({totalDownloadSize} bytes).";
		text += "These files are required when you want to change the backoffice connection of a charging station.";
		text += "These files are required when you want to change the backoffice connection of a charging station.";
		if (list4.Count > 100)
		{
			text += " This may take some minutes, please be patience.";
		}
		if (list4.Count > 0)
		{
			if (MessageDialog.AskQuestion(text, "Do you want to download these new backoffice preset files to your PC?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
			{
				Logger.AddChargerContext(m_currentDevice).Information("Back office presets update accepted");
				DlgDownloadFiles dlgDownloadFiles = new DlgDownloadFiles(list4, AppProperties.FTPBackofficePresetsFolder, AppProperties.LocalBackofficePresetsFolder);
				dlgDownloadFiles.Run(this);
				MessageDialog.ShowMessage(this, $"Downloaded {dlgDownloadFiles.DownloadCount} new backoffice preset files.");
				flag = true;
			}
			else
			{
				Logger.AddChargerContext(m_currentDevice).Information("Back office presets update rejected");
			}
		}
		if (flag && Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			try
			{
				foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
				{
					activePanel.ChangeDevice(m_currentDevice);
				}
			}
			finally
			{
				Monitor.Exit(MainWindowBase.ActivePanels);
			}
		}
		return result;
	}

	private void RemoveOldPresetFolder()
	{
		DirectoryInfo directoryInfo = new DirectoryInfo(AppProperties.LocalOldPresetsFolder);
		if (!directoryInfo.Exists)
		{
			return;
		}
		FileInfo[] files = directoryInfo.GetFiles();
		foreach (FileInfo fileInfo in files)
		{
			try
			{
				File.Delete(fileInfo.FullName);
			}
			catch (IOException ex)
			{
				Logger.Error(ex, ex.Message);
			}
		}
		Directory.Delete(AppProperties.LocalOldPresetsFolder, recursive: true);
	}

	private HBox CreateDeviceFilterUI()
	{
		HBox hBox = new HBox();
		hBox.MarginRight = 8.0;
		hBox.MarginBottom = 4.0;
		Label widget = new Label("Filter:")
		{
			MarginRight = 4.0
		};
		hBox.PackStart(widget);
		hBox.TooltipText = "Filter the device list by S/N, CPID, IP address or model name";
		m_txtDeviceFilter = new TextEntry
		{
			ExpandHorizontal = true
		};
		m_txtDeviceFilter.Changed += OnDeviceFilterChanged;
		hBox.PackStart(m_txtDeviceFilter, expand: true);
		return hBox;
	}

	private void OnPanelChanged(object sender, EventArgs e)
	{
		CheckIfPropertiesHasChanged();
	}

	private void CheckIfPropertiesHasChanged()
	{
		m_fPropertyChanges = false;
		if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			try
			{
				foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
				{
					if (activePanel.IsChanged)
					{
						m_fPropertyChanges = true;
						break;
					}
				}
			}
			finally
			{
				Monitor.Exit(MainWindowBase.ActivePanels);
			}
		}
		m_btnSave.Sensitive = m_fPropertyChanges;
		m_btnRevert.Sensitive = m_fPropertyChanges;
	}

	private Widget ShowDownloadMessage(string message)
	{
		Widget content = Content;
		Table table = new Table
		{
			MinHeight = 100.0,
			MinWidth = 400.0
		};
		Label widget = new Label(message)
		{
			Font = Font.SystemSansSerifFont.WithWeight(FontWeight.Bold)
		};
		table.Add(widget, 0, 0, 1, 1, hexpand: true, vexpand: true, WidgetPlacement.Center, WidgetPlacement.Center);
		Spinner spinner = new Spinner();
		table.Add(spinner, 0, 1, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Center, WidgetPlacement.Center);
		spinner.Animate = true;
		Content = table;
		return content;
	}

	private void OnMainWindowShown(object sender, EventArgs e)
	{
		if (m_dlgSplash != null)
		{
			m_dlgSplash.Hide();
			m_dlgSplash = null;
		}
		ShowLogon();
	}

	private void ShowLogon()
	{
		Application.Invoke(() =>
		{
			Sensitive = false;
			DlgLogon dlgLogon = new DlgLogon(AppProperties.ICUConfig);
			if (dlgLogon.Run(this) == Command.Cancel)
			{
				Close();
			}
			else
			{
				CheckForUpdates_Settings();
				Sensitive = true;
				m_currentUser = dlgLogon.User;
				m_btnTabs.Visible = false;
				FillPanelList();
				string text = FileVersionInfo.GetVersionInfo(Assembly.GetExecutingAssembly().Location).FileVersion;
				string[] array = text.Split(new char[1] { '.' });
				if (array.Count() >= 4)
				{
					text = array[0] + "." + array[1] + "." + array[2] + "-" + array[3];
				}
				if (m_currentUser == null)
				{
					Title = AppProperties.AppName + " " + text + " - Settings: " + AppProperties.ICUConfig.Version;
				}
				else if (m_fFTPAvailable)
				{
					Title = AppProperties.AppName + " " + text + " - Settings: " + AppProperties.ICUConfig.Version + " - " + m_currentUser.Fullname + " (" + m_currentUser.Group?.Name.ToString() + ")";
				}
				else
				{
					Title = AppProperties.AppName + " " + text + " - Settings: " + AppProperties.ICUConfig.Version + " - " + m_currentUser.Fullname + " (" + m_currentUser.Group?.Name.ToString() + ") *";
				}
				if (m_lanConnection != null)
				{
					m_lanConnection.StartSearch();
				}
			}
			dlgLogon.Dispose();
		});
	}

	private void FillPanelList(bool fSCN = false)
	{
		if (!Monitor.TryEnter(MainWindowBase.ActivePanels, -1))
		{
			return;
		}
		try
		{
			MainWindowBase.ActivePanels.Clear();
			foreach (PanelBase allPanel in m_allPanels)
			{
				if (!fSCN && allPanel.IsVisible(m_currentUser))
				{
					MainWindowBase.ActivePanels.Add(allPanel);
				}
				allPanel.IsPanelVisible = false;
			}
			foreach (PanelBase allSCNPanel in m_allSCNPanels)
			{
				if (fSCN && allSCNPanel.IsVisible(m_currentUser))
				{
					MainWindowBase.ActivePanels.Add(allSCNPanel);
				}
				allSCNPanel.IsPanelVisible = false;
			}
			ReCreateTabControl();
			foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
			{
				activePanel.SetUser(m_currentUser);
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(MainWindowBase.ActivePanels);
		}
	}

	private void OnDeviceUnregistered(object sender, DeviceEventArgs e)
	{
		Logger.Verbose("Device '{DeviceName}' unregistered", e.Device.Name);
	}

	private void OnDeviceReregistered(object sender, DeviceEventArgs e)
	{
		Application.Invoke(() =>
		{
			if (m_currentDevice == e.Device && Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
			{
				try
				{
					if (!m_currentDevice.IsLoggedIn && !m_currentDevice.IsRequestingLoginData)
					{
						ReselectCurrentItem(showLoginDialog: false);
					}
				}
				finally
				{
					Monitor.Exit(MainWindowBase.ActivePanels);
				}
			}
		});
	}

	private void OnDeviceFilterChanged(object sender, EventArgs e)
	{
		if (m_txtDeviceFilter != null)
		{
			m_currentFilter = m_txtDeviceFilter.Text?.Trim() ?? string.Empty;
			RefreshDeviceList();
		}
	}

	private bool DeviceMatchesFilter(ICULanDevice device)
	{
		if (string.IsNullOrEmpty(m_currentFilter))
		{
			return true;
		}
		if (!string.IsNullOrEmpty(device.HostName) && device.HostName.IndexOf(m_currentFilter, StringComparison.OrdinalIgnoreCase) >= 0)
		{
			return true;
		}
		if (!string.IsNullOrEmpty(device.Identification) && device.Identification.IndexOf(m_currentFilter, StringComparison.OrdinalIgnoreCase) >= 0)
		{
			return true;
		}
		if (!string.IsNullOrEmpty(device.Address) && device.Address.IndexOf(m_currentFilter, StringComparison.OrdinalIgnoreCase) >= 0)
		{
			return true;
		}
		return false;
	}

	private void OnAddManualDeviceClicked(object sender, EventArgs e)
	{
		DlgManualIP dlg = new DlgManualIP();
		try
		{
			if (dlg.Run(this) != Command.Ok)
			{
				return;
			}
			if (m_colDevices.FirstOrDefault((ICUDevice a) => a.Address == dlg.IPAddress.ToString()) != null)
			{
				new DlgInfo($"Device with IP: {dlg.IPAddress} is already present in the overview and cannot be added.", showInTaskbar: false).Run(this);
				return;
			}
			ICULanDevice lanDev = m_lanConnection.AddManualDevice(dlg.IPAddress, dlg.Port, dlg.Hostname, dlg.NumberOfSockets, dlg.LoginRequired);
			if (lanDev != null)
			{
				RefreshDeviceList();
				Application.Invoke(() =>
				{
					TreeNavigator firstNode = m_treeDeviceStore.GetFirstNode();
					bool flag = false;
					do
					{
						UIListLanDevice value = firstNode.GetValue(m_dfDeviceData);
						if (value != null && value.GetDevice() == lanDev)
						{
							m_treeDevices.SelectRow(firstNode.CurrentPosition);
							break;
						}
						flag = firstNode.MoveToChild();
						if (!flag)
						{
							flag = firstNode.MoveNext();
						}
						if (!flag)
						{
							flag = firstNode.MoveToParent();
							if (flag)
							{
								flag = firstNode.MoveNext();
							}
						}
					}
					while (flag);
				});
				new DlgInfo($"Device '{lanDev.Identification}' with IP: {dlg.IPAddress} sucessfully added to the overview.", showInTaskbar: false).Run(this);
			}
			else
			{
				new DlgInfo($"Device with IP: {dlg.IPAddress} could not be added to the overview.", showInTaskbar: false).Run(this);
			}
		}
		finally
		{
			if (dlg != null)
			{
				((IDisposable)dlg).Dispose();
			}
		}
	}

	private void OnRemoveManualDeviceClicked(object sender, EventArgs e)
	{
		if (m_treeDevices != null && m_treeDevices.IsRowSelected(m_treeDevices.SelectedRow))
		{
			UIListLanDevice value = m_treeDeviceStore.GetNavigatorAt(m_treeDevices.SelectedRow).GetValue(m_dfDeviceData);
			if (value != null)
			{
				m_lanConnection.RemoveManualDevice(value.GetDevice());
			}
		}
	}

	private void OnRebootClicked(object sender, EventArgs e)
	{
		if (m_currentDevice == null || !m_currentDevice.Login().IsLoggedIn)
		{
			return;
		}
		SuspendUpdateTimers(fSuspend: true);
		try
		{
			using DlgReboot dlgReboot = new DlgReboot(m_currentDevice);
			dlgReboot.Run(this);
		}
		finally
		{
			SuspendUpdateTimers(fSuspend: false);
		}
	}

	private void OnShowTabs(object sender, EventArgs e)
	{
	}

	public void ReselectCurrentItem(bool showLoginDialog)
	{
		m_fForceShowDeviceLoginDialog = showLoginDialog;
		OnDeviceSelectionChanged(null, null);
	}

	private void OnDeviceSelectionChanged(object sender, EventArgs e)
	{
		Logger.Verbose("OnDeviceSelectionChanged: BEGIN");
		if (m_currentDevice != null && m_currentDevice.IsRebooting)
		{
			Logger.Verbose("OnDeviceSelectionChanged: IsRebooting");
			return;
		}
		bool flag = m_treeDevices.SelectedRows.Length >= 1;
		if ((m_treeDevices.SelectedRow != null) & flag)
		{
			m_fAutomaticDeviceLoginAttempt = true;
			UIListLanDevice value = m_treeDeviceStore.GetNavigatorAt(m_treeDevices.SelectedRow).GetValue(m_dfDeviceData);
			if (value != null)
			{
				Logger.Verbose("OnDeviceSelectionChanged: SelectedRow SCN={SCNName}", value.SCNName);
				TreePosition selectedRow = m_treeDevices.SelectedRow;
				TreeNavigator firstNode = m_treeDeviceStore.GetFirstNode();
				if (firstNode == null || firstNode.CurrentPosition == null)
				{
					return;
				}
				bool flag2 = false;
				do
				{
					if (m_dfDeviceSelection == null)
					{
						if (firstNode == null)
						{
							break;
						}
						continue;
					}
					Image image = null;
					firstNode.SetValue(data: (firstNode.CurrentPosition != selectedRow) ? m_imgEmpty : Image.FromResource(typeof(App), AppProperties.ResourcePath("left-arrow.png")), field: m_dfDeviceSelection);
					flag2 = firstNode.MoveToChild();
					if (!flag2)
					{
						flag2 = firstNode.MoveNext();
					}
					if (!flag2)
					{
						flag2 = firstNode.MoveToParent();
						if (flag2)
						{
							flag2 = firstNode.MoveNext();
						}
					}
				}
				while (flag2 && firstNode != null);
				ICULanDevice device = value.GetDevice();
				if (m_currentDevice != device || device == null || (device != null && !device.IsLoggedIn))
				{
					if (device != null)
					{
						Logger.Verbose("OnDeviceSelectionChanged: Selecting device {0} ({1})", device.Identification, device.IPAddress);
					}
					if (m_currentDevice != null)
					{
						SuspendUpdateTimers(fSuspend: true);
						if (m_fPropertyChanges && AllowStoreChanges)
						{
							Logger.Verbose("OnDeviceSelectionChanged: Some properties have changed");
							bool flag3 = false;
							if (Settings.Default.AskSaveChanges && MessageDialog.AskQuestion("You have changed some properties for device '" + m_currentDevice.Identity + "', do you want to save the changes?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
							{
								Logger.Verbose("OnDeviceSelectionChanged: Saved all changes");
								SaveAllChanges();
								flag3 = true;
							}
							if (!flag3)
							{
								Logger.Verbose("OnDeviceSelectionChanged: Reverted all changes");
								RevertAllChanges();
							}
							m_fPropertyChanges = false;
						}
						Logger.Verbose("OnDeviceSelectionChanged: Device logout (if there is a previous Device, then logout from that device first.)");
						m_currentDevice.Logout();
						Logger.Verbose("OnDeviceSelectionChanged: Device deallocate");
						m_currentDevice.Deallocate();
					}
					else
					{
						Logger.Verbose("OnDeviceSelectionChanged: Current device is null");
					}
					m_btnRemoveLan.Sensitive = true;
					m_currentDevice = device;
					if (m_currentDevice != null)
					{
						Logger.Verbose("OnDeviceSelectionChanged: Select the new device");
						m_currentDevice.Allocate();
						if (!m_currentDevice.IsLoggedIn && m_fShowingDeviceLoginDialog == 0)
						{
							Logger.Verbose("OnDeviceSelectionChanged: Device login");
							(bool, HttpStatusCode, string) tuple = m_currentDevice.Login();
							if (!tuple.Item1)
							{
								Logger.Verbose("OnDeviceSelectionChanged: Device not logged in {HttpStatusCode} {Response}", tuple.Item2, tuple.Item3);
								switch (tuple.Item2)
								{
								case HttpStatusCode.HttpVersionNotSupported:
									Thread.Sleep(500);
									break;
								default:
									if (!m_currentDevice.LoginData.IsLoginCancelled)
									{
										MessageDialog.ShowError(this, $"Failed to communicate with device at IP address {m_currentDevice.IPAddress}\nError: {tuple.Item3}");
										Logger.Error("Failed to communicate with device at IP address {DeviceIp}\nError: {LoginError}", m_currentDevice.IPAddress, tuple.Item3);
									}
									break;
								case HttpStatusCode.OK:
								case HttpStatusCode.Forbidden:
								case HttpStatusCode.RequestTimeout:
								case HttpStatusCode.TooManyRequests:
									break;
								}
								Logger.Verbose("OnDeviceSelectionChanged: Device {DeviceName} logout", m_currentDevice.Name);
								m_currentDevice.Logout();
								m_btnReboot.Sensitive = false;
								m_btnSave.Sensitive = false;
								m_btnRevert.Sensitive = false;
								m_btnRemoveLan.Sensitive = true;
							}
							else
							{
								Logger.Verbose("OnDeviceSelectionChanged: Device {DeviceName} logged in", m_currentDevice.Name);
							}
						}
						else
						{
							Logger.Verbose("OnDeviceSelectionChanged: Device DeviceName} was already logged in", m_currentDevice.Name);
						}
						if (m_currentDevice.IsLoggedIn)
						{
							if (!m_currentDevice.IsDefaultCategoryCollected)
							{
								Logger.Verbose("OnDeviceSelectionChanged: collect main property category generic, generic2");
								m_currentDevice.UpdateCategories("generic", "generic2");
								m_currentDevice.IsDefaultCategoryCollected = true;
							}
							if (m_currentDevice.HasProperty(8784))
							{
								ETamperState propertyInt = (ETamperState)m_currentDevice.GetPropertyInt(8784, 0);
								if (propertyInt == ETamperState.Tampered || propertyInt == ETamperState.TamperActive)
								{
									Logger.Verbose("OnDeviceSelectionChanged: The charging station might have been tampered");
									if (MessageDialog.Confirm(this, "The charging station might have been tampered", "Do you want to acknowledge the tamper alarm?", Command.Yes))
									{
										ICUProperty property = m_currentDevice.GetProperty(8784, 0);
										if (property != null)
										{
											property.Value = 4;
											m_currentDevice.StoreProperties(property);
										}
									}
								}
							}
						}
						else
						{
							Logger.Verbose("OnDeviceSelectionChanged: Device {DeviceName} it not logged", m_currentDevice.Name);
						}
					}
					else
					{
						Logger.Verbose("OnDeviceSelectionChanged: Current device is null");
					}
					Logger.Verbose("OnDeviceSelectionChanged: Update the panels");
					UpdatePanels(value);
				}
			}
			else
			{
				Logger.Verbose("OnDeviceSelectionChanged: No device selected, clear all panels");
				ClearDeviceSelection();
				m_currentDevice = null;
				m_currentSCNName = string.Empty;
				m_btnReboot.Sensitive = false;
			}
			m_treeDeviceStore.GetNavigatorAt(m_treeDevices.SelectedRow).SetValue(m_dfDeviceImage, value.Icon);
			m_mnuDevice.Sensitive = m_currentDevice != null && m_currentDevice.IsLoggedIn;
			m_mnuUploadLogo.Sensitive = m_currentDevice != null && m_currentDevice.HasDisplay && IWSFirmwareFeatures.IsFeatureUnlocked(m_currentDevice.FirmwareVersionNumber, m_currentDevice.GetPropertyUInt(8610, 0), IWSFirmwareFeatures.Features.PersonalizedDisplay, m_currentDevice.isAHP);
			m_mnuAddToSCN.Sensitive = m_currentDevice != null && !m_currentDevice.HasSCNNetwork;
			m_mnuUnlockFeatures.Sensitive = m_currentDevice != null && m_currentDevice.GetProperty(8608, 0) != null;
		}
		if (m_currentDevice == null)
		{
			m_btnSave.Sensitive = false;
			m_btnRevert.Sensitive = false;
			m_mnuDevice.Sensitive = false;
		}
		Logger.Verbose("OnDeviceSelectionChanged: END");
	}

	public void RefreshPanels()
	{
		if (m_treeDevices.SelectedRow != null)
		{
			UIListLanDevice value = m_treeDeviceStore.GetNavigatorAt(m_treeDevices.SelectedRow).GetValue(m_dfDeviceData);
			if (value != null)
			{
				UpdatePanels(value);
			}
		}
	}

	private void UpdatePanels(UIListLanDevice uiLanDevice)
	{
		SuspendUpdateTimers(fSuspend: false);
		if (uiLanDevice.IsSCN)
		{
			if (m_scnNetwork == null)
			{
				m_scnNetwork = SCNNetwork.Instance;
				m_scnNetwork.Start();
			}
			if (m_currentSCNName != uiLanDevice.SCNName)
			{
				if (!m_fIsShowingSCNTabPages)
				{
					FillPanelList(fSCN: true);
					m_fIsShowingSCNTabPages = true;
				}
				m_currentSCNName = uiLanDevice.SCNName;
				if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
				{
					try
					{
						foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
						{
							activePanel.OnChangeSCN(m_scnNetwork, uiLanDevice.SCNName, m_lanConnection);
						}
					}
					finally
					{
						Monitor.Exit(MainWindowBase.ActivePanels);
					}
				}
				if (m_nSelectedTab < 0 || m_nSelectedTab >= m_allSCNPanels.Count)
				{
					m_nSelectedTab = 0;
				}
				SelectTab(m_nSelectedTab);
			}
		}
		else
		{
			if (m_scnNetwork != null)
			{
				m_scnNetwork.Stop();
				m_scnNetwork = null;
			}
			if (m_fIsShowingSCNTabPages)
			{
				FillPanelList();
				m_fIsShowingSCNTabPages = false;
			}
			m_currentSCNName = "";
			if (m_currentDevice == null || !m_currentDevice.IsLoggedIn)
			{
				m_btnReboot.Sensitive = false;
				m_btnSave.Sensitive = false;
				m_btnRevert.Sensitive = false;
				m_btnRemoveLan.Sensitive = true;
				m_nSelectedTab = -2;
				SelectTab(m_nSelectedTab);
				m_panelNotLoggedIn.ChangeDevice(m_currentDevice);
			}
			else
			{
				if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
				{
					try
					{
						foreach (PanelBase activePanel2 in MainWindowBase.ActivePanels)
						{
							try
							{
								activePanel2.BusyChangingDevice = true;
								activePanel2.ChangeDevice(m_currentDevice, forceRefresh: false);
								activePanel2.BusyChangingDevice = false;
							}
							catch (Exception ex)
							{
								Logger.Error(ex, ex.Message);
							}
						}
					}
					finally
					{
						Monitor.Exit(MainWindowBase.ActivePanels);
					}
				}
				if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
				{
					try
					{
						if (m_nSelectedTab >= MainWindowBase.ActivePanels.Count)
						{
							m_nSelectedTab = 0;
						}
					}
					finally
					{
						Monitor.Exit(MainWindowBase.ActivePanels);
					}
				}
				else
				{
					m_nSelectedTab = 0;
				}
				if (m_nSelectedTab < 0 || m_nSelectedTab >= m_allPanels.Count)
				{
					m_nSelectedTab = 0;
				}
				if (m_tabFrame != null && m_tabButtons != null)
				{
					m_tabButtons.Sensitive = true;
					SelectTab(m_nSelectedTab);
				}
			}
		}
		m_btnReboot.Sensitive = m_currentDevice != null && !m_currentDevice.IsRebooting;
	}

	private void ClearDeviceSelection()
	{
		m_btnRemoveLan.Sensitive = false;
		if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			try
			{
				foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
				{
					activePanel.ChangeDevice(null);
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
			finally
			{
				Monitor.Exit(MainWindowBase.ActivePanels);
			}
		}
		if (m_tabFrame != null && m_tabButtons != null)
		{
			m_tabButtons.Sensitive = false;
			SelectTab(-1);
		}
		m_mnuDevice.Sensitive = false;
	}

	private void OnDevicesCollectionChanged(object sender, NotifyCollectionChangedEventArgs e)
	{
		m_timRefreshDevices.Stop();
		m_timRefreshDevices.Start();
	}

	protected void OnShouldUpdateDevices(object sender, ElapsedEventArgs e)
	{
		RefreshDeviceList();
	}

	public void RefreshDeviceList()
	{
		Logger.Verbose("Refresh device list");
		Application.Invoke(() =>
		{
			try
			{
				m_lanConnection?.StartBrowsing();
				if (m_colDevices.Count > 0)
				{
					bool flag = m_colDevices.Count > 6;
					m_currentSCNName = null;
					m_treeDevices.SelectionChanged -= OnDeviceSelectionChanged;
					m_treeDeviceStore.Clear();
					TreePosition treePosition = null;
					foreach (ICULanDevice item in from ICULanDevice d in m_colDevices
						orderby d.HasSCNNetwork descending, d.SCNNetwork ?? string.Empty, d.Identification ?? string.Empty
						select d)
					{
						if (DeviceMatchesFilter(item))
						{
							TreeNavigator treeNavigator = null;
							UIListLanDevice uIListLanDevice = new UIListLanDevice(item, flag || item.HasSCNNetwork);
							if (item.HasSCNNetwork)
							{
								TreeNavigator treeNavigator2 = null;
								TreeNavigator firstNode = m_treeDeviceStore.GetFirstNode();
								if (firstNode.CurrentPosition != null)
								{
									do
									{
										UIListLanDevice value = firstNode.GetValue(m_dfDeviceData);
										if (value != null && value.IsSCN && value.SCNName == item.SCNNetwork)
										{
											treeNavigator2 = firstNode;
											break;
										}
									}
									while (firstNode.MoveNext());
								}
								if (treeNavigator2 != null)
								{
									treeNavigator = treeNavigator2.AddChild();
								}
								else
								{
									UIListLanDevice uIListLanDevice2 = new UIListLanDevice(item.SCNNetwork);
									treeNavigator2 = m_treeDeviceStore.AddNode().SetValues(m_dfDeviceImage, uIListLanDevice2.Icon, m_dfDeviceData, uIListLanDevice2, m_dfDeviceSelection, m_imgEmpty);
									if (treePosition == null && m_currentDevice == null && m_currentSCNName == item.SCNNetwork)
									{
										treePosition = treeNavigator2.CurrentPosition;
									}
									treeNavigator = treeNavigator2.AddChild();
								}
							}
							else
							{
								treeNavigator = m_treeDeviceStore.AddNode();
							}
							if (treeNavigator != null)
							{
								treeNavigator.SetValues(m_dfDeviceImage, uIListLanDevice.Icon, m_dfDeviceData, uIListLanDevice, m_dfDeviceSelection, m_imgEmpty);
								if (treePosition == null && m_currentDevice == item)
								{
									treePosition = treeNavigator.CurrentPosition;
								}
							}
						}
					}
					m_treeDevices.SelectionChanged += OnDeviceSelectionChanged;
					if (treePosition != null)
					{
						m_treeDevices.ExpandToRow(treePosition);
						m_treeDevices.ScrollToRow(treePosition);
						m_treeDevices.SelectRow(treePosition);
						Resizable = false;
						Resizable = true;
					}
					else if (m_colDevices.Count > 0)
					{
						m_nSelectedTab = -3;
						SelectTab(m_nSelectedTab);
						m_panelNoDeviceSelected.ChangeDevice(null);
					}
				}
				else
				{
					m_treeDeviceStore.Clear();
					m_currentDevice = null;
					ClearDeviceSelection();
					m_btnReboot.Sensitive = false;
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
		});
	}

	private void HandleCloseRequested(object sender, CloseRequestedEventArgs args)
	{
		if (m_currentUser == null)
		{
			args.AllowClose = true;
		}
		else if (Settings.Default.AskConfirmExit)
		{
			if (m_currentDevice != null && m_fPropertyChanges)
			{
				args.AllowClose = MessageDialog.AskQuestion("You have unsaved changed!", "Are you sure you want to close this application?", Command.Yes, Command.No, Command.Cancel) == Command.Yes;
			}
			else
			{
				args.AllowClose = MessageDialog.AskQuestion("Are you sure you want to close this application?", Command.Yes, Command.No, Command.Cancel) == Command.Yes;
			}
		}
		if (args.AllowClose)
		{
			if (m_currentDevice != null)
			{
				m_currentDevice.Logout();
			}
			Application.Exit();
		}
	}

	private void OnLogout(object sender, EventArgs e)
	{
		if (m_currentUser != null)
		{
			m_currentUser = null;
			m_currentDevice?.Logout();
			m_lanConnection.StopSearch();
		}
		Title = $"{AppProperties.AppName} {AppProperties.ICUConfig.Version}";
		ShowLogon();
	}

	private void OnCheckForUpdates(object sender, EventArgs e)
	{
		bool flag = CheckForUpdates_Installer();
		bool flag2 = CheckForUpdates_Settings();
		if (!flag && !flag2)
		{
			MessageDialog.ShowMessage(this, string.Format("No new updates found, you already have the latest version installed.", Array.Empty<object>()));
		}
	}

	public void OnRefreshLAN(object sender, EventArgs e)
	{
		foreach (ICULanDevice item in m_colDevices.Cast<ICULanDevice>().ToList())
		{
			m_lanConnection.RemoveDevice(item);
		}
		RefreshDeviceList();
	}

	private void RevertAllChanges()
	{
		if (m_currentDevice != null)
		{
			m_currentDevice.RevertChanges();
		}
		if (!Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			return;
		}
		try
		{
			foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
			{
				activePanel.OnRevertChanges();
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(MainWindowBase.ActivePanels);
		}
	}

	private void SaveAllChanges()
	{
		Sensitive = false;
		bool flag = false;
		if (Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			try
			{
				foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
				{
					if (!activePanel.OnSaveChanges())
					{
						Sensitive = true;
						return;
					}
				}
				if (m_currentDevice != null)
				{
					ICUProperty property = m_currentDevice.GetProperty(8273, 0);
					ICUProperty property2 = m_currentDevice.GetProperty(8275, 0);
					ICUProperty property3 = m_currentDevice.GetProperty(8272, 0);
					if (property != null && property2 != null && property3 != null && (property.IsChanged || property2.IsChanged || property3.IsChanged))
					{
						flag = true;
					}
					if (property != null && property.IsChanged && !m_currentDevice.StoreProperties(property))
					{
						Sensitive = true;
						return;
					}
					ICUProperty property4 = m_currentDevice.GetProperty(8609, 0);
					if (property4 != null && property4.IsChanged && !m_currentDevice.StoreProperties(property4))
					{
						Sensitive = true;
						return;
					}
					ICUProperty property5 = m_currentDevice.GetProperty(8583, 0);
					if (property5 != null)
					{
						double num = Math.Round((DateTime.UtcNow - ICUDevice.UnixEpoch).TotalMilliseconds);
						property5.Value = num;
						if (!m_currentDevice.StoreProperties(property5))
						{
							Sensitive = true;
							return;
						}
					}
					ValidateCSConfiguration(m_currentDevice);
					m_currentDevice.StoreChangedProperties();
					foreach (PanelBase activePanel2 in MainWindowBase.ActivePanels)
					{
						activePanel2.OnPostSaveChanges();
					}
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
			finally
			{
				Monitor.Exit(MainWindowBase.ActivePanels);
			}
		}
		Sensitive = true;
		if (flag)
		{
			m_lanConnection.RemoveManualDevice(m_currentDevice);
			MessageDialog.ShowMessage(this, "The Identity or model has changed, the device will be re-added automatically to the Service Installer");
		}
	}

	private void OnSaveClicked(object sender, EventArgs e)
	{
		SaveAllChanges();
	}

	private void OnRevertClicked(object sender, EventArgs e)
	{
		RevertAllChanges();
	}

	private void OnStoreSettings(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		using SaveFileDialog saveFileDialog = new SaveFileDialog("Save Settings As")
		{
			InitialFileName = m_currentDevice.Identification + ".exml",
			Multiselect = false
		};
		saveFileDialog.Filters.Add(new FileDialogFilter("Encrypted Setting files", "*.exml"));
		saveFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
		if (saveFileDialog.Run(this))
		{
			PropertyStorage.SaveProperties(saveFileDialog.FileName, m_currentDevice);
		}
	}

	private void LoadProperties(string filename)
	{
		if (m_currentDevice != null)
		{
			int num = PropertyStorage.LoadProperties(filename, m_currentDevice);
			if (num > 0)
			{
				MessageDialog.ShowMessage(this, "Settings file successfully loaded.", $"Changed {num} properties.\nPlease save the changes to the device.");
				CheckIfPropertiesHasChanged();
			}
		}
	}

	private void OnLoadSettings(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		using OpenFileDialog openFileDialog = new OpenFileDialog("Open Settings File")
		{
			Multiselect = false
		};
		openFileDialog.Filters.Add(new FileDialogFilter("Encrypted Setting files", "*.exml"));
		openFileDialog.Filters.Add(new FileDialogFilter("Setting files", "*.xml"));
		openFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
		if (openFileDialog.Run(this))
		{
			LoadProperties(openFileDialog.FileName);
		}
	}

	private void OnLoadPresets(object sender, EventArgs e)
	{
		using DlgPresets dlgPresets = new DlgPresets(m_currentDevice);
		if (dlgPresets.Run(this) == Command.Ok)
		{
			LoadProperties(dlgPresets.FileName);
		}
	}

	private void OnSyncTime(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			currentDevice.SetDate(DateTime.UtcNow);
			MessageDialog.ShowMessage(this, $"The device time has changed to {DateTime.UtcNow.ToString()}.");
		}
	}

	private void OnUploadFirmware(object sender, EventArgs e)
	{
		ShowUploadFirmwareDialog();
	}

	private void OnUploadLogo(object sender, EventArgs e)
	{
		ShowUploadLogoDialog();
	}

	private void OnUnlockFeature(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		using DlgUnlockFeature dlgUnlockFeature = new DlgUnlockFeature(m_currentDevice);
		if (dlgUnlockFeature.Run(this) != Command.Ok)
		{
			return;
		}
		SuspendUpdateTimers(fSuspend: true);
		try
		{
			using DlgReboot dlgReboot = new DlgReboot(m_currentDevice, fAutoStartReboot: true);
			dlgReboot.Run(this);
		}
		finally
		{
			SuspendUpdateTimers(fSuspend: false);
		}
	}

	private void OnCreateNewSCN(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		SuspendUpdateTimers(fSuspend: true);
		try
		{
			using DlgAddToSCN dlgAddToSCN = new DlgAddToSCN(m_currentDevice, m_scnNetwork);
			if (dlgAddToSCN.Run(this) == Command.Ok)
			{
				Thread.Sleep(2000);
				Application.Invoke(() =>
				{
					RefreshDeviceList();
				});
			}
		}
		finally
		{
			SuspendUpdateTimers(fSuspend: false);
		}
	}

	public void SuspendUpdateTimers(bool fSuspend)
	{
		Logger.Verbose("SuspendUpdateTimers {Suspend}", new object[1] { fSuspend });
		if (!Monitor.TryEnter(MainWindowBase.ActivePanels, 5000))
		{
			return;
		}
		try
		{
			foreach (PanelBase activePanel in MainWindowBase.ActivePanels)
			{
				activePanel.SuspendUpdateTimer(fSuspend);
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(MainWindowBase.ActivePanels);
		}
	}

	public void ShowUploadFirmwareDialog()
	{
		SuspendUpdateTimers(fSuspend: true);
		try
		{
			m_dlgUpload.ChangeDevice(m_currentDevice);
			m_dlgUpload.Run(this);
			ReselectCurrentItem(showLoginDialog: false);
			m_treeDevices.Hide();
			m_treeDevices.Show();
			if (m_currentDevice != null && !m_currentDevice.IsUploading && !m_currentDevice.IsRebooting && m_currentDevice.IsLoggedIn)
			{
				m_currentDevice.UpdateCategories("generic", "generic2");
				m_currentDevice.IsDefaultCategoryCollected = true;
			}
		}
		finally
		{
			SuspendUpdateTimers(fSuspend: false);
		}
	}

	public void ShowUploadLogoDialog(string filePath = "")
	{
		SuspendUpdateTimers(fSuspend: true);
		try
		{
			m_dlgUploadResources.SetObjects(m_currentDevice, m_currentUser, filePath);
			m_dlgUploadResources.Run(this);
			_ = Command.Ok;
		}
		finally
		{
			SuspendUpdateTimers(fSuspend: false);
		}
	}

	public void ValidateCSConfiguration(ICUDevice device, bool updateValues = false)
	{
		if (device == null || device.NumberOfSockets <= 1 || device.NumberOfFeederCables > 1)
		{
			return;
		}
		if (updateValues)
		{
			device.UpdateProperties(2173184u, 3221760u, 2122240u, 2195457u, 2122753u);
		}
		int propertyInt = device.GetPropertyInt(8489, 0);
		int propertyInt2 = device.GetPropertyInt(12585, 0);
		int propertyInt3 = device.GetPropertyInt(8290, 0);
		if (propertyInt + propertyInt2 <= propertyInt3)
		{
			return;
		}
		bool flag = device.GetPropertyString(8576, 1, 0).Length == 0;
		bool flag2 = device.GetPropertyInt(8292, 0, 1) == 0;
		if (flag & flag2)
		{
			device.GetProperty(8489, 0).Value = propertyInt3 / 2;
			device.GetProperty(12585, 0).Value = propertyInt3 / 2;
			if ((device as ICULanDevice).HasProperty(33793, 1))
			{
				device.GetProperty(33793, 1).Value = propertyInt3 / 2 * 230 * 3;
				device.GetProperty(33793, 2).Value = propertyInt3 / 2 * 230 * 3;
			}
			new DlgInfo($"The total sum of the connectors maximum current ({propertyInt + propertyInt2}A) is more than the station current ({propertyInt3}A)!\n" + "This is only allowed in combination with Static Loadbalancing or Smart Charging Network.\n\n" + $"NOTE: The charing station {device.Identification} connector maximum currents will be set to {propertyInt3 / 2}A.", showInTaskbar: true).Run();
		}
	}

	private void OnResetToFactoryDefaults(object sender, EventArgs e)
	{
		GetInfoPanel()?.ResetToFactoryDefaults();
	}

	private void OnMenuAbout(object sender, EventArgs e)
	{
		using DlgAbout dlgAbout = new DlgAbout(AppProperties.ICUConfig);
		dlgAbout.Run(this);
	}

	private void OnMenuSettings(object sender, EventArgs e)
	{
		using DlgSettings dlgSettings = new DlgSettings();
		dlgSettings.Run(this);
	}

	private PanelInformation GetInfoPanel()
	{
		PanelBase panelBase = m_allPanels.FirstOrDefault((PanelBase a) => a.PageID == "PAGE_INFORMATION");
		if (panelBase != null)
		{
			return panelBase as PanelInformation;
		}
		return null;
	}

	private void OnSearchTimerElapsed(object sender, ElapsedEventArgs e)
	{
		Application.Invoke(() =>
		{
			if (m_lanConnection != null && m_lanConnection.Devices.Count == 0)
			{
				m_lanConnection.StartBrowsing();
			}
		});
	}

	private void OnShowError(string primaryText, string secondaryText)
	{
		Application.Invoke(() =>
		{
			MessageDialog.ShowError(this, primaryText, secondaryText);
		});
	}

	private void OnShowError(string primaryText)
	{
		if (!string.IsNullOrEmpty(primaryText))
		{
			Application.Invoke(() =>
			{
				m_fAutomaticDeviceLoginAttempt = false;
				MessageDialog.ShowError(this, primaryText);
				RefreshPanels();
			});
		}
	}

	public void AllowAutoLogonOnce()
	{
		m_fAutomaticDeviceLoginAttempt = true;
	}

	internal void RefreshMenu()
	{
		m_mnuDevice.Sensitive = m_currentDevice != null && m_currentDevice.IsLoggedIn;
	}
}
