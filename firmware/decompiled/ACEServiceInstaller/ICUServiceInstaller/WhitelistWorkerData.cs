using System.ComponentModel;
using ICUNetwork;

namespace ICUServiceInstaller;

public class WhitelistWorkerData
{
	public ICULanDevice CurrentDevice { get; set; }

	public BackgroundWorker Worker { get; set; }

	public string Filename { get; set; }
}
