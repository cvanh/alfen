namespace SimplePaletteQuantizer.Helpers.Pixels;

public interface IIndexedPixel
{
	byte GetIndex(int offset);

	void SetIndex(int offset, byte value);
}
