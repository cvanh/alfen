package isah

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// fullObjectJSON has every IWSObject member, shaped like the C# model.
const fullObjectJSON = `{
  "Version": 3,
  "ObjectId": "ace0001234",
  "PartCode": "904460xxx",
  "Description": "Eve Single Pro-line",
  "OrderHeader": "OH-1",
  "OrderNumber": "ON-2",
  "Backoffice": "bo",
  "BackofficeSettingsVersion": "1.2",
  "IsEichrechtOrder": true,
  "PublicKeyBaseURL": "https://example.invalid/keys",
  "Language": "nl",
  "IsLanguageDefined": false,
  "LoadBalancing": 2,
  "IsPersonalizedDisplay": true,
  "LogoFileName": "logo.png",
  "Logo": "aGVs\nbG8=",
  "IsPartManagementEnabled": false,
  "FeaturesUnlockedText": "RFID reader",
  "IsThreePhases": true,
  "IsSimNeeded": false,
  "IsSSAEnabled": true,
  "Communication": {"SmallModule": true, "LargeModule": false, "GPRS": true, "UTP": false},
  "Interface": {"PlugCharge": false, "Display": true, "RFID": true, "OrangeLed": false, "HeartbeatLed": true},
  "Properties": [
    {"Id": 8609, "SubId": 0, "Value": "LICENSE"},
    {"Id": 8292, "SubId": 2, "Value": 16, "DebuggerDisplay": "ignored"},
    {"Id": 8448, "SubId": 1, "Value": 1.5},
    {"Id": 8449, "SubId": 0, "Value": true}
  ],
  "ArticleCollection": [
    {"Name": "N1", "PartCode": "P1", "DetailCode": "D1", "Quantity": 2, "Description": "first"}
  ],
  "InternalInfo": "info",
  "SomethingUnknown": {"nested": [1, 2, 3]}
}`

func TestDecodeIWSObjectFull(t *testing.T) {
	o, err := DecodeIWSObject([]byte(fullObjectJSON))
	if err != nil {
		t.Fatal(err)
	}
	want := &IWSObject{
		Version: 3, ObjectID: "ace0001234", PartCode: "904460xxx", Description: "Eve Single Pro-line",
		OrderHeader: "OH-1", OrderNumber: "ON-2", Backoffice: "bo", BackofficeSettingsVersion: "1.2",
		IsEichrechtOrder: true, PublicKeyBaseURL: "https://example.invalid/keys", Language: "nl",
		LoadBalancing: 2, IsPersonalizedDisplay: true, LogoFileName: "logo.png", Logo: "aGVs\nbG8=",
		FeaturesUnlockedText: "RFID reader", IsThreePhases: true, IsSSAEnabled: true,
		Communication: &CSCommunication{SmallModule: true, GPRS: true},
		Interface:     &CSInterface{Display: true, RFID: true, HeartbeatLed: true},
		Properties: []*IWSPropertyValue{
			{ID: 8609, SubID: 0, Value: "LICENSE"},
			{ID: 8292, SubID: 2, Value: int64(16)},
			{ID: 8448, SubID: 1, Value: 1.5},
			{ID: 8449, SubID: 0, Value: true},
		},
		ArticleCollection: []*Article{{Name: "N1", PartCode: "P1", DetailCode: "D1", Quantity: 2, Description: "first"}},
		InternalInfo:      "info",
	}
	if !reflect.DeepEqual(o, want) {
		gj, _ := json.MarshalIndent(o, "", " ")
		t.Fatalf("decoded object mismatch:\n%s", gj)
	}
	if !o.IsLanguageDefined() {
		t.Error("IsLanguageDefined should follow Language, not the JSON member")
	}
	if p := o.FindProperty(PropLicenseKey); p == nil || p.ValueString() != "LICENSE" {
		t.Errorf("FindProperty(8609) = %v", p)
	}
	if p := o.FindProperty(1); p != nil {
		t.Errorf("FindProperty(1) = %v, want nil", p)
	}
	if !o.HasPersonalizedLogo() {
		t.Error("HasPersonalizedLogo = false")
	}
	if b, err := o.LogoData(); err != nil || string(b) != "hello" {
		t.Errorf("LogoData = %q, %v", b, err)
	}
}

func TestDecodeIWSObjectNilAndErrors(t *testing.T) {
	nilCases := []string{"", "   \r\n\t", "null", "\xEF\xBB\xBFnull"}
	for _, in := range nilCases {
		o, err := DecodeIWSObject([]byte(in))
		if o != nil || err != nil {
			t.Errorf("DecodeIWSObject(%q) = %v, %v; want nil, nil", in, o, err)
		}
	}
	if o, err := DecodeIWSObject([]byte("\xEF\xBB\xBF{\"Version\":1}")); err != nil || o == nil || o.Version != 1 {
		t.Errorf("BOM-prefixed object = %+v, %v", o, err)
	}
	errCases := []string{
		`[]`, `"str"`, `12`, `true`, // not an object
		`{"Version":1} {}`,   // additional content
		`{"Version":1`,       // truncated
		`{"Version":01}`,     // Newtonsoft octal literal: strict JSON only here
		`{'Version':1}`,      // single quotes: strict JSON only here
		`{"Properties":{}}`,  // object for a list
		`{"Properties":[1]}`, // primitive element
		`{"Communication":[]}`,
		`{"Communication":"x"}`,
		`{"ArticleCollection":"x"}`,
	}
	for _, in := range errCases {
		if o, err := DecodeIWSObject([]byte(in)); err == nil {
			t.Errorf("DecodeIWSObject(%q) = %+v, want error", in, o)
		}
	}
}

func TestDecodeMemberMatching(t *testing.T) {
	o, err := DecodeIWSObject([]byte(`{"objectid":"a","VERSION":"7","Version":8,"language":"de","PROPERTIES":[{"id":1,"subid":2,"value":"v"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if o.ObjectID != "a" || o.Version != 8 || o.Language != "de" {
		t.Errorf("case-insensitive / last-wins matching: %+v", o)
	}
	if len(o.Properties) != 1 || *o.Properties[0] != (IWSPropertyValue{ID: 1, SubID: 2, Value: "v"}) {
		t.Errorf("properties = %v", o.Properties)
	}
}

func TestDecodePopulatesExisting(t *testing.T) {
	o, err := DecodeIWSObject([]byte(`{
		"Properties":[{"Id":1,"SubId":0,"Value":1}],
		"Communication":{"GPRS":true},
		"Properties":[null,{"Id":2,"SubId":0,"Value":2}],
		"Communication":{"UTP":true},
		"Interface":{"RFID":true},
		"Interface":null,
		"ArticleCollection":[{"PartCode":"A"}],
		"ArticleCollection":[null]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Properties) != 3 || o.Properties[0].ID != 1 || o.Properties[1] != nil || o.Properties[2].ID != 2 {
		t.Errorf("duplicate Properties should append: %v", o.Properties)
	}
	if o.Communication == nil || !o.Communication.GPRS || !o.Communication.UTP {
		t.Errorf("duplicate Communication should merge: %+v", o.Communication)
	}
	if o.Interface != nil {
		t.Errorf("null after object should reset Interface: %+v", o.Interface)
	}
	if len(o.ArticleCollection) != 2 || o.ArticleCollection[0].PartCode != "A" || o.ArticleCollection[1] != nil {
		t.Errorf("ArticleCollection = %v", o.ArticleCollection)
	}
	if p := o.FindProperty(2); p == nil || p.ValueString() != "2" {
		t.Errorf("FindProperty must skip nil entries: %v", p)
	}
	o2, err := DecodeIWSObject([]byte(`{"Properties":null,"ArticleCollection":null,"Communication":null}`))
	if err != nil || o2.Properties != nil || o2.ArticleCollection != nil || o2.Communication != nil {
		t.Errorf("nulls = %+v, %v", o2, err)
	}
}

func TestDecodeInt32Coercion(t *testing.T) {
	cases := []struct {
		raw    string
		want   int
		errSub string
	}{
		{`5`, 5, ""},
		{`-5`, -5, ""},
		{`"7"`, 7, ""},
		{`" +8 "`, 8, ""},
		{`2147483647`, 2147483647, ""},
		{`2147483648`, 0, "too large or small for an Int32"},
		{`1.0`, 0, "is not a valid integer"},
		{`1e2`, 0, "is not a valid integer"},
		{`true`, 0, "Unexpected character encountered while parsing value: t."},
		{`{}`, 0, "Unexpected character"},
		{`null`, 0, "Error converting value {null} to type 'System.Int32'."},
		{`""`, 0, "Error converting value {null} to type 'System.Int32'."},
		{`"1.5"`, 0, "Could not convert string to integer: 1.5."},
		{`"abc"`, 0, "Could not convert string to integer"},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"LoadBalancing":` + c.raw + `}`))
		checkCoercion(t, "LoadBalancing", c.raw, err, c.errSub, func() any { return o.LoadBalancing }, c.want)
	}
}

func TestDecodeBoolCoercion(t *testing.T) {
	cases := []struct {
		raw    string
		want   bool
		errSub string
	}{
		{`true`, true, ""},
		{`false`, false, ""},
		{`"True"`, true, ""},
		{`" false "`, false, ""},
		{`"TRUE"`, true, ""},
		{`1`, true, ""},
		{`0`, false, ""},
		{`0.0`, false, ""},
		{`-2.5`, true, ""},
		{`99999999999999999999`, true, ""}, // BigInteger != 0
		{`"yes"`, false, "Could not convert string to boolean: yes."},
		{`"1"`, false, "Could not convert string to boolean"},
		{`null`, false, "Error converting value {null} to type 'System.Boolean'."},
		{`""`, false, "Error converting value {null} to type 'System.Boolean'."},
		{`[]`, false, "Unexpected character encountered while parsing value: [."},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"IsSimNeeded":` + c.raw + `}`))
		checkCoercion(t, "IsSimNeeded", c.raw, err, c.errSub, func() any { return o.IsSimNeeded }, c.want)
	}
}

func TestDecodeStringCoercion(t *testing.T) {
	cases := []struct {
		raw    string
		want   string
		errSub string
	}{
		{`"nl"`, "nl", ""},
		{`12`, "12", ""},
		{`1.50`, "1.50", ""}, // literal text, not re-formatted
		{`-1E3`, "-1E3", ""},
		{`true`, "true", ""}, // JSON literal, lower case
		{`false`, "false", ""},
		{`null`, "", ""},
		{`1e400`, "", "Input string '1e400' is not a valid number."},
		{`{}`, "", "Unexpected character encountered while parsing value: {."},
		{`[]`, "", "Unexpected character encountered while parsing value: [."},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"Language":` + c.raw + `}`))
		checkCoercion(t, "Language", c.raw, err, c.errSub, func() any { return o.Language }, c.want)
	}
}

func TestDecodeReadOnlyMembers(t *testing.T) {
	// Get-only members are still read with their type's reader.
	if _, err := DecodeIWSObject([]byte(`{"IsLanguageDefined":"nope"}`)); err == nil {
		t.Error(`IsLanguageDefined:"nope" should fail ReadAsBoolean`)
	}
	if o, err := DecodeIWSObject([]byte(`{"IsLanguageDefined":true}`)); err != nil || o.IsLanguageDefined() {
		t.Errorf("IsLanguageDefined:true should be discarded: %v, %v", o, err)
	}
	if o, err := DecodeIWSObject([]byte(`{"IsLanguageDefined":null}`)); err != nil || o == nil {
		t.Errorf("IsLanguageDefined:null = %v, %v", o, err)
	}
	if _, err := DecodeIWSObject([]byte(`{"Properties":[{"Id":1,"DebuggerDisplay":{}}]}`)); err == nil {
		t.Error("DebuggerDisplay:{} should fail ReadAsString")
	}
}

func TestDecodePropertyValueMembers(t *testing.T) {
	cases := []struct {
		elem   string
		want   IWSPropertyValue
		errSub string
	}{
		{`{"Id":8609,"SubId":0}`, IWSPropertyValue{ID: 8609}, ""},
		{`{"Id":"8609","SubId":"3"}`, IWSPropertyValue{ID: 8609, SubID: 3}, ""},
		{`{"Id":" 12 "}`, IWSPropertyValue{ID: 12}, ""},
		{`{"Id":8609.5}`, IWSPropertyValue{ID: 8610}, ""}, // Convert.ToUInt16(double): half to even
		{`{"Id":8610.5}`, IWSPropertyValue{ID: 8610}, ""},
		{`{"Id":1.4}`, IWSPropertyValue{ID: 1}, ""},
		{`{"Id":-0.4}`, IWSPropertyValue{ID: 0}, ""},
		{`{"Id":true}`, IWSPropertyValue{ID: 1}, ""},
		{`{"Id":65535}`, IWSPropertyValue{ID: 65535}, ""},
		{`{"Id":65536}`, IWSPropertyValue{}, "Error converting value 65536 to type 'System.UInt16'."},
		{`{"Id":-1}`, IWSPropertyValue{}, "System.UInt16"},
		{`{"Id":65535.5}`, IWSPropertyValue{}, "System.UInt16"},
		{`{"Id":""}`, IWSPropertyValue{}, "System.UInt16"},
		{`{"Id":"x"}`, IWSPropertyValue{}, "System.UInt16"},
		{`{"Id":null}`, IWSPropertyValue{}, "Error converting value {null} to type 'System.UInt16'."},
		{`{"Id":[]}`, IWSPropertyValue{}, "Cannot deserialize the current JSON array"},
		{`{"SubId":255}`, IWSPropertyValue{SubID: 255}, ""},
		{`{"SubId":256}`, IWSPropertyValue{}, "Error converting value 256 to type 'System.Byte'."},
		{`{"SubId":-1}`, IWSPropertyValue{}, "System.Byte"},
		{`{"SubId":1.0}`, IWSPropertyValue{}, "is not a valid integer"}, // byte uses ReadAsInt32
		{`{"SubId":true}`, IWSPropertyValue{}, "Unexpected character"},
		{`{"SubId":null}`, IWSPropertyValue{}, "Error converting value {null} to type 'System.Byte'."},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"Properties":[` + c.elem + `]}`))
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("%s: error = %v, want containing %q", c.elem, err, c.errSub)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.elem, err)
			continue
		}
		if got := *o.Properties[0]; !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, want %+v", c.elem, got, c.want)
		}
	}
}

func TestDecodePropertyValueObjectTypes(t *testing.T) {
	big1, _ := new(big.Int).SetString("9223372036854775808", 10)
	cases := []struct {
		raw     string
		want    any
		wantStr string
	}{
		{`5`, int64(5), "5"},
		{`-9223372036854775808`, int64(-9223372036854775808), "-9223372036854775808"},
		{`9223372036854775808`, big1, "9223372036854775808"},
		{`1.5`, 1.5, "1.5"},
		{`1e2`, 100.0, "100"},
		{`0.1`, 0.1, "0.1"},
		{`"text"`, "text", "text"},
		{`"2020-01-01T00:00:00"`, "2020-01-01T00:00:00", "2020-01-01T00:00:00"}, // DateParseHandling not replicated
		{`true`, true, "True"},
		{`false`, false, "False"},
		{`null`, nil, ""},
		{`{"a": [1, "x"]}`, json.RawMessage(`{"a":[1,"x"]}`), `{"a":[1,"x"]}`},
		{`[ ]`, json.RawMessage(`[]`), `[]`},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"Properties":[{"Id":1,"SubId":0,"Value":` + c.raw + `}]}`))
		if err != nil {
			t.Errorf("Value %s: %v", c.raw, err)
			continue
		}
		p := o.Properties[0]
		if !reflect.DeepEqual(p.Value, c.want) {
			t.Errorf("Value %s = %#v (%T), want %#v (%T)", c.raw, p.Value, p.Value, c.want, c.want)
		}
		if got := p.ValueString(); got != c.wantStr {
			t.Errorf("Value %s ValueString = %q, want %q", c.raw, got, c.wantStr)
		}
	}
	if _, err := DecodeIWSObject([]byte(`{"Properties":[{"Value":1e400}]}`)); err == nil {
		t.Error("Value 1e400 should fail like .NET Framework double.TryParse")
	}
}

func TestDecodeArticleCreator(t *testing.T) {
	cases := []struct {
		name   string
		elem   string
		want   Article
		errSub string
	}{
		{"members", `{"PartCode":"P1","DetailCode":"D","Quantity":2,"Description":"desc"}`,
			Article{Name: "P1", PartCode: "P1", DetailCode: "D", Quantity: 2, Description: "desc"}, ""},
		{"name set after ctor even if first", `{"Name":"N","PartCode":"P"}`,
			Article{Name: "N", PartCode: "P"}, ""},
		{"ctor-only parameter names", `{"row":"R","desc":"X","partcode":"p","QUANTITY":"4"}`,
			Article{Name: "p", PartCode: "p", DetailCode: "R", Quantity: 4, Description: "X"}, ""},
		{"member overrides ctor param", `{"DetailCode":"member","row":"param","Description":"m","desc":"p"}`,
			Article{DetailCode: "member", Description: "m"}, ""},
		{"duplicate ctor param last wins", `{"PartCode":"a","PartCode":"b"}`,
			Article{Name: "b", PartCode: "b"}, ""},
		{"missing quantity is 0", `{}`, Article{}, ""},
		{"null quantity fails", `{"Quantity":null}`, Article{}, "Error converting value {null} to type 'System.Int32'."},
		{"null name allowed", `{"PartCode":"P","Name":null}`, Article{PartCode: "P"}, ""},
	}
	for _, c := range cases {
		o, err := DecodeIWSObject([]byte(`{"ArticleCollection":[` + c.elem + `]}`))
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("%s: error = %v, want containing %q", c.name, err, c.errSub)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got := *o.ArticleCollection[0]; got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func checkCoercion(t *testing.T, member, raw string, err error, errSub string, got func() any, want any) {
	t.Helper()
	if errSub != "" {
		if err == nil || !strings.Contains(err.Error(), errSub) {
			t.Errorf("%s: %s error = %v, want containing %q", member, raw, err, errSub)
		}
		return
	}
	if err != nil {
		t.Errorf("%s: %s: %v", member, raw, err)
		return
	}
	if g := got(); g != want {
		t.Errorf("%s: %s = %#v, want %#v", member, raw, g, want)
	}
}
