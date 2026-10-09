using System.ComponentModel;
using ICUNetwork;

namespace ICUServiceInstaller;

public class TransactionWorkerData
{
	public ICULanDevice CurrentDevice { get; set; }

	public BackgroundWorker Worker { get; set; }

	public string Filename { get; set; }
}
