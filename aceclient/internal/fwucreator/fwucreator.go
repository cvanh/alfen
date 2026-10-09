// Package fwucreator ports ACEFWUCreator.dll (display-resource authoring) and
// the non-UI logic of the installer's logo dialog.
//
// Ported from the decompiled C# (firmware/decompiled/…):
//
//   - ICUFWUCreator.CreateFWUData, AddObjectToStream, AddLanguage,
//     AddLanguageToStream, AddAllLanguageFilesFromFolder, AppendCFileBlock and
//     GetImage (ACEFWUCreator/ICUFWUCreator/ICUFWUCreator.cs). WriteFWUFile,
//     GetDataInBin, AESEncrypt, CreateFWIHeader and DeflateData are NOT
//     re-ported: they are reused from internal/fwi.
//   - ICUObjects (generated into objects_gen.go), ICUObjectTypes,
//     ICUImageFormats, ICUDisplayStrings (enums_gen.go).
//   - ICUTVFCreator.CreateTvfData and Model/{TvfHeader, AhpManifest,
//     AhpManifestOptions, AhpMetadata, AhpVideoResource} (tvf.go); TvfHeader
//     itself is internal/tvf.BuildHeader.
//   - DlgUploadResources / UploadResourceWorkerData (dialog.go) and
//     ICULanDevice.UploadResource -> StartUpload(isFirmwareFile: false)
//     (upload.go).
//   - DisplayObjectNew / DisplayObjectOld and the PanelMonitoring tables that
//     use them (displaystates.go).
//
// The object stream that CreateFWUData builds is a sequence of 32-byte-headed,
// 4-byte-aligned objects terminated by a u32 0; see AddObjectToStream.
// ParseObjects is a reader for that stream (not in the C#; for verification).
//
// No Fyne imports. Debug logging (Serilog Logger.Debug in the C#) goes to
// Debugf, a no-op unless the caller replaces it.
package fwucreator

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"alfen/aceclient/internal/alfencrc"
	"alfen/aceclient/internal/fwi"
)

// Debugf receives the messages the C# writes with Logger.Debug. No-op by
// default.
var Debugf = func(format string, args ...any) {}

// objectHeaderSize is the fixed size of every object header (the 32 in
// `32 + num3 * 3 + array.Length`).
const objectHeaderSize = 32

// languageTextSize is the fixed text field of one AddLanguage record
// (byte[] array6 = new byte[80]).
const languageTextSize = 80

// writer is a little-endian BinaryWriter over a byte buffer.
type writer struct{ bytes.Buffer }

func (w *writer) u8(v byte) { w.WriteByte(v) }
func (w *writer) u16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.Write(b[:])
}
func (w *writer) u32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.Write(b[:])
}

// GetImage ports ICUFWUCreator.GetImage (ICUFWUCreator.cs:18-35): Width bytes from each scan line of
// the locked bitmap. For the 8bpp image the installer passes (ConvertImage
// with 128 colours) that is exactly one palette index per pixel.
func GetImage(img *image.Paletted) []byte {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	out := make([]byte, 0, w*h)
	for i := 0; i < h; i++ {
		row := img.Pix[i*img.Stride:]
		out = append(out, row[:w]...)
	}
	return out
}

// paletteRGB returns Color.R/G/B of a palette entry (non-premultiplied).
func paletteRGB(c color.Color) (r, g, b uint8) {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return n.R, n.G, n.B
}

// AddObjectToStream ports ICUFWUCreator.AddObjectToStream
// (ACEFWUCreator/ICUFWUCreator/ICUFWUCreator.cs:122-182). Layout (LE):
//
//	+0  u8   1 (object version)
//	+1  u8   object type
//	+2  u16  width            +4 u16 height     +6 u16 stride
//	+8  u32  total object size (header + maxColors*3 + compressed, 4-aligned)
//	+12 u32  compressed length
//	+16 u8   image format
//	+17 u8   palette entries - 1 (0 when no palette)
//	+18 u16  palette offset (32) or 0      +20 u16 data offset
//	+22 i16  vertical offset
//	+24 u32  CRC32 of the raw data         +28 u32 CRC32 of the compressed data
//	+32      palette RGB triplets, compressed data, zero padding
//
// img is the paletted image for the customer logo, nil for the built-in
// objects (as in the C#, where `image` is null for them).
func AddObjectToStream(w *bytes.Buffer, data []byte, objectType ObjectType, format ImageFormat,
	width, height, stride, maxColors, verticalOffset int, img *image.Paletted) error {
	array, err := fwi.DeflateData(data)
	if err != nil {
		return err
	}
	num := alfencrc.ComputeChecksum(data, 0)
	num2 := alfencrc.ComputeChecksum(array, 0)
	Debugf("CRC32-Poly2 original: %08X", num)
	Debugf("CRC32-Poly2 compressed: %08X", num2)
	position := w.Len()
	num3 := 0
	if img != nil {
		num3 = maxColors
	}
	num4 := uint32(objectHeaderSize + num3*3 + len(array))
	var num5 uint32
	if num4%4 != 0 {
		num5 = 4 - num4%4
		num4 += num5
	}
	bw := &writer{}
	bw.u8(1)
	bw.u8(byte(objectType))
	bw.u16(uint16(width))
	bw.u16(uint16(height))
	bw.u16(uint16(stride))
	bw.u32(num4)
	bw.u32(uint32(len(array)))
	bw.u8(byte(format))
	num6 := maxColors
	if img != nil {
		num6 = min(len(img.Palette), maxColors)
	}
	num7 := 0
	if img != nil {
		num7 = num6 - 1
	}
	bw.u8(byte(num7))
	if num7 > 0 {
		bw.u16(objectHeaderSize)
		bw.u16(uint16(objectHeaderSize + num6*3))
	} else {
		bw.u16(0)
		bw.u16(objectHeaderSize)
	}
	bw.u16(uint16(int16(verticalOffset)))
	bw.u32(num)
	bw.u32(num2)
	if img != nil {
		for i := 0; i < num6; i++ {
			r, g, b := paletteRGB(img.Palette[i])
			bw.u8(r)
			bw.u8(g)
			bw.u8(b)
		}
	}
	bw.Write(array)
	for j := uint32(0); j < num5; j++ {
		bw.u8(0)
	}
	w.Write(bw.Bytes())
	Debugf("Object %v written at: %d, length: %d Pos: %d", objectType, position, w.Len()-position, w.Len())
	return nil
}

// utf16Units returns the UTF-16 code units of s (a .NET string's chars).
func utf16Units(s string) []uint16 { return utf16.Encode([]rune(s)) }

// AddLanguageToStream ports ICUFWUCreator.AddLanguageToStream
// (ICUFWUCreator.cs:184-224): the same
// 32-byte header as AddObjectToStream with width/height/stride 0, format 0xFF,
// +17 = language name length, +18 = 32 (name offset), +20 = 32 + name length
// (data offset), +22 = 0; then the name as (byte)char, the compressed record
// table and zero padding.
func AddLanguageToStream(w *bytes.Buffer, data []byte, objectType ObjectType, language string) error {
	array, err := fwi.DeflateData(data)
	if err != nil {
		return err
	}
	num := alfencrc.ComputeChecksum(data, 0)
	num2 := alfencrc.ComputeChecksum(array, 0)
	Debugf("CRC32-Poly2 original: %08X", num)
	Debugf("CRC32-Poly2 compressed: %08X", num2)
	position := w.Len()
	lang := utf16Units(language)
	num3 := uint32(objectHeaderSize + len(array) + len(lang))
	var num4 uint32
	if num3%4 != 0 {
		num4 = 4 - num3%4
		num3 += num4
	}
	bw := &writer{}
	bw.u8(1)
	bw.u8(byte(objectType))
	bw.u16(0)
	bw.u16(0)
	bw.u16(0)
	bw.u32(num3)
	bw.u32(uint32(len(array)))
	bw.u8(0xFF)
	bw.u8(byte(len(lang)))
	bw.u16(objectHeaderSize)
	bw.u16(uint16(objectHeaderSize + len(lang)))
	bw.u16(0)
	bw.u32(num)
	bw.u32(num2)
	for _, c := range lang {
		bw.u8(byte(c))
	}
	bw.Write(array)
	for j := uint32(0); j < num4; j++ {
		bw.u8(0)
	}
	w.Write(bw.Bytes())
	Debugf("Object %v written at: %d, length: %d Pos: %d", objectType, position, w.Len()-position, w.Len())
	return nil
}

// readAllLines ports File.ReadAllLines: BOM-detected encoding (UTF-8 default,
// UTF-16/32 LE/BE by BOM), lines split on "\r\n", "\n" or "\r", no trailing
// empty line after a final terminator. Invalid UTF-8 becomes U+FFFD.
func readAllLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := decodeText(raw)
	var lines []string
	start := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '\n':
			lines = append(lines, text[start:i])
			start = i + 1
		case '\r':
			lines = append(lines, text[start:i])
			if i+1 < len(text) && text[i+1] == '\n' {
				i++
			}
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines, nil
}

func decodeText(raw []byte) string {
	switch {
	case len(raw) >= 4 && raw[0] == 0xFF && raw[1] == 0xFE && raw[2] == 0 && raw[3] == 0:
		return decodeUTF32(raw[4:], binary.LittleEndian)
	case len(raw) >= 4 && raw[0] == 0 && raw[1] == 0 && raw[2] == 0xFE && raw[3] == 0xFF:
		return decodeUTF32(raw[4:], binary.BigEndian)
	case len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF:
		raw = raw[3:]
	case len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE:
		return decodeUTF16(raw[2:], binary.LittleEndian)
	case len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF:
		return decodeUTF16(raw[2:], binary.BigEndian)
	}
	return strings.ToValidUTF8(string(raw), string(utf8.RuneError))
}

func decodeUTF16(b []byte, bo binary.ByteOrder) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = bo.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

func decodeUTF32(b []byte, bo binary.ByteOrder) string {
	var sb strings.Builder
	for i := 0; i+4 <= len(b); i += 4 {
		sb.WriteRune(rune(bo.Uint32(b[i:])))
	}
	return sb.String()
}

// encodeLatin1 ports Encoding.GetEncoding("ISO-8859-1",
// new EncoderReplacementFallback(""), …).GetBytes: chars above U+00FF are
// dropped.
func encodeLatin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s { // U+FFFD from bad input is above U+00FF: dropped too
		if r <= 0xFF {
			out = append(out, byte(r))
		}
	}
	return out
}

// parseUInt16 ports Convert.ToUInt16(string) (UInt16.Parse with
// NumberStyles.Integer): optional surrounding white space (U+0009–U+000D,
// U+0020), an optional leading sign, ASCII digits, value 0..65535.
func parseUInt16(s string) (uint16, error) {
	t := strings.Trim(s, "\t\n\v\f\r ")
	neg := false
	if t != "" && (t[0] == '+' || t[0] == '-') {
		neg = t[0] == '-'
		t = t[1:]
	}
	if t == "" {
		return 0, fmt.Errorf("Input string was not in a correct format.")
	}
	var v uint64
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("Input string was not in a correct format.")
		}
		v = v*10 + uint64(c-'0')
		if v > 0xFFFF {
			return 0, fmt.Errorf("Value was either too large or too small for a UInt16.")
		}
	}
	if neg && v != 0 {
		return 0, fmt.Errorf("Value was either too large or too small for a UInt16.")
	}
	return uint16(v), nil
}

// LanguageRecordSize is the size of one AddLanguage record:
// u8 string id, u8 80, u8 font, u8 0, u16 x, u16 y, [80]byte text.
const LanguageRecordSize = 8 + languageTextSize

// buildLanguageData ports the record-building half of
// ICUFWUCreator.AddLanguage (ICUFWUCreator.cs:226-297): it returns the language name from the
// "LANGUAGE: xx" header line and the uncompressed record table. An error is
// what the C# throws (I/O, Enum.Parse, Convert.ToUInt16); malformed lines with
// fewer than five fields are skipped with a debug message, as in the C#.
func buildLanguageData(fileName string) (language string, data []byte, err error) {
	array3, err := readAllLines(fileName)
	if err != nil {
		return "", nil, err
	}
	bw := &writer{}
	if len(array3) != 0 {
		array4 := strings.Split(array3[0], ":")
		if len(array4) > 1 && strings.ToLower(strings.TrimSpace(array4[0])) == "language" {
			language = strings.TrimSpace(array4[1])
		}
		for i := 1; i < len(array3); i++ {
			array5 := strings.Split(array3[i], ",")
			if len(array5) < 5 {
				Debugf("Error, invalid line formatting in file %s: line %d is missing one or more required fields", fileName, i)
				continue
			}
			id, err := ParseDisplayString(array5[0])
			if err != nil {
				return language, nil, err
			}
			text2 := array5[1]
			if len(array5) > 5 {
				text2 = strings.Join(array5[1:len(array5)-3], ",")
			}
			text2 = strings.Trim(text2, " \t\"")
			text2 = strings.TrimSpace(text2)
			text2 = strings.ReplaceAll(text2, `\n`, "\n")
			array2 := encodeLatin1(text2)
			if len(array2) > languageTextSize {
				Debugf("Error, text string: '%s' is too large, text truncated! (%d chars, allowed 80 chars)", text2, len(array2))
			}
			var array6 [languageTextSize]byte
			copy(array6[:], array2[:min(len(array2), languageTextSize-1)])
			bw.u8(byte(id))
			bw.u8(languageTextSize)
			font, err := parseUInt16(array5[len(array5)-3])
			if err != nil {
				return language, nil, err
			}
			bw.u8(byte(font))
			bw.u8(0)
			x, err := parseUInt16(array5[len(array5)-2])
			if err != nil {
				return language, nil, err
			}
			bw.u16(x)
			y, err := parseUInt16(array5[len(array5)-1])
			if err != nil {
				return language, nil, err
			}
			bw.u16(y)
			bw.Write(array6[:])
		}
	}
	data = bw.Bytes()
	Debugf("AddLanguageToStream for '%s' = %d bytes", language, len(data))
	return language, data, nil
}

// AddLanguage ports ICUFWUCreator.AddLanguage (ICUFWUCreator.cs:226-299): parse one language CSV and
// append it as an OBJECT_LANGUAGE object. Nothing is written when parsing
// fails.
func AddLanguage(w *bytes.Buffer, objectType ObjectType, fileName string) error {
	language, data, err := buildLanguageData(fileName)
	if err != nil {
		return err
	}
	return AddLanguageToStream(w, data, objectType, language)
}

// languageOrder is the preferred order AddAllLanguageFilesFromFolder uses.
var languageOrder = []string{"_NL", "_EN", "_DE"}

// listFilesLikeWindows stands in for Directory.GetFiles(path): regular files
// only, ordered like NTFS returns them (case-insensitive name order).
func listFilesLikeWindows(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.Type().IsRegular() {
			names = append(names, e.Name())
		}
	}
	sort.SliceStable(names, func(i, j int) bool {
		return strings.ToUpper(names[i]) < strings.ToUpper(names[j])
	})
	return names, nil
}

// AddAllLanguageFilesFromFolder ports
// ICUFWUCreator.AddAllLanguageFilesFromFolder (ICUFWUCreator.cs:324-353): create the folder if missing,
// add the first file whose name contains "_NL", then "_EN", then "_DE", then
// every other file in directory order. The C# catches every exception and
// only logs it, which also stops adding further languages; the swallowed
// error is returned here for diagnostics (CreateFWUData ignores it).
func AddAllLanguageFilesFromFolder(w *bytes.Buffer, path string) error {
	err := addAllLanguageFiles(w, path)
	if err != nil {
		Debugf("%v", err)
	}
	return err
}

func addAllLanguageFiles(w *bytes.Buffer, path string) error {
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	list, err := listFilesLikeWindows(path)
	if err != nil {
		return err
	}
	for _, lang := range languageOrder {
		for i, name := range list {
			if strings.Contains(name, lang) {
				if err := AddLanguage(w, ObjectLanguage, filepath.Join(path, name)); err != nil {
					return err
				}
				list = append(list[:i:i], list[i+1:]...)
				break
			}
		}
	}
	for _, name := range list {
		if err := AddLanguage(w, ObjectLanguage, filepath.Join(path, name)); err != nil {
			return err
		}
	}
	return nil
}

// AppendCFileBlock ports ICUFWUCreator.AppendCFileBlock (ICUFWUCreator.cs:355-372), including the
// Windows "\r\n" that StringBuilder.AppendLine emits.
func AppendCFileBlock(sb *strings.Builder, name string, data []byte) {
	fmt.Fprintf(sb, "const uint8_t %s[%d] = {\n", name, len(data))
	for i, b := range data {
		if i > 0 {
			sb.WriteString(", ")
			if i%16 == 0 {
				sb.WriteString("\r\n")
			}
		}
		fmt.Fprintf(sb, "0x%02X", b)
	}
	sb.WriteString("\n};\r\n")
	sb.WriteString("\r\n")
}

// builtinObject is one fixed AddObjectToStream call of CreateFWUData.
type builtinObject struct {
	Data           []byte
	Type           ObjectType
	Format         ImageFormat
	Width, Height  int
	Stride         int
	VerticalOffset int
}

// builtinObjects are the fixed objects CreateFWUData writes after the
// customer logo, in order; Robotica 30 only for a large screen or C file.
var builtinObjects = []builtinObject{
	{LogoAcceptedL1, ObjectLogoAccepted, ImageFormatL1, 56, 59, 7, 0},
	{LogoChargingL1, ObjectLogoCharging, ImageFormatL1, 80, 50, 10, 0},
	{LogoErrorL1, ObjectLogoSocketerror, ImageFormatL1, 56, 56, 7, 0},
	{LogoCommunicatingL1, ObjectLogoCommunicating, ImageFormatL1, 56, 78, 7, 0},
	{FontRobotica28, ObjectRoboticaRegular28, ImageFormatL4, 20, 28, 10, -2},
	{FontRobotica29, ObjectRoboticaRegular29, ImageFormatL4, 22, 30, 11, -2},
}

var builtinRobotica30 = builtinObject{FontRobotica30, ObjectRoboticaRegular30, ImageFormatL4, 30, 41, 15, -4}

// createObjectStream is the MemoryStream half of CreateFWUData: every object
// plus the u32 0 terminator, before WriteFWUFile.
func createObjectStream(image1 *image.Paletted, maxColors int, uiFolderPath string, createCFile, largeScreen bool) ([]byte, error) {
	if image1 == nil {
		return nil, fmt.Errorf("Object reference not set to an instance of an object.")
	}
	image2 := GetImage(image1)
	w := &bytes.Buffer{}
	width, height := image1.Rect.Dx(), image1.Rect.Dy()
	if err := AddObjectToStream(w, image2, ObjectLogoCustomer, ImagePaletted8, width, height, width, maxColors, 0, image1); err != nil {
		return nil, err
	}
	objs := builtinObjects
	if largeScreen || createCFile {
		objs = append(objs[:len(objs):len(objs)], builtinRobotica30)
	}
	for _, o := range objs {
		if err := AddObjectToStream(w, o.Data, o.Type, o.Format, o.Width, o.Height, o.Stride, 0, o.VerticalOffset, nil); err != nil {
			return nil, err
		}
	}
	_ = AddAllLanguageFilesFromFolder(w, uiFolderPath)
	var term [4]byte
	w.Write(term[:])
	Debugf("Total objects size: %d", w.Len())
	return w.Bytes(), nil
}

// CreateFWUData ports ICUFWUCreator.CreateFWUData (ICUFWUCreator.cs:374-409): the customer logo
// (PALETTED8, stride = width, palette = first maxColors entries), the four
// 1-bit status icons, the Robotica 28/29 fonts (plus 30 when largeScreen or
// createCFile), every language CSV in uiFolderPath, a u32 0 terminator — then
// fwi.WrapDisplayPayload (WriteFWUFile: GetDataInBin, AES, FWI header).
//
// With createCFile the raw object stream is also written to
// pathToCFile/default_objects.c via AppendCFileBlock("object_data", …).
// The installer passes maxColors 128 and uiFolderPath "UILanguages".
func CreateFWUData(image1 *image.Paletted, maxColors int, uiFolderPath string, createCFile bool, pathToCFile string, largeScreen bool) ([]byte, error) {
	array, err := createObjectStream(image1, maxColors, uiFolderPath, createCFile, largeScreen)
	if err != nil {
		return nil, err
	}
	if createCFile {
		var sb strings.Builder
		AppendCFileBlock(&sb, "object_data", array)
		if err := os.WriteFile(filepath.Join(pathToCFile, "default_objects.c"), []byte(sb.String()), 0o644); err != nil {
			return nil, err
		}
	}
	return fwi.WrapDisplayPayload(array)
}
