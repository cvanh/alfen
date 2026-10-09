namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class BurkesDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[3, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 8, 4 },
			{ 2, 4, 8, 4, 2 }
		};
	}
}
