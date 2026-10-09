package backoffice

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// loadConfigArrays reads a decrypted InstallerConfig*.json fixture and returns
// the raw "Backoffices"/"PMBackOffices" values (the config package keeps them
// raw in the same way).
func loadConfigArrays(t *testing.T, name string) (bo, pm json.RawMessage) {
	t.Helper()
	data, err := os.ReadFile("../../../firmware/" + name)
	if err != nil {
		t.Skipf("fixture %s not available: %v", name, err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return top["Backoffices"], top["PMBackOffices"]
}

var reEntryLine = regexp.MustCompile(`^\t\t\{"Key":"([^"]*)","Value":".*"\},?$`)

// expectedV441 rewrites a fixture array the way v4.4.1 would re-emit it: the
// newer tool that wrote the fixtures knew more keys (OD_commProfileN_*, ...);
// ICUBackOffice/ICUPMBackOffice.SetProperty drop those, so their lines go and
// the trailing comma of the new last entry of each section is removed.
func expectedV441(raw string, known func(section, key string) bool) string {
	lines := strings.Split(raw, "\n")
	var kept []string
	section := ""
	for _, l := range lines {
		switch {
		case strings.HasSuffix(l, `"ValuesEx":[`):
			section = "ValuesEx"
		case strings.HasSuffix(l, `"Values":[`):
			section = "Values"
		}
		if m := reEntryLine.FindStringSubmatch(l); m != nil && !known(section, m[1]) {
			continue
		}
		kept = append(kept, l)
	}
	for i := 0; i+1 < len(kept); i++ {
		if reEntryLine.MatchString(kept[i]) && strings.HasPrefix(kept[i+1], "\t]") {
			kept[i] = strings.TrimSuffix(kept[i], ",")
		}
	}
	return strings.Join(kept, "\n")
}

func knownBO(section, key string) bool {
	for _, d := range backOfficeDefs {
		if d.name == key {
			return (section == "Values" && d.typ == Normal) || (section == "ValuesEx" && d.typ == Extended)
		}
	}
	return false
}

func knownPM(_, key string) bool {
	for _, d := range pmDefs {
		if d.name == key {
			return true
		}
	}
	return false
}

func TestFixtureInstallerConfigs(t *testing.T) {
	cases := []struct {
		file   string
		bo, pm int
	}{
		{"InstallerConfig.json", 131, 519},
		{"InstallerConfigV2.json", 131, 467},
		{"InstallerConfigV3.json", 0, 0},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			rawBO, rawPM := loadConfigArrays(t, c.file)
			bos, err := ParseBackoffices(rawBO)
			if err != nil {
				t.Fatal(err)
			}
			pms, err := ParsePMBackoffices(rawPM)
			if err != nil {
				t.Fatal(err)
			}
			if len(bos) != c.bo || len(pms) != c.pm {
				t.Fatalf("counts: %d backoffices, %d PM backoffices; want %d, %d", len(bos), len(pms), c.bo, c.pm)
			}

			// Byte-exact against the C#-written file, minus the keys v4.4.1 drops.
			if got, want := string(MarshalBackoffices(bos)), expectedV441(string(rawBO), knownBO); got != want {
				t.Errorf("Backoffices re-emit differs from fixture (len %d vs %d)\n%s", len(got), len(want), firstDiff(got, want))
			}
			if got, want := string(MarshalPMBackoffices(pms)), expectedV441(string(rawPM), knownPM); got != want {
				t.Errorf("PMBackOffices re-emit differs from fixture (len %d vs %d)\n%s", len(got), len(want), firstDiff(got, want))
			}

			// Re-parsing our own output is a fixed point.
			bos2, err := ParseBackoffices(MarshalBackoffices(bos))
			if err != nil || BackofficesJSON(bos2) != BackofficesJSON(bos) {
				t.Errorf("backoffice round trip not idempotent: %v", err)
			}
			pms2, err := ParsePMBackoffices(MarshalPMBackoffices(pms))
			if err != nil || PMBackofficesJSON(pms2) != PMBackofficesJSON(pms) {
				t.Errorf("PM round trip not idempotent: %v", err)
			}

			for i, b := range bos {
				if b.Dirty() {
					t.Errorf("backoffice %d dirty after load", i)
				}
				if b.ConnectMethod() < 0 || b.ConnectMethod() > 3 {
					t.Errorf("backoffice %d: connect method %d", i, b.ConnectMethod())
				}
			}
			if c.bo == 0 {
				return
			}
			feats := Features(bos)
			if len(feats) == 0 || len(feats) > len(bos) {
				t.Errorf("features: %d", len(feats))
			}
			for _, f := range feats {
				if !strings.HasPrefix(f.ID, "BO_") || strings.Contains(f.ID, " ") {
					t.Errorf("feature id %q", f.ID)
				}
			}

			// UpdatePMBackOffices over the real list: every derived entry is
			// named after a source title's "<name> - " prefix (+ " sandbox")
			// and has at least one interface enabled. (The fixture's own PM
			// list was curated by a newer tool, so it is not compared.)
			derived, err := DerivePMBackOffices(bos)
			if err != nil {
				t.Fatal(err)
			}
			if len(derived) == 0 || len(derived) > len(bos) {
				t.Fatalf("derived %d PM backoffices from %d", len(derived), len(bos))
			}
			prefixes := map[string]bool{}
			for _, b := range bos {
				ti := b.Title()
				p := csTrim(ti[:strings.LastIndexByte(ti, '-')])
				prefixes[p] = true
				prefixes[p+" sandbox"] = true
			}
			overlap := 0
			have := map[string]bool{}
			for _, p := range pms {
				have[p.Title()] = true
			}
			for _, d := range derived {
				if !prefixes[d.Title()] {
					t.Errorf("derived PM %q is not a backoffice title prefix", d.Title())
				}
				if !d.IsLANEnabled() && !d.IsGPRSEnabled() {
					t.Errorf("derived PM %q has no interface", d.Title())
				}
				if have[d.Title()] {
					overlap++
				}
			}
			t.Logf("derived %d PM backoffices, %d also in the fixture PM list", len(derived), overlap)
		})
	}
}

func firstDiff(a, b string) string {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			lo := max(0, i-120)
			return "at " + strconv.Itoa(i) + ":\n got ..." + quoteTail(a[lo:min(len(a), i+120)]) + "\nwant ..." + quoteTail(b[lo:min(len(b), i+120)])
		}
	}
	return "length differs"
}

func quoteTail(s string) string { b, _ := json.Marshal(s); return string(b) }
