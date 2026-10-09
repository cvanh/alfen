using System.Diagnostics;
using ICUNetwork;
using Xwt;

namespace ICUServiceInstaller;

[DebuggerDisplay("{DebuggerDisplay,nq}")]
public class ModbusRegmapUI
{
	public EnergyMeterMeasurand Measurand { get; private set; }

	public Label Name { get; private set; }

	public TextEntry Register { get; private set; }

	public ComboBox DataType { get; private set; }

	public ComboBox Scale { get; private set; }

	public bool Optional { get; set; }

	private string DebuggerDisplay => string.Format("{0} Optional:{1} DT:{2} S:{3}", new object[4]
	{
		Measurand.ToString(),
		Optional,
		DataType.SelectedItem,
		Scale.SelectedItem
	});

	public bool Sensitive
	{
		get
		{
			return Register.Sensitive;
		}
		set
		{
			Register.Sensitive = value;
			DataType.Sensitive = value;
			Scale.Sensitive = value;
		}
	}

	public ModbusRegmapUI(EnergyMeterMeasurand measurand, string labelName, bool optional)
	{
		Measurand = measurand;
		Name = new Label(labelName);
		Register = new TextEntry();
		DataType = new ComboBox();
		Scale = new ComboBox();
		Optional = optional;
	}

	public void Clear()
	{
		Register.Text = "";
		DataType.SelectedIndex = -1;
		Scale.SelectedIndex = -1;
	}
}
