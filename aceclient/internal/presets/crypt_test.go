package presets

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestHashPrefix(t *testing.T) {
	for in, want := range map[int]string{1: "1", 9: "9", 10: "10", 42: "42", 100: "100", 105: "105", 110: "110", 999: "999"} {
		if got := string(hashPrefix(in)); got != want {
			t.Errorf("hashPrefix(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestPasswordDeriveBytesExtension(t *testing.T) {
	// The first 20 bytes are PBKDF1; bytes past the hash size come from the
	// "1"-prefixed block. Deriving more must extend, not change, the prefix.
	k32 := passwordDeriveBytes([]byte("pw"), []byte("salt"), 2, 32)
	k60 := passwordDeriveBytes([]byte("pw"), []byte("salt"), 2, 60)
	if len(k32) != 32 || len(k60) != 60 || string(k60[:32]) != string(k32) {
		t.Error("derived bytes are not a stable stream")
	}
	if string(k60[20:40]) == string(k60[:20]) {
		t.Error("second block must differ from the first")
	}
}

func TestEncryptMatchesDotNet(t *testing.T) {
	for _, v := range dotnetEncrypt {
		plain, _ := base64.StdEncoding.DecodeString(v.plainB64)
		if got := encryptString(string(plain), settingsPassPhrase); got != v.cipher {
			t.Errorf("Encrypt(%q) = %q, want %q", plain, got, v.cipher)
		}
		back, err := decryptString(v.cipher, settingsPassPhrase)
		if err != nil || back != string(plain) {
			t.Errorf("Decrypt(%q) = %q, %v", v.cipher, back, err)
		}
	}
	doc, err := decryptString(dotnetSettingsDocEnc, settingsPassPhrase)
	if err != nil || doc != mustB64(t, dotnetSettingsDocB64) {
		t.Errorf("decrypting the .NET settings file: %v", err)
	}
}

func TestDecryptErrors(t *testing.T) {
	good := encryptString("<Settings />", settingsPassPhrase)
	raw, _ := base64.StdEncoding.DecodeString(good)
	raw[len(raw)-1] ^= 0x55 // corrupt the last block -> bad padding
	tests := []struct{ name, in, want string }{
		{"not base64", "@@@@", "Base-64"},
		{"partial block", base64.StdEncoding.EncodeToString([]byte("short")), "Length of the data"},
		{"bad padding", base64.StdEncoding.EncodeToString(raw), "Padding is invalid"},
	}
	for _, tt := range tests {
		if _, err := decryptString(tt.in, settingsPassPhrase); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v", tt.name, err)
		}
	}
	if s, err := decryptString("", settingsPassPhrase); s != "" || err != nil {
		t.Error("empty input decrypts to empty")
	}
	if s, err := decryptString(" "+good[:8]+"\r\n\t"+good[8:]+"\n", settingsPassPhrase); err != nil || s != "<Settings />" {
		t.Errorf("white space must be ignored: %q %v", s, err)
	}
}
