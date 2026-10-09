namespace SimplePaletteQuantizer.Quantizers.Uniform;

internal struct UniformColorSlot
{
	private int value;

	private int pixelCount;

	public void AddValue(int component)
	{
		value += component;
		pixelCount++;
	}

	public int GetAverage()
	{
		int result = 0;
		if (pixelCount > 0)
		{
			result = ((pixelCount == 1) ? value : (value / pixelCount));
		}
		return result;
	}
}
