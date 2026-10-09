namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class AtkinsonDithering : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 2;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[5, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 1, 1 },
			{ 0, 1, 1, 1, 0 },
			{ 0, 0, 1, 0, 0 }
		};
	}
}
