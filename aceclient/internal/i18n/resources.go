package i18n

import _ "embed"

// logoAlfenSmall is the PNG stream carried by the only entry of
// ICUServiceInstaller.Properties.Resources.resx ("logo_alfen_small", a
// BinaryFormatter-serialised System.Drawing.Bitmap, 126x32 RGBA). It was
// extracted byte-for-byte from that blob; the test re-extracts it from the
// .resx and compares.
//
//go:embed logo_alfen_small.png
var logoAlfenSmall []byte

// LogoAlfenSmall ports Properties.Resources.logo_alfen_small
// (ICUServiceInstaller.Properties/Resources.cs) and returns a copy of the PNG
// bytes. The resx holds no string resources — this bitmap is the entire
// table — and nothing in the decompiled installer references it.
func LogoAlfenSmall() []byte {
	return append([]byte(nil), logoAlfenSmall...)
}
