using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.PathProviders;

public class SerpentinePathProvider : IPathProvider
{
	public IList<Point> GetPointPath(int width, int height)
	{
		bool flag = true;
		List<Point> list = new List<Point>(width * height);
		for (int i = 0; i < height; i++)
		{
			for (int j = ((!flag) ? (width - 1) : 0); flag ? (j < width) : (j >= 0); j += (flag ? 1 : (-1)))
			{
				Point item = new Point(j, i);
				list.Add(item);
			}
			flag = !flag;
		}
		return list;
	}
}
