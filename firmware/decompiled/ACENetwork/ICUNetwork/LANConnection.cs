using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Net;
using System.Net.NetworkInformation;
using System.Threading;
using Serilog;
using Tmds.MDns;

namespace ICUNetwork;

public class LANConnection : BaseConnection
{
	public delegate void DeviceEventHandler(object sender, DeviceEventArgs e);

	public delegate void LoginEventHandler(ICULanDevice lanDevice, ACEWebLoginData loginData);

	private readonly ILogger Logger = Log.Logger.ForContext<LANConnection>();

	private readonly int DEFAULT_LOCK_TIMEOUT = 5000;

	private readonly Lazy<ServiceBrowser> m_serviceBrowserLazy;

	private List<ICULanDevice> m_doNotRemoveDevices = new List<ICULanDevice>();

	private readonly List<string> serviceTypes = new List<string> { "_lolo3._http._tcp", "_alfen._tcp" };

	private ServiceBrowser m_serviceBrowser => m_serviceBrowserLazy.Value;

	public bool DeviceFound { get; }

	public event DeviceEventHandler DeviceRegistered;

	public event DeviceEventHandler DeviceReRegistered;

	public event DeviceEventHandler DeviceUnregistered;

	public event LoginEventHandler LoginRequest;

	public event Action<string> ErrorHandler;

	public LANConnection(ObservableCollection<ICUDevice> lstDevices)
		: base(lstDevices)
	{
		m_sName = "Ethernet";
		m_serviceBrowserLazy = new Lazy<ServiceBrowser>(() => new ServiceBrowser());
	}

	public void StartSearch()
	{
		Logger.Verbose("Start browsing for types: {ServiceTypes}", serviceTypes);
		lock (m_serviceBrowser)
		{
			m_serviceBrowser.QueryParameters.Robustness = 10;
			m_serviceBrowser.QueryParameters.ResponseTime = 2000;
			m_serviceBrowser.ServiceAdded += OnServiceAdded;
			m_serviceBrowser.ServiceChanged += OnServiceAdded;
			m_serviceBrowser.ServiceRemoved += OnServiceRemoved;
			m_serviceBrowser.NetworkInterfaceAdded += OnNetworkInterfaceAdded;
			m_serviceBrowser.NetworkInterfaceRemoved += OnNetworkInterfaceRemoved;
			if (!m_serviceBrowser.IsBrowsing)
			{
				m_serviceBrowser.StartBrowse(serviceTypes);
			}
		}
	}

	public void StopSearch()
	{
		Logger.Verbose("Stop browsing for types: {ServiceTypes}", serviceTypes);
		lock (m_serviceBrowser)
		{
			m_serviceBrowser.ServiceAdded -= OnServiceAdded;
			m_serviceBrowser.ServiceChanged -= OnServiceAdded;
			m_serviceBrowser.ServiceRemoved -= OnServiceRemoved;
			m_serviceBrowser.NetworkInterfaceAdded -= OnNetworkInterfaceAdded;
			m_serviceBrowser.NetworkInterfaceRemoved -= OnNetworkInterfaceRemoved;
			if (m_serviceBrowser.IsBrowsing)
			{
				m_serviceBrowser.StopBrowse();
			}
		}
		Devices.Clear();
		lock (NetworkInterfaces)
		{
			NetworkInterfaces.Clear();
		}
	}

	public void CallErrorHandler(string msg, ICULanDevice device)
	{
		if (Devices.OfType<ICULanDevice>().ToList().Exists((ICULanDevice a) => a.HostName != null && a.HostName == device.HostName))
		{
			Logger.Error("Error: {Message} (HostName={HostName}, IP={IPAddress})", msg, device.HostName, device.IPAddress);
			ErrorHandler?.Invoke(msg);
		}
	}

	public void DeviceLoginCallback(ICULanDevice lanDevice, ACEWebLoginData loginData)
	{
		LoginRequest?.Invoke(lanDevice, loginData);
	}

	private void OnServiceRemoved(object sender, ServiceAnnouncementEventArgs e)
	{
		if (!(sender is ServiceBrowser) || !e.Announcement.Addresses.Any() || !Monitor.TryEnter(Devices, DEFAULT_LOCK_TIMEOUT))
		{
			return;
		}
		try
		{
			string propertyValue = e.Announcement.Addresses.First().ToString();
			if (m_doNotRemoveDevices.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == e.Announcement.Hostname) == null)
			{
				ICULanDevice iCULanDevice = Devices.Cast<ICULanDevice>().ToList().FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == e.Announcement.Hostname);
				if (iCULanDevice != null && iCULanDevice.OnDeviceRemoved())
				{
					Logger.Debug("Device removed: {DeviceName} } at {IpAddress}, SCN: {SCNNetwork}", e.Announcement.Hostname.Split(new char[1] { '-' }).Last(), propertyValue, iCULanDevice.SCNNetwork);
					DeviceUnregistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
					Devices.Remove(iCULanDevice);
				}
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(Devices);
		}
	}

	private void OnServiceAdded(object sender, ServiceAnnouncementEventArgs e)
	{
		if (!(sender is ServiceBrowser) || !e.Announcement.Addresses.Any() || !Monitor.TryEnter(Devices, DEFAULT_LOCK_TIMEOUT))
		{
			return;
		}
		try
		{
			string text = e.Announcement.Addresses.First().ToString();
			List<ICULanDevice> source = Devices.Cast<ICULanDevice>().ToList();
			string newHostName = e.Announcement.Hostname;
			ICULanDevice iCULanDevice = source.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName == newHostName);
			if (iCULanDevice == null)
			{
				string objectID = newHostName.Split(new char[1] { '-' }).Last();
				iCULanDevice = source.FirstOrDefault((ICULanDevice a) => a.HostName != null && a.HostName.Split(new char[1] { '-' }).Last() == objectID);
			}
			if (iCULanDevice == null)
			{
				iCULanDevice = new ICULanDevice(this, e.Announcement.Addresses.First(), e.Announcement.Port, e.Announcement);
				iCULanDevice.RegisterLoginCallback(DeviceLoginCallback);
				Devices.Add(iCULanDevice);
				string text2 = e.Announcement.Hostname.Split(new char[1] { '-' })[^1];
				Logger.Debug("Added Device: {SerialNumber} at {IpAddress} SCN: '{SCNNetwork}' using: {Protocol}", text2, text, iCULanDevice.SCNNetwork, iCULanDevice.Protocol);
				Logger.Information("Discovered device: {SerialNumber:X} in SCN: {SCN:X}", text2.GetHashCode(), (!string.IsNullOrEmpty(iCULanDevice.SCNNetwork)) ? iCULanDevice.SCNNetwork.GetHashCode() : 0);
				DeviceRegistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
			}
			else
			{
				iCULanDevice.ReInitialize(e.Announcement.Addresses.First(), e.Announcement.Port, e.Announcement, newDevice: false);
				iCULanDevice.RegisterLoginCallback(DeviceLoginCallback);
				Logger.Debug("Reinitialized Device: {DeviceName} at {IpAddress} SCN: '{SCNNetwork}' using: {Protocol}", e.Announcement.Hostname.Split(new char[1] { '-' }).Last(), text, iCULanDevice.SCNNetwork, iCULanDevice.Protocol);
				DeviceReRegistered?.Invoke(this, new DeviceEventArgs(iCULanDevice));
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(Devices);
		}
	}

	public override void StopBrowsing()
	{
		lock (m_serviceBrowser)
		{
			m_serviceBrowser.StopBrowse();
		}
	}

	public override void StartBrowsing()
	{
		lock (m_serviceBrowser)
		{
			try
			{
				m_doNotRemoveDevices = Devices.Cast<ICULanDevice>().ToList();
				if (m_serviceBrowser.IsBrowsing)
				{
					m_serviceBrowser.StopBrowse();
				}
				m_serviceBrowser.StartBrowse(serviceTypes);
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
		}
	}

	public ICULanDevice AddManualDevice(IPAddress address, int port, string hostName = "", int numberOfSockets = 2, bool LoginRequired = false)
	{
		foreach (ICUDevice device in Devices)
		{
			if (device is ICULanDevice iCULanDevice && iCULanDevice.IPAddress == address && iCULanDevice.Port == port)
			{
				return iCULanDevice;
			}
		}
		ICULanDevice iCULanDevice2 = new ICULanDevice(this, address, port, null, isManuallyAdded: true);
		if (!string.IsNullOrEmpty(hostName))
		{
			iCULanDevice2.SetHostInfo(hostName, numberOfSockets);
		}
		if (LoginRequired)
		{
			iCULanDevice2.IsUniquePasswordRequired = true;
		}
		iCULanDevice2.RegisterLoginCallback(DeviceLoginCallback);
		if (iCULanDevice2.Login().IsLoggedIn)
		{
			iCULanDevice2.UpdateCategories("generic", "generic2");
			iCULanDevice2.NumberOfSockets = iCULanDevice2.GetPropertyInt(8286, 0);
			iCULanDevice2.SocketTypes[0] = iCULanDevice2.GetPropertyInt(8485, 0);
			iCULanDevice2.SocketTypes[1] = ((iCULanDevice2.NumberOfSockets > 1) ? iCULanDevice2.GetPropertyInt(12581, 0) : 0);
			EMeterTypes propertyInt = (EMeterTypes)iCULanDevice2.GetPropertyInt(16919, 0);
			EMeterTypes propertyInt2 = (EMeterTypes)iCULanDevice2.GetPropertyInt(21015, 0);
			iCULanDevice2.HasCentralMeter = propertyInt == EMeterTypes.ENERGYMETER_MODBUS_CENTRAL || propertyInt == EMeterTypes.ENERGYMETER_P1 || propertyInt == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL || propertyInt == EMeterTypes.ENERGYMETER_FKN_METER;
			iCULanDevice2.HasSmartMeter = propertyInt2 == EMeterTypes.ENERGYMETER_P1 || propertyInt2 == EMeterTypes.ENERGYMETER_TCPIP_SMART;
			Logger.Debug("New device firmware: {FirmwareVersionNumber}", iCULanDevice2.FirmwareVersionNumber);
			Devices.Add(iCULanDevice2);
			return iCULanDevice2;
		}
		return null;
	}

	public void RemoveManualDevice(ICUDevice device)
	{
		if (device != null)
		{
			if (device is ICULanDevice iCULanDevice)
			{
				iCULanDevice.Deallocate();
			}
			Devices.Remove(device);
		}
	}

	public void RemoveDevice(ICULanDevice device)
	{
		if (device.OnDeviceRemoved())
		{
			Logger.Debug("Device removed: {DeviceName} at {IpAddress}, SCN: {SCNNetwork}", device.Identification, device.Address, device.SCNNetwork);
			DeviceUnregistered(this, new DeviceEventArgs(device));
			device.Deallocate();
			Devices.Remove(device);
		}
	}

	public ICULanDevice FindLanDevice(IPAddress ipAddress)
	{
		return (Devices?.Cast<ICULanDevice>()).FirstOrDefault((ICULanDevice device) => device.Address.ToLowerInvariant() == ipAddress.ToString().ToLowerInvariant());
	}

	public List<ICULanDevice> FindDevicesInSCN(string scnName)
	{
		return Devices?.Cast<ICULanDevice>().Where((ICULanDevice a) => a.SCNNetwork?.ToLowerInvariant() == scnName.ToString().ToLowerInvariant()).ToList();
	}

	private void OnNetworkInterfaceAdded(object sender, NetworkInterfaceEventArgs args)
	{
		if (!(sender is ServiceBrowser) || NetworkInterfaces == null)
		{
			return;
		}
		lock (NetworkInterfaces)
		{
			if (NetworkInterfaces.Count > 0)
			{
				NetworkInterface networkInterface = NetworkInterfaces.ToList().FirstOrDefault((NetworkInterface a) => a.Id == args.NetworkInterface.Id);
				if (networkInterface != null)
				{
					NetworkInterfaces.Remove(networkInterface);
				}
			}
			NetworkInterfaces.Add(args.NetworkInterface);
		}
	}

	private void OnNetworkInterfaceRemoved(object sender, NetworkInterfaceEventArgs args)
	{
		if (!(sender is ServiceBrowser) || NetworkInterfaces == null)
		{
			return;
		}
		lock (NetworkInterfaces)
		{
			NetworkInterfaces.Remove(args.NetworkInterface);
		}
	}
}
