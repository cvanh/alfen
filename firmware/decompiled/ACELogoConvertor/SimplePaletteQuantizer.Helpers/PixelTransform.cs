namespace SimplePaletteQuantizer.Helpers;

public class PixelTransform
{
	public Pixel SourcePixel { get; private set; }

	public Pixel TargetPixel { get; private set; }

	public PixelTransform(Pixel sourcePixel, Pixel targetPixel)
	{
		SourcePixel = sourcePixel;
		TargetPixel = targetPixel;
	}
}
