using System.ComponentModel;
using ICUNetwork;

namespace ICUServiceInstaller;

public class UploadWorkerData
{
	public ICUDevice CurrentDevice { get; set; }

	public BackgroundWorker Worker { get; set; }

	public string Filename { get; set; }
}
