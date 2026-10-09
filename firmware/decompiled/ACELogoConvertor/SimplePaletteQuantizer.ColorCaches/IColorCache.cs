using System.Collections.Generic;
using System.Drawing;

namespace SimplePaletteQuantizer.ColorCaches;

public interface IColorCache
{
	void Prepare();

	void CachePalette(IList<Color> palette);

	void GetColorPaletteIndex(Color color, out int paletteIndex);
}
