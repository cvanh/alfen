namespace SimplePaletteQuantizer.Helpers;

public class FastRandom
{
	private const double RealUnitInt = 4.656612873077393E-10;

	private uint x;

	private uint y;

	private uint z;

	private uint w;

	public FastRandom(uint seed)
	{
		x = seed;
		y = 842502087u;
		z = 3579807591u;
		w = 273326509u;
	}

	public int Next(int upperBound)
	{
		uint num = x ^ (x << 11);
		x = y;
		y = z;
		z = w;
		return (int)(4.656612873077393E-10 * (double)(int)(0x7FFFFFFF & (w = w ^ (w >> 19) ^ (num ^ (num >> 8)))) * (double)upperBound);
	}
}
