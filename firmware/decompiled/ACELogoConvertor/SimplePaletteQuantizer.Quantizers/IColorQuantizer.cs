using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;
using SimplePaletteQuantizer.PathProviders;

namespace SimplePaletteQuantizer.Quantizers;

public interface IColorQuantizer : IPathProvider
{
	bool AllowParallel { get; }

	void Prepare(ImageBuffer image);

	void AddColor(Color color, int x, int y);

	List<Color> GetPalette(int colorCount);

	int GetPaletteIndex(Color color, int x, int y);

	int GetColorCount();

	void Finish();
}
