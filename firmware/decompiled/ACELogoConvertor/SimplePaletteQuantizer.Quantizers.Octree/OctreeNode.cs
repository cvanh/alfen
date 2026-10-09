using System;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;

namespace SimplePaletteQuantizer.Quantizers.Octree;

internal class OctreeNode
{
	private static readonly byte[] Mask = new byte[8] { 128, 64, 32, 16, 8, 4, 2, 1 };

	private int red;

	private int green;

	private int blue;

	private int pixelCount;

	private int paletteIndex;

	private readonly OctreeNode[] nodes;

	public bool IsLeaf => pixelCount > 0;

	public Color Color
	{
		get
		{
			if (IsLeaf)
			{
				return (pixelCount == 1) ? Color.FromArgb(255, red, green, blue) : Color.FromArgb(255, red / pixelCount, green / pixelCount, blue / pixelCount);
			}
			throw new InvalidOperationException("Cannot retrieve a color for other node than leaf.");
		}
	}

	public int ActiveNodesPixelCount
	{
		get
		{
			int num = pixelCount;
			for (int i = 0; i < 8; i++)
			{
				OctreeNode octreeNode = nodes[i];
				if (octreeNode != null)
				{
					num += octreeNode.pixelCount;
				}
			}
			return num;
		}
	}

	public IEnumerable<OctreeNode> ActiveNodes
	{
		get
		{
			List<OctreeNode> list = new List<OctreeNode>();
			for (int i = 0; i < 8; i++)
			{
				OctreeNode octreeNode = nodes[i];
				if (octreeNode != null)
				{
					if (octreeNode.IsLeaf)
					{
						list.Add(octreeNode);
					}
					else
					{
						list.AddRange(octreeNode.ActiveNodes);
					}
				}
			}
			return list;
		}
	}

	public OctreeNode(int level, OctreeQuantizer parent)
	{
		nodes = new OctreeNode[8];
		if (level < 7)
		{
			parent.AddLevelNode(level, this);
		}
	}

	public void AddColor(Color color, int level, OctreeQuantizer parent)
	{
		if (level == 8)
		{
			red += color.R;
			green += color.G;
			blue += color.B;
			pixelCount++;
		}
		else if (level < 8)
		{
			int colorIndexAtLevel = GetColorIndexAtLevel(color, level);
			if (nodes[colorIndexAtLevel] == null)
			{
				nodes[colorIndexAtLevel] = new OctreeNode(level, parent);
			}
			nodes[colorIndexAtLevel].AddColor(color, level + 1, parent);
		}
	}

	public int GetPaletteIndex(Color color, int level)
	{
		if (IsLeaf)
		{
			return paletteIndex;
		}
		int colorIndexAtLevel = GetColorIndexAtLevel(color, level);
		return (nodes[colorIndexAtLevel] != null) ? nodes[colorIndexAtLevel].GetPaletteIndex(color, level + 1) : nodes.Where((OctreeNode node) => node != null).First().GetPaletteIndex(color, level + 1);
	}

	public int RemoveLeaves(int level, int activeColorCount, int targetColorCount, OctreeQuantizer parent)
	{
		int num = 0;
		for (int i = 0; i < 8; i++)
		{
			OctreeNode octreeNode = nodes[i];
			if (octreeNode != null)
			{
				red += octreeNode.red;
				green += octreeNode.green;
				blue += octreeNode.blue;
				pixelCount += octreeNode.pixelCount;
				num++;
			}
		}
		return num - 1;
	}

	private static int GetColorIndexAtLevel(Color color, int level)
	{
		return (((color.R & Mask[level]) == Mask[level]) ? 4 : 0) | (((color.G & Mask[level]) == Mask[level]) ? 2 : 0) | (((color.B & Mask[level]) == Mask[level]) ? 1 : 0);
	}

	internal void SetPaletteIndex(int index)
	{
		paletteIndex = index;
	}
}
