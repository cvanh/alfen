package config

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Fixtures are real installer configs; tests only assert structure, counts
// and round trips and never print decrypted values.
const fixtureDir = "../../../firmware/"

type fixture struct {
	dat, twin string // twin: already-decrypted JSON ("" when none)
	hashed    bool   // users stored hashed (EncryptRijndaelHashed)
}

var fixtures = []fixture{
	{"InstallerConfig.dat", "InstallerConfig.json", false},
	{"InstallerConfigV2.dat", "InstallerConfigV2.json", true},
	{"InstallerConfigV3.dat", "InstallerConfigV3.json", true},
	{"msi_work/files3/InstallerConfigV3.dat", "", true},
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(fixtureDir + name)
	if err != nil {
		t.Skipf("fixture %s not available: %v", name, err)
	}
	return b
}

// firstDiff returns the first differing offset (or -1); used instead of
// printing content so no decrypted data ends up in test logs.
func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

func genericJSON(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("generic decode: %v", err)
	}
	return v
}

func TestFixtureDecryptMatchesTwin(t *testing.T) {
	for _, fx := range fixtures {
		if fx.twin == "" {
			continue
		}
		t.Run(fx.dat, func(t *testing.T) {
			dat := readFixture(t, fx.dat)
			twin := readFixture(t, fx.twin)
			plain, err := Decrypt(readAllText(dat), ConfigPassPhrase)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if !reflect.DeepEqual(genericJSON(t, []byte(plain)), genericJSON(t, twin)) {
				t.Fatalf("decrypted %s is not semantically equal to %s", fx.dat, fx.twin)
			}
			if d := firstDiff([]byte(plain), twin); d >= 0 {
				t.Logf("semantically equal but bytes differ at offset %d (len %d vs %d)", d, len(plain), len(twin))
			}
		})
	}
}

func TestFixtureLoad(t *testing.T) {
	for _, fx := range fixtures {
		t.Run(fx.dat, func(t *testing.T) {
			readFixture(t, fx.dat)
			c, err := Load(fixtureDir+fx.dat, EncryptRijndaelHashed)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if c.Version == "" || c.Date == "" {
				t.Errorf("Version/Date empty")
			}
			if len(c.Users) == 0 || len(c.Groups) == 0 || len(c.Firmwares) == 0 {
				t.Fatalf("empty collections: users=%d groups=%d firmwares=%d", len(c.Users), len(c.Groups), len(c.Firmwares))
			}
			// Every manual feature must be present exactly once.
			ref := &Config{}
			ref.addManualFeatures()
			seen := map[string]int{}
			for _, f := range c.Features {
				seen[f.ID]++
			}
			for _, f := range ref.Features {
				if seen[f.ID] != 1 {
					t.Errorf("manual feature %s present %d times", f.ID, seen[f.ID])
				}
			}
			// One BO_ feature per titled backoffice.
			nBO := 0
			for _, f := range c.Features {
				if f.Type == FeatureTypeBackoffice {
					nBO++
					if !strings.HasPrefix(f.ID, "BO_") || f.Default != RightsReadOnly {
						t.Errorf("bad backoffice feature %s/%v", f.ID, f.Default)
					}
				}
			}
			titled := 0
			for _, raw := range c.BackOffices {
				h, err := backOfficeHeader(raw)
				if err != nil {
					t.Fatal(err)
				}
				if h.Title != "" {
					titled++
				}
			}
			if nBO != titled {
				t.Errorf("backoffice features = %d, titled backoffices = %d", nBO, titled)
			}
			// Group membership counts and feature-right lists.
			total := 0
			for _, g := range c.Groups {
				total += g.NumberOfUsers
				if len(g.Features) != len(c.Features) {
					t.Errorf("group %d has %d feature rights, config has %d features", total, len(g.Features), len(c.Features))
				}
			}
			withGroup := 0
			for _, u := range c.Users {
				if u.Group != nil {
					withGroup++
				}
			}
			if total != withGroup || withGroup != len(c.Users) {
				t.Errorf("NumberOfUsers sum %d, users with group %d, users %d", total, withGroup, len(c.Users))
			}
			// Hashed configs: every user is "NN:SALT:b64", names are SHA-256
			// Base64 and the group credentials are blank.
			if fx.hashed {
				for i, u := range c.Users {
					p := strings.Split(u.Password, ":")
					if len(p) != 3 || len(p[1]) != 8 || len(u.User) != 44 {
						t.Fatalf("user %d is not in hashed form", i)
					}
					if _, err := parseUInt16Hex(p[0]); err != nil {
						t.Fatalf("user %d: %v", i, err)
					}
				}
				for _, g := range c.Groups {
					if g.HTTPUser != "" || g.HTTPPassword != "" {
						t.Errorf("hashed config group carries credentials")
					}
				}
			}
			if fx.twin == "" {
				return
			}
			tw := genericJSON(t, readFixture(t, fx.twin))
			for key, got := range map[string]int{
				"Users": len(c.Users), "Groups": len(c.Groups), "Firmwares": len(c.Firmwares),
				"Backoffices": len(c.BackOffices), "PMBackOffices": len(c.PMBackOffices),
			} {
				if want := len(tw[key].([]any)); got != want {
					t.Errorf("%s: got %d, twin has %d", key, got, want)
				}
			}
		})
	}
}

// TestFixtureRewriteByteExact re-serialises each decrypted twin with the
// C#-format writer (EncryptNone keeps users and credentials as they are) and
// expects the original bytes back. V3 was written by the C# without its
// backoffices (addBackoffices=false) while its groups still reference "BO_*"
// features, so for V3 those FeatureRights entries cannot survive a reload; it
// is compared after dropping them.
func TestFixtureRewriteByteExact(t *testing.T) {
	for _, fx := range fixtures {
		if fx.twin == "" {
			continue
		}
		t.Run(fx.twin, func(t *testing.T) {
			twin := readFixture(t, fx.twin)
			c, err := ParseJSON(twin)
			if err != nil {
				t.Fatalf("ParseJSON: %v", err)
			}
			out, err := c.Marshal(EncryptNone, true, c.Date)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if d := firstDiff([]byte(out), twin); d < 0 {
				return
			}
			t.Logf("not byte-exact (first diff at %d); comparing semantically without orphan BO_ rights", firstDiff([]byte(out), twin))
			// Semantic comparison with BO_ feature rights removed from the original.
			want := genericJSON(t, twin)
			for _, g := range want["Groups"].([]any) {
				gm := g.(map[string]any)
				var keep []any
				for _, fr := range gm["FeatureRights"].([]any) {
					if !strings.HasPrefix(fr.(map[string]any)["ID"].(string), "BO_") || len(c.BackOffices) > 0 {
						keep = append(keep, fr)
					}
				}
				if keep == nil {
					keep = []any{}
				}
				gm["FeatureRights"] = keep
			}
			if len(c.BackOffices) > 0 {
				t.Fatalf("rewrite differs at offset %d (len %d vs %d)", firstDiff([]byte(out), twin), len(out), len(twin))
			}
			if !reflect.DeepEqual(genericJSON(t, []byte(out)), want) {
				t.Fatalf("rewrite is not semantically equal (ignoring orphan BO_ rights)")
			}
		})
	}
}

// TestFixtureEnvelopeRoundTrip re-encrypts a real config with every
// ICUEncryptionType envelope and reads it back.
func TestFixtureEnvelopeRoundTrip(t *testing.T) {
	twin := readFixture(t, "InstallerConfigV3.json")
	c, err := ParseJSON(twin)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := c.Marshal(EncryptNone, true, c.Date)
	if err != nil {
		t.Fatal(err)
	}
	for _, enc := range []EncryptionType{EncryptNone, EncryptBase64, EncryptRijndael} {
		text, err := c.Marshal(enc, true, c.Date)
		if err != nil {
			t.Fatalf("%v: %v", enc, err)
		}
		if enc != EncryptNone && bytes.Contains([]byte(text), []byte(FileID)) {
			t.Fatalf("%v: envelope leaves plaintext visible", enc)
		}
		back, err := Parse(text, enc)
		if err != nil {
			t.Fatalf("%v: Parse: %v", enc, err)
		}
		again, _ := back.Marshal(EncryptNone, true, back.Date)
		if again != ref {
			t.Fatalf("%v: round trip differs at %d", enc, firstDiff([]byte(again), []byte(ref)))
		}
	}
}
