using System;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.ColorCaches;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers;

public abstract class BaseColorCacheQuantizer : BaseColorQuantizer
{
	private IColorCache colorCache;

	protected BaseColorCacheQuantizer()
	{
		colorCache = null;
	}

	public void ChangeCacheProvider(IColorCache colorCache)
	{
		this.colorCache = colorCache;
	}

	public void CachePalette(IList<Color> palette)
	{
		GetColorCache().CachePalette(palette);
	}

	private IColorCache GetColorCache()
	{
		IColorCache colorCache = this.colorCache ?? (this.colorCache = OnCreateDefaultCache());
		if (colorCache == null)
		{
			throw new ArgumentNullException(string.Format("The color cache is not initialized! Please use SetColorCache() method on quantizer.", Array.Empty<object>()));
		}
		return colorCache;
	}

	protected abstract IColorCache OnCreateDefaultCache();

	protected abstract List<Color> OnGetPaletteToCache(int colorCount);

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		GetColorCache().Prepare();
	}

	protected sealed override List<Color> OnGetPalette(int colorCount)
	{
		List<Color> list = base.OnGetPalette(colorCount) ?? OnGetPaletteToCache(colorCount);
		GetColorCache().CachePalette(list);
		return list;
	}

	protected override void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		base.OnGetPaletteIndex(color, key, x, y, out paletteIndex);
		if (paletteIndex == -1)
		{
			GetColorCache().GetColorPaletteIndex(color, out paletteIndex);
		}
	}
}
