using System;
using System.Collections.Generic;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;

namespace ICUServiceInstaller;

public class PanelPower : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelPower>();

	protected List<UIPropertyBase> m_lstCentralModbusTCPControls = new List<UIPropertyBase>();

	protected UIPropertyString m_stCentralModbusTCPIPAddress;

	protected UIPropertyNumber m_cnCentralModbusTCPSlaveAddress;

	protected UIPropertyLabel m_lblInfoMainCurrent1;

	protected UIPropertyLabel m_lblInfoMainCurrent2;

	protected UIPropertyLabel m_lblInfoStaticLoadBalancing;

	protected UIPropertyLabel m_lblInfoActiveLoadBalancing;

	protected UIPropertyLabel m_lblInfoStationCurrent;

	protected UIPropertySelect m_slCentralMeterType;

	protected EMeterTypes m_selectedCentralMeter = EMeterTypes.ENERGYMETER_NONE;

	protected UIPropertySelect m_slCentralModbusType;

	protected UIPropertyNumber m_cnMaxImbalanceCurrent;

	private readonly UIPropertyNumber[] m_numConnectorCurrent = new UIPropertyNumber[2];

	private readonly UIPropertyNumber[] m_numConnectorPower = new UIPropertyNumber[2];

	private UIPropertyNumber m_numMaxStationCurrent;

	private UIPropertyNumber m_cnChameleonMinCurrent;

	protected UICheckbox m_chkEnabledImbalanceCurrent;

	protected UICheckbox[] m_chkEnabledMaxCurrent = new UICheckbox[2];

	private static readonly bool[] m_isMaxCurrentEnabled = new bool[2] { true, true };

	protected bool m_showLegacyModBusTCPSettings;

	protected bool m_showFW41orHigherSettings;

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catConnector1;

	protected UIConfigCategory m_catConnector2;

	protected UIConfigCategory m_catInstallation;

	protected UIConfigCategory m_catCarSpecific;

	protected UIConfigCategory m_catLoadBalancing;

	protected UIConfigCategory m_catCentralMeter;

	protected UIConfigCategory m_catIVUAdapter;

	private static readonly string m_maxDCCurrentInfoText = "Limit the current based on the used fuses in the installation";

	protected Dictionary<string, string> m_dicMaxPhases = new Dictionary<string, string>();

	public PanelPower(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Power";
		Tooltip = "Power settings";
		IconName = "electricity.png";
	}

	protected override void OnChangeProperty()
	{
		base.OnChangeProperty();
		if (m_configPanel != null)
		{
			m_configPanel.UpdateControls();
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
			uint propertyUInt = m_currentDevice.GetPropertyUInt(8610, 0);
			Version firmwareVersionNumber = m_currentDevice.FirmwareVersionNumber;
			if (m_slCentralMeterType != null)
			{
				OnCentralMeterTypeChanged();
			}
			string text = string.Empty;
			string value = string.Empty;
			string value2 = string.Empty;
			int propertyInt = m_currentDevice.GetPropertyInt(8489, 0);
			int num = ((m_currentDevice.NumberOfSockets > 1) ? m_currentDevice.GetPropertyInt(12585, 0) : 0);
			bool flag = m_currentDevice.GetPropertyInt(8292, 0, 2) > 0;
			int propertyInt2 = m_currentDevice.GetPropertyInt(8290, 0);
			int num2 = ((flag || !m_showFW41orHigherSettings) ? m_currentDevice.GetPropertyInt(8295, 0) : int.MaxValue);
			string text2;
			if (propertyInt2 < num2)
			{
				text2 = $"maximum station current of {propertyInt2} A.";
			}
			else
			{
				text2 = (m_showFW41orHigherSettings ? $"maximum smart meter current {num2} A." : $"maximum installation current {num2} A.");
			}
			_ = m_currentDevice;
			bool flag2 = m_currentDevice.GetDeviceValueInt(8292, 0, 1) != 0;
			bool flag3 = m_currentDevice.GetDeviceValueInt(21015, 0) != -1;
			if (flag & flag3)
			{
				if (propertyInt > Math.Min(propertyInt2, num2))
				{
					text = "Warning! The max current(s) cannot be higher than the " + text2;
				}
				if (num > Math.Min(propertyInt2, num2))
				{
					value = "Warning! The max current(s) cannot be higher than the " + text2;
				}
			}
			else if (flag2)
			{
				if (propertyInt > propertyInt2)
				{
					text = $"Warning! The max current(s) cannot be higher than the maximum installation current {propertyInt2} A.";
				}
				if (num > propertyInt2)
				{
					value = $"Warning! The max current(s) cannot be higher than the maximum installation current {propertyInt2} A.";
				}
			}
			else if (propertyInt + num > Math.Min(propertyInt2, num2))
			{
				bool flag4 = false;
				if (m_currentDevice != null && m_currentDevice.UpdateProperties(2195457u, 2195464u) && m_currentDevice.GetPropertyBool(8576, 8))
				{
					flag4 = !string.IsNullOrEmpty(m_currentDevice.GetPropertyString(8576, 1, 0).Trim());
				}
				if (!flag4)
				{
					text = "Warning! The max current(s) cannot be higher than the " + text2;
					if (m_currentDevice.NumberOfSockets > 1)
					{
						value = text;
					}
				}
			}
			if (propertyInt > 16 && !IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.HighPowerSockets, m_currentDevice.isAHP) && string.IsNullOrEmpty(text))
			{
				text = string.Format("This product is not licensed to use more then 16 A", Array.Empty<object>());
			}
			if (num > 16 && !IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.HighPowerSockets, m_currentDevice.isAHP) && string.IsNullOrEmpty(value))
			{
				value = string.Format("This product is not licensed to use more then 16 A", Array.Empty<object>());
			}
			if ((flag & flag3) && propertyInt2 > num2)
			{
				value2 = string.Format("Warning! Maximum station current cannot be higher than the maximum " + (m_showFW41orHigherSettings ? "smart meter " : "installation ") + "current of {0} A", num2);
			}
			if (m_lblInfoMainCurrent1 != null)
			{
				m_lblInfoMainCurrent1.Hide = string.IsNullOrEmpty(text);
				m_lblInfoMainCurrent1.SetValue(text);
			}
			if (m_lblInfoMainCurrent2 != null)
			{
				m_lblInfoMainCurrent2.Hide = string.IsNullOrEmpty(value);
				m_lblInfoMainCurrent2.SetValue(value);
			}
			if (m_lblInfoStationCurrent != null)
			{
				m_lblInfoStationCurrent.Hide = string.IsNullOrEmpty(value2);
				m_lblInfoStationCurrent.SetValue(value2);
			}
			if (m_chkEnabledImbalanceCurrent != null && m_chkEnabledImbalanceCurrent.IsChanged)
			{
				m_cnMaxImbalanceCurrent.SetEnable(m_chkEnabledImbalanceCurrent.IsChecked);
			}
		}
		else
		{
			m_lstCentralModbusTCPControls.ForEach((UIPropertyBase a) =>
			{
				a.SetEnable(fEnable: false);
			});
		}
		return true;
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		newDevice.UpdateCategories("states");
		bool isDC = iCULanDevice.isDC;
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			m_catInstallation = m_configPanel.AddCategory("Installation");
			m_numMaxStationCurrent = (UIPropertyNumber)m_catInstallation.Add(AddCustomNumber(8290, 0, 0, "Station maximum current (A)"));
			m_numMaxStationCurrent.SetValueMinMax(1.0, (isDC || iCULanDevice.NumberOfFeederCables > 1 || iCULanDevice.IsTwin) ? 64 : (iCULanDevice.IsDualPG ? 40 : 32));
			m_showFW41orHigherSettings = iCULanDevice.isAHP || iCULanDevice.FirmwareVersionNumber >= new Version("4.1.0");
			if (!m_showFW41orHigherSettings)
			{
				m_catInstallation.Add(AddCustomNumber(8295, 0, 0, "Installation maximum current (A)", 1.0, null, 2123521u));
			}
			if (!iCULanDevice.isDC)
			{
				m_catInstallation.Add(AddCustomText("Number of Feeder Cables", $"{iCULanDevice.NumberOfFeederCables}"));
			}
			if (iCULanDevice.HasProperty(8585))
			{
				m_dicMaxPhases.Clear();
				if (iCULanDevice.isDC)
				{
					m_dicMaxPhases.Add("3", "3-Phase");
				}
				else
				{
					m_dicMaxPhases.Add("1", "1-Phase");
					if (iCULanDevice.isAHP)
					{
						m_dicMaxPhases.Add("2", "2-Phase");
					}
					m_dicMaxPhases.Add("3", "3-Phase");
				}
				m_catInstallation.Add(AddCustomSelect(8585, 0, "Maximum Allowed Phases", m_dicMaxPhases));
			}
			ICUProperty property = newDevice.GetProperty(2190336u);
			if (property != null && !string.IsNullOrEmpty(property.Category))
			{
				m_catInstallation.Add(AddSelect(8556, 0, 0, 0));
			}
			if (iCULanDevice.HasProperty(8564))
			{
				bool flag = iCULanDevice.GetPropertyInt(8564, 0) != 0;
				m_chkEnabledImbalanceCurrent = (UICheckbox)m_catInstallation.Add(AddPlainBool("Enable Max. Imbalance current", flag, "", null, 2192385u));
				m_cnMaxImbalanceCurrent = (UIPropertyNumber)m_catInstallation.Add(AddCustomNumber(8564, 0, 0, "Maximum Imbalance current (A)", 1.0, null, 2192386u));
				m_cnMaxImbalanceCurrent.SetEnable(flag);
				m_cnMaxImbalanceCurrent.SetValueMinMax(0.0, 32.0);
			}
			m_lblInfoStationCurrent = (UIPropertyLabel)m_catInstallation.Add(AddWarningText("", "", 0, null, 2));
			m_catConnector1 = m_configPanel.AddCategory("Connector 1");
			if (isDC)
			{
				AddCurrentPowerControlBlock(0, m_catConnector1, isDC, OnEnableMaxCurrent1Changed);
			}
			else
			{
				m_numConnectorCurrent[0] = (UIPropertyNumber)m_catConnector1.Add(AddCustomNumber(8489, 0, 0, "Max current (A)"));
				m_numConnectorCurrent[0].SetValueMinMax(1.0, iCULanDevice.isEcogDC ? 50 : 32);
			}
			if (iCULanDevice.HasProperty(8563) && iCULanDevice.GetPropertyInt(8271, 0, 128) == 128)
			{
				m_catConnector1.Add(AddCustomNumber(8563, 0, 0, "Connector 1.2 Max current (A)"));
			}
			m_catConnector1.Add(AddSelect(8485, 0, 0, 0, "Connector type"), advancedProperty: true);
			if (!isDC)
			{
				m_catConnector1.Add(AddReadOnlyText(12590, 0, "Max phases"), advancedProperty: true);
			}
			m_lblInfoMainCurrent1 = (UIPropertyLabel)m_catConnector1.Add(AddWarningText("", "", 0, null, 2));
			if (m_newDevice.NumberOfSockets > 1)
			{
				m_catConnector2 = m_configPanel.AddCategory("Connector 2");
				if (isDC)
				{
					AddCurrentPowerControlBlock(1, m_catConnector2, isDC, OnEnableMaxCurrent2Changed);
				}
				else
				{
					m_numConnectorCurrent[1] = (UIPropertyNumber)m_catConnector2.Add(AddCustomNumber(12585, 0, 0, "Max current (A)"));
					m_numConnectorCurrent[1].SetValueMinMax(1.0, isDC ? 50 : 32);
				}
				if (iCULanDevice.HasProperty(12659) && iCULanDevice.GetPropertyInt(8271, 0, 128) == 128)
				{
					m_catConnector2.Add(AddCustomNumber(12659, 0, 0, "Connector 2.2 Max current (A)"));
				}
				m_catConnector2.Add(AddSelect(12581, 0, 0, 0, "Connector type"), advancedProperty: true);
				if (!isDC)
				{
					m_catConnector2.Add(AddReadOnlyText(12591, 0, "Max phases"), advancedProperty: true);
				}
				m_lblInfoMainCurrent2 = (UIPropertyLabel)m_catConnector2.Add(AddWarningText("", "", 0, null, 2));
			}
			if (!isDC)
			{
				m_catCarSpecific = m_configPanel.AddCategory("Car specific", "Car specific settings");
				m_catCarSpecific.Add(AddCheckBox(8537, 0, 0, "ZE ready"), advancedProperty: true);
				m_catCarSpecific.Add(AddCheckBox(8541, 0, 0, "Disable 105 percent overcurrent"), advancedProperty: true);
				m_cnChameleonMinCurrent = (UIPropertyNumber)m_catCarSpecific.Add(AddCustomNumber(8298, 0, 0, "Chameleon min current (A)"), advancedProperty: true);
				m_cnChameleonMinCurrent.SetValueMinMax(0.0, 3.4028234663852886E+38);
			}
			if (newDevice.GetProperty(4331264u).Category != null && m_newDevice.NumberOfSockets > 1)
			{
				m_lstCentralModbusTCPControls.Clear();
				m_catCentralMeter = m_configPanel.AddCategory("Central meter");
				m_slCentralMeterType = (UIPropertySelect)m_catCentralMeter.Add(AddSelect(16919, 0, 0, 0, "Protocol Selection"));
				m_slCentralModbusType = (UIPropertySelect)m_catCentralMeter.Add(AddSelect(16920, 0, 0, 0, "Modbus Type"));
				m_stCentralModbusTCPIPAddress = (UIPropertyString)m_catCentralMeter.Add(AddText(9506, 4, "IP Address"));
				m_cnCentralModbusTCPSlaveAddress = (UIPropertyNumber)m_catCentralMeter.Add(AddCustomNumber(9506, 6, 0, "Slave Address"));
				m_lstCentralModbusTCPControls.Add(m_stCentralModbusTCPIPAddress);
				m_lstCentralModbusTCPControls.Add(m_cnCentralModbusTCPSlaveAddress);
			}
			m_catIVUAdapter = m_configPanel.AddCategory("IVU adapter");
			if (newDevice.GetPropertyString(8728, 0, 0) == "IVU")
			{
				m_catIVUAdapter.Add(AddSmallHeader("Socket 1"));
				m_catIVUAdapter.Add(AddReadOnlyText(9537, 0, "Public Key", UIPropertyStringType.PublicKey));
			}
			if (newDevice.GetPropertyString(12824, 0, 0) == "IVU")
			{
				m_catIVUAdapter.Add(AddSmallHeader("Socket 2"));
				m_catIVUAdapter.Add(AddReadOnlyText(9538, 0, "Public Key", UIPropertyStringType.PublicKey));
			}
		}
		return true;
	}

	private void AddCurrentPowerControlBlock(int connector, UIConfigCategory cat, bool isEcogDC, EventHandler changedHandler)
	{
		if (connector >= 0 && connector <= 1)
		{
			cat.Add(AddInfoText(m_maxDCCurrentInfoText));
			m_chkEnabledMaxCurrent[connector] = (UICheckbox)cat.Add(AddPlainBool("Configure maximum using current", m_isMaxCurrentEnabled[connector]));
			m_chkEnabledMaxCurrent[connector].Changed += changedHandler;
			m_numConnectorCurrent[connector] = (UIPropertyNumber)cat.Add(AddCustomNumber((ushort)((connector == 0) ? 8489u : 12585u), 0, 0, "Max current (A)"));
			m_numConnectorCurrent[connector].SetValueMinMax(1.0, isEcogDC ? 50 : 32);
			m_numConnectorCurrent[connector].SetEnable(m_isMaxCurrentEnabled[connector]);
			m_numConnectorPower[connector] = (UIPropertyNumber)cat.Add(AddCustomNumber(33793, (byte)(connector + 1), 0, "Max power (W)"));
			m_numConnectorPower[connector].SetEnable(!m_isMaxCurrentEnabled[connector]);
		}
	}

	private void OnCentralMeterTypeChanged()
	{
		bool showMBTCP = false;
		bool flag = false;
		m_selectedCentralMeter = (EMeterTypes)Convert.ToInt16(m_slCentralMeterType.GetValue());
		switch (m_selectedCentralMeter)
		{
		case EMeterTypes.ENERGYMETER_MODBUS_CENTRAL:
			flag = true;
			break;
		case EMeterTypes.ENERGYMETER_TCPIP_CENTRAL:
			showMBTCP = true;
			break;
		}
		m_slCentralModbusType.Hide = !flag;
		m_lstCentralModbusTCPControls.ForEach((UIPropertyBase a) =>
		{
			a.Hide = !showMBTCP;
		});
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		if (m_currentDevice != null)
		{
			bool num = m_slCentralMeterType != null && m_slCentralMeterType.IsChanged && m_selectedCentralMeter == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL;
			bool flag = m_slCentralMeterType != null && m_slCentralMeterType.IsChanged && m_selectedCentralMeter == EMeterTypes.ENERGYMETER_FKN_METER;
			List<ICUProperty> list = new List<ICUProperty>();
			if (num)
			{
				ICUProperty property = m_currentDevice.GetProperty(9506, 1);
				property.Value = 1;
				list.Add(property);
				property = m_currentDevice.GetProperty(9506, 2);
				property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Socomec");
				list.Add(property);
				property = m_currentDevice.GetProperty(9506, 3);
				if (property != null)
				{
					property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "Modbus_master_TCP");
					list.Add(property);
				}
			}
			else if (flag)
			{
				ICUProperty property = m_currentDevice.GetProperty(16920, 0);
				property.Value = DataSheet.SelectParameterOption(property.Id, property.SubId, "ABB");
				list.Add(property);
			}
			if (m_chkEnabledImbalanceCurrent != null)
			{
				m_chkEnabledImbalanceCurrent.CustomValue = m_chkEnabledImbalanceCurrent.IsChecked;
				if (!m_chkEnabledImbalanceCurrent.IsChecked)
				{
					ICUProperty property = m_currentDevice.GetProperty(8564, 0);
					if (property != null)
					{
						property.Value = 0;
						list.Add(property);
					}
				}
			}
			if (m_chkEnabledMaxCurrent != null)
			{
				for (int i = 0; i < 2; i++)
				{
					if (m_chkEnabledMaxCurrent[i] != null)
					{
						if (m_chkEnabledMaxCurrent[i].IsChecked)
						{
							m_numConnectorPower[i].SetValue(Convert.ToInt32(m_numConnectorCurrent[i].GetValue()) * 230 * 3);
						}
						else
						{
							m_numConnectorCurrent[i].SetValue(Convert.ToInt32(m_numConnectorPower[i].GetValue()) / 230 / 3);
						}
						m_chkEnabledMaxCurrent[i].CustomValue = m_chkEnabledMaxCurrent[i].IsChecked;
					}
				}
			}
			if (list.Count > 0)
			{
				m_currentDevice.StoreProperties(list.ToArray());
			}
		}
		return true;
	}

	public override void OnPostSaveChanges()
	{
		base.OnPostSaveChanges();
		base.OnChangeProperty();
	}

	public override void OnRevertChanges()
	{
		if (m_currentDevice != null)
		{
			bool flag = m_currentDevice.GetPropertyInt(8564, 0) != 0;
			if (m_chkEnabledImbalanceCurrent != null)
			{
				m_chkEnabledImbalanceCurrent.CustomValue = flag;
			}
			if (m_cnMaxImbalanceCurrent != null)
			{
				m_cnMaxImbalanceCurrent.SetEnable(flag);
			}
		}
		base.OnRevertChanges();
	}

	private void OnEnableMaxCurrent1Changed(object sender, EventArgs e)
	{
		HandleMaxCurrentEnabledChanged(0);
	}

	private void OnEnableMaxCurrent2Changed(object sender, EventArgs e)
	{
		HandleMaxCurrentEnabledChanged(1);
	}

	private void HandleMaxCurrentEnabledChanged(int connector)
	{
		if (connector >= 0 && connector <= 1)
		{
			m_isMaxCurrentEnabled[connector] = m_chkEnabledMaxCurrent[connector].IsChecked;
			m_numConnectorCurrent[connector].SetEnable(m_chkEnabledMaxCurrent[connector].IsChecked);
			m_numConnectorPower[connector].SetEnable(!m_chkEnabledMaxCurrent[connector].IsChecked);
		}
	}
}
