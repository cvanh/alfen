namespace ICUFWUCreator;

public class Crc32
{
	private uint[] Table { get; set; }

	public Crc32(uint poly = 79764919u)
	{
		Table = new uint[256];
		for (uint num = 0u; num < Table.Length; num++)
		{
			uint num2 = num;
			for (int num3 = 8; num3 > 0; num3--)
			{
				num2 = (((num2 & 1) != 1) ? (num2 >> 1) : ((num2 >> 1) ^ poly));
			}
			Table[num] = num2;
		}
	}

	public uint ComputeChecksum(byte[] bytes, int startIndex = 0)
	{
		uint num = uint.MaxValue;
		for (int i = startIndex; i < bytes.Length; i++)
		{
			byte b = (byte)((num & 0xFF) ^ bytes[i]);
			num = (num >> 8) ^ Table[b];
		}
		return ~num;
	}
}
