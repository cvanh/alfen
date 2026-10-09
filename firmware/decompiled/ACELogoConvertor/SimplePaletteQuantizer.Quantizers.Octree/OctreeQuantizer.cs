using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.Octree;

public class OctreeQuantizer : BaseColorQuantizer
{
	private OctreeNode root;

	private int lastColorCount;

	private List<OctreeNode>[] levels;

	internal IEnumerable<OctreeNode> Leaves => root.ActiveNodes.Where((OctreeNode node) => node.IsLeaf);

	public override bool AllowParallel => false;

	internal void AddLevelNode(int level, OctreeNode octreeNode)
	{
		levels[level].Add(octreeNode);
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		OnFinish();
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		root.AddColor(color, 0, this);
	}

	protected override List<Color> OnGetPalette(int colorCount)
	{
		List<Color> list = base.OnGetPalette(colorCount);
		if (list != null)
		{
			return list;
		}
		List<Color> list2 = new List<Color>();
		int num = (lastColorCount = Leaves.Count());
		int num2 = 0;
		for (int num3 = 6; num3 >= 0; num3--)
		{
			if (levels[num3].Count > 0)
			{
				foreach (OctreeNode item in levels[num3].OrderBy((OctreeNode node) => node.ActiveNodesPixelCount))
				{
					num -= item.RemoveLeaves(num3, num, colorCount, this);
					if (num <= colorCount)
					{
						break;
					}
				}
				if (num <= colorCount)
				{
					break;
				}
				levels[num3].Clear();
			}
		}
		foreach (OctreeNode item2 in Leaves.OrderByDescending((OctreeNode node) => node.ActiveNodesPixelCount))
		{
			if (num2 >= colorCount)
			{
				break;
			}
			if (item2.IsLeaf)
			{
				list2.Add(item2.Color);
			}
			item2.SetPaletteIndex(num2++);
		}
		if (list2.Count == 0)
		{
			throw new NotSupportedException("The Octree contains after the reduction 0 colors, it may happen for 1-16 colors because it reduces by 1-8 nodes at time. Should be used on 8 or above to ensure the correct functioning.");
		}
		return list2;
	}

	protected override void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		paletteIndex = root.GetPaletteIndex(color, 0);
	}

	protected override int OnGetColorCount()
	{
		return lastColorCount;
	}

	protected override void OnFinish()
	{
		base.OnFinish();
		levels = new List<OctreeNode>[7];
		for (int i = 0; i < 7; i++)
		{
			levels[i] = new List<OctreeNode>();
		}
		root = new OctreeNode(0, this);
	}
}
