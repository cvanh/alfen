using System;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net;
using System.Net.Sockets;
using System.Security.Cryptography;
using System.Threading;
using System.Threading.Tasks;
using Serilog;

namespace ICUNetwork;

public class SCNNetwork
{
	private readonly ILogger Logger = Log.ForContext<SCNNetwork>();

	public static int UDPPORT = 36549;

	private readonly TimeSpan m_16SecondsTimeSpan = new TimeSpan(0, 0, 16);

	private const int udpMaxBufferLength = 1024;

	private const int udpMinPacketLength = 96;

	private static readonly byte[] s_aesKey = new byte[16]
	{
		222, 11, 77, 232, 113, 88, 244, 239, 138, 139,
		108, 36, 96, 130, 116, 112
	};

	private ICryptoTransform m_decryptor;

	private readonly ConcurrentDictionary<string, SCNSocket> Sockets = new ConcurrentDictionary<string, SCNSocket>();

	private static readonly Lazy<SCNNetwork> lazySCNNetwork = new Lazy<SCNNetwork>(() => new SCNNetwork());

	private readonly BlockingCollection<UdpReceiveResult> _udpPackets = new BlockingCollection<UdpReceiveResult>();

	private static readonly object _lock = new object();

	private CancellationTokenSource m_cancellationTokenSource;

	private long totalUdpPacketsReceived;

	private long totalUdpPacketsProcessed;

	public static SCNNetwork Instance => lazySCNNetwork.Value;

	private SCNNetwork()
	{
	}

	private static string GetSocketKey(SCNSocket socket)
	{
		return $"{socket.UniqueID}_{socket.SocketIndex}_{socket.IPAddress}";
	}

	private static string GenerateSocketKey(ulong uniqueId, int socketIndex, IPAddress ipAddress)
	{
		return $"{uniqueId}_{socketIndex}_{ipAddress}";
	}

	private void InitializeAESDecryptor()
	{
		byte[] rgbIV = new byte[16];
		using RijndaelManaged rijndaelManaged = new RijndaelManaged
		{
			Padding = PaddingMode.None
		};
		m_decryptor = rijndaelManaged.CreateDecryptor(s_aesKey, rgbIV);
	}

	private void AESDecrypt(ref byte[] binDataDecrypted, byte[] dataToDecrypt)
	{
		using MemoryStream stream = new MemoryStream(dataToDecrypt);
		using CryptoStream cryptoStream = new CryptoStream(stream, m_decryptor, CryptoStreamMode.Read);
		cryptoStream.Read(binDataDecrypted, 0, binDataDecrypted.Length);
	}

	private void StartUdpReceiveTask(CancellationToken cancellationToken)
	{
		Logger.Debug("Start SCN UDP packet listening task");
		Task.Run(async () =>
		{
			using (UdpClient udpClient = new UdpClient
			{
				ExclusiveAddressUse = false
			})
			{
				udpClient.Client.SetSocketOption(SocketOptionLevel.Socket, SocketOptionName.ReuseAddress, optionValue: true);
				IPEndPoint localEP = new IPEndPoint(IPAddress.Any, UDPPORT);
				udpClient.Client.Bind(localEP);
				InitializeAESDecryptor();
				Stopwatch stopwatchUdpPacketsReceived = new Stopwatch();
				stopwatchUdpPacketsReceived.Start();
				int udpPacketsReceived = 0;
				while (!cancellationToken.IsCancellationRequested)
				{
					try
					{
						UdpReceiveResult item = await udpClient.ReceiveAsync();
						_udpPackets.Add(item);
						udpPacketsReceived++;
						totalUdpPacketsReceived++;
					}
					catch (Exception ex)
					{
						Logger.Debug(ex, ex.Message);
					}
					if (totalUdpPacketsReceived % 250 == 0L)
					{
						double totalSeconds = stopwatchUdpPacketsReceived.Elapsed.TotalSeconds;
						Logger.Debug("Received {Packets} packets in {TimeInSeconds:F3}s, {PacketsReceivedPerSecond:F0} packets/s, total={TotalPackets}", udpPacketsReceived, totalSeconds, (double)udpPacketsReceived / totalSeconds, totalUdpPacketsReceived);
						stopwatchUdpPacketsReceived.Reset();
						udpPacketsReceived = 0;
						stopwatchUdpPacketsReceived.Start();
					}
				}
			}
			Logger.Debug("Finished SCN UDP listening task");
		}, m_cancellationTokenSource.Token);
	}

	public void StartUdpProcessTask(CancellationToken cancellationToken)
	{
		Logger.Debug("Starting SCN UDP packet processing task");
		Task.Run(() =>
		{
			try
			{
				Stopwatch stopwatch = new Stopwatch();
				stopwatch.Start();
				int num = 0;
				foreach (UdpReceiveResult item in _udpPackets.GetConsumingEnumerable(cancellationToken))
				{
					try
					{
						ProcessUdpPacket(item);
						num++;
						totalUdpPacketsProcessed++;
					}
					catch (Exception ex)
					{
						Logger.Debug(ex, ex.Message);
					}
					if (totalUdpPacketsProcessed % 250 == 0L)
					{
						double totalSeconds = stopwatch.Elapsed.TotalSeconds;
						Logger.Debug("Processed {Packets} packets in {TimeInSeconds:F3}s, {PacketsProcessedPerSecond:F0} packets/s, total={TotalPackets}", num, totalSeconds, (double)num / totalSeconds, totalUdpPacketsProcessed);
						stopwatch.Reset();
						num = 0;
						stopwatch.Start();
					}
				}
			}
			catch (OperationCanceledException)
			{
				Logger.Debug("Finished SCN UDP packet processing task");
			}
		});
	}

	private void ProcessUdpPacket(UdpReceiveResult udpPacket)
	{
		DateTime now = DateTime.Now;
		Stopwatch.StartNew();
		if (udpPacket.Buffer.Length >= 1024)
		{
			return;
		}
		byte[] binDataDecrypted = new byte[1024];
		AESDecrypt(ref binDataDecrypted, udpPacket.Buffer);
		if (binDataDecrypted.Length < 96)
		{
			return;
		}
		int item = binDataDecrypted[0];
		if (!SCNSocket.ValidLibraryVersions.Contains(item) || binDataDecrypted.Length < 96)
		{
			return;
		}
		int socketIndex = binDataDecrypted[23];
		ulong uniqueId = BitConverter.ToUInt64(binDataDecrypted, 48);
		try
		{
			string socketKey = GenerateSocketKey(uniqueId, socketIndex, udpPacket.RemoteEndPoint.Address);
			Sockets.GetOrAdd(socketKey, (string key) =>
			{
				Logger.Debug("Socket added: {SocketKey}", socketKey);
				return new SCNSocket();
			}).ParseData(binDataDecrypted, now, udpPacket.RemoteEndPoint.Address);
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, ex.Message);
		}
	}

	public void Start()
	{
		lock (_lock)
		{
			m_cancellationTokenSource?.Dispose();
			m_cancellationTokenSource = new CancellationTokenSource();
			StartUdpReceiveTask(m_cancellationTokenSource.Token);
			StartUdpProcessTask(m_cancellationTokenSource.Token);
		}
	}

	public void Stop()
	{
		lock (_lock)
		{
			m_cancellationTokenSource?.Cancel();
		}
	}

	public void Clear()
	{
		Sockets.Clear();
	}

	public void SetPhasemapping(SCNSocket socket, string phasemapping)
	{
		if (Sockets.TryGetValue(GetSocketKey(socket), out var value))
		{
			value.PhaseMapping = phasemapping;
		}
		else
		{
			Logger.Warning("SetPhasemapping {PhaseMapping}: Socket not found for key {UniqueID}_{SocketIndex}", phasemapping, socket.UniqueID, socket.SocketIndex);
		}
	}

	public void Remove(SCNSocket socket)
	{
		if (!Sockets.TryRemove(GetSocketKey(socket), out var _))
		{
			Logger.Debug("Failed removing {SocketId}/{SocketIp} from the socket list", socket.Id, socket.IPAddress);
		}
		Logger.Debug("Removed {SocketId}/{SocketIp} from the socket list", socket.Id, socket.IPAddress);
	}

	public IReadOnlyList<SCNSocket> GetCopyOfSockets(string scnNetworkName)
	{
		return (from a in Sockets.Values
			where a.NetworkName?.ToLowerInvariant() == scnNetworkName.ToLowerInvariant() && DateTime.UtcNow - a.LastUpdate <= m_16SecondsTimeSpan
			orderby a.Id, a.PropChangedCounter descending
			select a).ToList();
	}

	public SCNSocket GetFirstSocket(string scnNetworkName)
	{
		return GetCopyOfSockets(scnNetworkName).FirstOrDefault();
	}

	public ICULanDevice GetLanDeviceForFirstSocket(string scnName, LANConnection lanConnection)
	{
		SCNSocket firstSocket = GetFirstSocket(scnName);
		if (firstSocket == null)
		{
			return null;
		}
		return lanConnection.FindLanDevice(firstSocket.IPAddress);
	}
}
