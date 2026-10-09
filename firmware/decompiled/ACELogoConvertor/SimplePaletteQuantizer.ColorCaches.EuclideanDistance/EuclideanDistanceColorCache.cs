using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.ColorCaches.Common;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.ColorCaches.EuclideanDistance;

public class EuclideanDistanceColorCache : BaseColorCache
{
	private IList<Color> palette;

	public override bool IsColorModelSupported => true;

	public EuclideanDistanceColorCache()
	{
		ColorModel = ColorModel.RedGreenBlue;
	}

	public EuclideanDistanceColorCache(ColorModel colorModel)
	{
		ColorModel = colorModel;
	}

	protected override void OnCachePalette(IList<Color> palette)
	{
		this.palette = palette;
	}

	protected override void OnGetColorPaletteIndex(Color color, out int paletteIndex)
	{
		paletteIndex = ColorModelHelper.GetEuclideanDistance(color, ColorModel, palette);
	}
}
