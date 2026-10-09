using System;
using System.Drawing;
using System.IO;
using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Interop;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using SimplePaletteQuantizer.ColorCaches.EuclideanDistance;
using SimplePaletteQuantizer.Ditherers;
using SimplePaletteQuantizer.Helpers;
using SimplePaletteQuantizer.Quantizers;
using SimplePaletteQuantizer.Quantizers.XiaolinWu;

namespace ICULogoConvertor;

public class ImageScaler
{
	[DllImport("gdi32.dll")]
	public static extern bool DeleteObject(IntPtr hObject);

	public static Image SetImageBackground(Image imgPhoto, Color newBackgroundColor)
	{
		//IL_000c: Unknown result type (might be due to invalid IL or missing references)
		//IL_0011: Unknown result type (might be due to invalid IL or missing references)
		//IL_0017: Expected Obj, but got Unknown
		//IL_0037: Expected Obj, but got Unknown
		//IL_0032: Unknown result type (might be due to invalid IL or missing references)
		//IL_0038: Expected Obj, but got Unknown
		Bitmap val = new Bitmap(imgPhoto.Width, imgPhoto.Height);
		Graphics val2 = Graphics.FromImage((Image)val);
		val2.Clear(newBackgroundColor);
		val2.DrawImage(imgPhoto, 0, 0, imgPhoto.Width, imgPhoto.Height);
		return (Image)new Bitmap((Image)val);
	}

	public static Image ResizeImage(Image imgPhoto, Size newSize)
	{
		//IL_0051: Unknown result type (might be due to invalid IL or missing references)
		//IL_0064: Unknown result type (might be due to invalid IL or missing references)
		//IL_0075: Unknown result type (might be due to invalid IL or missing references)
		//IL_007f: Expected Obj, but got Unknown
		//IL_007a: Unknown result type (might be due to invalid IL or missing references)
		//IL_0081: Expected Obj, but got Unknown
		//IL_0088: Unknown result type (might be due to invalid IL or missing references)
		//IL_008d: Unknown result type (might be due to invalid IL or missing references)
		//IL_00a8: Unknown result type (might be due to invalid IL or missing references)
		//IL_00af: Expected Obj, but got Unknown
		int width = imgPhoto.Width;
		int height = imgPhoto.Height;
		if (width < newSize.Width)
		{
			newSize.Width = width;
		}
		if (height < newSize.Height)
		{
			newSize.Height = height;
		}
		double val = (double)newSize.Width / (double)width;
		double val2 = (double)newSize.Height / (double)height;
		double num = Math.Min(val, val2);
		IntPtr hbitmap = new Bitmap(imgPhoto).GetHbitmap();
		TransformedBitmap val3 = new TransformedBitmap(Imaging.CreateBitmapSourceFromHBitmap(hbitmap, IntPtr.Zero, Int32Rect.Empty, BitmapSizeOptions.FromEmptyOptions()), (Transform)new ScaleTransform(num, num));
		Bitmap result;
		using (MemoryStream memoryStream = new MemoryStream())
		{
			((BitmapEncoder)new BmpBitmapEncoder
			{
				Frames = { BitmapFrame.Create((BitmapSource)(object)val3) }
			}).Save((Stream)memoryStream);
			result = new Bitmap((Stream)memoryStream);
		}
		DeleteObject(hbitmap);
		return (Image)(object)result;
	}

	public static Image QuantizeImage(Image image, int colorCount)
	{
		IColorQuantizer colorQuantizer = new WuColorQuantizer();
		if (colorQuantizer is BaseColorCacheQuantizer baseColorCacheQuantizer)
		{
			baseColorCacheQuantizer.ChangeCacheProvider(new EuclideanDistanceColorCache());
		}
		BaseColorDitherer ditherer = null;
		return ImageBuffer.QuantizeImage(image, colorQuantizer, ditherer, colorCount, 1);
	}
}
