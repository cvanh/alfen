namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class FanDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 3;

	protected override int MatrixSideHeight => 1;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[3, 7]
		{
			{ 0, 0, 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 0, 8, 0, 0 },
			{ 1, 1, 2, 4, 0, 0, 0 }
		};
	}
}
