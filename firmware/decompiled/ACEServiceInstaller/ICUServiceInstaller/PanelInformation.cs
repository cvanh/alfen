using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Diagnostics;
using System.Drawing;
using System.IO;
using System.Linq;
using System.Net;
using System.Text.RegularExpressions;
using System.Threading;
using System.Threading.Tasks;
using ICUFWUCreator;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Enums;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelInformation : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelInformation>();

	protected const int s_nUpdateInterval = 1000;

	protected Button m_btnSetDate;

	protected Button m_btnUpload;

	protected Button m_btnUpdateLicense;

	protected Button m_btnFactoryDefaults;

	protected Button m_btnChangePassword;

	protected Button m_btnTempPassword;

	protected Button m_btnlogout;

	protected Button m_btnlogout2;

	protected Button m_btnEndUserAccess;

	protected UIPropertyBase m_txtEndUserAccessType;

	protected UIPropertyBase m_txtIdentity;

	protected UIPropertyCheckbox m_chkEnableSSA;

	protected Dictionary<string, string> m_dicModel = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicTimezones = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicTimezonesMinutes = new Dictionary<string, string>();

	protected UIPassword m_txtPrc;

	protected UIPassword m_txtDefaultPassword;

	public PanelInformation(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Information";
		Tooltip = "General";
		IconName = "information.png";
		foreach (ICUDeviceModel value in Enum.GetValues(typeof(ICUDeviceModel)))
		{
			string text = ICUDeviceModelExtension.ToString(value, "");
			if (!m_dicModel.ContainsKey(text))
			{
				m_dicModel.Add(text, text);
			}
		}
		StartUpdateTimer(1000);
	}

	private void OnSyncTimeClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			currentDevice.SetDate(DateTime.UtcNow);
			DateTime utcNow = DateTime.UtcNow;
			ReadOnlyCollection<TimeZoneInfo> systemTimeZones = TimeZoneInfo.GetSystemTimeZones();
			int utcOffset = (currentDevice.HasProperty(8291) ? currentDevice.GetPropertyInt(8302, 0) : (currentDevice.GetPropertyInt(8282, 0) * 6));
			utcNow = TimeZoneInfo.ConvertTimeFromUtc(utcNow, TimeZoneInfo.FindSystemTimeZoneById(systemTimeZones.FirstOrDefault((TimeZoneInfo x) => x.BaseUtcOffset.TotalMinutes == (double)utcOffset).Id));
			Logger.AddChargerContext(m_currentDevice).Information("Sync time");
			MessageDialog.ShowMessage(ParentWindow, $"The device time has changed to {utcNow}.", "Sync time");
		}
	}

	public void ResetToFactoryDefaults()
	{
		ICULanDevice lanDev = m_currentDevice;
		if (lanDev == null)
		{
			return;
		}
		Task.Run(async () =>
		{
			bool result = await IWSConnection.TestIWSConnection(AppProperties.IsahSite);
			Application.Invoke(async () =>
			{
				if (!result)
				{
					MessageDialog.ShowError(ParentWindow, "We cannot connect to the Alfen server to download the default settings!", "Please make sure you have a working internet on this PC!");
				}
				else
				{
					string identity = lanDev.Identity;
					if (MessageDialog.AskQuestion($"Are you sure you want to reset '{identity}' to the factory defaults?", string.Format("ALL settings will be set the factory defaults!\nThe whitelist and locallist will remain unchanged." + ((!lanDev.isAHP) ? "\n\nThe CS will reboot 3 times during this process." : ""), Array.Empty<object>()), Command.Yes, Command.No, Command.Cancel) == Command.Yes)
					{
						MainWindow mw = ParentWindow as MainWindow;
						try
						{
							if (mw != null)
							{
								mw.AllowStoreChanges = false;
								mw.Sensitive = false;
							}
							if (!lanDev.ClearSettings())
							{
								MessageDialog.ShowError(ParentWindow, "Failed to reset settings. Please try again.");
								return;
							}
							await Task.Delay(5000);
							if (!lanDev.isAHP)
							{
								Stopwatch sw = Stopwatch.StartNew();
								using (DlgReboot dlgReboot = new DlgReboot(lanDev, fAutoStartReboot: true))
								{
									dlgReboot.Run(ParentWindow);
								}
								while (sw.ElapsedMilliseconds < 30000)
								{
									await Task.Delay(100);
								}
							}
							ResetToFactorySettings(lanDev, askToUpdateLogo: true);
						}
						finally
						{
							if (mw != null)
							{
								mw.AllowStoreChanges = true;
								mw.Sensitive = true;
							}
						}
					}
				}
			});
		});
	}

	private void OnUploadClicked(object sender, EventArgs e)
	{
		if (m_parent != null)
		{
			m_parent.ShowUploadFirmwareDialog();
			UpdateButtons();
		}
	}

	private void OnUpdateLicenseKeyClicked(object sender, EventArgs e)
	{
		if (m_currentUser == null)
		{
			return;
		}
		ICULanDevice lanDev = m_currentDevice;
		if (lanDev == null)
		{
			return;
		}
		Task.Run(async () =>
		{
			Application.Invoke(() =>
			{
				Parent.Sensitive = false;
			});
			if (await IWSConnection.TestIWSConnection(AppProperties.IsahSite))
			{
				Application.Invoke(() =>
				{
					lanDev.GetPropertyString(8609, 0, 0);
					IWSObject objData = new IWSObject();
					if (lanDev.LoadSettingsFromIWS(m_currentUser, AppProperties.IsahSite, m_currentUser.Company, ref objData, includeLogo: false))
					{
						IWSPropertyValue iWSPropertyValue = objData.Properties.FirstOrDefault((IWSPropertyValue a) => a.Id == 8609);
						if (lanDev.storeProperty(8609, 0, iWSPropertyValue?.Value))
						{
							MessageDialog.ShowMessage("Your license key has been updated. The Charging Station will now reboot to activate the changes.");
							MainWindow mainWindow = (MainWindow)ParentWindow;
							mainWindow.SuspendUpdateTimers(fSuspend: true);
							try
							{
								using DlgReboot dlgReboot = new DlgReboot(lanDev, fAutoStartReboot: true);
								dlgReboot.Run(ParentWindow);
								return;
							}
							finally
							{
								mainWindow.SuspendUpdateTimers(fSuspend: false);
							}
						}
						MessageDialog.ShowMessage("Your license key has not changed.");
					}
				});
			}
			else
			{
				Application.Invoke(() =>
				{
					MessageDialog.ShowError(ParentWindow, "There is a problem with the internet connection!\nWe cannot connect to the Alfen server to download the license key!\nPlease make sure you have a working internet on this PC!");
				});
			}
			Application.Invoke(() =>
			{
				Parent.Sensitive = true;
			});
		});
	}

	private void OnResetToFactoryDefaults(object sender, EventArgs e)
	{
		MainWindow mainWindow = (MainWindow)ParentWindow;
		mainWindow.SuspendUpdateTimers(fSuspend: true);
		try
		{
			ResetToFactoryDefaults();
		}
		finally
		{
			mainWindow.SuspendUpdateTimers(fSuspend: false);
		}
	}

	private void OnChangePassword(object sender, EventArgs e)
	{
		DlgDeviceChangePassword dlgDeviceChangePassword = new DlgDeviceChangePassword(m_currentDevice);
		if (dlgDeviceChangePassword.Run(ParentWindow) != Command.Ok)
		{
			return;
		}
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice.ChangePassword(dlgDeviceChangePassword.NewPassword))
		{
			MessageDialog.ShowMessage(ParentWindow, "The password for the CS '" + currentDevice.Identification + "' has been updated.");
			currentDevice.UpdateProperties(2208768u);
			if (currentDevice.LoginData.IsAdminUser)
			{
				currentDevice.LoginData.Password = dlgDeviceChangePassword.NewPassword;
			}
		}
	}

	private void OnTempPassword(object sender, EventArgs e)
	{
		DlgDeviceTempPassword dlgDeviceTempPassword = new DlgDeviceTempPassword(m_currentDevice);
		if (dlgDeviceTempPassword.Run(ParentWindow) == Command.Ok)
		{
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice.CreateTempPassword(dlgDeviceTempPassword.NewPassword, dlgDeviceTempPassword.ExpirationTime))
			{
				MessageDialog.ShowMessage(ParentWindow, "The temporary password for the CS '" + currentDevice.Identification + "' has been updated.");
				currentDevice.UpdateProperties(2208512u);
			}
		}
	}

	private void OnLogout(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice.Logout())
		{
			currentDevice.UserData = null;
			currentDevice.LoginData.Username = string.Empty;
			currentDevice.LoginData.Password = string.Empty;
			if (ParentWindow is MainWindow mainWindow)
			{
				mainWindow.RefreshMenu();
				mainWindow.RefreshPanels();
			}
		}
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		CheckIfCSIsInitialize(iCULanDevice);
		newDevice.UpdateCategories("comm");
		using (UIConfigurationPanel uIConfigurationPanel = AddConfigurationPanel(Tooltip))
		{
			UIConfigCategory uIConfigCategory = uIConfigurationPanel.AddCategory("General");
			uIConfigCategory.Add(AddSmallHeader("Identification"));
			uIConfigCategory.Add(AddReadOnlyCustomSelect(8272, 0, "Model", m_dicModel));
			uIConfigCategory.Add(AddReadOnlyText(8273, 0));
			uIConfigCategory.Add(m_txtIdentity = AddText(8275, 0));
			uIConfigCategory.Add(AddReadOnlyText(8277, 0, "Charge point vendor"));
			uIConfigCategory.Add(AddSmallHeader("Information"));
			if (iCULanDevice.HasProperty(8583))
			{
				uIConfigCategory.Add(AddReadOnlyText(8583, 0, "Last time Configuration Changed", UIPropertyStringType.DateTime));
			}
			uIConfigCategory.Add(AddReadOnlyText(4104, 0, "Platform type"));
			if (iCULanDevice.isAHP)
			{
				uIConfigCategory.Add(AddCustomText("Hardware version SCB", ParseRevision(newDevice.GetPropertyInt(8269, 1), newDevice.GetPropertyInt(8269, 2)), "", null, 1, 2116865u));
				uIConfigCategory.Add(AddReadOnlyText(4106, 0, "Software version SCB"));
			}
			else
			{
				uIConfigCategory.Add(AddCustomText("Hardware version controller board", ParseRevision(newDevice.GetPropertyInt(8269, 1), newDevice.GetPropertyInt(8269, 2)), "", null, 1, 2116865u));
				uIConfigCategory.Add(AddCustomText("Hardware version power board", ParseRevision(newDevice.GetPropertyInt(8269, 3), newDevice.GetPropertyInt(8269, 4)), "", null, 1, 2116867u));
				uIConfigCategory.Add(AddReadOnlyText(4106, 0, "Software version controller board"));
				if (iCULanDevice.HasProperty(12674))
				{
					uIConfigCategory.Add(AddReadOnlyText(12674, 0, "Bootloader version controller board"));
				}
			}
			uIConfigCategory.Add(AddCustomText("WiFi supported", iCULanDevice.HasWifiSupport ? "Yes" : "No", "", null, 1, 3313408u));
			bool flag = iCULanDevice.GetPropertyBool(8785, 1) || iCULanDevice.GetPropertyInt(8784, 0) != 0;
			uIConfigCategory.Add(AddCustomText("Tamper detection supported", flag ? "Yes" : "No", "", null, 1, 2248961u));
			uIConfigCategory.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
			if (iCULanDevice.IsUniquePasswordRequired)
			{
				uIConfigCategory.AddWidget(m_btnlogout2 = AddCustomButton("Logout", "Logout from this CS", OnLogout, true));
			}
			uIConfigCategory.AddWidget(m_btnUpload = AddCustomButton("Upload Firmware...", "Upload new Firmware", OnUploadClicked, true));
			uIConfigCategory.AddWidget(m_btnFactoryDefaults = AddCustomButton("Factory Defaults...", "Reset to factory default settings", OnResetToFactoryDefaults, true));
			m_btnFactoryDefaults.Sensitive = newDevice != null;
			m_btnUpload.Sensitive = newDevice != null;
			UIConfigCategory uIConfigCategory2 = uIConfigurationPanel.AddCategory("Sub devices");
			if (!iCULanDevice.isAHP)
			{
				if (iCULanDevice.FirmwareVersionNumber >= new Version("4.3.0"))
				{
					uIConfigCategory2.Add(AddReadOnlyText(12672, 0, "NFC-RFID reader 1 hardw. version", UIPropertyStringType.NFC1_HW, null, 3244032u));
					uIConfigCategory2.Add(AddReadOnlyText(12672, 0, "NFC-RFID reader 1 softw. version", UIPropertyStringType.NFC1_SW, null, 3244033u));
					if (iCULanDevice.NumberOfSockets > 1)
					{
						uIConfigCategory2.Add(AddReadOnlyText(12673, 0, "NFC-RFID reader 2 hardw. version", UIPropertyStringType.NFC2_HW, null, 3244288u));
						uIConfigCategory2.Add(AddReadOnlyText(12673, 0, "NFC-RFID reader 2 softw. version", UIPropertyStringType.NFC2_SW, null, 3244289u));
					}
				}
				else
				{
					uIConfigCategory2.Add(AddReadOnlyText(8276, 0, "NFC-RFID reader hardware version", UIPropertyStringType.NFC1_HW, null, 3244032u));
					uIConfigCategory2.Add(AddReadOnlyText(8276, 0, "NFC-RFID reader software version", UIPropertyStringType.NFC1_SW, null, 3244033u));
				}
			}
			else
			{
				for (byte b = 1; b <= iCULanDevice.NumberOfSockets; b++)
				{
					uIConfigCategory2.Add(AddSmallHeader($"SocketBoard {b}"));
					uIConfigCategory2.Add(AddReadOnlyText(33025, b, "Device Id"));
					uIConfigCategory2.Add(AddReadOnlyText(33026, b, "Hardware version"));
					uIConfigCategory2.Add(AddReadOnlyText(33027, b, "Software version"));
					if (iCULanDevice.HasProperty(33794, b))
					{
						uIConfigCategory2.Add(AddReadOnlyText(33794, b, "Extended software info"));
					}
					uIConfigCategory2.Add(AddReadOnlyText(33032, b, "Energymeter info"));
					uIConfigCategory2.Add(AddReadOnlyText(33031, b, "ISO15118 info"));
				}
				if (!iCULanDevice.isAHPV2)
				{
					uIConfigCategory2.Add(AddSmallHeader("NFC reader"));
					uIConfigCategory2.Add(AddReadOnlyText(33281, 0, "Device Id"));
					uIConfigCategory2.Add(AddReadOnlyText(33282, 0, "Hardware version"));
					uIConfigCategory2.Add(AddReadOnlyText(33283, 0, "Software version"));
				}
				if (iCULanDevice.isAHPV2 && iCULanDevice.HasProperty(33537) && !iCULanDevice.isEcogDC)
				{
					uIConfigCategory2.Add(AddSmallHeader("Auxiliary board"));
					uIConfigCategory2.Add(AddReadOnlyText(33537, 0, "Device Id"));
					uIConfigCategory2.Add(AddReadOnlyText(33538, 0, "Hardware version"));
					uIConfigCategory2.Add(AddReadOnlyText(33539, 0, "Software version"));
				}
				if (iCULanDevice.HasProperty(8785, 1) && iCULanDevice.GetPropertyBool(8785, 1) && iCULanDevice.HasProperty(33792, 1))
				{
					uIConfigCategory2.Add(AddSmallHeader("Tamper detection"));
					uIConfigCategory2.Add(AddReadOnlyText(33792, 1, "Extension board assembly"));
				}
			}
			if (iCULanDevice.HasProperty(8472))
			{
				UIConfigCategory uIConfigCategory3 = uIConfigurationPanel.AddCategory("Modem Info");
				uIConfigCategory3.Add(AddReadOnlyText(8472, 0, "Modem manufacturer"));
				uIConfigCategory3.Add(AddReadOnlyText(8473, 0, "Modem model"));
				uIConfigCategory3.Add(AddReadOnlyText(8480, 0, "Modem revision"));
				uIConfigCategory3.Add(AddReadOnlyText(8481, 0, "Modem IMEI"));
			}
			UIConfigCategory uIConfigCategory4 = uIConfigurationPanel.AddCategory("License key");
			uIConfigCategory4.Add(AddReadOnlyText(8273, 0));
			uIConfigCategory4.Add(AddText(8609, 0, "Feature license key"));
			ICUProperty property = newDevice.GetProperty(8610, 0);
			if (property != null && property.Value != null)
			{
				IList<string> featureTextLongList = IWSFirmwareFeatures.GetFeatureTextLongList((IWSFirmwareFeatures.Features)uint.MaxValue, iCULanDevice.isAHP, iCULanDevice.isDC);
				IList<string> featureTextLongList2 = IWSFirmwareFeatures.GetFeatureTextLongList((IWSFirmwareFeatures.Features)newDevice.GetPropertyUInt(8610, 0), iCULanDevice.isAHP, iCULanDevice.isDC);
				uIConfigCategory4.Add(AddSmallHeader("Features"));
				foreach (string item in featureTextLongList)
				{
					bool flag2 = featureTextLongList2.Contains(item);
					Xwt.Drawing.Color textColor = (flag2 ? new Xwt.Drawing.Color(0.0, 0.0, 0.0) : new Xwt.Drawing.Color(0.5, 0.5, 0.5));
					Label widget = new Label
					{
						Text = item,
						TextColor = textColor
					};
					GetTable().Add(uIConfigCategory4.AddWidget(widget), m_tableColCounter, m_tableRowCounter, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, m_nLeftMargin);
					GetTable().Add(uIConfigCategory4.AddWidget(new Label(flag2 ? "Unlocked" : "Locked")), m_tableColCounter + 1, m_tableRowCounter++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, m_nLeftMargin);
				}
			}
			uIConfigCategory4.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
			uIConfigCategory4.AddWidget(m_btnUpdateLicense = AddCustomButton("Update license key", "Update your current license key with the latest from our server.", OnUpdateLicenseKeyClicked, true));
			foreach (TimeZoneInfo systemTimeZone in TimeZoneInfo.GetSystemTimeZones())
			{
				int num = (int)systemTimeZone.BaseUtcOffset.TotalMinutes / 6;
				if (!m_dicTimezones.ContainsKey(num.ToString()))
				{
					m_dicTimezones.Add(num.ToString(), systemTimeZone.DisplayName);
				}
				num = (int)systemTimeZone.BaseUtcOffset.TotalMinutes;
				if (!m_dicTimezonesMinutes.ContainsKey(num.ToString()))
				{
					m_dicTimezonesMinutes.Add(num.ToString(), systemTimeZone.DisplayName);
				}
			}
			UIConfigCategory uIConfigCategory5 = uIConfigurationPanel.AddCategory("Location");
			uIConfigCategory5.Add(AddReadOnlyText(8281, 0, "Charger date and time", UIPropertyStringType.DateTime));
			if (newDevice.GetProperty(2125312u).DeviceValue != null)
			{
				uIConfigCategory5.Add(AddCustomSelect(8302, 0, "Time zone", m_dicTimezonesMinutes));
			}
			else
			{
				uIConfigCategory5.Add(AddCustomSelect(8282, 0, "Time zone", m_dicTimezones));
			}
			uIConfigCategory5.Add(AddSelect(8283, 0, 0, 0));
			uIConfigCategory5.Add(AddText(8284, 1, "Latitude"));
			uIConfigCategory5.Add(AddText(8284, 2, "Longitude"));
			uIConfigCategory5.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
			uIConfigCategory5.AddWidget(m_btnSetDate = AddCustomButton("Sync time", "Sync the date and time to the PC time", OnSyncTimeClicked, true));
			m_btnSetDate.Sensitive = newDevice != null;
			if (iCULanDevice.IsUniquePasswordRequired)
			{
				UIConfigCategory uIConfigCategory6 = uIConfigurationPanel.AddCategory("Station Password");
				uIConfigCategory6.Add(AddSmallHeader("Secure Service Access"));
				uIConfigCategory6.Add(AddInfoText("With Secure Service Access (SSA) enabled, an Alfen certified service person can\nconfigure this charging station without the need to share any password.\nThey can also regain access in case of a lost password.\n\nWe recommend enabling SSA in your charging station."));
				m_chkEnableSSA = (UIPropertyCheckbox)uIConfigCategory6.Add(AddCheckBox(8626, 0, 0, "Enable Secure Service Access", null, 2208256u));
				uIConfigCategory6.Add(AddSmallHeader("Passwords"));
				uIConfigCategory6.Add(AddReadOnlyText(8627, 0, "Temp password expiration date", UIPropertyStringType.DateTime));
				if (iCULanDevice.HasProperty(8628))
				{
					uIConfigCategory6.Add(AddReadOnlySelect(8628, 0, 0, 0, "Is default owner password"));
				}
				if (iCULanDevice.EndUserAccessType != EndUserAccessType.NotAvailable)
				{
					uIConfigCategory6.Add(m_txtEndUserAccessType = AddCustomText("Eve Connect access", iCULanDevice.EndUserAccessType.GetDescription()));
					m_txtEndUserAccessType.SetTooltip("This will allow access via Eve Connect\n\nThis is not recommended for publicly accessible chargers.");
				}
				uIConfigCategory6.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
				uIConfigCategory6.AddWidget(m_btnlogout = AddCustomButton("Logout", "Logout from this CS", OnLogout, true));
				uIConfigCategory6.AddWidget(m_btnChangePassword = AddCustomButton("Change password...", "Change the current password", OnChangePassword, true));
				uIConfigCategory6.AddWidget(m_btnTempPassword = AddCustomButton("Temporary password...", "Create temporary password", OnTempPassword, true));
				if (iCULanDevice.EndUserAccessType != EndUserAccessType.NotAvailable)
				{
					uIConfigCategory6.AddWidget(m_btnEndUserAccess = AddCustomButton("Eve Connect access", "Configure Eve Connect access", OnEndUserAccessClicked, true));
					m_btnEndUserAccess.Sensitive = true;
				}
				SetAccessOptions();
				m_btnlogout.Sensitive = m_currentDevice != null;
				m_btnlogout2.Sensitive = m_currentDevice != null;
			}
			else
			{
				m_chkEnableSSA = null;
				m_btnTempPassword = null;
				m_btnChangePassword = null;
				m_btnlogout = null;
				m_btnlogout2 = null;
			}
			if (iCULanDevice.HasProperty(9744))
			{
				UIConfigCategory uIConfigCategory7 = uIConfigurationPanel.AddCategory("Experimental");
				uIConfigCategory7.Add(AddSmallHeader("Alpha release"), advancedProperty: true);
				uIConfigCategory7.Add(AddInfoText("When you enable the 'Allow alpha releases' checkbox,\nyou can install alpha release firmware version on this Charging Station.\nAlpha releases offer you earlier access to new features and improvements\nof upcoming major releases. Please be aware that alpha versions are not\nfinished products and might contain bugs!\n\nOnly enable the alpha release when you understand the risks!"), advancedProperty: true);
				uIConfigCategory7.Add(AddCheckBox(9744, 0, 0, "Allow alpha releases", null, 2494464u), advancedProperty: true);
			}
		}
		UpdateButtons();
		return true;
	}

	private void SetAccessOptions()
	{
		if (m_currentDevice != null)
		{
			if (m_currentDevice.LoginData.IsAdminUser)
			{
				m_chkEnableSSA.SetEnable(fEnable: true);
				m_btnTempPassword.Sensitive = true;
				m_btnChangePassword.Sensitive = true;
			}
			else if (m_currentDevice.LoginData.IsServiceUser)
			{
				m_chkEnableSSA.SetEnable(!m_currentDevice.IsAhpFirmwareEqualOrHigherThan(new Version("2.4.0")));
				m_btnTempPassword.Sensitive = true;
				m_btnChangePassword.Sensitive = true;
			}
			else if (m_currentDevice.LoginData.IsTempUser && (m_currentDevice.IsNg9xxFirmwareEqualOrHigherThan(new Version("7.4.0")) || m_currentDevice.IsAhpFirmwareEqualOrHigherThan(new Version("2.4.0"))))
			{
				m_chkEnableSSA.SetEnable(fEnable: false);
				m_btnTempPassword.Sensitive = false;
				m_btnChangePassword.Sensitive = true;
			}
			else
			{
				m_chkEnableSSA.SetEnable(fEnable: false);
				m_btnTempPassword.Sensitive = false;
				m_btnChangePassword.Sensitive = false;
			}
		}
	}

	private void OnEndUserAccessClicked(object sender, EventArgs e)
	{
		DlgEndUserPin dlgEndUserPin = new DlgEndUserPin(m_currentDevice);
		Command command = dlgEndUserPin.Run(ParentWindow);
		if (command == Command.Ok)
		{
			m_currentDevice.SetEndUserPin(dlgEndUserPin.Pin);
		}
		else if (command == Command.Remove)
		{
			m_currentDevice.DisableEndUserAccess();
		}
		m_txtEndUserAccessType.SetValue(m_currentDevice.EndUserAccessType.GetDescription());
	}

	protected string ParseRevision(int revision, int assembly)
	{
		EBoardRevision eBoardRevision = (EBoardRevision)Enum.ToObject(typeof(EBoardRevision), revision);
		EBoardAssy eBoardAssy = (EBoardAssy)Enum.ToObject(typeof(EBoardAssy), assembly);
		return eBoardRevision.ToString().Split(new char[1] { '_' })[1] + "-" + eBoardAssy.ToString().Split(new char[1] { '_' })[1];
	}

	protected void UpdateButtons()
	{
		bool sensitive = false;
		bool sensitive2 = false;
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			sensitive = currentDevice.FirmwareVersionNumber >= new Version("3.4.0") || currentDevice.isAHP;
			sensitive2 = currentDevice.FirmwareVersionNumber >= new Version("3.3.0") || currentDevice.isAHP;
		}
		m_btnFactoryDefaults.Sensitive = sensitive;
		m_btnUpdateLicense.Sensitive = sensitive2;
	}

	protected override void OnUpdateTick()
	{
		ICUProperty property = m_currentDevice.GetProperty(8281, 0);
		if (property != null && property.Value != null)
		{
			property.SetInitialValue((ulong)property.Value + 1000);
		}
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		if (m_currentDevice == null)
		{
			return true;
		}
		UIPropertyCheckbox chkEnableSSA = m_chkEnableSSA;
		if (chkEnableSSA != null && chkEnableSSA.IsChanged)
		{
			if (!m_chkEnableSSA.IsChecked)
			{
				return MessageDialog.AskQuestion("Are you sure you want to disable Secure Service Access?\n\nBy disabling SSA, Alfen certified service engineers will need your (temporary) password to service your charging station " + m_currentDevice.Identity + ".\n\nAdditionally they cannot help you with resetting your password in case it gets lost.", Command.Yes, Command.No, Command.Cancel) == Command.Yes;
			}
			return MessageDialog.AskQuestion("Are you sure you want to enable Secure Service Access?\n\nWith SSA enabled, an Alfen certified service engineer can configure this charging station (" + m_currentDevice.Identity + ") without the need to share any password.\n\nThey can also recover access in case of a lost password.", Command.Yes, Command.No, Command.Cancel) == Command.Yes;
		}
		UIPropertyBase txtIdentity = m_txtIdentity;
		if (txtIdentity != null && txtIdentity.IsChanged && !new Regex("^[a-zA-Z0-9*\\-_=:+|@.]{1," + (m_currentDevice.isAHP ? 48 : 20) + "}$").IsMatch(m_txtIdentity.GetProperty().Value.ToString()))
		{
			MessageDialog.ShowError(ParentWindow, "The field \"Customer Ident. Number\" is invalid.\nOnly characters, digits and *-_=:+|@. are allowed" + (m_currentDevice.isAHP ? "\nand the maximum length is 48 positions." : "."));
			return false;
		}
		return true;
	}

	protected void CheckIfCSIsInitialize(ICULanDevice lanDev)
	{
		string text = lanDev.GetPropertyString(8275, 0, 0).Trim().ToUpperInvariant();
		string text2 = lanDev.GetPropertyString(8273, 0, 0).Trim().ToUpperInvariant();
		if (!(text == "AL1000"))
		{
			return;
		}
		MainWindow mainWindow = (MainWindow)ParentWindow;
		mainWindow.SuspendUpdateTimers(fSuspend: true);
		try
		{
			if (string.IsNullOrWhiteSpace(text2) || text2 == "UNKNOWN")
			{
				DlgObjectID dlgObjectID = new DlgObjectID(lanDev, m_currentUser);
				dlgObjectID.Run(ParentWindow);
				if (!string.IsNullOrEmpty(dlgObjectID.NewObjectID))
				{
					lanDev.storeProperty(8273, 0, dlgObjectID.NewObjectID);
					lanDev.HostName = dlgObjectID.NewObjectID;
				}
			}
			LoadIsahSettings(lanDev);
			if (!lanDev.isAHP && lanDev.HasProperty(8625) && lanDev.GetPropertyInt(8625, 0) == 0)
			{
				lanDev.storeProperty(8625, 0, 2);
			}
		}
		finally
		{
			mainWindow.SuspendUpdateTimers(fSuspend: false);
		}
	}

	protected void LoadIsahSettings(ICULanDevice lanDev)
	{
		if (lanDev == null)
		{
			return;
		}
		ManualResetEvent finishedEvent = new ManualResetEvent(initialState: false);
		Task task = Task.Run(async () =>
		{
			bool flag = await IWSConnection.TestIWSConnection(AppProperties.IsahSite);
			string defaultMessage = "It seems that this Charging Station is not yet initialized correctly. This Installer tool can initialize this Charging Station to it's factory default settings.";
			if (flag)
			{
				Application.Invoke(() =>
				{
					if (MessageDialog.AskQuestion($"{defaultMessage}\nDo you want to initialize this Charging Station to it's default settings?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
					{
						ResetToFactorySettings(lanDev);
					}
				});
			}
			else
			{
				Application.Invoke(() =>
				{
					MessageDialog.ShowError(ParentWindow, $"{defaultMessage}\nBut there is a problem with the internet connection!\nWe cannot connect to the Alfen server to download the default settings!\nPlease make sure you have a working internet on this PC!");
				});
			}
			finishedEvent.Set();
		});
		while (!finishedEvent.WaitOne(10))
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
		task.GetAwaiter().GetResult();
	}

	private void ResetToFactorySettings(ICULanDevice lanDev, bool askToUpdateLogo = false)
	{
		string identity = lanDev.Identity;
		MainWindow mainWindow = ParentWindow as MainWindow;
		try
		{
			Logger.AddChargerContext(m_currentDevice).Information("Factory default reset started");
			if (mainWindow != null)
			{
				mainWindow.AllowStoreChanges = false;
				mainWindow.Sensitive = false;
			}
			IWSObject objData = new IWSObject();
			if (lanDev.LoadSettingsFromIWS(m_currentUser, AppProperties.IsahSite, m_currentUser.Company, ref objData))
			{
				if (lanDev.storeProperty(8609, 0, objData.Properties.FirstOrDefault((IWSPropertyValue a) => a.Id == 8609)?.Value))
				{
					if (lanDev.isAHP)
					{
						Logger.Debug("{DeviceName} {IpAddress} license key is not implemented yet. Also for now not sure if that requires a reboot to take effect.", lanDev.Name, lanDev.Address);
					}
					else
					{
						Logger.Debug("{DeviceName} {IpAddress} 1st reboot", lanDev.Name, lanDev.IPAddress);
						using DlgReboot dlgReboot = new DlgReboot(lanDev, fAutoStartReboot: true);
						dlgReboot.Run(ParentWindow);
					}
				}
				List<ICUProperty> list = new List<ICUProperty>();
				foreach (IWSPropertyValue property2 in objData.Properties)
				{
					ICUProperty property = lanDev.GetProperty(property2.Id, property2.SubId);
					if (property != null)
					{
						property.Value = property2.Value;
						list.Add(property);
						Logger.Debug("IWS setting property: {Id}={Value}", property2.Id, property2.Value);
					}
					else
					{
						Logger.Debug("Cannot set Isah property: {Id}={Value}", property2.Id, property2.Value);
					}
				}
				(bool, HttpStatusCode, string) tuple = lanDev.Login();
				if (!tuple.Item1)
				{
					Logger.Debug("Cancelled factory default step or user doesn't have the SSA for the provided ACE number. ({Error})", tuple.Item3);
					return;
				}
				lanDev.StoreProperties(list.ToArray());
				if (lanDev.isAHP)
				{
					Logger.Debug("TODO: AHWP-3999 upload logo");
				}
				else if (objData.Logo != null && objData.Logo.Length > 0 && objData.IsPersonalizedDisplay)
				{
					bool flag = true;
					if (askToUpdateLogo)
					{
						flag = MessageDialog.AskQuestion("Do you want to reset the logo to the original logo?", Command.Yes, Command.No, Command.Cancel) == Command.Yes;
					}
					if (flag)
					{
						using MemoryStream memoryStream = new MemoryStream(Convert.FromBase64String(objData.Logo));
						byte[] resourceData = global::ICUFWUCreator.ICUFWUCreator.CreateFWUData(Image.FromStream((Stream)memoryStream), DlgUploadResources.s_nMaxColors, AppProperties.UILanguagesFolder, createCFile: false, "", largeScreen: true);
						lanDev.UploadResource(null, resourceData);
					}
				}
				if (((lanDev.isAHP && lanDev.FirmwareVersionNumber >= new Version(2, 4, 0)) || (!lanDev.isAHP && lanDev.FirmwareVersionNumber >= new Version(7, 4, 0))) && MessageDialog.AskQuestion("Do you want to clear the personal data?", "This will clear all logs, backoffice credentials and will restore the owner password to default.\n\nThis action cannot be undone!", Command.Yes, Command.No) == Command.Yes && !lanDev.ClearPersonalData())
				{
					MessageDialog.ShowError(ParentWindow, "Failed to clear personal data.");
				}
				Thread.Sleep(1500);
				bool flag2 = !lanDev.isAHP;
				Logger.Debug("{DeviceName} {IpAddress} 2nd reboot HardReboot={HardReboot}", lanDev.Name, lanDev.IPAddress, flag2);
				using (DlgReboot dlgReboot2 = new DlgReboot(lanDev, fAutoStartReboot: true, flag2))
				{
					dlgReboot2.Run(ParentWindow);
				}
				Logger.AddChargerContext(m_currentDevice).Information("Factory default reset finished successfully");
				Logger.Debug("Charging Station '{LanName}' is now changed to the factory default settings (the identity has changed to '{Identity}').", identity, lanDev.Identity);
				MessageDialog.ShowMessage(ParentWindow, $"Charging Station '{identity}' is now changed to the factory default settings (the identity has changed to '{lanDev.Identity}').");
			}
			else
			{
				Logger.Debug("Error, failed to set charging Station '{LanName}' back to the factory default settings.", identity);
				MessageDialog.ShowError(ParentWindow, "Error, failed to set charging Station '" + identity + "' back to the factory default settings.");
			}
		}
		catch (Exception ex)
		{
			string text = $"Error '{ex.Message}' while trying to load the default settings.";
			Logger.Error(ex, text);
			MessageDialog.ShowError(ParentWindow, text);
		}
		finally
		{
			if (mainWindow != null)
			{
				mainWindow.AllowStoreChanges = true;
				mainWindow.Sensitive = true;
				mainWindow.RefreshDeviceList();
			}
		}
	}
}
