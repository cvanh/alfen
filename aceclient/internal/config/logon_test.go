package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
	"testing"
)

// Synthetic, test-only identities; nothing here comes from a real config.
const (
	synthISAH       = "synthetic-isah-data"
	synthSvcHTTP    = "synthSvcHTTP"
	synthSvcPass    = "synthSvcPass"
	synthAdminHTTP  = "synthAdminHTTP"
	synthAdminPass  = "synthAdminPass"
	synthAlicePass  = "pa55word"
	synthBobPass    = "hunter2x"
	synthNewPass    = "n3wSecret"
	synthNewerPass  = "n3werSecret"
	synthSvcPageLog = RightsNone
)

// hashedFixture writes a synthetic plain config as EncryptRijndaelHashed and
// reads it back, which is exactly how the shipped V2/V3 files come about.
func hashedFixture(t testing.TB) (*Config, string) {
	t.Helper()
	c := NewConfig()
	c.Version = "9.9.9-test"
	c.addManualFeatures()
	admin := NewGroup(c.Features, "Admin", "Administrators", synthAdminHTTP, synthAdminPass)
	svc := NewGroup(c.Features, "Service", "Our own service engineers", synthSvcHTTP, synthSvcPass)
	for _, fr := range svc.Features {
		switch fr.Feature.ID {
		case PageLog:
			fr.Rights = synthSvcPageLog
		case PropertyFeatureID(8290):
			fr.Rights = RightsReadOnly
		case FeatureCreateFWU:
			fr.Rights = RightsNone
		}
	}
	c.Groups = []*Group{admin, svc}
	c.Users = []*User{
		{User: "ISAH", Password: synthISAH, Group: admin},
		{User: "Alice", Password: synthAlicePass, Group: svc, Fullname: "Alice Example"},
		{User: "bob", Password: synthBobPass, Group: admin},
	}
	text, err := c.Marshal(EncryptRijndaelHashed, true, "1/2/2026 3:04:05 PM")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(text, ConfigPassPhrase)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Parse(text, EncryptRijndaelHashed)
	if err != nil {
		t.Fatal(err)
	}
	return loaded, plain
}

func TestHashedConfigShape(t *testing.T) {
	c, plain := hashedFixture(t)
	for _, secret := range []string{synthAlicePass, synthBobPass, synthISAH, synthSvcHTTP, synthSvcPass, synthAdminPass, "Alice"} {
		if strings.Contains(plain, secret) {
			t.Fatalf("hashed config leaks %q", secret)
		}
	}
	if len(c.Users) != 2 {
		t.Fatalf("isah must be excluded: %d users", len(c.Users))
	}
	if !slices.IsSortedFunc(c.Users, func(a, b *User) int { return strings.Compare(a.Password, b.Password) }) {
		t.Fatal("users not ordered by Password")
	}
	names := []string{c.Users[0].User, c.Users[1].User}
	if !slices.Contains(names, HashUserName("alice")) || !slices.Contains(names, HashUserName("BOB")) {
		t.Fatal("user names are not Base64(SHA256(lower(name)))")
	}
	for _, g := range c.Groups {
		if g.HTTPUser != "" || g.HTTPPassword != "" {
			t.Fatal("hashed config must blank group credentials")
		}
	}
	for _, u := range c.Users {
		if u.Fullname != "" {
			t.Fatal("hashed users have an empty Fullname")
		}
	}
}

func TestLogon(t *testing.T) {
	c, _ := hashedFixture(t)

	u, err := c.Logon("ALICE", synthAlicePass, nil)
	if err != nil {
		t.Fatalf("Logon: %v", err)
	}
	if u.Group.Name != "Service" || u.Group.HTTPUser != synthSvcHTTP || u.Group.HTTPPassword != synthSvcPass {
		t.Fatal("group or HTTP credentials not recovered")
	}
	if u.Company != synthISAH || u.Fullname != "" || u.User != HashUserName("alice") {
		t.Fatal("Company/Fullname/User not as CreateNewUser builds them")
	}
	if u.GetRights(PageLog) != RightsNone || u.GetRights(PropertyFeatureID(8290)) != RightsReadOnly || u.GetRights(PageInformation) != RightsReadOnly {
		t.Fatal("rights not copied from the config group")
	}
	if slices.Contains(VisiblePages(u, false), PageLog) || CanCreateFWU(u) || CanUseLogCommands(u) {
		t.Fatal("Service user should not see the log page / FWU button")
	}

	b, err := c.Logon("bob", synthBobPass, nil)
	if err != nil {
		t.Fatal(err)
	}
	if b.Group.Name != "Admin" || b.Group.HTTPUser != synthAdminHTTP || b.GetRights("PAGE_PRODUCTION") != RightsFull || !CanCreateFWU(b) {
		t.Fatal("admin logon")
	}
	if b.Group == c.FindGroup("Admin") {
		t.Fatal("Logon must return a copy of the group, not the config group")
	}

	if _, err := c.Logon("alice", "wrong", nil); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("wrong password: %v", err)
	}
	if _, err := c.Logon("isah", synthISAH, nil); !errors.Is(err, ErrUnknownUser) {
		t.Fatalf("isah is not a logon user: %v", err)
	}
	if _, err := c.Logon("mallory", "x", nil); !errors.Is(err, ErrUnknownUser) {
		t.Fatalf("unknown user: %v", err)
	}
	// A corrupt local entry for this user aborts the logon (C#: logged, null).
	if _, err := c.Logon("alice", "wrong", []string{HashUserName("alice") + ";not json"}); err == nil || errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("corrupt local entry: %v", err)
	}
	// Entries of other users are skipped.
	if _, err := c.Logon("alice", "wrong", []string{HashUserName("bob") + ";not json"}); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("foreign local entry: %v", err)
	}
}

func TestChangeLocalPassword(t *testing.T) {
	c, _ := hashedFixture(t)

	if _, err := c.ChangeLocalPassword("alice", "wrong", synthNewPass, synthNewPass, nil); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("bad current password: %v", err)
	}
	if _, err := c.ChangeLocalPassword("alice", synthAlicePass, synthNewPass, "other", nil); !errors.Is(err, ErrNewPasswordMismatch) {
		t.Fatalf("mismatch: %v", err)
	}
	local, err := c.ChangeLocalPassword("Alice", synthAlicePass, synthNewPass, synthNewPass, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(local) != 1 || !strings.HasPrefix(local[0], HashUserName("alice")+";\n\t{\"User\":") {
		t.Fatal("local entry is not \"<hash>;<ICUUser.Json>\"")
	}
	if strings.Contains(local[0], synthNewPass) || strings.Contains(local[0], synthSvcPass) {
		t.Fatal("local entry leaks a secret")
	}
	u, err := c.Logon("alice", synthNewPass, local)
	if err != nil {
		t.Fatalf("logon with local password: %v", err)
	}
	if u.Group.Name != "Service" || u.Group.HTTPPassword != synthSvcPass || u.Company != synthISAH || u.GetRights(PageLog) != RightsNone {
		t.Fatal("local-password logon lost group/credentials/rights")
	}
	if _, err := c.Logon("alice", synthAlicePass, local); err != nil {
		t.Fatalf("original password must keep working: %v", err)
	}
	if _, err := c.Logon("alice", synthNewPass, nil); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatal("local password must only work with the local list")
	}
	// Changing again appends: the C# removal only matches entries with more
	// than two ';' fields, so the previous entry stays (and keeps working).
	local2, err := c.ChangeLocalPassword("alice", synthNewPass, synthNewerPass, synthNewerPass, local)
	if err != nil {
		t.Fatal(err)
	}
	if len(local2) != 2 || local2[0] != local[0] {
		t.Fatalf("expected the old entry to be kept, got %d entries", len(local2))
	}
	for _, pw := range []string{synthNewPass, synthNewerPass} {
		if _, err := c.Logon("alice", pw, local2); err != nil {
			t.Fatalf("local password %d: %v", len(pw), err)
		}
	}
	// An entry with an extra ';' field is the one case the C# removes.
	odd := HashUserName("alice") + ";x;y"
	local3, err := c.ChangeLocalPassword("alice", synthAlicePass, synthNewPass, synthNewPass, []string{odd})
	if err != nil {
		t.Fatal(err)
	}
	if len(local3) != 1 || local3[0] == odd {
		t.Fatal("entry with >2 fields should have been removed")
	}
}

// TestCreateHashedUserDeterministic pins the "NN:SALT:b64" scheme with a
// fixed RNG stream (4 salt bytes, then GetNonZeroBytes skipping 0x00).
func TestCreateHashedUserDeterministic(t *testing.T) {
	rng := bytes.NewReader([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x03})
	g := NewGroup(nil, "Service", "", "", "")
	u, err := createHashedUser(rng, "Alice", "pw", g, "isahdata", "hu:hp")
	if err != nil {
		t.Fatal(err)
	}
	h := []byte("DEADBEEFpw")
	for i := 0; i < 3; i++ {
		s := sha256.Sum256(h)
		h = s[:]
	}
	if want := "03:DEADBEEF:" + base64.StdEncoding.EncodeToString(h); u.Password != want {
		t.Fatalf("Password = %s, want %s", u.Password, want)
	}
	nameSum := sha256.Sum256([]byte("alice"))
	if u.User != base64.StdEncoding.EncodeToString(nameSum[:]) || u.Group != g || u.Fullname != "" {
		t.Fatal("User/Group/Fullname")
	}
	if c, _ := Decrypt(u.Comment, "DEADBEEFpw"); c != "hu:hp" {
		t.Fatal("Comment must be Encrypt(httpData, salt+password)")
	}
	if c, _ := Decrypt(u.Company, "DEADBEEFpw"); c != "isahdata" {
		t.Fatal("Company must be Encrypt(isahData, salt+password)")
	}
	for pw, want := range map[string]bool{"pw": true, "PW": false, "": false} {
		if ok, err := u.ValidateHashedPassword(pw); err != nil || ok != want {
			t.Errorf("ValidateHashedPassword(%q) = %v, %v", pw, ok, err)
		}
	}
}

func TestValidateHashedPasswordFormat(t *testing.T) {
	tests := []struct {
		password string
		ok       bool
		wantErr  bool
	}{
		{"plain", false, false},      // not 3 parts
		{"a:b", false, false},        // not 3 parts
		{"a:b:c:d", false, false},    // not 3 parts
		{"ZZ:00:xx", false, true},    // Convert.ToUInt16("ZZ", 16) throws
		{"0x01:00:xx", false, false}, // 0x prefix accepted, digest mismatch
		{"10000:00:xx", false, true}, // overflow
	}
	for _, tc := range tests {
		ok, err := (&User{Password: tc.password}).ValidateHashedPassword("pw")
		if ok != tc.ok || (err != nil) != tc.wantErr {
			t.Errorf("%q: %v, %v", tc.password, ok, err)
		}
	}
}

func TestMarshalHashedRequirements(t *testing.T) {
	c := smallConfig() // no "isah" user
	if _, err := c.Marshal(EncryptRijndaelHashed, true, "D"); err == nil {
		t.Fatal("hashed write without isah user must fail")
	}
	c.Users = append(c.Users, &User{User: "isah", Password: "x"}, &User{User: "nogroup", Password: "y"})
	if _, err := c.Marshal(EncryptRijndaelHashed, true, "D"); err == nil {
		t.Fatal("hashed write with a group-less user must fail")
	}
}

func TestLogonTexts(t *testing.T) {
	if InitialLogonUserName("") != "User" || InitialLogonUserName("bob") != "bob" {
		t.Error("InitialLogonUserName")
	}
	if got := LogonVersionText("2.3.0-1187", "4.1.2.345"); got != "Settings version: 2.3.0-1187\nApplication version: 4.1.2-345" {
		t.Errorf("LogonVersionText = %q", got)
	}
	if FormatAppVersion("4.1.2") != "4.1.2" || FormatAppVersion("1.2.3.4.5") != "1.2.3-4" {
		t.Error("FormatAppVersion")
	}
}
