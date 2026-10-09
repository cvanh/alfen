namespace ICUServiceInstaller.Utils;

public static class Flags
{
	public static int UpdateFlag(int currentValue, int flag, bool value)
	{
		if ((currentValue & flag) == flag != value)
		{
			return currentValue ^ flag;
		}
		return currentValue;
	}
}
