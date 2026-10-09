using System;
using System.Collections.Generic;
using System.Net;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelLoadbalancing : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelLoadbalancing>();

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catStaticLB;

	protected UIConfigCategory m_catActiveLB;

	protected UIConfigCategory m_catTCPIPMaster;

	protected UIConfigCategory m_catP1Options;

	protected UIConfigCategory m_catTCPIPSlave;

	protected UIConfigCategory m_catRTUmodbus;

	protected UIConfigCategory m_catSCN;

	protected UIConfigCategory m_catUKSmartCharging;

	protected UIConfigCategory m_catSolarCharging;

	protected UIPropertyCheckbox m_chkStaticLBEnabled;

	protected UIPropertyCheckbox m_chkActiveLBEnabled;

	protected UIPropertyCheckbox m_chkSwitch13phases;

	protected UIPropertyNumber m_cnMaxSmartMeterCurrent;

	protected UIPropertyNumber m_cnActiveLBSafeCurrent;

	protected UIPropertyCheckbox m_chkTCPIPMasterEnabled;

	protected UIPropertySelect m_slSmartMeterType;

	protected UIPropertySelect m_slPhaseRotation;

	protected UIPropertyString m_stTCPIPMasterAddress;

	protected UIPropertyNumber m_cnTCPIPMasterSlaveAddress;

	protected UIPropertySelect m_slTCPIPMasterMode;

	protected UIPropertySelect m_slTCPIPMasterWordOrder;

	protected UIPropertySelect m_slTCPIPMasterConnection;

	protected UIPropertyString m_stRTUModbusAddress;

	protected UIPropertySelect m_slRTUModbusParity;

	protected UIPropertySelect m_slRTUModbusBaudrate;

	protected UIPropertySelect m_slRTUModbusWordOrder;

	protected UIPropertyNumber m_cnRTUModbusUpdateTime;

	protected UIPropertyNumber m_cnRTUModbusReadTimeOut;

	protected UIPropertySelect m_slRTUModbusFunction;

	protected UIPropertySelect m_slActiveLBDataSource;

	protected UISelect m_slTCPIPSlaveMode;

	protected UIPropertySelect m_slSmartMeterInclEV;

	protected UIPropertySelect m_slP1Interface;

	protected UIPropertyString m_stP1ServerAddress;

	protected UIPropertyNumber m_cnP1ServerPort;

	protected UIPropertyString m_stSCNName;

	protected UIPropertyLabel m_lblInfoStaticLoadBalancing;

	protected UIPropertyLabel m_lblInfoActiveLoadBalancing;

	protected UIPropertyLabel m_lblInfoTCPIPSlave;

	protected UIPropertyLabel m_lblInfoSCN;

	protected UIPropertyLabel m_lblUKSmartCharging;

	protected UICheckbox m_chkEnabledUKSmartCharging;

	protected UIPropertyNumber m_cnUKRandomDelay;

	protected UICheckbox m_chkOverrideSocket1;

	protected UICheckbox m_chkOverrideSocket2;

	private UIPropertyNumber m_numSCNTotalCurrent;

	private UIPropertyNumber m_numSCNSocketSafeCurrent;

	private UIPropertyNumber m_numSCNTotalSafeCurrent;

	private UIPropertyNumber m_numSCNAltPeriod;

	protected Button m_btnConfigureModbusTCPIP;

	protected Button m_btnTestModbusMeterTCPIP;

	protected Button m_btnConfigureModbusRTU;

	protected Button m_btnTestModbusMeterRTU;

	protected List<UIPropertyBase> m_lstActiveLBControls = new List<UIPropertyBase>();

	protected List<UIPropertyBase> m_lstActiveLBMeter = new List<UIPropertyBase>();

	protected EMeterTypes m_selectedSmartMeter = EMeterTypes.ENERGYMETER_NONE;

	protected ETCPIPSlaveOptions m_selectedDataSource;

	protected ETCPIPSlaveMode m_selectedSlaveMode;

	protected UIPropertySelect m_slSolarChargingMode;

	protected UIPropertyNumber m_cnSolarChargingComfortLevel;

	protected UIPropertyNumber m_cnSolarChargingGreenShare;

	protected UIPropertyCheckbox m_chkSolarChargingBoost1;

	protected UIPropertyCheckbox m_chkSolarChargingBoost2;

	protected UIPropertyLabel m_lblSolarChargingInfo;

	protected bool m_hasSCNNameChanged;

	protected bool m_showLegacyModBusTCPSettings;

	protected bool m_showTCPIPSlaveSettings;

	protected bool m_showAdvancedSmartmeterSettings;

	protected bool m_showSmartMeterInclEVSetting;

	protected Dictionary<string, string> m_dicPhaseRotation = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSlaveModes = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSmartMeters = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicP1Interface = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicOverrides = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSolarChargingModes = new Dictionary<string, string>();

	private string m_bopresetName;

	private string m_meterName;

	private const uint m_compliantRandomDelay = 600u;

	private bool m_fRefreshPage;

	public PanelLoadbalancing(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Load balancing";
		Tooltip = "Load balancing";
		IconName = "loadbalancing.png";
		m_dicPhaseRotation.Add("L1", "L1");
		m_dicPhaseRotation.Add("L2", "L2");
		m_dicPhaseRotation.Add("L3", "L3");
		m_dicPhaseRotation.Add("L1L2L3", "L1L2L3");
		m_dicPhaseRotation.Add("L1L3L2", "L1L3L2");
		m_dicPhaseRotation.Add("L2L1L3", "L2L1L3");
		m_dicPhaseRotation.Add("L2L3L1", "L2L3L1");
		m_dicPhaseRotation.Add("L3L1L2", "L3L1L2");
		m_dicPhaseRotation.Add("L3L2L1", "L3L2L1");
		m_dicSlaveModes.Add("1", "SCN");
		m_dicSlaveModes.Add("2", "Socket");
		m_dicP1Interface.Add("0", "Serial");
		m_dicP1Interface.Add("1", "Telnet");
		m_dicP1Interface.Add("2", "HomeWizard Wi-Fi P1");
		m_dicSmartMeters.Add("4", "Modbus TCP/IP");
		m_dicSmartMeters.Add("5", "DSMR4.x / SMR5.0 (P1)");
		m_dicSmartMeters.Add("6", "Modbus RTU");
		m_dicSmartMeters.Add("7", "TIC (Linky)");
		m_dicOverrides.Add("0", "Follow profile");
		m_dicOverrides.Add("1", "Override charging profile (direct start)");
		m_dicSolarChargingModes.Add(Convert.ToInt16(ESolarChargingModes.SOLAR_CHARGING_OFF).ToString(), "Off");
		m_dicSolarChargingModes.Add(Convert.ToInt16(ESolarChargingModes.SOLAR_CHARGING_COMFORT).ToString(), "Comfort");
		m_dicSolarChargingModes.Add(Convert.ToInt16(ESolarChargingModes.SOLAR_CHARGING_GREEN).ToString(), "Green");
		StartUpdateTimer(2000);
	}

	protected override void OnChangeProperty()
	{
		base.OnChangeProperty();
		if (m_configPanel != null)
		{
			m_configPanel.UpdateControls();
		}
	}

	protected override void OnUpdateTick()
	{
		if (m_currentDevice == null || !m_currentDevice.IsConnected || m_currentDevice.LastHttpStatusCode != HttpStatusCode.OK)
		{
			return;
		}
		UIPropertyString stP1ServerAddress = m_stP1ServerAddress;
		if (stP1ServerAddress == null || !stP1ServerAddress.IsChanged)
		{
			UIPropertyNumber cnP1ServerPort = m_cnP1ServerPort;
			if (cnP1ServerPort == null || !cnP1ServerPort.IsChanged)
			{
				m_currentDevice.UpdateProperties(2199810u, 2199811u);
			}
		}
	}

	public override bool OnUpdateControls(string pageID = "")
	{
		if (!string.IsNullOrEmpty(pageID) && string.Compare(pageID, PageID) != 0)
		{
			return true;
		}
		if (m_currentDevice != null)
		{
			string value = string.Empty;
			string value2 = string.Empty;
			string value3 = string.Empty;
			string value4 = string.Empty;
			bool activelbEnabled = false;
			bool flag = false;
			bool flag2 = false;
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice == null)
			{
				return false;
			}
			uint propertyUInt = currentDevice.GetPropertyUInt(8610, 0);
			Version firmwareVersionNumber = currentDevice.FirmwareVersionNumber;
			if (IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_SCN, currentDevice.isAHP))
			{
				if (currentDevice.HasSCNNetwork)
				{
					flag = false;
					value = "Since an SCN is active, the loadbalancing is now controlled by the SCN.";
				}
				else
				{
					flag = true;
				}
				activelbEnabled = true;
				flag2 = true;
			}
			else if (IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_Active, currentDevice.isAHP))
			{
				activelbEnabled = true;
				flag = true;
			}
			else if (IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_Static, currentDevice.isAHP))
			{
				activelbEnabled = false;
				flag = true;
			}
			else
			{
				activelbEnabled = false;
				flag = false;
			}
			if (!activelbEnabled)
			{
				if (string.IsNullOrEmpty(value2))
				{
					value2 = "You don't have a valid license for active loadbalancing.";
					if (m_lblInfoActiveLoadBalancing != null)
					{
						m_lblInfoActiveLoadBalancing.SetType(EUILabelType.LargeInfo);
					}
				}
				m_chkActiveLBEnabled?.SetValue(0);
				m_chkTCPIPMasterEnabled?.SetValue(0);
				m_slActiveLBDataSource?.SetValue(0);
			}
			if (!flag)
			{
				if (string.IsNullOrEmpty(value))
				{
					value = "You don't have a valid license for static loadbalancing.";
				}
				m_chkStaticLBEnabled?.SetValue(0);
			}
			if (!flag2 && string.IsNullOrEmpty(value3))
			{
				value3 = "You don't have a valid license for Smart Charging Network";
			}
			m_chkStaticLBEnabled?.SetEnable(flag);
			m_lstActiveLBControls.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(activelbEnabled);
			});
			m_lstActiveLBMeter.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(activelbEnabled);
			});
			bool flag3 = m_chkActiveLBEnabled != null && m_chkActiveLBEnabled.IsChecked;
			if (m_chkStaticLBEnabled != null)
			{
				if (flag3)
				{
					m_chkStaticLBEnabled.SetValue(1);
					if (string.IsNullOrEmpty(value))
					{
						value = "Static Load Balancing is enabled when Active Load Balancing is used.";
					}
				}
				m_chkStaticLBEnabled.SetEnable(!flag3 && !currentDevice.HasSCNNetwork);
				flag = m_chkStaticLBEnabled.IsChecked;
			}
			m_chkSwitch13phases?.SetEnable(fEnable: true);
			m_selectedSmartMeter = (EMeterTypes)Convert.ToInt16(m_slSmartMeterType?.GetValue());
			if (m_selectedSmartMeter < EMeterTypes.ENERGYMETER_TCPIP_SMART || m_selectedSmartMeter > EMeterTypes.ENERGYMETER_TIC)
			{
				m_selectedSmartMeter = EMeterTypes.ENERGYMETER_NONE;
			}
			if (m_chkTCPIPMasterEnabled != null)
			{
				bool isChecked = m_chkTCPIPMasterEnabled.IsChecked;
				m_chkTCPIPMasterEnabled.SetEnable(!flag3 & activelbEnabled);
				if (flag3)
				{
					m_chkTCPIPMasterEnabled.SetValue(0);
				}
				m_configPanel.ShowCategory(m_catTCPIPMaster, isChecked);
				m_chkActiveLBEnabled.SetEnable(!isChecked & activelbEnabled);
				if (isChecked)
				{
					m_chkActiveLBEnabled.SetValue(0);
				}
				m_cnActiveLBSafeCurrent.Hide = !(isChecked | flag3);
			}
			else if (m_slActiveLBDataSource != null)
			{
				m_selectedDataSource = (ETCPIPSlaveOptions)Convert.ToInt16(m_slActiveLBDataSource?.GetValue());
				m_selectedSlaveMode = (ETCPIPSlaveMode)Convert.ToInt16(m_slTCPIPSlaveMode.GetValue());
				bool flag4 = m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_ALL;
				m_configPanel.ShowCategory(m_catTCPIPSlave, flag3 & flag4);
				m_configPanel.ShowCategory(m_catTCPIPMaster, flag3 && !flag4 && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART);
				m_configPanel.ShowCategory(m_catP1Options, flag3 && !flag4 && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_P1);
				m_configPanel.ShowCategory(m_catRTUmodbus, flag3 && !flag4 && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_RTU_SMART);
				bool enable = Convert.ToInt16(m_slP1Interface?.GetValue()) > 0;
				m_stP1ServerAddress?.SetEnable(enable);
				m_cnP1ServerPort?.SetEnable(enable);
				m_slActiveLBDataSource.SetEnable(flag3);
				m_slSmartMeterType.Hide = !flag3 || flag4;
				m_slSmartMeterInclEV?.SetEnable(flag3 && !flag4);
				if (m_cnMaxSmartMeterCurrent != null)
				{
					m_cnMaxSmartMeterCurrent.Hide = !flag3 || flag4 || m_selectedSmartMeter == EMeterTypes.ENERGYMETER_NONE;
				}
				m_cnActiveLBSafeCurrent.Hide = !flag3 || !((m_selectedSmartMeter != EMeterTypes.ENERGYMETER_NONE) | flag4);
				if (flag3 & flag4)
				{
					m_selectedSmartMeter = EMeterTypes.ENERGYMETER_NONE;
					m_slSmartMeterType.SetValue(-1);
					if (!flag2)
					{
						value4 = "You don't have a valid license for Smart Charging Network, so EMS Mode is set to Socket";
						if (m_selectedSlaveMode != ETCPIPSlaveMode.BALANCEMODE_SOCKET)
						{
							m_selectedSlaveMode = ETCPIPSlaveMode.BALANCEMODE_SOCKET;
							UISelect slTCPIPSlaveMode = m_slTCPIPSlaveMode;
							int selectedSlaveMode = (int)m_selectedSlaveMode;
							slTCPIPSlaveMode.CustomValue = selectedSlaveMode.ToString();
						}
						m_slTCPIPSlaveMode.SetEnable(fEnable: false);
					}
					else if (m_selectedSlaveMode == ETCPIPSlaveMode.BALANCEMODE_NONE)
					{
						m_selectedSlaveMode = ETCPIPSlaveMode.BALANCEMODE_SOCKET;
						UISelect slTCPIPSlaveMode2 = m_slTCPIPSlaveMode;
						int selectedSlaveMode = (int)m_selectedSlaveMode;
						slTCPIPSlaveMode2.CustomValue = selectedSlaveMode.ToString();
					}
					if (m_selectedSlaveMode != ETCPIPSlaveMode.BALANCEMODE_SOCKET && m_selectedSlaveMode != ETCPIPSlaveMode.BALANCEMODE_SCN && string.IsNullOrEmpty(value4))
					{
						value4 = "Warning! Please Select a valid EMS Mode!";
						m_lblInfoTCPIPSlave?.SetType(EUILabelType.LargeWarning);
					}
				}
			}
			else
			{
				m_configPanel.ShowCategory(m_catTCPIPMaster, flag3 && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART);
				m_cnActiveLBSafeCurrent.Hide = !flag3 || m_selectedSmartMeter == EMeterTypes.ENERGYMETER_NONE;
				if (m_cnMaxSmartMeterCurrent != null)
				{
					m_cnMaxSmartMeterCurrent.Hide = !flag3 || m_selectedSmartMeter == EMeterTypes.ENERGYMETER_NONE;
				}
			}
			if (m_btnConfigureModbusTCPIP != null)
			{
				bool flag5 = currentDevice.isAHP && !currentDevice.HasProperty(9507, 2);
				bool? flag6 = m_slTCPIPMasterMode?.GetSelectedText().Equals("Custom register mapping");
				UIPropertySelect slTCPIPMasterMode = m_slTCPIPMasterMode;
				bool? flag7 = ((slTCPIPMasterMode != null) ? new bool?(!slTCPIPMasterMode.IsChanged) : ((bool?)null));
				m_btnConfigureModbusTCPIP.Sensitive = flag5 || (flag6 == true && flag7 == true);
			}
			m_configPanel.ShowCategory(m_catSCN, flag2);
			ICUChargingProfiles chargingProfiles = currentDevice.ChargingProfiles;
			if (chargingProfiles != null && chargingProfiles.IsChargingProfileSupported)
			{
				m_lblUKSmartCharging?.SetValue(currentDevice.ChargingProfiles.IsUKSmartChargingProfileInstalled ? "The UK Smart charging profile is currently active." : "");
			}
			int propertyInt = m_currentDevice.GetPropertyInt(8290, 0);
			int propertyInt2 = m_currentDevice.GetPropertyInt(8295, 0);
			int propertyInt3 = m_currentDevice.GetPropertyInt(8296, 0);
			int propertyInt4 = currentDevice.GetPropertyInt(9520, 1);
			if (m_cnActiveLBSafeCurrent.IsEnabled() && propertyInt3 > Math.Min(propertyInt, propertyInt2))
			{
				value2 = "Warning! Active LoadBalancing Safe current cannot be higher then the";
				string text = value2;
				string text2;
				if (propertyInt < propertyInt2)
				{
					text2 = $"maximum station current {propertyInt} A.";
				}
				else
				{
					text2 = (m_showAdvancedSmartmeterSettings ? $"maximum smart meter current {propertyInt2} A." : $"maximum installation current {propertyInt2} A.");
				}
				value2 = text + text2;
				m_lblInfoActiveLoadBalancing?.SetType(EUILabelType.Warning);
			}
			else if (flag3 && m_slSmartMeterType != null && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_NONE && propertyInt4 != DataSheet.SelectParameterOption(9520, 1, "Energy Management System"))
			{
				value2 = "Warning! Active LoadBalacing is enabled, but no Protocol is selected!";
				m_lblInfoActiveLoadBalancing?.SetType(EUILabelType.LargeWarning);
			}
			if (m_currentDevice.GetPropertyInt(8632, 0, 1) == 1 && m_currentDevice.GetPropertyInt(8632, 0, 2) == 0)
			{
				value2 = "Warning! The Charging Station cannot communicate to the external board, but the protocol can still be selected.";
				m_lblInfoActiveLoadBalancing?.SetType(EUILabelType.LargeWarning);
			}
			if (m_lblInfoTCPIPSlave != null)
			{
				m_lblInfoTCPIPSlave.Hide = string.IsNullOrEmpty(value4);
				m_lblInfoTCPIPSlave.SetValue(value4);
			}
			if (m_lblInfoStaticLoadBalancing != null)
			{
				m_lblInfoStaticLoadBalancing.Hide = string.IsNullOrEmpty(value);
				m_lblInfoStaticLoadBalancing.SetValue(value);
			}
			if (m_lblInfoActiveLoadBalancing != null)
			{
				m_lblInfoActiveLoadBalancing.Hide = string.IsNullOrEmpty(value2);
				m_lblInfoActiveLoadBalancing.SetValue(value2);
			}
			if (m_lblInfoSCN != null)
			{
				m_lblInfoSCN.Hide = string.IsNullOrEmpty(value3);
				m_lblInfoSCN.SetValue(value3);
			}
			if (m_slSolarChargingMode != null)
			{
				bool flag8 = true;
				bool flag9 = false;
				if (!activelbEnabled)
				{
					flag9 = true;
					m_lblSolarChargingInfo?.SetValue("Solar charging is only allowed when the active loadbalancing feature is enabled.");
				}
				else if (currentDevice.HasSCNNetwork)
				{
					flag9 = true;
					m_lblSolarChargingInfo?.SetValue("Solar charging is not allowed for Smart Charging Networks (SCN).");
				}
				else if (currentDevice.NumberOfSockets > 1)
				{
					flag9 = true;
					m_lblSolarChargingInfo?.SetValue("Solar charging is not allowed for double socket charging stations.");
				}
				if (flag9)
				{
					m_slSolarChargingMode.Hide = true;
					m_lblSolarChargingInfo.Hide = false;
					m_cnSolarChargingComfortLevel.Hide = true;
					m_cnSolarChargingGreenShare.Hide = true;
				}
				else
				{
					m_slSolarChargingMode.Hide = false;
					m_lblSolarChargingInfo.Hide = true;
					switch ((ESolarChargingModes)Convert.ToInt16(m_slSolarChargingMode?.GetValue()))
					{
					case ESolarChargingModes.SOLAR_CHARGING_COMFORT:
						m_cnSolarChargingComfortLevel.Hide = false;
						m_cnSolarChargingGreenShare.Hide = true;
						flag8 = false;
						break;
					case ESolarChargingModes.SOLAR_CHARGING_GREEN:
						m_cnSolarChargingComfortLevel.Hide = true;
						m_cnSolarChargingGreenShare.Hide = false;
						flag8 = false;
						break;
					default:
						m_cnSolarChargingComfortLevel.Hide = true;
						m_cnSolarChargingGreenShare.Hide = true;
						break;
					}
				}
				m_chkSolarChargingBoost1.Hide = flag8;
				m_chkSolarChargingBoost2.Hide = flag8 || currentDevice.NumberOfSockets < 2;
			}
		}
		else
		{
			m_chkStaticLBEnabled?.SetEnable(fEnable: false);
			m_chkActiveLBEnabled?.SetEnable(fEnable: false);
			m_chkSwitch13phases?.SetEnable(fEnable: false);
			if (m_catTCPIPMaster != null)
			{
				m_configPanel.ShowCategory(m_catTCPIPMaster, show: false);
			}
			if (m_catTCPIPSlave != null)
			{
				m_configPanel.ShowCategory(m_catTCPIPSlave, show: false);
			}
			if (m_catSCN != null)
			{
				m_configPanel.ShowCategory(m_catSCN, show: false);
			}
		}
		m_configPanel.RefreshCurrentCategory();
		return true;
	}

	public void OnP1InterfaceChanged(object sender, EventArgs e)
	{
		UIPropertySelect slP1Interface = m_slP1Interface;
		if (slP1Interface != null && slP1Interface.IsChanged && Convert.ToInt16(m_slP1Interface?.GetValue()) == 2)
		{
			new DlgInfo("When using HomeWizard Wi-Fi P1, the charging station will try to automatically discover the module.\nAfter saving the configuration, please wait a minute for the IP and port to be updated automatically.\nIf this does not happen, then please fill them in manually.", showInTaskbar: true).Run();
		}
	}

	private void OnScnNameChanged(object sender, EventArgs e)
	{
		if (m_currentDevice == null || m_stSCNName == null)
		{
			return;
		}
		ICUProperty property = m_currentDevice.GetProperty(8576, 1);
		if (m_stSCNName.GetValue().ToString() == string.Empty && property.IsChanged)
		{
			if (!MessageDialog.Confirm("You are removing this Charging Station from the SCN, it will be rebooted after saving the changes.", Command.Ok))
			{
				property.Rollback();
			}
		}
		else if (property != null && property.IsChanged)
		{
			MessageDialog.Confirm("You are not allowed to change the name of SCN, you can only clear it!\nSelect the complete name and press the delete button", Command.Ok);
			property.Rollback();
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
		newDevice.UpdateCategories("MbusTCP", "meter4", "scn");
		newDevice.UpdateProperties(2127360u);
		if (iCULanDevice.HasProperty(8310))
		{
			string[] array = iCULanDevice.GetPropertyString(8310, 0, 0).Split(new char[1] { ',' });
			m_bopresetName = ((array.Length != 0) ? array[0] : null);
			m_meterName = ((array.Length > 1) ? array[1] : null);
		}
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			m_showLegacyModBusTCPSettings = !iCULanDevice.HasProperty(21015);
			m_showTCPIPSlaveSettings = iCULanDevice.HasProperty(9520, 1);
			m_showAdvancedSmartmeterSettings = iCULanDevice.FirmwareVersionNumber >= new Version("4.1.0") || iCULanDevice.isAHP;
			m_showSmartMeterInclEVSetting = iCULanDevice.HasProperty(8303);
			if (newDevice.NumberOfSockets > 1)
			{
				m_catStaticLB = m_configPanel.AddCategory("Static balancing", "Static load balancing");
				if (iCULanDevice.isAHP)
				{
					m_chkStaticLBEnabled = (UIPropertyCheckbox)m_catStaticLB.Add(AddCheckBox(36864, 0, 0, "Static Load Balancing", null, 2122753u));
				}
				else
				{
					m_chkStaticLBEnabled = (UIPropertyCheckbox)m_catStaticLB.Add(AddCheckBox(8292, 0, 1, "Static Load Balancing", null, 2122753u));
				}
				m_lblInfoStaticLoadBalancing = (UIPropertyLabel)m_catStaticLB.Add(AddLabelText("", EUILabelType.LargeInfo));
			}
			m_catActiveLB = m_configPanel.AddCategory("Active balancing", "Active load balancing");
			m_lstActiveLBControls.Clear();
			m_lstActiveLBMeter.Clear();
			m_btnConfigureModbusTCPIP = new Button("Custom register mapping")
			{
				MinWidth = 150.0,
				MinHeight = AppProperties.ButtonHeight,
				TooltipText = "Configure TCP/IP modbus register mapping"
			};
			m_btnConfigureModbusTCPIP.Clicked += OnConfigureModbusClicked;
			m_btnTestModbusMeterTCPIP = new Button("Test smart meter")
			{
				MinWidth = AppProperties.ButtonWidth,
				MinHeight = AppProperties.ButtonHeight,
				TooltipText = "Show modbus TCP/IP values"
			};
			m_btnTestModbusMeterTCPIP.Clicked += OnButtonTestSmartMeterClicked;
			m_btnConfigureModbusRTU = new Button("Custom register mapping")
			{
				MinWidth = 150.0,
				MinHeight = AppProperties.ButtonHeight,
				TooltipText = "Configure RTU modbus register mapping"
			};
			m_btnConfigureModbusRTU.Clicked += OnConfigureModbusClicked;
			m_btnTestModbusMeterRTU = new Button("Test smart meter")
			{
				MinWidth = AppProperties.ButtonWidth,
				MinHeight = AppProperties.ButtonHeight,
				TooltipText = "Show modbus RTU values"
			};
			m_btnTestModbusMeterRTU.Clicked += OnButtonTestSmartMeterClicked;
			m_chkActiveLBEnabled = (UIPropertyCheckbox)m_catActiveLB.Add(AddCheckBox(8292, 0, 2, m_showAdvancedSmartmeterSettings ? "Active Load Balancing" : "P1 Load Balancing", null, 2122754u));
			m_lstActiveLBControls.Add(m_chkActiveLBEnabled);
			if (m_showTCPIPSlaveSettings)
			{
				m_slActiveLBDataSource = (UIPropertySelect)m_catActiveLB.Add(AddSelect(9520, 1, 0, 0, "Data Source"));
				m_lstActiveLBControls.Add(m_slActiveLBDataSource);
			}
			else
			{
				m_selectedDataSource = ETCPIPSlaveOptions.TCPIPSLAVE_NONE;
			}
			if (m_showSmartMeterInclEVSetting)
			{
				m_slSmartMeterInclEV = (UIPropertySelect)m_catActiveLB.Add(AddSelect(8303, 0, 0, 0, "Received Measurements"));
				m_lstActiveLBMeter.Add(m_slSmartMeterInclEV);
			}
			if (m_showLegacyModBusTCPSettings)
			{
				m_chkTCPIPMasterEnabled = (UIPropertyCheckbox)m_catActiveLB.Add(AddCheckBox(9506, 1, 1, "ModbusTCP balancing enabled"));
				m_lstActiveLBControls.Add(m_chkTCPIPMasterEnabled);
			}
			else
			{
				m_slSmartMeterType = (UIPropertySelect)m_catActiveLB.Add(AddCustomSelect(21015, 0, "Protocol Selection", m_dicSmartMeters));
				m_lstActiveLBMeter.Add(m_slSmartMeterType);
				Dictionary<string, string> dictionary = new Dictionary<string, string>(m_dicSmartMeters);
				if (iCULanDevice.HasProperty(8632))
				{
					if (iCULanDevice.GetPropertyInt(8632, 0, 1) == 0)
					{
						dictionary.Remove("6");
						dictionary.Remove("7");
					}
				}
				else
				{
					dictionary.Remove("6");
					dictionary.Remove("7");
				}
				m_slSmartMeterType.SetCustomList(dictionary);
			}
			if (m_showAdvancedSmartmeterSettings)
			{
				m_cnMaxSmartMeterCurrent = (UIPropertyNumber)m_catActiveLB.Add(AddCustomNumber(8295, 0, 0, "Maximum smart meter current (A)"));
				m_cnMaxSmartMeterCurrent.SetValueMinMax(0.0, 3.4028234663852886E+38);
				m_lstActiveLBMeter.Add(m_cnMaxSmartMeterCurrent);
			}
			m_cnActiveLBSafeCurrent = (UIPropertyNumber)m_catActiveLB.Add(AddCustomNumber(8296, 0, 0, "Safe current (A)"));
			m_cnActiveLBSafeCurrent.SetValueMinMax(0.0, 3.4028234663852886E+38);
			m_lstActiveLBControls.Add(m_cnActiveLBSafeCurrent);
			if (!iCULanDevice.isEcogDC)
			{
				m_slPhaseRotation = (UIPropertySelect)m_catActiveLB.Add(AddCustomSelect(8297, 0, "Phase rotation", m_dicPhaseRotation), advancedProperty: true);
				m_lstActiveLBMeter.Add(m_slPhaseRotation);
				if (iCULanDevice.HasProperty(8581))
				{
					m_chkSwitch13phases = (UIPropertyCheckbox)m_catActiveLB.Add(AddCheckBox(8581, 0, 0, "Allow single-/multiphase charging"));
					m_lstActiveLBMeter.Add(m_chkSwitch13phases);
				}
			}
			m_lblInfoActiveLoadBalancing = (UIPropertyLabel)m_catActiveLB.Add(AddLabelText("\n", EUILabelType.Warning));
			m_catTCPIPMaster = m_configPanel.AddCategory("TCP/IP Meter", "Modbus TCP/IP Meter");
			if (m_showLegacyModBusTCPSettings)
			{
				m_slTCPIPMasterMode = (UIPropertySelect)m_catTCPIPMaster.Add(AddSelect(9506, 2, 0, 0, "ModbusTCP mode"));
				m_slTCPIPMasterMode.SetValue(DataSheet.SelectParameterOption(9506, 2, "Socomec"));
				m_slTCPIPMasterMode.ForceReadonly(fForce: true);
				if (((ICULanDevice)newDevice).HasProperty(9506, 3))
				{
					m_slTCPIPMasterConnection = (UIPropertySelect)m_catTCPIPMaster.Add(AddSelect(9506, 3, 0, 0, "ModbusTCP connection type"));
				}
				m_stTCPIPMasterAddress = (UIPropertyString)m_catTCPIPMaster.Add(AddText(9506, 4, "ModbusTCP IP address"));
				m_cnTCPIPMasterSlaveAddress = (UIPropertyNumber)m_catTCPIPMaster.Add(AddCustomNumber(9506, 6, 0, "ModbusTCP slave address"));
			}
			else
			{
				m_chkTCPIPMasterEnabled = null;
				m_slTCPIPMasterMode = null;
				m_slTCPIPMasterConnection = null;
				m_stTCPIPMasterAddress = (UIPropertyString)m_catTCPIPMaster.Add(AddText(9507, 4, "IP address"));
				m_cnTCPIPMasterSlaveAddress = (UIPropertyNumber)m_catTCPIPMaster.Add(AddCustomNumber(9507, 6, 0, "Slave address"));
			}
			if (m_showAdvancedSmartmeterSettings)
			{
				if (((ICULanDevice)newDevice).HasProperty(9507, 2))
				{
					m_slTCPIPMasterMode = (UIPropertySelect)m_catTCPIPMaster.Add(AddSelect(9507, 2, 0, 0, "Mode"));
				}
				if (((ICULanDevice)newDevice).HasProperty(9588))
				{
					m_slTCPIPMasterWordOrder = (UIPropertySelect)m_catTCPIPMaster.Add(AddSelect(9588, 0, 0, 0, "Word Order"));
				}
				m_catTCPIPMaster.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
				AddCustomButton(m_btnConfigureModbusTCPIP);
				m_catTCPIPMaster.AddWidget(m_btnConfigureModbusTCPIP);
				AddCustomButton(m_btnTestModbusMeterTCPIP);
				m_catTCPIPMaster.AddWidget(m_btnTestModbusMeterTCPIP);
			}
			m_configPanel.ShowCategory(m_catTCPIPMaster, show: false);
			if (((ICULanDevice)newDevice).HasProperty(9520, 3))
			{
				m_catTCPIPSlave = m_configPanel.AddCategory("TCP/IP EMS", "Modbus TCP/IP EMS");
				m_slTCPIPSlaveMode = (UISelect)m_catTCPIPSlave.Add(AddPlainSelect("Mode", m_dicSlaveModes, "", null, 2437123u));
				m_selectedSlaveMode = (ETCPIPSlaveMode)((newDevice.GetPropertyInt(9520, 2) << 1) + newDevice.GetPropertyInt(9520, 3));
				UISelect slTCPIPSlaveMode = m_slTCPIPSlaveMode;
				int selectedSlaveMode = (int)m_selectedSlaveMode;
				slTCPIPSlaveMode.CustomValue = selectedSlaveMode.ToString();
				m_catTCPIPSlave.Add(AddCustomNumber(9520, 4, 0, "ValidityTime (s)"));
				m_lblInfoTCPIPSlave = (UIPropertyLabel)m_catTCPIPSlave.Add(AddLabelText("\n", EUILabelType.LargeInfo));
				m_configPanel.ShowCategory(m_catTCPIPSlave, show: false);
			}
			if (newDevice.GetPropertyString(8576, 0, 0) != null)
			{
				m_catSCN = m_configPanel.AddCategory("SCN", "Smart charging network");
				m_catSCN.Add(AddSmallHeader("Managing SCN has been moved"));
				m_catSCN.Add(AddInfoText("For creating a new SCN go to the menu\n\t‘Device’ → ‘Add to a new SCN’"));
				m_catSCN.Add(AddInfoText("For adding a charger to an existing SCN or change SCN settings\nclick on the SCN symbol in the charger overview."));
				m_stSCNName = (UIPropertyString)m_catSCN.Add(AddText(8576, 1, "Network name"));
				m_stSCNName.SetEnable(newDevice.GetPropertyString(8576, 1, 0) != string.Empty);
				m_stSCNName.Changed += OnScnNameChanged;
				m_numSCNTotalCurrent = (UIPropertyNumber)m_catSCN.Add(AddCustomNumber(8576, 5, 0, "Total current (A)"));
				m_numSCNTotalCurrent.SetValueMinMax(1.0, 3.4028234663852886E+38);
				m_numSCNTotalCurrent.SetEnable(fEnable: false);
				m_numSCNSocketSafeCurrent = (UIPropertyNumber)m_catSCN.Add(AddCustomNumber(8576, 6, 0, "Socket Safe Current (A)"));
				m_numSCNSocketSafeCurrent.SetValueMinMax(6.0, 64.0);
				m_numSCNSocketSafeCurrent.SetEnable(fEnable: false);
				if ((newDevice as ICULanDevice).HasProperty(8576, 10))
				{
					m_numSCNTotalSafeCurrent = (UIPropertyNumber)m_catSCN.Add(AddCustomNumber(8576, 10, 0, "Total Safe Current (A)"));
					m_numSCNTotalSafeCurrent.SetValueMinMax(0.0, 3.4028234663852886E+38);
					m_numSCNTotalSafeCurrent.SetEnable(fEnable: false);
				}
				m_numSCNAltPeriod = (UIPropertyNumber)m_catSCN.Add(AddCustomNumber(8576, 4, 0, "Alternating period (s)"));
				m_numSCNAltPeriod.SetValueMinMax(900.0, 36000.0);
				m_numSCNAltPeriod.SetEnable(fEnable: false);
				m_catSCN.Add(AddReadOnlyText(8576, 2, "Socket ID"), advancedProperty: true);
				m_catSCN.Add(AddReadOnlyText(8576, 3, "Number of sockets"), advancedProperty: true);
				m_catSCN.Add(AddSelect(8576, 7, 0, 0, "Phase mapping 1"), advancedProperty: true);
				if (iCULanDevice.NumberOfFeederCables > 1)
				{
					m_catSCN.Add(AddSelect(8576, 9, 0, 0, "Phase mapping 2"), advancedProperty: true);
				}
				m_lblInfoSCN = (UIPropertyLabel)m_catSCN.Add(AddLabelText("", EUILabelType.LargeInfo));
			}
			if (iCULanDevice.HasProperty(8593, 1))
			{
				m_catP1Options = m_configPanel.AddCategory("DSMR/SMR (P1)", "DSMR/SMR (P1)");
				m_slP1Interface = (UIPropertySelect)m_catP1Options.Add(AddCustomSelect(8593, 1, "DSMR/SMR interface", m_dicP1Interface));
				m_slP1Interface.Changed -= OnP1InterfaceChanged;
				m_slP1Interface.Changed += OnP1InterfaceChanged;
				m_stP1ServerAddress = (UIPropertyString)m_catP1Options.Add(AddText(8593, 2, "DSMR/SMR Server IP address"));
				m_cnP1ServerPort = (UIPropertyNumber)m_catP1Options.Add(AddCustomNumber(8593, 3, 0, "DSMR/SMR Server port"));
			}
			if (iCULanDevice.HasProperty(12920, 1) && iCULanDevice.ChargingProfiles.SupportsChargingProfiles())
			{
				iCULanDevice.ChargingProfiles.Initialize();
				bool isUKSmartChargingProfileInstalled = iCULanDevice.ChargingProfiles.IsUKSmartChargingProfileInstalled;
				m_catUKSmartCharging = m_configPanel.AddCategory("Charging profiles", "Charging profiles");
				m_chkEnabledUKSmartCharging = (UICheckbox)m_catUKSmartCharging.Add(AddPlainBool("UK Smart Charging compliance", isUKSmartChargingProfileInstalled));
				m_lblUKSmartCharging = (UIPropertyLabel)m_catUKSmartCharging.Add(AddLabelText("\n", EUILabelType.Info));
				m_cnUKRandomDelay = (UIPropertyNumber)m_catUKSmartCharging.Add(AddCustomNumber(8633, 0, 0, "Random delay (s)"));
				m_catUKSmartCharging.Add(AddCustomSelect(12920, 1, "Direct start on socket 1", m_dicOverrides), advancedProperty: true);
				m_catUKSmartCharging.Add(AddCustomSelect(12920, 2, "Direct start on socket 2", m_dicOverrides), advancedProperty: true);
			}
			m_catRTUmodbus = m_configPanel.AddCategory("Modbus RTU", "Modbus RTU");
			m_stRTUModbusAddress = (UIPropertyString)m_catRTUmodbus.Add(AddText(9589, 2, "Address"));
			m_slRTUModbusParity = (UIPropertySelect)m_catRTUmodbus.Add(AddSelect(9589, 1, 0, 0));
			m_slRTUModbusBaudrate = (UIPropertySelect)m_catRTUmodbus.Add(AddSelect(9589, 0, 0, 0));
			m_slRTUModbusWordOrder = (UIPropertySelect)m_catRTUmodbus.Add(AddSelect(9588, 0, 0, 0));
			m_cnRTUModbusUpdateTime = (UIPropertyNumber)m_catRTUmodbus.Add(AddCustomNumber(9588, 1, 0, "Update Time (ms)"));
			m_cnRTUModbusReadTimeOut = (UIPropertyNumber)m_catRTUmodbus.Add(AddCustomNumber(9588, 2, 0, "Read TimeOut (ms)"));
			m_slRTUModbusFunction = (UIPropertySelect)m_catRTUmodbus.Add(AddSelect(9588, 3, 0, 0));
			m_catRTUmodbus.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
			AddCustomButton(m_btnConfigureModbusRTU);
			m_catRTUmodbus.AddWidget(m_btnConfigureModbusRTU);
			AddCustomButton(m_btnTestModbusMeterRTU);
			m_catRTUmodbus.AddWidget(m_btnTestModbusMeterRTU);
			if (iCULanDevice.HasProperty(12928, 1) && (iCULanDevice.FirmwareVersionNumber >= new Version("6.3.0") || iCULanDevice.isAHP))
			{
				m_catSolarCharging = m_configPanel.AddCategory("Solar charging", "Solar Charging");
				m_lblSolarChargingInfo = (UIPropertyLabel)m_catSolarCharging.Add(AddLabelText("", EUILabelType.Info));
				m_slSolarChargingMode = (UIPropertySelect)m_catSolarCharging.Add(AddCustomSelect(12928, 1, "Charging mode", m_dicSolarChargingModes));
				m_cnSolarChargingGreenShare = (UIPropertyNumber)m_catSolarCharging.Add(AddCustomNumber(12928, 2, 0, "Green share (%)"));
				m_cnSolarChargingComfortLevel = (UIPropertyNumber)m_catSolarCharging.Add(AddCustomNumber(12928, 3, 0, "Comfort level (W)"));
				m_cnSolarChargingComfortLevel.SetValueMinMax(0.0, 22000.0, 100.0);
				m_chkSolarChargingBoost1 = (UIPropertyCheckbox)m_catSolarCharging.Add(AddCheckBox(12928, 4, 0, "Socket 1: boost charging"));
				m_chkSolarChargingBoost2 = (UIPropertyCheckbox)m_catSolarCharging.Add(AddCheckBox(12928, 5, 0, "Socket 2: boost charging"));
			}
		}
		OnUpdateControls(PageID);
		return true;
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		if (m_currentDevice != null)
		{
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice == null)
			{
				return false;
			}
			List<UIConfigCategory> list = new List<UIConfigCategory> { m_catActiveLB, m_catTCPIPMaster, m_catRTUmodbus, m_catStaticLB };
			List<UIConfigCategory> list2 = new List<UIConfigCategory> { m_catActiveLB, m_catTCPIPMaster, m_catRTUmodbus, m_catStaticLB, m_catP1Options, m_catTCPIPSlave };
			bool flag = DidPropertiesChangeInCategories(currentDevice.isAHP ? list : list2);
			if (flag)
			{
				currentDevice.storeProperty(8292, 0, 0);
			}
			if (m_slSmartMeterType != null && m_slSmartMeterType.IsChanged)
			{
				ICUProperty property;
				if (m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART)
				{
					if (!m_showAdvancedSmartmeterSettings)
					{
						property = m_currentDevice.GetProperty(9507, 2);
						if (property != null)
						{
							property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Socomec");
						}
					}
					if (currentDevice.HasProperty(9507, 3))
					{
						property = m_currentDevice.GetProperty(9507, 3);
						property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Modbus_master_TCP");
					}
				}
				else if (m_selectedSmartMeter == EMeterTypes.ENERGYMETER_RTU_SMART)
				{
					property = m_currentDevice.GetProperty(21016, 0);
					property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Custom");
				}
				property = currentDevice.GetProperty(9507, 1);
				property.Value = ((m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART) ? 1 : 0);
			}
			if (m_chkTCPIPMasterEnabled != null && m_chkTCPIPMasterEnabled.IsChecked && currentDevice.HasProperty(9506, 2))
			{
				ICUProperty property = m_currentDevice.GetProperty(9506, 2);
				property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Socomec");
			}
			if (m_slActiveLBDataSource != null && m_slActiveLBDataSource.IsChanged)
			{
				ICUProperty property = m_currentDevice.GetProperty(9520, 1);
				property.Value = (byte)((m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_WRITE) ? ETCPIPSlaveOptions.TCPIPSLAVE_ALL : m_selectedDataSource);
				if (m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_ALL)
				{
					property = m_currentDevice.GetProperty(21015, 0);
					property.Value = -1;
				}
				property = m_currentDevice.GetProperty(9507, 1);
				property.Value = ((m_selectedDataSource != ETCPIPSlaveOptions.TCPIPSLAVE_ALL && m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART) ? 1 : 0);
			}
			if (m_slTCPIPSlaveMode != null)
			{
				if (m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_ALL && m_selectedSlaveMode != ETCPIPSlaveMode.BALANCEMODE_SOCKET && m_selectedSlaveMode != ETCPIPSlaveMode.BALANCEMODE_SCN)
				{
					m_selectedSlaveMode = ETCPIPSlaveMode.BALANCEMODE_SOCKET;
				}
				ICUProperty property = m_currentDevice.GetProperty(9520, 2);
				if (property != null)
				{
					property.Value = ((m_selectedSlaveMode == ETCPIPSlaveMode.BALANCEMODE_SOCKET && m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_ALL) ? 1 : 0);
				}
				property = m_currentDevice.GetProperty(9520, 3);
				if (property != null)
				{
					property.Value = ((m_selectedSlaveMode == ETCPIPSlaveMode.BALANCEMODE_SCN && m_selectedDataSource == ETCPIPSlaveOptions.TCPIPSLAVE_ALL) ? 1 : 0);
				}
			}
			if (m_chkEnabledUKSmartCharging != null && m_chkEnabledUKSmartCharging.IsChanged)
			{
				if (m_chkEnabledUKSmartCharging.IsChecked)
				{
					if (currentDevice.ChargingProfiles.AddUkSmartChargingProfile())
					{
						if (m_currentDevice.GetPropertyUInt(8633, 0) < 600)
						{
							ICUProperty property = m_currentDevice.GetProperty(8633, 0);
							if (property != null)
							{
								property.Value = 600u;
								m_fRefreshPage = true;
							}
						}
						m_chkEnabledUKSmartCharging.CustomValue = m_chkEnabledUKSmartCharging.IsChecked;
					}
				}
				else if (currentDevice.ChargingProfiles.ClearUKSmartChargingProfile())
				{
					m_chkEnabledUKSmartCharging.CustomValue = m_chkEnabledUKSmartCharging.IsChecked;
				}
			}
			m_hasSCNNameChanged = m_stSCNName != null && m_stSCNName.IsChanged;
			m_currentDevice.StoreChangedProperties();
			if (flag)
			{
				currentDevice.storeProperty(8292, 0, m_chkActiveLBEnabled.IsChecked ? 3 : ((m_chkStaticLBEnabled != null && m_chkStaticLBEnabled.IsChecked) ? 1 : 0));
				if (m_chkTCPIPMasterEnabled == null)
				{
					currentDevice.storeProperty(9507, 1, (m_selectedSmartMeter == EMeterTypes.ENERGYMETER_TCPIP_SMART && m_chkActiveLBEnabled.IsChecked) ? 1 : 0);
				}
			}
		}
		return true;
	}

	public override void OnPostSaveChanges()
	{
		if (m_currentDevice == null)
		{
			return;
		}
		m_selectedSlaveMode = (ETCPIPSlaveMode)((m_currentDevice.GetPropertyInt(9520, 2) << 1) + m_currentDevice.GetPropertyInt(9520, 3));
		if (m_slTCPIPSlaveMode != null)
		{
			UISelect slTCPIPSlaveMode = m_slTCPIPSlaveMode;
			int selectedSlaveMode = (int)m_selectedSlaveMode;
			slTCPIPSlaveMode.CustomValue = selectedSlaveMode.ToString();
		}
		if (m_hasSCNNameChanged)
		{
			MainWindow mainWindow = (MainWindow)ParentWindow;
			mainWindow.SuspendUpdateTimers(fSuspend: true);
			try
			{
				using DlgReboot dlgReboot = new DlgReboot(m_currentDevice, fAutoStartReboot: true);
				dlgReboot.Run(ParentWindow);
			}
			finally
			{
				mainWindow.SuspendUpdateTimers(fSuspend: false);
				m_parent.RefreshDeviceList();
			}
		}
		if (m_fRefreshPage)
		{
			ChangeDevice(m_currentDevice);
			m_fRefreshPage = false;
		}
	}

	public override void OnRevertChanges()
	{
		if (m_currentDevice != null)
		{
			if (m_slTCPIPSlaveMode != null)
			{
				m_selectedSlaveMode = (ETCPIPSlaveMode)((m_currentDevice.GetPropertyInt(9520, 2) << 1) + m_currentDevice.GetPropertyInt(9520, 3));
				UISelect slTCPIPSlaveMode = m_slTCPIPSlaveMode;
				int selectedSlaveMode = (int)m_selectedSlaveMode;
				slTCPIPSlaveMode.CustomValue = selectedSlaveMode.ToString();
			}
			OnUpdateControls(PageID);
			base.OnRevertChanges();
		}
	}

	private void OnConfigureModbusClicked(object sender, EventArgs e)
	{
		new DlgModbusRegisterMap(m_currentDevice, m_selectedSmartMeter).Run(m_parent);
	}

	private void OnButtonTestSmartMeterClicked(object sender, EventArgs e)
	{
		new DlgSmartMeterTest(m_currentDevice, m_selectedSmartMeter).Run(m_parent);
	}
}
