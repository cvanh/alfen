using System;
using System.Collections.Generic;
using System.Windows;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelInterface : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelInterface>();

	protected Button m_btnUploadLogo;

	protected UIPropertyColorHolder m_pchColorHolder;

	protected UIPropertySelect m_cmbDisplayItems;

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catIntensity;

	protected UIConfigCategory m_catDisplay;

	protected UIConfigCategory m_catLED;

	protected UIConfigCategory m_catSocket;

	protected UIPropertyString m_txtCurrency;

	protected UIPropertyNumber m_numPriceStart;

	protected UICheckbox m_chkPriceStartEnabled;

	protected UIPropertyNumber m_numPriceKwh;

	protected UICheckbox m_chkPriceKwhEnabled;

	protected UIPropertyNumber m_numPriceMinute;

	protected UICheckbox m_chkPriceMinuteEnabled;

	protected UIPropertyNumber m_numPriceOther;

	protected UICheckbox m_chkPriceOtherEnabled;

	protected UIPropertyString m_txtPriceOther;

	protected UICheckbox m_chkDisclaimer;

	protected UICheckbox m_chkAdhocOnlyDisclaimer;

	protected UIPropertyCheckbox m_chkLEDEnabled;

	protected UIPropertySelect m_cmbLEDEnabled;

	protected UIPropertyCheckbox m_chkHeartBeatEnabled;

	protected UIPropertyNumber m_numHeartBeatIntensity;

	private const float MAX_TARIFF = 999.99f;

	private const float MIN_TARIFF = -999.99f;

	public PanelInterface(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Interface";
		Tooltip = "Interface";
		IconName = "edit_image.png";
	}

	protected void AddLED(ICUDevice dev, ushort id, string name)
	{
		if (m_pchColorHolder != null && dev.GetProperty(id, 0) != null)
		{
			UIPropertyColor uiProp = AddColor(id, 0, name);
			m_pchColorHolder.Add(uiProp);
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
		m_catLED = null;
		newDevice.UpdateCategories("leds", "display");
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			m_catIntensity = m_configPanel.AddCategory("Intensity");
			if (newDevice.GetProperty(8289, 1).MaxLength >= 7)
			{
				m_catIntensity.Add(AddSmallHeader("Auto dim options"));
				m_catIntensity.Add(AddCheckBox(8289, 1, 1, "Time"));
				m_catIntensity.Add(AddCheckBox(8289, 1, 2, "Inactivity"));
				m_catIntensity.Add(AddCheckBox(8289, 1, 4, "QR codes"));
				m_catIntensity.Add(AddSmallHeader("Intensity"));
			}
			else
			{
				m_catIntensity.Add(AddCheckBox(8289, 1, 0, "Auto dim"));
			}
			m_catIntensity.Add(AddCustomNumber(8289, 2, 0, "Led/Display light intensity (%)"));
			if (iCULanDevice.HasDisplay)
			{
				m_catDisplay = m_configPanel.AddCategory("Display");
				m_catDisplay.Add(AddCustomSelect(8285, 0, "Display Language", DisplayLanguageHelper.GetDisplayLanguages(iCULanDevice.isAHP, iCULanDevice.isDC, iCULanDevice.FirmwareVersionNumber, iCULanDevice.GetPropertyString(8285, 0, 0))));
				if (newDevice.GetProperty(12897, 0).DeviceValue != null)
				{
					m_cmbDisplayItems = (UIPropertySelect)m_catDisplay.Add(AddSelect(12897, 0, 0, 0));
				}
				bool flag = iCULanDevice.HasProperty(12898, 6);
				TariffDisplayOptionsType tariffDisplayOptions = iCULanDevice.GetTariffDisplayOptions();
				if (iCULanDevice.HasProperty(12898, 1))
				{
					m_txtCurrency = (UIPropertyString)m_catDisplay.Add(AddText(12898, 1, "Currency (ISO 4217)"));
					m_txtCurrency.ForceReadonly(iCULanDevice.IsEichrechtEnabled);
					if (flag)
					{
						m_chkPriceStartEnabled = (UICheckbox)m_catDisplay.Add(AddPlainBool("Start tariff enabled", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.perSession)));
					}
					m_numPriceStart = (UIPropertyNumber)m_catDisplay.Add(AddCustomNumber(12898, 2, 2, "Start tariff"));
					m_numPriceStart.SetValueMinMax(-999.989990234375, 999.989990234375);
					if (flag)
					{
						m_chkPriceKwhEnabled = (UICheckbox)m_catDisplay.Add(AddPlainBool("Price per kWh enabled", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.perKwh)));
					}
					m_numPriceKwh = (UIPropertyNumber)m_catDisplay.Add(AddCustomNumber(12898, 3, 2, "Price per kWh"));
					m_numPriceKwh.SetValueMinMax(-999.989990234375, 999.989990234375);
					if (iCULanDevice.HasProperty(12898, 4))
					{
						if (flag)
						{
							m_chkPriceMinuteEnabled = (UICheckbox)m_catDisplay.Add(AddPlainBool("Price per minute enabled", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.perMinute)));
							m_chkPriceMinuteEnabled.ForceReadonly(iCULanDevice.IsEichrechtEnabled);
						}
						m_numPriceMinute = (UIPropertyNumber)m_catDisplay.Add(AddCustomNumber(12898, 4, 2, "Price per minute"));
						m_numPriceMinute.SetValueMinMax(-999.989990234375, 999.989990234375);
						m_numPriceMinute.ForceReadonly(iCULanDevice.IsEichrechtEnabled);
					}
					if (flag)
					{
						m_chkPriceOtherEnabled = (UICheckbox)m_catDisplay.Add(AddPlainBool("Other tariff enabled", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.perOther)));
						m_chkPriceOtherEnabled.ForceReadonly(iCULanDevice.IsEichrechtEnabled);
						m_txtPriceOther = (UIPropertyString)m_catDisplay.Add((UIPropertyString)AddText(12898, 7, "Other tariff name"));
						m_numPriceOther = (UIPropertyNumber)m_catDisplay.Add(AddCustomNumber(12898, 6, 2, "Other tariff price"));
						m_numPriceOther.SetValueMinMax(-999.989990234375, 999.989990234375);
						m_numPriceOther.ForceReadonly(iCULanDevice.IsEichrechtEnabled);
					}
					m_chkDisclaimer = (UICheckbox)m_catDisplay.Add(AddPlainBool("Show 'additional costs' disclaimer", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.disclaimer)));
					if (flag && iCULanDevice.GetProperty(12898, 5).MaxLength > 32)
					{
						m_chkAdhocOnlyDisclaimer = (UICheckbox)m_catDisplay.Add(AddPlainBool("Show 'adhoc only' disclaimer", tariffDisplayOptions.HasFlag(TariffDisplayOptionsType.adhocOnlyDisclaimer)));
					}
					else
					{
						m_chkAdhocOnlyDisclaimer = null;
					}
				}
				m_catDisplay.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
				m_catDisplay.AddWidget(m_btnUploadLogo = AddCustomButton("Upload Image...", "Upload a new Image/Logo to the device", OnUploadLogoClicked, true));
				m_btnUploadLogo.Sensitive = iCULanDevice.IsFeatureUnlocked(IWSFirmwareFeatures.Features.PersonalizedDisplay);
			}
			if (!iCULanDevice.HasDisplay)
			{
				m_catLED = m_configPanel.AddCategory("LED");
				m_catLED.Add(m_pchColorHolder = AddColorHolder("LED colors", "FEATURE_COLORS"));
				AddLED(newDevice, 8960, "Unknown");
				AddLED(newDevice, 8961, "Off");
				AddLED(newDevice, 8962, "Booting");
				AddLED(newDevice, 8963, "Check Mains");
				AddLED(newDevice, 8964, "Available");
				AddLED(newDevice, 8965, "Authorizing");
				AddLED(newDevice, 8966, "Authorized");
				AddLED(newDevice, 8967, "Cable Connected");
				AddLED(newDevice, 8968, "EV Connected");
				AddLED(newDevice, 8969, "Preparing Charging");
				AddLED(newDevice, 8976, "Wait Vehicle Charging");
				AddLED(newDevice, 8977, "Charging Normal");
				AddLED(newDevice, 8978, "Charging Simplified");
				AddLED(newDevice, 8979, "Suspended Over-Current");
				AddLED(newDevice, 8980, "Suspended HF Switching");
				AddLED(newDevice, 8981, "Suspended EV Disconnected");
				AddLED(newDevice, 8982, "Finish Wait Vehicle");
				AddLED(newDevice, 8983, "Finish Wait Disconnect");
				AddLED(newDevice, 8984, "Error Protective Earth");
				AddLED(newDevice, 8985, "Power-line Fault");
				AddLED(newDevice, 8992, "Contactor Fault");
				AddLED(newDevice, 8993, "Error Charging");
				AddLED(newDevice, 8994, "Power Failure");
				AddLED(newDevice, 8995, "Error Temperature");
				AddLED(newDevice, 8996, "Illegal CP Value");
				AddLED(newDevice, 8997, "Illegal PP Value");
				AddLED(newDevice, 8998, "Too Many Restarts");
				AddLED(newDevice, 8999, "Error");
				AddLED(newDevice, 9000, "Error Message");
				AddLED(newDevice, 9001, "Not Authorised");
				AddLED(newDevice, 9008, "Cable Not Supported");
				AddLED(newDevice, 9009, "S2 Not Opened");
				AddLED(newDevice, 9010, "Time-out");
				AddLED(newDevice, 9011, "Reserved");
				AddLED(newDevice, 9012, "In Operative");
				AddLED(newDevice, 9013, "Load Balancing Limited");
				AddLED(newDevice, 9014, "Load Balancing Forced Off");
			}
			if (iCULanDevice.HasProperty(8592))
			{
				if (m_catLED == null)
				{
					m_catLED = m_configPanel.AddCategory("LED");
				}
				m_catLED.Add(AddSmallHeader(iCULanDevice.HasDisplay ? "RFID LED" : "RFID & UI LED"));
				ICUProperty property = iCULanDevice.GetProperty(8592, 0);
				if (iCULanDevice.HasDisplay || property.MaxLength == 1)
				{
					m_catLED.Add(AddCheckBox(8592, 0, 0, "LED enabled"));
				}
				else
				{
					m_cmbLEDEnabled = (UIPropertySelect)m_catLED.Add(AddSelect(8592, 0, 0, 0));
				}
			}
			if (iCULanDevice.HasProperty(8561))
			{
				if (m_catLED == null)
				{
					m_catLED = m_configPanel.AddCategory("LED");
				}
				m_chkHeartBeatEnabled = (UIPropertyCheckbox)m_catLED.Add(AddCheckBox(8561, 0, 0, "Heart beat mode enabled"));
				m_numHeartBeatIntensity = (UIPropertyNumber)m_catLED.Add(AddCustomNumber(8562, 0, 0, "Heart beat intensity"));
			}
			m_catSocket = m_configPanel.AddCategory("Socket");
			m_catSocket.Add(AddCheckBox(8579, 0, 0, "Cover lock enabled"));
		}
		return true;
	}

	private void SetCheckboxRelatedField(UICheckbox checkBox, UIPropertyBase relatedProperty)
	{
		if (checkBox == null || relatedProperty == null || relatedProperty.IsEnabled() == checkBox.IsChecked)
		{
			return;
		}
		relatedProperty.SetEnable(checkBox.IsChecked);
		if (!checkBox.IsChecked)
		{
			relatedProperty.ClearValue();
			if (relatedProperty is UIPropertyNumber)
			{
				relatedProperty.SetValue(0);
			}
			if (relatedProperty is UIPropertyString)
			{
				relatedProperty.SetValue("");
			}
		}
	}

	public override bool OnUpdateControls(string pageID = "")
	{
		if (!string.IsNullOrEmpty(pageID) && string.Compare(pageID, PageID) != 0)
		{
			return true;
		}
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			if (currentDevice.HasProperty(8592))
			{
				bool enable = currentDevice.GetPropertyInt(8592, 0) != 0;
				m_chkHeartBeatEnabled?.SetEnable(enable);
				m_numHeartBeatIntensity?.SetEnable(enable);
			}
			SetCheckboxRelatedField(m_chkPriceStartEnabled, m_numPriceStart);
			SetCheckboxRelatedField(m_chkPriceKwhEnabled, m_numPriceKwh);
			SetCheckboxRelatedField(m_chkPriceMinuteEnabled, m_numPriceMinute);
			SetCheckboxRelatedField(m_chkPriceOtherEnabled, m_numPriceOther);
			SetCheckboxRelatedField(m_chkPriceOtherEnabled, m_txtPriceOther);
		}
		return true;
	}

	private void OnUploadLogoClicked(object sender, EventArgs e)
	{
		//IL_0084: Unknown result type (might be due to invalid IL or missing references)
		if (m_parent == null)
		{
			return;
		}
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		if (currentDevice.UpdateProperties(2204160u))
		{
			ICUProperty property = currentDevice.GetProperty(2204160u);
			if (property != null && currentDevice.FirmwareVersionNumber >= new Version("3.4.0") && (Convert.ToUInt32(property.Value) & 0x1000) == 0)
			{
				MessageBox.Show($"The personalized display feature is not installed on this Charging Station ({m_currentDevice.Identification}).\nPlease install the correct license!", "ICU Installer Configuration", (MessageBoxButton)0, (MessageBoxImage)64);
				return;
			}
		}
		m_parent.ShowUploadLogoDialog();
	}

	public override void SaveChanges()
	{
		base.SaveChanges();
		m_pchColorHolder.RefreshDisplay();
	}

	public override void OnRevertChanges()
	{
		m_chkPriceMinuteEnabled?.SetValue(m_chkPriceMinuteEnabled.CustomValue);
		m_chkPriceStartEnabled?.SetValue(m_chkPriceStartEnabled.CustomValue);
		m_chkPriceKwhEnabled?.SetValue(m_chkPriceKwhEnabled.CustomValue);
		m_chkPriceOtherEnabled?.SetValue(m_chkPriceOtherEnabled.CustomValue);
		m_chkDisclaimer?.SetValue(m_chkDisclaimer.CustomValue);
		m_chkAdhocOnlyDisclaimer?.SetValue(m_chkAdhocOnlyDisclaimer.CustomValue);
		base.OnRevertChanges();
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		UICheckbox chkAdhocOnlyDisclaimer = m_chkAdhocOnlyDisclaimer;
		if (chkAdhocOnlyDisclaimer != null && chkAdhocOnlyDisclaimer.IsChecked)
		{
			UICheckbox chkDisclaimer = m_chkDisclaimer;
			if (chkDisclaimer != null && chkDisclaimer.IsChecked)
			{
				MessageDialog.ShowError("Only one disclaimer can be selected.");
				return false;
			}
		}
		if (m_currentDevice == null)
		{
			return true;
		}
		if (m_currentDevice.HasDisplay && m_currentDevice.HasProperty(12898, 6))
		{
			int currentValue = 0;
			currentValue = Flags.UpdateFlag(currentValue, 4, m_chkPriceMinuteEnabled.IsChecked);
			currentValue = Flags.UpdateFlag(currentValue, 8, m_chkPriceStartEnabled.IsChecked);
			currentValue = Flags.UpdateFlag(currentValue, 2, m_chkPriceKwhEnabled.IsChecked);
			currentValue = Flags.UpdateFlag(currentValue, 16, m_chkPriceOtherEnabled.IsChecked);
			currentValue = Flags.UpdateFlag(currentValue, 1, m_chkDisclaimer.IsChecked);
			if (m_chkAdhocOnlyDisclaimer != null)
			{
				currentValue = Flags.UpdateFlag(currentValue, 32, m_chkAdhocOnlyDisclaimer.IsChecked);
			}
			m_chkPriceMinuteEnabled.CustomValue = m_chkPriceMinuteEnabled.IsChecked;
			m_chkPriceStartEnabled.CustomValue = m_chkPriceStartEnabled.IsChecked;
			m_chkPriceKwhEnabled.CustomValue = m_chkPriceKwhEnabled.IsChecked;
			m_chkPriceOtherEnabled.CustomValue = m_chkPriceOtherEnabled.IsChecked;
			m_chkDisclaimer.CustomValue = m_chkDisclaimer.IsChecked;
			if (m_chkAdhocOnlyDisclaimer != null)
			{
				m_chkAdhocOnlyDisclaimer.CustomValue = m_chkAdhocOnlyDisclaimer.IsChecked;
			}
			m_currentDevice.SetTariffDisplayOptions((TariffDisplayOptionsType)currentValue);
			Version firmwareVersionNumber = m_currentDevice.FirmwareVersionNumber;
			if (firmwareVersionNumber >= new Version(6, 6) && firmwareVersionNumber < new Version(7, 0))
			{
				SetDisplayPanelPropertiesDirty();
			}
		}
		return true;
	}

	private IList<uint> GetActiveOdsOnDisplay()
	{
		return new List<uint> { 2120960u, 2121984u, 3301888u, 3301889u, 3301890u, 3301891u, 3301892u, 3301893u, 3301894u, 3301895u };
	}

	private void SetDisplayPanelPropertiesDirty()
	{
		bool flag = false;
		foreach (uint item in GetActiveOdsOnDisplay())
		{
			ICUProperty property = m_currentDevice.GetProperty(item);
			if (property != null && property.IsChanged)
			{
				flag = true;
				break;
			}
		}
		if (!flag)
		{
			return;
		}
		foreach (uint item2 in GetActiveOdsOnDisplay())
		{
			m_currentDevice.GetProperty(item2)?.SetDirty();
		}
	}
}
