using Xwt;

namespace ICUServiceInstaller;

internal class ModbusMeasurandUIList
{
	public byte SubId { get; set; }

	public Label Label { get; set; }

	public string Unit { get; set; }

	public int Decimals { get; set; }

	public int Divider { get; set; }

	public ModbusMeasurandUIList(byte subId, string label, string unit, int decimals, int divider)
	{
		SubId = subId;
		Label = new Label(label);
		Unit = unit;
		Decimals = decimals;
		Divider = divider;
	}
}
