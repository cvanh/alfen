using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.XiaolinWu;

public class WuColorQuantizer : BaseColorQuantizer
{
	private const int MaxColor = 512;

	private const int Red = 2;

	private const int Green = 1;

	private const int Blue = 0;

	private const int SideSize = 33;

	private const int MaxSideIndex = 32;

	private const int MaxVolume = 35937;

	private int[] reds;

	private int[] greens;

	private int[] blues;

	private int[] sums;

	private int[] indices;

	private long[,,] weights;

	private long[,,] momentsRed;

	private long[,,] momentsGreen;

	private long[,,] momentsBlue;

	private float[,,] moments;

	private int[] tag;

	private int[] quantizedPixels;

	private int[] table;

	private int[] pixels;

	private int imageWidth;

	private int imageSize;

	private int pixelIndex;

	private WuColorCube[] cubes;

	public override bool AllowParallel => false;

	private void CalculateMoments()
	{
		long[] array = new long[33];
		long[] array2 = new long[33];
		long[] array3 = new long[33];
		long[] array4 = new long[33];
		float[] array5 = new float[33];
		for (int i = 1; i <= 32; i++)
		{
			for (int j = 0; j <= 32; j++)
			{
				array[j] = 0L;
				array2[j] = 0L;
				array3[j] = 0L;
				array4[j] = 0L;
				array5[j] = 0f;
			}
			for (int k = 1; k <= 32; k++)
			{
				long num = 0L;
				long num2 = 0L;
				long num3 = 0L;
				long num4 = 0L;
				float num5 = 0f;
				for (int l = 1; l <= 32; l++)
				{
					num += weights[i, k, l];
					num2 += momentsRed[i, k, l];
					num3 += momentsGreen[i, k, l];
					num4 += momentsBlue[i, k, l];
					num5 += moments[i, k, l];
					array[l] += num;
					array2[l] += num2;
					array3[l] += num3;
					array4[l] += num4;
					array5[l] += num5;
					weights[i, k, l] = weights[i - 1, k, l] + array[l];
					momentsRed[i, k, l] = momentsRed[i - 1, k, l] + array2[l];
					momentsGreen[i, k, l] = momentsGreen[i - 1, k, l] + array3[l];
					momentsBlue[i, k, l] = momentsBlue[i - 1, k, l] + array4[l];
					moments[i, k, l] = moments[i - 1, k, l] + array5[l];
				}
			}
		}
	}

	private static long Volume(WuColorCube cube, long[,,] moment)
	{
		return moment[cube.RedMaximum, cube.GreenMaximum, cube.BlueMaximum] - moment[cube.RedMaximum, cube.GreenMaximum, cube.BlueMinimum] - moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMaximum] + moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMinimum] - moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMaximum] + moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMinimum] + moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMaximum] - moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMinimum];
	}

	private static float VolumeFloat(WuColorCube cube, float[,,] moment)
	{
		return moment[cube.RedMaximum, cube.GreenMaximum, cube.BlueMaximum] - moment[cube.RedMaximum, cube.GreenMaximum, cube.BlueMinimum] - moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMaximum] + moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMinimum] - moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMaximum] + moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMinimum] + moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMaximum] - moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMinimum];
	}

	private static long Top(WuColorCube cube, int direction, int position, long[,,] moment)
	{
		return direction switch
		{
			2 => moment[position, cube.GreenMaximum, cube.BlueMaximum] - moment[position, cube.GreenMaximum, cube.BlueMinimum] - moment[position, cube.GreenMinimum, cube.BlueMaximum] + moment[position, cube.GreenMinimum, cube.BlueMinimum], 
			1 => moment[cube.RedMaximum, position, cube.BlueMaximum] - moment[cube.RedMaximum, position, cube.BlueMinimum] - moment[cube.RedMinimum, position, cube.BlueMaximum] + moment[cube.RedMinimum, position, cube.BlueMinimum], 
			0 => moment[cube.RedMaximum, cube.GreenMaximum, position] - moment[cube.RedMaximum, cube.GreenMinimum, position] - moment[cube.RedMinimum, cube.GreenMaximum, position] + moment[cube.RedMinimum, cube.GreenMinimum, position], 
			_ => 0L, 
		};
	}

	private static long Bottom(WuColorCube cube, int direction, long[,,] moment)
	{
		return direction switch
		{
			2 => -moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMaximum] + moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMinimum] + moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMaximum] - moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMinimum], 
			1 => -moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMaximum] + moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMinimum] + moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMaximum] - moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMinimum], 
			0 => -moment[cube.RedMaximum, cube.GreenMaximum, cube.BlueMinimum] + moment[cube.RedMaximum, cube.GreenMinimum, cube.BlueMinimum] + moment[cube.RedMinimum, cube.GreenMaximum, cube.BlueMinimum] - moment[cube.RedMinimum, cube.GreenMinimum, cube.BlueMinimum], 
			_ => 0L, 
		};
	}

	private float CalculateVariance(WuColorCube cube)
	{
		float num = Volume(cube, momentsRed);
		float num2 = Volume(cube, momentsGreen);
		float num3 = Volume(cube, momentsBlue);
		float num4 = VolumeFloat(cube, moments);
		float num5 = Volume(cube, weights);
		float num6 = num * num + num2 * num2 + num3 * num3;
		return num4 - num6 / num5;
	}

	private float Maximize(WuColorCube cube, int direction, int first, int last, IList<int> cut, long wholeRed, long wholeGreen, long wholeBlue, long wholeWeight)
	{
		long num = Bottom(cube, direction, momentsRed);
		long num2 = Bottom(cube, direction, momentsGreen);
		long num3 = Bottom(cube, direction, momentsBlue);
		long num4 = Bottom(cube, direction, weights);
		float num5 = 0f;
		cut[0] = -1;
		for (int i = first; i < last; i++)
		{
			long num6 = num + Top(cube, direction, i, momentsRed);
			long num7 = num2 + Top(cube, direction, i, momentsGreen);
			long num8 = num3 + Top(cube, direction, i, momentsBlue);
			long num9 = num4 + Top(cube, direction, i, weights);
			if (num9 == 0L)
			{
				continue;
			}
			float num10 = num6 * num6 + num7 * num7 + num8 * num8;
			float num11 = num10 / (float)num9;
			num6 = wholeRed - num6;
			num7 = wholeGreen - num7;
			num8 = wholeBlue - num8;
			num9 = wholeWeight - num9;
			if (num9 != 0L)
			{
				num10 = num6 * num6 + num7 * num7 + num8 * num8;
				num11 += num10 / (float)num9;
				if (num11 > num5)
				{
					num5 = num11;
					cut[0] = i;
				}
			}
		}
		return num5;
	}

	private bool Cut(WuColorCube first, WuColorCube second)
	{
		int[] array = new int[1];
		int[] array2 = new int[1];
		int[] array3 = new int[1];
		long wholeRed = Volume(first, momentsRed);
		long wholeGreen = Volume(first, momentsGreen);
		long wholeBlue = Volume(first, momentsBlue);
		long wholeWeight = Volume(first, weights);
		float num = Maximize(first, 2, first.RedMinimum + 1, first.RedMaximum, array, wholeRed, wholeGreen, wholeBlue, wholeWeight);
		float num2 = Maximize(first, 1, first.GreenMinimum + 1, first.GreenMaximum, array2, wholeRed, wholeGreen, wholeBlue, wholeWeight);
		float num3 = Maximize(first, 0, first.BlueMinimum + 1, first.BlueMaximum, array3, wholeRed, wholeGreen, wholeBlue, wholeWeight);
		int num4;
		if (!(num >= num2) || !(num >= num3))
		{
			num4 = ((num2 >= num && num2 >= num3) ? 1 : 0);
		}
		else
		{
			num4 = 2;
			if (array[0] < 0)
			{
				return false;
			}
		}
		second.RedMaximum = first.RedMaximum;
		second.GreenMaximum = first.GreenMaximum;
		second.BlueMaximum = first.BlueMaximum;
		switch (num4)
		{
		case 2:
		{
			int blueMinimum = (first.RedMaximum = array[0]);
			second.RedMinimum = blueMinimum;
			second.GreenMinimum = first.GreenMinimum;
			second.BlueMinimum = first.BlueMinimum;
			break;
		}
		case 1:
		{
			int blueMinimum = (first.GreenMaximum = array2[0]);
			second.GreenMinimum = blueMinimum;
			second.RedMinimum = first.RedMinimum;
			second.BlueMinimum = first.BlueMinimum;
			break;
		}
		case 0:
		{
			int blueMinimum = (first.BlueMaximum = array3[0]);
			second.BlueMinimum = blueMinimum;
			second.RedMinimum = first.RedMinimum;
			second.GreenMinimum = first.GreenMinimum;
			break;
		}
		}
		first.Volume = (first.RedMaximum - first.RedMinimum) * (first.GreenMaximum - first.GreenMinimum) * (first.BlueMaximum - first.BlueMinimum);
		second.Volume = (second.RedMaximum - second.RedMinimum) * (second.GreenMaximum - second.GreenMinimum) * (second.BlueMaximum - second.BlueMinimum);
		return true;
	}

	private static void Mark(WuColorCube cube, int label, IList<int> tag)
	{
		for (int i = cube.RedMinimum + 1; i <= cube.RedMaximum; i++)
		{
			for (int j = cube.GreenMinimum + 1; j <= cube.GreenMaximum; j++)
			{
				for (int k = cube.BlueMinimum + 1; k <= cube.BlueMaximum; k++)
				{
					tag[(i << 10) + (i << 6) + i + (j << 5) + j + k] = label;
				}
			}
		}
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		cubes = new WuColorCube[512];
		for (int i = 0; i < 512; i++)
		{
			cubes[i] = new WuColorCube();
		}
		cubes[0].RedMinimum = 0;
		cubes[0].GreenMinimum = 0;
		cubes[0].BlueMinimum = 0;
		cubes[0].RedMaximum = 32;
		cubes[0].GreenMaximum = 32;
		cubes[0].BlueMaximum = 32;
		weights = new long[33, 33, 33];
		momentsRed = new long[33, 33, 33];
		momentsGreen = new long[33, 33, 33];
		momentsBlue = new long[33, 33, 33];
		moments = new float[33, 33, 33];
		table = new int[256];
		for (int j = 0; j < 256; j++)
		{
			table[j] = j * j;
		}
		pixelIndex = 0;
		imageWidth = image.Width;
		imageSize = image.Width * image.Height;
		quantizedPixels = new int[imageSize];
		pixels = new int[imageSize];
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		int num = (color.R >> 3) + 1;
		int num2 = (color.G >> 3) + 1;
		int num3 = (color.B >> 3) + 1;
		weights[num, num2, num3]++;
		momentsRed[num, num2, num3] += color.R;
		momentsGreen[num, num2, num3] += color.G;
		momentsBlue[num, num2, num3] += color.B;
		moments[num, num2, num3] += table[color.R] + table[color.G] + table[color.B];
		quantizedPixels[pixelIndex] = (num << 10) + (num << 6) + num + (num2 << 5) + num2 + num3;
		pixels[pixelIndex] = color.ToArgb();
		pixelIndex++;
	}

	protected override List<Color> OnGetPalette(int colorCount)
	{
		CalculateMoments();
		int num = 0;
		float[] array = new float[512];
		for (int i = 1; i < colorCount; i++)
		{
			if (Cut(cubes[num], cubes[i]))
			{
				array[num] = ((cubes[num].Volume > 1) ? CalculateVariance(cubes[num]) : 0f);
				array[i] = ((cubes[i].Volume > 1) ? CalculateVariance(cubes[i]) : 0f);
			}
			else
			{
				array[num] = 0f;
				i--;
			}
			num = 0;
			float num2 = array[0];
			for (int j = 1; j <= i; j++)
			{
				if (array[j] > num2)
				{
					num2 = array[j];
					num = j;
				}
			}
			if ((double)num2 <= 0.0)
			{
				colorCount = i + 1;
				break;
			}
		}
		int[] array2 = new int[512];
		int[] array3 = new int[512];
		int[] array4 = new int[512];
		tag = new int[35937];
		for (int k = 0; k < colorCount; k++)
		{
			Mark(cubes[k], k, tag);
			long num3 = Volume(cubes[k], weights);
			if (num3 > 0)
			{
				array2[k] = (int)(Volume(cubes[k], momentsRed) / num3);
				array3[k] = (int)(Volume(cubes[k], momentsGreen) / num3);
				array4[k] = (int)(Volume(cubes[k], momentsBlue) / num3);
			}
			else
			{
				array2[k] = 0;
				array3[k] = 0;
				array4[k] = 0;
			}
		}
		for (int l = 0; l < imageSize; l++)
		{
			quantizedPixels[l] = tag[quantizedPixels[l]];
		}
		reds = new int[colorCount + 1];
		greens = new int[colorCount + 1];
		blues = new int[colorCount + 1];
		sums = new int[colorCount + 1];
		indices = new int[imageSize];
		for (int m = 0; m < imageSize; m++)
		{
			Color color = Color.FromArgb(pixels[m]);
			int num4 = quantizedPixels[m];
			int num5 = 100000000;
			for (int n = 0; n < colorCount; n++)
			{
				int num6 = array2[n];
				int num7 = array3[n];
				int num8 = array4[n];
				int num9 = color.R - num6;
				int num10 = color.G - num7;
				int num11 = color.B - num8;
				int num12 = num9 * num9 + num10 * num10 + num11 * num11;
				if (num12 < num5)
				{
					num5 = num12;
					num4 = n;
				}
			}
			reds[num4] += color.R;
			greens[num4] += color.G;
			blues[num4] += color.B;
			sums[num4]++;
			indices[m] = num4;
		}
		List<Color> list = new List<Color>();
		for (int num13 = 0; num13 < colorCount; num13++)
		{
			if (sums[num13] > 0)
			{
				reds[num13] /= sums[num13];
				greens[num13] /= sums[num13];
				blues[num13] /= sums[num13];
			}
			Color item = Color.FromArgb(255, reds[num13], greens[num13], blues[num13]);
			list.Add(item);
		}
		pixelIndex = 0;
		return list;
	}

	protected override void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		paletteIndex = indices[x + y * imageWidth];
	}

	protected override void OnFinish()
	{
		base.OnFinish();
		cubes = null;
		weights = null;
		momentsRed = null;
		momentsGreen = null;
		momentsBlue = null;
		moments = null;
		quantizedPixels = null;
		pixels = null;
	}
}
