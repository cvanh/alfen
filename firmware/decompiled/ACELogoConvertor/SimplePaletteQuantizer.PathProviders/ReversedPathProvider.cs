using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.PathProviders;

public class ReversedPathProvider : IPathProvider
{
	public IList<Point> GetPointPath(int width, int height)
	{
		List<Point> list = new List<Point>(width * height);
		for (int num = height - 1; num >= 0; num--)
		{
			for (int num2 = width - 1; num2 >= 0; num2--)
			{
				Point item = new Point(num2, num);
				list.Add(item);
			}
		}
		return list;
	}
}
