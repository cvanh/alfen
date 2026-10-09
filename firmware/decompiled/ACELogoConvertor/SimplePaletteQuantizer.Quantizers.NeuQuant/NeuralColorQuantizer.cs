using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.NeuQuant;

public class NeuralColorQuantizer : BaseColorQuantizer
{
	private const byte DefaultQuality = 10;

	private const int AlphaBiasShift = 10;

	private const int AlphaRadiusBias = 262144;

	private const int AlphaRadiusBetaShift = 18;

	private const int Beta = 64;

	private const int BetaShift = 10;

	private const int BetaGamma = 65536;

	private const int DefaultRadius = 32;

	private const int DefaultRadiusBiasShift = 6;

	private const int DefaultRadiusBias = 64;

	private const int GammaShift = 10;

	private const int InitialAlpha = 1024;

	private const int InitialBias = 65536;

	private const int InitialBiasShift = 16;

	private const int InitialRadius = 2048;

	private const int MaximalNetworkPosition = 255;

	private const int NetworkSize = 256;

	private const int NetworkBiasShift = 4;

	private const int RadiusBiasShift = 8;

	private const int RadiusDecrease = 30;

	private const int RadiusBias = 256;

	private readonly FastRandom random;

	private readonly ConcurrentDictionary<int, bool> uniqueColors;

	private int[] bias;

	private int[] frequency;

	private int[] networkIndexLookup;

	private int[] radiusPower;

	private byte quality;

	private int delta;

	private int radius;

	private int alpha;

	private int initialRadius;

	private int alphaDecrease;

	private int[][] network;

	public byte Quality
	{
		get
		{
			return quality;
		}
		set
		{
			quality = value;
			alphaDecrease = 30 + (quality - 1);
		}
	}

	public override bool AllowParallel => true;

	public NeuralColorQuantizer()
	{
		Quality = 10;
		random = new FastRandom(0u);
		uniqueColors = new ConcurrentDictionary<int, bool>();
	}

	public NeuralColorQuantizer(byte quality)
		: this()
	{
		Quality = quality;
	}

	private int FindClosestNeuron(int red, int green, int blue)
	{
		int num = -1;
		int num2 = int.MaxValue;
		int result = num;
		int num3 = num2;
		for (int i = 0; i < 256; i++)
		{
			int[] obj = network[i];
			int num4 = obj[2] - red;
			int num5 = obj[1] - green;
			int num6 = obj[0] - blue;
			if (num4 < 0)
			{
				num4 = -num4;
			}
			if (num5 < 0)
			{
				num5 = -num5;
			}
			if (num6 < 0)
			{
				num6 = -num6;
			}
			int num7 = num4 + num5 + num6;
			if (num7 < num2)
			{
				num2 = num7;
				num = i;
			}
			int num8 = num7 - (bias[i] >> 12);
			if (num8 < num3)
			{
				num3 = num8;
				result = i;
			}
			int num9 = frequency[i] >> 10;
			frequency[i] -= num9;
			bias[i] += num9 << 10;
		}
		frequency[num] += 64;
		bias[num] -= 65536;
		return result;
	}

	private void LearnNeuron(int alpha, int red, int green, int blue, int networkIndex)
	{
		int[] array = network[networkIndex];
		array[2] -= alpha * (array[2] - red) / 1024;
		array[1] -= alpha * (array[1] - green) / 1024;
		array[0] -= alpha * (array[0] - blue) / 1024;
	}

	protected void LearnNeuronNeighbors(int red, int green, int blue, int networkIndex, int radius)
	{
		int num = networkIndex - radius;
		if (num < -1)
		{
			num = -1;
		}
		int num2 = networkIndex + radius;
		if (num2 > 256)
		{
			num2 = 256;
		}
		int num3 = networkIndex + 1;
		int num4 = networkIndex - 1;
		int num5 = 1;
		while (num3 < num2 || num4 > num)
		{
			int num6 = radiusPower[num5++];
			if (num3 < num2)
			{
				int[] array = network[num3++];
				array[0] -= num6 * (array[0] - blue) / 262144;
				array[1] -= num6 * (array[1] - green) / 262144;
				array[2] -= num6 * (array[2] - red) / 262144;
			}
			if (num4 > num)
			{
				int[] array = network[num4--];
				array[0] -= num6 * (array[0] - blue) / 262144;
				array[1] -= num6 * (array[1] - green) / 262144;
				array[2] -= num6 * (array[2] - red) / 262144;
			}
		}
	}

	private void UnbiasNetwork()
	{
		for (int i = 0; i < 256; i++)
		{
			network[i][0] >>= 4;
			network[i][1] >>= 4;
			network[i][2] >>= 4;
			network[i][3] = i;
		}
	}

	private void SortNetwork()
	{
		int num = 0;
		int num2 = 0;
		for (int i = 0; i < 256; i++)
		{
			int[] array = network[i];
			int num3 = i;
			int num4 = array[1];
			for (int j = i + 1; j < 256; j++)
			{
				int[] array2 = network[j];
				if (array2[1] < num4)
				{
					num3 = j;
					num4 = array2[1];
				}
			}
			if (i != num3)
			{
				int[] array3 = network[num3];
				for (int k = 0; k < 4; k++)
				{
					int num5 = array3[k];
					array3[k] = array[k];
					array[k] = num5;
				}
			}
			if (num4 != num2)
			{
				networkIndexLookup[num2] = num + i >> 1;
				for (int l = num2 + 1; l < num4; l++)
				{
					networkIndexLookup[l] = i;
				}
				num2 = num4;
				num = i;
			}
		}
		networkIndexLookup[num2] = num + 255 >> 1;
		for (int m = num2 + 1; m < 256; m++)
		{
			networkIndexLookup[m] = 255;
		}
	}

	private void LearnSampleColor(Color color)
	{
		int red = color.R << 4;
		int green = color.G << 4;
		int blue = color.B << 4;
		int networkIndex = FindClosestNeuron(red, green, blue);
		LearnNeuron(alpha, red, green, blue, networkIndex);
		if (radius != 0)
		{
			LearnNeuronNeighbors(red, green, blue, networkIndex, radius);
		}
		if (delta == 0)
		{
			delta = 1;
		}
		alpha -= alpha / alphaDecrease;
		initialRadius -= initialRadius / 30;
		radius = initialRadius >> 6;
		if (radius <= 1)
		{
			radius = 0;
		}
		int num = radius * radius;
		for (int i = 0; i < radius; i++)
		{
			int num2 = i * i;
			radiusPower[i] = alpha * ((num - num2) * 256 / num);
		}
	}

	protected override void OnPrepare(ImageBuffer image)
	{
		base.OnPrepare(image);
		OnFinish();
		network = new int[256][];
		uniqueColors.Clear();
		for (int i = 0; i < 256; i++)
		{
			int[] array = new int[4];
			array[2] = (array[1] = (array[0] = (i << 12) / 256));
			bias[i] = 0;
			network[i] = array;
			frequency[i] = 256;
		}
		alpha = 1024;
		initialRadius = 2048;
		int num = 32;
		radius = ((num > 1) ? num : 0);
		int num2 = radius * radius;
		for (int j = 0; j < radius; j++)
		{
			int num3 = j * j;
			radiusPower[j] = alpha * ((num2 - num3) * 256 / num2);
		}
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		base.OnAddColor(color, key, x, y);
		if (random.Next(10) == 0)
		{
			LearnSampleColor(color);
		}
	}

	protected override List<Color> OnGetPalette(int colorCount)
	{
		UnbiasNetwork();
		SortNetwork();
		int[] array = new int[256];
		for (int i = 0; i < 256; i++)
		{
			array[network[i][3]] = i;
		}
		List<Color> list = new List<Color>();
		for (int j = 0; j < 256; j++)
		{
			int num = array[j];
			int red = network[num][2];
			int green = network[num][1];
			int blue = network[num][0];
			Color item = Color.FromArgb(255, red, green, blue);
			list.Add(item);
		}
		return list;
	}

	protected override void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		int num = 1000;
		int num2 = networkIndexLookup[color.G];
		int num3 = num2 - 1;
		paletteIndex = -1;
		while (num2 < 256 || num3 >= 0)
		{
			if (num2 < 256)
			{
				int[] array = network[num2];
				int num4 = array[1] - color.G;
				if (num4 < 0)
				{
					num4 = -num4;
				}
				int num5 = num4;
				if (num5 >= num)
				{
					num2 = 256;
				}
				else
				{
					num2++;
					int num6 = array[0] - color.B;
					if (num6 < 0)
					{
						num6 = -num6;
					}
					num5 += num6;
					if (num5 < num)
					{
						int num7 = array[2] - color.R;
						if (num7 < 0)
						{
							num7 = -num7;
						}
						num5 += num7;
						if (num5 < num)
						{
							num = num5;
							paletteIndex = array[3];
						}
					}
				}
			}
			if (num3 < 0)
			{
				continue;
			}
			int[] array2 = network[num3];
			int num8 = color.G - array2[1];
			if (num8 < 0)
			{
				num8 = -num8;
			}
			int num9 = num8;
			if (num9 >= num)
			{
				num3 = -1;
				continue;
			}
			num3--;
			int num10 = array2[0] - color.B;
			if (num10 < 0)
			{
				num10 = -num10;
			}
			num9 += num10;
			if (num9 < num)
			{
				int num11 = array2[2] - color.R;
				if (num11 < 0)
				{
					num11 = -num11;
				}
				num9 += num11;
				if (num9 < num)
				{
					num = num9;
					paletteIndex = array2[3];
				}
			}
		}
	}

	protected override void OnFinish()
	{
		base.OnFinish();
		bias = new int[256];
		frequency = new int[256];
		networkIndexLookup = new int[256];
		radiusPower = new int[32];
		network = null;
	}
}
