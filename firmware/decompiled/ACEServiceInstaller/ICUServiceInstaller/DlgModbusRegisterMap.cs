using System;
using System.Globalization;
using System.IO;
using System.Linq;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

internal class DlgModbusRegisterMap : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgModbusRegisterMap>();

	private const string manuallyEnterMapping = "Manually enter mapping";

	protected ICULanDevice m_currentDevice;

	private readonly EMeterTypes m_energyMeterType;

	private readonly ComboBox comboBoxPresets;

	private readonly CheckBox chkHexadecimal;

	private readonly ModbusRegmapUIList ls_regmap = new ModbusRegmapUIList();

	private string m_meterName = string.Empty;

	private string m_bopresetName = string.Empty;

	public DlgModbusRegisterMap(ICULanDevice m_currentDevice, EMeterTypes m_energyMeterType)
	{
		this.m_currentDevice = m_currentDevice;
		this.m_energyMeterType = m_energyMeterType;
		Title = "Modbus register mapping configuration";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		if (m_currentDevice.HasProperty(8310))
		{
			string[] array = m_currentDevice.GetPropertyString(8310, 0, 0).Split(new char[1] { ',' });
			m_bopresetName = array[0];
			if (array.Length > 1)
			{
				m_meterName = array[1];
			}
		}
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinHeight = 80.0,
			MinWidth = 425.0,
			Margin = 8.0
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		Label widget2 = new Label("Select Modbus custom preset type");
		comboBoxPresets = new ComboBox
		{
			SelectedIndex = 0
		};
		PopulateComboBoxPresets(m_energyMeterType);
		if (!string.IsNullOrEmpty(m_meterName))
		{
			bool flag = false;
			for (int i = 1; i < comboBoxPresets.Items.Count; i++)
			{
				comboBoxPresets.SelectedIndex = i;
				if (comboBoxPresets.SelectedItem is FileInfo fileInfo && Path.GetFileNameWithoutExtension(fileInfo.FullName).Equals(m_meterName, StringComparison.OrdinalIgnoreCase))
				{
					flag = true;
					break;
				}
			}
			if (!flag)
			{
				comboBoxPresets.SelectedIndex = 0;
			}
		}
		comboBoxPresets.SelectionChanged += ComboBoxPresets_SelectionChanged;
		table.Add(widget2, 0, 0, 1, 1, hexpand: true);
		table.Add(comboBoxPresets, 0, 1, 1, 1, hexpand: true);
		Table table2 = new Table
		{
			BackgroundColor = Colors.White,
			Margin = 8.0
		};
		table.Add(table2, 0, 3, 1, 1, hexpand: true);
		chkHexadecimal = new CheckBox("Show and enter register numbers in hexadecimal");
		chkHexadecimal.Toggled += ChkHexadecimal_Toggled;
		table.Add(chkHexadecimal, 0, 4, 1, 1, hexpand: true);
		ls_regmap.Add(EnergyMeterMeasurand.CURRENT_L1, "Current L1");
		ls_regmap.Add(EnergyMeterMeasurand.CURRENT_L2, "Current L2", optional: true);
		ls_regmap.Add(EnergyMeterMeasurand.CURRENT_L3, "Current L3", optional: true);
		ls_regmap.Add(EnergyMeterMeasurand.CURRENT_N, "Current N", optional: true);
		ls_regmap.Add(EnergyMeterMeasurand.POWER_REAL_L1, "Real Power L1");
		ls_regmap.Add(EnergyMeterMeasurand.POWER_REAL_L2, "Real Power L2", optional: true);
		ls_regmap.Add(EnergyMeterMeasurand.POWER_REAL_L3, "Real Power L3", optional: true);
		for (int j = 0; j < ls_regmap.Count; j++)
		{
			int top = j + 1;
			table2.Add(ls_regmap[j].Name, 0, top, 1, 1, hexpand: true);
			table2.Add(ls_regmap[j].Register, 1, top, 1, 1, hexpand: true);
			if (ls_regmap[j].Measurand == EnergyMeterMeasurand.CURRENT_L1 || ls_regmap[j].Measurand == EnergyMeterMeasurand.POWER_REAL_L1)
			{
				table2.Add(ls_regmap[j].DataType, 2, top, 1, 1, hexpand: true);
				table2.Add(ls_regmap[j].Scale, 3, top, 1, 1, hexpand: true);
			}
		}
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		DialogButton dialogButton = new DialogButton(Command.Save);
		dialogButton.Clicked += DbtnSave_Clicked;
		Buttons.Add(dialogButton);
		Buttons.Add(new DialogButton(Command.Close));
		UpdateEditValues();
		EvalulateEnableEditValues();
	}

	private void ChkHexadecimal_Toggled(object sender, EventArgs e)
	{
		foreach (ModbusRegmapUI item in ls_regmap)
		{
			if (m_currentDevice.ModbusTcpIpRegmap.GetEntry(item.Measurand) != null)
			{
				ushort result = 0;
				if (chkHexadecimal.State == CheckBoxState.On)
				{
					ushort.TryParse(item.Register.Text, out result);
				}
				else
				{
					ushort.TryParse(item.Register.Text, NumberStyles.HexNumber, null, out result);
				}
				item.Register.Text = ((chkHexadecimal.State == CheckBoxState.On) ? result.ToString("X4") : result.ToString());
			}
		}
	}

	private void DbtnSave_Clicked(object sender, EventArgs e)
	{
		if (IsManuallyEnterMapping())
		{
			m_currentDevice.ModbusTcpIpRegmap.RegmapData.Regmap.Clear();
			if (!WriteEditValuesToMemory())
			{
				MessageDialog.ShowError(this, "Invalid input");
				return;
			}
		}
		else if (!LoadPresetFileToMemory())
		{
			string text = "Details: ";
			foreach (string loadFromJSONError in m_currentDevice.ModbusTcpIpRegmap.LoadFromJSONErrorList)
			{
				text = text + loadFromJSONError + Environment.NewLine;
			}
			MessageDialog.ShowError(this, "Reading the preset file failed.", text);
			return;
		}
		if (m_currentDevice.ModbusTcpIpRegmap.WriteToDevice(m_energyMeterType))
		{
			if (m_currentDevice.HasProperty(8310))
			{
				string text2 = m_bopresetName;
				if (comboBoxPresets.SelectedItem is FileInfo)
				{
					text2 = text2 + "," + Path.GetFileNameWithoutExtension((comboBoxPresets.SelectedItem as FileInfo).FullName);
				}
				m_currentDevice.storeProperty(8310, 0, text2);
			}
			MessageDialog.ShowMessage(this, "Load preset successful.");
			return;
		}
		string text3 = "Details: ";
		foreach (string writeToDeviceError in m_currentDevice.ModbusTcpIpRegmap.WriteToDeviceErrorList)
		{
			text3 = text3 + writeToDeviceError + Environment.NewLine;
		}
		MessageDialog.ShowError(this, "Couldn't write data to the charging station.", text3);
	}

	private void ComboBoxPresets_SelectionChanged(object sender, EventArgs e)
	{
		UpdateEditValues();
		EvalulateEnableEditValues();
	}

	private void PopulateComboBoxPresets(EMeterTypes energyMeterType)
	{
		comboBoxPresets.Items.Clear();
		comboBoxPresets.Items.Add("Manually enter mapping", "<Manually enter mapping>");
		try
		{
			string path = ((energyMeterType == EMeterTypes.ENERGYMETER_TCPIP_SMART) ? AppProperties.LocalTCPPresetsFolder : AppProperties.LocalRTUPresetsFolder);
			if (!Directory.Exists(path))
			{
				Directory.CreateDirectory(path);
			}
			foreach (FileInfo item in (from a in (from a in Directory.GetFiles(path)
					select new FileInfo(a)).ToList()
				where a.Extension.ToLowerInvariant() == ".json"
				select a).ToList())
			{
				comboBoxPresets.Items.Add(item, Path.GetFileNameWithoutExtension(item.FullName));
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
	}

	private bool LoadPresetFileToMemory()
	{
		if (!(comboBoxPresets.SelectedItem is FileInfo))
		{
			throw new Exception("Can't load a file that isn't a FileInfo type");
		}
		string json = File.ReadAllText((comboBoxPresets.SelectedItem as FileInfo).FullName);
		m_currentDevice.ModbusTcpIpRegmap.LoadFromJSON(json);
		return !m_currentDevice.ModbusTcpIpRegmap.LoadFromJSONErrorList.Any();
	}

	private bool IsManuallyEnterMapping()
	{
		if (comboBoxPresets != null)
		{
			return string.Compare("Manually enter mapping", comboBoxPresets.SelectedItem as string) == 0;
		}
		return true;
	}

	private void EvalulateEnableEditValues()
	{
		SetEditValuesSensitive(IsManuallyEnterMapping());
	}

	private void SetEditValuesSensitive(bool isSensitive)
	{
		ls_regmap.ForEach((ModbusRegmapUI a) =>
		{
			a.Sensitive = isSensitive;
		});
	}

	private void UpdateEditValues()
	{
		ls_regmap.ForEach((ModbusRegmapUI a) =>
		{
			a.Clear();
		});
		if (IsManuallyEnterMapping())
		{
			if (!m_currentDevice.ModbusTcpIpRegmap.LoadFromDevice(m_energyMeterType))
			{
				return;
			}
		}
		else if (!LoadPresetFileToMemory())
		{
			return;
		}
		foreach (ModbusRegmapUI item in ls_regmap)
		{
			ModbusRegmapEntry entry = m_currentDevice.ModbusTcpIpRegmap.GetEntry(item.Measurand);
			if (entry != null)
			{
				item.Register.Text = ((chkHexadecimal.State == CheckBoxState.On) ? entry.RegNum.ToString("X4") : entry.RegNum.ToString());
				item.DataType.SelectedItem = entry.DataType;
				item.Scale.SelectedItem = (int)entry.ScaleE;
			}
		}
	}

	private bool WriteEditValuesToMemory()
	{
		try
		{
			foreach (ModbusRegmapUI item in ls_regmap)
			{
				ushort result = 0;
				string text = item.Register.Text;
				if (item.Optional && string.IsNullOrEmpty(text))
				{
					continue;
				}
				if (chkHexadecimal.State == CheckBoxState.On)
				{
					if (!ushort.TryParse(item.Register.Text, NumberStyles.HexNumber, null, out result))
					{
						return false;
					}
				}
				else if (!ushort.TryParse(item.Register.Text, out result))
				{
					return false;
				}
				WriteOnePhaseEditValuesToMemory(item.Measurand, result, ls_regmap.GetDataType(item.Measurand), ls_regmap.GetScaleFactor(item.Measurand));
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return false;
		}
		return true;
	}

	private void WriteOnePhaseEditValuesToMemory(EnergyMeterMeasurand key, ushort regNum, ModbusDataType dataType, sbyte ScaleE)
	{
		ModbusRegmapEntry entry = m_currentDevice.ModbusTcpIpRegmap.GetEntry(key);
		if (entry != null)
		{
			entry.RegNum = regNum;
			entry.DataType = dataType;
			entry.ScaleE = ScaleE;
		}
		else
		{
			m_currentDevice.ModbusTcpIpRegmap.RegmapData.Regmap.Add(new ModbusRegmapEntry
			{
				Key = key,
				RegNum = regNum,
				DataType = dataType,
				ScaleE = ScaleE
			});
		}
	}
}
