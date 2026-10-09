package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ICUConfig.ReadInstallerSettings constructs an ICUBackOffice / ICUPMBackOffice
// for every "Backoffices" / "PMBackOffices" element, and a constructor that
// throws makes the whole read fail. internal/backoffice owns those models;
// this file only ports the parts of their dynamic constructors that can throw
// (so Load accepts and rejects exactly the files the C# does) plus the
// Title / Groups values ICUConfig itself reads.

// boHeader holds the ICUBackOffice properties ICUConfig reads
// (AddBackofficesToFeatures, AddDefaultGroups).
type boHeader struct {
	Title  string // ICUBackOffice.Title
	Groups string // ICUBackOffice.Groups
}

func keyNotFound(key string) error {
	return fmt.Errorf("the given key %q was not present in the dictionary", key)
}

// backOfficeHeader ports new ICUBackOffice(dynamic obj)
// (ACESettings/ICUSettings/ICUBackOffice.cs:652-686, LoadFromDynamic) as far
// as ICUConfig depends on it. It fails exactly where that constructor throws
// and returns the resulting Title and Groups.
//
// LoadFromDynamic reads obj["Title"], obj["TitleNL"], obj["TitleDE"],
// obj["TitleFR"], obj["Groups"] and obj["Values"] unconditionally, so a
// missing key throws KeyNotFoundException. It then enumerates Values, and
// ValuesEx when present, and calls SetProperty(item["Key"], item["Value"]) for
// each entry. These indexer reads and the dynamic SetProperty binding happen
// outside SetProperty's try/catch. SetProperty itself never throws.
//
// Title and Groups are hidden string properties. Their value is therefore
// Convert.ToString(value).Trim() for any JSON type (an array becomes
// "System.Object[]"), and a Values/ValuesEx entry whose Key is "Title" or
// "Groups" overwrites them in file order.
func backOfficeHeader(raw json.RawMessage) (boHeader, error) {
	o, err := decodeObject(raw, "backoffice")
	if err != nil {
		return boHeader{}, err
	}
	var h boHeader
	set := func(name string, value json.RawMessage) {
		switch name {
		case "Title":
			h.Title = strings.TrimSpace(convertToString(value))
		case "Groups":
			h.Groups = strings.TrimSpace(convertToString(value))
		}
	}
	for _, key := range []string{"Title", "TitleNL", "TitleDE", "TitleFR", "Groups"} {
		v, ok := o[key]
		if !ok {
			return boHeader{}, keyNotFound(key)
		}
		set(key, v)
	}
	values, ok := o["Values"]
	if !ok {
		return boHeader{}, keyNotFound("Values")
	}
	if err := eachKeyValue(values, "Values", set); err != nil {
		return boHeader{}, err
	}
	if ex, ok := o["ValuesEx"]; ok { // obj.ContainsKey("ValuesEx")
		if err := eachKeyValue(ex, "ValuesEx", set); err != nil {
			return boHeader{}, err
		}
	}
	return h, nil
}

// checkPMBackOffice ports the throwing parts of new ICUPMBackOffice(dynamic
// obj) (ACESettings/ICUSettings/ICUPMBackOffice.cs:426-430, LoadFromDynamic
// :568-584). obj["Title"] and obj["Values"] are read unconditionally.
// GPRSEnabled and LANEnabled are optional, but when present they go through
// Convert.ToBoolean, which can throw. Values entries follow the same rules as
// in ICUBackOffice.
func checkPMBackOffice(raw json.RawMessage) error {
	o, err := decodeObject(raw, "PM backoffice")
	if err != nil {
		return err
	}
	if _, ok := o["Title"]; !ok {
		return keyNotFound("Title")
	}
	for _, key := range []string{"GPRSEnabled", "LANEnabled"} {
		if v, ok := o[key]; ok {
			if _, err := convertToBoolean(v); err != nil {
				return fmt.Errorf("%q: %w", key, err)
			}
		}
	}
	values, ok := o["Values"]
	if !ok {
		return keyNotFound("Values")
	}
	return eachKeyValue(values, "Values", func(string, json.RawMessage) {})
}

// eachKeyValue ports `foreach (dynamic item in obj[what])
// SetProperty(item["Key"], item["Value"])` of both LoadFromDynamic methods.
// Every item must be a dictionary holding "Key" and "Value". The dynamic call
// binds SetProperty(string name, object value), so Key must be a string or
// null; a number, bool, array or object Key is a RuntimeBinderException. A
// null Key matches no property, so set is not called for it.
func eachKeyValue(raw json.RawMessage, what string, set func(key string, value json.RawMessage)) error {
	items, err := foreachItems(raw)
	if err != nil {
		return fmt.Errorf("%q: %w", what, err)
	}
	for i, it := range items {
		where := fmt.Sprintf("%s[%d]", what, i)
		o, err := decodeObject(it, where)
		if err != nil {
			return err
		}
		k, ok := o["Key"]
		if !ok {
			return fmt.Errorf("%s: %w", where, keyNotFound("Key"))
		}
		v, ok := o["Value"]
		if !ok {
			return fmt.Errorf("%s: %w", where, keyNotFound("Value"))
		}
		if isNull(k) {
			continue
		}
		var key string
		if err := json.Unmarshal(k, &key); err != nil {
			return fmt.Errorf("%s: \"Key\" is not a string", where)
		}
		set(key, v)
	}
	return nil
}
