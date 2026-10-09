using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Net;
using System.Text;

namespace ICUNetwork;

public class SCNSocket
{
	public int ScnLibVersion { get; set; }

	public string Name { get; set; }

	public uint Id { get; set; }

	public byte Mode3State { get; set; }

	public byte SocketIndex { get; set; }

	public byte SocketCount { get; set; }

	public EChargingState State { get; set; }

	public DateTime LastUpdate { get; set; }

	public string NetworkName { get; set; }

	public uint Timestamp { get; set; }

	public int TotalNumberOfSockets { get; set; }

	public uint PhaseMask { get; set; }

	public double ActiveCurrentL1 { get; set; }

	public double ActiveCurrentL2 { get; set; }

	public double ActiveCurrentL3 { get; set; }

	public ulong UniqueID { get; set; }

	public int MaximumGroupID { get; set; }

	public ulong Clock { get; set; }

	public DateTime LastClockUpdate { get; set; }

	public uint WaitingSince { get; set; }

	public int AlterningSince { get; set; }

	public double MinimumCurrent { get; set; }

	public double MaximumCurrent { get; set; }

	public double AvailableCurrentL1 { get; set; }

	public double AvailableCurrentL2 { get; set; }

	public double AvailableCurrentL3 { get; set; }

	public double SetPointCurrent { get; set; }

	public uint ActiveChargingTime { get; set; }

	public uint AlternatingCountDown { get; set; }

	public IPAddress IPAddress { get; set; }

	public double PropSocketSafeCurrent { get; set; }

	public double PropMaximumStaticCurrent { get; set; }

	public int PropAlternatingPeriod { get; set; }

	public int PropChangedCounter { get; set; }

	public byte OptionByte { get; set; }

	public double ExtraCurrentL1 { get; set; }

	public double ExtraCurrentL2 { get; set; }

	public double ExtraCurrentL3 { get; set; }

	public double MaximumGroupCurrent { get; set; }

	public double PropTotalSafeCurrent { get; set; }

	public string PhaseMapping { get; set; }

	public static HashSet<int> ValidLibraryVersions { get; private set; }

	static SCNSocket()
	{
		ValidLibraryVersions = Enum.GetValues(typeof(ScnLibraryVersions)).Cast<int>().ToHashSet();
	}

	public SCNSocket()
	{
		NetworkName = string.Empty;
		Id = 0u;
		Name = string.Empty;
		State = EChargingState.Idle;
		LastClockUpdate = DateTime.MinValue;
	}

	public void ParseData(byte[] data, DateTime dtReceived, IPAddress ipAddress)
	{
		int num = data[0];
		if (!ValidLibraryVersions.Contains(num) || data.Length < 96)
		{
			return;
		}
		IPAddress = ipAddress;
		LastClockUpdate = dtReceived;
		LastUpdate = DateTime.UtcNow;
		ScnLibVersion = num;
		using MemoryStream input = new MemoryStream(data);
		using BinaryReader binaryReader = new BinaryReader(input);
		binaryReader.ReadByte();
		binaryReader.ReadByte();
		binaryReader.ReadByte();
		binaryReader.ReadByte();
		uint timestamp = binaryReader.ReadUInt32();
		byte[] bytes = binaryReader.ReadBytes(8);
		NetworkName = Encoding.UTF8.GetString(bytes).Trim(new char[1]).Trim();
		if (NetworkName.Length == 0)
		{
			return;
		}
		Id = binaryReader.ReadUInt16();
		binaryReader.ReadByte();
		Mode3State = binaryReader.ReadByte();
		State = (EChargingState)binaryReader.ReadByte();
		PhaseMask = binaryReader.ReadByte();
		TotalNumberOfSockets = binaryReader.ReadByte();
		SocketIndex = binaryReader.ReadByte();
		Timestamp = timestamp;
		byte[] bytes2 = binaryReader.ReadBytes(21);
		Name = Encoding.UTF8.GetString(bytes2).Trim(new char[1]).Trim();
		SocketCount = binaryReader.ReadByte();
		MaximumGroupID = binaryReader.ReadUInt16();
		UniqueID = binaryReader.ReadUInt64();
		Clock = binaryReader.ReadUInt64();
		WaitingSince = binaryReader.ReadUInt32();
		AlterningSince = binaryReader.ReadInt32();
		MinimumCurrent = binaryReader.ReadSingle();
		MaximumCurrent = binaryReader.ReadSingle();
		ActiveCurrentL1 = binaryReader.ReadSingle();
		ActiveCurrentL2 = binaryReader.ReadSingle();
		ActiveCurrentL3 = binaryReader.ReadSingle();
		AvailableCurrentL1 = binaryReader.ReadSingle();
		AvailableCurrentL2 = binaryReader.ReadSingle();
		AvailableCurrentL3 = binaryReader.ReadSingle();
		SetPointCurrent = binaryReader.ReadSingle();
		ActiveChargingTime = binaryReader.ReadUInt32();
		AlternatingCountDown = binaryReader.ReadUInt32();
		if (num >= 3)
		{
			PropSocketSafeCurrent = binaryReader.ReadSingle();
			PropMaximumStaticCurrent = binaryReader.ReadSingle();
		}
		else
		{
			PropSocketSafeCurrent = (double)(int)binaryReader.ReadUInt16() / 10.0;
			PropMaximumStaticCurrent = (double)(int)binaryReader.ReadUInt16() / 10.0;
		}
		PropAlternatingPeriod = binaryReader.ReadUInt16();
		PropChangedCounter = binaryReader.ReadByte();
		OptionByte = binaryReader.ReadByte();
		if (num >= 2)
		{
			ExtraCurrentL1 = (int)binaryReader.ReadUInt16();
			ExtraCurrentL2 = (int)binaryReader.ReadUInt16();
			ExtraCurrentL3 = (int)binaryReader.ReadUInt16();
			binaryReader.ReadInt16();
			MaximumGroupCurrent = binaryReader.ReadSingle();
			if (num >= 3)
			{
				PropTotalSafeCurrent = binaryReader.ReadSingle();
			}
		}
	}
}
