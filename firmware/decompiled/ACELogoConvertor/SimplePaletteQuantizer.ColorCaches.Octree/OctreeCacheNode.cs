using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.ColorCaches.Octree;

public class OctreeCacheNode
{
	private static readonly byte[] Mask = new byte[8] { 128, 64, 32, 16, 8, 4, 2, 1 };

	private readonly OctreeCacheNode[] nodes;

	private readonly Dictionary<int, Color> entries;

	public OctreeCacheNode()
	{
		nodes = new OctreeCacheNode[8];
		entries = new Dictionary<int, Color>();
	}

	public void AddColor(Color color, int paletteIndex, int level)
	{
		entries.Add(paletteIndex, color);
		if (level < 8)
		{
			int colorIndexAtLevel = GetColorIndexAtLevel(color, level);
			if (nodes[colorIndexAtLevel] == null)
			{
				nodes[colorIndexAtLevel] = new OctreeCacheNode();
			}
			nodes[colorIndexAtLevel].AddColor(color, paletteIndex, level + 1);
		}
	}

	public Dictionary<int, Color> GetPaletteIndex(Color color, int level)
	{
		Dictionary<int, Color> paletteIndex = entries;
		if (level < 8)
		{
			int colorIndexAtLevel = GetColorIndexAtLevel(color, level);
			if (nodes[colorIndexAtLevel] != null)
			{
				paletteIndex = nodes[colorIndexAtLevel].GetPaletteIndex(color, level + 1);
			}
		}
		return paletteIndex;
	}

	private static int GetColorIndexAtLevel(Color color, int level)
	{
		return (((color.R & Mask[level]) == Mask[level]) ? 4 : 0) | (((color.G & Mask[level]) == Mask[level]) ? 2 : 0) | (((color.B & Mask[level]) == Mask[level]) ? 1 : 0);
	}
}
