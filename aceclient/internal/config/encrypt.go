package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Constants shared by both EncryptDecrypt variants
// (ACESettings/ICUSettings/EncryptDecrypt.cs, ACEServiceInstaller/ICU_Diag/EncryptDecrypt.cs).
const (
	saltValue          = "s@1tVaLue"        // EncryptDecrypt.saltValue
	passwordIterations = 2                  // EncryptDecrypt.passwordIterations ("SHA1")
	initVector         = "@1B2c3D4e5F6g7H8" // EncryptDecrypt.initVector (16 bytes => 128-bit block)
	keySize            = 256                // EncryptDecrypt.keySize (bits)
)

// ConfigPassPhrase is the pass phrase ICUConfig.ReadInstallerSettings /
// WriteInstallerSettings hand to EncryptDecrypt for InstallerConfig*.dat
// (literal "Pas5pR@sE" in ACESettings/ICUSettings/ICUConfig.cs). It is also the
// fixed ICU_Diag.EncryptDecrypt.passPhrase.
const ConfigPassPhrase = "Pas5pR@sE"

// ErrInvalidPadding mirrors the CryptographicException .NET raises when the
// PKCS7 padding of a decrypted block is wrong ("Padding is invalid and cannot
// be removed."), which in practice means a wrong pass phrase.
var ErrInvalidPadding = errors.New("config: Padding is invalid and cannot be removed")

// ErrInvalidLength mirrors "Length of the data to decrypt is invalid."
var ErrInvalidLength = errors.New("config: Length of the data to decrypt is invalid")

// PasswordDeriveBytes ports the first GetBytes(cb) call on a freshly
// constructed System.Security.Cryptography.PasswordDeriveBytes(password, salt,
// "SHA1", iterations) (.NET Framework 4.8). This is Microsoft's PBKDF1 variant:
//
//	base   = SHA1(UTF8(password) || salt), re-hashed while i < iterations-1
//	          (ComputeBaseValue: with iterations == 2 that is a single hash)
//	block0 = SHA1(base)
//	blockN = SHA1(ASCII(decimal N) || base)   (HashPrefix, N = 1..999)
//	output = block0 || block1 || ... truncated to cb
//
// The non-standard extension beyond 20 bytes is the HashPrefix counter. Both
// EncryptDecrypt variants construct a new instance per call and call GetBytes
// exactly once, so the stateful "_extra" buffering of repeated GetBytes calls
// (and its known offset bug) is never reached and is not ported.
func PasswordDeriveBytes(password string, salt []byte, iterations, cb int) ([]byte, error) {
	if iterations <= 0 {
		return nil, fmt.Errorf("config: PasswordDeriveBytes: iteration count must be positive")
	}
	if cb < 0 {
		return nil, fmt.Errorf("config: PasswordDeriveBytes: negative byte count")
	}
	// HashPrefix allows the counters 0..999, so the C# throws TooManyBytes for
	// anything beyond 1000 blocks. Fail before allocating for such a cb.
	if cb > (maxHashPrefix+1)*sha1.Size {
		return nil, errTooManyBytes
	}
	// ComputeBaseValue
	h := sha1.New()
	h.Write([]byte(password)) // new UTF8Encoding(false).GetBytes(strPassword)
	h.Write(salt)
	base := h.Sum(nil)
	for i := 1; i < iterations-1; i++ {
		s := sha1.Sum(base)
		base = s[:]
	}
	// ComputeBytes: the first block is always computed, then while (cb > ib).
	out := make([]byte, 0, ((cb+sha1.Size-1)/sha1.Size+1)*sha1.Size)
	for prefix := 0; ; prefix++ {
		p, err := hashPrefix(prefix)
		if err != nil {
			return nil, err
		}
		h.Reset()
		h.Write(p)
		h.Write(base)
		out = h.Sum(out)
		if len(out) >= cb {
			break
		}
	}
	return out[:cb], nil
}

// maxHashPrefix is the largest counter PasswordDeriveBytes.HashPrefix accepts.
const maxHashPrefix = 999

// errTooManyBytes mirrors CryptographicException
// "Cryptography_PasswordDerivedBytes_TooManyBytes".
var errTooManyBytes = errors.New("config: PasswordDeriveBytes: requested number of bytes exceeds the maximum")

// hashPrefix ports PasswordDeriveBytes.HashPrefix: nothing for 0, otherwise the
// ASCII decimal digits of the counter; > 999 throws
// ("Cryptography_PasswordDerivedBytes_TooManyBytes").
func hashPrefix(prefix int) ([]byte, error) {
	if prefix > maxHashPrefix {
		return nil, errTooManyBytes
	}
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
	if prefix > 0 {
		rgb[cb] += byte(prefix % 10)
		cb++
		return rgb[:cb], nil
	}
	return nil, nil
}

// deriveKey is PasswordDeriveBytes(passPhrase, saltValue, "SHA1", 2).GetBytes(keySize / 8).
func deriveKey(passPhrase string) []byte {
	k, err := PasswordDeriveBytes(passPhrase, []byte(saltValue), passwordIterations, keySize/8)
	if err != nil { // unreachable for the fixed parameters above
		panic(err)
	}
	return k
}

// encryptBytes is the RijndaelManaged{Mode = CBC} encryptor of EncryptDecrypt.Encrypt.
// RijndaelManaged defaults: BlockSize 128 (the 16-byte IV requires it),
// Padding PKCS7. With a 256-bit key that is exactly AES-256-CBC.
func encryptBytes(plain []byte, passPhrase string) []byte {
	block, err := aes.NewCipher(deriveKey(passPhrase))
	if err != nil { // unreachable: key is always 32 bytes
		panic(err)
	}
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	buf := make([]byte, len(plain)+pad)
	copy(buf, plain)
	for i := len(plain); i < len(buf); i++ {
		buf[i] = byte(pad)
	}
	cipher.NewCBCEncrypter(block, []byte(initVector)).CryptBlocks(buf, buf)
	return buf
}

// decryptBytes is the CryptoStream(CreateDecryptor) read of EncryptDecrypt.Decrypt.
// .NET Framework's CryptoStream.Read loops until the buffer is full or the
// stream ends, so the single Read in the C# returns the whole plaintext.
func decryptBytes(data []byte, passPhrase string) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, ErrInvalidLength
	}
	block, err := aes.NewCipher(deriveKey(passPhrase))
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, []byte(initVector)).CryptBlocks(out, data)
	// PKCS7 depadding as RijndaelManagedTransform does it: every pad byte checked.
	p := int(out[len(out)-1])
	if p < 1 || p > aes.BlockSize {
		return nil, ErrInvalidPadding
	}
	for _, b := range out[len(out)-p:] {
		if int(b) != p {
			return nil, ErrInvalidPadding
		}
	}
	return out[:len(out)-p], nil
}

// fromBase64 ports Convert.FromBase64String, which ignores white space
// (space, tab, CR, LF) anywhere in the input.
func fromBase64(s string) ([]byte, error) {
	if strings.ContainsAny(s, " \t\r\n") {
		s = strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\r', '\n':
				return -1
			}
			return r
		}, s)
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("config: invalid Base-64 string: %w", err)
	}
	return b, nil
}

// Encrypt ports ICUSettings.EncryptDecrypt.Encrypt(plainText, passPhrase)
// (ACESettings/ICUSettings/EncryptDecrypt.cs): UTF-8 plaintext, AES-256-CBC
// under PasswordDeriveBytes(passPhrase), fixed IV, Base64 result.
func Encrypt(plainText, passPhrase string) string {
	return base64.StdEncoding.EncodeToString(encryptBytes([]byte(plainText), passPhrase))
}

// Decrypt ports ICUSettings.EncryptDecrypt.Decrypt(cipherText, passPhrase)
// (ACESettings/ICUSettings/EncryptDecrypt.cs). An empty cipherText returns ""
// (string.IsNullOrEmpty guard). The bytes are returned as-is; .NET's
// Encoding.UTF8.GetString would additionally replace invalid UTF-8 with U+FFFD.
func Decrypt(cipherText, passPhrase string) (string, error) {
	if cipherText == "" {
		return "", nil
	}
	b, err := decryptBase64(cipherText, passPhrase)
	return string(b), err
}

func decryptBase64(cipherText, passPhrase string) ([]byte, error) {
	data, err := fromBase64(cipherText)
	if err != nil {
		return nil, err
	}
	return decryptBytes(data, passPhrase)
}

// DiagEncrypt ports ICU_Diag.EncryptDecrypt.Encrypt(plainText)
// (ACEServiceInstaller/ICU_Diag/EncryptDecrypt.cs): identical algorithm with the
// pass phrase fixed to ConfigPassPhrase.
func DiagEncrypt(plainText string) string {
	return Encrypt(plainText, ConfigPassPhrase)
}

// DiagDecrypt ports ICU_Diag.EncryptDecrypt.Decrypt(cipherText)
// (ACEServiceInstaller/ICU_Diag/EncryptDecrypt.cs). Unlike the ICUSettings
// variant it has no empty-input guard: "" decodes to zero bytes, which is
// rejected here as an invalid length.
func DiagDecrypt(cipherText string) (string, error) {
	b, err := decryptBase64(cipherText, ConfigPassPhrase)
	return string(b), err
}
