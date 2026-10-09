using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 3)]
public struct PixelDataRgb888 : INonIndexedPixel
{
	[FieldOffset(0)]
	private byte blue;

	[FieldOffset(1)]
	private byte green;

	[FieldOffset(2)]
	private byte red;

	public int Alpha => 255;

	public int Red => red;

	public int Green => green;

	public int Blue => blue;

	public int Argb => -16777216 | (Red << 16) | (Green << 8) | Blue;

	public ulong Value
	{
		get
		{
			return (uint)Argb;
		}
		set
		{
			red = (byte)((value >> 16) & 0xFF);
			green = (byte)((value >> 8) & 0xFF);
			blue = (byte)(value & 0xFF);
		}
	}

	public Color GetColor()
	{
		return Color.FromArgb(Argb);
	}

	public void SetColor(Color color)
	{
		red = color.R;
		green = color.G;
		blue = color.B;
	}
}
