namespace SimplePaletteQuantizer.ColorCaches.Common;

public enum ColorModel
{
	RedGreenBlue = 0,
	HueSaturationBrightness = 1,
	HueSaturationLuminance = HueSaturationBrightness,
	LabColorSpace = 2,
	XYZ = 3
}
