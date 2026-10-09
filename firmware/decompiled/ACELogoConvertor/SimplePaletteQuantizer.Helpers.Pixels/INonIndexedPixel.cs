using System.Drawing;

namespace SimplePaletteQuantizer.Helpers.Pixels;

public interface INonIndexedPixel
{
	int Alpha { get; }

	int Red { get; }

	int Green { get; }

	int Blue { get; }

	int Argb { get; }

	ulong Value { get; set; }

	Color GetColor();

	void SetColor(Color color);
}
