using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 2)]
public struct PixelDataRgb565 : INonIndexedPixel
{
	[FieldOffset(0)]
	private byte blue;

	[FieldOffset(0)]
	private ushort green;

	[FieldOffset(2)]
	private byte red;

	[FieldOffset(0)]
	private ushort raw;

	public int Alpha => 255;

	public int Red => (red >> 3) & 0xF;

	public int Green => (green >> 5) & 0x1F;

	public int Blue => blue & 0xF;

	public int Argb => -16777216 | raw;

	public ulong Value
	{
		get
		{
			return raw;
		}
		set
		{
			raw = (ushort)(value & 0xFFFF);
		}
	}

	public Color GetColor()
	{
		return Color.FromArgb(Argb);
	}

	public void SetColor(Color color)
	{
		red = (byte)(color.R >> 3);
		green = (byte)(color.G >> 2);
		blue = (byte)(color.B >> 3);
	}
}
