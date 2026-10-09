using Newtonsoft.Json;
using Newtonsoft.Json.Converters;

namespace ICUNetwork;

public class ModbusRegmapEntry
{
	[JsonConverter(typeof(StringEnumConverter))]
	public EnergyMeterMeasurand Key { get; set; }

	public ushort RegNum { get; set; }

	[JsonConverter(typeof(StringEnumConverter))]
	public ModbusDataType DataType { get; set; }

	public sbyte ScaleE { get; set; }
}
