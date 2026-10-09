using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using ICSharpCode.SharpZipLib.GZip;
using ICSharpCode.SharpZipLib.Tar;
using ICUFWUCreator.Model;
using Newtonsoft.Json;
using Serilog;

namespace ICUFWUCreator;

public class ICUTVFCreator
{
	private static readonly ILogger Logger = Log.ForContext<ICUTVFCreator>();

	private static readonly string INNER_PACKAGE_NAME = "inner_package.tar";

	private static readonly string UPDATE_PACKAGE_NAME = "update_package.tar.gz";

	private static readonly string MANIFEST_FILE_NAME = "manifest.json";

	private static readonly string VIDEO_RESOURCE_FILE_NAME = "videoresources.json";

	private static readonly int MAX_FILE_SIZE_IN_KB = 500;

	private static readonly int MAX_FILE_SIZE = MAX_FILE_SIZE_IN_KB * 1024;

	public static byte[] CreateTvfData(string inputImageFile, string creator, int margin, int manifestVersion = 1)
	{
		ValidateImageFile(inputImageFile);
		string text = CreateTemporaryFolder();
		string path = Path.Combine(text, UPDATE_PACKAGE_NAME);
		string text2 = CreateInnerPackageTar(text, inputImageFile, creator, margin, manifestVersion);
		int dataLength = (int)new FileInfo(text2).Length;
		byte[] bytes = new TvfHeader(manifestVersion, Path.GetFileNameWithoutExtension(inputImageFile), dataLength).GetBytes();
		using (FileStream fileStream = File.Create(path))
		{
			fileStream.Write(bytes, 0, bytes.Length);
			using GZipOutputStream outputStream = new GZipOutputStream(fileStream);
			using TarArchive tarArchive = TarArchive.CreateOutputTarArchive(outputStream);
			AddFileToTar(tarArchive, text2);
		}
		byte[] result = ValidateOutput(File.ReadAllBytes(path));
		DeleteTemporaryFolder(text);
		return result;
	}

	private static void DeleteTemporaryFolder(string path)
	{
		try
		{
			Directory.Delete(path, recursive: true);
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "Temp folder not deleted");
		}
	}

	private static byte[] ValidateOutput(byte[] output)
	{
		if (output.Length > MAX_FILE_SIZE)
		{
			throw new NotSupportedException($"Compressed image size exceeds the limit off {MAX_FILE_SIZE_IN_KB} kB");
		}
		return output;
	}

	private static void ValidateImageFile(string pathToImage)
	{
		Image val = Image.FromFile(pathToImage);
		try
		{
			if (!((object)val.RawFormat).Equals((object?)ImageFormat.Png))
			{
				throw new NotSupportedException("Only the png image format is supported on this CS platform");
			}
		}
		finally
		{
			((IDisposable)val)?.Dispose();
		}
	}

	private static string CreateInnerPackageTar(string tmpFolderName, string pathToImageFile, string creator, int margin, int manifestVersion)
	{
		string fileNameWithoutExtension = Path.GetFileNameWithoutExtension(pathToImageFile);
		string fileName = Path.GetFileName(pathToImageFile);
		File.Copy(pathToImageFile, Path.Combine(tmpFolderName, fileName));
		AhpManifest value = new AhpManifest(creator, fileNameWithoutExtension, manifestVersion);
		AhpVideoResource value2 = new AhpVideoResource(fileName, margin);
		string fullFileName = WriteTextFile(tmpFolderName, MANIFEST_FILE_NAME, JsonConvert.SerializeObject(value));
		string fullFileName2 = WriteTextFile(tmpFolderName, VIDEO_RESOURCE_FILE_NAME, JsonConvert.SerializeObject(value2));
		string text = Path.Combine(tmpFolderName, INNER_PACKAGE_NAME);
		using FileStream outputStream = File.Create(text);
		using TarArchive tarArchive = TarArchive.CreateOutputTarArchive(outputStream);
		AddFileToTar(tarArchive, Path.Combine(tmpFolderName, fileName));
		AddFileToTar(tarArchive, fullFileName2);
		AddFileToTar(tarArchive, fullFileName);
		return text;
	}

	private static string WriteTextFile(string path, string fileName, string textContent)
	{
		string text = Path.Combine(path, fileName);
		using StreamWriter streamWriter = new StreamWriter(text);
		streamWriter.WriteLine(textContent);
		return text;
	}

	private static string CreateTemporaryFolder()
	{
		DirectoryInfo directoryInfo = Directory.CreateDirectory(Path.Combine(Path.GetTempPath(), Path.GetRandomFileName()));
		directoryInfo.Create();
		return directoryInfo.FullName;
	}

	private static void AddFileToTar(TarArchive tarArchive, string fullFileName)
	{
		TarEntry tarEntry = TarEntry.CreateEntryFromFile(fullFileName);
		tarEntry.Name = Path.GetFileName(fullFileName);
		tarEntry.TarHeader.Mode = Convert.ToInt32("644", 8);
		tarArchive.WriteEntry(tarEntry, recurse: false);
	}
}
