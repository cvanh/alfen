package logo

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestGDIPlusDefaultPalette(t *testing.T) {
	p := gdiplusDefaultPalette(256)
	if len(p) != 256 {
		t.Fatalf("len %d", len(p))
	}
	tests := []struct {
		i    int
		want color.NRGBA
	}{
		{0, color.NRGBA{0, 0, 0, 255}},
		{1, color.NRGBA{0x80, 0, 0, 255}},
		{2, color.NRGBA{0, 0x80, 0, 255}},
		{4, color.NRGBA{0, 0, 0x80, 255}},
		{7, color.NRGBA{0x80, 0x80, 0x80, 255}},
		{8, color.NRGBA{0xc0, 0xc0, 0xc0, 255}},
		{9, color.NRGBA{0xff, 0, 0, 255}},
		{15, color.NRGBA{0xff, 0xff, 0xff, 255}},
		{16, color.NRGBA{}},
		{39, color.NRGBA{}},
		{40, color.NRGBA{0, 0, 0, 255}},
		{41, color.NRGBA{0, 0, 0x33, 255}},
		{46, color.NRGBA{0, 0x33, 0, 255}},
		{76, color.NRGBA{0x33, 0, 0, 255}},
		{255, color.NRGBA{0xff, 0xff, 0xff, 255}},
	}
	for _, tc := range tests {
		if p[tc.i] != tc.want {
			t.Errorf("entry %d = %v, want %v", tc.i, p[tc.i], tc.want)
		}
	}
	p16 := gdiplusDefaultPalette(16)
	for i := range p16 {
		if p16[i] != p[i] {
			t.Errorf("4bpp entry %d = %v, want %v", i, p16[i], p[i])
		}
	}
	p2 := gdiplusDefaultPalette(2)
	if p2[0] != (color.NRGBA{0, 0, 0, 255}) || p2[1] != (color.NRGBA{255, 255, 255, 255}) {
		t.Errorf("1bpp palette %v", p2)
	}
}

func TestPaletteEntriesByColorCount(t *testing.T) {
	tests := []struct {
		count, want int
		err         bool
	}{
		{0, 0, true}, {1, 2, false}, {2, 2, false}, {3, 16, false}, {16, 16, false},
		{17, 256, false}, {128, 256, false}, {256, 256, false}, {257, 0, true},
	}
	for _, tc := range tests {
		got, err := paletteEntriesByColorCount(tc.count)
		if (err != nil) != tc.err || got != tc.want {
			t.Errorf("%d: got %d, %v", tc.count, got, err)
		}
	}
	if _, err := paletteEntriesByColorCount(300); err == nil || err.Error() != "A color count '300' not supported!" {
		t.Errorf("message %v", err)
	}
}

func TestConvertAlpha(t *testing.T) {
	c, argb := convertAlpha(color.NRGBA{10, 20, 30, 255})
	if c != (color.NRGBA{10, 20, 30, 255}) || argb != 10<<16|20<<8|30 {
		t.Fatalf("opaque: %v %x", c, argb)
	}
	c, _ = convertAlpha(color.NRGBA{10, 20, 30, 0})
	if c != (color.NRGBA{240, 240, 240, 255}) {
		t.Fatalf("transparent: %v", c)
	}
	// (int)(100*128/255 + 240*127/255) = (int)(50.196 + 119.529) = 169
	c, _ = convertAlpha(color.NRGBA{100, 100, 100, 128})
	if c.R != 169 || c.A != 255 {
		t.Fatalf("half: %v", c)
	}
}

func TestSetImageBackground(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	src.SetNRGBA(0, 0, color.NRGBA{10, 20, 30, 255})
	src.SetNRGBA(1, 0, color.NRGBA{10, 20, 30, 0})
	src.SetNRGBA(2, 0, color.NRGBA{0, 100, 200, 128})
	out := SetImageBackground(src, White)
	want := []color.NRGBA{
		{10, 20, 30, 255},
		{255, 255, 255, 255},
		// round(c*128/255) + round(255*127/255)
		{0 + 127, 50 + 127, 100 + 127, 255},
	}
	for i, w := range want {
		if got := out.NRGBAAt(i, 0); got != w {
			t.Errorf("pixel %d = %v, want %v", i, got, w)
		}
	}
}

func TestResizeImageSizes(t *testing.T) {
	tests := []struct {
		name         string
		w, h, nw, nh int
		wantW, wantH int
	}{
		{"no upscale", 100, 50, 304, 144, 100, 50},
		{"exact fit is identity", 304, 144, 304, 144, 304, 144},
		{"half", 1000, 500, 288, 144, 288, 144},
		// min(28/100, 144/500) = 0.28 -> (uint)(0.28*500+0.5) = 140
		{"axis rounding", 100, 500, 28, 144, 28, 140},
		{"one pixel floor", 1000, 3, 1, 3, 1, 1},
	}
	for _, tc := range tests {
		img := image.NewNRGBA(image.Rect(0, 0, tc.w, tc.h))
		for i := range img.Pix {
			img.Pix[i] = 255
		}
		out := ResizeImage(img, tc.nw, tc.nh)
		if out.Rect.Dx() != tc.wantW || out.Rect.Dy() != tc.wantH {
			t.Errorf("%s: got %dx%d, want %dx%d", tc.name, out.Rect.Dx(), out.Rect.Dy(), tc.wantW, tc.wantH)
		}
	}
}

func TestResizeImageNegativeScaleFlips(t *testing.T) {
	// min(-2/4, -1/1) = -1: |scale| is one (no scaler), the sign flips both axes.
	src := image.NewNRGBA(image.Rect(0, 0, 4, 1))
	for x, v := range []uint8{10, 20, 30, 40} {
		src.SetNRGBA(x, 0, color.NRGBA{v, v, v, 255})
	}
	out := ResizeImage(src, -2, -1)
	for x, v := range []uint8{40, 30, 20, 10} {
		if got := out.NRGBAAt(x, 0).R; got != v {
			t.Fatalf("pixel %d = %d, want %d", x, got, v)
		}
	}
}

func TestFantResizeAreaAverage(t *testing.T) {
	// 3 -> 2: [0,1.5) = (0 + 90*0.5)/1.5 = 30, [1.5,3) = (90*0.5 + 180)/1.5 = 150.
	src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	for x, v := range []uint8{0, 90, 180} {
		src.SetNRGBA(x, 0, color.NRGBA{v, v, v, 255})
	}
	out := fantResize(src, 2, 1)
	if a, b := out.NRGBAAt(0, 0).R, out.NRGBAAt(1, 0).R; a != 30 || b != 150 {
		t.Fatalf("got %d,%d want 30,150", a, b)
	}
	// 4x4 -> 2x2 averages each 2x2 block.
	src = image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			v := uint8(x*10 + y*40)
			src.SetNRGBA(x, y, color.NRGBA{v, v, v, 255})
		}
	}
	out = fantResize(src, 2, 2)
	if got := out.NRGBAAt(1, 1).R; got != (100+110+140+150)/4 { // x in {2,3}, y in {2,3}
		t.Fatalf("block avg %d", got)
	}
}

func TestQuantizeDeterministic(t *testing.T) {
	img := syntheticCases()["gradient200x100"]
	a, err := QuantizeImage(img, 128)
	if err != nil {
		t.Fatal(err)
	}
	b, err := QuantizeImage(img, 128)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("indices differ between runs")
	}
	for i := range a.Palette {
		if a.Palette[i] != b.Palette[i] {
			t.Fatalf("palette[%d] differs", i)
		}
	}
}

func TestConvertImage(t *testing.T) {
	img, _, err := LoadImage(logoAlfenPNG)
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	// DlgUploadResources: 320x160 device, margin 8 -> desired 304x144.
	out, err := ConvertImage(img, 304, 144, 128)
	if err != nil {
		t.Fatal(err)
	}
	w, h := out.Rect.Dx(), out.Rect.Dy()
	if w > 304 || h > 144 || w < 300 {
		t.Fatalf("converted %dx%d", w, h)
	}
	if len(out.Palette) != 256 {
		t.Fatalf("palette entries %d", len(out.Palette))
	}
	for i := 128; i < 256; i++ {
		if out.Palette[i] != (color.NRGBA{255, 0, 0, 255}) {
			t.Fatalf("entry %d = %v, want red", i, out.Palette[i])
		}
	}
	for i, p := range out.Pix {
		if p >= 128 {
			t.Fatalf("pixel %d index %d >= maxColors", i, p)
		}
	}
	// A small image is not upscaled.
	small := image.NewNRGBA(image.Rect(0, 0, 10, 5))
	o2, err := ConvertImage(small, 304, 144, 128)
	if err != nil || o2.Rect.Dx() != 10 || o2.Rect.Dy() != 5 {
		t.Fatalf("small: %v %v", o2.Rect, err)
	}
	// Transparent pixels become white and the palette shrinks to one colour.
	if o2.Palette[0] != (color.NRGBA{255, 255, 255, 255}) {
		t.Fatalf("transparent -> %v", o2.Palette[0])
	}
	// A 0 desired size is not an error in C#: num = +Inf, Size(0, 0), and
	// TransformedBitmap clamps the scaled size to 1x1.
	if o3, err := ConvertImage(small, 0, 144, 128); err != nil || o3.Rect.Dx() != 1 || o3.Rect.Dy() != 1 {
		t.Fatalf("zero desired width: %v %v", o3, err)
	}
	if _, err := ConvertImage(image.NewNRGBA(image.Rect(0, 0, 0, 5)), 304, 144, 128); err == nil {
		t.Fatal("empty image accepted")
	}
	if _, err := ConvertImage(small, 304, 144, 0); err == nil {
		t.Fatal("maxColors 0 accepted")
	}
}
