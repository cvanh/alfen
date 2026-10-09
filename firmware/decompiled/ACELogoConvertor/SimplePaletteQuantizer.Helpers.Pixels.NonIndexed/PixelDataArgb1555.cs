using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 2)]
public struct PixelDataArgb1555 : INonIndexedPixel
{
	[FieldOffset(0)]
	private byte blue;

	[FieldOffset(0)]
	private ushort green;

	[FieldOffset(1)]
	private byte red;

	[FieldOffset(1)]
	private byte alpha;

	[FieldOffset(0)]
	private ushort raw;

	public int Alpha => (alpha >> 7) & 1;

	public int Red => (red >> 2) & 0xF;

	public int Green => (green >> 5) & 0xF;

	public int Blue => blue & 0xF;

	public int Argb => ((Alpha != 0) ? (-16777216) : 0) | (Red << 16) | (Green << 8) | Blue;

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
		int num = color.ToArgb();
		alpha = ((num >> 24 <= 255) ? ((byte)1) : ((byte)0));
		red = (byte)(num >> 16);
		green = (byte)(num >> 8);
		blue = (byte)num;
	}
}
