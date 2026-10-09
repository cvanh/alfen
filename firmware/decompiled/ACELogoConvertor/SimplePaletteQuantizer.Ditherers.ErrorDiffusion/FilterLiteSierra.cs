namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class FilterLiteSierra : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 1;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[3, 3]
		{
			{ 0, 0, 0 },
			{ 0, 0, 2 },
			{ 1, 1, 0 }
		};
	}
}
