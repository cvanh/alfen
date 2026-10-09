package config

import (
	"encoding/hex"
	"errors"
	"math"
	"testing"
)

// Vectors were generated independently of this package with Python hashlib
// (PasswordDeriveBytes) and `openssl enc -aes-256-cbc -K <key> -iv <IV>`.
func TestPasswordDeriveBytes(t *testing.T) {
	salt := []byte(saltValue)
	tests := []struct {
		name       string
		pw         string
		iterations int
		cb         int
		want       string
	}{
		{"config key", ConfigPassPhrase, 2, 32, "736a9bf3e4c3c499355a5bedc978defdef395a0191ba75731b165298d03e38c9"},
		{"Alfen key", "Alfen", 2, 32, "1ad46d616f986c48b8e039cf39017641749da3a2219d2dda405daf5f67869d9e"},
		{"5 iterations", "pw", 5, 40, "3079dacb58910b2d906a25842ef66bd28139f751d0e842cf36c139d8c5fca4a8347a62f875b49814"},
		{"prefix >= 10", "pw", 2, 220, "390072eb96587014a482cf6258aa0b8b64b4f8cfe1a413de411441f68a9108e202ba79fce773f4c6c485b52a2b4ebd84d4b0da8561af171e4c3f21c66a7d3185e47923fcfa5554ca9c01ee3eb62536c91fb152adc4dc7813df2553e82bc52e2be3b8c3ac5a8a9cf7339a03667190ffb819d74e52e52d9504cb59341c0fc60782e35fb529cb02aa592570a22fa1c3fd722d7733f4f10d4b4cc378bf6da02eb9b97705cc213d523493f0d671c4b7e7b5f7fa3eeb3865586c52f53feed439f2fab73699b0b9d1668a74d5d49be0e1f9616f9071dc5a23217ca4734dcb62"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PasswordDeriveBytes(tc.pw, salt, tc.iterations, tc.cb)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(got) != tc.want {
				t.Fatalf("got %x", got)
			}
		})
	}
	if b, err := PasswordDeriveBytes("pw", salt, 2, 1000*20); err != nil || len(b) != 1000*20 {
		t.Fatalf("1000 blocks (prefix 0..999) must work: %v", err)
	}
	for _, cb := range []int{1000*20 + 1, 1 << 30, math.MaxInt/2 + 1, math.MaxInt} { // MaxInt/2+1 is 1<<62 on 64-bit
		if _, err := PasswordDeriveBytes("pw", salt, 2, cb); !errors.Is(err, errTooManyBytes) {
			t.Fatalf("cb=%d: expected TooManyBytes, got %v", cb, err)
		}
	}
	if _, err := PasswordDeriveBytes("pw", salt, 2, -1); err == nil {
		t.Fatal("negative cb must fail")
	}
}

func TestHashPrefix(t *testing.T) {
	for n, want := range map[int]string{0: "", 1: "1", 9: "9", 10: "10", 99: "99", 100: "100", 105: "105", 999: "999"} {
		got, err := hashPrefix(n)
		if err != nil || string(got) != want {
			t.Errorf("hashPrefix(%d) = %q, %v; want %q", n, got, err, want)
		}
	}
}

func TestEncryptDecryptVectors(t *testing.T) {
	tests := []struct {
		pass, plain, cipher string
	}{
		{ConfigPassPhrase, "Hello, ACE!", "gCvhoPEsyD43+6u8ZwHTyw=="},
		{ConfigPassPhrase, "", "/tbsBXACUaOOuzQMJ5/mAw=="},
		{ConfigPassPhrase, "0123456789abcdef", "6dJxIgxnGU3hnSlbGpyAz2Yq87Rk5PTnq12mt0lPr4A="},
		{ConfigPassPhrase, "Ünïcødé ✓", "Mp3qdjxctcPDLdWm9Yjsmg=="},
		{"Alfen", "Hello, ACE!", "4Z7EnUHsPp1OiyCU4yJj/Q=="},
		{"Alfen", "", "SdjaJpNv4qGpwJEYyE/GGQ=="},
		{"Alfen", "0123456789abcdef", "VOoCn4U4JsshWSZ7EwiOb/9jq+6Ts1QCOaCz6MwXBWQ="},
		{"Alfen", "Ünïcødé ✓", "D4Y3cXai6Kd3dkFxG19TiA=="},
	}
	for _, tc := range tests {
		t.Run(tc.pass+"/"+tc.plain, func(t *testing.T) {
			if got := Encrypt(tc.plain, tc.pass); got != tc.cipher {
				t.Fatalf("Encrypt = %s, want %s", got, tc.cipher)
			}
			got, err := Decrypt(tc.cipher, tc.pass)
			if err != nil || got != tc.plain {
				t.Fatalf("Decrypt = %q, %v", got, err)
			}
			if tc.pass == ConfigPassPhrase {
				if DiagEncrypt(tc.plain) != tc.cipher {
					t.Fatal("DiagEncrypt differs from Encrypt(ConfigPassPhrase)")
				}
				if got, err := DiagDecrypt(tc.cipher); err != nil || got != tc.plain {
					t.Fatalf("DiagDecrypt = %q, %v", got, err)
				}
			}
		})
	}
}

func TestDecryptEdgeCases(t *testing.T) {
	// ICUSettings variant short-circuits empty input; ICU_Diag does not.
	if got, err := Decrypt("", "x"); got != "" || err != nil {
		t.Fatalf("Decrypt(\"\") = %q, %v", got, err)
	}
	if _, err := DiagDecrypt(""); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("DiagDecrypt(\"\") err = %v", err)
	}
	// Convert.FromBase64String ignores white space.
	if got, err := Decrypt(" gCvhoPEsy\r\nD43+6u8Z\twHTyw== ", ConfigPassPhrase); err != nil || got != "Hello, ACE!" {
		t.Fatalf("whitespace tolerant decode = %q, %v", got, err)
	}
	// A wrong pass phrase surfaces as a padding error (overwhelmingly likely).
	if _, err := Decrypt("gCvhoPEsyD43+6u8ZwHTyw==", "wrong"); !errors.Is(err, ErrInvalidPadding) {
		t.Fatalf("wrong pass phrase err = %v", err)
	}
	if _, err := Decrypt("AAAA", ConfigPassPhrase); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("short input err = %v", err)
	}
	if _, err := Decrypt("not base64!", ConfigPassPhrase); err == nil {
		t.Fatal("expected base64 error")
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	for _, plain := range []string{"a", "exactly16bytes!!", "{\n\"Type\":\"ICUConfigFile\"}", string(make([]byte, 1000))} {
		for _, pass := range []string{"p", "0A1B2C3Dsecret", ConfigPassPhrase} {
			got, err := Decrypt(Encrypt(plain, pass), pass)
			if err != nil || got != plain {
				t.Fatalf("round trip failed for len %d / %q: %v", len(plain), pass, err)
			}
		}
	}
}
