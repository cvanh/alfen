using SimplePaletteQuantizer.Helpers;
using SimplePaletteQuantizer.PathProviders;
using SimplePaletteQuantizer.Quantizers;

namespace SimplePaletteQuantizer.Ditherers;

public interface IColorDitherer : IPathProvider
{
	bool IsInplace { get; }

	void Prepare(IColorQuantizer quantizer, int colorCount, ImageBuffer sourceBuffer, ImageBuffer targetBuffer);

	bool ProcessPixel(Pixel sourcePixel, Pixel targetPixel);

	void Finish();
}
