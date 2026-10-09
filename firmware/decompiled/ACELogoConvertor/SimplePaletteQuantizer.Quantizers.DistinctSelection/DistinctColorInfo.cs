using System;
using System.Drawing;

namespace SimplePaletteQuantizer.Quantizers.DistinctSelection;

public class DistinctColorInfo
{
	private const int Factor = 5000000;

	public int Color { get; private set; }

	public int Count { get; private set; }

	public int Hue { get; private set; }

	public int Saturation { get; private set; }

	public int Brightness { get; private set; }

	public DistinctColorInfo(Color color)
	{
		Color = color.ToArgb();
		Count = 1;
		Hue = Convert.ToInt32(color.GetHue() * 5000000f);
		Saturation = Convert.ToInt32(color.GetSaturation() * 5000000f);
		Brightness = Convert.ToInt32(color.GetBrightness() * 5000000f);
	}

	public DistinctColorInfo IncreaseCount()
	{
		Count++;
		return this;
	}
}
