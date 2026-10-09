package fwucreator

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"alfen/aceclient/internal/tvf"
)

// ICUTVFCreator constants.
const (
	InnerPackageName       = "inner_package.tar"     // INNER_PACKAGE_NAME
	UpdatePackageName      = "update_package.tar.gz" // UPDATE_PACKAGE_NAME
	ManifestFileName       = "manifest.json"         // MANIFEST_FILE_NAME
	VideoResourceFileName  = "videoresources.json"   // VIDEO_RESOURCE_FILE_NAME
	MaxTvfFileSizeInKB     = 500                     // MAX_FILE_SIZE_IN_KB
	MaxTvfFileSize         = MaxTvfFileSizeInKB * 1024
	DefaultManifestVersion = 1 // CreateTvfData(…, int manifestVersion = 1)
)

// newtonsoftString writes s as a JSON string the way Newtonsoft.Json does by
// default (StringEscapeHandling.Default): escapes \b \t \n \f \r \\ " and the
// other control characters as lower-case \u00xx, plus U+0085, U+2028, U+2029;
// everything else, including non-ASCII and <>&', is written verbatim.
func newtonsoftString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\f':
			b.WriteString(`\f`)
		case '\b':
			b.WriteString(`\b`)
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\u0085', '\u2028', '\u2029':
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// AhpMetadata ports ICUFWUCreator.Model.AhpMetadata.
type AhpMetadata struct {
	Description string // [JsonProperty("description")], default "Company logo"
	CreatedBy   string // [JsonProperty("created-by")]
	CreatedAt   string // [JsonProperty("created-at")] = DateTime.UtcNow.ToString("s") + "Z"
}

// AhpManifestOptions ports ICUFWUCreator.Model.AhpManifestOptions.
type AhpManifestOptions struct {
	Reboot        bool // [JsonProperty("reboot-scb-after-stage-install")]
	WaitForFinish bool // [JsonProperty("wait-for-charging-sessions-to-finish")]
}

// AhpManifest ports ICUFWUCreator.Model.AhpManifest.
type AhpManifest struct {
	ManifestVersion int // [JsonProperty("manifest-version")]
	Metadata        AhpMetadata
	Options         AhpManifestOptions
}

// NewAhpManifest ports the AhpManifest(creator, logoName, version) constructor;
// now supplies DateTime.UtcNow for AhpMetadata.CreatedAt.
func NewAhpManifest(creator, logoName string, version int, now time.Time) AhpManifest {
	return AhpManifest{
		ManifestVersion: version,
		Metadata: AhpMetadata{
			Description: logoName,
			CreatedBy:   creator,
			CreatedAt:   now.UTC().Format("2006-01-02T15:04:05") + "Z",
		},
		Options: AhpManifestOptions{},
	}
}

func jsonBool(v bool) string { return strconv.FormatBool(v) }

// JSON is JsonConvert.SerializeObject(manifest): compact, properties in
// declaration order.
func (m AhpManifest) JSON() string {
	return `{"manifest-version":` + strconv.Itoa(m.ManifestVersion) +
		`,"meta":{"description":` + newtonsoftString(m.Metadata.Description) +
		`,"created-by":` + newtonsoftString(m.Metadata.CreatedBy) +
		`,"created-at":` + newtonsoftString(m.Metadata.CreatedAt) +
		`},"options":{"reboot-scb-after-stage-install":` + jsonBool(m.Options.Reboot) +
		`,"wait-for-charging-sessions-to-finish":` + jsonBool(m.Options.WaitForFinish) + `}}`
}

// AhpVideoResource ports ICUFWUCreator.Model.AhpVideoResource.
type AhpVideoResource struct {
	LogoFileName string // [JsonProperty("logo")]
	LogoMargins  [4]int // [JsonProperty("logoMargins")] = { margin, margin, margin, margin }
}

// NewAhpVideoResource ports the AhpVideoResource(logoFileName, margin) constructor.
func NewAhpVideoResource(logoFileName string, margin int) AhpVideoResource {
	return AhpVideoResource{LogoFileName: logoFileName, LogoMargins: [4]int{margin, margin, margin, margin}}
}

// JSON is JsonConvert.SerializeObject(videoResource).
func (v AhpVideoResource) JSON() string {
	m := make([]string, len(v.LogoMargins))
	for i, x := range v.LogoMargins {
		m[i] = strconv.Itoa(x)
	}
	return `{"logo":` + newtonsoftString(v.LogoFileName) + `,"logoMargins":[` + strings.Join(m, ",") + `]}`
}

// writeTextFileContent ports ICUTVFCreator.WriteTextFile (ICUTVFCreator.cs:104-110): StreamWriter.WriteLine
// (UTF-8 without BOM, Windows "\r\n").
func writeTextFileContent(text string) []byte { return []byte(text + "\r\n") }

// ---------------------------------------------------------------------------
// Tar output in the exact header format of ICSharpCode.SharpZipLib 1.4.2
// (TarArchive.CreateOutputTarArchive(stream) -> TarOutputStream, block factor
// 20, nameEncoding null), so that the archive length — which TvfHeader stores
// as dataLength — matches the installer's.
// ---------------------------------------------------------------------------

const (
	tarBlockSize  = 512
	tarRecordSize = 20 * tarBlockSize // TarBuffer.DefaultRecordSize
)

// sharpTarEntry is what TarEntry.CreateEntryFromFile + the ICUTVFCreator
// overrides produce: Name = file name, Mode = 0644, TypeFlag '0', uid/gid 0,
// UserName "user" (TarHeader default for a null user), GroupName "None".
type sharpTarEntry struct {
	Name    string
	Data    []byte
	ModTime time.Time // File.GetLastWriteTime(...).ToUniversalTime()
}

// sharpOctal ports TarHeader.GetOctalBytes: NUL in the last byte, octal digits
// right-aligned and left-padded with '0'.
func sharpOctal(buf []byte, value int64, length int) {
	num := length - 1
	buf[num] = 0
	num--
	if value > 0 {
		for v := value; num >= 0 && v > 0; v >>= 3 {
			buf[num] = byte('0' + v&7)
			num--
		}
	}
	for ; num >= 0; num-- {
		buf[num] = '0'
	}
}

// sharpNameBytes ports TarHeader.GetNameBytes/GetAsciiBytes with a null
// encoding: (byte)char per UTF-16 unit, NUL padded/truncated to length.
func sharpNameBytes(buf []byte, s []uint16, length int) {
	for i := 0; i < length; i++ {
		if i < len(s) {
			buf[i] = byte(s[i])
		} else {
			buf[i] = 0
		}
	}
}

// sharpHeader ports TarHeader.WriteHeader for a regular-file or LongLink header.
func sharpHeader(name []uint16, mode int64, size int64, mtime int64, typeFlag byte) []byte {
	h := make([]byte, tarBlockSize)
	sharpNameBytes(h[0:100], name, 100)
	sharpOctal(h[100:108], mode, 8)
	sharpOctal(h[108:116], 0, 8) // UserId
	sharpOctal(h[116:124], 0, 8) // GroupId
	sharpOctal(h[124:136], size, 12)
	sharpOctal(h[136:148], mtime, 12)
	for i := 148; i < 156; i++ {
		h[i] = ' '
	}
	h[156] = typeFlag
	sharpNameBytes(h[157:257], nil, 100)                         // LinkName ""
	sharpNameBytes(h[257:263], utf16.Encode([]rune("ustar")), 6) // Magic
	sharpNameBytes(h[263:265], utf16.Encode([]rune(" ")), 2)     // Version
	sharpNameBytes(h[265:297], utf16.Encode([]rune("user")), 32) // UserName
	sharpNameBytes(h[297:329], utf16.Encode([]rune("None")), 32) // GroupName
	var sum int64
	for _, b := range h {
		sum += int64(b)
	}
	sharpOctal(h[148:155], sum, 7) // GetCheckSumOctalBytes: 6 digits, NUL, keeps ' '
	return h
}

// sharpCTime ports TarHeader.GetCTime after the ModTime setter truncated to
// whole seconds; dates before 1970 are rejected by that setter.
func sharpCTime(t time.Time) int64 { return t.UTC().Unix() }

// writeSharpTar builds a complete archive: per entry (an optional GNU
// "././@LongLink" header for names over 100 chars) header + data padded to
// 512, then TarOutputStream.Finish's two zero blocks, then TarBuffer.Close
// padding the last 10240-byte record.
func writeSharpTar(entries []sharpTarEntry) []byte {
	var out bytes.Buffer
	for _, e := range entries {
		name := utf16.Encode([]rune(e.Name))
		if len(name) > 100 {
			out.Write(sharpHeader(utf16.Encode([]rune("././@LongLink")), 420, int64(len(name)+1), 0, 'L'))
			for idx := 0; idx < len(name)+1; idx += tarBlockSize {
				blk := make([]byte, tarBlockSize)
				end := min(len(name), idx+tarBlockSize)
				if idx < end {
					sharpNameBytes(blk, name[idx:end], tarBlockSize)
				}
				out.Write(blk)
			}
		}
		out.Write(sharpHeader(name, 420, int64(len(e.Data)), sharpCTime(e.ModTime), '0'))
		out.Write(e.Data)
		if r := len(e.Data) % tarBlockSize; r != 0 {
			out.Write(make([]byte, tarBlockSize-r))
		}
	}
	out.Write(make([]byte, 2*tarBlockSize))
	if r := out.Len() % tarRecordSize; r != 0 {
		out.Write(make([]byte, tarRecordSize-r))
	}
	return out.Bytes()
}

// validateTvfImageFile ports ICUTVFCreator.ValidateImageFile (ICUTVFCreator.cs:70-84): the file must be
// a decodable image whose RawFormat is PNG.
func validateTvfImageFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, format, err := image.DecodeConfig(f)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if format != "png" {
		return fmt.Errorf("Only the png image format is supported on this CS platform")
	}
	return nil
}

// fileNameWithoutExtension ports Path.GetFileNameWithoutExtension.
func fileNameWithoutExtension(p string) string {
	base := filepath.Base(p)
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		return base[:i]
	}
	return base
}

// CreateInnerPackageTar ports ICUTVFCreator.CreateInnerPackageTar
// (ICUTVFCreator.cs:86-102, with AddFileToTar 119-125; in memory
// instead of via a temp folder): a tar with the image file (its original
// bytes and last-write time, as File.Copy preserves it), videoresources.json
// and manifest.json, in that order.
func CreateInnerPackageTar(pathToImageFile, creator string, margin, manifestVersion int, now time.Time) ([]byte, error) {
	imgBytes, err := os.ReadFile(pathToImageFile)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(pathToImageFile)
	if err != nil {
		return nil, err
	}
	fileNameWithoutExt := fileNameWithoutExtension(pathToImageFile)
	fileName := filepath.Base(pathToImageFile)
	manifest := NewAhpManifest(creator, fileNameWithoutExt, manifestVersion, now)
	video := NewAhpVideoResource(fileName, margin)
	return writeSharpTar([]sharpTarEntry{
		{Name: fileName, Data: imgBytes, ModTime: st.ModTime()},
		{Name: VideoResourceFileName, Data: writeTextFileContent(video.JSON()), ModTime: now},
		{Name: ManifestFileName, Data: writeTextFileContent(manifest.JSON()), ModTime: now},
	}), nil
}

// CreateTvfData ports ICUTVFCreator.CreateTvfData
// (ACEFWUCreator/ICUFWUCreator/ICUTVFCreator.cs:29-47): validate the PNG, build the
// inner package tar, then write TvfHeader(manifestVersion, file name without
// extension, len(inner tar)) followed by gzip(tar{inner_package.tar}), and
// reject results over 500 kB. The installer passes creator
// AppProperties.AppName and manifestVersion 1.
func CreateTvfData(inputImageFile, creator string, margin, manifestVersion int) ([]byte, error) {
	return createTvfData(inputImageFile, creator, margin, manifestVersion, time.Now())
}

func createTvfData(inputImageFile, creator string, margin, manifestVersion int, now time.Time) ([]byte, error) {
	if err := validateTvfImageFile(inputImageFile); err != nil {
		return nil, err
	}
	inner, err := CreateInnerPackageTar(inputImageFile, creator, margin, manifestVersion, now)
	if err != nil {
		return nil, err
	}
	dataLength := len(inner)
	header := tvf.BuildHeader(manifestVersion, fileNameWithoutExtension(inputImageFile), dataLength)

	var out bytes.Buffer
	out.Write(header)
	// GZipOutputStream: Deflater level -1 (6), header 1F 8B 08 00 <mtime> 00 FF.
	gz, err := gzip.NewWriterLevel(&out, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	gz.ModTime = now
	gz.OS = 255
	outer := writeSharpTar([]sharpTarEntry{{Name: InnerPackageName, Data: inner, ModTime: now}})
	if _, err := gz.Write(outer); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return validateTvfOutput(out.Bytes())
}

// validateTvfOutput ports ICUTVFCreator.ValidateOutput (ICUTVFCreator.cs:61-68) (message typo "off" kept).
func validateTvfOutput(output []byte) ([]byte, error) {
	if len(output) > MaxTvfFileSize {
		return nil, fmt.Errorf("Compressed image size exceeds the limit off %d kB", MaxTvfFileSizeInKB)
	}
	return output, nil
}
