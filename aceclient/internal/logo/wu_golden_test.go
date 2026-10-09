package logo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"strconv"
	"strings"
	"testing"
)

// testdata/wu_golden.json was produced by compiling the decompiled, unmodified
// WuColorQuantizer / WuColorCube / BaseColorQuantizer / QuantizationHelper /
// StandardPathProvider C# (ACELogoConvertor) with a stub ImageBuffer and
// running it over raw RGB dumps of exactly the images below (the logo_alfen
// case is firmware/msi_work/files3/logo_alfen.png through SetImageBackground).
// The harness lives outside the repo; only its output is checked in.

type xorshift32 uint32

func (x *xorshift32) next() uint32 {
	v := uint32(*x)
	v ^= v << 13
	v ^= v >> 17
	v ^= v << 5
	*x = xorshift32(v)
	return v
}

// syntheticCases are deterministic test images for the Wu golden data.
func syntheticCases() map[string]*image.NRGBA {
	mk := func(w, h int, f func(x, y int) color.NRGBA) *image.NRGBA {
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.SetNRGBA(x, y, f(x, y))
			}
		}
		return img
	}
	out := map[string]*image.NRGBA{}
	rng := xorshift32(0x12345678)
	random := func(x, y int) color.NRGBA {
		v := rng.next()
		return color.NRGBA{uint8(v), uint8(v >> 8), uint8(v >> 16), 255}
	}
	out["random64x48"] = mk(64, 48, random)
	rng = xorshift32(0x9E3779B9)
	out["random320x160"] = mk(320, 160, random)
	out["gradient200x100"] = mk(200, 100, func(x, y int) color.NRGBA {
		return color.NRGBA{uint8(x * 255 / 199), uint8(y * 255 / 99), uint8(x * y), 255}
	})
	five := []color.NRGBA{{255, 255, 255, 255}, {0, 0, 0, 255}, {200, 30, 30, 255}, {30, 200, 30, 255}, {30, 30, 200, 255}}
	out["fewcolors50x50"] = mk(50, 50, func(x, y int) color.NRGBA { return five[(x/10+y/10)%5] })
	out["logo800x350"] = mk(800, 350, func(x, y int) color.NRGBA {
		dx, dy := x-400, y-175
		switch {
		case dx*dx+dy*dy < 150*150:
			return color.NRGBA{uint8(200 + x%56), 6, 19, 255}
		case x > 600 && y < 100:
			return color.NRGBA{0, uint8(80 + y), 160, 255}
		}
		return color.NRGBA{255, 255, 255, 255}
	})
	return out
}

const logoAlfenPNG = "../../../firmware/msi_work/files3/logo_alfen.png"

type wuGolden struct {
	Name          string     `json:"name"`
	Width         int        `json:"width"`
	Height        int        `json:"height"`
	ColorCount    int        `json:"colorCount"`
	Palette       [][3]uint8 `json:"palette"`
	IndicesSHA256 string     `json:"indicesSHA256"`
}

func loadWuGolden(t *testing.T) []wuGolden {
	t.Helper()
	raw, err := os.ReadFile("testdata/wu_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var g []wuGolden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func goldenInput(t *testing.T, name string) *image.NRGBA {
	t.Helper()
	if name == "logo_alfen" {
		img, _, err := LoadImage(logoAlfenPNG)
		if err != nil {
			t.Skipf("fixture missing: %v", err)
		}
		return SetImageBackground(img, White)
	}
	img, ok := syntheticCases()[name]
	if !ok {
		t.Fatalf("unknown golden case %q", name)
	}
	return img
}

// TestWuMatchesDecompiledCSharp runs the Go Wu port the way the harness drove
// the C# (Prepare, AddColor row-major, GetPalette, GetPaletteIndex) and
// requires an identical palette and identical per-pixel indices.
func TestWuMatchesDecompiledCSharp(t *testing.T) {
	for _, g := range loadWuGolden(t) {
		name, countText, _ := strings.Cut(g.Name, "__")
		count, err := strconv.Atoi(countText)
		if err != nil {
			t.Fatalf("bad golden name %q", g.Name)
		}
		t.Run(g.Name, func(t *testing.T) {
			img := goldenInput(t, name)
			if img.Rect.Dx() != g.Width || img.Rect.Dy() != g.Height || count != g.ColorCount {
				t.Fatalf("input %dx%d/%d, golden %dx%d/%d", img.Rect.Dx(), img.Rect.Dy(), count, g.Width, g.Height, g.ColorCount)
			}
			q := &wuColorQuantizer{}
			q.onPrepare(g.Width, g.Height)
			for y := 0; y < g.Height; y++ {
				for x := 0; x < g.Width; x++ {
					c, _ := convertAlpha(img.NRGBAAt(x, y))
					q.onAddColor(c)
				}
			}
			pal := q.onGetPalette(g.ColorCount)
			if len(pal) != len(g.Palette) {
				t.Fatalf("palette len %d, C# %d", len(pal), len(g.Palette))
			}
			for i, c := range pal {
				if want := g.Palette[i]; c.R != want[0] || c.G != want[1] || c.B != want[2] || c.A != 255 {
					t.Fatalf("palette[%d] = %v, C# %v", i, c, want)
				}
			}
			idx := make([]byte, g.Width*g.Height)
			for y := 0; y < g.Height; y++ {
				for x := 0; x < g.Width; x++ {
					idx[y*g.Width+x] = byte(q.onGetPaletteIndex(x, y))
				}
			}
			sum := sha256.Sum256(idx)
			if got := hex.EncodeToString(sum[:]); got != g.IndicesSHA256 {
				t.Fatalf("indices sha256 %s, C# %s", got, g.IndicesSHA256)
			}

			// QuantizeImage must produce the same indices and Wu entries,
			// with the GDI+ default palette behind them.
			qi, err := QuantizeImage(img, g.ColorCount)
			if err != nil {
				t.Fatal(err)
			}
			sum2 := sha256.Sum256(qi.Pix)
			if hex.EncodeToString(sum2[:]) != g.IndicesSHA256 {
				t.Fatalf("QuantizeImage indices differ from C#")
			}
			entries, _ := paletteEntriesByColorCount(g.ColorCount)
			def := gdiplusDefaultPalette(entries)
			if len(qi.Palette) != entries {
				t.Fatalf("palette entries %d, want %d", len(qi.Palette), entries)
			}
			for i := range qi.Palette {
				want := def[i]
				if i < len(g.Palette) {
					want = color.NRGBA{g.Palette[i][0], g.Palette[i][1], g.Palette[i][2], 255}
				}
				if qi.Palette[i] != want {
					t.Fatalf("QuantizeImage palette[%d] = %v, want %v", i, qi.Palette[i], want)
				}
			}
		})
	}
}
