using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.PathProviders;

public class StandardPathProvider : IPathProvider
{
	public IList<Point> GetPointPath(int width, int height)
	{
		List<Point> list = new List<Point>(width * height);
		for (int i = 0; i < height; i++)
		{
			for (int j = 0; j < width; j++)
			{
				Point item = new Point(j, i);
				list.Add(item);
			}
		}
		return list;
	}
}
