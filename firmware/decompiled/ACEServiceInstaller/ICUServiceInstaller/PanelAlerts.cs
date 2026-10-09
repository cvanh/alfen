using System;
using System.Collections.Generic;
using System.Net;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelAlerts : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelAlerts>();

	protected UIConfigurationPanel m_configPanel;

	protected UIConfigCategory m_catTemperature;

	protected UIConfigCategory m_catTiltSensor;

	protected UIConfigCategory m_catRCD;

	protected UIConfigCategory m_catGrid;

	protected UIConfigCategory m_catTamper;

	protected UIPropertyString m_txtSetXYZ;

	protected UIPropertyNumber m_numTemperatureLogInterval;

	protected UIPropertyNumber m_numTemperatureCheckInterval;

	protected UIPropertyNumber m_numTiltSensorLogInterval;

	protected Button m_btnCalibrateAcceleroMeter;

	protected Dictionary<string, string> m_dicTamperStates = new Dictionary<string, string>();

	public PanelAlerts(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Alerts";
		Tooltip = "Alerts";
		IconName = "alert.png";
		StartUpdateTimer(500);
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		newDevice.UpdateCategories("temp", "accelero");
		m_dicTamperStates.Clear();
		m_dicTamperStates.Add(0.ToString(), "Unknown");
		m_dicTamperStates.Add(1.ToString(), "Not tampered");
		m_dicTamperStates.Add(2.ToString(), "Tamper active");
		m_dicTamperStates.Add(3.ToString(), "Tampered");
		m_dicTamperStates.Add(4.ToString(), "Tampered acknowledged");
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			m_catTemperature = m_configPanel.AddCategory("Temperature");
			m_catTemperature.Add(AddCustomNumber(8707, 0, 0, "Alarm value high (°C)"));
			m_catTemperature.Add(AddCustomNumber(8706, 0, 0, "Alarm value low (°C)"));
			m_numTemperatureCheckInterval = (UIPropertyNumber)m_catTemperature.Add(AddCustomNumber(8708, 0, 0, "Check interval (s)"));
			m_numTemperatureCheckInterval.SetValueMinMax(0.0, 32767.0);
			m_numTemperatureLogInterval = (UIPropertyNumber)m_catTemperature.Add(AddCustomNumber(8709, 0, 0, "Log interval (s)"));
			m_numTemperatureLogInterval.SetValueMinMax(300.0, 32767.0);
			m_catTiltSensor = m_configPanel.AddCategory("Tilt sensor");
			m_catTiltSensor.Add(AddSelect(8710, 0, 0, 0));
			m_txtSetXYZ = (UIPropertyString)m_catTiltSensor.Add(AddCustomText("Setpoint X,Y,Z values", "", "", null, 1, 2232320u));
			m_catTiltSensor.Add(AddCustomNumber(8723, 0, 0, "Margin X"));
			m_catTiltSensor.Add(AddCustomNumber(8724, 0, 0, "Margin Y"));
			m_catTiltSensor.Add(AddCustomNumber(8725, 0, 0, "Margin Z"));
			m_numTiltSensorLogInterval = (UIPropertyNumber)m_catTiltSensor.Add(AddCustomNumber(8726, 0, 0, "Log interval (s)"));
			m_numTiltSensorLogInterval.SetValueMinMax(0.0, 3599.0);
			m_catTiltSensor.AddWidget(AddButtonBox(fExpandVert: true, fExpandHor: false, forceNewInstace: true));
			m_catTiltSensor.AddWidget(m_btnCalibrateAcceleroMeter = AddCustomButton("Calibrate Tilt Sensor", "Calibrate the tilt sensor", OnCalibrateClicked, true));
			m_btnCalibrateAcceleroMeter.Sensitive = false;
			m_catRCD = m_configPanel.AddCategory("RCD", "Residual current device");
			if ((newDevice as ICULanDevice).FirmwareVersionNumber >= new Version("4.2.0"))
			{
				m_catRCD.Add(AddSelect(8304, 0, 0, 0));
			}
			else
			{
				m_catRCD.Add(AddCustomNumber(8577, 0, 0, "Delay time (s)"));
			}
			if (!string.IsNullOrEmpty(newDevice.GetProperty(2207744u).Category))
			{
				m_catGrid = m_configPanel.AddCategory("Grid", "Grid");
				m_catGrid.Add(AddCheckBox(8624, 0, 1));
			}
			if ((newDevice as ICULanDevice).HasProperty(8784))
			{
				m_catTamper = m_configPanel.AddCategory("Tamper");
				UIPropertyBase uIPropertyBase = AddCustomSelect(8784, 0, "Tamper sensor", m_dicTamperStates, 0, null, 2248704u);
				uIPropertyBase.ForceReadonly(fForce: true);
				m_catTamper.Add(uIPropertyBase);
			}
		}
		return true;
	}

	protected override void OnUpdateTick()
	{
		if (m_currentDevice != null && m_currentDevice.IsConnected && m_currentDevice.LastHttpStatusCode == HttpStatusCode.OK)
		{
			m_currentDevice.UpdateProperties(2232320u, 2232576u, 2232832u);
			Application.Invoke(() =>
			{
				m_txtSetXYZ.CustomValue = "X: " + m_currentDevice.GetPropertyString(8720, 0, 0) + ", Y: " + m_currentDevice.GetPropertyString(8721, 0, 0) + ", Z: " + m_currentDevice.GetPropertyString(8722, 0, 0);
			});
		}
	}

	protected override void OnChangeProperty()
	{
		base.OnChangeProperty();
		if (m_currentDevice != null && m_btnCalibrateAcceleroMeter != null)
		{
			m_btnCalibrateAcceleroMeter.Sensitive = m_currentDevice.GetPropertyInt(8710, 0) > 0;
		}
	}

	private void OnCalibrateClicked(object sender, EventArgs e)
	{
		ICUProperty property = m_currentDevice.GetProperty(8720, 0);
		ICUProperty property2 = m_currentDevice.GetProperty(8721, 0);
		ICUProperty property3 = m_currentDevice.GetProperty(8722, 0);
		property.Value = m_currentDevice.GetProperty(8711, 0).Value;
		property2.Value = m_currentDevice.GetProperty(8712, 0).Value;
		property3.Value = m_currentDevice.GetProperty(8713, 0).Value;
		m_currentDevice.StoreProperties(property, property2, property3);
		Logger.AddChargerContext(m_currentDevice).Information("Tilt sensor calibrated");
	}
}
