using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.ColorCaches;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.MedianCut;

public class MedianCutQuantizer : BaseColorCacheQuantizer
{
	private ConcurrentBag<MedianCutCube> cubeList;

	public override bool AllowParallel => true;

	private void SplitCubes(int colorCount)
	{
		List<MedianCutCube> list = new List<MedianCutCube>();
		foreach (MedianCutCube cube in cubeList)
		{
			if (list.Count >= colorCount)
			{
				break;
			}
			MedianCutCube firstMedianCutCube;
			MedianCutCube secondMedianCutCube;
			if (cube.RedSize >= cube.GreenSize && cube.RedSize >= cube.BlueSize)
			{
				cube.SplitAtMedian(0, out firstMedianCutCube, out secondMedianCutCube);
			}
			else if (cube.GreenSize >= cube.BlueSize)
			{
				cube.SplitAtMedian(1, out firstMedianCutCube, out secondMedianCutCube);
			}
			else
			{
				cube.SplitAtMedian(2, out firstMedianCutCube, out secondMedianCutCube);
			}
			list.Add(firstMedianCutCube);
			if (list.Count >= colorCount)
			{
				break;
			}
			list.Add(secondMedianCutCube);
		}
		cubeList = new ConcurrentBag<MedianCutCube>();
		foreach (MedianCutCube item in list)
		{
			cubeList.Add(item);
		}
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		OnFinish();
	}

	protected override IColorCache OnCreateDefaultCache()
	{
		return null;
	}

	protected override List<Color> OnGetPaletteToCache(int colorCount)
	{
		MedianCutCube item = new MedianCutCube(UniqueColors.Keys);
		cubeList.Add(item);
		int i;
		for (i = 1; 1 << i < colorCount; i++)
		{
		}
		for (int j = 0; j < i; j++)
		{
			SplitCubes(colorCount);
		}
		List<Color> list = new List<Color>();
		int num = 0;
		foreach (MedianCutCube cube in cubeList)
		{
			list.Add(cube.Color);
			cube.SetPaletteIndex(num++);
		}
		return list;
	}

	protected override void OnFinish()
	{
		base.OnFinish();
		cubeList = new ConcurrentBag<MedianCutCube>();
	}

	public void GetPaletteIndex(Color color, out int paletteIndex)
	{
		paletteIndex = 0;
		color = QuantizationHelper.ConvertAlpha(color);
		foreach (MedianCutCube cube in cubeList)
		{
			if (cube.IsColorIn(color))
			{
				paletteIndex = cube.PaletteIndex;
				break;
			}
		}
	}
}
