using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using System.Linq;
using System.Threading;
using SimplePaletteQuantizer.Helpers;
using SimplePaletteQuantizer.PathProviders;

namespace SimplePaletteQuantizer.Quantizers;

public abstract class BaseColorQuantizer : IColorQuantizer, IPathProvider
{
	protected const int InvalidIndex = -1;

	private bool paletteFound;

	private long uniqueColorIndex;

	private IPathProvider pathProvider;

	protected readonly ConcurrentDictionary<int, short> UniqueColors;

	public abstract bool AllowParallel { get; }

	protected BaseColorQuantizer()
	{
		pathProvider = null;
		uniqueColorIndex = -1L;
		UniqueColors = new ConcurrentDictionary<int, short>();
	}

	public void ChangePathProvider(IPathProvider pathProvider)
	{
		this.pathProvider = pathProvider;
	}

	private IPathProvider GetPathProvider()
	{
		IPathProvider pathProvider = this.pathProvider ?? (this.pathProvider = OnCreateDefaultPathProvider());
		if (pathProvider == null)
		{
			throw new ArgumentNullException(string.Format("The path provider is not initialized! Please use SetPathProvider() method on quantizer.", Array.Empty<object>()));
		}
		return pathProvider;
	}

	protected virtual void OnPrepare(ImageBuffer image)
	{
		uniqueColorIndex = -1L;
		paletteFound = false;
		UniqueColors.Clear();
	}

	protected virtual void OnAddColor(Color color, int key, int x, int y)
	{
		UniqueColors.AddOrUpdate(key, (int colorKey) => (byte)Interlocked.Increment(ref uniqueColorIndex), (int colorKey, short colorIndex) => colorIndex);
	}

	protected virtual IPathProvider OnCreateDefaultPathProvider()
	{
		pathProvider = new StandardPathProvider();
		return new StandardPathProvider();
	}

	protected virtual List<Color> OnGetPalette(int colorCount)
	{
		if (UniqueColors.Count > 0 && colorCount >= UniqueColors.Count)
		{
			paletteFound = true;
			return (from pair in UniqueColors
				orderby pair.Value
				select Color.FromArgb(pair.Key) into color
				select Color.FromArgb(255, color.R, color.G, color.B)).ToList();
		}
		return null;
	}

	protected virtual void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		paletteIndex = -1;
		if (paletteFound && UniqueColors.TryGetValue(key, out var value))
		{
			paletteIndex = value;
		}
	}

	protected virtual int OnGetColorCount()
	{
		return UniqueColors.Count;
	}

	protected virtual void OnFinish()
	{
	}

	public IList<Point> GetPointPath(int width, int heigth)
	{
		return GetPathProvider().GetPointPath(width, heigth);
	}

	public void Prepare(ImageBuffer image)
	{
		OnPrepare(image);
	}

	public void AddColor(Color color, int x, int y)
	{
		color = QuantizationHelper.ConvertAlpha(color, out var argb);
		OnAddColor(color, argb, x, y);
	}

	public int GetColorCount()
	{
		return OnGetColorCount();
	}

	public List<Color> GetPalette(int colorCount)
	{
		return OnGetPalette(colorCount);
	}

	public int GetPaletteIndex(Color color, int x, int y)
	{
		color = QuantizationHelper.ConvertAlpha(color, out var argb);
		OnGetPaletteIndex(color, argb, x, y, out var paletteIndex);
		return paletteIndex;
	}

	public void Finish()
	{
		OnFinish();
	}
}
