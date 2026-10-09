using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using SimplePaletteQuantizer.ColorCaches;
using SimplePaletteQuantizer.ColorCaches.Octree;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.Popularity;

public class PopularityQuantizer : BaseColorCacheQuantizer
{
	private List<Color> palette;

	private ConcurrentDictionary<int, PopularityColorSlot> colorMap;

	public override bool AllowParallel => true;

	private static int GetColorIndex(Color color)
	{
		int num = color.R >> 2;
		int num2 = color.G >> 2;
		int num3 = color.B >> 2;
		return (num << 12) + (num2 << 6) + num3;
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		palette = new List<Color>();
		colorMap = new ConcurrentDictionary<int, PopularityColorSlot>();
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		base.OnAddColor(color, key, x, y);
		int colorIndex = GetColorIndex(color);
		colorMap.AddOrUpdate(colorIndex, (int colorKey) => new PopularityColorSlot(color), (int colorKey, PopularityColorSlot slot) => slot.AddValue(color));
	}

	protected override IColorCache OnCreateDefaultCache()
	{
		return new OctreeColorCache();
	}

	protected override List<Color> OnGetPaletteToCache(int colorCount)
	{
		FastRandom random = new FastRandom(0u);
		IEnumerable<Color> collection = from entry in (from entry in colorMap
				orderby random.Next(colorMap.Count)
				orderby entry.Value.PixelCount descending
				select entry).Take(colorCount)
			select entry.Value.GetAverage();
		palette.Clear();
		palette.AddRange(collection);
		return palette;
	}
}
