package backoffice

import (
	"encoding/json"
	"strings"
	"testing"
)

// boJSON is one backoffice exactly as ICUBackOffice.Json emits it
// (hand-built string.Format, InitializeProperties order).
const boJSON = "\n\t{\"Title\":\"Acme - production gprs\",\"TitleNL\":\"Acme NL\",\"TitleDE\":\"Acme DE\",\"TitleFR\":\"Acme FR\",\"Groups\":\"ICU|ACME\",\"Values\":[" +
	"\n\t\t{\"Key\":\"OD_commConnectMethod\",\"Value\":\"2\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNname\",\"Value\":\"apn.example\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNuser\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNpassword\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS1_1_value\",\"Value\":\"10.0.0.1\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS1_2_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS2_1_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS2_2_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverDomainAndPort\",\"Value\":\"ws://gprs.example:9000\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverPath\",\"Value\":\"ocpp\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverDomainAndPort\",\"Value\":\"ws://wired.example:9000\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverPath\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commSendStationStatus\",\"Value\":\"True\"}," +
	"\n\t\t{\"Key\":\"OD_mainOfflineNFCAuthorization\",\"Value\":\"1\"}," +
	"\n\t\t{\"Key\":\"OD_commProtocolName\",\"Value\":\"ocpp/json\"}," +
	"\n\t\t{\"Key\":\"OD_commProtocolVersion\",\"Value\":\"1.6\"}," +
	"\n\t\t{\"Key\":\"OD_mainEVDisconnectTimeout\",\"Value\":\"10\"}," +
	"\n\t\t{\"Key\":\"OD_mainEVDisconnectAction\",\"Value\":\"0\"}" +
	"\n\t], \"ValuesEx\":[" +
	"\n\t\t{\"Key\":\"OD_sysIntensity_auto\",\"Value\":\"False\"}," +
	"\n\t\t{\"Key\":\"OD_sysIntensity_intensity\",\"Value\":\"80\"}," +
	"\n\t\t{\"Key\":\"OD_sysTimeZoneMinutes\",\"Value\":\"60\"}," +
	"\n\t\t{\"Key\":\"OD_sysLanguage\",\"Value\":\"en_GB\"}," +
	"\n\t\t{\"Key\":\"OD_commPingPongInterval\",\"Value\":\"120\"}," +
	"\n\t\t{\"Key\":\"OD_mainOnlineNFCAuthorization\",\"Value\":\"True\"}," +
	"\n\t\t{\"Key\":\"OD_sysOCPP15SmartCharging_smartChargingType\",\"Value\":\"0\"}," +
	"\n\t\t{\"Key\":\"OD_gprsSIMpin\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commMeteringAlignment\",\"Value\":\"1\"}," +
	"\n\t\t{\"Key\":\"OD_commTransactionMessageAttempts\",\"Value\":\"3\"}," +
	"\n\t\t{\"Key\":\"OD_commTransactionMessageRetryInterval\",\"Value\":\"60\"}" +
	"\n\t]}"

func TestBackofficeJSONByteExactRoundTrip(t *testing.T) {
	raw := json.RawMessage("[" + boJSON + "," + boJSON + "\n]")
	bos, err := ParseBackoffices(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bos) != 2 {
		t.Fatalf("got %d backoffices", len(bos))
	}
	if got := string(MarshalBackoffices(bos)); got != string(raw) {
		t.Fatalf("round trip mismatch:\n got %q\nwant %q", got, raw)
	}
	b := bos[0]
	if b.Title() != "Acme - production gprs" || b.Groups() != "ICU|ACME" || b.TitleFR() != "Acme FR" {
		t.Errorf("titles: %q %q %q", b.Title(), b.Groups(), b.TitleFR())
	}
	if b.ConnectMethod() != 2 || !b.IsGPRS() || b.IsAuto() {
		t.Errorf("connect method %d gprs=%v auto=%v", b.ConnectMethod(), b.IsGPRS(), b.IsAuto())
	}
	if !b.SendStationStatus() || b.IntensityAuto() || b.IntensityIntensity() != 80 || b.Language() != "en_GB" {
		t.Errorf("typed getters wrong: %v %v %d %q", b.SendStationStatus(), b.IntensityAuto(), b.IntensityIntensity(), b.Language())
	}
	if b.TransactionMessageAttempts() != 3 || b.OfflineNFCAuthorization() != 1 || b.EVDisconnectTimeout() != 10 {
		t.Error("int getters wrong")
	}
	if _, ok := b.GetProperty("OD_commSendStationStatus").Value.(bool); !ok {
		t.Error("SendStationStatus must be held as a bool")
	}
	if b.Dirty() {
		t.Error("a loaded backoffice must not be dirty (ICUBackOffice(dynamic) uses base())")
	}
	if b.OriginalValues == nil || b.OriginalValues.GetProperty("Title") != nil {
		t.Error("OriginalValues must exist and (like CopyFrom) omit hidden properties")
	}
}

func TestNewBackOfficeDefaults(t *testing.T) {
	b := NewBackOffice()
	if !b.Dirty() {
		t.Error("new ICUBackOffice() is dirty")
	}
	want := "\n\t{\"Title\":\"\",\"TitleNL\":\"\",\"TitleDE\":\"\",\"TitleFR\":\"\",\"Groups\":\"\",\"Values\":[" +
		"\n\t\t{\"Key\":\"OD_commConnectMethod\",\"Value\":\"0\"}," +
		"\n\t\t{\"Key\":\"OD_gprsAPNname\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_gprsAPNuser\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_gprsAPNpassword\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commDNS1_1_value\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commDNS1_2_value\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commDNS2_1_value\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commDNS2_2_value\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverDomainAndPort\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverPath\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverDomainAndPort\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverPath\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commSendStationStatus\",\"Value\":\"False\"}," +
		"\n\t\t{\"Key\":\"OD_mainOfflineNFCAuthorization\",\"Value\":\"0\"}," +
		"\n\t\t{\"Key\":\"OD_commProtocolName\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commProtocolVersion\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_mainEVDisconnectTimeout\",\"Value\":\"0\"}," +
		"\n\t\t{\"Key\":\"OD_mainEVDisconnectAction\",\"Value\":\"0\"}" +
		"\n\t], \"ValuesEx\":[" +
		"\n\t\t{\"Key\":\"OD_sysIntensity_auto\",\"Value\":\"True\"}," +
		"\n\t\t{\"Key\":\"OD_sysIntensity_intensity\",\"Value\":\"100\"}," +
		"\n\t\t{\"Key\":\"OD_sysTimeZoneMinutes\",\"Value\":\"60\"}," +
		"\n\t\t{\"Key\":\"OD_sysLanguage\",\"Value\":\"nl_NL\"}," +
		"\n\t\t{\"Key\":\"OD_commPingPongInterval\",\"Value\":\"120\"}," +
		"\n\t\t{\"Key\":\"OD_mainOnlineNFCAuthorization\",\"Value\":\"True\"}," +
		"\n\t\t{\"Key\":\"OD_sysOCPP15SmartCharging_smartChargingType\",\"Value\":\"0\"}," +
		"\n\t\t{\"Key\":\"OD_gprsSIMpin\",\"Value\":\"\"}," +
		"\n\t\t{\"Key\":\"OD_commMeteringAlignment\",\"Value\":\"1\"}," +
		"\n\t\t{\"Key\":\"OD_commTransactionMessageAttempts\",\"Value\":\"0\"}," +
		"\n\t\t{\"Key\":\"OD_commTransactionMessageRetryInterval\",\"Value\":\"60\"}" +
		"\n\t]}"
	if got := b.JSON(); got != want {
		t.Fatalf("defaults:\n got %q\nwant %q", got, want)
	}
	if n := len(GetPropertyIds()); n != 29 {
		t.Fatalf("GetPropertyIds: %d ids", n)
	}
	for _, id := range GetPropertyIds() {
		p := b.GetPropertyByNumber(id)
		if p == nil || p.Type == Hidden {
			t.Errorf("id %d has no non-hidden property", id)
		}
	}
	if string(MarshalBackoffices(nil)) != "[\n]" {
		t.Error("empty array must be \"[\\n]\"")
	}
}

func TestBackofficeLoadConversions(t *testing.T) {
	raw := `[{"Title":"  X - sandbox auto ","TitleNL":null,"TitleDE":"d","TitleFR":"f","Groups":"G",
	  "Values":[
	    {"Key":"OD_commConnectMethod","Value":" 3 "},
	    {"Key":"OD_commSendStationStatus","Value":"1"},
	    {"Key":"OD_mainOfflineNFCAuthorization","Value":"bogus"},
	    {"Key":"OD_commBackOfficeURL_serverPath","Value":"a\r\nb\nc"},
	    {"Key":"OD_commProfile1_CSMSUrl","Value":"dropped"},
	    {"Key":"OD_sysLanguage","Value":"de_DE"},
	    {"Key":"OD_mainEVDisconnectTimeout","Value":5}
	  ]}]`
	bos, err := ParseBackoffices(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	b := bos[0]
	if b.Title() != "X - sandbox auto" || b.TitleNL() != "" {
		t.Errorf("Title %q TitleNL %q", b.Title(), b.TitleNL())
	}
	if b.ConnectMethod() != 3 || !b.IsAuto() || !b.IsGPRS() {
		t.Errorf("ConnectMethod %d", b.ConnectMethod())
	}
	if !b.SendStationStatus() {
		t.Error(`"1" must load as true`)
	}
	if b.OfflineNFCAuthorization() != 0 {
		t.Error("an unparsable int leaves the default")
	}
	if b.Language() != "de_DE" {
		t.Error("an extended key inside Values still sets the property by name")
	}
	if b.EVDisconnectTimeout() != 5 {
		t.Error("numeric JSON value")
	}
	if b.GetProperty("OD_commProfile1_CSMSUrl") != nil {
		t.Error("unknown keys must be dropped")
	}
	js := b.JSON()
	if !strings.Contains(js, `{"Key":"OD_commBackOfficeURL_serverPath","Value":"a, b, c"}`) {
		t.Errorf("newline replacement missing in %q", js)
	}
	if strings.Contains(js, "OD_commProfile1") {
		t.Error("unknown keys must not be written back")
	}
}

func TestBackofficeParseErrors(t *testing.T) {
	for _, raw := range []string{
		``,
		`null`,
		`{}`,
		`[1]`,
		`[{"TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[]}]`,
		`[{"Title":"","TitleNL":"","TitleDE":"","TitleFR":"","Groups":""}]`,
		`[{"Title":"","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":[{"Value":"x"}]}]`,
		`[{"Title":"","TitleNL":"","TitleDE":"","TitleFR":"","Groups":"","Values":null}]`,
	} {
		if _, err := ParseBackoffices(json.RawMessage(raw)); err == nil {
			t.Errorf("ParseBackoffices(%q): expected error", raw)
		}
	}
}

func TestBackofficeDirtyCommitRollback(t *testing.T) {
	bos, err := ParseBackoffices(json.RawMessage("[" + boJSON + "\n]"))
	if err != nil {
		t.Fatal(err)
	}
	b := bos[0]
	b.SetTitle("renamed")
	if b.Dirty() {
		t.Error("hidden (title) edits are not compared by OnCheckDirty")
	}
	b.SetAPNName("other.apn")
	if !b.Dirty() {
		t.Error("APN edit must make it dirty")
	}
	b.Rollback()
	if b.APNName() != "apn.example" || b.Dirty() {
		t.Errorf("rollback: APN %q dirty %v", b.APNName(), b.Dirty())
	}
	if b.Title() != "" || b.GetProperty("Title") != nil {
		t.Error("Rollback drops the hidden title properties, as CopyFrom(OriginalValues) does in the C#")
	}
	b.SetPingPongInterval(30)
	b.Commit()
	if b.Dirty() || b.OriginalValues.PingPongInterval() != 30 {
		t.Error("commit must snapshot and clear dirty")
	}
}

func TestCopyBackOffice(t *testing.T) {
	bos, _ := ParseBackoffices(json.RawMessage("[" + boJSON + "\n]"))
	c := CopyBackOffice(bos[0])
	if c.Title() != "Acme - production gprs(copy)" || c.TitleNL() != "Acme NL(copy)" ||
		c.TitleDE() != "Acme DE(copy)" || c.TitleFR() != "Acme FR(copy)" {
		t.Errorf("copy titles: %q %q %q %q", c.Title(), c.TitleNL(), c.TitleDE(), c.TitleFR())
	}
	if c.Groups() != "ICU|ACME" || c.BackOfficeURL_Domain() != "ws://gprs.example:9000" {
		t.Error("copy must take all properties (allProperties: true)")
	}
	if !c.Dirty() || c.OriginalValues != nil {
		t.Error("a copy is a new, dirty backoffice")
	}
	if e := CopyBackOffice(nil); e.Title() != "(copy)" {
		t.Errorf("copy of nil: %q", e.Title())
	}
	if got := c.GetValue(2127616); got != 2 {
		t.Errorf("GetValue(ConnectMethod) = %#v", got)
	}
	if got := c.GetValueSub(8448, 0); got != "apn.example" {
		t.Errorf("GetValue(8448,0) = %#v", got)
	}
	if got := c.GetValue(0x123456); got != nil {
		t.Errorf("unknown id = %#v", got)
	}
}

func TestIsGPRSIsAuto(t *testing.T) {
	for cm, want := range map[int][2]bool{0: {false, false}, 1: {false, true}, 2: {true, false}, 3: {true, true}, 99: {true, false}} {
		b := NewBackOffice()
		b.SetConnectMethod(cm)
		if b.IsGPRS() != want[0] || b.IsAuto() != want[1] {
			t.Errorf("ConnectMethod %d: gprs=%v auto=%v", cm, b.IsGPRS(), b.IsAuto())
		}
	}
}

func TestFeatures(t *testing.T) {
	mk := func(title, groups string) *BackOffice {
		b := NewBackOffice()
		b.SetTitle(title)
		b.SetGroups(groups)
		return b
	}
	bos := []*BackOffice{mk("Acme - production gprs", "ICU|ACME"), mk("", "ICU"), mk("acme - production gprs", "TNM"), mk("Other one", "ACMECORP")}
	fs := Features(bos)
	if len(fs) != 2 || fs[0].ID != "BO_ACME_-_PRODUCTION_GPRS" || fs[0].Name != "BackOffice 'Acme - production gprs'" || fs[1].ID != "BO_OTHER_ONE" {
		t.Errorf("features: %+v", fs)
	}
	ids := CompanyFeatureIDs(bos, "acme")
	if len(ids) != 2 || ids[0] != "BO_ACME_-_PRODUCTION_GPRS" || ids[1] != "BO_OTHER_ONE" {
		t.Errorf("company ids (ordinal substring): %v", ids)
	}
	if FeatureID(" a b\t") != "BO__A_B" {
		t.Errorf("FeatureID trims after replacing spaces: %q", FeatureID(" a b\t"))
	}
}
