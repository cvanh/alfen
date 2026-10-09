using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 2)]
public struct PixelDataGray16 : INonIndexedPixel
{
	[FieldOffset(0)]
	private ushort gray;

	public int Gray => 0;

	public int Alpha => 255;

	public int Red => Gray;

	public int Green => Gray;

	public int Blue => Gray;

	public int Argb => -16777216 | (Red << 16) | (Green << 8) | Blue;

	public ulong Value
	{
		get
		{
			return gray;
		}
		set
		{
			gray = (ushort)(value & 0xFFFF);
		}
	}

	public Color GetColor()
	{
		return Color.FromArgb(Argb);
	}

	public void SetColor(Color color)
	{
		int num = color.ToArgb() & 0xFFFFFF;
		gray = (byte)(num >> 16);
	}
}
