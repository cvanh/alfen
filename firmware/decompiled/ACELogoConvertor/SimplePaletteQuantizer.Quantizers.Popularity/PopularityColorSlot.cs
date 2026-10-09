using System.Drawing;

namespace SimplePaletteQuantizer.Quantizers.Popularity;

internal class PopularityColorSlot
{
	private int red;

	private int green;

	private int blue;

	public int PixelCount { get; private set; }

	public PopularityColorSlot(Color color)
	{
		AddValue(color);
	}

	public PopularityColorSlot AddValue(Color color)
	{
		red += color.R;
		green += color.G;
		blue += color.B;
		PixelCount++;
		return this;
	}

	public Color GetAverage()
	{
		int num = red / PixelCount;
		int num2 = green / PixelCount;
		int num3 = blue / PixelCount;
		if (num < 0)
		{
			num = 0;
		}
		if (num > 255)
		{
			num = 255;
		}
		if (num2 < 0)
		{
			num2 = 0;
		}
		if (num2 > 255)
		{
			num2 = 255;
		}
		if (num3 < 0)
		{
			num3 = 0;
		}
		if (num3 > 255)
		{
			num3 = 255;
		}
		return Color.FromArgb(255, num, num2, num3);
	}
}
