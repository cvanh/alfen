package presets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"strings"
)

// Constants of ICUSettings.EncryptDecrypt (ACESettings/ICUSettings/EncryptDecrypt.cs).
// They are compiled into the original assembly and are required to read and
// write .exml files.
const (
	edSaltValue          = "s@1tVaLue"
	edPasswordIterations = 2
	edInitVector         = "@1B2c3D4e5F6g7H8"
	edKeySize            = 256 // bits -> AES-256 (RijndaelManaged, 128-bit block)
)

// settingsPassPhrase ports PropertyStorage.s_key, the pass phrase handed to
// EncryptDecrypt for .exml settings files.
const settingsPassPhrase = "Alfen"

// passwordDeriveBytes ports System.Security.Cryptography.PasswordDeriveBytes
// (SHA1) for a single GetBytes(cb) call on a fresh instance, as used by
// EncryptDecrypt: a PBKDF1 base value, extended past the hash size with the
// Microsoft-specific "counter prefix" scheme (ComputeBaseValue/ComputeBytes/
// HashPrefix in the .NET reference source).
func passwordDeriveBytes(password, salt []byte, iterations, cb int) []byte {
	// ComputeBaseValue: H(password || salt), then iterations-2 more rounds.
	h := sha1.New()
	h.Write(password)
	h.Write(salt)
	base := h.Sum(nil)
	for i := 1; i < iterations-1; i++ {
		s := sha1.Sum(base)
		base = s[:]
	}
	// ComputeBytes: block k (k = 0, 1, ...) is H(prefix(k) || base) where
	// prefix(0) is empty and prefix(k>0) is the decimal ASCII of k.
	var out []byte
	for prefix := 0; len(out) < cb; prefix++ {
		h.Reset()
		if prefix > 0 {
			h.Write(hashPrefix(prefix))
		}
		h.Write(base)
		out = h.Sum(out)
	}
	return out[:cb]
}

// hashPrefix ports PasswordDeriveBytes.HashPrefix for 0 < prefix <= 999.
func hashPrefix(prefix int) []byte {
	rgb := []byte{'0', '0', '0'}
	cb := 0
	if prefix >= 100 {
		rgb[0] += byte(prefix / 100)
		cb++
	}
	if prefix >= 10 {
		rgb[cb] += byte((prefix % 100) / 10)
		cb++
	}
	rgb[cb] += byte(prefix % 10)
	cb++
	return rgb[:cb]
}

// encryptDecryptKey returns the AES key/IV EncryptDecrypt derives for passPhrase.
func encryptDecryptKey(passPhrase string) (key, iv []byte) {
	key = passwordDeriveBytes([]byte(passPhrase), []byte(edSaltValue), edPasswordIterations, edKeySize/8)
	return key, []byte(edInitVector)
}

// encryptString ports EncryptDecrypt.Encrypt: UTF-8 plaintext, AES-256-CBC
// with PKCS7 padding, Base64 output.
func encryptString(plainText, passPhrase string) string {
	key, iv := encryptDecryptKey(passPhrase)
	block, _ := aes.NewCipher(key) // key is always 32 bytes
	data := []byte(toValidUTF8(plainText))
	pad := aes.BlockSize - len(data)%aes.BlockSize
	data = append(data, bytes.Repeat([]byte{byte(pad)}, pad)...)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(data, data)
	return base64.StdEncoding.EncodeToString(data)
}

// errBadPadding mirrors the CryptographicException message .NET raises for a
// wrong key or corrupt data.
var errBadPadding = errors.New("Padding is invalid and cannot be removed.")

// decryptString ports EncryptDecrypt.Decrypt. An empty cipherText yields "".
// Convert.FromBase64String ignores white space, which is stripped first.
func decryptString(cipherText, passPhrase string) (string, error) {
	if cipherText == "" {
		return "", nil
	}
	clean := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n':
			return -1
		}
		return r
	}, cipherText)
	data, err := base64.StdEncoding.DecodeString(clean)
	if err != nil {
		return "", errors.New("The input is not a valid Base-64 string as it contains a non-base 64 character, more than two padding characters, or an illegal character among the padding characters.")
	}
	if len(data) == 0 {
		return "", nil
	}
	if len(data)%aes.BlockSize != 0 {
		return "", errors.New("Length of the data to decrypt is invalid.")
	}
	key, iv := encryptDecryptKey(passPhrase)
	block, _ := aes.NewCipher(key)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(data, data)
	pad := int(data[len(data)-1])
	if pad < 1 || pad > aes.BlockSize {
		return "", errBadPadding
	}
	for _, b := range data[len(data)-pad:] {
		if int(b) != pad {
			return "", errBadPadding
		}
	}
	return decodeUTF8Lenient(data[:len(data)-pad]), nil
}
