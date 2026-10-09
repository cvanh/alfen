using System.ComponentModel;
using System.Drawing;
using ICUNetwork;

namespace ICUServiceInstaller;

public class UploadResourceWorkerData
{
	public ICUDevice CurrentDevice { get; set; }

	public BackgroundWorker Worker { get; set; }

	public Image Image { get; set; }

	public string Filename { get; set; }

	public string SpecialLanguage { get; set; }

	public int Margin { get; set; }
}
