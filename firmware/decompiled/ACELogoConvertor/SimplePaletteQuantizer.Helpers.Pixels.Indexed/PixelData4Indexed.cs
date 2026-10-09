using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.Indexed;

[StructLayout(LayoutKind.Sequential, Size = 1)]
public struct PixelData4Indexed : IIndexedPixel
{
	private byte index;

	public byte GetIndex(int offset)
	{
		return (byte)GetBitRange(8 - offset - 4, 7 - offset);
	}

	public void SetIndex(int offset, byte value)
	{
		SetBitRange(8 - offset - 4, 7 - offset, value);
	}

	private int GetBitRange(int startOffset, int endOffset)
	{
		int num = 0;
		byte b = 0;
		for (int i = startOffset; i <= endOffset; i++)
		{
			int num2 = 1 << (int)b;
			num += (GetBit(i) ? num2 : 0);
			b++;
		}
		return num;
	}

	private bool GetBit(int offset)
	{
		return (index & (1 << offset)) != 0;
	}

	private void SetBitRange(int startOffset, int endOffset, int value)
	{
		byte b = 0;
		for (int i = startOffset; i <= endOffset; i++)
		{
			int num = 1 << (int)b;
			SetBit(i, (value & num) != 0);
			b++;
		}
	}

	private void SetBit(int offset, bool value)
	{
		if (value)
		{
			index |= (byte)(1 << offset);
		}
		else
		{
			index &= (byte)(~(1 << offset));
		}
	}
}
