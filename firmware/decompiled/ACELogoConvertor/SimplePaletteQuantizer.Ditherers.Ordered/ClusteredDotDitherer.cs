namespace SimplePaletteQuantizer.Ditherers.Ordered;

public class ClusteredDotDitherer : BaseOrderedDitherer
{
	protected override byte MatrixWidth => 4;

	protected override byte MatrixHeight => 4;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[4, 4]
		{
			{ 13, 5, 12, 16 },
			{ 6, 0, 4, 11 },
			{ 7, 2, 3, 10 },
			{ 14, 8, 9, 15 }
		};
	}
}
