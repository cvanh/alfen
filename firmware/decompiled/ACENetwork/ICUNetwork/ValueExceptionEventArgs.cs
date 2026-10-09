using System;

namespace ICUNetwork;

public class ValueExceptionEventArgs : EventArgs
{
	public Exception Exception { get; set; }

	public string InvalidValue { get; set; }
}
