namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class TwoRowSierraDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[3, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 4, 3 },
			{ 1, 2, 3, 2, 1 }
		};
	}
}
