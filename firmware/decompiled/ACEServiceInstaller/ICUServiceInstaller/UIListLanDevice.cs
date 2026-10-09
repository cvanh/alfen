using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using ICUNetwork;
using Serilog;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIListLanDevice
{
	private readonly ILogger Logger = Log.ForContext<UIListLanDevice>();

	private readonly ICULanDevice m_lanDevice;

	private readonly bool m_fSmall;

	public bool IsSCN { get; }

	public string SCNName { get; } = string.Empty;

	public Image Icon
	{
		get
		{
			Image image = null;
			try
			{
				if (IsSCN)
				{
					image = Image.FromResource(typeof(App), AppProperties.ResourcePath("networking.png"));
					return image.Scale(0.8);
				}
				string resource = string.Format(AppProperties.ResourcePath($"{Enum.GetName(typeof(ICUDeviceModel), m_lanDevice.ModelType)}.png"), Array.Empty<object>());
				image = Image.FromResource(typeof(App), resource);
				if (m_lanDevice.IsUniquePasswordRequired)
				{
					image = CombineImages(image, "lock.png");
				}
			}
			catch (Exception ex)
			{
				Logger.Debug(ex, "Error '{Error}' loading model image, now using 'unknown' model image", ex.Message);
				image = Image.FromResource(typeof(App), AppProperties.ResourcePath("Unknown.png"));
			}
			if (m_fSmall)
			{
				return image.Scale(0.5);
			}
			return image.Scale(0.8);
		}
	}

	public UIListLanDevice(ICULanDevice lanDevice, bool smallScale = false)
	{
		m_lanDevice = lanDevice;
		m_fSmall = smallScale;
		IsSCN = false;
	}

	public UIListLanDevice(string sSCNName)
	{
		SCNName = sSCNName;
		m_fSmall = true;
		IsSCN = true;
	}

	public override string ToString()
	{
		if (IsSCN)
		{
			return $"\n{SCNName}\nSmart Charging Network\n";
		}
		if (m_lanDevice == null)
		{
			return "";
		}
		if (m_fSmall)
		{
			if (m_lanDevice.HasSCNNetwork)
			{
				return $"    {m_lanDevice.Identification} ({m_lanDevice.FirmwareVersionNumber})\n    {m_lanDevice.DisplayNameLine2}";
			}
			return $"{m_lanDevice.Identification} ({m_lanDevice.FirmwareVersionNumber})\n{m_lanDevice.DisplayNameLine2}";
		}
		return string.Format("{0} ({1})\n{2}\n{3} {4}", new object[5]
		{
			m_lanDevice.Identification,
			m_lanDevice.FirmwareVersionNumber,
			m_lanDevice.DisplayNameLine2,
			m_lanDevice.Address,
			m_lanDevice.Discovered ? "" : "(manual)"
		});
	}

	private Image CombineImages(Image orginalImage, string overlayResourceName)
	{
		//IL_0028: Unknown result type (might be due to invalid IL or missing references)
		//IL_002e: Expected Obj, but got Unknown
		//IL_004e: Unknown result type (might be due to invalid IL or missing references)
		//IL_0054: Expected Obj, but got Unknown
		try
		{
			Image image = Image.FromResource(typeof(App), AppProperties.ResourcePath(overlayResourceName));
			Bitmap val;
			using (MemoryStream memoryStream = new MemoryStream())
			{
				orginalImage.Save(memoryStream, ImageFileType.Png);
				val = new Bitmap((Stream)memoryStream);
			}
			Bitmap val2;
			using (MemoryStream memoryStream2 = new MemoryStream())
			{
				image.Save(memoryStream2, ImageFileType.Png);
				val2 = new Bitmap((Stream)memoryStream2);
			}
			Graphics val3 = Graphics.FromImage((Image)(object)val);
			try
			{
				val3.DrawImage((Image)(object)val2, 0, ((Image)val).Height - ((Image)val2).Height);
			}
			finally
			{
				((IDisposable)val3)?.Dispose();
			}
			Image result = null;
			using (MemoryStream memoryStream3 = new MemoryStream())
			{
				((Image)val).Save((Stream)memoryStream3, ImageFormat.Png);
				memoryStream3.Position = 0L;
				result = Image.FromStream(memoryStream3);
			}
			((Image)val).Dispose();
			return result;
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, "Error '{Error}' setting overlay icon", ex.Message);
			return orginalImage;
		}
	}

	public ICULanDevice GetDevice()
	{
		return m_lanDevice;
	}
}
