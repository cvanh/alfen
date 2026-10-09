namespace ICUNetwork.Helpers;

internal class ProgressHelper
{
	protected double[] _steps;

	protected double[] _tresholds;

	protected int _currtentStage;

	internal ProgressHelper(bool isAhp)
	{
		if (isAhp)
		{
			_steps = new double[4] { 0.1, 0.1, 0.25, 0.3 };
			_tresholds = new double[4] { 4.0, 8.0, 97.0, 100.0 };
		}
		else
		{
			_steps = new double[3] { 0.25, 0.5, 0.5 };
			_tresholds = new double[3] { 50.0, 97.0, 100.0 };
		}
	}

	internal double GetNextStage()
	{
		if (_currtentStage < _steps.Length - 1)
		{
			return _tresholds[_currtentStage++];
		}
		return 0.0;
	}

	internal double GetProgress(double progress)
	{
		double num = progress + _steps[_currtentStage];
		if (num >= _tresholds[_currtentStage])
		{
			return progress;
		}
		return num;
	}
}
