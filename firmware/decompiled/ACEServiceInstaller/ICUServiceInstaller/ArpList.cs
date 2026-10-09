using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Net;
using System.Net.NetworkInformation;
using System.Runtime.InteropServices;
using Serilog;

namespace ICUServiceInstaller;

public class ArpList
{
	private struct MIB_IPNETROW
	{
		[MarshalAs(UnmanagedType.U4)]
		public int dwIndex;

		[MarshalAs(UnmanagedType.U4)]
		public int dwPhysAddrLen;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac0;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac1;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac2;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac3;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac4;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac5;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac6;

		[MarshalAs(UnmanagedType.U1)]
		public byte mac7;

		[MarshalAs(UnmanagedType.U4)]
		public int dwAddr;

		[MarshalAs(UnmanagedType.U4)]
		public int dwType;
	}

	private static readonly ILogger Logger = Log.ForContext<ArpList>();

	private const int ERROR_INSUFFICIENT_BUFFER = 122;

	[DllImport("IpHlpApi.dll")]
	[return: MarshalAs(UnmanagedType.U4)]
	private static extern int GetIpNetTable(IntPtr pIpNetTable, [MarshalAs(UnmanagedType.U4)] ref int pdwSize, bool bOrder);

	public static Dictionary<IPAddress, PhysicalAddress> GetAllDevicesOnLAN()
	{
		Dictionary<IPAddress, PhysicalAddress> dictionary = new Dictionary<IPAddress, PhysicalAddress>();
		int pdwSize = 0;
		Stopwatch stopwatch = Stopwatch.StartNew();
		Logger.Debug("GetAllDevicesOnLAN starting");
		GetIpNetTable(IntPtr.Zero, ref pdwSize, bOrder: false);
		IntPtr intPtr = IntPtr.Zero;
		try
		{
			intPtr = Marshal.AllocCoTaskMem(pdwSize);
			int ipNetTable = GetIpNetTable(intPtr, ref pdwSize, bOrder: false);
			if (ipNetTable != 0)
			{
				Logger.Debug("Unable to retrieve network table. Error code {ErrorCode}", ipNetTable);
			}
			int num = Marshal.ReadInt32(intPtr);
			IntPtr intPtr2 = new IntPtr(intPtr.ToInt64() + Marshal.SizeOf(typeof(int)));
			MIB_IPNETROW[] array = new MIB_IPNETROW[num];
			for (int i = 0; i < num; i++)
			{
				array[i] = (MIB_IPNETROW)Marshal.PtrToStructure(new IntPtr(intPtr2.ToInt64() + i * Marshal.SizeOf(typeof(MIB_IPNETROW))), typeof(MIB_IPNETROW));
			}
			PhysicalAddress obj = new PhysicalAddress(new byte[6]);
			PhysicalAddress obj2 = new PhysicalAddress(new byte[6] { 255, 255, 255, 255, 255, 255 });
			MIB_IPNETROW[] array2 = array;
			for (int j = 0; j < array2.Length; j++)
			{
				MIB_IPNETROW mIB_IPNETROW = array2[j];
				IPAddress iPAddress = new IPAddress(BitConverter.GetBytes(mIB_IPNETROW.dwAddr));
				PhysicalAddress physicalAddress = new PhysicalAddress(new byte[6] { mIB_IPNETROW.mac0, mIB_IPNETROW.mac1, mIB_IPNETROW.mac2, mIB_IPNETROW.mac3, mIB_IPNETROW.mac4, mIB_IPNETROW.mac5 });
				if (!physicalAddress.Equals(obj) && !physicalAddress.Equals(obj2) && !IsMulticast(iPAddress))
				{
					Logger.Debug("IP: {IPAddress}\t\tMAC: {PhysicalAddress}", iPAddress.ToString(), physicalAddress.ToString());
					if (!dictionary.ContainsKey(iPAddress))
					{
						dictionary.Add(iPAddress, physicalAddress);
					}
				}
			}
			Logger.Debug("GetAllDevicesOnLAN finished in {ElapsedMillisecond} ms", stopwatch.ElapsedMilliseconds);
			return dictionary;
		}
		finally
		{
			Marshal.FreeCoTaskMem(intPtr);
		}
	}

	private static bool IsMulticast(IPAddress ip)
	{
		bool result = true;
		if (!ip.IsIPv6Multicast)
		{
			byte b = ip.GetAddressBytes()[0];
			if (b < 224 || b > 239)
			{
				result = false;
			}
		}
		return result;
	}
}
