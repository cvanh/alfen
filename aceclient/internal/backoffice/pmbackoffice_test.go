package backoffice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// pmJSON is one part-management backoffice exactly as ICUPMBackOffice.Json
// emits it.
const pmJSON = "\n\t{\"Title\":\"Acme\",\"LANEnabled\":true,\"GPRSEnabled\":false,\"Values\":[" +
	"\n\t\t{\"Key\":\"Title\",\"Value\":\"Acme\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNname\",\"Value\":\"apn.example\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNuser\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_gprsAPNpassword\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_gprsSIMpin\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS1_1_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS1_2_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS2_1_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commDNS2_2_value\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverDomainAndPort\",\"Value\":\"ws://gprs.example:9000\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURL_serverPath\",\"Value\":\"\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverDomainAndPort\",\"Value\":\"ws://wired.example:9000\"}," +
	"\n\t\t{\"Key\":\"OD_commBackOfficeURLwired_serverPath\",\"Value\":\"a&b\"}," +
	"\n\t\t{\"Key\":\"OD_commProtocolName\",\"Value\":\"ocpp/json\"}," +
	"\n\t\t{\"Key\":\"OD_commProtocolVersion\",\"Value\":\"1.6\"}," +
	"\n\t\t{\"Key\":\"OD_mainEVDisconnectTimeout\",\"Value\":\"10\"}," +
	"\n\t\t{\"Key\":\"OD_sysIntensity_auto\",\"Value\":\"False\"}," +
	"\n\t\t{\"Key\":\"OD_sysIntensity_intensity\",\"Value\":\"100\"}," +
	"\n\t\t{\"Key\":\"OD_sysTimeZoneMinutes\",\"Value\":\"60\"}," +
	"\n\t\t{\"Key\":\"OD_sysOCPP15SmartCharging_smartChargingType\",\"Value\":\"0\"}," +
	"\n\t\t{\"Key\":\"OD_commSendStationStatus\",\"Value\":\"True\"}," +
	"\n\t\t{\"Key\":\"OD_commTransactionMessageAttempts\",\"Value\":\"0\"}," +
	"\n\t\t{\"Key\":\"OD_commTransactionMessageRetryInterval\",\"Value\":\"60\"}" +
	"\n\t] }"

func TestPMBackofficeJSONByteExactRoundTrip(t *testing.T) {
	raw := json.RawMessage("[" + pmJSON + "\n]")
	pms, err := ParsePMBackoffices(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(MarshalPMBackoffices(pms)); got != string(raw) {
		t.Fatalf("round trip mismatch:\n got %q\nwant %q", got, raw)
	}
	p := pms[0]
	if !p.IsLANEnabled() || p.IsGPRSEnabled() || p.Dirty() {
		t.Errorf("flags lan=%v gprs=%v dirty=%v", p.IsLANEnabled(), p.IsGPRSEnabled(), p.Dirty())
	}
	p.SetGPRSEnabled(true)
	if !p.Dirty() {
		t.Error("flag change must be dirty")
	}
	p.Rollback()
	if p.IsGPRSEnabled() {
		t.Error("rollback restores the flag")
	}
	p.SetEVDisconnectTimeout(20)
	p.CheckDirty()
	if !p.Dirty() {
		t.Error("value change must be dirty")
	}
	p.Commit()
	if p.Dirty() {
		t.Error("commit clears dirty")
	}
	// Properties without a backing entry read as defaults; setters are no-ops.
	p.SetLanguage("de_DE")
	p.SetPingPongInterval(5)
	if p.Language() != "" || p.PingPongInterval() != 0 || p.OnlineNFCAuthorization() {
		t.Error("PM has no Language/PingPong/OnlineNFC entries")
	}
}

func TestPMBackofficeOptionalKeys(t *testing.T) {
	pms, err := ParsePMBackoffices(json.RawMessage(`[{"Title":"T","Values":[]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !pms[0].IsLANEnabled() || !pms[0].IsGPRSEnabled() {
		t.Error("LAN/GPRS default to true when absent")
	}
	if pms[0].ProtocolName() != "ocpp/json" || pms[0].ProtocolVersion() != "1.6" || pms[0].EVDisconnectTimeout() != 10 {
		t.Error("PM defaults")
	}
	if got, _ := ParsePMBackoffices(nil); got != nil {
		t.Error("absent PMBackOffices key yields nothing")
	}
	for _, raw := range []string{`null`, `[{"Values":[]}]`, `[{"Title":"T"}]`, `[{"Title":"T","LANEnabled":"maybe","Values":[]}]`} {
		if _, err := ParsePMBackoffices(json.RawMessage(raw)); err == nil {
			t.Errorf("ParsePMBackoffices(%s): expected error", raw)
		}
	}
	if string(MarshalPMBackoffices(nil)) != "[\n]" {
		t.Error("empty PM array")
	}
}

func TestPMBackOfficeSettingsXML(t *testing.T) {
	pms, err := ParsePMBackoffices(json.RawMessage("[" + pmJSON + "\n]"))
	if err != nil {
		t.Fatal(err)
	}
	pms[0].SetTitle(" Acme Corp ")
	name, data := PMBackOfficeSettingsXML(pms[0])
	if name != "Acme-Corp.xml" {
		t.Errorf("file name %q", name)
	}
	want := "<Config>\r\n" +
		"  <Setting>\r\n" +
		"    <Product Model=\"NG9xx\" Device=\"NG9xx\">\r\n" +
		"      <Object Id=\"1.2100\" Value=\"apn.example\">APNName</Object>\r\n" +
		"      <Object Id=\"1.2078sub1\" Value=\"ws://gprs.example:9000\">BackOfficeURL_Domain</Object>\r\n" +
		"      <Object Id=\"1.2071sub1\" Value=\"ws://wired.example:9000\">BackOfficeURLwired_Domain</Object>\r\n" +
		"      <Object Id=\"1.2071sub2\" Value=\"a&amp;b\">BackOfficeURLwired_Path</Object>\r\n" +
		"      <Object Id=\"1.2081\" Value=\"ocpp/json\">ProtocolName</Object>\r\n" +
		"      <Object Id=\"1.2082\" Value=\"1.6\">ProtocolVersion</Object>\r\n" +
		"      <Object Id=\"1.2136\" Value=\"10\">EVDisconnectTimeout</Object>\r\n" +
		"      <Object Id=\"1.2061sub1\" Value=\"0\">IntensityAuto</Object>\r\n" +
		"      <Object Id=\"1.2061sub2\" Value=\"100\">IntensityIntensity</Object>\r\n" +
		"      <Object Id=\"1.206e\" Value=\"60\">TimezoneMinutes</Object>\r\n" +
		"      <Object Id=\"1.206csub1\" Value=\"0\">OCPP15SmartChargingType</Object>\r\n" +
		"      <Object Id=\"1.2093\" Value=\"1\">SendStationStatus</Object>\r\n" +
		"      <Object Id=\"1.2096\" Value=\"0\">TransactionMessageAttempts</Object>\r\n" +
		"      <Object Id=\"1.2097\" Value=\"60\">TransactionMessageRetryInterval</Object>\r\n" +
		"    </Product>\r\n" +
		"  </Setting>\r\n" +
		"</Config>\r\n"
	if string(data) != want {
		t.Fatalf("xml:\n got %q\nwant %q", data, want)
	}
	empty := NewPMBackOffice()
	for _, q := range empty.Properties {
		q.Value = ""
	}
	_, data = PMBackOfficeSettingsXML(empty)
	if string(data) != "<Config>\r\n  <Setting>\r\n    <Product Model=\"NG9xx\" Device=\"NG9xx\" />\r\n  </Setting>\r\n</Config>\r\n" {
		t.Errorf("empty product: %q", data)
	}
	dir := t.TempDir()
	if err := WritePMBackOfficeSettingsXML(dir, pms); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "Acme-Corp.xml")); err != nil || string(b) != want {
		t.Errorf("written file mismatch: %v", err)
	}
}

func TestDerivePMBackOffices(t *testing.T) {
	mk := func(title, apn, gprsURL, wiredURL string) *BackOffice {
		b := NewBackOffice()
		b.SetTitle(title)
		b.SetAPNName(apn)
		b.SetBackOfficeURL_Domain(gprsURL)
		b.SetBackOfficeURLwired_Domain(wiredURL)
		b.SetDNS2_1("dns-" + title)
		return b
	}
	bos := []*BackOffice{
		mk("Acme - production gprs", "acme.apn", "ws://g", ""),
		mk("Acme - production wired", "", "", "ws://w"),
		mk("Solo - production wired", "", "", "ws://solo"),
		mk("Both - production auto", "both.apn", "ws://bg", "ws://bw"),
		mk("Both - production gprs", "ignored", "", ""), // auto variant exists -> skipped
		mk("Sand - sandbox gprs", "s.apn", "ws://sg", ""),
		mk("Plain - other", "", "", ""),
		mk("Upper - Production GPRS", "u.apn", "", ""), // Replace("gprs") misses -> finds itself -> skipped
	}
	pms, err := DerivePMBackOffices(bos)
	if err != nil {
		t.Fatal(err)
	}
	type exp struct {
		title, apn, gprsURL, wiredURL, dns21 string
		lan, gprs                            bool
	}
	want := []exp{
		{"Acme", "acme.apn", "ws://g", "ws://w", "dns-Acme - production wired", true, true},
		{"Solo", "", "", "ws://solo", "dns-Solo - production wired", true, false},
		{"Both", "both.apn", "ws://bg", "ws://bw", "dns-Both - production auto", true, true},
		{"Sand sandbox", "s.apn", "ws://sg", "", "dns-Sand - sandbox gprs", false, true},
	}
	if len(pms) != len(want) {
		t.Fatalf("got %d PM backoffices: %v", len(pms), titles(pms))
	}
	for i, w := range want {
		p := pms[i]
		got := exp{p.Title(), p.APNName(), p.BackOfficeURL_Domain(), p.BackOfficeURLwired_Domain(), p.DNS2_1(), p.IsLANEnabled(), p.IsGPRSEnabled()}
		if got != w {
			t.Errorf("pm[%d] = %+v; want %+v", i, got, w)
		}
	}
	if _, err := DerivePMBackOffices([]*BackOffice{mk("no dash", "", "", "")}); err == nil {
		t.Error("a title without '-' must fail like Substring(0, -1)")
	}
}

func titles(pms []*PMBackOffice) []string {
	var out []string
	for _, p := range pms {
		out = append(out, p.Title())
	}
	return out
}

func TestCopyPMBackOffice(t *testing.T) {
	pms, _ := ParsePMBackoffices(json.RawMessage("[" + pmJSON + "\n]"))
	c := CopyPMBackOffice(pms[0])
	if c.Title() != "Acme(copy)" || c.APNName() != "apn.example" || c.IsGPRSEnabled() {
		t.Errorf("copy: %q %q %v", c.Title(), c.APNName(), c.IsGPRSEnabled())
	}
}
