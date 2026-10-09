using System.Collections.ObjectModel;
using System.Net.NetworkInformation;

namespace ICUNetwork;

public class BaseConnection
{
	protected string m_sName = "<base>";

	public ObservableCollection<ICUDevice> Devices { get; set; }

	public ObservableCollection<NetworkInterface> NetworkInterfaces { get; set; } = new ObservableCollection<NetworkInterface>();

	public BaseConnection(ObservableCollection<ICUDevice> lstDevices)
	{
		Devices = lstDevices;
	}

	public virtual void Initialize()
	{
	}

	public virtual void StopBrowsing()
	{
	}

	public virtual void StartBrowsing()
	{
	}
}
