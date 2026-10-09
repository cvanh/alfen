using System;
using System.Collections.Generic;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using Serilog;

namespace ICUFWUCreator;

public class ICUFWUCreator
{
	private static readonly ILogger Logger = Log.ForContext<ICUFWUCreator>();

	private unsafe static byte[] GetImage(Bitmap image)
	{
		//IL_0029: Unknown result type (might be due to invalid IL or missing references)
		byte[] array = new byte[((Image)image).Width * ((Image)image).Height];
		BitmapData val = image.LockBits(new Rectangle(0, 0, ((Image)image).Width, ((Image)image).Height), (ImageLockMode)1, ((Image)image).PixelFormat);
		IntPtr scan = val.Scan0;
		int num = 0;
		for (int i = 0; i < ((Image)image).Height; i++)
		{
			byte* ptr = (byte*)(void*)(scan + i * val.Stride);
			for (int j = 0; j < val.Width; j++)
			{
				array[num++] = ptr[j];
			}
		}
		image.UnlockBits(val);
		return array;
	}

	protected static byte[] AESEncrypt(byte[] dataToEncrypt)
	{
		byte[] rgbKey = new byte[16]
		{
			41, 198, 106, 32, 174, 22, 196, 186, 4, 106,
			33, 213, 122, 120, 102, 79
		};
		byte[] rgbIV = new byte[16];
		ICryptoTransform transform = new RijndaelManaged().CreateEncryptor(rgbKey, rgbIV);
		byte[] array = null;
		using MemoryStream memoryStream = new MemoryStream();
		using (CryptoStream cryptoStream = new CryptoStream(memoryStream, transform, CryptoStreamMode.Write))
		{
			cryptoStream.Write(dataToEncrypt, 0, dataToEncrypt.Length);
			cryptoStream.FlushFinalBlock();
		}
		return memoryStream.ToArray();
	}

	protected static byte[] CreateFWIHeader(int dataLength)
	{
		byte[] array = null;
		using MemoryStream memoryStream = new MemoryStream();
		using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
		{
			binaryWriter.Write(0u);
			binaryWriter.Write(160u);
			binaryWriter.Write(2717777939u);
			binaryWriter.Write(4294967279u);
			binaryWriter.Write((ushort)41473);
			binaryWriter.Write((ushort)39);
			binaryWriter.Write((uint)dataLength);
			binaryWriter.Write(0u);
			binaryWriter.Write(uint.MaxValue);
			for (int i = 0; i < 32; i++)
			{
				binaryWriter.Write(uint.MaxValue);
			}
			memoryStream.Flush();
			Crc32 crc = new Crc32(3988292384u);
			binaryWriter.Seek(0, SeekOrigin.Begin);
			binaryWriter.Write(crc.ComputeChecksum(memoryStream.ToArray(), 4));
		}
		return memoryStream.ToArray();
	}

	protected static byte[] GetDataInBin(byte[] originalData)
	{
		byte[] array = null;
		using MemoryStream memoryStream = new MemoryStream();
		using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
		{
			binaryWriter.Write(0u);
			binaryWriter.Write((uint)(16 + originalData.Length));
			binaryWriter.Write(0u);
			binaryWriter.Write(0u);
			binaryWriter.Write(originalData);
			memoryStream.Flush();
			Crc32 crc = new Crc32(3988292384u);
			binaryWriter.Seek(0, SeekOrigin.Begin);
			binaryWriter.Write(crc.ComputeChecksum(memoryStream.ToArray(), 4));
		}
		return memoryStream.ToArray();
	}

	private static byte[] DeflateData(byte[] data)
	{
		byte[] array;
		using (MemoryStream memoryStream = new MemoryStream())
		{
			using (DeflateStream deflateStream = new DeflateStream(memoryStream, CompressionLevel.Optimal))
			{
				deflateStream.Write(data, 0, data.Length);
			}
			array = memoryStream.ToArray();
		}
		string messageTemplate = "Data: " + string.Join(",", Array.ConvertAll(array, (byte x) => $"0x{x:X2}"));
		Logger.Debug(messageTemplate);
		byte[] array2 = new byte[array.Length + 2];
		array2[0] = 120;
		array2[1] = 156;
		array.CopyTo(array2, 2);
		return array2;
	}

	protected static bool AddObjectToStream(BinaryWriter writer, byte[] data, ICUObjectTypes objectType, ICUImageFormats format, int width, int height, int stride, int maxColors, int verticalOffset, Image image)
	{
		byte[] array = DeflateData(data);
		uint num = new Crc32(3988292384u).ComputeChecksum(data);
		uint num2 = new Crc32(3988292384u).ComputeChecksum(array);
		Logger.Debug("CRC32-Poly2 original: {Original}", $"{num:X08}");
		Logger.Debug("CRC32-Poly2 compressed: {Compressed}", $"{num2:X08}");
		long position = writer.BaseStream.Position;
		int num3 = ((image != null) ? maxColors : 0);
		uint num4 = (uint)(32 + num3 * 3 + array.Length);
		uint num5 = 0u;
		if (num4 % 4 != 0)
		{
			num5 = 4 - num4 % 4;
			num4 += num5;
		}
		writer.Write((byte)1);
		writer.Write((byte)objectType);
		writer.Write((ushort)width);
		writer.Write((ushort)height);
		writer.Write((ushort)stride);
		writer.Write(num4);
		writer.Write((uint)array.Length);
		writer.Write((byte)format);
		int num6 = maxColors;
		if (image != null)
		{
			num6 = Math.Min(image.Palette.Entries.Length, maxColors);
		}
		int num7 = ((image != null) ? (num6 - 1) : 0);
		writer.Write((byte)num7);
		if (num7 > 0)
		{
			writer.Write((ushort)32);
			writer.Write((ushort)(32 + num6 * 3));
		}
		else
		{
			writer.Write((ushort)0);
			writer.Write((ushort)32);
		}
		writer.Write((short)verticalOffset);
		writer.Write(num);
		writer.Write(num2);
		if (image != null)
		{
			for (int i = 0; i < num6; i++)
			{
				writer.Write(image.Palette.Entries[i].R);
				writer.Write(image.Palette.Entries[i].G);
				writer.Write(image.Palette.Entries[i].B);
			}
		}
		writer.Write(array);
		for (int j = 0; j < num5; j++)
		{
			writer.Write((byte)0);
		}
		Logger.Debug("Object {Object} written at: {StartPos}, length: {Length} Pos: {Pos}", objectType, position, writer.BaseStream.Position - position, writer.BaseStream.Position);
		return true;
	}

	protected static bool AddLanguageToStream(BinaryWriter writer, byte[] data, ICUObjectTypes objectType, string language)
	{
		byte[] array = DeflateData(data);
		uint num = new Crc32(3988292384u).ComputeChecksum(data);
		uint num2 = new Crc32(3988292384u).ComputeChecksum(array);
		Logger.Debug("CRC32-Poly2 original: {Original}", $"{num:X08}");
		Logger.Debug("CRC32-Poly2 compressed: {Compressed}", $"{num2:X08}");
		long position = writer.BaseStream.Position;
		uint num3 = (uint)(32 + array.Length + language.Length);
		uint num4 = 0u;
		if (num3 % 4 != 0)
		{
			num4 = 4 - num3 % 4;
			num3 += num4;
		}
		writer.Write((byte)1);
		writer.Write((byte)objectType);
		writer.Write((ushort)0);
		writer.Write((ushort)0);
		writer.Write((ushort)0);
		writer.Write(num3);
		writer.Write((uint)array.Length);
		writer.Write(byte.MaxValue);
		writer.Write((byte)language.Length);
		writer.Write((ushort)32);
		writer.Write((ushort)(32 + language.Length));
		writer.Write((ushort)0);
		writer.Write(num);
		writer.Write(num2);
		for (int i = 0; i < language.Length; i++)
		{
			writer.Write((byte)language[i]);
		}
		writer.Write(array);
		for (int j = 0; j < num4; j++)
		{
			writer.Write((byte)0);
		}
		Logger.Debug("Object {ObjectType} written at: {StartPosition}, length: {Length} Pos: {Position}", objectType, position, writer.BaseStream.Position - position, writer.BaseStream.Position);
		return true;
	}

	protected static bool AddLanguage(BinaryWriter writer, ICUObjectTypes objectType, string fileName)
	{
		string text = "";
		byte[] array = null;
		byte[] array2 = null;
		using (MemoryStream memoryStream = new MemoryStream())
		{
			using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
			{
				string[] array3 = File.ReadAllLines(fileName);
				if (array3.Length != 0)
				{
					string[] array4 = array3[0].Split(new char[1] { ':' });
					if (array4.Length > 1 && array4[0].Trim().ToLowerInvariant() == "language")
					{
						text = array4[1].Trim();
					}
					for (int i = 1; i < array3.Length; i++)
					{
						string[] array5 = array3[i].Split(new char[1] { ',' });
						if (array5.Length >= 5)
						{
							_ = binaryWriter.BaseStream.Position;
							ICUDisplayStrings iCUDisplayStrings = (ICUDisplayStrings)Enum.Parse(typeof(ICUDisplayStrings), array5[0]);
							string text2 = array5[1];
							if (array5.Length > 5)
							{
								List<string> list = new List<string>();
								for (int j = 1; j < array5.Length - 3; j++)
								{
									list.Add(array5[j]);
								}
								text2 = string.Join(",", list.ToArray());
							}
							text2 = text2.Trim(new char[3] { ' ', '\t', '"' });
							text2 = text2.Trim();
							text2 = text2.Replace("\\n", "\n");
							try
							{
								array2 = Encoding.GetEncoding("ISO-8859-1", new EncoderReplacementFallback(""), new DecoderReplacementFallback("")).GetBytes(text2);
							}
							catch (ArgumentException exception)
							{
								Logger.Debug(exception, "");
								continue;
							}
							if (array2.Length > 80)
							{
								Logger.Debug("Error, text string: '{Text}' is too large, text truncated! ({Length} chars, allowed 80 chars)", text2, array2.Length);
							}
							byte[] array6 = new byte[80];
							Array.Copy(array2, array6, Math.Min(array2.Length, 79));
							binaryWriter.Write((byte)iCUDisplayStrings);
							binaryWriter.Write((byte)array6.Length);
							binaryWriter.Write((byte)Convert.ToUInt16(array5[^3]));
							binaryWriter.Write((byte)0);
							binaryWriter.Write(Convert.ToUInt16(array5[^2]));
							binaryWriter.Write(Convert.ToUInt16(array5[^1]));
							binaryWriter.Write(array6);
						}
						else
						{
							Logger.Debug("Error, invalid line formatting in file {0}: line {1} is missing one or more required fields", fileName, i);
						}
					}
				}
			}
			array = memoryStream.ToArray();
			Logger.Debug("AddLanguageToStream for '{Language}' = {Length} bytes", text, array.Length);
			string messageTemplate = "Data: " + string.Join(",", Array.ConvertAll(array, (byte x) => $"0x{x:X2}"));
			Logger.Debug(messageTemplate);
		}
		return AddLanguageToStream(writer, array, objectType, text);
	}

	protected static byte[] WriteFWUFile(byte[] allData)
	{
		MemoryStream memoryStream = new MemoryStream();
		using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
		{
			Logger.Debug("---------------------------------------------");
			Logger.Debug("Raw data size: {Length}", allData.Length);
			byte[] dataInBin = GetDataInBin(allData);
			Logger.Debug("Bin size: {Length}", dataInBin.Length);
			byte[] array = AESEncrypt(dataInBin);
			Logger.Debug("Encrypted bin size: {Length}", array.Length);
			byte[] array2 = CreateFWIHeader(array.Length);
			Logger.Debug("FWI header: {Length}", array2.Length);
			byte[] array3 = new byte[array2.Length + array.Length];
			array2.CopyTo(array3, 0);
			array.CopyTo(array3, 160);
			Logger.Debug("Combined: {Length}", array3.Length);
			binaryWriter.Write(array3);
			Logger.Debug("---------------------------------------------");
		}
		return memoryStream.ToArray();
	}

	protected static void AddAllLanguageFilesFromFolder(BinaryWriter writer, string path)
	{
		try
		{
			if (!Directory.Exists(path))
			{
				Directory.CreateDirectory(path);
			}
			List<FileInfo> list = (from a in Directory.GetFiles(path)
				select new FileInfo(a)).ToList();
			string[] array = new string[3] { "_NL", "_EN", "_DE" };
			foreach (string lang in array)
			{
				FileInfo fileInfo = list.FirstOrDefault((FileInfo a) => a.Name.Contains(lang));
				if (fileInfo != null)
				{
					AddLanguage(writer, ICUObjectTypes.OBJECT_LANGUAGE, fileInfo.FullName);
					list.Remove(fileInfo);
				}
			}
			foreach (FileInfo item in list)
			{
				AddLanguage(writer, ICUObjectTypes.OBJECT_LANGUAGE, item.FullName);
			}
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "");
		}
	}

	public static void AppendCFileBlock(StringBuilder sb, string name, byte[] data)
	{
		sb.AppendFormat("const uint8_t {0}[{1}] = {{\n", name, data.Length);
		for (int i = 0; i < data.Length; i++)
		{
			if (i > 0)
			{
				sb.Append(", ");
				if (i % 16 == 0)
				{
					sb.AppendLine();
				}
			}
			sb.AppendFormat("0x{0:X2}", data[i]);
		}
		sb.AppendLine("\n};");
		sb.AppendLine();
	}

	public static byte[] CreateFWUData(Image image1, int maxColors, string uiFolderPath, bool createCFile = false, string pathToCFile = "", bool largeScreen = false)
	{
		//IL_0001: Unknown result type (might be due to invalid IL or missing references)
		//IL_000b: Expected Obj, but got Unknown
		byte[] image2 = GetImage((Bitmap)image1);
		byte[] array = null;
		using (MemoryStream memoryStream = new MemoryStream())
		{
			using (BinaryWriter binaryWriter = new BinaryWriter(memoryStream))
			{
				AddObjectToStream(binaryWriter, image2, ICUObjectTypes.OBJECT_LOGO_CUSTOMER, ICUImageFormats.PALETTED8, image1.Width, image1.Height, image1.Width, maxColors, 0, image1);
				AddObjectToStream(binaryWriter, ICUObjects.logo_accepted_L1, ICUObjectTypes.OBJECT_LOGO_ACCEPTED, ICUImageFormats.FORMAT_L1, 56, 59, 7, 0, 0, null);
				AddObjectToStream(binaryWriter, ICUObjects.logo_charging_L1, ICUObjectTypes.OBJECT_LOGO_CHARGING, ICUImageFormats.FORMAT_L1, 80, 50, 10, 0, 0, null);
				AddObjectToStream(binaryWriter, ICUObjects.logo_error_L1, ICUObjectTypes.OBJECT_LOGO_SOCKETERROR, ICUImageFormats.FORMAT_L1, 56, 56, 7, 0, 0, null);
				AddObjectToStream(binaryWriter, ICUObjects.logo_communicating_L1, ICUObjectTypes.OBJECT_LOGO_COMMUNICATING, ICUImageFormats.FORMAT_L1, 56, 78, 7, 0, 0, null);
				AddObjectToStream(binaryWriter, ICUObjects.font_robotica_28, ICUObjectTypes.OBJECT_ROBOTICA_REGULAR_28, ICUImageFormats.FORMAT_L4, 20, 28, 10, 0, -2, null);
				AddObjectToStream(binaryWriter, ICUObjects.font_robotica_29, ICUObjectTypes.OBJECT_ROBOTICA_REGULAR_29, ICUImageFormats.FORMAT_L4, 22, 30, 11, 0, -2, null);
				if (largeScreen | createCFile)
				{
					AddObjectToStream(binaryWriter, ICUObjects.font_robotica_30, ICUObjectTypes.OBJECT_ROBOTICA_REGULAR_30, ICUImageFormats.FORMAT_L4, 30, 41, 15, 0, -4, null);
				}
				AddAllLanguageFilesFromFolder(binaryWriter, uiFolderPath);
				binaryWriter.Write(0u);
			}
			memoryStream.Flush();
			array = memoryStream.ToArray();
		}
		Logger.Debug("Total objects size: {Length}", array.Length);
		if (createCFile)
		{
			StringBuilder stringBuilder = new StringBuilder();
			AppendCFileBlock(stringBuilder, "object_data", array);
			File.WriteAllText(Path.Combine(pathToCFile, "default_objects.c"), stringBuilder.ToString());
		}
		return WriteFWUFile(array);
	}
}
