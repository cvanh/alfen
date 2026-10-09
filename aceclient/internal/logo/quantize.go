package logo

import (
	"fmt"
	"image"
	"image/color"
)

// ControlColor is SystemColors.Control with the default Windows theme
// (RGB 240,240,240): QuantizationHelper.BackgroundColor, the colour
// ConvertAlpha blends translucent pixels against.
var ControlColor = color.NRGBA{R: 240, G: 240, B: 240, A: 255}

// convertAlpha ports QuantizationHelper.ConvertAlpha(Color, out int argb)
// (ACELogoConvertor/SimplePaletteQuantizer.Helpers/QuantizationHelper.cs:35-60).
// Opaque colours pass through; others are blended against ControlColor with
// the C# double arithmetic and (int) truncation.
func convertAlpha(c color.NRGBA) (color.NRGBA, int) {
	if c.A < 255 {
		num := float64(c.A) / 255.0
		num2 := float64(255-c.A) / 255.0
		// explicit float64() conversions prevent FMA fusion of a*b+c*d.
		num4 := int(float64(float64(c.R)*num) + float64(float64(ControlColor.R)*num2))
		num6 := int(float64(float64(c.G)*num) + float64(float64(ControlColor.G)*num2))
		num8 := int(float64(float64(c.B)*num) + float64(float64(ControlColor.B)*num2))
		argb := num4<<16 | num6<<8 | num8
		return color.NRGBA{R: uint8(num4), G: uint8(num6), B: uint8(num8), A: 255}, argb
	}
	return c, int(c.R)<<16 | int(c.G)<<8 | int(c.B)
}

// paletteEntriesByColorCount ports Extend.GetFormatByColorCount followed by
// Extend.GetColorCount (ACELogoConvertor/SimplePaletteQuantizer.Extensions/Extend.cs:440-460, 224-252): 1bpp (2 entries) for <= 2 colours, 4bpp (16) for
// <= 16, else 8bpp (256). Errors like the C# for counts outside 1..256.
func paletteEntriesByColorCount(colorCount int) (int, error) {
	if colorCount <= 0 || colorCount > 256 {
		return 0, fmt.Errorf("A color count '%d' not supported!", colorCount)
	}
	switch {
	case colorCount > 16:
		return 256, nil // PixelFormat.Format8bppIndexed (198659)
	case colorCount > 2:
		return 16, nil // PixelFormat.Format4bppIndexed (197634)
	default:
		return 2, nil // PixelFormat.Format1bppIndexed (196865)
	}
}

// halftoneValues are the six channel levels of the GDI+ halftone palette.
var halftoneValues = [6]uint8{0x00, 0x33, 0x66, 0x99, 0xcc, 0xff}

// gdiplusDefaultPalette returns the palette GDI+ gives a freshly created
// indexed Bitmap(w, h, format): black/white for 1bpp, otherwise the halftone
// palette (8 dark VGA colours, silver, 7 bright VGA colours, 24 transparent
// black entries, then the 6x6x6 web cube from index 40), truncated to the
// entry count. ImageBuffer.Quantize only overwrites the first len(Wu palette)
// entries (Extend.SetPalette), so the rest of these survive into the FWU.
func gdiplusDefaultPalette(entries int) color.Palette {
	pal := make(color.Palette, entries)
	if entries == 2 {
		pal[0] = color.NRGBA{A: 255}
		pal[1] = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		return pal
	}
	for i := 0; i < entries; i++ {
		var c color.NRGBA
		switch {
		case i < 8:
			c.A = 255
			if i&1 != 0 {
				c.R = 0x80
			}
			if i&2 != 0 {
				c.G = 0x80
			}
			if i&4 != 0 {
				c.B = 0x80
			}
		case i == 8:
			c = color.NRGBA{R: 0xc0, G: 0xc0, B: 0xc0, A: 255}
		case i < 16:
			c.A = 255
			if i&1 != 0 {
				c.R = 0xff
			}
			if i&2 != 0 {
				c.G = 0xff
			}
			if i&4 != 0 {
				c.B = 0xff
			}
		case i < 40:
			c = color.NRGBA{} // 0x00000000
		default:
			c.A = 255
			c.B = halftoneValues[(i-40)%6]
			c.G = halftoneValues[((i-40)/6)%6]
			c.R = halftoneValues[((i-40)/36)%6]
		}
		pal[i] = c
	}
	return pal
}

// QuantizeImage ports ImageScaler.QuantizeImage (ImageScaler.cs:77-86), i.e.
// ImageBuffer.QuantizeImage(image, new WuColorQuantizer(), ditherer: null,
// colorCount, parallelTaskCount: 1):
//
//  1. target = new Bitmap(w, h, GetFormatByColorCount(colorCount));
//  2. SynthetizePalette: Wu.Prepare, ScanColors (row-major, each colour
//     through ConvertAlpha), Wu.GetPalette(colorCount);
//  3. SetPalette onto the target's default GDI+ palette;
//  4. TransformPerPixel: index = Wu.GetPaletteIndex(ConvertAlpha(c), x, y).
//
// (ImageBuffer.cs:499-530 QuantizeImage, 437-443 SynthetizePalette, 418-428
// ScanColors, Quantize; Extend.cs:74-100 SetPalette.)
//
// Pix holds one palette index per pixel regardless of the bit depth.
func QuantizeImage(img image.Image, colorCount int) (*image.Paletted, error) {
	if img == nil {
		return nil, fmt.Errorf("Cannot use 'sourceImage' when it is null!")
	}
	entries, err := paletteEntriesByColorCount(colorCount)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	q := &wuColorQuantizer{}
	q.onPrepare(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c, _ := convertAlpha(nrgbaAt(img, b.Min.X+x, b.Min.Y+y))
			q.onAddColor(c)
		}
	}
	wuPal := q.onGetPalette(colorCount)

	pal := gdiplusDefaultPalette(entries)
	if len(wuPal) > len(pal) {
		return nil, fmt.Errorf("Cannot store a palette with '%d' colors intto an image palette where only '%d' colors are allowed.", len(wuPal), len(pal))
	}
	for i, c := range wuPal {
		pal[i] = c
	}

	out := &image.Paletted{
		Pix:     make([]uint8, w*h),
		Stride:  w,
		Rect:    image.Rect(0, 0, w, h),
		Palette: pal,
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// SetColorToPixel -> quantizer.GetPaletteIndex(color, x, y): Wu
			// ignores the (ConvertAlpha'd) colour and indexes by position.
			out.Pix[y*w+x] = byte(q.onGetPaletteIndex(x, y))
		}
	}
	q.onFinish()
	return out, nil
}
