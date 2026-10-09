using System;
using System.Collections.Generic;
using System.Linq;
using ICUNetwork;
using Xwt;

namespace ICUServiceInstaller;

public class ModbusRegmapUIList : List<ModbusRegmapUI>
{
	public sbyte GetScaleFactor(EnergyMeterMeasurand measurand)
	{
		if ((uint)(measurand - 6) <= 3u)
		{
			return Convert.ToSByte(Find((ModbusRegmapUI x) => x.Measurand == EnergyMeterMeasurand.CURRENT_L1).Scale.SelectedItem);
		}
		return Convert.ToSByte(Find((ModbusRegmapUI x) => x.Measurand == EnergyMeterMeasurand.POWER_REAL_L1).Scale.SelectedItem);
	}

	public ModbusDataType GetDataType(EnergyMeterMeasurand measurand)
	{
		if ((uint)(measurand - 6) <= 3u)
		{
			return (ModbusDataType)Find((ModbusRegmapUI x) => x.Measurand == EnergyMeterMeasurand.CURRENT_L1).DataType.SelectedItem;
		}
		return (ModbusDataType)Find((ModbusRegmapUI x) => x.Measurand == EnergyMeterMeasurand.POWER_REAL_L1).DataType.SelectedItem;
	}

	public void Add(EnergyMeterMeasurand measurand, string labelName, bool optional = false)
	{
		ModbusRegmapUI modbusRegmapUI = new ModbusRegmapUI(measurand, labelName, optional);
		PopulateDataTypeComboBox(modbusRegmapUI.DataType);
		PopulateScaleComboBox(modbusRegmapUI.Scale);
		Add(modbusRegmapUI);
	}

	private void PopulateDataTypeComboBox(ComboBox comboBox)
	{
		foreach (ModbusDataType item in Enum.GetValues(typeof(ModbusDataType)).Cast<ModbusDataType>().ToList())
		{
			comboBox.Items.Add(item, item.ToString());
		}
	}

	private void PopulateScaleComboBox(ComboBox comboBox)
	{
		comboBox.Items.Add(-4, "x 0.0001");
		comboBox.Items.Add(-3, "x 0.001");
		comboBox.Items.Add(-2, "x 0.01");
		comboBox.Items.Add(-1, "x 0.1");
		comboBox.Items.Add(0, "x 1");
		comboBox.Items.Add(1, "x 10");
		comboBox.Items.Add(2, "x 100");
		comboBox.Items.Add(3, "x 1000");
		comboBox.Items.Add(4, "x 10000");
	}
}
