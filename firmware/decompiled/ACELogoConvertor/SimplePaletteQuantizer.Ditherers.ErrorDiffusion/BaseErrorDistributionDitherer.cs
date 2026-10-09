using System.Drawing;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public abstract class BaseErrorDistributionDitherer : BaseColorDitherer
{
	protected abstract int MatrixSideWidth { get; }

	protected abstract int MatrixSideHeight { get; }

	public override bool IsInplace => false;

	private void ProcessNeighbor(Pixel targetPixel, int x, int y, float factor, int redError, int greenError, int blueError)
	{
		Color color = TargetBuffer.ReadColorUsingPixelFrom(targetPixel, x, y);
		color = QuantizationHelper.ConvertAlpha(color);
		int clampedColorElementWithError = GetClampedColorElementWithError(color.R, factor, redError);
		int clampedColorElementWithError2 = GetClampedColorElementWithError(color.G, factor, greenError);
		int clampedColorElementWithError3 = GetClampedColorElementWithError(color.B, factor, blueError);
		Color color2 = Color.FromArgb(255, clampedColorElementWithError, clampedColorElementWithError2, clampedColorElementWithError3);
		TargetBuffer.WriteColorUsingPixelAt(targetPixel, x, y, color2, Quantizer);
	}

	protected override bool OnProcessPixel(Pixel sourcePixel, Pixel targetPixel)
	{
		if (!TargetBuffer.IsIndexed)
		{
			return false;
		}
		Color colorFromPixel = SourceBuffer.GetColorFromPixel(sourcePixel);
		Color colorFromPixel2 = TargetBuffer.GetColorFromPixel(targetPixel);
		colorFromPixel = QuantizationHelper.ConvertAlpha(colorFromPixel);
		int num = colorFromPixel.R - colorFromPixel2.R;
		int num2 = colorFromPixel.G - colorFromPixel2.G;
		int num3 = colorFromPixel.B - colorFromPixel2.B;
		if (num != 0 || num2 != 0 || num3 != 0)
		{
			for (int i = -MatrixSideHeight; i <= MatrixSideHeight; i++)
			{
				for (int j = -MatrixSideWidth; j <= MatrixSideWidth; j++)
				{
					int num4 = sourcePixel.X + j;
					int num5 = sourcePixel.Y + i;
					byte b = CachedMatrix[i + MatrixSideHeight, j + MatrixSideWidth];
					float factor = CachedSummedMatrix[i + MatrixSideHeight, j + MatrixSideWidth];
					if (b != 0 && num4 >= 0 && num4 < TargetBuffer.Width && num5 >= 0 && num5 < TargetBuffer.Height)
					{
						ProcessNeighbor(targetPixel, num4, num5, factor, num, num2, num3);
					}
				}
			}
		}
		return false;
	}
}
