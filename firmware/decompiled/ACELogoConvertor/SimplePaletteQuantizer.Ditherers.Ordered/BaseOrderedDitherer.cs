using System;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Ditherers.Ordered;

public abstract class BaseOrderedDitherer : BaseColorDitherer
{
	protected abstract byte MatrixWidth { get; }

	protected abstract byte MatrixHeight { get; }

	public override bool IsInplace => true;

	protected override bool OnProcessPixel(Pixel sourcePixel, Pixel targetPixel)
	{
		Color colorFromPixel = SourceBuffer.GetColorFromPixel(sourcePixel);
		colorFromPixel = QuantizationHelper.ConvertAlpha(colorFromPixel);
		int num = targetPixel.X % MatrixWidth;
		int num2 = targetPixel.Y % MatrixHeight;
		int num3 = Convert.ToInt32(CachedMatrix[num, num2]);
		if (num3 > 0)
		{
			int clampedColorElement = GetClampedColorElement(colorFromPixel.R + num3);
			int clampedColorElement2 = GetClampedColorElement(colorFromPixel.G + num3);
			int clampedColorElement3 = GetClampedColorElement(colorFromPixel.B + num3);
			Color color = Color.FromArgb(255, clampedColorElement, clampedColorElement2, clampedColorElement3);
			if (TargetBuffer.IsIndexed)
			{
				byte index = (byte)Quantizer.GetPaletteIndex(color, targetPixel.X, targetPixel.Y);
				targetPixel.Index = index;
			}
			else
			{
				targetPixel.Color = color;
			}
		}
		return true;
	}
}
