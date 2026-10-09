package api

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCategories(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    []string
		wantErr bool
	}{
		{"strings and scalars", 200, `{"version":1,"categories":["generic"," generic2 ","\"scn\"",5,null,true,1.50]}`,
			[]string{"generic", "generic2", "scn", "5", "True", "1.50"}, false},
		{"string walks characters", 200, `{"categories":"ab"}`, []string{"a", "b"}, false},
		{"no categories key", 200, `{"version":1}`, []string{}, false},
		{"empty body", 200, ``, []string{}, false},
		{"404 is valid and empty", 404, `x`, []string{}, false},
		{"not enumerable", 200, `{"categories":5}`, []string{}, true},
		{"top-level array", 200, `["generic"]`, []string{}, true},
		{"request fails", 500, `{}`, []string{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, ss := newScripted(t, always(tc.status, tc.body))
			got, err := c.Categories(context.Background())
			if !reflect.DeepEqual(got, tc.want) || (err != nil) != tc.wantErr {
				t.Fatalf("got %q err %v", got, err)
			}
			r := ss.requests()[0]
			if r.Method != "GET" || r.Path != "/api/categories" || r.Query != "" {
				t.Fatalf("request %+v", r)
			}
		})
	}
}

func TestFetchPropertiesPagingAndParsing(t *testing.T) {
	pages := map[string]string{
		// "count" precedes "properties": the C# adds the parsed items to it,
		// so the next offset is 0 + 2 + 2 = 4.
		"cat=generic": `{"version":2,"count":2,"properties":[` +
			`{"id":"2059_0","access":1,"type":5,"len":0,"cat":"generic","value":1},` +
			`{"id":"2053_0","access":0,"type":9,"len":32,"cat":"generic","value":"ACE0099"}` +
			`],"offset":0,"total":6}`,
		// "count" follows "properties" and overrides the running count.
		"cat=generic&offset=4": `{"version":2,"properties":[` +
			`{"id":"2062_3","access":0,"type":8,"cat":"generic","value":nan},` +
			`{"id":"12345_1FF","access":0,"type":17,"value":1e3},` +
			`{"id":"zz_0","access":0,"type":5,"value":1},` +
			`{"id":"2070_0","access":0,"type":1,"value":true},` +
			`{"id":"2071_0","access":0,"type":64,"value":"0A,0B"}` +
			`],"offset":4,"total":6,"count":2}`,
	}
	c, ss := newScripted(t, func(n int, r seenRequest) reply {
		if body, ok := pages[r.Query]; ok {
			return reply{status: 200, body: body}
		}
		return reply{status: 400}
	})
	c.Identity = ""
	props, err := c.FetchProperties(context.Background(), "cat=generic")
	if err != nil {
		t.Fatal(err)
	}
	var queries []string
	for _, r := range ss.requests() {
		queries = append(queries, r.Query)
	}
	if !reflect.DeepEqual(queries, []string{"cat=generic", "cat=generic&offset=4"}) {
		t.Fatalf("queries %q", queries)
	}
	want := []Property{
		{ID: 0x2059, Sub: 0, Name: "2059_0", DataType: SDTUnsigned8, Value: "1", ReadOnly: true, Category: "generic"},
		{ID: 0x2053, Sub: 0, Name: "2053_0", DataType: SDTVisibleString, Value: "ACE0099", Category: "generic", MaxLength: 32},
		{ID: 0x2062, Sub: 3, Name: "2062_3", DataType: SDTReal32, Value: "0", Category: "generic"},
		{ID: 0x2345, Sub: 0xFF, Name: "12345_1FF", DataType: SDTReal64, Value: "1000"},
		{ID: 0x2070, Sub: 0, Name: "2070_0", DataType: SDTBoolean, Value: "True"},
		{ID: 0x2071, Sub: 0, Name: "2071_0", DataType: SDTByteArray, Value: "0A,0B"},
	}
	if !reflect.DeepEqual(props, want) {
		t.Fatalf("props:\n got %+v\nwant %+v", props, want)
	}
	if c.Identity != "ACE0099" {
		t.Fatalf("Identity = %q (taken from 8275 when empty)", c.Identity)
	}
}

func TestFetchPropertiesLegacyTopLevel(t *testing.T) {
	c, _ := newScripted(t, always(200, `{"OD_sysFoo":{"id":"2059_0","access":1,"type":5,"value":"7"},"success":true,"total":0}`))
	props, err := c.FetchProperties(context.Background(), "ids=2059_0")
	if err != nil {
		t.Fatal(err)
	}
	if len(props) != 1 || props[0].Name != "OD_sysFoo" || props[0].Value != "7" || props[0].ID != 0x2059 {
		t.Fatalf("props %+v", props)
	}
}

func TestFetchPropertiesAborts(t *testing.T) {
	tests := map[string]string{
		"missing value":       `{"properties":[{"id":"2059_0","type":5,"access":1}]}`,
		"item without id":     `{"properties":[{"type":5,"access":1,"value":1}]}`,
		"properties not list": `{"properties":{"a":1}}`,
		"total not int32":     `{"total":1.5,"properties":[]}`,
		"unknown string key":  `{"message":"x"}`,
		"top-level null":      `null`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			c, _ := newScripted(t, always(200, body))
			if _, err := c.FetchProperties(context.Background(), "cat=x"); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestFetchPropertiesRetries(t *testing.T) {
	// First page: ExecuteWebRequest("prop", ..., 5000, 2).
	c, ss := newScripted(t, always(418, ""))
	if _, err := c.FetchProperties(context.Background(), "cat=x"); err == nil || len(ss.requests()) != 2 {
		t.Fatalf("err %v attempts %d", err, len(ss.requests()))
	}
	// Later pages: default 4 attempts.
	c, ss = newScripted(t, func(n int, r seenRequest) reply {
		if n == 0 {
			return reply{status: 200, body: `{"properties":[{"id":"2059_0","access":1,"type":5,"value":1}],"offset":0,"total":5}`}
		}
		return reply{status: 418}
	})
	props, err := c.FetchProperties(context.Background(), "cat=x")
	if err == nil || len(props) != 1 || len(ss.requests()) != 5 {
		t.Fatalf("err %v props %d attempts %d", err, len(props), len(ss.requests()))
	}
}

func TestUpdateCategories(t *testing.T) {
	ok := `{"properties":[],"total":0}`
	tests := []struct {
		name     string
		limit500 bool
		cats     []string
		want     []string
	}{
		{"all with limit", true, nil, []string{"/api/prop?limit=500"}},
		{"named with limit", true, []string{"scn", "meter1"}, []string{"/api/prop?cat=scn&limit=500", "/api/prop?cat=meter1&limit=500"}},
		{"named", false, []string{"generic", "generic2"}, []string{"/api/prop?cat=generic", "/api/prop?cat=generic2"}},
		{"from categories", false, nil, []string{"/api/categories?", "/api/prop?cat=generic", "/api/prop?cat=states"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, ss := newScripted(t, func(n int, r seenRequest) reply {
				if r.Path == "/api/categories" {
					return reply{status: 200, body: `{"categories":["generic","states"]}`}
				}
				return reply{status: 200, body: ok}
			})
			if _, err := c.UpdateCategories(context.Background(), tc.limit500, tc.cats...); err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range ss.requests() {
				got = append(got, r.Path+"?"+r.Query)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("requests %q", got)
			}
		})
	}
	if !UseLimit500(true, 2, 2) || !UseLimit500(true, 3, 0) || UseLimit500(true, 2, 1) || UseLimit500(false, 5, 0) {
		t.Error("UseLimit500")
	}
}

func TestUpdatePropertiesIDs(t *testing.T) {
	ids := CombinedIDs(3538945, 2122752, 8317)
	if ids != "3600_1,2064_0,20_7D" {
		t.Fatalf("CombinedIDs = %q", ids)
	}
	c, ss := newScripted(t, always(200, `{"properties":[],"total":0}`))
	if _, err := c.UpdateProperties(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if q := ss.requests()[0].Query; q != "ids=3600_1,2064_0,20_7D" {
		t.Fatalf("query %q", q)
	}
	if s := ODIndexes(Property{ID: 0x207D, Sub: 2}, Property{ID: 0x2059}); s != "207D_2,2059_0" {
		t.Fatalf("ODIndexes = %q", s)
	}
}

func TestStorePropertiesContext(t *testing.T) {
	c, ss := newScripted(t, always(200, "{}"))
	props := []Property{
		{Name: "OD_a", ID: 0x2059, DataType: SDTBoolean, Value: "true"},
		{Name: "OD_b", ID: 0x2062, Sub: 3, DataType: SDTReal32, Value: "1,5"},
		{Name: "OD_c", ID: 0x2065, Sub: 1, DataType: SDTUnsigned8, Value: "true"},
		{Name: "OD_d", ID: 0x2070, DataType: SDTByteArray, Value: "a,1ff, 3"},
		{Name: "OD_e", ID: 0x2071, DataType: SDTArray16, Value: "1,ffff"},
		{Name: "OD_f", ID: 0x2072, DataType: SDTVisibleString, Value: "abc"},
		{Name: "OD_g", ID: 0x2073, DataType: SDTInteger32, Value: " -12 "},
		{Name: "OD_h", ID: 0x2074, DataType: SDTReal64, Value: "0.1"},
	}
	if err := c.StorePropertiesContext(context.Background(), props...); err != nil {
		t.Fatal(err)
	}
	want := `{"OD_a":{"id":"2059_0","value":"True"},"OD_b":{"id":"2062_3","value":1.5},` +
		`"OD_c":{"id":"2065_1","value":1},"OD_d":{"id":"2070_0","value":"0A,00,03"},` +
		`"OD_e":{"id":"2071_0","value":"0001,FFFF"},"OD_f":{"id":"2072_0","value":"abc"},` +
		`"OD_g":{"id":"2073_0","value":-12},"OD_h":{"id":"2074_0","value":0.1}}`
	r := ss.requests()[0]
	if r.Method != "POST" || r.Path != "/api/prop" || r.Body != want {
		t.Fatalf("request %+v\nwant body %s", r, want)
	}
}

func TestStorePropertiesContextBatchesAndFailures(t *testing.T) {
	c, ss := newScripted(t, always(200, "{}"))
	var props []Property
	for i := 0; i < 16; i++ {
		props = append(props, Property{Name: "OD_x", ID: uint16(0x2000 + i), DataType: SDTUnsigned16, Value: "1"})
	}
	if err := c.StorePropertiesContext(context.Background(), props...); err != nil {
		t.Fatal(err)
	}
	reqs := ss.requests()
	if len(reqs) != 2 || strings.Count(reqs[0].Body, `"id"`) != 15 || strings.Count(reqs[1].Body, `"id"`) != 1 {
		t.Fatalf("batches: %d", len(reqs))
	}

	c, ss = newScripted(t, always(200, "{}"))
	err := c.StorePropertiesContext(context.Background(), Property{Name: "OD_dom", ID: 0x2913, DataType: SDTDomain, Value: "x"})
	if err == nil || len(ss.requests()) != 0 {
		t.Fatalf("DOMAIN has no value in the C#: err %v requests %d", err, len(ss.requests()))
	}

	c, _ = newScripted(t, always(400, ""))
	err = c.StorePropertiesContext(context.Background(), Property{Name: "OD_x", ID: 1, DataType: SDTUnsigned8, Value: "1"})
	if !errors.Is(err, ErrUnsupportedRequest) {
		t.Fatalf("err %v", err)
	}
}

func TestStoreProperty(t *testing.T) {
	c, ss := newScripted(t, always(200, "{}"))
	p := Property{Name: "OD_alb", ID: 8292, DataType: SDTUnsigned8, Value: "1"}
	stored, err := c.StoreProperty(context.Background(), p, "true")
	if stored || err != nil || len(ss.requests()) != 0 {
		t.Fatalf("\"true\" equals device value 1: stored=%v err=%v", stored, err)
	}
	stored, err = c.StoreProperty(context.Background(), p, "0")
	if !stored || err != nil {
		t.Fatalf("stored=%v err=%v", stored, err)
	}
	if b := ss.requests()[0].Body; b != `{"OD_alb":{"id":"2064_0","value":0}}` {
		t.Fatalf("body %s", b)
	}
	// Arrays compare by reference in the C#: always changed.
	stored, _ = c.StoreProperty(context.Background(), Property{Name: "OD_b", ID: 1, DataType: SDTByteArray, Value: "01"}, "01")
	if !stored {
		t.Fatal("BYTEARRAY is always stored")
	}
	// No SetValue branch: never stored.
	stored, _ = c.StoreProperty(context.Background(), Property{Name: "OD_d", ID: 2, DataType: SDTDomain, Value: "a"}, "b")
	if stored {
		t.Fatal("DOMAIN cannot change")
	}
}

func TestIDHelpers(t *testing.T) {
	if id, sub := SplitCombinedID(3301377); id != 0x3260 || sub != 1 {
		t.Errorf("SplitCombinedID(3301377) = %X %X", id, sub)
	}
	if id, sub := SplitCombinedID(8317); id != 8317 || sub != 0 {
		t.Errorf("SplitCombinedID(8317) = %d %d", id, sub)
	}
	tests := []struct {
		in  string
		id  uint16
		sub byte
		ok  bool
	}{
		{"207D_02", 0x207D, 2, true},
		{"207d_2_x", 0x207D, 2, true},
		{"zz_2", 0, 2, true},
		{"207D", 0, 0, false},
		{"12345_0", 0, 0, false},
		{"2059_100", 0, 0, false},
	}
	for _, tc := range tests {
		id, sub, ok := ParseIDSub(tc.in)
		if id != tc.id || sub != tc.sub || ok != tc.ok {
			t.Errorf("ParseIDSub(%q) = %X %X %v", tc.in, id, sub, ok)
		}
	}
	p := Property{ID: 0x207D, Sub: 2}
	if p.PaddedIDSub() != "207D_02" || p.ODIndex() != "207D_2" || p.IDSub() != "207D_2" {
		t.Errorf("id formats %s %s %s", p.PaddedIDSub(), p.ODIndex(), p.IDSub())
	}
}
