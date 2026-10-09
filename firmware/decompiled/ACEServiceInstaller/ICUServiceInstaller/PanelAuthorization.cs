using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Net;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelAuthorization : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelAuthorization>();

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catWhiteList;

	protected UIConfigCategory m_catCar;

	protected UIConfigCategory m_catAuthorization;

	protected UIConfigCategory m_catMasterTag;

	protected UIConfigCategory m_catOnlineOffline;

	protected UIConfigCategory m_catDirectPayment;

	protected UIConfigCategory m_catTamper;

	protected ListView m_lstWhitelist;

	protected ListStore m_lsWhitelist;

	protected DataField<string> m_dfTag;

	protected DataField<string> m_dfParent;

	protected DataField<ICUTagStatus> m_dfStatus;

	protected DataField<string> m_dfExpiryDate;

	protected DataField<ICUWhitelistItem> m_dfItem;

	protected Dictionary<string, string> m_dicMasterTagEnable = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicOfflineNFC = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicOnlineNFC = new Dictionary<string, string>();

	protected int m_nAutoAddModeCountDown;

	protected int m_previousWhitelistCount;

	protected Button m_btnRefresh;

	protected Button m_btnClearList;

	protected Button m_btnSaveList;

	protected Button m_btnLoadList;

	protected Button m_btnAddTag;

	protected Button m_btnEditTag;

	protected Button m_btnRemoveTag;

	protected Button m_btnStartTagAddMode;

	protected ProgressBar m_prbUpload;

	protected UIPropertySelect m_cmbBackMainAuthorizationMode;

	protected UIPropertySelect m_cmbMasterTag;

	protected UIPropertyString m_txtPlugAndChargeIdentifier;

	protected UIPropertySelect m_cmbDisconnectAction;

	protected UIPropertyNumber m_numDisconnectTimeout;

	protected UISelect m_cmbOfflineNFCAuthorization;

	protected UIPropertySelect m_cmbOnlineNFCAuthorization;

	protected UIPropertyNumber m_numTimeUnlockNotCharging;

	protected UIPropertyNumber m_numTimeReportNotCharging;

	protected UIPropertyLabel m_lblInfoTimeNotCharging;

	protected UIPropertyNumber m_numConnectionTimeout;

	protected UIPropertyNumber m_numAuthorizationTimeout;

	protected UIPropertyCheckbox m_chkGiroEReady;

	protected UIPropertyCheckbox m_chkRestartOutage;

	protected UIPropertyNumber m_numOutageDuration;

	protected UIPropertyString m_txtQRCodeURLSocket1;

	protected UIPropertyString m_txtQRCodeURLSocket2;

	protected UICheckbox m_chkDirectPaymentOptionsEnabled;

	protected UICheckbox m_chkExternalPaymentOptionsEnabled;

	protected UICheckbox m_chkGiroEEnabled;

	protected UICheckbox m_chkQRCodeEnabled;

	protected ICUProperty m_currentDirectPaymentOptions;

	protected UIPropertyLabel m_lblQrBackOfficeWarning;

	protected Stopwatch m_swLastClickedTime = Stopwatch.StartNew();

	protected static int s_nDoubleClickTimeMilli = 300;

	protected static string s_sNoExpiryDate = "1970-01-01";

	protected BackgroundWorker m_bgwUpload = new BackgroundWorker();

	private ICULanDevice LanDevice => m_currentDevice;

	public PanelAuthorization(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Authorization";
		Tooltip = "Authorization";
		IconName = "whitelist.png";
		m_dicMasterTagEnable.Add("0", "Disabled");
		m_dicMasterTagEnable.Add("1", "Enabled");
		m_dicOfflineNFC.Add("0", "Refuse all tags");
		m_dicOfflineNFC.Add("1", "Accept known valid tags");
		m_dicOfflineNFC.Add("3", "Accept all tags");
		m_dicOnlineNFC.Add("0", "Pre Authorize with local lists");
		m_dicOnlineNFC.Add("1", "Wait for authorization by BackOffice");
		StartUpdateTimer(1000);
		m_bgwUpload.WorkerReportsProgress = true;
		m_bgwUpload.DoWork += OnUploadDoWork;
		m_bgwUpload.ProgressChanged += OnUploadProgressChanged;
		m_bgwUpload.RunWorkerCompleted += OnUploadCompleted;
	}

	private void WhitelistUpdated(object sender, WhiteListArgs args)
	{
		if (m_currentDevice != args.Device || !IsPanelVisible)
		{
			args.Device.Whitelist.ClearAllEvents();
			args.Device.Whitelist.StopReading();
		}
		else
		{
			if (args.CancellationPending)
			{
				return;
			}
			Application.Invoke(() =>
			{
				foreach (ICUWhitelistItem item in args.Whitelist)
				{
					m_lsWhitelist?.SetValues(m_lsWhitelist.AddRow(), m_dfTag, item.Tag, m_dfParent, item.ParentTag, m_dfStatus, item.Status, m_dfExpiryDate, item.ExpireDate.ToString(), m_dfItem, item);
				}
			});
		}
	}

	private void WhiteListCompleted(object sender, WhiteListArgs args)
	{
		if (m_nAutoAddModeCountDown > 0)
		{
			if (m_previousWhitelistCount != args.Device.Whitelist.Whitelist.Count)
			{
				m_previousWhitelistCount = args.Device.Whitelist.Whitelist.Count;
				m_nAutoAddModeCountDown = 10;
			}
		}
		else
		{
			UpdateButtons();
		}
	}

	private void OnWhitelistClicked(object sender, ButtonEventArgs e)
	{
		if (m_swLastClickedTime.ElapsedMilliseconds > s_nDoubleClickTimeMilli)
		{
			m_swLastClickedTime.Restart();
		}
		else if (m_nAutoAddModeCountDown == 0)
		{
			OnEditTagClicked(sender, null);
		}
	}

	private bool IsGiroEAvailable()
	{
		UICheckbox chkDirectPaymentOptionsEnabled = m_chkDirectPaymentOptionsEnabled;
		if (chkDirectPaymentOptionsEnabled != null && chkDirectPaymentOptionsEnabled.IsChecked)
		{
			bool flag = Convert.ToInt32(m_cmbBackMainAuthorizationMode.GetValue()) == 2;
			if ((LanDevice.IsEichrechtEnabled & flag) && LanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.RFIDReader))
			{
				return LanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.Payment_Options);
			}
			return false;
		}
		return false;
	}

	private bool IsQRAvailable()
	{
		UICheckbox chkDirectPaymentOptionsEnabled = m_chkDirectPaymentOptionsEnabled;
		if (chkDirectPaymentOptionsEnabled != null && chkDirectPaymentOptionsEnabled.IsChecked)
		{
			return LanDevice.HasDisplay;
		}
		return false;
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		newDevice.UpdateCategories("comm", "generic", "display");
		iCULanDevice.Whitelist.ClearAllEvents();
		iCULanDevice.Whitelist.WhitelistUpdated += WhitelistUpdated;
		iCULanDevice.Whitelist.WhitelistCompleted += WhiteListCompleted;
		m_prbUpload = new ProgressBar
		{
			Visible = false,
			MinHeight = 16.0
		};
		m_dfTag = new DataField<string>();
		m_dfParent = new DataField<string>();
		m_dfStatus = new DataField<ICUTagStatus>();
		m_dfExpiryDate = new DataField<string>();
		m_dfItem = new DataField<ICUWhitelistItem>();
		m_lsWhitelist = new ListStore(m_dfTag, m_dfParent, m_dfStatus, m_dfExpiryDate, m_dfItem);
		m_lstWhitelist = new ListView
		{
			DataSource = m_lsWhitelist,
			SelectionMode = SelectionMode.Single
		};
		m_lstWhitelist.Columns.Add(new ListViewColumn("Tag             ", new TextCellView(m_dfTag)));
		m_lstWhitelist.Columns.Add(new ListViewColumn("Parent          ", new TextCellView(m_dfParent)));
		m_lstWhitelist.Columns.Add(new ListViewColumn("Status          ", new TextCellView(m_dfStatus)));
		m_lstWhitelist.Columns.Add(new ListViewColumn("Expiry date     ", new TextCellView(m_dfExpiryDate)));
		m_lstWhitelist.GridLinesVisible = GridLines.Horizontal;
		ListView lstWhitelist = m_lstWhitelist;
		double num = (m_lstWhitelist.WidthRequest = 100.0);
		lstWhitelist.HeightRequest = num;
		m_lstWhitelist.SelectionChanged += OnTagSelectionChanged;
		m_lstWhitelist.MinHeight = 240.0;
		m_lstWhitelist.SelectionMode = SelectionMode.Single;
		m_lstWhitelist.ButtonReleased += OnWhitelistClicked;
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			m_catWhiteList = m_configPanel.AddCategory("Whitelist");
			m_catWhiteList.AddWidget(AddToolBox());
			m_btnAddTag = AddToolImageButton("add24.png", "Add a new tag", OnAddTagClicked, rightSide: false);
			m_btnEditTag = AddToolImageButton("edit24.png", "Edit an existing tag", OnEditTagClicked, rightSide: false);
			m_btnRemoveTag = AddToolImageButton("remove24.png", "Remove an existing tag", OnRemoveTagsClicked, rightSide: false);
			m_btnClearList = AddToolImageButton("closed-trash-can.png", "Clear the entire whitelist", OnClearClicked);
			m_btnSaveList = AddToolImageButton("save.png", "Save all tags to a file", OnSaveListClicked);
			m_btnLoadList = AddToolImageButton("open-folder.png", "Load all tags from a file", OnLoadListClicked);
			m_btnRefresh = AddToolImageButton("refresh24.png", "Refresh the whitelist", OnRefreshClicked);
			GetTable().Add(m_lstWhitelist, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
			m_tableRowCounter++;
			m_catWhiteList.AddWidget(m_lstWhitelist);
			GetTable().Add(m_prbUpload, 0, m_tableRowCounter, 1, 3, hexpand: true);
			m_tableRowCounter++;
			m_catWhiteList.AddWidget(m_prbUpload);
			m_catWhiteList.AddWidget(AddButtonBox(fExpandVert: false, fExpandHor: false, forceNewInstace: true));
			m_catWhiteList.AddWidget(m_btnStartTagAddMode = AddCustomButton("Auto add", "Automatically add tags by presenting them to the NFC reader", OnAutoAddTagModeClicked, rightSide: false));
			m_catCar = m_configPanel.AddCategory("Car");
			m_catCar.Add(m_cmbDisconnectAction = (UIPropertySelect)AddSelect(8503, 0, 0, 0, "Disconnect action"));
			m_cmbDisconnectAction.SetTooltip("Action that should be executed when the charging cable is removed from the EV");
			m_catCar.Add(m_numDisconnectTimeout = (UIPropertyNumber)AddCustomNumber(8502, 0, 0, "Disconnect timeout (s)"));
			m_numDisconnectTimeout.SetTooltip("Number of seconds to wait before the disconnect action is executed when an EV is disconnected");
			m_numDisconnectTimeout.SetValueMinMax(0.0, 32767.0);
			if (iCULanDevice.HasProperty(9728) && iCULanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.ISO15118))
			{
				m_catCar.Add(AddCheckBox(9728, 0, 0, "Enable AutoCharge"));
			}
			if (iCULanDevice.HasProperty(8580))
			{
				m_catCar.Add(m_numTimeReportNotCharging = (UIPropertyNumber)AddCustomNumber(8580, 0, 0, "Time to report not charging (s)"));
			}
			if (iCULanDevice.HasProperty(8552))
			{
				m_catCar.Add(m_numTimeUnlockNotCharging = (UIPropertyNumber)AddCustomNumber(8552, 0, 0, "Time to unlock not charging (s)"));
				m_catCar.Add(m_lblInfoTimeNotCharging = (UIPropertyLabel)AddWarningText("The time to report when not charging is higher or equal then the time to unlock the cable.", "", 0, null, 2));
				m_lblInfoTimeNotCharging.Hide = true;
			}
			m_catAuthorization = m_configPanel.AddCategory("Authorization");
			Dictionary<string, string> dictionary = new Dictionary<string, string>();
			if (!LanDevice.IsEichrechtEnabled)
			{
				dictionary.Add("0", "Plug & Charge");
			}
			dictionary.Add("2", "RFID");
			m_catAuthorization.Add(m_cmbBackMainAuthorizationMode = (UIPropertySelect)AddCustomSelect(8486, 0, "Authorization mode", dictionary));
			m_cmbBackMainAuthorizationMode.Changed += OnMainAuthorizationMethodChanged;
			m_catAuthorization.Add(m_txtPlugAndChargeIdentifier = (UIPropertyString)AddText(8291, 0, "Plug & charge ID"));
			m_txtPlugAndChargeIdentifier.SetEnable(!LanDevice.IsEichrechtEnabled);
			m_catAuthorization.Add(AddCheckBox(8507, 0, 0, "White list enabled"));
			m_catAuthorization.Add(AddCheckBox(8509, 0, 0, "Local list enabled"));
			m_catAuthorization.Add(m_chkRestartOutage = (UIPropertyCheckbox)AddCheckBox(8542, 0, 0, "Restart after Power Outage"));
			m_catAuthorization.Add(m_numOutageDuration = (UIPropertyNumber)AddCustomNumber(8553, 0, 0, "Re-authorize after Power Outage (s)"));
			m_numOutageDuration.SetValueMinMax(0.0, 3600.0);
			if (m_numOutageDuration.GetProperty() != null && !m_numOutageDuration.GetProperty().ReadOnly)
			{
				m_numOutageDuration.SetEnable(iCULanDevice.GetPropertyBool(8542, 0));
			}
			uint propertyUInt = iCULanDevice.GetPropertyUInt(8610, 0);
			if (iCULanDevice.HasProperty(8584) && LanDevice.IsEichrechtEnabled && IWSFirmwareFeatures.IsFeatureUnlocked(iCULanDevice.FirmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.RFIDReader, iCULanDevice.isAHP) && IWSFirmwareFeatures.IsFeatureUnlocked(iCULanDevice.FirmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.Payment_Options, iCULanDevice.isAHP))
			{
				m_catAuthorization.Add(m_chkGiroEReady = (UIPropertyCheckbox)AddCheckBox(8584, 0, 0, "Giro-e ready"));
			}
			m_catAuthorization.Add(AddCheckBox(8347, 0, 0, "Remote transaction requests"));
			m_catAuthorization.Add(AddCheckBox(8341, 0, 0, "Stop transaction on invalid tag"), advancedProperty: true);
			m_catAuthorization.Add(AddCheckBox(8340, 0, 0, "Abort concurrent transaction"), advancedProperty: true);
			m_catAuthorization.Add(m_numConnectionTimeout = (UIPropertyNumber)AddCustomNumber(8501, 0, 0, "Connection timeout (s)"));
			m_numConnectionTimeout.SetValueMinMax(0.0, 32767.0);
			if (iCULanDevice.HasProperty(8511))
			{
				m_catAuthorization.Add(m_numAuthorizationTimeout = (UIPropertyNumber)AddCustomNumber(8511, 0, 0, "Authorization timeout (s)"));
				m_numConnectionTimeout.SetValueMinMax(0.0, 32767.0);
			}
			m_catOnlineOffline = m_configPanel.AddCategory("Online/Offline");
			m_catOnlineOffline.Add(m_cmbOfflineNFCAuthorization = (UISelect)AddPlainSelect("Offline action", m_dicOfflineNFC, "", null, 2172672u));
			m_catOnlineOffline.Add(m_cmbOnlineNFCAuthorization = (UIPropertySelect)AddCustomSelect(8508, 0, "Online action", m_dicOnlineNFC));
			int num3 = 0;
			if (Enum.IsDefined(typeof(EOfflineAuthorisationMethod), (newDevice.GetPropertyInt(8487, 0) << 1) + newDevice.GetPropertyInt(8510, 0)))
			{
				num3 = (newDevice.GetPropertyInt(8487, 0) << 1) + newDevice.GetPropertyInt(8510, 0);
			}
			m_cmbOfflineNFCAuthorization.CustomValue = num3.ToString();
			if (((ICULanDevice)newDevice).MasterTag.IsFeatureSupported())
			{
				m_catMasterTag = m_configPanel.AddCategory("Master key");
				m_catMasterTag.Add(m_cmbMasterTag = (UIPropertySelect)AddCustomSelect(9216, 1, "Master key mode", m_dicMasterTagEnable));
			}
			m_catDirectPayment = m_configPanel.AddCategory("Direct payment solutions");
			if (LanDevice.HasProperty(8594))
			{
				m_currentDirectPaymentOptions = LanDevice.GetProperty(8594, 0);
				EDirectPaymentOptions eDirectPaymentOptions = (EDirectPaymentOptions)Convert.ToInt32(m_currentDirectPaymentOptions.Value);
				m_chkDirectPaymentOptionsEnabled = (UICheckbox)m_catDirectPayment.Add(AddPlainBool("Direct payment solutions", eDirectPaymentOptions != EDirectPaymentOptions.None));
				m_chkDirectPaymentOptionsEnabled.CustomValue = eDirectPaymentOptions != EDirectPaymentOptions.None;
				m_catDirectPayment.Add(m_chkGiroEEnabled = (UICheckbox)AddPlainBool("Giro-E", eDirectPaymentOptions.HasFlag(EDirectPaymentOptions.GiroE)));
				m_chkGiroEEnabled.CustomValue = m_chkGiroEEnabled.IsChecked;
				m_chkGiroEEnabled.SetEnable(IsGiroEAvailable());
				m_catDirectPayment.Add(m_chkExternalPaymentOptionsEnabled = (UICheckbox)AddPlainBool("External payment options", eDirectPaymentOptions.HasFlag(EDirectPaymentOptions.OTS)));
				m_chkExternalPaymentOptionsEnabled.CustomValue = m_chkExternalPaymentOptionsEnabled.IsChecked;
				m_chkExternalPaymentOptionsEnabled.SetEnable(m_chkDirectPaymentOptionsEnabled.IsChecked);
				m_catDirectPayment.Add(m_chkQRCodeEnabled = (UICheckbox)AddPlainBool("QR codes", eDirectPaymentOptions.HasFlag(EDirectPaymentOptions.QRCode)));
				m_chkQRCodeEnabled.CustomValue = m_chkQRCodeEnabled.IsChecked;
				m_chkQRCodeEnabled.SetEnable(IsQRAvailable());
				m_chkQRCodeEnabled.Changed += OnQRCodeEnabledChanged;
				m_catDirectPayment.Add(m_txtQRCodeURLSocket1 = (UIPropertyString)AddText(8567, 0, "QR Code URL Socket 1"));
				m_txtQRCodeURLSocket1.SetEnable(m_chkQRCodeEnabled.IsChecked);
				m_catDirectPayment.Add(m_txtQRCodeURLSocket2 = (UIPropertyString)AddText(8568, 0, "QR Code URL Socket 2"));
				m_txtQRCodeURLSocket2.SetEnable(m_chkQRCodeEnabled.IsChecked && newDevice.NumberOfSockets > 1);
				m_txtQRCodeURLSocket2.Hide = newDevice.NumberOfSockets < 2;
				m_catDirectPayment.Add(m_lblQrBackOfficeWarning = (UIPropertyLabel)AddLabelText("To use QR codes, an active back office connection is required.\nPlease check if there is a back office configured.", EUILabelType.Warning));
				m_lblQrBackOfficeWarning.SetEnable(m_chkQRCodeEnabled.IsChecked);
				m_configPanel.ShowCategory(m_catDirectPayment, iCULanDevice.HasDisplay && iCULanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.Payment_Options));
			}
			m_catTamper = m_configPanel.AddCategory("Tamper detection");
			if (LanDevice.HasProperty(8785, 1) && LanDevice.GetPropertyBool(8785, 1))
			{
				m_catTamper.Add(AddSelect(8785, 2, 0, 0));
			}
		}
		UpdateButtons();
		FillWhitelist();
		return false;
	}

	private void OnQRCodeEnabledChanged(object sender, EventArgs e)
	{
		CanEnableQRCodeFields();
	}

	public override bool OnUpdateControls(string pageID = "")
	{
		if (m_numTimeReportNotCharging != null && m_numTimeUnlockNotCharging != null)
		{
			int num = Convert.ToInt32(m_numTimeReportNotCharging.GetValue());
			int num2 = Convert.ToInt32(m_numTimeUnlockNotCharging.GetValue());
			m_lblInfoTimeNotCharging.SetVisible(num2 != 0 && num >= num2);
		}
		if (m_chkRestartOutage.IsChanged && m_numOutageDuration.GetProperty() != null && !m_numOutageDuration.GetProperty().ReadOnly)
		{
			m_numOutageDuration.SetEnable(m_chkRestartOutage.IsChecked);
		}
		if (m_chkDirectPaymentOptionsEnabled != null && m_chkQRCodeEnabled != null && m_chkExternalPaymentOptionsEnabled != null && m_chkGiroEEnabled != null)
		{
			m_chkQRCodeEnabled.SetEnable(IsQRAvailable());
			m_chkExternalPaymentOptionsEnabled.SetEnable(m_chkDirectPaymentOptionsEnabled.IsChecked);
			m_chkGiroEEnabled.SetEnable(IsGiroEAvailable());
			if (!m_chkDirectPaymentOptionsEnabled.IsChecked)
			{
				m_chkQRCodeEnabled.SetValue(false);
				m_chkExternalPaymentOptionsEnabled.SetValue(false);
				m_chkGiroEEnabled.SetValue(false);
				m_txtQRCodeURLSocket1.SetValue(string.Empty);
				m_txtQRCodeURLSocket2.SetValue(string.Empty);
			}
		}
		return true;
	}

	private void UpdateButtons()
	{
		m_btnStartTagAddMode.Sensitive = m_currentDevice != null && !m_currentDevice.isAHP;
		m_btnAddTag.Sensitive = m_currentDevice != null;
		bool sensitive = false;
		if (m_lsWhitelist != null && m_lstWhitelist != null)
		{
			sensitive = m_lsWhitelist.RowCount > 0 && m_lstWhitelist.SelectedRow != -1;
		}
		m_btnRemoveTag.Sensitive = sensitive;
		m_btnEditTag.Sensitive = sensitive;
		m_btnRefresh.Sensitive = m_currentDevice != null;
		m_btnClearList.Sensitive = m_currentDevice != null;
		m_btnSaveList.Sensitive = m_currentDevice != null;
		m_btnLoadList.Sensitive = m_currentDevice != null;
		if (m_currentDevice != null)
		{
			bool enable = true;
			EAuthorisationMethod eAuthorisationMethod = EAuthorisationMethod.AUTHORIZE_NFCREADER;
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice != null)
			{
				enable = IWSFirmwareFeatures.IsFeatureUnlocked(currentDevice.FirmwareVersionNumber, currentDevice.GetPropertyUInt(8610, 0), IWSFirmwareFeatures.Features.RFIDReader, currentDevice.isAHP);
				eAuthorisationMethod = (EAuthorisationMethod)currentDevice.GetPropertyInt(8486, 0);
				CanEnableQRCodeFields();
			}
			m_cmbBackMainAuthorizationMode.SetEnable(enable);
			m_txtPlugAndChargeIdentifier.SetEnable(eAuthorisationMethod == EAuthorisationMethod.AUTHORIZE_PLUG_AND_CHARGE);
			if (m_chkGiroEReady != null)
			{
				EGiroEState propertyInt = (EGiroEState)m_currentDevice.GetPropertyInt(8584, 0);
				m_chkGiroEReady.SetEnable(eAuthorisationMethod == EAuthorisationMethod.AUTHORIZE_CANBUS && propertyInt != EGiroEState.OFFLINE);
			}
		}
	}

	protected override void OnUpdateTick()
	{
		if (m_nAutoAddModeCountDown <= 0)
		{
			return;
		}
		Application.Invoke(() =>
		{
			if (m_currentDevice != null && m_currentDevice.IsConnected && m_currentDevice.LastHttpStatusCode == HttpStatusCode.OK)
			{
				FillWhitelist(forceRefresh: false, refresh: true);
			}
			m_nAutoAddModeCountDown--;
			m_prbUpload.Fraction = (double)m_nAutoAddModeCountDown / 10.0;
			m_btnStartTagAddMode.Label = $"Auto add ({m_nAutoAddModeCountDown})";
			if (m_nAutoAddModeCountDown == 0)
			{
				UpdateButtons();
				m_btnStartTagAddMode.Label = "Auto add";
				m_prbUpload.Visible = false;
			}
		});
	}

	private void OnMainAuthorizationMethodChanged(object sender, EventArgs e)
	{
		EAuthorisationMethod eAuthorisationMethod = (EAuthorisationMethod)Convert.ToInt32(m_cmbBackMainAuthorizationMode.GetValue());
		m_txtPlugAndChargeIdentifier.SetEnable(eAuthorisationMethod == EAuthorisationMethod.AUTHORIZE_PLUG_AND_CHARGE);
		m_chkGiroEReady?.SetEnable(eAuthorisationMethod == EAuthorisationMethod.AUTHORIZE_CANBUS);
		m_chkGiroEEnabled?.SetEnable(IsGiroEAvailable());
	}

	private void OnDirectPaymentOptionsChanged(object sender, EventArgs e)
	{
		CanEnableQRCodeFields();
	}

	private void CanEnableQRCodeFields()
	{
		if (m_chkQRCodeEnabled != null)
		{
			bool flag = (bool)m_chkQRCodeEnabled.GetValue();
			m_txtQRCodeURLSocket1.SetEnable(flag);
			m_txtQRCodeURLSocket2.SetEnable(flag && LanDevice.NumberOfSockets > 1);
			m_lblQrBackOfficeWarning.SetEnable(flag);
			if (!flag)
			{
				m_txtQRCodeURLSocket1.SetValue(string.Empty);
				m_txtQRCodeURLSocket2.SetValue(string.Empty);
			}
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
		List<ICUProperty> list = new List<ICUProperty>();
		if (m_currentDirectPaymentOptions != null)
		{
			if (m_chkExternalPaymentOptionsEnabled.IsEnabled())
			{
				m_chkExternalPaymentOptionsEnabled.CustomValue = m_chkExternalPaymentOptionsEnabled.IsChecked;
			}
			if (m_chkQRCodeEnabled.IsEnabled())
			{
				m_chkQRCodeEnabled.CustomValue = m_chkQRCodeEnabled.IsChecked;
			}
			if (m_chkExternalPaymentOptionsEnabled.IsEnabled())
			{
				m_chkExternalPaymentOptionsEnabled.CustomValue = m_chkExternalPaymentOptionsEnabled.IsChecked;
			}
			if (m_chkGiroEEnabled.IsEnabled())
			{
				m_chkGiroEEnabled.CustomValue = m_chkGiroEEnabled.IsChecked;
			}
			UICheckbox chkDirectPaymentOptionsEnabled = m_chkDirectPaymentOptionsEnabled;
			if (chkDirectPaymentOptionsEnabled != null && chkDirectPaymentOptionsEnabled.IsChecked)
			{
				m_chkDirectPaymentOptionsEnabled.CustomValue = m_chkDirectPaymentOptionsEnabled.IsChecked;
				int currentValue = 0;
				currentValue = Flags.UpdateFlag(currentValue, 2, m_chkQRCodeEnabled?.IsChecked ?? false);
				UICheckbox chkQRCodeEnabled = m_chkQRCodeEnabled;
				if (chkQRCodeEnabled != null && !chkQRCodeEnabled.IsChecked)
				{
					m_txtQRCodeURLSocket1.SetValue(string.Empty);
					m_txtQRCodeURLSocket2.SetValue(string.Empty);
				}
				else
				{
					m_chkQRCodeEnabled.CustomValue = m_chkQRCodeEnabled.IsChecked;
				}
				currentValue = Flags.UpdateFlag(currentValue, 1, m_chkExternalPaymentOptionsEnabled?.IsChecked ?? false);
				currentValue = Flags.UpdateFlag(currentValue, 4, m_chkGiroEEnabled?.IsChecked ?? false);
				m_currentDirectPaymentOptions.Value = currentValue;
			}
			else
			{
				m_chkQRCodeEnabled.CustomValue = m_chkQRCodeEnabled.IsChecked;
				m_currentDirectPaymentOptions.Value = 0;
				m_txtQRCodeURLSocket1.SetValue(string.Empty);
				m_txtQRCodeURLSocket2.SetValue(string.Empty);
			}
			list.Add(m_currentDirectPaymentOptions);
		}
		if (m_cmbBackMainAuthorizationMode != null)
		{
			if (Convert.ToInt32(m_cmbBackMainAuthorizationMode.GetValue()) != 0)
			{
				m_txtPlugAndChargeIdentifier.SetValue("");
			}
			else
			{
				if (m_txtPlugAndChargeIdentifier.GetValue() != null && m_txtPlugAndChargeIdentifier.GetValue().ToString().Length < 8)
				{
					MessageDialog.ShowError(ParentWindow, string.Format("Error! The plug & charge identifier should be at least 8 characters long!", Array.Empty<object>()));
					return false;
				}
				if (m_chkGiroEReady != null)
				{
					ICUProperty property = m_currentDevice.GetProperty(8584, 0);
					if (Enum.TryParse<EGiroEState>(property.Value.ToString(), ignoreCase: true, out var result) && result != EGiroEState.OFFLINE)
					{
						property.Value = 0;
						list.Add(property);
					}
				}
			}
		}
		if (m_cmbOfflineNFCAuthorization != null && m_cmbOfflineNFCAuthorization.IsChanged)
		{
			Enum.TryParse<EOfflineAuthorisationMethod>(m_cmbOfflineNFCAuthorization.GetValue().ToString(), out var result2);
			ICUProperty property2 = m_currentDevice.GetProperty(8487, 0);
			ICUProperty property3 = m_currentDevice.GetProperty(8510, 0);
			switch (result2)
			{
			case EOfflineAuthorisationMethod.OFFLINE_ACCEPT_KNOWN:
				property2.Value = false;
				property3.Value = true;
				break;
			case EOfflineAuthorisationMethod.OFFLINE_ACCEPT_ALL:
				property2.Value = true;
				property3.Value = true;
				break;
			default:
				property2.Value = false;
				property3.Value = false;
				break;
			}
			list.Add(property2);
			list.Add(property3);
		}
		if (list.Count > 0)
		{
			m_currentDevice.StoreChangedProperties();
		}
		return true;
	}

	public override void OnPostSaveChanges()
	{
		if (m_currentDevice != null)
		{
			int num = (m_currentDevice.GetPropertyInt(8487, 0) << 1) + m_currentDevice.GetPropertyInt(8510, 0);
			m_cmbOfflineNFCAuthorization.CustomValue = num.ToString();
		}
		base.OnPostSaveChanges();
	}

	public override void OnRevertChanges()
	{
		if (m_currentDevice != null)
		{
			int num = 0;
			if (Enum.IsDefined(typeof(EOfflineAuthorisationMethod), (m_currentDevice.GetPropertyInt(8487, 0) << 1) + m_currentDevice.GetPropertyInt(8510, 0)))
			{
				num = (m_currentDevice.GetPropertyInt(8487, 0) << 1) + m_currentDevice.GetPropertyInt(8510, 0);
			}
			m_cmbOfflineNFCAuthorization.CustomValue = num.ToString();
		}
		base.OnRevertChanges();
	}

	private void FillWhitelist(bool forceRefresh = false, bool refresh = false)
	{
		try
		{
			m_lsWhitelist.Clear();
			if (m_configPanel?.CurrentCategory == m_catWhiteList)
			{
				m_configPanel?.ShowContent(m_catWhiteList);
			}
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice == null)
			{
				return;
			}
			if (currentDevice.MasterTag.IsFeatureSupported())
			{
				string tag = currentDevice.MasterTag.Tag;
				if (!string.IsNullOrWhiteSpace(tag))
				{
					ICUWhitelistItem iCUWhitelistItem = new ICUWhitelistItem
					{
						Tag = tag,
						Status = ICUTagStatus.MasterCard
					};
					m_lsWhitelist.SetValues(m_lsWhitelist.AddRow(), m_dfTag, iCUWhitelistItem.Tag, m_dfParent, iCUWhitelistItem.ParentTag, m_dfStatus, iCUWhitelistItem.Status, m_dfExpiryDate, iCUWhitelistItem.ExpireDate.ToString(), m_dfItem, iCUWhitelistItem);
				}
			}
			bool flag = false;
			if (forceRefresh)
			{
				flag = currentDevice.Whitelist.Read(forceRestart: true);
			}
			else if (((currentDevice.Whitelist.Whitelist.Count == 0) | refresh) && !currentDevice.Whitelist.IsUpdating)
			{
				flag = currentDevice.Whitelist.Read();
			}
			else
			{
				foreach (ICUWhitelistItem item in currentDevice.Whitelist.Whitelist)
				{
					int row = m_lsWhitelist.AddRow();
					m_lsWhitelist.SetValues(row, m_dfTag, item.Tag, m_dfParent, item.ParentTag, m_dfStatus, item.Status, m_dfExpiryDate, item.ExpireDate.ToString(), m_dfItem, item);
				}
			}
			if (flag)
			{
				m_btnLoadList.Sensitive = false;
				m_btnSaveList.Sensitive = false;
				m_btnClearList.Sensitive = false;
				m_btnAddTag.Sensitive = false;
				m_btnRemoveTag.Sensitive = false;
				m_btnEditTag.Sensitive = false;
			}
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "");
		}
	}

	private void OnTagSelectionChanged(object sender, EventArgs e)
	{
		if (m_nAutoAddModeCountDown == 0)
		{
			UpdateButtons();
		}
	}

	private void OnAddTagClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		DlgModifyTag dlgModifyTag = new DlgModifyTag(null, currentDevice.MasterTag.Enabled);
		if (dlgModifyTag.Run(ParentWindow) == Command.Ok)
		{
			ICUWhitelistItem whitelistItem = dlgModifyTag.WhitelistItem;
			if (whitelistItem != null)
			{
				UpdateWhitelistAndMasterTag(whitelistItem);
				FillWhitelist(forceRefresh: true);
			}
		}
	}

	private void UpdateWhitelistAndMasterTag(ICUWhitelistItem wl)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		if (wl.Status == ICUTagStatus.MasterCard)
		{
			UpdateMasterTag(currentDevice, wl);
			return;
		}
		if (wl.Tag == currentDevice.MasterTag.Tag)
		{
			currentDevice.MasterTag.Clear();
		}
		currentDevice.Whitelist.UpdateOrAdd(wl.Tag, wl.ParentTag, wl.Status, wl.HasExpiryDate ? wl.ExpiryDate.ToString("yyyy-MM-dd") : s_sNoExpiryDate);
	}

	private void OnEditTagClicked(object sender, EventArgs e)
	{
		int selectedRow = m_lstWhitelist.SelectedRow;
		if (selectedRow < 0)
		{
			return;
		}
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		DlgModifyTag dlgModifyTag = new DlgModifyTag(m_lsWhitelist.GetValue(selectedRow, m_dfItem), currentDevice.MasterTag.Enabled);
		if (dlgModifyTag.Run(ParentWindow) == Command.Ok)
		{
			ICUWhitelistItem whitelistItem = dlgModifyTag.WhitelistItem;
			if (whitelistItem != null)
			{
				UpdateWhitelistAndMasterTag(whitelistItem);
				FillWhitelist(forceRefresh: true);
			}
		}
	}

	private void UpdateMasterTag(ICULanDevice lanDev, ICUWhitelistItem wl)
	{
		if (lanDev == null)
		{
			return;
		}
		if (wl != null && !string.IsNullOrEmpty(wl.Tag))
		{
			lanDev.MasterTag.Set(wl.Tag);
			if (lanDev.Whitelist.ContainsTag(wl.Tag))
			{
				lanDev.Whitelist.Remove(wl.Tag);
			}
		}
		FillWhitelist(forceRefresh: true);
	}

	private void OnRemoveTagsClicked(object sender, EventArgs e)
	{
		int selectedRow = m_lstWhitelist.SelectedRow;
		if (selectedRow < 0)
		{
			return;
		}
		ICUWhitelistItem value = m_lsWhitelist.GetValue(selectedRow, m_dfItem);
		if (value == null || MessageDialog.AskQuestion($"Are you sure you want to remove tag '{value.Tag}'?", Command.Yes, Command.No, Command.Cancel) != Command.Yes)
		{
			return;
		}
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			if (value.Tag == currentDevice.MasterTag.Tag)
			{
				currentDevice.MasterTag.Clear();
			}
			else
			{
				currentDevice.Whitelist.Remove(value.Tag);
			}
			FillWhitelist(forceRefresh: true);
		}
	}

	private void OnRefreshClicked(object sender, EventArgs e)
	{
		FillWhitelist(forceRefresh: true);
	}

	private void OnClearClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null && MessageDialog.AskQuestion("Are you sure you want to clear the entire whitelist?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
		{
			currentDevice.MasterTag.Clear();
			if (currentDevice.Whitelist.Clear())
			{
				FillWhitelist(forceRefresh: true);
			}
		}
	}

	private void OnSaveListClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			SaveFileDialog saveFileDialog = new SaveFileDialog("Save Whitelist As")
			{
				InitialFileName = m_currentDevice.Identification + ".csv",
				Multiselect = false
			};
			saveFileDialog.Filters.Add(new FileDialogFilter("Whitelist files", "*.csv"));
			saveFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
			if (saveFileDialog.Run(ParentWindow))
			{
				currentDevice.Whitelist.SaveToCSV(saveFileDialog.FileName);
			}
		}
	}

	private void OnLoadListClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			OpenFileDialog openFileDialog = new OpenFileDialog("Open Whitelist File")
			{
				Multiselect = false
			};
			openFileDialog.Filters.Add(new FileDialogFilter("Whitelist files", "*.csv"));
			openFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
			if (openFileDialog.Run(ParentWindow))
			{
				m_btnLoadList.Sensitive = false;
				m_btnSaveList.Sensitive = false;
				m_btnAddTag.Sensitive = false;
				m_btnRemoveTag.Sensitive = false;
				m_btnEditTag.Sensitive = false;
				m_prbUpload.Visible = true;
				m_prbUpload.Fraction = 0.0;
				WhitelistWorkerData argument = new WhitelistWorkerData
				{
					CurrentDevice = currentDevice,
					Filename = openFileDialog.FileName,
					Worker = m_bgwUpload
				};
				m_bgwUpload.RunWorkerAsync(argument);
			}
		}
	}

	private void OnUploadDoWork(object sender, DoWorkEventArgs e)
	{
		if (e.Argument is WhitelistWorkerData whitelistWorkerData)
		{
			whitelistWorkerData.CurrentDevice.Whitelist.LoadCSV(whitelistWorkerData.Filename, whitelistWorkerData.Worker);
		}
	}

	private void OnUploadCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		UpdateButtons();
		m_prbUpload.Visible = false;
		FillWhitelist(forceRefresh: true);
	}

	private void OnUploadProgressChanged(object sender, ProgressChangedEventArgs e)
	{
		if (e.ProgressPercentage > 100)
		{
			m_prbUpload.Fraction = 1.0;
		}
		else
		{
			m_prbUpload.Fraction = (double)e.ProgressPercentage / 100.0;
		}
	}

	private void OnAutoAddTagModeClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null && currentDevice.Whitelist.StartAutoAddMode())
		{
			m_prbUpload.Fraction = 1.0;
			m_prbUpload.Visible = true;
			m_btnRefresh.Sensitive = false;
			m_btnClearList.Sensitive = false;
			m_btnSaveList.Sensitive = false;
			m_btnLoadList.Sensitive = false;
			m_btnAddTag.Sensitive = false;
			m_btnRemoveTag.Sensitive = false;
			m_btnEditTag.Sensitive = false;
			m_btnStartTagAddMode.Sensitive = false;
			m_nAutoAddModeCountDown = 10;
		}
	}
}
