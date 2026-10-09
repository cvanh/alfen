using System.IO;
using System.Text;

namespace ICUFWUCreator.Model;

internal class TvfHeader
{
	private readonly string _description;

	private readonly string _manifestVersion;

	private readonly int _dataLength;

	private readonly int _baseHeaderLength = 1350;

	private readonly Crc32 _crc32;

	internal TvfHeader(int manifestVersion, string description, int dataLength)
	{
		_manifestVersion = manifestVersion.ToString();
		_description = description;
		_dataLength = dataLength;
		_crc32 = new Crc32(3988292384u);
	}

	internal byte[] GetBytes()
	{
		byte[] headerBody = null;
		using (MemoryStream memoryStream = new MemoryStream())
		{
			using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
			{
				binaryWriter.Write((ushort)1);
				binaryWriter.Write((ushort)CalculateHeaderLength());
				binaryWriter.Write(3102571221u);
				binaryWriter.Write((uint)_dataLength);
				binaryWriter.Write(0u);
				binaryWriter.Write((byte)0);
				binaryWriter.Write((byte)0);
				binaryWriter.Write((byte)0);
				binaryWriter.Write((byte)0);
				binaryWriter.Write(0u);
				binaryWriter.Write(0u);
				WriteUtf8WithLength(binaryWriter, _manifestVersion);
				WriteUtf8WithLength(binaryWriter, _description);
				binaryWriter.Write(new byte[32]);
				binaryWriter.Write((ushort)0);
				binaryWriter.Write(new byte[256]);
				binaryWriter.Write((ushort)0);
				binaryWriter.Write(new byte[1024]);
			}
			headerBody = memoryStream.ToArray();
		}
		return CreateFinalTvfHeader(headerBody);
	}

	internal void WriteUtf8WithLength(BinaryWriter binWriter, string text)
	{
		byte[] bytes = Encoding.UTF8.GetBytes(text);
		binWriter.Write((byte)bytes.Length);
		binWriter.Write(bytes);
	}

	private byte[] CreateFinalTvfHeader(byte[] headerBody)
	{
		uint value = _crc32.ComputeChecksum(headerBody);
		byte[] array = null;
		using MemoryStream memoryStream = new MemoryStream();
		using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
		{
			binaryWriter.Write(headerBody);
			binaryWriter.Write(value);
		}
		return memoryStream.ToArray();
	}

	private uint CalculateHeaderLength()
	{
		return (uint)(_baseHeaderLength + Encoding.UTF8.GetBytes(_manifestVersion).Length + Encoding.UTF8.GetBytes(_description).Length);
	}
}
