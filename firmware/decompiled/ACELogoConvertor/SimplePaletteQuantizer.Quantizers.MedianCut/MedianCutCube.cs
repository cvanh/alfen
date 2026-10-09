using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using SimplePaletteQuantizer.ColorCaches.Common;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.MedianCut;

internal class MedianCutCube
{
	private int redLowBound;

	private int redHighBound;

	private int greenLowBound;

	private int greenHighBound;

	private int blueLowBound;

	private int blueHighBound;

	private readonly ICollection<int> colorList;

	public ColorModel ColorModel { get; private set; }

	public int PaletteIndex { get; private set; }

	public int RedSize => redHighBound - redLowBound;

	public int GreenSize => greenHighBound - greenLowBound;

	public int BlueSize => blueHighBound - blueLowBound;

	public Color Color
	{
		get
		{
			int num = 0;
			int num2 = 0;
			int num3 = 0;
			foreach (int color2 in colorList)
			{
				Color color = Color.FromArgb(color2);
				num += ColorModelHelper.GetComponentA(ColorModel, color);
				num2 += ColorModelHelper.GetComponentB(ColorModel, color);
				num3 += ColorModelHelper.GetComponentC(ColorModel, color);
			}
			num = ((colorList.Count != 0) ? (num / colorList.Count) : 0);
			num2 = ((colorList.Count != 0) ? (num2 / colorList.Count) : 0);
			num3 = ((colorList.Count != 0) ? (num3 / colorList.Count) : 0);
			return Color.FromArgb(255, num, num2, num3);
		}
	}

	public MedianCutCube(ICollection<int> colors)
	{
		ColorModel = ColorModel.RedGreenBlue;
		colorList = colors;
		Shrink();
	}

	private void Shrink()
	{
		redLowBound = (greenLowBound = (blueLowBound = 255));
		redHighBound = (greenHighBound = (blueHighBound = 0));
		foreach (int color2 in colorList)
		{
			Color color = Color.FromArgb(color2);
			int componentA = ColorModelHelper.GetComponentA(ColorModel, color);
			int componentB = ColorModelHelper.GetComponentB(ColorModel, color);
			int componentC = ColorModelHelper.GetComponentC(ColorModel, color);
			if (componentA < redLowBound)
			{
				redLowBound = componentA;
			}
			if (componentA > redHighBound)
			{
				redHighBound = componentA;
			}
			if (componentB < greenLowBound)
			{
				greenLowBound = componentB;
			}
			if (componentB > greenHighBound)
			{
				greenHighBound = componentB;
			}
			if (componentC < blueLowBound)
			{
				blueLowBound = componentC;
			}
			if (componentC > blueHighBound)
			{
				blueHighBound = componentC;
			}
		}
	}

	public void SplitAtMedian(byte componentIndex, out MedianCutCube firstMedianCutCube, out MedianCutCube secondMedianCutCube)
	{
		List<int> list = componentIndex switch
		{
			0 => colorList.OrderBy((int argb) => ColorModelHelper.GetComponentA(ColorModel, Color.FromArgb(argb))).ToList(), 
			1 => colorList.OrderBy((int argb) => ColorModelHelper.GetComponentB(ColorModel, Color.FromArgb(argb))).ToList(), 
			2 => colorList.OrderBy((int argb) => ColorModelHelper.GetComponentC(ColorModel, Color.FromArgb(argb))).ToList(), 
			_ => throw new NotSupportedException("Only three color components are supported (R, G and B)."), 
		};
		int num = colorList.Count >> 1;
		firstMedianCutCube = new MedianCutCube(list.GetRange(0, num));
		secondMedianCutCube = new MedianCutCube(list.GetRange(num, list.Count - num));
	}

	public void SetPaletteIndex(int newPaletteIndex)
	{
		PaletteIndex = newPaletteIndex;
	}

	public bool IsColorIn(Color color)
	{
		int componentA = ColorModelHelper.GetComponentA(ColorModel, color);
		int componentB = ColorModelHelper.GetComponentB(ColorModel, color);
		int componentC = ColorModelHelper.GetComponentC(ColorModel, color);
		if (componentA >= redLowBound && componentA <= redHighBound && componentB >= greenLowBound && componentB <= greenHighBound)
		{
			if (componentC >= blueLowBound)
			{
				return componentC <= blueHighBound;
			}
			return false;
		}
		return false;
	}
}
