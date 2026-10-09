package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"strconv"
	"strings"
)

// PasswordLength is ICUUser.s_passwordLength.
const PasswordLength = 6

const (
	passwordChars = "aeiouyaeiouyaeiouybcdfghjklmnpqrstvwxz" // ICUUser.CreatePassword alphabet
	passwordDigit = "1234567890"
)

// NewUser ports the parameterless ICUUser() constructor (ICUConfig.AddUser):
// no group, a fresh CreatePassword(PasswordLength) password.
func NewUser() *User {
	return &User{Password: CreatePassword(PasswordLength)}
}

// CreatePassword ports ICUUser.CreatePassword: length characters drawn from a
// vowel-weighted alphabet with one random position replaced by a digit
// (System.Random, i.e. not cryptographically secure). The C# throws for
// length <= 0; this port returns "".
func CreatePassword(length int) string {
	if length <= 0 {
		return ""
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = passwordChars[mrand.IntN(len(passwordChars))]
	}
	b[mrand.IntN(len(b))] = passwordDigit[mrand.IntN(len(passwordDigit))]
	return string(b)
}

// HashUserName is the user-name hash used by hashed configs and DlgLogon:
// Convert.ToBase64String(SHA256(UTF8(username.ToLowerInvariant()))).
func HashUserName(username string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(username)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// hashPassword iterates SHA256 over UTF8(saltHex + password) n times and
// returns the Base64 digest (shared by CreateHashedUser / ValidateHashedPassword).
func hashPassword(saltHex, password string, n int) string {
	b := []byte(saltHex + password)
	for i := 0; i < n; i++ {
		s := sha256.Sum256(b)
		b = s[:]
	}
	return base64.StdEncoding.EncodeToString(b)
}

// CreateHashedUser ports ICUUser.CreateHashedUser
// (ACESettings/ICUSettings/ICUUser.cs) using crypto/rand as the
// RNGCryptoServiceProvider:
//
//	salt  = 4 random bytes, hex upper-case (BitConverter.ToString without '-')
//	n     = 1 random non-zero byte (iteration count 1..255)
//	User     = HashUserName(username)
//	Password = "{n:X2}:{salt}:{Base64(SHA256^n(UTF8(salt + password)))}"
//	Group    = group, Fullname = ""
//	Comment  = Encrypt(httpData, salt + password)   // "HTTPUser:HTTPPassword"
//	Company  = Encrypt(isahData, salt + password)   // ISAH credentials
func CreateHashedUser(username, password string, group *Group, isahData, httpData string) (*User, error) {
	return createHashedUser(rand.Reader, username, password, group, isahData, httpData)
}

func createHashedUser(rng io.Reader, username, password string, group *Group, isahData, httpData string) (*User, error) {
	salt := make([]byte, 4)
	if _, err := io.ReadFull(rng, salt); err != nil { // rng.GetBytes(array)
		return nil, fmt.Errorf("config: CreateHashedUser: %w", err)
	}
	var one [1]byte
	for one[0] == 0 { // rng.GetNonZeroBytes(array2)
		if _, err := io.ReadFull(rng, one[:]); err != nil {
			return nil, fmt.Errorf("config: CreateHashedUser: %w", err)
		}
	}
	n := int(one[0])
	saltHex := strings.ToUpper(hex.EncodeToString(salt))
	passPhrase := saltHex + password
	return &User{
		User:     HashUserName(username),
		Password: fmt.Sprintf("%02X:%s:%s", n, saltHex, hashPassword(saltHex, password, n)),
		Group:    group,
		Fullname: "",
		Comment:  strings.TrimSpace(Encrypt(httpData, passPhrase)),
		Company:  strings.TrimSpace(Encrypt(isahData, passPhrase)),
	}, nil
}

// ValidateHashedPassword ports ICUUser.ValidateHashedPassword: Password must
// split on ':' into exactly three parts; the iteration count is parsed with
// Convert.ToUInt16(part, 16) (an error here is the FormatException/
// OverflowException the C# would throw); the result is the comparison of the
// recomputed Base64 digest with the third part.
func (u *User) ValidateHashedPassword(otherPassword string) (bool, error) {
	parts := strings.Split(u.Password, ":")
	if len(parts) != 3 {
		return false, nil
	}
	n, err := parseUInt16Hex(parts[0])
	if err != nil {
		return false, err
	}
	return parts[2] == hashPassword(parts[1], otherPassword, int(n)), nil
}

// parseUInt16Hex mirrors Convert.ToUInt16(string, 16): optional "0x"/"0X"
// prefix, hexadecimal digits only, at most 0xFFFF.
func parseUInt16Hex(s string) (uint16, error) {
	t := s
	if len(t) >= 2 && t[0] == '0' && (t[1] == 'x' || t[1] == 'X') {
		t = t[2:]
	}
	v, err := strconv.ParseUint(t, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("config: invalid iteration count %q in hashed password: %w", s, err)
	}
	return uint16(v), nil
}
