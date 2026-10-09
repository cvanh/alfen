namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class ShiauDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[3, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 4, 0 },
			{ 1, 1, 2, 0, 0 }
		};
	}
}
