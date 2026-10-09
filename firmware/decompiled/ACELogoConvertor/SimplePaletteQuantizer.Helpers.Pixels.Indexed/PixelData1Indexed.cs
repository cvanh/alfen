using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.Indexed;

[StructLayout(LayoutKind.Sequential, Size = 1)]
public struct PixelData1Indexed : IIndexedPixel
{
	private byte index;

	public byte GetIndex(int offset)
	{
		return ((index & (1 << 7 - offset)) != 0) ? ((byte)1) : ((byte)0);
	}

	public void SetIndex(int offset, byte value)
	{
		value = ((value == 0) ? ((byte)1) : ((byte)0));
		if (value == 0)
		{
			index |= (byte)(1 << 7 - offset);
		}
		else
		{
			index &= (byte)(~(1 << 7 - offset));
		}
	}
}
