using System.Drawing;
using System.Runtime.InteropServices;

namespace SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

[StructLayout(LayoutKind.Explicit, Size = 4)]
public struct PixelDataArgb8888 : INonIndexedPixel
{
	[FieldOffset(0)]
	private readonly byte blue;

	[FieldOffset(1)]
	private readonly byte green;

	[FieldOffset(2)]
	private readonly byte red;

	[FieldOffset(3)]
	private readonly byte alpha;

	[FieldOffset(0)]
	private int raw;

	public int Alpha => alpha;

	public int Red => red;

	public int Green => green;

	public int Blue => blue;

	public int Argb => raw;

	public ulong Value
	{
		get
		{
			return (uint)raw;
		}
		set
		{
			raw = (int)(value & 0xFFFFFFFFu);
		}
	}

	public Color GetColor()
	{
		return Color.FromArgb(Argb);
	}

	public void SetColor(Color color)
	{
		raw = color.ToArgb();
	}
}
