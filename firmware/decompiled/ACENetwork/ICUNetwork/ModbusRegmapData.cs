using System.Collections.Generic;

namespace ICUNetwork;

public class ModbusRegmapData
{
	public string Name { get; set; }

	public string Version { get; set; }

	public byte? Address { get; set; }

	public string Parity { get; set; }

	public ushort? Baudrate { get; set; }

	public string WordOrder { get; set; }

	public uint? UpdateTime { get; set; }

	public uint? ReadTimeout { get; set; }

	public string FunctionCode { get; set; }

	public ushort SampleIntervalMs { get; set; }

	public List<ModbusRegmapEntry> Regmap { get; set; }
}
