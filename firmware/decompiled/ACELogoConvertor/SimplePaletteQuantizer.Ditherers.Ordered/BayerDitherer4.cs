namespace SimplePaletteQuantizer.Ditherers.Ordered;

public class BayerDitherer4 : BaseOrderedDitherer
{
	protected override byte MatrixWidth => 4;

	protected override byte MatrixHeight => 4;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[4, 4]
		{
			{ 1, 9, 3, 11 },
			{ 13, 5, 15, 7 },
			{ 4, 12, 2, 10 },
			{ 16, 8, 14, 6 }
		};
	}
}
