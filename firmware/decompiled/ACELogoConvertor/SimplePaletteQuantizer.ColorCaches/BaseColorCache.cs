using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.ColorCaches.Common;

namespace SimplePaletteQuantizer.ColorCaches;

public abstract class BaseColorCache : IColorCache
{
	private readonly ConcurrentDictionary<int, int> cache;

	protected ColorModel ColorModel { get; set; }

	public abstract bool IsColorModelSupported { get; }

	protected BaseColorCache()
	{
		cache = new ConcurrentDictionary<int, int>();
	}

	public void ChangeColorModel(ColorModel colorModel)
	{
		ColorModel = colorModel;
	}

	protected abstract void OnCachePalette(IList<Color> palette);

	protected abstract void OnGetColorPaletteIndex(Color color, out int paletteIndex);

	public virtual void Prepare()
	{
		cache.Clear();
	}

	public void CachePalette(IList<Color> palette)
	{
		OnCachePalette(palette);
	}

	public void GetColorPaletteIndex(Color color, out int paletteIndex)
	{
		int key = (color.R << 16) | (color.G << 8) | color.B;
		paletteIndex = cache.AddOrUpdate(key, (int colorKey) =>
		{
			OnGetColorPaletteIndex(color, out var paletteIndex2);
			return paletteIndex2;
		}, (int colorKey, int inputIndex) => inputIndex);
	}
}
