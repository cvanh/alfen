using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.Linq;

namespace SimplePaletteQuantizer.Extensions;

public static class Extend
{
	public static IEnumerable<T> Distinct<T, TKey>(this IEnumerable<T> items, Func<T, TKey> selector)
	{
		HashSet<TKey> keys = new HashSet<TKey>();
		return items.Where((T item) => keys.Add(selector(item)));
	}

	public static TSource MaxBy<TSource, TKey>(this IEnumerable<TSource> source, Func<TSource, TKey> selector)
	{
		return source.MaxBy(selector, Comparer<TKey>.Default);
	}

	public static T MaxBy<T, TKey>(this IEnumerable<T> source, Func<T, TKey> selector, IComparer<TKey> comparer)
	{
		using IEnumerator<T> enumerator = source.GetEnumerator();
		if (!enumerator.MoveNext())
		{
			throw new InvalidOperationException("Sequence was empty");
		}
		T val = enumerator.Current;
		TKey y = selector(val);
		while (enumerator.MoveNext())
		{
			T current = enumerator.Current;
			TKey val2 = selector(current);
			if (comparer.Compare(val2, y) > 0)
			{
				val = current;
				y = val2;
			}
		}
		return val;
	}

	public static int GetPaletteColorCount(this Image image)
	{
		//IL_000f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0021: Unknown result type (might be due to invalid IL or missing references)
		if (image == null)
		{
			throw new ArgumentNullException("Cannot assign a palette to a null image.");
		}
		if (!image.PixelFormat.IsIndexed())
		{
			throw new InvalidOperationException($"Cannot retrieve a color count from a non-indexed image with pixel format '{image.PixelFormat}'.");
		}
		return image.Palette.Entries.Length;
	}

	public static List<Color> GetPalette(this Image image)
	{
		//IL_000f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0021: Unknown result type (might be due to invalid IL or missing references)
		if (image == null)
		{
			throw new ArgumentNullException("Cannot assign a palette to a null image.");
		}
		if (!image.PixelFormat.IsIndexed())
		{
			throw new InvalidOperationException($"Cannot retrieve a palette from a non-indexed image with pixel format '{image.PixelFormat}'.");
		}
		return image.Palette.Entries.ToList();
	}

	public static void SetPalette(this Image image, List<Color> palette)
	{
		//IL_001d: Unknown result type (might be due to invalid IL or missing references)
		//IL_002f: Unknown result type (might be due to invalid IL or missing references)
		if (palette == null)
		{
			throw new ArgumentNullException("Cannot assign a null palette.");
		}
		if (image == null)
		{
			throw new ArgumentNullException("Cannot assign a palette to a null image.");
		}
		if (!image.PixelFormat.IsIndexed())
		{
			throw new InvalidOperationException($"Cannot store a palette to a non-indexed image with pixel format '{image.PixelFormat}'.");
		}
		ColorPalette palette2 = image.Palette;
		if (palette.Count > palette2.Entries.Length)
		{
			throw new ArgumentOutOfRangeException($"Cannot store a palette with '{palette.Count}' colors intto an image palette where only '{palette2.Entries.Length}' colors are allowed.");
		}
		for (int i = 0; i < palette.Count; i++)
		{
			palette2.Entries[i] = palette[i];
		}
		image.Palette = palette2;
	}

	public static byte GetBitDepth(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_0049: Unknown result type (might be due to invalid IL or missing references)
		//IL_004f: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_006b: Unknown result type (might be due to invalid IL or missing references)
		//IL_0071: Invalid comparison between Unknown and I4
		//IL_0051: Unknown result type (might be due to invalid IL or missing references)
		//IL_0057: Invalid comparison between Unknown and I4
		//IL_002f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0035: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Unknown result type (might be due to invalid IL or missing references)
		//IL_0018: Invalid comparison between Unknown and I4
		//IL_0085: Unknown result type (might be due to invalid IL or missing references)
		//IL_008b: Invalid comparison between Unknown and I4
		//IL_0073: Unknown result type (might be due to invalid IL or missing references)
		//IL_0079: Invalid comparison between Unknown and I4
		//IL_0059: Unknown result type (might be due to invalid IL or missing references)
		//IL_005f: Invalid comparison between Unknown and I4
		//IL_0037: Unknown result type (might be due to invalid IL or missing references)
		//IL_003d: Invalid comparison between Unknown and I4
		//IL_001d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0023: Invalid comparison between Unknown and I4
		//IL_008d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0093: Invalid comparison between Unknown and I4
		//IL_007b: Unknown result type (might be due to invalid IL or missing references)
		//IL_0081: Invalid comparison between Unknown and I4
		//IL_0061: Unknown result type (might be due to invalid IL or missing references)
		//IL_0067: Invalid comparison between Unknown and I4
		//IL_003f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0045: Invalid comparison between Unknown and I4
		//IL_0025: Unknown result type (might be due to invalid IL or missing references)
		//IL_002b: Invalid comparison between Unknown and I4
		//IL_00b1: Unknown result type (might be due to invalid IL or missing references)
		if ((int)pixelFormat <= 198659)
		{
			if ((int)pixelFormat <= 139273)
			{
				if (pixelFormat - 135173 <= 1)
				{
					goto IL_009d;
				}
				if ((int)pixelFormat == 137224)
				{
					return 24;
				}
				if ((int)pixelFormat == 139273)
				{
					goto IL_00a3;
				}
			}
			else
			{
				if ((int)pixelFormat == 196865)
				{
					return 1;
				}
				if ((int)pixelFormat == 197634)
				{
					return 4;
				}
				if ((int)pixelFormat == 198659)
				{
					return 8;
				}
			}
		}
		else
		{
			if ((int)pixelFormat <= 1052676)
			{
				if ((int)pixelFormat != 397319)
				{
					if ((int)pixelFormat == 925707)
					{
						goto IL_00a3;
					}
					if ((int)pixelFormat != 1052676)
					{
						goto IL_00ac;
					}
				}
				goto IL_009d;
			}
			if ((int)pixelFormat <= 1851406)
			{
				if ((int)pixelFormat == 1060876)
				{
					return 48;
				}
				if ((int)pixelFormat == 1851406)
				{
					goto IL_00a9;
				}
			}
			else
			{
				if ((int)pixelFormat == 2498570)
				{
					goto IL_00a3;
				}
				if ((int)pixelFormat == 3424269)
				{
					goto IL_00a9;
				}
			}
		}
		goto IL_00ac;
		IL_00a3:
		return 32;
		IL_00a9:
		return 64;
		IL_009d:
		return 16;
		IL_00ac:
		throw new NotSupportedException($"A pixel format '{pixelFormat}' not supported!");
	}

	public static ushort GetColorCount(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_001e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0024: Invalid comparison between Unknown and I4
		//IL_000d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0026: Unknown result type (might be due to invalid IL or missing references)
		//IL_002c: Invalid comparison between Unknown and I4
		//IL_002e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0034: Invalid comparison between Unknown and I4
		//IL_0048: Unknown result type (might be due to invalid IL or missing references)
		if (!pixelFormat.IsIndexed())
		{
			throw new NotSupportedException($"Cannot retrieve color count for a non-indexed format '{pixelFormat}'.");
		}
		if ((int)pixelFormat != 196865)
		{
			if ((int)pixelFormat != 197634)
			{
				if ((int)pixelFormat == 198659)
				{
					return 256;
				}
				throw new NotSupportedException($"A pixel format '{pixelFormat}' not supported!");
			}
			return 16;
		}
		return 2;
	}

	public static string GetFriendlyName(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_0055: Unknown result type (might be due to invalid IL or missing references)
		//IL_005b: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_0077: Unknown result type (might be due to invalid IL or missing references)
		//IL_007d: Invalid comparison between Unknown and I4
		//IL_005d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0063: Invalid comparison between Unknown and I4
		//IL_0038: Unknown result type (might be due to invalid IL or missing references)
		//IL_003e: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Unknown result type (might be due to invalid IL or missing references)
		//IL_0018: Invalid comparison between Unknown and I4
		//IL_0091: Unknown result type (might be due to invalid IL or missing references)
		//IL_0097: Invalid comparison between Unknown and I4
		//IL_007f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0085: Invalid comparison between Unknown and I4
		//IL_0065: Unknown result type (might be due to invalid IL or missing references)
		//IL_006b: Invalid comparison between Unknown and I4
		//IL_0040: Unknown result type (might be due to invalid IL or missing references)
		//IL_0046: Invalid comparison between Unknown and I4
		//IL_001d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0023: Invalid comparison between Unknown and I4
		//IL_0099: Unknown result type (might be due to invalid IL or missing references)
		//IL_009f: Invalid comparison between Unknown and I4
		//IL_0087: Unknown result type (might be due to invalid IL or missing references)
		//IL_008d: Invalid comparison between Unknown and I4
		//IL_006d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0073: Invalid comparison between Unknown and I4
		//IL_0048: Unknown result type (might be due to invalid IL or missing references)
		//IL_004e: Invalid comparison between Unknown and I4
		//IL_0028: Unknown result type (might be due to invalid IL or missing references)
		//IL_002e: Invalid comparison between Unknown and I4
		//IL_00ea: Unknown result type (might be due to invalid IL or missing references)
		if ((int)pixelFormat <= 198659)
		{
			if ((int)pixelFormat <= 139273)
			{
				if (pixelFormat - 135173 <= 1)
				{
					return "Highcolor (65536 colors)";
				}
				if ((int)pixelFormat == 137224)
				{
					return "Truecolor (24-bit)";
				}
				if ((int)pixelFormat == 139273)
				{
					return "Truecolor (32-bit)";
				}
			}
			else
			{
				if ((int)pixelFormat == 196865)
				{
					return "Indexed (2 colors)";
				}
				if ((int)pixelFormat == 197634)
				{
					return "Indexed (16 colors)";
				}
				if ((int)pixelFormat == 198659)
				{
					return "Indexed (256 colors)";
				}
			}
		}
		else if ((int)pixelFormat <= 1052676)
		{
			if ((int)pixelFormat == 397319)
			{
				return "Highcolor + Alpha mask (32768 colors)";
			}
			if ((int)pixelFormat == 925707)
			{
				goto IL_00cd;
			}
			if ((int)pixelFormat == 1052676)
			{
				return "Grayscale (65536 shades)";
			}
		}
		else if ((int)pixelFormat <= 1851406)
		{
			if ((int)pixelFormat == 1060876)
			{
				return "Truecolor (48-bit)";
			}
			if ((int)pixelFormat == 1851406)
			{
				goto IL_00df;
			}
		}
		else
		{
			if ((int)pixelFormat == 2498570)
			{
				goto IL_00cd;
			}
			if ((int)pixelFormat == 3424269)
			{
				goto IL_00df;
			}
		}
		throw new NotSupportedException($"A pixel format '{pixelFormat}' not supported!");
		IL_00df:
		return "Truecolor + Alpha (64-bit)";
		IL_00cd:
		return "Truecolor + Alpha (32-bit)";
	}

	public static bool IsIndexed(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Unknown result type (might be due to invalid IL or missing references)
		//IL_000c: Invalid comparison between Unknown and I4
		return (pixelFormat & 0x10000) == 65536;
	}

	public static bool IsSupported(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_0046: Unknown result type (might be due to invalid IL or missing references)
		//IL_004c: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_0068: Unknown result type (might be due to invalid IL or missing references)
		//IL_006e: Invalid comparison between Unknown and I4
		//IL_004e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0054: Invalid comparison between Unknown and I4
		//IL_002c: Unknown result type (might be due to invalid IL or missing references)
		//IL_0032: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Unknown result type (might be due to invalid IL or missing references)
		//IL_0018: Invalid comparison between Unknown and I4
		//IL_0070: Unknown result type (might be due to invalid IL or missing references)
		//IL_0076: Invalid comparison between Unknown and I4
		//IL_0056: Unknown result type (might be due to invalid IL or missing references)
		//IL_005c: Invalid comparison between Unknown and I4
		//IL_0034: Unknown result type (might be due to invalid IL or missing references)
		//IL_003a: Invalid comparison between Unknown and I4
		//IL_001a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0020: Invalid comparison between Unknown and I4
		//IL_0078: Unknown result type (might be due to invalid IL or missing references)
		//IL_007e: Invalid comparison between Unknown and I4
		//IL_005e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0064: Invalid comparison between Unknown and I4
		//IL_003c: Unknown result type (might be due to invalid IL or missing references)
		//IL_0042: Invalid comparison between Unknown and I4
		//IL_0022: Unknown result type (might be due to invalid IL or missing references)
		//IL_0028: Invalid comparison between Unknown and I4
		if ((int)pixelFormat <= 198659)
		{
			if ((int)pixelFormat <= 139273)
			{
				if (pixelFormat - 135173 <= 1 || (int)pixelFormat == 137224 || (int)pixelFormat == 139273)
				{
					goto IL_0080;
				}
			}
			else if ((int)pixelFormat == 196865 || (int)pixelFormat == 197634 || (int)pixelFormat == 198659)
			{
				goto IL_0080;
			}
		}
		else if ((int)pixelFormat <= 1060876)
		{
			if ((int)pixelFormat == 397319 || (int)pixelFormat == 925707 || (int)pixelFormat == 1060876)
			{
				goto IL_0080;
			}
		}
		else if ((int)pixelFormat == 1851406 || (int)pixelFormat == 2498570 || (int)pixelFormat == 3424269)
		{
			goto IL_0080;
		}
		return false;
		IL_0080:
		return true;
	}

	public static PixelFormat GetFormatByColorCount(int colorCount)
	{
		//IL_0027: Unknown result type (might be due to invalid IL or missing references)
		//IL_0032: Unknown result type (might be due to invalid IL or missing references)
		//IL_003f: Unknown result type (might be due to invalid IL or missing references)
		//IL_003e: Unknown result type (might be due to invalid IL or missing references)
		if (colorCount <= 0 || colorCount > 256)
		{
			throw new NotSupportedException($"A color count '{colorCount}' not supported!");
		}
		PixelFormat result = (PixelFormat)196865;
		if (colorCount > 16)
		{
			result = (PixelFormat)198659;
		}
		else if (colorCount > 2)
		{
			result = (PixelFormat)197634;
		}
		return result;
	}

	public static bool HasAlpha(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Unknown result type (might be due to invalid IL or missing references)
		//IL_000c: Invalid comparison between Unknown and I4
		//IL_000e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0014: Unknown result type (might be due to invalid IL or missing references)
		//IL_001a: Invalid comparison between Unknown and I4
		if ((pixelFormat & 0x40000) != 262144)
		{
			return (pixelFormat & 0x80000) == 524288;
		}
		return true;
	}

	public static bool IsDeepColor(this PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_001a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0020: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_0022: Unknown result type (might be due to invalid IL or missing references)
		//IL_0028: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Invalid comparison between Unknown and I4
		if ((int)pixelFormat <= 1060876)
		{
			if ((int)pixelFormat == 1052676 || (int)pixelFormat == 1060876)
			{
				goto IL_002a;
			}
		}
		else if ((int)pixelFormat == 1851406 || (int)pixelFormat == 3424269)
		{
			goto IL_002a;
		}
		return false;
		IL_002a:
		return true;
	}
}
