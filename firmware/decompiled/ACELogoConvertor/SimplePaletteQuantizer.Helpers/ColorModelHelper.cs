using System;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.ColorCaches.Common;

namespace SimplePaletteQuantizer.Helpers;

public class ColorModelHelper
{
	private const int X = 0;

	private const int Y = 1;

	private const int Z = 2;

	private const float Epsilon = 1E-05f;

	private const float OneThird = 1f / 3f;

	private const float TwoThirds = 2f / 3f;

	public const double HueFactor = 1.411764705882353;

	private static readonly float[] XYZWhite = new float[3] { 95.05f, 100f, 108.9f };

	private static int GetColorComponent(float v1, float v2, float hue)
	{
		if (hue < 0f)
		{
			hue++;
		}
		if (hue > 1f)
		{
			hue--;
		}
		float num;
		if (6f * hue < 1f)
		{
			num = v1 + (v2 - v1) * 6f * hue;
		}
		else if (2f * hue < 1f)
		{
			num = v2;
		}
		else
		{
			num = ((!(3f * hue < 2f)) ? v1 : (v1 + (v2 - v1) * (2f / 3f - hue) * 6f));
		}
		return Convert.ToInt32(255f * num);
	}

	public static Color HSBtoRGB(float hue, float saturation, float brightness)
	{
		int num = 0;
		int num2 = 0;
		int num3 = 0;
		if (brightness > 0f)
		{
			if (Math.Abs(saturation - 0f) < 1E-05f)
			{
				num = (num2 = (num3 = Convert.ToInt32(255f * brightness)));
			}
			else
			{
				float num4 = hue / 360f;
				float num5 = ((brightness < 0.5f) ? (brightness * (1f + saturation)) : (brightness + saturation - brightness * saturation));
				float v = 2f * brightness - num5;
				num = GetColorComponent(v, num5, num4 + 1f / 3f);
				num2 = GetColorComponent(v, num5, num4);
				num3 = GetColorComponent(v, num5, num4 - 1f / 3f);
			}
		}
		return Color.FromArgb(-16777216 | (num << 16) | (num2 << 8) | num3);
	}

	public static void RGBtoLab(int red, int green, int blue, out float l, out float a, out float b)
	{
		RGBtoXYZ(red, green, blue, out var x, out var y, out var z);
		XYZtoLab(x, y, z, out l, out a, out b);
	}

	public static void RGBtoXYZ(int red, int green, int blue, out float x, out float y, out float z)
	{
		double num = (double)red / 255.0;
		double num2 = (double)green / 255.0;
		double num3 = (double)blue / 255.0;
		double num4 = ((num > 0.04045) ? Math.Pow((num + 0.055) / 1.055, 2.2) : (num / 12.92));
		double num5 = ((num2 > 0.04045) ? Math.Pow((num2 + 0.055) / 1.055, 2.2) : (num2 / 12.92));
		double num6 = ((num3 > 0.04045) ? Math.Pow((num3 + 0.055) / 1.055, 2.2) : (num3 / 12.92));
		x = Convert.ToSingle(num4 * 0.4124 + num5 * 0.3576 + num6 * 0.1805);
		y = Convert.ToSingle(num4 * 0.2126 + num5 * 0.7152 + num6 * 0.0722);
		z = Convert.ToSingle(num4 * 0.0193 + num5 * 0.1192 + num6 * 0.9505);
	}

	private static float GetXYZValue(float value)
	{
		if (!(value > 0.008856f))
		{
			return 7.787f * value + 0.13793103f;
		}
		return (float)Math.Pow(value, 0.3333333432674408);
	}

	public static void XYZtoLab(float x, float y, float z, out float l, out float a, out float b)
	{
		l = 116f * GetXYZValue(y / XYZWhite[1]) - 16f;
		a = 500f * (GetXYZValue(x / XYZWhite[0]) - GetXYZValue(y / XYZWhite[1]));
		b = 200f * (GetXYZValue(y / XYZWhite[1]) - GetXYZValue(z / XYZWhite[2]));
	}

	public static long GetColorEuclideanDistance(ColorModel colorModel, Color requestedColor, Color realColor)
	{
		GetColorComponents(colorModel, requestedColor, realColor, out var componentA, out var componentB, out var componentC);
		return (long)(componentA * componentA + componentB * componentB + componentC * componentC);
	}

	public static int GetEuclideanDistance(Color color, ColorModel colorModel, IList<Color> palette)
	{
		long num = long.MaxValue;
		int result = 0;
		for (int i = 0; i < palette.Count; i++)
		{
			Color realColor = palette[i];
			long colorEuclideanDistance = GetColorEuclideanDistance(colorModel, color, realColor);
			if (colorEuclideanDistance == 0L)
			{
				result = i;
				break;
			}
			if (colorEuclideanDistance < num)
			{
				num = colorEuclideanDistance;
				result = i;
			}
		}
		return result;
	}

	public static int GetComponentA(ColorModel colorModel, Color color)
	{
		int result = 0;
		switch (colorModel)
		{
		case ColorModel.RedGreenBlue:
			result = color.R;
			break;
		case ColorModel.HueSaturationBrightness:
			result = Convert.ToInt32((double)color.GetHue() / 1.411764705882353);
			break;
		case ColorModel.LabColorSpace:
		{
			RGBtoLab(color.R, color.G, color.B, out var l, out var _, out var _);
			result = Convert.ToInt32(l * 255f);
			break;
		}
		}
		return result;
	}

	public static int GetComponentB(ColorModel colorModel, Color color)
	{
		int result = 0;
		switch (colorModel)
		{
		case ColorModel.RedGreenBlue:
			result = color.G;
			break;
		case ColorModel.HueSaturationBrightness:
			result = Convert.ToInt32(color.GetSaturation() * 255f);
			break;
		case ColorModel.LabColorSpace:
		{
			RGBtoLab(color.R, color.G, color.B, out var _, out var a, out var _);
			result = Convert.ToInt32(a * 255f);
			break;
		}
		}
		return result;
	}

	public static int GetComponentC(ColorModel colorModel, Color color)
	{
		int result = 0;
		switch (colorModel)
		{
		case ColorModel.RedGreenBlue:
			result = color.B;
			break;
		case ColorModel.HueSaturationBrightness:
			result = Convert.ToInt32(color.GetBrightness() * 255f);
			break;
		case ColorModel.LabColorSpace:
		{
			RGBtoLab(color.R, color.G, color.B, out var _, out var _, out var b);
			result = Convert.ToInt32(b * 255f);
			break;
		}
		}
		return result;
	}

	public static void GetColorComponents(ColorModel colorModel, Color color, out float componentA, out float componentB, out float componentC)
	{
		componentA = 0f;
		componentB = 0f;
		componentC = 0f;
		switch (colorModel)
		{
		case ColorModel.RedGreenBlue:
			componentA = (int)color.R;
			componentB = (int)color.G;
			componentC = (int)color.B;
			break;
		case ColorModel.HueSaturationBrightness:
			componentA = color.GetHue();
			componentB = color.GetSaturation();
			componentC = color.GetBrightness();
			break;
		case ColorModel.LabColorSpace:
			RGBtoLab(color.R, color.G, color.B, out componentA, out componentB, out componentC);
			break;
		case ColorModel.XYZ:
			RGBtoXYZ(color.R, color.G, color.B, out componentA, out componentB, out componentC);
			break;
		}
	}

	public static void GetColorComponents(ColorModel colorModel, Color color, Color targetColor, out float componentA, out float componentB, out float componentC)
	{
		componentA = 0f;
		componentB = 0f;
		componentC = 0f;
		switch (colorModel)
		{
		case ColorModel.RedGreenBlue:
			componentA = color.R - targetColor.R;
			componentB = color.G - targetColor.G;
			componentC = color.B - targetColor.B;
			break;
		case ColorModel.HueSaturationBrightness:
			componentA = color.GetHue() - targetColor.GetHue();
			componentB = color.GetSaturation() - targetColor.GetSaturation();
			componentC = color.GetBrightness() - targetColor.GetBrightness();
			break;
		case ColorModel.LabColorSpace:
		{
			RGBtoLab(color.R, color.G, color.B, out var l, out var a, out var b);
			RGBtoLab(targetColor.R, targetColor.G, targetColor.B, out var l2, out var a2, out var b2);
			componentA = l - l2;
			componentB = a - a2;
			componentC = b - b2;
			break;
		}
		case ColorModel.XYZ:
		{
			RGBtoXYZ(color.R, color.G, color.B, out var x, out var y, out var z);
			RGBtoXYZ(targetColor.R, targetColor.G, targetColor.B, out var x2, out var y2, out var z2);
			componentA = x - x2;
			componentB = y - y2;
			componentC = z - z2;
			break;
		}
		}
	}
}
