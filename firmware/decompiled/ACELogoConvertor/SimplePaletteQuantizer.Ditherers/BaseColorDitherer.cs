using System;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;
using SimplePaletteQuantizer.PathProviders;
using SimplePaletteQuantizer.Quantizers;

namespace SimplePaletteQuantizer.Ditherers;

public abstract class BaseColorDitherer : IColorDitherer, IPathProvider
{
	private IPathProvider pathProvider;

	protected int ColorCount { get; private set; }

	protected ImageBuffer SourceBuffer { get; private set; }

	protected ImageBuffer TargetBuffer { get; private set; }

	protected IColorQuantizer Quantizer { get; private set; }

	protected byte[,] CachedMatrix { get; private set; }

	protected float[,] CachedSummedMatrix { get; private set; }

	public abstract bool IsInplace { get; }

	public void ChangePathProvider(IPathProvider pathProvider)
	{
		this.pathProvider = pathProvider;
	}

	private int GetMatrixFactor()
	{
		int num = 0;
		for (int i = 0; i < CachedMatrix.GetLength(0); i++)
		{
			for (int j = 0; j < CachedMatrix.GetLength(1); j++)
			{
				int num2 = CachedMatrix[i, j];
				if (num2 > num)
				{
					num = num2;
				}
			}
		}
		return num;
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

	protected int GetClampedColorElementWithError(int colorElement, float factor, int error)
	{
		int colorElement2 = Convert.ToInt32((float)colorElement + factor * (float)error);
		return GetClampedColorElement(colorElement2);
	}

	protected int GetClampedColorElement(int colorElement)
	{
		int num = colorElement;
		if (num < 0)
		{
			num = 0;
		}
		if (num > 255)
		{
			num = 255;
		}
		return num;
	}

	protected virtual IPathProvider OnCreateDefaultPathProvider()
	{
		return new StandardPathProvider();
	}

	protected virtual void OnPrepare()
	{
		CachedMatrix = CreateCoeficientMatrix();
		float num = GetMatrixFactor();
		int length = CachedMatrix.GetLength(1);
		int length2 = CachedMatrix.GetLength(0);
		CachedSummedMatrix = new float[length2, length];
		for (int i = 0; i < length2; i++)
		{
			for (int j = 0; j < length; j++)
			{
				CachedSummedMatrix[i, j] = (float)(int)CachedMatrix[i, j] / num;
			}
		}
	}

	protected abstract byte[,] CreateCoeficientMatrix();

	protected abstract bool OnProcessPixel(Pixel sourcePixel, Pixel targetPixel);

	protected virtual void OnFinish()
	{
	}

	public IList<Point> GetPointPath(int width, int heigth)
	{
		return GetPathProvider().GetPointPath(width, heigth);
	}

	public void Prepare(IColorQuantizer quantizer, int colorCount, ImageBuffer sourceBuffer, ImageBuffer targetBuffer)
	{
		SourceBuffer = sourceBuffer;
		TargetBuffer = targetBuffer;
		ColorCount = colorCount;
		Quantizer = quantizer;
		OnPrepare();
	}

	public IList<Point> GetPointPath()
	{
		return null;
	}

	public bool ProcessPixel(Pixel sourcePixel, Pixel targetPixel)
	{
		return OnProcessPixel(sourcePixel, targetPixel);
	}

	public void Finish()
	{
		OnFinish();
	}
}
