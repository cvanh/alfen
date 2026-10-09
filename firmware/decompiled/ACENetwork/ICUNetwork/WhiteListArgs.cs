using System;
using System.Collections.ObjectModel;

namespace ICUNetwork;

public class WhiteListArgs : EventArgs
{
	public ICULanDevice Device { get; private set; }

	public ObservableCollection<ICUWhitelistItem> Whitelist { get; private set; }

	public bool CancellationPending { get; private set; }

	public WhiteListArgs(ICULanDevice lanDevice, ObservableCollection<ICUWhitelistItem> whitelist, bool cancellationPending = false)
	{
		Device = lanDevice;
		Whitelist = new ObservableCollection<ICUWhitelistItem>(whitelist);
		CancellationPending = cancellationPending;
	}
}
