using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;
using System.Threading.Tasks;
using SimplePaletteQuantizer.ColorCaches.Common;
using SimplePaletteQuantizer.Ditherers;
using SimplePaletteQuantizer.Extensions;
using SimplePaletteQuantizer.PathProviders;
using SimplePaletteQuantizer.Quantizers;

namespace SimplePaletteQuantizer.Helpers;

public class ImageBuffer : IDisposable
{
	public delegate bool ProcessPixelFunction(Pixel pixel);

	public delegate bool ProcessPixelAdvancedFunction(Pixel pixel, ImageBuffer buffer);

	public delegate bool TransformPixelFunction(Pixel sourcePixel, Pixel targetPixel);

	public delegate bool TransformPixelAdvancedFunction(Pixel sourcePixel, Pixel targetPixel, ImageBuffer sourceBuffer, ImageBuffer targetBuffer);

	private class LineTask
	{
		public int StartOffset { get; private set; }

		public int EndOffset { get; private set; }

		public LineTask(int startOffset, int endOffset)
		{
			StartOffset = startOffset;
			EndOffset = endOffset;
		}
	}

	private int[] fastBitX;

	private int[] fastByteX;

	private int[] fastY;

	private readonly Bitmap bitmap;

	private readonly BitmapData bitmapData;

	private readonly ImageLockMode lockMode;

	private List<Color> cachedPalette;

	public int Width { get; private set; }

	public int Height { get; private set; }

	public int Size { get; private set; }

	public int Stride { get; private set; }

	public int BitDepth { get; private set; }

	public int BytesPerPixel { get; private set; }

	public bool IsIndexed { get; private set; }

	public PixelFormat PixelFormat
	{
		[CompilerGenerated]
		get
		{
			//IL_0001: Unknown result type (might be due to invalid IL or missing references)
			return field;
		}
		[CompilerGenerated]
		private set
		{
			//IL_0001: Unknown result type (might be due to invalid IL or missing references)
			//IL_0002: Unknown result type (might be due to invalid IL or missing references)
			field = value;
		}
	}

	public bool CanRead
	{
		get
		{
			//IL_0001: Unknown result type (might be due to invalid IL or missing references)
			//IL_0007: Invalid comparison between Unknown and I4
			//IL_000a: Unknown result type (might be due to invalid IL or missing references)
			//IL_0010: Invalid comparison between Unknown and I4
			if ((int)lockMode != 1)
			{
				return (int)lockMode == 3;
			}
			return true;
		}
	}

	public bool CanWrite
	{
		get
		{
			//IL_0001: Unknown result type (might be due to invalid IL or missing references)
			//IL_0007: Invalid comparison between Unknown and I4
			//IL_000a: Unknown result type (might be due to invalid IL or missing references)
			//IL_0010: Invalid comparison between Unknown and I4
			if ((int)lockMode != 2)
			{
				return (int)lockMode == 3;
			}
			return true;
		}
	}

	public List<Color> Palette
	{
		get
		{
			return UpdatePalette();
		}
		set
		{
			((Image)(object)bitmap).SetPalette(value);
			cachedPalette = value;
		}
	}

	public ImageBuffer(Image bitmap, ImageLockMode lockMode)
		: this((Bitmap)bitmap, lockMode)
	{
		//IL_0002: Unknown result type (might be due to invalid IL or missing references)
		//IL_0007: Unknown result type (might be due to invalid IL or missing references)
		//IL_000d: Expected Obj, but got Unknown
	}

	public ImageBuffer(Bitmap bitmap, ImageLockMode lockMode)
	{
		//IL_000e: Unknown result type (might be due to invalid IL or missing references)
		//IL_000f: Unknown result type (might be due to invalid IL or missing references)
		//IL_002e: Unknown result type (might be due to invalid IL or missing references)
		//IL_003a: Unknown result type (might be due to invalid IL or missing references)
		//IL_004b: Unknown result type (might be due to invalid IL or missing references)
		//IL_0091: Unknown result type (might be due to invalid IL or missing references)
		//IL_0093: Unknown result type (might be due to invalid IL or missing references)
		this.bitmap = bitmap;
		this.lockMode = lockMode;
		Width = ((Image)bitmap).Width;
		Height = ((Image)bitmap).Height;
		PixelFormat = ((Image)bitmap).PixelFormat;
		IsIndexed = PixelFormat.IsIndexed();
		BitDepth = PixelFormat.GetBitDepth();
		BytesPerPixel = Math.Max(1, BitDepth >> 3);
		Rectangle rectangle = Rectangle.FromLTRB(0, 0, Width, Height);
		lock (bitmap)
		{
			bitmapData = bitmap.LockBits(rectangle, lockMode, PixelFormat);
		}
		Stride = ((bitmapData.Stride < 0) ? (-bitmapData.Stride) : bitmapData.Stride);
		Size = Stride * Height;
		Precalculate();
	}

	private void Precalculate()
	{
		fastBitX = new int[Width];
		fastByteX = new int[Width];
		fastY = new int[Height];
		for (int i = 0; i < Width; i++)
		{
			fastBitX[i] = i * BitDepth;
			fastByteX[i] = fastBitX[i] >> 3;
			fastBitX[i] %= 8;
		}
		for (int j = 0; j < Height; j++)
		{
			fastY[j] = j * bitmapData.Stride;
		}
	}

	public int GetBitOffset(int x)
	{
		return fastBitX[x];
	}

	public byte[] Copy()
	{
		byte[] array = new byte[Size];
		Marshal.Copy(bitmapData.Scan0, array, 0, Size);
		return array;
	}

	public void Paste(byte[] buffer)
	{
		Marshal.Copy(buffer, 0, bitmapData.Scan0, Size);
	}

	public void ReadPixel(Pixel pixel, byte[] buffer = null)
	{
		int num = fastByteX[pixel.X] + fastY[pixel.Y];
		if (buffer == null)
		{
			pixel.ReadRawData(bitmapData.Scan0 + num);
		}
		else
		{
			pixel.ReadData(buffer, num);
		}
	}

	public int GetIndexFromPixel(Pixel pixel)
	{
		if (IsIndexed)
		{
			return pixel.Index;
		}
		throw new NotSupportedException(string.Format("Cannot retrieve index for a non-indexed format. Please use Color (or Value) property instead.", Array.Empty<object>()));
	}

	public Color GetColorFromPixel(Pixel pixel)
	{
		if (pixel.IsIndexed)
		{
			int index = pixel.Index;
			return pixel.Parent.GetPaletteColor(index);
		}
		return pixel.Color;
	}

	public int ReadIndexUsingPixel(Pixel pixel, byte[] buffer = null)
	{
		ReadPixel(pixel, buffer);
		return GetIndexFromPixel(pixel);
	}

	public Color ReadColorUsingPixel(Pixel pixel, byte[] buffer = null)
	{
		ReadPixel(pixel, buffer);
		return GetColorFromPixel(pixel);
	}

	public int ReadIndexUsingPixelFrom(Pixel pixel, int x, int y, byte[] buffer = null)
	{
		pixel.Update(x, y);
		return ReadIndexUsingPixel(pixel, buffer);
	}

	public Color ReadColorUsingPixelFrom(Pixel pixel, int x, int y, byte[] buffer = null)
	{
		pixel.Update(x, y);
		return ReadColorUsingPixel(pixel, buffer);
	}

	private void WritePixel(Pixel pixel, byte[] buffer = null)
	{
		int num = fastByteX[pixel.X] + fastY[pixel.Y];
		if (buffer == null)
		{
			pixel.WriteRawData(bitmapData.Scan0 + num);
		}
		else
		{
			pixel.WriteData(buffer, num);
		}
	}

	public void SetIndexToPixel(Pixel pixel, int index, byte[] buffer = null)
	{
		if (IsIndexed)
		{
			pixel.Index = (byte)index;
			return;
		}
		throw new NotSupportedException(string.Format("Cannot set index for a non-indexed format. Please use Color (or Value) property instead.", Array.Empty<object>()));
	}

	public void SetColorToPixel(Pixel pixel, Color color, IColorQuantizer quantizer)
	{
		if (pixel.IsIndexed)
		{
			if (quantizer == null)
			{
				throw new NotSupportedException(string.Format("Cannot retrieve color for an indexed format. Use GetPixelIndex() instead.", Array.Empty<object>()));
			}
			byte index = (byte)quantizer.GetPaletteIndex(color, pixel.X, pixel.Y);
			pixel.Index = index;
		}
		else
		{
			pixel.Color = color;
		}
	}

	public void WriteIndexUsingPixel(Pixel pixel, int index, byte[] buffer = null)
	{
		SetIndexToPixel(pixel, index, buffer);
		WritePixel(pixel, buffer);
	}

	public void WriteColorUsingPixel(Pixel pixel, Color color, IColorQuantizer quantizer, byte[] buffer = null)
	{
		SetColorToPixel(pixel, color, quantizer);
		WritePixel(pixel, buffer);
	}

	public void WriteIndexUsingPixelAt(Pixel pixel, int x, int y, int index, byte[] buffer = null)
	{
		pixel.Update(x, y);
		WriteIndexUsingPixel(pixel, index, buffer);
	}

	public void WriteColorUsingPixelAt(Pixel pixel, int x, int y, Color color, IColorQuantizer quantizer, byte[] buffer = null)
	{
		pixel.Update(x, y);
		WriteColorUsingPixel(pixel, color, quantizer, buffer);
	}

	private void ProcessInParallel(ICollection<Point> path, Action<LineTask> process, int parallelTaskCount = 4)
	{
		Guard.CheckNull(process, "process");
		UpdatePalette();
		double num = 1.0 * (double)path.Count / (double)parallelTaskCount;
		LineTask[] array = new LineTask[parallelTaskCount];
		double num2 = 0.0;
		for (int i = 0; i < parallelTaskCount; i++)
		{
			array[i] = new LineTask((int)num2, (int)(num2 + num));
			num2 += num;
		}
		Parallel.ForEach(array, process);
	}

	private void ProcessPerPixelBase(IList<Point> path, Delegate processingAction, int parallelTaskCount = 4)
	{
		Guard.CheckNull(path, "path");
		Guard.CheckNull(processingAction, "processPixelFunction");
		bool isAdvanced = processingAction is ProcessPixelAdvancedFunction;
		Action<LineTask> process = (LineTask lineTask) =>
		{
			Pixel pixel = new Pixel(this);
			for (int i = lineTask.StartOffset; i < lineTask.EndOffset; i++)
			{
				Point point = path[i];
				pixel.Update(point.X, point.Y);
				if (CanRead)
				{
					ReadPixel(pixel);
				}
				bool flag = ((!isAdvanced) ? ((ProcessPixelFunction)processingAction)(pixel) : ((ProcessPixelAdvancedFunction)processingAction)(pixel, this));
				if (CanWrite & flag)
				{
					WritePixel(pixel);
				}
			}
		};
		ProcessInParallel(path, process, parallelTaskCount);
	}

	public void ProcessPerPixel(IList<Point> path, ProcessPixelFunction processPixelFunction, int parallelTaskCount = 4)
	{
		ProcessPerPixelBase(path, processPixelFunction, parallelTaskCount);
	}

	public void ProcessPerPixelAdvanced(IList<Point> path, ProcessPixelAdvancedFunction processPixelAdvancedFunction, int parallelTaskCount = 4)
	{
		ProcessPerPixelBase(path, processPixelAdvancedFunction, parallelTaskCount);
	}

	private void TransformPerPixelBase(ImageBuffer target, IList<Point> path, Delegate transformAction, int parallelTaskCount = 4)
	{
		Guard.CheckNull(path, "path");
		Guard.CheckNull(target, "target");
		Guard.CheckNull(transformAction, "transformAction");
		UpdatePalette();
		target.UpdatePalette();
		if (Width != target.Width || Height != target.Height)
		{
			throw new ArgumentOutOfRangeException("Both images have to have the same dimensions.");
		}
		bool isAdvanced = transformAction is TransformPixelAdvancedFunction;
		Action<LineTask> process = (LineTask lineTask) =>
		{
			Pixel pixel = new Pixel(this);
			Pixel pixel2 = new Pixel(target);
			for (int i = lineTask.StartOffset; i < lineTask.EndOffset; i++)
			{
				Point point = path[i];
				pixel.Update(point.X, point.Y);
				pixel2.Update(point.X, point.Y);
				if (CanRead)
				{
					ReadPixel(pixel);
				}
				if (target.CanRead)
				{
					target.ReadPixel(pixel2);
				}
				bool flag = ((!isAdvanced) ? ((TransformPixelFunction)transformAction)(pixel, pixel2) : ((TransformPixelAdvancedFunction)transformAction)(pixel, pixel2, this, target));
				if (target.CanWrite & flag)
				{
					target.WritePixel(pixel2);
				}
			}
		};
		ProcessInParallel(path, process, parallelTaskCount);
	}

	public void TransformPerPixel(ImageBuffer target, IList<Point> path, TransformPixelFunction transformPixelFunction, int parallelTaskCount = 4)
	{
		TransformPerPixelBase(target, path, transformPixelFunction, parallelTaskCount);
	}

	public void TransformPerPixelAdvanced(ImageBuffer target, IList<Point> path, TransformPixelAdvancedFunction transformPixelAdvancedFunction, int parallelTaskCount = 4)
	{
		TransformPerPixelBase(target, path, transformPixelAdvancedFunction, parallelTaskCount);
	}

	public void ScanColors(IColorQuantizer quantizer, int parallelTaskCount = 4)
	{
		Guard.CheckNull(quantizer, "quantizer");
		IList<Point> pointPath = quantizer.GetPointPath(Width, Height);
		ProcessPixelFunction processPixelFunction = (Pixel pixel) =>
		{
			quantizer.AddColor(GetColorFromPixel(pixel), pixel.X, pixel.Y);
			return false;
		};
		ProcessPerPixel(pointPath, processPixelFunction, parallelTaskCount);
	}

	public static void ScanImageColors(Image sourceImage, IColorQuantizer quantizer, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		imageBuffer.ScanColors(quantizer, parallelTaskCount);
	}

	public List<Color> SynthetizePalette(IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(quantizer, "quantizer");
		quantizer.Prepare(this);
		ScanColors(quantizer, parallelTaskCount);
		return quantizer.GetPalette(colorCount);
	}

	public static List<Color> SynthetizeImagePalette(Image sourceImage, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		return imageBuffer.SynthetizePalette(quantizer, colorCount, parallelTaskCount);
	}

	public void Quantize(ImageBuffer target, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Quantize(target, quantizer, null, colorCount, parallelTaskCount);
	}

	public void Quantize(ImageBuffer target, IColorQuantizer quantizer, IColorDitherer ditherer, int colorCount, int parallelTaskCount = 4)
	{
		//IL_0037: Unknown result type (might be due to invalid IL or missing references)
		Guard.CheckNull(target, "target");
		Guard.CheckNull(quantizer, "quantizer");
		List<Color> palette = (target.PixelFormat.IsIndexed() ? SynthetizePalette(quantizer, colorCount, parallelTaskCount) : null);
		((Image)(object)target.bitmap).SetPalette(palette);
		target.UpdatePalette(forceUpdate: true);
		if (ditherer != null)
		{
			ditherer.Prepare(quantizer, colorCount, this, target);
		}
		TransformPixelFunction transformPixelFunction = (Pixel sourcePixel, Pixel targetPixel) =>
		{
			Color colorFromPixel = GetColorFromPixel(sourcePixel);
			colorFromPixel = QuantizationHelper.ConvertAlpha(colorFromPixel);
			SetColorToPixel(targetPixel, colorFromPixel, quantizer);
			bool result = true;
			if (ditherer != null && ditherer.IsInplace)
			{
				result = ditherer.ProcessPixel(sourcePixel, targetPixel);
			}
			return result;
		};
		IList<Point> pointPath = quantizer.GetPointPath(Width, Height);
		TransformPerPixel(target, pointPath, transformPixelFunction, parallelTaskCount);
		if (ditherer != null && !ditherer.IsInplace)
		{
			Dither(target, ditherer, quantizer, colorCount, 1);
		}
		if (ditherer != null)
		{
			ditherer.Finish();
		}
		quantizer.Finish();
	}

	public static Image QuantizeImage(ImageBuffer source, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		return QuantizeImage(source, quantizer, null, colorCount, parallelTaskCount);
	}

	public static Image QuantizeImage(ImageBuffer source, IColorQuantizer quantizer, IColorDitherer ditherer, int colorCount, int parallelTaskCount = 4)
	{
		//IL_000c: Unknown result type (might be due to invalid IL or missing references)
		//IL_0011: Unknown result type (might be due to invalid IL or missing references)
		//IL_001e: Unknown result type (might be due to invalid IL or missing references)
		//IL_001f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0025: Expected Obj, but got Unknown
		//IL_002c: Unknown result type (might be due to invalid IL or missing references)
		//IL_002e: Unknown result type (might be due to invalid IL or missing references)
		Guard.CheckNull(source, "source");
		PixelFormat formatByColorCount = Extend.GetFormatByColorCount(colorCount);
		Image result = (Image)new Bitmap(source.Width, source.Height, formatByColorCount);
		ImageLockMode val = (ImageLockMode)((ditherer == null) ? 2 : 3);
		using ImageBuffer target = new ImageBuffer(result, val);
		source.Quantize(target, quantizer, ditherer, colorCount, parallelTaskCount);
		return result;
	}

	public static Image QuantizeImage(Image sourceImage, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		return QuantizeImage(sourceImage, quantizer, null, colorCount, parallelTaskCount);
	}

	public static Image QuantizeImage(Image sourceImage, IColorQuantizer quantizer, IColorDitherer ditherer, int colorCount, int parallelTaskCount = 4)
	{
		//IL_0012: Unknown result type (might be due to invalid IL or missing references)
		//IL_0014: Unknown result type (might be due to invalid IL or missing references)
		Guard.CheckNull(sourceImage, "sourceImage");
		ImageLockMode val = (ImageLockMode)((ditherer == null) ? 1 : 3);
		using ImageBuffer source = new ImageBuffer(sourceImage, val);
		return QuantizeImage(source, quantizer, ditherer, colorCount, parallelTaskCount);
	}

	public double CalculateMeanError(ImageBuffer target, int parallelTaskCount = 4)
	{
		Guard.CheckNull(target, "target");
		long totalError = 0L;
		TransformPixelFunction transformPixelFunction = (Pixel sourcePixel, Pixel targetPixel) =>
		{
			Color colorFromPixel = GetColorFromPixel(sourcePixel);
			Color colorFromPixel2 = GetColorFromPixel(targetPixel);
			totalError += ColorModelHelper.GetColorEuclideanDistance(ColorModel.RedGreenBlue, colorFromPixel, colorFromPixel2);
			return false;
		};
		IList<Point> pointPath = new StandardPathProvider().GetPointPath(Width, Height);
		TransformPerPixel(target, pointPath, transformPixelFunction, parallelTaskCount);
		return Math.Sqrt((double)totalError / (3.0 * (double)Width * (double)Height));
	}

	public static double CalculateImageMeanError(ImageBuffer source, ImageBuffer target, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		return source.CalculateMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageMeanError(ImageBuffer source, Image targetImage, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		return source.CalculateMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageMeanError(Image sourceImage, ImageBuffer target, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		return imageBuffer.CalculateMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageMeanError(Image sourceImage, Image targetImage, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		return imageBuffer.CalculateMeanError(target, parallelTaskCount);
	}

	public double CalculateNormalizedMeanError(ImageBuffer target, int parallelTaskCount = 4)
	{
		return CalculateMeanError(target, parallelTaskCount) / 255.0;
	}

	public static double CalculateImageNormalizedMeanError(ImageBuffer source, Image targetImage, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		return source.CalculateNormalizedMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageNormalizedMeanError(Image sourceImage, ImageBuffer target, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		return imageBuffer.CalculateNormalizedMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageNormalizedMeanError(ImageBuffer source, ImageBuffer target, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		return source.CalculateNormalizedMeanError(target, parallelTaskCount);
	}

	public static double CalculateImageNormalizedMeanError(Image sourceImage, Image targetImage, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		return imageBuffer.CalculateNormalizedMeanError(target, parallelTaskCount);
	}

	public void ChangeFormat(ImageBuffer target, IColorQuantizer quantizer, int parallelTaskCount = 4)
	{
		//IL_0031: Unknown result type (might be due to invalid IL or missing references)
		//IL_0042: Unknown result type (might be due to invalid IL or missing references)
		//IL_0052: Unknown result type (might be due to invalid IL or missing references)
		//IL_005e: Unknown result type (might be due to invalid IL or missing references)
		//IL_006f: Unknown result type (might be due to invalid IL or missing references)
		//IL_0088: Unknown result type (might be due to invalid IL or missing references)
		Guard.CheckNull(target, "target");
		Guard.CheckNull(quantizer, "quantizer");
		bool hasSourceAlpha = PixelFormat.HasAlpha();
		bool hasTargetAlpha = target.PixelFormat.HasAlpha();
		bool flag = target.PixelFormat.IsIndexed();
		bool isSourceDeepColor = PixelFormat.IsDeepColor();
		bool isTargetDeepColor = target.PixelFormat.IsDeepColor();
		if (flag)
		{
			SynthetizePalette(quantizer, target.PixelFormat.GetColorCount(), parallelTaskCount);
		}
		TransformPixelFunction transformPixelFunction = (Pixel sourcePixel, Pixel targetPixel) =>
		{
			if (!(isSourceDeepColor & isTargetDeepColor))
			{
				Color color = GetColorFromPixel(sourcePixel);
				if (!hasSourceAlpha & hasTargetAlpha)
				{
					color = Color.FromArgb(-16777216 | (color.R << 16) | (color.G << 8) | color.B);
				}
				SetColorToPixel(targetPixel, color, quantizer);
			}
			return true;
		};
		IList<Point> pointPath = new StandardPathProvider().GetPointPath(Width, Height);
		TransformPerPixel(target, pointPath, transformPixelFunction, parallelTaskCount);
	}

	public static void ChangeFormat(ImageBuffer source, PixelFormat targetFormat, IColorQuantizer quantizer, out Image targetImage, int parallelTaskCount = 4)
	{
		//IL_0018: Unknown result type (might be due to invalid IL or missing references)
		//IL_0019: Unknown result type (might be due to invalid IL or missing references)
		//IL_001f: Expected Obj, but got Unknown
		Guard.CheckNull(source, "source");
		targetImage = (Image)new Bitmap(source.Width, source.Height, targetFormat);
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)2);
		source.ChangeFormat(target, quantizer, parallelTaskCount);
	}

	public static void ChangeFormat(Image sourceImage, PixelFormat targetFormat, IColorQuantizer quantizer, out Image targetImage, int parallelTaskCount = 4)
	{
		//IL_0014: Unknown result type (might be due to invalid IL or missing references)
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer source = new ImageBuffer(sourceImage, (ImageLockMode)1);
		ChangeFormat(source, targetFormat, quantizer, out targetImage, parallelTaskCount);
	}

	public void Dither(ImageBuffer target, IColorDitherer ditherer, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(target, "target");
		Guard.CheckNull(ditherer, "ditherer");
		Guard.CheckNull(quantizer, "quantizer");
		ditherer.Prepare(quantizer, colorCount, this, target);
		IList<Point> pointPath = ditherer.GetPointPath(Width, Height);
		TransformPerPixel(target, pointPath, ditherer.ProcessPixel, parallelTaskCount);
	}

	public static void DitherImage(ImageBuffer source, ImageBuffer target, IColorDitherer ditherer, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		source.Dither(target, ditherer, quantizer, colorCount, parallelTaskCount);
	}

	public static void DitherImage(ImageBuffer source, Image targetImage, IColorDitherer ditherer, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(source, "source");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		source.Dither(target, ditherer, quantizer, colorCount, parallelTaskCount);
	}

	public static void DitherImage(Image sourceImage, ImageBuffer target, IColorDitherer ditherer, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		imageBuffer.Dither(target, ditherer, quantizer, colorCount, parallelTaskCount);
	}

	public static void DitherImage(Image sourceImage, Image targetImage, IColorDitherer ditherer, IColorQuantizer quantizer, int colorCount, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		Guard.CheckNull(targetImage, "targetImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		using ImageBuffer target = new ImageBuffer(targetImage, (ImageLockMode)1);
		imageBuffer.Dither(target, ditherer, quantizer, colorCount, parallelTaskCount);
	}

	public void CorrectGamma(float gamma, IColorQuantizer quantizer, int parallelTaskCount = 4)
	{
		Guard.CheckNull(quantizer, "quantizer");
		IList<Point> pointPath = quantizer.GetPointPath(Width, Height);
		int[] gammaRamp = new int[256];
		for (int i = 0; i < 256; i++)
		{
			gammaRamp[i] = Clamp((int)(255.0 * Math.Pow((float)i / 255f, 1f / gamma) + 0.5));
		}
		ProcessPixelFunction processPixelFunction = (Pixel pixel) =>
		{
			Color colorFromPixel = GetColorFromPixel(pixel);
			int red = gammaRamp[colorFromPixel.R];
			int green = gammaRamp[colorFromPixel.G];
			int blue = gammaRamp[colorFromPixel.B];
			Color color = Color.FromArgb(red, green, blue);
			SetColorToPixel(pixel, color, quantizer);
			return true;
		};
		ProcessPerPixel(pointPath, processPixelFunction, parallelTaskCount);
	}

	public static void CorrectImageGamma(Image sourceImage, float gamma, IColorQuantizer quantizer, int parallelTaskCount = 4)
	{
		Guard.CheckNull(sourceImage, "sourceImage");
		using ImageBuffer imageBuffer = new ImageBuffer(sourceImage, (ImageLockMode)1);
		imageBuffer.CorrectGamma(gamma, quantizer, parallelTaskCount);
	}

	public static int Clamp(int value, int minimum = 0, int maximum = 255)
	{
		if (value < minimum)
		{
			value = minimum;
		}
		if (value > maximum)
		{
			value = maximum;
		}
		return value;
	}

	private List<Color> UpdatePalette(bool forceUpdate = false)
	{
		if (IsIndexed && ((cachedPalette == null) | forceUpdate))
		{
			cachedPalette = ((Image)(object)bitmap).GetPalette();
		}
		return cachedPalette;
	}

	public Color GetPaletteColor(int paletteIndex)
	{
		return cachedPalette[paletteIndex];
	}

	public void Dispose()
	{
		lock (bitmap)
		{
			bitmap.UnlockBits(bitmapData);
		}
	}
}
