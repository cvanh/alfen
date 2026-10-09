using System;

namespace ICUNetwork;

public class DeviceEventArgs : EventArgs
{
	public ICUDevice Device { get; set; }

	public DeviceEventArgs(ICUDevice device)
	{
		Device = device;
	}
}
