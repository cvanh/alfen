using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 8)]
public struct PixelDataArgb64 : INonIndexedPixel
{
	[FieldOffset(0)]
	private ushort blue;

	[FieldOffset(2)]
	private ushort green;

	[FieldOffset(4)]
	private ushort red;

	[FieldOffset(6)]
	private ushort alpha;

	[FieldOffset(0)]
	private ulong raw;

	public int Alpha => alpha >> 8;

	public int Red => red >> 8;

	public int Green => green >> 8;

	public int Blue => blue >> 8;

	public int Argb => (Alpha << 16) | Red | (Green << 16) | Blue;

	public ulong Value
	{
		get
		{
			return raw;
		}
		set
		{
			raw = value;
		}
	}

	public Color GetColor()
	{
		return Color.FromArgb(Argb);
	}

	public void SetColor(Color color)
	{
		alpha = (ushort)(color.A << 8);
		red = (ushort)(color.R << 8);
		green = (ushort)(color.G << 8);
		blue = (ushort)(color.B << 8);
	}
}
