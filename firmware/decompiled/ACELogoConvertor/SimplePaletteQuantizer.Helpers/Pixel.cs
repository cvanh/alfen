using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.Linq.Expressions;
using System.Runtime.InteropServices;
using SimplePaletteQuantizer.Helpers.Pixels;
using SimplePaletteQuantizer.Helpers.Pixels.Indexed;
using SimplePaletteQuantizer.Helpers.Pixels.NonIndexed;

namespace SimplePaletteQuantizer.Helpers;

public class Pixel : IDisposable
{
	internal const byte Zero = 0;

	internal const byte One = 1;

	internal const byte Two = 2;

	internal const byte Four = 4;

	internal const byte Eight = 8;

	internal const byte NibbleMask = 15;

	internal const byte ByteMask = byte.MaxValue;

	internal const int AlphaShift = 24;

	internal const int RedShift = 16;

	internal const int GreenShift = 8;

	internal const int BlueShift = 0;

	internal const int AlphaMask = -16777216;

	internal const int RedGreenBlueMask = 16777215;

	private Type pixelType;

	private int bitOffset;

	private object pixelData;

	private IntPtr pixelDataPointer;

	public int X { get; private set; }

	public int Y { get; private set; }

	public ImageBuffer Parent { get; private set; }

	public byte Index
	{
		get
		{
			return ((IIndexedPixel)pixelData).GetIndex(bitOffset);
		}
		set
		{
			((IIndexedPixel)pixelData).SetIndex(bitOffset, value);
		}
	}

	public Color Color
	{
		get
		{
			return ((INonIndexedPixel)pixelData).GetColor();
		}
		set
		{
			((INonIndexedPixel)pixelData).SetColor(value);
		}
	}

	public bool IsIndexed => Parent.IsIndexed;

	public Pixel(ImageBuffer parent)
	{
		Parent = parent;
		Initialize();
	}

	private void Initialize()
	{
		//IL_0023: Unknown result type (might be due to invalid IL or missing references)
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		pixelType = (IsIndexed ? GetIndexedType(Parent.PixelFormat) : GetNonIndexedType(Parent.PixelFormat));
		Expression<Func<object>> expression = Expression.Lambda<Func<object>>(Expression.Convert(Expression.New(pixelType), typeof(object)), Array.Empty<ParameterExpression>());
		pixelData = expression.Compile()();
		pixelDataPointer = MarshalToPointer(pixelData);
	}

	internal Type GetIndexedType(PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Invalid comparison between Unknown and I4
		//IL_0040: Unknown result type (might be due to invalid IL or missing references)
		if ((int)pixelFormat != 196865)
		{
			if ((int)pixelFormat != 197634)
			{
				if ((int)pixelFormat == 198659)
				{
					return typeof(PixelData8Indexed);
				}
				throw new NotSupportedException($"This pixel format '{pixelFormat}' is either non-indexed, or not supported.");
			}
			return typeof(PixelData4Indexed);
		}
		return typeof(PixelData1Indexed);
	}

	internal Type GetNonIndexedType(PixelFormat pixelFormat)
	{
		//IL_0000: Unknown result type (might be due to invalid IL or missing references)
		//IL_0006: Invalid comparison between Unknown and I4
		//IL_003a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0040: Invalid comparison between Unknown and I4
		//IL_0008: Unknown result type (might be due to invalid IL or missing references)
		//IL_000e: Invalid comparison between Unknown and I4
		//IL_0054: Unknown result type (might be due to invalid IL or missing references)
		//IL_005a: Invalid comparison between Unknown and I4
		//IL_0042: Unknown result type (might be due to invalid IL or missing references)
		//IL_0048: Invalid comparison between Unknown and I4
		//IL_0025: Unknown result type (might be due to invalid IL or missing references)
		//IL_002b: Invalid comparison between Unknown and I4
		//IL_0010: Unknown result type (might be due to invalid IL or missing references)
		//IL_0016: Invalid comparison between Unknown and I4
		//IL_005c: Unknown result type (might be due to invalid IL or missing references)
		//IL_0062: Invalid comparison between Unknown and I4
		//IL_004a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0050: Invalid comparison between Unknown and I4
		//IL_002d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0033: Invalid comparison between Unknown and I4
		//IL_0018: Unknown result type (might be due to invalid IL or missing references)
		//IL_001e: Invalid comparison between Unknown and I4
		//IL_0064: Unknown result type (might be due to invalid IL or missing references)
		//IL_006a: Invalid comparison between Unknown and I4
		//IL_00d6: Unknown result type (might be due to invalid IL or missing references)
		if ((int)pixelFormat <= 139273)
		{
			if ((int)pixelFormat <= 135174)
			{
				if ((int)pixelFormat == 135173)
				{
					return typeof(PixelDataRgb555);
				}
				if ((int)pixelFormat == 135174)
				{
					return typeof(PixelDataRgb565);
				}
			}
			else
			{
				if ((int)pixelFormat == 137224)
				{
					return typeof(PixelDataRgb888);
				}
				if ((int)pixelFormat == 139273)
				{
					return typeof(PixelDataRgb8888);
				}
			}
		}
		else if ((int)pixelFormat <= 1052676)
		{
			if ((int)pixelFormat == 397319)
			{
				return typeof(PixelDataArgb1555);
			}
			if ((int)pixelFormat == 1052676)
			{
				return typeof(PixelDataGray16);
			}
		}
		else
		{
			if ((int)pixelFormat == 1060876)
			{
				return typeof(PixelDataRgb48);
			}
			if ((int)pixelFormat == 2498570)
			{
				return typeof(PixelDataArgb8888);
			}
			if ((int)pixelFormat == 3424269)
			{
				return typeof(PixelDataArgb64);
			}
		}
		throw new NotSupportedException($"This pixel format '{pixelFormat}' is either indexed, or not supported.");
	}

	private static IntPtr MarshalToPointer(object data)
	{
		IntPtr intPtr = Marshal.AllocHGlobal(Marshal.SizeOf(data));
		Marshal.StructureToPtr(data, intPtr, fDeleteOld: false);
		return intPtr;
	}

	public void Update(int x, int y)
	{
		X = x;
		Y = y;
		bitOffset = Parent.GetBitOffset(x);
	}

	public void ReadRawData(IntPtr imagePointer)
	{
		pixelData = Marshal.PtrToStructure(imagePointer, pixelType);
	}

	public void ReadData(byte[] buffer, int offset)
	{
		Marshal.Copy(buffer, offset, pixelDataPointer, Parent.BytesPerPixel);
		pixelData = Marshal.PtrToStructure(pixelDataPointer, pixelType);
	}

	public void WriteRawData(IntPtr imagePointer)
	{
		Marshal.StructureToPtr(pixelData, imagePointer, fDeleteOld: false);
	}

	public void WriteData(byte[] buffer, int offset)
	{
		Marshal.Copy(pixelDataPointer, buffer, offset, Parent.BytesPerPixel);
	}

	public void Dispose()
	{
		Marshal.FreeHGlobal(pixelDataPointer);
	}
}
