namespace SimplePaletteQuantizer.Ditherers.ErrorDiffusion;

public class JarvisJudiceNinkeDitherer : BaseErrorDistributionDitherer
{
	protected override int MatrixSideWidth => 2;

	protected override int MatrixSideHeight => 2;

	protected override byte[,] CreateCoeficientMatrix()
	{
		return new byte[5, 5]
		{
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 0, 0 },
			{ 0, 0, 0, 7, 5 },
			{ 3, 5, 7, 5, 3 },
			{ 1, 3, 5, 3, 1 }
		};
	}
}
