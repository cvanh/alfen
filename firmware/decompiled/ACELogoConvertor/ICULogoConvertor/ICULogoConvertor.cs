using System;
using System.Drawing;
using System.Drawing.Imaging;

namespace ICULogoConvertor;

public class ICULogoConvertor
{
	public static Image ConvertImage(Image inputImage, Size desiredSize, int maxColors)
	{
		double val = (double)inputImage.Width / (double)desiredSize.Width;
		double val2 = (double)inputImage.Height / (double)desiredSize.Height;
		double num = Math.Max(val, val2);
		int width = (int)Math.Floor((double)inputImage.Width / num);
		int height = (int)Math.Floor((double)inputImage.Height / num);
		Image val3 = ImageScaler.QuantizeImage(ImageScaler.ResizeImage(ImageScaler.SetImageBackground(inputImage, Color.White), new Size(width, height)), maxColors);
		ColorPalette palette = val3.Palette;
		for (int i = 0; i < val3.Palette.Entries.Length; i++)
		{
			if (i < maxColors)
			{
				palette.Entries[i] = val3.Palette.Entries[i];
			}
			else
			{
				palette.Entries[i] = Color.FromArgb(255, 255, 0, 0);
			}
		}
		val3.Palette = palette;
		return val3;
	}
}
