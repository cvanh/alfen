namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class SierraDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[5, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 5, 3 },
			{ 2, 4, 5, 4, 2 },
			{ 0, 2, 3, 2, 0 }
		};
	}
}
