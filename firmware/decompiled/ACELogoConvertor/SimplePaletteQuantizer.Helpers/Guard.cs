using System;

namespace SimplePaletteQuantizer.Helpers;

public static class Guard
{
	public static void CheckNull(object argument, string argumentName)
	{
		if (argument == null)
		{
			throw new ArgumentNullException($"Cannot use '{argumentName}' when it is null!");
		}
	}
}
