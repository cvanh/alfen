using System.Drawing;

namespace SimplePaletteQuantizer.Helpers;

public class QuantizationHelper
{
	private const int Alpha = -16777216;

	private static readonly Color BackgroundColor;

	private static readonly double[] Factors;

	static QuantizationHelper()
	{
		BackgroundColor = SystemColors.Control;
		Factors = PrecalculateFactors();
	}

	private static double[] PrecalculateFactors()
	{
		double[] array = new double[256];
		for (int i = 0; i < 256; i++)
		{
			array[i] = (double)i / 255.0;
		}
		return array;
	}

	internal static Color ConvertAlpha(Color color)
	{
		int argb;
		return ConvertAlpha(color, out argb);
	}

	internal static Color ConvertAlpha(Color color, out int argb)
	{
		Color result = color;
		if (color.A < byte.MaxValue)
		{
			double num = Factors[color.A];
			double num2 = Factors[255 - color.A];
			double num3 = (double)(int)color.R * num;
			Color backgroundColor = BackgroundColor;
			int num4 = (int)(num3 + (double)(int)backgroundColor.R * num2);
			double num5 = (double)(int)color.G * num;
			backgroundColor = BackgroundColor;
			int num6 = (int)(num5 + (double)(int)backgroundColor.G * num2);
			double num7 = (double)(int)color.B * num;
			backgroundColor = BackgroundColor;
			int num8 = (int)(num7 + (double)(int)backgroundColor.B * num2);
			argb = (num4 << 16) | (num6 << 8) | num8;
			Color.FromArgb(num4, num6, num8);
			result = Color.FromArgb(-16777216 | argb);
		}
		else
		{
			argb = (color.R << 16) | (color.G << 8) | color.B;
		}
		return result;
	}
}
