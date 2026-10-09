using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using SimplePaletteQuantizer.ColorCaches.Common;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.ColorCaches.Octree;

public class OctreeColorCache : BaseColorCache
{
	private OctreeCacheNode root;

	public override bool IsColorModelSupported => false;

	public OctreeColorCache()
	{
		ColorModel = ColorModel.RedGreenBlue;
		root = new OctreeCacheNode();
	}

	public override void Prepare()
	{
		base.Prepare();
		root = new OctreeCacheNode();
	}

	protected override void OnCachePalette(IList<Color> palette)
	{
		int num = 0;
		foreach (Color item in palette)
		{
			root.AddColor(item, num++, 0);
		}
	}

	protected override void OnGetColorPaletteIndex(Color color, out int paletteIndex)
	{
		Dictionary<int, Color> paletteIndex2 = root.GetPaletteIndex(color, 0);
		paletteIndex = 0;
		int num = 0;
		int euclideanDistance = ColorModelHelper.GetEuclideanDistance(color, ColorModel, paletteIndex2.Values.ToList());
		foreach (int key in paletteIndex2.Keys)
		{
			if (num == euclideanDistance)
			{
				paletteIndex = key;
				break;
			}
			num++;
		}
	}
}
