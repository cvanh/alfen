using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.PathProviders;

public interface IPathProvider
{
	IList<Point> GetPointPath(int width, int height);
}
