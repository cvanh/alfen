package logo

import "image/color"

// Wu colour-space constants from WuColorQuantizer.
const (
	wuMaxColor     = 512 // MaxColor
	wuRed          = 2   // Red
	wuGreen        = 1   // Green
	wuBlue         = 0   // Blue
	wuSideSize     = 33  // SideSize
	wuMaxSideIndex = 32  // MaxSideIndex
	wuMaxVolume    = 35937
)

// wuColorCube ports SimplePaletteQuantizer.Quantizers.XiaolinWu.WuColorCube
// (ACELogoConvertor/SimplePaletteQuantizer.Quantizers.XiaolinWu/WuColorCube.cs).
type wuColorCube struct {
	RedMinimum, RedMaximum     int
	GreenMinimum, GreenMaximum int
	BlueMinimum, BlueMaximum   int
	Volume                     int
}

type wuMoments [wuSideSize][wuSideSize][wuSideSize]int64
type wuMomentsF [wuSideSize][wuSideSize][wuSideSize]float32

// wuColorQuantizer ports WuColorQuantizer
// (ACELogoConvertor/SimplePaletteQuantizer.Quantizers.XiaolinWu/WuColorQuantizer.cs).
//
// All `float` arithmetic of the C# is kept in float32 (the installer is an
// AnyCPU, not-32-bit-preferred assembly, so it runs on x64 RyuJIT with SSE
// single precision). Products are wrapped in explicit float32(...) conversions
// so the Go compiler cannot fuse them into FMA instructions.
type wuColorQuantizer struct {
	reds, greens, blues, sums []int
	indices                   []int

	weights, momentsRed, momentsGreen, momentsBlue *wuMoments
	moments                                        *wuMomentsF

	tag             []int
	quantizedPixels []int
	table           []int
	pixels          []uint32 // Color.ToArgb()

	imageWidth int
	imageSize  int
	pixelIndex int

	cubes []*wuColorCube
}

// calculateMoments ports WuColorQuantizer.CalculateMoments (WuColorQuantizer.cs:61-105).
func (q *wuColorQuantizer) calculateMoments() {
	var array, array2, array3, array4 [wuSideSize]int64
	var array5 [wuSideSize]float32
	for i := 1; i <= wuMaxSideIndex; i++ {
		for j := 0; j <= wuMaxSideIndex; j++ {
			array[j], array2[j], array3[j], array4[j] = 0, 0, 0, 0
			array5[j] = 0
		}
		for k := 1; k <= wuMaxSideIndex; k++ {
			var num, num2, num3, num4 int64
			var num5 float32
			for l := 1; l <= wuMaxSideIndex; l++ {
				num += q.weights[i][k][l]
				num2 += q.momentsRed[i][k][l]
				num3 += q.momentsGreen[i][k][l]
				num4 += q.momentsBlue[i][k][l]
				num5 += q.moments[i][k][l]
				array[l] += num
				array2[l] += num2
				array3[l] += num3
				array4[l] += num4
				array5[l] += num5
				q.weights[i][k][l] = q.weights[i-1][k][l] + array[l]
				q.momentsRed[i][k][l] = q.momentsRed[i-1][k][l] + array2[l]
				q.momentsGreen[i][k][l] = q.momentsGreen[i-1][k][l] + array3[l]
				q.momentsBlue[i][k][l] = q.momentsBlue[i-1][k][l] + array4[l]
				q.moments[i][k][l] = q.moments[i-1][k][l] + array5[l]
			}
		}
	}
}

// wuVolume ports WuColorQuantizer.Volume (WuColorQuantizer.cs:107-110).
func wuVolume(c *wuColorCube, m *wuMoments) int64 {
	return m[c.RedMaximum][c.GreenMaximum][c.BlueMaximum] -
		m[c.RedMaximum][c.GreenMaximum][c.BlueMinimum] -
		m[c.RedMaximum][c.GreenMinimum][c.BlueMaximum] +
		m[c.RedMaximum][c.GreenMinimum][c.BlueMinimum] -
		m[c.RedMinimum][c.GreenMaximum][c.BlueMaximum] +
		m[c.RedMinimum][c.GreenMaximum][c.BlueMinimum] +
		m[c.RedMinimum][c.GreenMinimum][c.BlueMaximum] -
		m[c.RedMinimum][c.GreenMinimum][c.BlueMinimum]
}

// wuVolumeFloat ports WuColorQuantizer.VolumeFloat (WuColorQuantizer.cs:112-115),
// left-to-right float32.
func wuVolumeFloat(c *wuColorCube, m *wuMomentsF) float32 {
	return m[c.RedMaximum][c.GreenMaximum][c.BlueMaximum] -
		m[c.RedMaximum][c.GreenMaximum][c.BlueMinimum] -
		m[c.RedMaximum][c.GreenMinimum][c.BlueMaximum] +
		m[c.RedMaximum][c.GreenMinimum][c.BlueMinimum] -
		m[c.RedMinimum][c.GreenMaximum][c.BlueMaximum] +
		m[c.RedMinimum][c.GreenMaximum][c.BlueMinimum] +
		m[c.RedMinimum][c.GreenMinimum][c.BlueMaximum] -
		m[c.RedMinimum][c.GreenMinimum][c.BlueMinimum]
}

// wuTop ports WuColorQuantizer.Top (WuColorQuantizer.cs:117-126).
func wuTop(c *wuColorCube, direction, position int, m *wuMoments) int64 {
	switch direction {
	case wuRed:
		return m[position][c.GreenMaximum][c.BlueMaximum] -
			m[position][c.GreenMaximum][c.BlueMinimum] -
			m[position][c.GreenMinimum][c.BlueMaximum] +
			m[position][c.GreenMinimum][c.BlueMinimum]
	case wuGreen:
		return m[c.RedMaximum][position][c.BlueMaximum] -
			m[c.RedMaximum][position][c.BlueMinimum] -
			m[c.RedMinimum][position][c.BlueMaximum] +
			m[c.RedMinimum][position][c.BlueMinimum]
	case wuBlue:
		return m[c.RedMaximum][c.GreenMaximum][position] -
			m[c.RedMaximum][c.GreenMinimum][position] -
			m[c.RedMinimum][c.GreenMaximum][position] +
			m[c.RedMinimum][c.GreenMinimum][position]
	}
	return 0
}

// wuBottom ports WuColorQuantizer.Bottom (WuColorQuantizer.cs:128-137).
func wuBottom(c *wuColorCube, direction int, m *wuMoments) int64 {
	switch direction {
	case wuRed:
		return -m[c.RedMinimum][c.GreenMaximum][c.BlueMaximum] +
			m[c.RedMinimum][c.GreenMaximum][c.BlueMinimum] +
			m[c.RedMinimum][c.GreenMinimum][c.BlueMaximum] -
			m[c.RedMinimum][c.GreenMinimum][c.BlueMinimum]
	case wuGreen:
		return -m[c.RedMaximum][c.GreenMinimum][c.BlueMaximum] +
			m[c.RedMaximum][c.GreenMinimum][c.BlueMinimum] +
			m[c.RedMinimum][c.GreenMinimum][c.BlueMaximum] -
			m[c.RedMinimum][c.GreenMinimum][c.BlueMinimum]
	case wuBlue:
		return -m[c.RedMaximum][c.GreenMaximum][c.BlueMinimum] +
			m[c.RedMaximum][c.GreenMinimum][c.BlueMinimum] +
			m[c.RedMinimum][c.GreenMaximum][c.BlueMinimum] -
			m[c.RedMinimum][c.GreenMinimum][c.BlueMinimum]
	}
	return 0
}

// calculateVariance ports WuColorQuantizer.CalculateVariance (WuColorQuantizer.cs:139-148).
func (q *wuColorQuantizer) calculateVariance(c *wuColorCube) float32 {
	num := float32(wuVolume(c, q.momentsRed))
	num2 := float32(wuVolume(c, q.momentsGreen))
	num3 := float32(wuVolume(c, q.momentsBlue))
	num4 := wuVolumeFloat(c, q.moments)
	num5 := float32(wuVolume(c, q.weights))
	num6 := float32(num*num) + float32(num2*num2) + float32(num3*num3)
	return num4 - num6/num5
}

// maximize ports WuColorQuantizer.Maximize (WuColorQuantizer.cs:150-186). The squares are long (int64)
// arithmetic in the C#, converted to float afterwards.
func (q *wuColorQuantizer) maximize(c *wuColorCube, direction, first, last int, cut []int,
	wholeRed, wholeGreen, wholeBlue, wholeWeight int64) float32 {
	num := wuBottom(c, direction, q.momentsRed)
	num2 := wuBottom(c, direction, q.momentsGreen)
	num3 := wuBottom(c, direction, q.momentsBlue)
	num4 := wuBottom(c, direction, q.weights)
	var num5 float32
	cut[0] = -1
	for i := first; i < last; i++ {
		num6 := num + wuTop(c, direction, i, q.momentsRed)
		num7 := num2 + wuTop(c, direction, i, q.momentsGreen)
		num8 := num3 + wuTop(c, direction, i, q.momentsBlue)
		num9 := num4 + wuTop(c, direction, i, q.weights)
		if num9 == 0 {
			continue
		}
		num10 := float32(num6*num6 + num7*num7 + num8*num8)
		num11 := num10 / float32(num9)
		num6 = wholeRed - num6
		num7 = wholeGreen - num7
		num8 = wholeBlue - num8
		num9 = wholeWeight - num9
		if num9 != 0 {
			num10 = float32(num6*num6 + num7*num7 + num8*num8)
			num11 += num10 / float32(num9)
			if num11 > num5 {
				num5 = num11
				cut[0] = i
			}
		}
	}
	return num5
}

// cut ports WuColorQuantizer.Cut (WuColorQuantizer.cs:188-246), including its asymmetry: only a failed red
// cut aborts; green/blue are taken as chosen.
func (q *wuColorQuantizer) cut(first, second *wuColorCube) bool {
	array := []int{0}
	array2 := []int{0}
	array3 := []int{0}
	wholeRed := wuVolume(first, q.momentsRed)
	wholeGreen := wuVolume(first, q.momentsGreen)
	wholeBlue := wuVolume(first, q.momentsBlue)
	wholeWeight := wuVolume(first, q.weights)
	num := q.maximize(first, wuRed, first.RedMinimum+1, first.RedMaximum, array, wholeRed, wholeGreen, wholeBlue, wholeWeight)
	num2 := q.maximize(first, wuGreen, first.GreenMinimum+1, first.GreenMaximum, array2, wholeRed, wholeGreen, wholeBlue, wholeWeight)
	num3 := q.maximize(first, wuBlue, first.BlueMinimum+1, first.BlueMaximum, array3, wholeRed, wholeGreen, wholeBlue, wholeWeight)
	var num4 int
	if !(num >= num2) || !(num >= num3) {
		if num2 >= num && num2 >= num3 {
			num4 = wuGreen
		} else {
			num4 = wuBlue
		}
	} else {
		num4 = wuRed
		if array[0] < 0 {
			return false
		}
	}
	second.RedMaximum = first.RedMaximum
	second.GreenMaximum = first.GreenMaximum
	second.BlueMaximum = first.BlueMaximum
	switch num4 {
	case wuRed:
		first.RedMaximum = array[0]
		second.RedMinimum = first.RedMaximum
		second.GreenMinimum = first.GreenMinimum
		second.BlueMinimum = first.BlueMinimum
	case wuGreen:
		first.GreenMaximum = array2[0]
		second.GreenMinimum = first.GreenMaximum
		second.RedMinimum = first.RedMinimum
		second.BlueMinimum = first.BlueMinimum
	case wuBlue:
		first.BlueMaximum = array3[0]
		second.BlueMinimum = first.BlueMaximum
		second.RedMinimum = first.RedMinimum
		second.GreenMinimum = first.GreenMinimum
	}
	first.Volume = (first.RedMaximum - first.RedMinimum) * (first.GreenMaximum - first.GreenMinimum) * (first.BlueMaximum - first.BlueMinimum)
	second.Volume = (second.RedMaximum - second.RedMinimum) * (second.GreenMaximum - second.GreenMinimum) * (second.BlueMaximum - second.BlueMinimum)
	return true
}

// wuTagIndex is (i << 10) + (i << 6) + i + (j << 5) + j + k, i.e. i*1089+j*33+k.
func wuTagIndex(i, j, k int) int { return (i << 10) + (i << 6) + i + (j << 5) + j + k }

// wuMark ports WuColorQuantizer.Mark (WuColorQuantizer.cs:248-260).
func wuMark(c *wuColorCube, label int, tag []int) {
	for i := c.RedMinimum + 1; i <= c.RedMaximum; i++ {
		for j := c.GreenMinimum + 1; j <= c.GreenMaximum; j++ {
			for k := c.BlueMinimum + 1; k <= c.BlueMaximum; k++ {
				tag[wuTagIndex(i, j, k)] = label
			}
		}
	}
}

// onPrepare ports WuColorQuantizer.OnPrepare (WuColorQuantizer.cs:262-290; it does NOT call
// BaseColorQuantizer.OnPrepare).
func (q *wuColorQuantizer) onPrepare(width, height int) {
	q.cubes = make([]*wuColorCube, wuMaxColor)
	for i := range q.cubes {
		q.cubes[i] = &wuColorCube{}
	}
	q.cubes[0].RedMinimum = 0
	q.cubes[0].GreenMinimum = 0
	q.cubes[0].BlueMinimum = 0
	q.cubes[0].RedMaximum = wuMaxSideIndex
	q.cubes[0].GreenMaximum = wuMaxSideIndex
	q.cubes[0].BlueMaximum = wuMaxSideIndex
	q.weights = new(wuMoments)
	q.momentsRed = new(wuMoments)
	q.momentsGreen = new(wuMoments)
	q.momentsBlue = new(wuMoments)
	q.moments = new(wuMomentsF)
	q.table = make([]int, 256)
	for j := range q.table {
		q.table[j] = j * j
	}
	q.pixelIndex = 0
	q.imageWidth = width
	q.imageSize = width * height
	q.quantizedPixels = make([]int, q.imageSize)
	q.pixels = make([]uint32, q.imageSize)
}

// onAddColor ports WuColorQuantizer.OnAddColor (WuColorQuantizer.cs:292-305; pixels arrive in
// StandardPathProvider order, row by row).
func (q *wuColorQuantizer) onAddColor(c color.NRGBA) {
	num := int(c.R>>3) + 1
	num2 := int(c.G>>3) + 1
	num3 := int(c.B>>3) + 1
	q.weights[num][num2][num3]++
	q.momentsRed[num][num2][num3] += int64(c.R)
	q.momentsGreen[num][num2][num3] += int64(c.G)
	q.momentsBlue[num][num2][num3] += int64(c.B)
	q.moments[num][num2][num3] += float32(q.table[c.R] + q.table[c.G] + q.table[c.B])
	q.quantizedPixels[q.pixelIndex] = wuTagIndex(num, num2, num3)
	q.pixels[q.pixelIndex] = uint32(c.A)<<24 | uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
	q.pixelIndex++
}

// onGetPalette ports WuColorQuantizer.OnGetPalette (WuColorQuantizer.cs:307-410): split the RGB cube into up
// to colorCount boxes, then map every scanned pixel to the nearest box mean
// and return the per-index average of the pixels mapped to it.
func (q *wuColorQuantizer) onGetPalette(colorCount int) []color.NRGBA {
	q.calculateMoments()
	num := 0
	array := make([]float32, wuMaxColor)
	for i := 1; i < colorCount; i++ {
		if q.cut(q.cubes[num], q.cubes[i]) {
			if q.cubes[num].Volume > 1 {
				array[num] = q.calculateVariance(q.cubes[num])
			} else {
				array[num] = 0
			}
			if q.cubes[i].Volume > 1 {
				array[i] = q.calculateVariance(q.cubes[i])
			} else {
				array[i] = 0
			}
		} else {
			array[num] = 0
			i--
		}
		num = 0
		num2 := array[0]
		for j := 1; j <= i; j++ {
			if array[j] > num2 {
				num2 = array[j]
				num = j
			}
		}
		if float64(num2) <= 0.0 {
			colorCount = i + 1
			break
		}
	}
	array2 := make([]int, wuMaxColor)
	array3 := make([]int, wuMaxColor)
	array4 := make([]int, wuMaxColor)
	q.tag = make([]int, wuMaxVolume)
	for k := 0; k < colorCount; k++ {
		wuMark(q.cubes[k], k, q.tag)
		num3 := wuVolume(q.cubes[k], q.weights)
		if num3 > 0 {
			array2[k] = int(wuVolume(q.cubes[k], q.momentsRed) / num3)
			array3[k] = int(wuVolume(q.cubes[k], q.momentsGreen) / num3)
			array4[k] = int(wuVolume(q.cubes[k], q.momentsBlue) / num3)
		} else {
			array2[k], array3[k], array4[k] = 0, 0, 0
		}
	}
	for l := 0; l < q.imageSize; l++ {
		q.quantizedPixels[l] = q.tag[q.quantizedPixels[l]]
	}
	q.reds = make([]int, colorCount+1)
	q.greens = make([]int, colorCount+1)
	q.blues = make([]int, colorCount+1)
	q.sums = make([]int, colorCount+1)
	q.indices = make([]int, q.imageSize)
	for m := 0; m < q.imageSize; m++ {
		p := q.pixels[m]
		r, g, b := int(p>>16&0xFF), int(p>>8&0xFF), int(p&0xFF)
		num4 := q.quantizedPixels[m]
		num5 := 100000000
		for n := 0; n < colorCount; n++ {
			num9 := r - array2[n]
			num10 := g - array3[n]
			num11 := b - array4[n]
			num12 := num9*num9 + num10*num10 + num11*num11
			if num12 < num5 {
				num5 = num12
				num4 = n
			}
		}
		q.reds[num4] += r
		q.greens[num4] += g
		q.blues[num4] += b
		q.sums[num4]++
		q.indices[m] = num4
	}
	list := make([]color.NRGBA, 0, colorCount)
	for n := 0; n < colorCount; n++ {
		if q.sums[n] > 0 {
			q.reds[n] /= q.sums[n]
			q.greens[n] /= q.sums[n]
			q.blues[n] /= q.sums[n]
		}
		list = append(list, color.NRGBA{R: uint8(q.reds[n]), G: uint8(q.greens[n]), B: uint8(q.blues[n]), A: 255})
	}
	q.pixelIndex = 0
	return list
}

// onGetPaletteIndex ports WuColorQuantizer.OnGetPaletteIndex (WuColorQuantizer.cs:412-415).
func (q *wuColorQuantizer) onGetPaletteIndex(x, y int) int {
	return q.indices[x+y*q.imageWidth]
}

// onFinish ports WuColorQuantizer.OnFinish (WuColorQuantizer.cs:417-428).
func (q *wuColorQuantizer) onFinish() {
	q.cubes = nil
	q.weights = nil
	q.momentsRed = nil
	q.momentsGreen = nil
	q.momentsBlue = nil
	q.moments = nil
	q.quantizedPixels = nil
	q.pixels = nil
}
