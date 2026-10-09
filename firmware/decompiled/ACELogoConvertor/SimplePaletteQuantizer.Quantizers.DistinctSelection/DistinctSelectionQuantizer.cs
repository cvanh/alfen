using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using SimplePaletteQuantizer.ColorCaches;
using SimplePaletteQuantizer.ColorCaches.Octree;
using SimplePaletteQuantizer.Extensions;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.DistinctSelection;

public class DistinctSelectionQuantizer : BaseColorCacheQuantizer
{
	private class ColorHueComparer : IEqualityComparer<DistinctColorInfo>
	{
		public bool Equals(DistinctColorInfo x, DistinctColorInfo y)
		{
			return x.Hue == y.Hue;
		}

		public int GetHashCode(DistinctColorInfo colorInfo)
		{
			return colorInfo.Hue.GetHashCode();
		}
	}

	private class ColorSaturationComparer : IEqualityComparer<DistinctColorInfo>
	{
		public bool Equals(DistinctColorInfo x, DistinctColorInfo y)
		{
			return x.Saturation == y.Saturation;
		}

		public int GetHashCode(DistinctColorInfo colorInfo)
		{
			return colorInfo.Saturation.GetHashCode();
		}
	}

	private class ColorBrightnessComparer : IEqualityComparer<DistinctColorInfo>
	{
		public bool Equals(DistinctColorInfo x, DistinctColorInfo y)
		{
			return x.Brightness == y.Brightness;
		}

		public int GetHashCode(DistinctColorInfo colorInfo)
		{
			return colorInfo.Brightness.GetHashCode();
		}
	}

	private List<Color> palette;

	private int foundColorCount;

	private ConcurrentDictionary<int, DistinctColorInfo> colorMap;

	public override bool AllowParallel => true;

	private static bool ProcessList(int colorCount, List<DistinctColorInfo> list, ICollection<IEqualityComparer<DistinctColorInfo>> comparers, out List<DistinctColorInfo> outputList)
	{
		IEqualityComparer<DistinctColorInfo> item = null;
		int num = 0;
		outputList = list;
		foreach (IEqualityComparer<DistinctColorInfo> comparer in comparers)
		{
			List<DistinctColorInfo> list2 = list.Distinct(comparer).ToList();
			int count = list2.Count;
			if (count > colorCount && count > num)
			{
				num = count;
				item = comparer;
				outputList = list2;
				if (num <= colorCount)
				{
					break;
				}
			}
		}
		comparers.Remove(item);
		if (comparers.Count > 0)
		{
			return num > colorCount;
		}
		return false;
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		OnFinish();
	}

	protected override IColorCache OnCreateDefaultCache()
	{
		return new OctreeColorCache();
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		colorMap.AddOrUpdate(key, (int colorKey) => new DistinctColorInfo(color), (int colorKey, DistinctColorInfo colorInfo) => colorInfo.IncreaseCount());
	}

	protected override List<Color> OnGetPaletteToCache(int colorCount)
	{
		palette.Clear();
		FastRandom random = new FastRandom(13u);
		List<DistinctColorInfo> outputList = colorMap.Values.ToList();
		foundColorCount = outputList.Count;
		if (foundColorCount >= colorCount)
		{
			outputList = outputList.OrderBy((DistinctColorInfo entry) => random.Next(foundColorCount)).ToList();
			DistinctColorInfo distinctColorInfo = Extend.MaxBy(outputList, (DistinctColorInfo info) => info.Count);
			outputList.Remove(distinctColorInfo);
			colorCount--;
			ColorHueComparer item = new ColorHueComparer();
			ColorSaturationComparer item2 = new ColorSaturationComparer();
			ColorBrightnessComparer item3 = new ColorBrightnessComparer();
			List<IEqualityComparer<DistinctColorInfo>> comparers = new List<IEqualityComparer<DistinctColorInfo>> { item, item2, item3 };
			while (ProcessList(colorCount, outputList, comparers, out outputList))
			{
			}
			int num = outputList.Count();
			if (num > 0)
			{
				int count = Math.Min(colorCount, num);
				outputList = outputList.Take(count).ToList();
			}
			palette.Add(Color.FromArgb(distinctColorInfo.Color));
		}
		palette.AddRange(outputList.Select((DistinctColorInfo colorInfo) => Color.FromArgb(colorInfo.Color)));
		return palette;
	}

	protected override int OnGetColorCount()
	{
		return foundColorCount;
	}

	protected override void OnFinish()
	{
		base.OnFinish();
		palette = new List<Color>();
		colorMap = new ConcurrentDictionary<int, DistinctColorInfo>();
	}
}
