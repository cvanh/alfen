using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using ICUIWSConnection;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelMonitoring : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelMonitoring>();

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catStates;

	protected UIConfigCategory m_catCommunicationCar;

	protected UIConfigCategory m_catVoltage;

	protected UIConfigCategory m_catConnectivity;

	protected UIConfigCategory m_catCurrent;

	protected UIConfigCategory m_catNetQuality;

	protected UIConfigCategory m_catSensors;

	protected UIConfigCategory m_catFans;

	protected Dictionary<string, string> m_dicAHWPMainStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicAHWPCCStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicAHWPCPROStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMainStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicLEDStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicMode3States = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSocketStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSocketPowerStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicOCPPBootStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicModbusTCPIPConnectionStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicSocketStatus = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicUserInterfaceStates = new Dictionary<string, string>();

	protected Dictionary<string, string> m_dicPaymentGiroEStates = new Dictionary<string, string>();

	protected UIPropertyManualSelect m_selOperationState1;

	protected UIPropertyManualSelect m_selOperationState2;

	protected UIPropertyString m_txtXYZ;

	protected UIPropertyString m_lblSocket1DisplayText;

	protected UIPropertyString m_lblSocket2DisplayText;

	protected List<DisplayObjectOld> TDisplayStatesMessagesOld;

	protected List<DisplayObjectNew> TDisplayStatesMessagesNew;

	protected bool m_showLegacySmartMeterValues;

	protected bool m_hasUserInterfaceStates;

	private static uint m_updateSwitch;

	public override bool IsChanged
	{
		get
		{
			List<UIPropertyBase> list = AllProperties.Where((UIPropertyBase a) => a.IsChanged).ToList();
			if (m_selOperationState1 != null && m_selOperationState1.IsChanged)
			{
				return true;
			}
			if (m_selOperationState2 != null && m_selOperationState2.IsChanged)
			{
				return true;
			}
			return list.Count > 0;
		}
	}

	public PanelMonitoring(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Live monitoring";
		Tooltip = "Live monitoring";
		IconName = "technology-3.png";
		string[] names = Enum.GetNames(typeof(EAHWPCSMMainStates));
		foreach (string value in names)
		{
			m_dicAHWPMainStates.Add(((int)Enum.Parse(typeof(EAHWPCSMMainStates), value)).ToString(), value);
		}
		names = Enum.GetNames(typeof(EAHWPCCStates));
		foreach (string value2 in names)
		{
			m_dicAHWPCCStates.Add(((int)Enum.Parse(typeof(EAHWPCCStates), value2)).ToString(), value2);
		}
		names = Enum.GetNames(typeof(EAHWPCPROStates));
		foreach (string value3 in names)
		{
			m_dicAHWPCPROStates.Add(((int)Enum.Parse(typeof(EAHWPCPROStates), value3)).ToString(), value3);
		}
		names = Enum.GetNames(typeof(EMainStates));
		foreach (string value4 in names)
		{
			m_dicMainStates.Add(((int)Enum.Parse(typeof(EMainStates), value4)).ToString(), value4);
		}
		names = Enum.GetNames(typeof(ELEDStates));
		foreach (string value5 in names)
		{
			m_dicLEDStates.Add(((int)Enum.Parse(typeof(ELEDStates), value5)).ToString(), value5);
		}
		names = Enum.GetNames(typeof(ESocketStates));
		foreach (string value6 in names)
		{
			m_dicSocketStates.Add(((int)Enum.Parse(typeof(ESocketStates), value6)).ToString(), value6);
		}
		names = Enum.GetNames(typeof(ESocketPowerStates));
		foreach (string value7 in names)
		{
			m_dicSocketPowerStates.Add(((int)Enum.Parse(typeof(ESocketPowerStates), value7)).ToString(), value7);
		}
		names = Enum.GetNames(typeof(EMode3States));
		foreach (string value8 in names)
		{
			m_dicMode3States.Add(((int)Enum.Parse(typeof(EMode3States), value8)).ToString(), value8);
		}
		names = Enum.GetNames(typeof(EBootNoticationStates));
		foreach (string value9 in names)
		{
			m_dicOCPPBootStates.Add(((int)Enum.Parse(typeof(EBootNoticationStates), value9)).ToString(), value9);
		}
		names = Enum.GetNames(typeof(EUserInterfaceStates));
		foreach (string value10 in names)
		{
			m_dicUserInterfaceStates.Add(((int)Enum.Parse(typeof(EUserInterfaceStates), value10)).ToString(), value10);
		}
		names = Enum.GetNames(typeof(EGiroEState));
		foreach (string value11 in names)
		{
			m_dicPaymentGiroEStates.Add(((int)Enum.Parse(typeof(EGiroEState), value11)).ToString(), value11);
		}
		names = Enum.GetNames(typeof(EModbusTCPIPConnectionStates));
		foreach (string text in names)
		{
			int num = (int)Enum.Parse(typeof(EModbusTCPIPConnectionStates), text);
			string text2 = "";
			text2 = num switch
			{
				0 => "NOT IN USE (" + text + ")", 
				1 => "CONNECTING (" + text + ")", 
				2 => "CONNECTED (" + text + ")", 
				3 => "CONNECTION INTERRUPTED (" + text + ")", 
				4 => "NO CONNECTION (" + text + ")", 
				_ => "UNKNOWN (" + text + ")", 
			};
			m_dicModbusTCPIPConnectionStates.Add(num.ToString(), text2);
		}
		m_dicSocketStatus.Add("0", "Operative");
		m_dicSocketStatus.Add("1", "In-operative");
		TDisplayStatesMessagesOld = new List<DisplayObjectOld>
		{
			new DisplayObjectOld(EMainStates.STATE_AVAILABLE, "Installation OK", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectOld(EMainStates.STATE_ERROR, "001: Not able to charge.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_MESSAGE, "002: Charging not started yet,\nto continue please reconnect cable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_ILLEGAL_MODE_3, "201: No communication with vehicle.\nPlease check your charging cable", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_TOO_MANY_RESTARTS, "003: Too many retries.\nPlease check your charging cable", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_CHARGING, "004: One moment please...\nYour charging session will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_CHARGING_OVERCURRENT, "005: One moment please...\nYour charging session will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING, "006: One moment please...\nYour charging session will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_POWERMETER, "202: Not able to charge.", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_TEMPERATURE, "203: Inside temperature high.\nCharging will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_INOPERATIVE, "204: Temporary set to unavailable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectOld(EMainStates.STATE_ERROR_S2_NOT_OPENED, "007: S2 not opened.\nPlease reconnect cable.", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_PROTECTIVE_EARTH, "101: Error in installation.\nPlease Check installation", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_RELAYS, "102: Not able to Charge", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_LOW_SUPPLY_VOLTAGE, "103: Input Voltage too low,\nnot able to charge.", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectOld(EMainStates.STATE_ERROR_INTERNAL_VOLTAGE, "104: Not able to charge.", EStatusIcon.STATUS_ICON_ERROR)
		};
		TDisplayStatesMessagesNew = new List<DisplayObjectNew>
		{
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_NONE, "Installation OK", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_GENERIC, "Not able to charge.\n Please call for support", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_CHARGING_RCD, "One moment please...\nYour charging session will resume shortly", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_RELAYS, "Not able to charge.\n Please call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_INTERNAL_VOLTAGE, "Not able to charge.\n Please call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_POWERMETER, "Not able to charge.\n Please call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_RCD, "Not able to charge.\n Please call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_SOCKET_MOTOR_STARTUP_OLD, "Not able to lock cable\nPlease call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_MISSINGPCID, "", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_NFCREADER, "", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_PROTECTIVE_EARTH, "Error in installation.\nPlease Check installation or call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_LOW_SUPPLY_VOLTAGE, "Input Voltage too low, not able to charge.\nPlease call your installer", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_INOPERATIVE, "Temporary set to unavailable.\nContact CPO or try again later", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_HIGH_SUPPLY_VOLTAGE, "Error high voltage", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_P1PPORT, "Error P1 port", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_MODBUSTCPIP, "Error ModbusTCPIP port", EStatusIcon.STATUS_ICON_VALID),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_SOCKET_MOTOR_STARTUP, "Not able to lock cable\nPlease call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_MISSINGPHASE, "Error in installation.\nPlease Check installation or call for support", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_TICPORT, "Error TIC port", EStatusIcon.STATUS_ICON_ERROR),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_CHARGING, "One moment please...\nYour charging session will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_CHARGING_OVERCURRENT, "One moment please...\nYour charging session will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_CHARGING_HF_SWITCHING, "Charging not started yet,\nto continue please reconnect cable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_CABLE_CONNECTED_TIMEOUT, "Charging not started yet\nto continue please reconnect cable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_TEMPERATURE_HIGH, "Inside temperature high.\nCharging will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_TEMPERATURE_LOW, "Inside temperature low. Charging will resume shortly.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_MESSAGE, "Charging not started yet,\nto continue please reconnect cable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_SOCKET_MOTOR, "Not able to lock cable.\nPlease reconnect cable.", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_ILLEGAL_MODE_3_PP, "Cable not supported\nPlease try connecting your cable again ", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_ILLEGAL_MODE_3_CP, "No communication with vehicle.\n Please check your charging cable", EStatusIcon.STATUS_ICON_WARNING),
			new DisplayObjectNew(EUserInterfaceError.UI_ERROR_TILT, "", EStatusIcon.STATUS_ICON_VALID)
		};
		StartUpdateTimer(250);
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		newDevice.UpdateCategories("states", "meter1", "MbusTCP", "temp");
		Version firmwareVersionNumber = iCULanDevice.FirmwareVersionNumber;
		m_hasUserInterfaceStates = firmwareVersionNumber >= new Version("4.4.0") && iCULanDevice.HasProperty(12688, 1);
		using (m_configPanel = AddConfigurationPanel(Title))
		{
			m_catStates = m_configPanel.AddCategory("States");
			m_catStates.Add(AddSmallHeader("General"));
			m_catStates.Add(AddReadOnlyText(8288, 0, "System uptime", UIPropertyStringType.DateTime));
			m_catStates.Add(AddReadOnlyText(8278, 0, "Number of bootups"));
			if (iCULanDevice.HasProperty(9536))
			{
				m_catStates.Add(AddReadOnlyCustomSelect(9536, 0, "Modbus TCP/IP Connection State", m_dicModbusTCPIPConnectionStates));
			}
			m_catStates.Add(AddSmallHeader("Socket 1 states"));
			if (!iCULanDevice.isAHP)
			{
				m_lblSocket1DisplayText = (UIPropertyString)m_catStates.Add(AddCustomText("Device state", "", "", null, 2, 2425241u));
			}
			if (iCULanDevice.HasProperty(8287))
			{
				m_catStates.Add(m_selOperationState1 = (UIPropertyManualSelect)AddManualSelect("Status", m_dicSocketStatus, "", null, 2121475u));
			}
			if (iCULanDevice.isAHP)
			{
				if (iCULanDevice.isDC)
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9473, 4, "Mode4 state", m_dicMode3States));
				}
				else
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9473, 4, "Mode3 state", m_dicMode3States));
				}
				m_catStates.Add(AddReadOnlyCustomSelect(9473, 1, "Main CSM state", m_dicAHWPMainStates));
				m_catStates.Add(AddReadOnlyCustomSelect(9473, 5, "Socket CC state", m_dicAHWPCCStates));
				if (!iCULanDevice.isDC)
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9473, 3, "Socket CPRO state", m_dicAHWPCPROStates));
				}
			}
			else
			{
				m_catStates.Add(AddReadOnlyCustomSelect(9473, 4, "Mode3 state", m_dicMode3States));
				m_catStates.Add(AddReadOnlyCustomSelect(9473, 1, "Main state", m_dicMainStates));
				m_catStates.Add(AddReadOnlyCustomSelect(9473, 2, "LED state", m_dicLEDStates));
				if (iCULanDevice.FirmwareVersionNumber < new Version(7, 3))
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9473, 3, "Power state", m_dicSocketStates));
				}
				else
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9473, 3, "Power state", m_dicSocketPowerStates));
				}
			}
			if (m_hasUserInterfaceStates)
			{
				m_catStates.Add(AddReadOnlyCustomSelect(12688, 1, "Display state", m_dicUserInterfaceStates));
			}
			if (newDevice.NumberOfSockets > 1)
			{
				m_catStates.Add(AddSmallHeader("Socket 2 states"));
				if (!iCULanDevice.isAHP)
				{
					m_lblSocket2DisplayText = (UIPropertyString)m_catStates.Add(AddCustomText("Device state", "", "", null, 2, 2425497u));
				}
				if (iCULanDevice.HasProperty(8287))
				{
					m_catStates.Add(m_selOperationState2 = (UIPropertyManualSelect)AddManualSelect("Status", m_dicSocketStatus, "", null, 2121477u));
				}
				if (iCULanDevice.isAHP)
				{
					if (iCULanDevice.isDC)
					{
						m_catStates.Add(AddReadOnlyCustomSelect(9474, 4, "Mode4 state", m_dicMode3States));
					}
					else
					{
						m_catStates.Add(AddReadOnlyCustomSelect(9474, 4, "Mode3 state", m_dicMode3States));
					}
					m_catStates.Add(AddReadOnlyCustomSelect(9474, 1, "Main CSM state", m_dicAHWPMainStates));
					m_catStates.Add(AddReadOnlyCustomSelect(9474, 5, "Socket CC state", m_dicAHWPCCStates));
					if (!iCULanDevice.isDC)
					{
						m_catStates.Add(AddReadOnlyCustomSelect(9474, 3, "Socket CPRO state", m_dicAHWPCPROStates));
					}
				}
				else
				{
					m_catStates.Add(AddReadOnlyCustomSelect(9474, 4, "Mode3 state", m_dicMode3States));
					m_catStates.Add(AddReadOnlyCustomSelect(9474, 1, "Main state", m_dicMainStates));
					m_catStates.Add(AddReadOnlyCustomSelect(9474, 2, "LED state", m_dicLEDStates));
					if (iCULanDevice.FirmwareVersionNumber < new Version(7, 3))
					{
						m_catStates.Add(AddReadOnlyCustomSelect(9474, 3, "Power state", m_dicSocketStates));
					}
					else
					{
						m_catStates.Add(AddReadOnlyCustomSelect(9474, 3, "Power state", m_dicSocketPowerStates));
					}
				}
				if (m_hasUserInterfaceStates)
				{
					m_catStates.Add(AddReadOnlyCustomSelect(12689, 1, "Display state", m_dicUserInterfaceStates));
				}
			}
			if (iCULanDevice.SupportsModem() && iCULanDevice.HasProperty(8351))
			{
				m_catStates.Add(AddSmallHeader("Mobile Network states"));
				m_catStates.Add(AddReadOnlyCustomSelect(8351, 0, "Mobile Technology", m_dicMobileNetworkTechnology));
			}
			if (iCULanDevice.HasProperty(8584))
			{
				m_catStates.Add(AddSmallHeader("Payments"));
				m_catStates.Add(AddReadOnlyCustomSelect(8584, 0, "Giro-e state", m_dicPaymentGiroEStates));
			}
			SetStatusMessages(1);
			if (m_currentDevice.NumberOfSockets > 1)
			{
				SetStatusMessages(2);
			}
			UpdateOperationState(newDevice);
			m_catCommunicationCar = m_configPanel.AddCategory("Communication car");
			m_catCommunicationCar.Add(AddSmallHeader("Socket 1"));
			AddMode3Signals(m_catCommunicationCar, 9489);
			if (newDevice.NumberOfSockets > 1)
			{
				m_catCommunicationCar.Add(AddSmallHeader("Socket 2"));
				AddMode3Signals(m_catCommunicationCar, 9490);
			}
			iCULanDevice.HasCentralMeter = false;
			iCULanDevice.HasSmartMeter = false;
			m_showLegacySmartMeterValues = false;
			if (newDevice.GetProperty(4331264u).Category != null && newDevice.NumberOfSockets > 1 && newDevice.GetPropertyInt(16919, 0) != -1)
			{
				iCULanDevice.HasCentralMeter = true;
			}
			ICUProperty property = newDevice.GetProperty(21015, 0);
			uint propertyUInt = iCULanDevice.GetPropertyUInt(8610, 0);
			if ((IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_Active, iCULanDevice.isAHP) || IWSFirmwareFeatures.IsFeatureUnlocked(firmwareVersionNumber, propertyUInt, IWSFirmwareFeatures.Features.LoadBalancing_SCN, iCULanDevice.isAHP)) && (newDevice.GetPropertyInt(8292, 0, 2) > 0 || newDevice.GetPropertyInt(9506, 1, 1) > 0))
			{
				if (property != null && property.Category != null)
				{
					if (newDevice.GetPropertyInt(21015, 0) != -1)
					{
						iCULanDevice.HasSmartMeter = true;
					}
				}
				else
				{
					m_showLegacySmartMeterValues = true;
					iCULanDevice.HasSmartMeter = true;
				}
			}
			m_catVoltage = m_configPanel.AddCategory("Voltages", "Voltage levels");
			m_catVoltage.Add(AddSmallHeader("Socket 1"));
			AddVoltages(m_catVoltage, 8737);
			if (newDevice.NumberOfSockets > 1)
			{
				m_catVoltage.Add(AddSmallHeader("Socket 2"));
				AddVoltages(m_catVoltage, 12833);
			}
			if (iCULanDevice.HasCentralMeter)
			{
				m_catVoltage.Add(AddSmallHeader("Central meter"));
				AddVoltages(m_catVoltage, 16929);
			}
			if (iCULanDevice.HasSmartMeter)
			{
				if (!m_showLegacySmartMeterValues)
				{
					m_catVoltage.Add(AddSmallHeader("Smart meter"));
					AddVoltages(m_catVoltage, 21025);
				}
				else
				{
					m_catVoltage.Add(AddSmallHeader("Smart meter"));
					AddVoltages(m_catVoltage, 16929);
				}
			}
			m_catCurrent = m_configPanel.AddCategory("Currents", "Current levels");
			m_catCurrent.Add(AddSmallHeader("Socket 1"));
			AddCurrents(m_catCurrent, 8737);
			if (newDevice.NumberOfSockets > 1)
			{
				m_catCurrent.Add(AddSmallHeader("Socket 2"));
				AddCurrents(m_catCurrent, 12833);
			}
			if (iCULanDevice.HasCentralMeter)
			{
				m_catCurrent.Add(AddSmallHeader("Central meter"));
				AddCurrents(m_catCurrent, 16929);
			}
			if (iCULanDevice.HasSmartMeter)
			{
				if (!m_showLegacySmartMeterValues)
				{
					m_catCurrent.Add(AddSmallHeader("Smart meter"));
					AddCurrents(m_catCurrent, 21025);
				}
				else
				{
					m_catCurrent.Add(AddSmallHeader("Smart meter"));
					AddCurrents(m_catCurrent, 16929);
				}
			}
			m_catNetQuality = m_configPanel.AddCategory("Net quality");
			m_catNetQuality.Add(AddSmallHeader("Socket 1"));
			AddNetQuality(m_catNetQuality, 8737);
			if (newDevice.NumberOfSockets > 1)
			{
				m_catNetQuality.Add(AddSmallHeader("Socket 2"));
				AddNetQuality(m_catNetQuality, 12833);
			}
			if (iCULanDevice.HasCentralMeter)
			{
				m_catNetQuality.Add(AddSmallHeader("Central meter"));
				AddNetQuality(m_catNetQuality, 16929);
			}
			if (iCULanDevice.HasSmartMeter)
			{
				if (!m_showLegacySmartMeterValues)
				{
					m_catNetQuality.Add(AddSmallHeader("Smart meter"));
					AddNetQuality(m_catNetQuality, 21025);
				}
				else
				{
					m_catNetQuality.Add(AddSmallHeader("Smart meter"));
					AddNetQuality(m_catNetQuality, 16929);
				}
			}
			m_catSensors = m_configPanel.AddCategory("Sensors");
			if (iCULanDevice.isDC)
			{
				m_catSensors.Add(AddSmallHeader("Frontchamber temperature sensors"));
				m_catSensors.Add(AddReadOnlyText(33568, 1, "P5: DC fuse ambient temp (°C)"));
				m_catSensors.Add(AddReadOnlyText(33568, 2, "P6: DC+ relay temp (°C)"));
				m_catSensors.Add(AddReadOnlyText(33568, 4, "P8: 12V auxiliary temp (°C)"));
				m_catSensors.Add(AddSmallHeader("Backchamber temperature sensors"));
				m_catSensors.Add(AddReadOnlyText(33568, 0, "P4: Exit temp (°C)"));
				m_catSensors.Add(AddReadOnlyText(33568, 3, "P7: Entrance temp (°C)"));
				m_catSensors.Add(AddSmallHeader("Other sensors"));
			}
			else
			{
				m_catSensors.Add(AddReadOnlyText(8705, 0, "Temperature (°C)"));
			}
			if (iCULanDevice.HasProperty(8777))
			{
				m_catSensors.Add(AddReadOnlyText(8777, 0, "Maximum Temperature (°C)"));
				m_catSensors.Add(AddReadOnlyText(8777, 1, "Minimum Temperature (°C)"));
			}
			m_txtXYZ = (UIPropertyString)m_catSensors.Add(AddCustomText("Tilt sensor (X,Y,Z)", "", "", null, 1, 2230016u));
			if (iCULanDevice.HasProperty(33553))
			{
				m_catFans = m_configPanel.AddCategory("Fans");
				m_catFans.Add(AddSmallHeader("Frontchamber"));
				m_catFans.Add(AddReadOnlyText(33553, 0, "Fan status 0"));
				m_catFans.Add(AddSmallHeader("Backchamber"));
				m_catFans.Add(AddReadOnlyText(33554, 0, "Fan status 0"));
				m_catFans.Add(AddReadOnlyText(33554, 1, "Fan status 1"));
				m_catFans.Add(AddReadOnlyText(33554, 2, "Fan status 2"));
			}
			m_catConnectivity = m_configPanel.AddCategory("Connectivity");
			m_catConnectivity.Add(AddSmallHeader("Back Office connectivity"));
			m_catConnectivity.Add(AddReadOnlyCustomSelect(13824, 1, "OCPP Boot notification state", m_dicOCPPBootStates));
			if (iCULanDevice.HasProperty(13648))
			{
				m_catConnectivity.Add(AddReadOnlySelect(13648, 0, 0, 0));
				m_catConnectivity.Add(AddReadOnlySelect(13651, 0, 0, 0));
			}
			if (m_currentDevice.HasWifiSupport)
			{
				m_catConnectivity.Add(AddSmallHeader("Wi-Fi"));
				m_catConnectivity.Add(AddReadOnlySelect(12942, 0, 0, 0));
				m_catConnectivity.Add(AddReadOnlySelect(12947, 0, 0, 0));
				m_catConnectivity.Add(AddReadOnlySelect(12940, 0, 0, 0));
				m_catConnectivity.Add(AddNumber(12941, 0));
				m_catConnectivity.Add(AddReadOnlySelect(12948, 0, 0, 0));
			}
		}
		return true;
	}

	protected void UpdateOperationState(ICUDevice newDevice)
	{
		if (m_currentDevice.GetProperty(8287, 0) != null)
		{
			int num = (int)newDevice.GetPropertyUInt64(8287, 0, 0uL);
			m_selOperationState1?.SetInitialValue(((num & 3) != 0) ? 1 : 0);
			m_selOperationState2?.SetInitialValue(((num & 5) != 0) ? 1 : 0);
		}
	}

	protected void AddMode3Signals(UIConfigCategory cat, ushort id)
	{
		cat.Add(AddReadOnlyText(id, 0, "CP voltage high (V)", UIPropertyStringType.Float2_0_01));
		cat.Add(AddReadOnlyText(id, 1, "CP voltage low (V)", UIPropertyStringType.Float2_0_01));
		cat.Add(AddReadOnlyText(id, 2, "PP resistance (Ω)", UIPropertyStringType.Float));
		cat.Add(AddReadOnlyText(id, 3, "PWM duty cycle (%)", UIPropertyStringType.PWM));
	}

	protected void AddVoltages(UIConfigCategory cat, ushort id)
	{
		cat.Add(AddReadOnlyText(id, 3, "Voltage L1N (V)"));
		cat.Add(AddReadOnlyText(id, 4, "Voltage L2N (V)"));
		cat.Add(AddReadOnlyText(id, 5, "Voltage L3N (V)"));
		cat.Add(AddReadOnlyText(id, 6, "Voltage L1L2 (V)"));
		cat.Add(AddReadOnlyText(id, 7, "Voltage L2L3 (V)"));
		cat.Add(AddReadOnlyText(id, 8, "Voltage L3L1 (V)"));
	}

	protected void AddCurrents(UIConfigCategory cat, ushort id)
	{
		cat.Add(AddReadOnlyText(id, 10, "Current L1 (A)"));
		cat.Add(AddReadOnlyText(id, 11, "Current L2 (A)"));
		cat.Add(AddReadOnlyText(id, 12, "Current L3 (A)"));
		cat.Add(AddReadOnlyText(id, 9, "Current N (A)"));
	}

	protected void AddNetQuality(UIConfigCategory cat, ushort id)
	{
		cat.Add(AddReadOnlyText(id, 18, "Frequency (Hz)"));
		cat.Add(AddReadOnlyText(id, 19, "Active Power L1 (kW)", UIPropertyStringType.Float_0_001));
		cat.Add(AddReadOnlyText(id, 20, "Active Power L2 (kW)", UIPropertyStringType.Float_0_001));
		cat.Add(AddReadOnlyText(id, 21, "Active Power L3 (kW)", UIPropertyStringType.Float_0_001));
		cat.Add(AddReadOnlyText(id, 22, "Active Power Total (kW)", UIPropertyStringType.Float_0_001));
		cat.Add(AddReadOnlyText(id, 14, "Cos φ L1", UIPropertyStringType.Float2));
		cat.Add(AddReadOnlyText(id, 15, "Cos φ L2", UIPropertyStringType.Float2));
		cat.Add(AddReadOnlyText(id, 16, "Cos φ L3", UIPropertyStringType.Float2));
		cat.Add(AddReadOnlyText(id, 17, "Cos φ Total", UIPropertyStringType.Float2));
	}

	protected override void OnUpdateTick()
	{
		if (m_currentDevice == null || !m_currentDevice.IsConnected || m_currentDevice.LastHttpStatusCode != HttpStatusCode.OK)
		{
			return;
		}
		ICULanDevice currentDevice = m_currentDevice;
		switch (m_updateSwitch++ % 5)
		{
		case 0u:
			if (m_currentDevice.NumberOfSockets > 1)
			{
				m_currentDevice.UpdateCategories("meter1", "meter2");
			}
			else
			{
				m_currentDevice.UpdateCategories("meter1");
			}
			break;
		case 1u:
			if (currentDevice.HasCentralMeter)
			{
				m_currentDevice.UpdateCategories("meter3");
			}
			break;
		case 2u:
			if (currentDevice.HasSmartMeter)
			{
				m_currentDevice.UpdateCategories("meter4");
			}
			break;
		case 3u:
			m_currentDevice.UpdateCategories("states", "generic2");
			Application.Invoke(() =>
			{
				SetStatusMessages(1);
				if (m_currentDevice.NumberOfSockets > 1)
				{
					SetStatusMessages(2);
				}
			});
			break;
		case 4u:
			m_currentDevice.UpdateProperties(2228480u, 2121472u, 2121728u, 2230016u, 2230272u, 2230528u, 2232320u, 2232576u, 2232832u);
			if (m_currentDevice.HasWifiSupport)
			{
				m_currentDevice.UpdateProperties(3312640u, 3312896u, 3313152u, 3314432u, 3314688u);
			}
			if (m_currentDevice.HasProperty(13648))
			{
				m_currentDevice.UpdateProperties(3493888u, 3494656u);
			}
			Application.Invoke(() =>
			{
				if (m_currentDevice.GetPropertyInt(8710, 0) < 2)
				{
					m_txtXYZ.CustomValue = "Tilt sensor disabled";
				}
				else
				{
					m_txtXYZ.CustomValue = $"{m_currentDevice.GetPropertyString(8711, 0, 0)}, {m_currentDevice.GetPropertyString(8712, 0, 0)}, {m_currentDevice.GetPropertyString(8713, 0, 0)}";
				}
				UpdateOperationState(m_currentDevice);
			});
			break;
		}
	}

	private void SetStatusMessages(int socketIndex)
	{
		ICULanDevice currentDevice = m_currentDevice;
		UIPropertyString uIPropertyString = ((socketIndex == 1) ? m_lblSocket1DisplayText : m_lblSocket2DisplayText);
		if (uIPropertyString == null)
		{
			return;
		}
		if (m_hasUserInterfaceStates)
		{
			int num = ((socketIndex == 1) ? 12688 : 12689);
			EUserInterfaceStates propertyInt = (EUserInterfaceStates)currentDevice.GetPropertyInt((ushort)num, 1);
			EUserInterfaceError errorNumber = ((propertyInt == EUserInterfaceStates.UI_STATE_ERROR) ? ((EUserInterfaceError)currentDevice.GetPropertyInt((ushort)num, 2)) : EUserInterfaceError.UI_ERROR_NONE);
			DisplayObjectNew displayObjectNew = TDisplayStatesMessagesNew.FirstOrDefault((DisplayObjectNew s) => s.ErrorCode == errorNumber);
			if (displayObjectNew == null)
			{
				displayObjectNew = new DisplayObjectNew(errorNumber, "Unknown Error/Warning state \nPlease call for support", EStatusIcon.STATUS_ICON_WARNING);
			}
			uIPropertyString.CustomValue = $"{(int)displayObjectNew.ErrorCode:000}: {displayObjectNew.Text}";
			uIPropertyString.SetToolTipIcon(displayObjectNew.Icon);
			return;
		}
		int num2 = ((socketIndex == 1) ? 9473 : 9474);
		if (currentDevice.HasProperty(num2, 1))
		{
			EMainStates socketState = (EMainStates)currentDevice.GetPropertyInt((ushort)num2, 1);
			DisplayObjectOld displayObjectOld = TDisplayStatesMessagesOld.FirstOrDefault((DisplayObjectOld s) => s.MainState == socketState);
			if (displayObjectOld == null)
			{
				displayObjectOld = new DisplayObjectOld(EMainStates.STATE_UNKNOWN, $"{socketState} \nUnknown Error/Warning state", EStatusIcon.STATUS_ICON_WARNING);
			}
			uIPropertyString.CustomValue = displayObjectOld.Text;
			uIPropertyString.SetToolTipIcon(displayObjectOld.Icon);
		}
	}

	public override void OnRevertChanges()
	{
		m_selOperationState1?.RevertChange();
		m_selOperationState2?.RevertChange();
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		if (m_currentDevice != null && ((m_selOperationState1 != null && m_selOperationState1.IsChanged) || (m_selOperationState2 != null && m_selOperationState2.IsChanged)))
		{
			int num = 0;
			int num2 = 0;
			if (m_selOperationState1 != null)
			{
				num = Convert.ToInt32(m_selOperationState1.GetValue());
			}
			if (m_selOperationState2 != null)
			{
				num2 = Convert.ToInt32(m_selOperationState2.GetValue());
			}
			int num3 = (num << 1) | (num2 << 2);
			ICUProperty property = m_currentDevice.GetProperty(8287, 0);
			if (property != null)
			{
				property.Value = num3;
				m_currentDevice.StoreProperties(property);
			}
		}
		m_selOperationState1?.CommitChange();
		m_selOperationState2?.CommitChange();
		return true;
	}
}
