// Package logo ports ACELogoConvertor.dll: the image pipeline the installer
// runs before a customer logo is packed into a display resource.
//
// Ported code paths (ACELogoConvertor/…, decompiled with ilspycmd):
//
//   - ICULogoConvertor.ConvertImage (ICULogoConvertor/ICULogoConvertor.cs)
//   - ImageScaler.SetImageBackground / ResizeImage / QuantizeImage
//     (ICULogoConvertor/ImageScaler.cs)
//   - the ONLY SimplePaletteQuantizer code reachable from those calls:
//     WuColorQuantizer + WuColorCube (Quantizers.XiaolinWu), the
//     BaseColorQuantizer AddColor/GetPaletteIndex plumbing, ImageBuffer's
//     QuantizeImage/Quantize/SynthetizePalette/ScanColors path with a nil
//     ditherer and parallelTaskCount 1, StandardPathProvider,
//     QuantizationHelper.ConvertAlpha, Extend.GetFormatByColorCount and
//     Extend.SetPalette.
//
// Unreachable SimplePaletteQuantizer code is deliberately NOT ported: the
// EuclideanDistanceColorCache that ImageScaler.QuantizeImage installs is never
// used because WuColorQuantizer derives from BaseColorQuantizer, not
// BaseColorCacheQuantizer (the `is BaseColorCacheQuantizer` test is false), and
// the ditherer is always null.
//
// System.Drawing / WPF / GDI+ behaviour that the C# relies on implicitly is
// reproduced where it determines output bytes (the GDI+ default palette of a
// new indexed bitmap, WPF TransformedBitmap's output-size rounding). The
// WIC "Fant" resampler and GDI+ codecs are Windows binaries without source;
// they are approximated and listed as unverifiable.
//
// No Fyne imports: the GUI calls ConvertImage and shows the *image.Paletted.
package logo

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // GDI+ Image.FromFile decodes GIF
	_ "image/jpeg" // GDI+ Image.FromFile decodes JPEG
	_ "image/png"  // GDI+ Image.FromFile decodes PNG
	"math"
	"os"
)

// White is System.Drawing.Color.White, the background ConvertImage passes to
// SetImageBackground.
var White = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

// overflowRed is Color.FromArgb(255, 255, 0, 0): ConvertImage writes it into
// every palette entry at index >= maxColors.
var overflowRed = color.NRGBA{R: 255, A: 255}

// LoadImage stands in for System.Drawing.Image.FromFile: it decodes the file
// and reports the codec name ("png", "jpeg" or "gif"). GDI+ also decodes BMP,
// TIFF, ICO, EMF and WMF; those need golang.org/x/image and are not wired in.
func LoadImage(path string) (image.Image, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	img, format, err := image.Decode(f)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	return img, format, nil
}

// ConvertImage ports ICULogoConvertor.ConvertImage
// (ACELogoConvertor/ICULogoConvertor/ICULogoConvertor.cs:9-31): fit the image into
// desiredWidth x desiredHeight keeping the aspect ratio (never upscaling),
// flatten it onto white, quantize it with Xiaolin Wu to maxColors colours and
// paint every palette entry at index >= maxColors opaque red.
//
// The result is the GDI+ indexed bitmap as a *image.Paletted: one palette
// index per Pix byte, and a palette of 2, 16 or 256 entries (the GDI+ entry
// count for 1, 4 or 8 bpp, picked by Extend.GetFormatByColorCount). The
// installer always passes maxColors 128, i.e. 8 bpp with 256 entries.
func ConvertImage(input image.Image, desiredWidth, desiredHeight, maxColors int) (*image.Paletted, error) {
	if input == nil {
		return nil, errors.New("Cannot use 'sourceImage' when it is null!")
	}
	b := input.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("Parameter is not valid. (empty %dx%d image)", w, h)
	}
	val := float64(w) / float64(desiredWidth)
	val2 := float64(h) / float64(desiredHeight)
	num := math.Max(val, val2)
	// A 0 desired size gives num = +Inf and width/height 0; ResizeImage then
	// yields 1x1 (TransformedBitmap clamps to Math.Max(1, ...)), as in C#.
	width := netFloorToInt(float64(w) / num)
	height := netFloorToInt(float64(h) / num)
	resized := ResizeImage(SetImageBackground(input, White), width, height)
	val3, err := QuantizeImage(resized, maxColors)
	if err != nil {
		return nil, err
	}
	pal := make(color.Palette, len(val3.Palette))
	for i := range val3.Palette {
		if i < maxColors {
			pal[i] = val3.Palette[i]
		} else {
			pal[i] = overflowRed
		}
	}
	val3.Palette = pal
	return val3, nil
}

// SetImageBackground ports ImageScaler.SetImageBackground
// (ACELogoConvertor/ICULogoConvertor/ImageScaler.cs:22-35): a new 32bpp ARGB
// bitmap cleared to bg, with the source drawn over it at 1:1 (GDI+
// SourceOver). Opaque pixels are copied, fully transparent pixels become bg,
// and partially transparent pixels are blended as
// round(c*a/255) + bg*(255-a)/255. The result is fully opaque when bg is.
func SetImageBackground(img image.Image, bg color.NRGBA) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := nrgbaAt(img, b.Min.X+x, b.Min.Y+y)
			o := out.PixOffset(x, y)
			switch c.A {
			case 255:
				out.Pix[o+0], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = c.R, c.G, c.B, 255
			case 0:
				out.Pix[o+0], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = bg.R, bg.G, bg.B, bg.A
			default:
				a := uint32(c.A)
				blend := func(s, d uint8) uint8 {
					return uint8((uint32(s)*a+127)/255 + (uint32(d)*(255-a)+127)/255)
				}
				outA := a + (uint32(bg.A)*(255-a)+127)/255
				out.Pix[o+0], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] =
					blend(c.R, bg.R), blend(c.G, bg.G), blend(c.B, bg.B), uint8(outA)
			}
		}
	}
	return out
}

// dblEpsilon is WPF DoubleUtil.DBL_EPSILON.
const dblEpsilon = 2.2204460492503131e-016

// isOne ports WPF DoubleUtil.IsOne, the check TransformedBitmap uses to decide
// whether a ScaleTransform needs a WIC scaler at all.
func isOne(v float64) bool { return math.Abs(v-1.0) < 10.0*dblEpsilon }

// ResizeImage ports ImageScaler.ResizeImage
// (ACELogoConvertor/ICULogoConvertor/ImageScaler.cs:37-75): clamp the requested size to the
// source size (no upscaling), take the smaller of the two axis ratios, and
// scale both axes by it through a WPF TransformedBitmap(ScaleTransform).
//
// TransformedBitmap.FinalizeCreation skips scaling when DoubleUtil.IsOne(scale)
// and otherwise sizes the output as Math.Max(1, (uint)(scale*PixelWidth+0.5))
// with WICInterpolationMode.Fant; both are reproduced, so a 0 target yields a
// 1x1 image. The GDI+ -> HBITMAP -> WIC -> BMP -> GDI+ round trip is lossless
// for the opaque image passed in.
func ResizeImage(img *image.NRGBA, newWidth, newHeight int) *image.NRGBA {
	width, height := img.Rect.Dx(), img.Rect.Dy()
	if width < newWidth {
		newWidth = width
	}
	if height < newHeight {
		newHeight = height
	}
	val := float64(newWidth) / float64(width)
	val2 := float64(newHeight) / float64(height)
	num := math.Min(val, val2)
	// GetParamsFromTransform: scale = Math.Abs(M11/M22); a negative scale
	// becomes a horizontal + vertical flip applied after the scaler.
	scale := math.Abs(num)
	out := img
	if !isOne(scale) {
		// float64(...) conversions stop the compiler fusing x*y+0.5 into an FMA.
		dw := max(1, int(uint32(float64(scale*float64(width))+0.5)))
		dh := max(1, int(uint32(float64(scale*float64(height))+0.5)))
		out = fantResize(img, dw, dh)
	} else {
		out = image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			copy(out.Pix[y*out.Stride:y*out.Stride+width*4], img.Pix[img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y+y):])
		}
	}
	if num < 0 {
		out = rotate180(out)
	}
	return out
}

// rotate180 is WICBitmapTransformFlipHorizontal | FlipVertical.
func rotate180(src *image.NRGBA) *image.NRGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetNRGBA(w-1-x, h-1-y, src.NRGBAAt(src.Rect.Min.X+x, src.Rect.Min.Y+y))
		}
	}
	return out
}

// netFloorToInt is (int)Math.Floor(v) on x64 .NET: NaN and out-of-range
// values become int.MinValue (cvttsd2si's "integer indefinite").
func netFloorToInt(v float64) int {
	f := math.Floor(v)
	if math.IsNaN(f) || f < math.MinInt32 || f > math.MaxInt32 {
		return math.MinInt32
	}
	return int(f)
}

// fantResize approximates WICInterpolationMode.Fant for downscaling: every
// output pixel is the area-weighted mean of the source pixels its footprint
// covers (fractional coverage at the footprint edges), rounded to nearest.
// windowscodecs.dll's fixed-point implementation is not public, so exact
// byte equality with WIC is unverified.
func fantResize(src *image.NRGBA, dw, dh int) *image.NRGBA {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	xw := fantWeights(sw, dw)
	yw := fantWeights(sh, dh)
	out := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for oy := 0; oy < dh; oy++ {
		for ox := 0; ox < dw; ox++ {
			var acc [4]float64
			var total float64
			for _, wy := range yw[oy] {
				for _, wx := range xw[ox] {
					w := float64(wy.w * wx.w)
					p := src.PixOffset(src.Rect.Min.X+wx.i, src.Rect.Min.Y+wy.i)
					a := float64(src.Pix[p+3])
					acc[0] += float64(float64(src.Pix[p+0])*a) * w
					acc[1] += float64(float64(src.Pix[p+1])*a) * w
					acc[2] += float64(float64(src.Pix[p+2])*a) * w
					acc[3] += float64(a * w)
					total += w
				}
			}
			o := out.PixOffset(ox, oy)
			if acc[3] > 0 {
				for c := 0; c < 3; c++ {
					out.Pix[o+c] = clamp8(acc[c] / acc[3])
				}
			}
			out.Pix[o+3] = clamp8(acc[3] / total)
		}
	}
	return out
}

type fantTap struct {
	i int
	w float64
}

// fantWeights returns, per output index, the source indices and coverage of
// the box [o*ratio, (o+1)*ratio).
func fantWeights(srcN, dstN int) [][]fantTap {
	ratio := float64(srcN) / float64(dstN)
	res := make([][]fantTap, dstN)
	for o := 0; o < dstN; o++ {
		lo := float64(o) * ratio
		hi := float64(o+1) * ratio
		if hi > float64(srcN) {
			hi = float64(srcN)
		}
		for i := int(math.Floor(lo)); i < srcN && float64(i) < hi; i++ {
			l := math.Max(lo, float64(i))
			r := math.Min(hi, float64(i+1))
			if r > l {
				res[o] = append(res[o], fantTap{i: i, w: r - l})
			}
		}
	}
	return res
}

func clamp8(v float64) uint8 {
	v = math.Floor(v + 0.5)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// nrgbaAt reads a pixel as non-premultiplied 8-bit RGBA (what GDI+ hands the
// SimplePaletteQuantizer pixel structs).
func nrgbaAt(img image.Image, x, y int) color.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		o := n.PixOffset(x, y)
		return color.NRGBA{R: n.Pix[o], G: n.Pix[o+1], B: n.Pix[o+2], A: n.Pix[o+3]}
	}
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}
