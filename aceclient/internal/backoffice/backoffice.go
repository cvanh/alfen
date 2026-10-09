package backoffice

import (
	"fmt"
	"strings"
)

// propDef is one AddProperty/AddPropertyEx/AddPropertyHidden call.
type propDef struct {
	typ      PropertyType
	name     string
	value    any
	number   uint32
	propName string
}

// backOfficeDefs ports ICUBackOffice.InitializeProperties
// (ACESettings/ICUSettings/ICUBackOffice.cs). The order is the JSON order.
var backOfficeDefs = []propDef{
	{Hidden, "Title", "", 0, "Title"},
	{Hidden, "TitleNL", "", 0, "TitleNL"},
	{Hidden, "TitleDE", "", 0, "TitleDE"},
	{Hidden, "TitleFR", "", 0, "TitleFR"},
	{Hidden, "Groups", "", 0, "Groups"},
	{Normal, "OD_commConnectMethod", 0, 2127616, "ConnectMethod"},
	{Normal, "OD_gprsAPNname", "", 2162688, "APNName"},
	{Normal, "OD_gprsAPNuser", "", 2162944, "APNUser"},
	{Normal, "OD_gprsAPNpassword", "", 2163200, "APNPassword"},
	{Normal, "OD_commDNS1_1_value", "", 2128129, "DNS1_1"},
	{Normal, "OD_commDNS1_2_value", "", 2129921, "DNS1_2"},
	{Normal, "OD_commDNS2_1_value", "", 2129409, "DNS2_1"},
	{Normal, "OD_commDNS2_2_value", "", 2129665, "DNS2_2"},
	{Normal, "OD_commBackOfficeURL_serverDomainAndPort", "", 2127873, "BackOfficeURL_Domain"},
	{Normal, "OD_commBackOfficeURL_serverPath", "", 2127874, "BackOfficeURL_Path"},
	{Normal, "OD_commBackOfficeURLwired_serverDomainAndPort", "", 2126081, "BackOfficeURLwired_Domain"},
	{Normal, "OD_commBackOfficeURLwired_serverPath", "", 2126082, "BackOfficeURLwired_Path"},
	{Normal, "OD_commSendStationStatus", false, 2134784, "SendStationStatus"},
	{Normal, "OD_mainOfflineNFCAuthorization", 0, 2172672, "OfflineNFCAuthorization"},
	{Normal, "OD_commProtocolName", "", 2130176, "ProtocolName"},
	{Normal, "OD_commProtocolVersion", "", 2130432, "ProtocolVersion"},
	{Normal, "OD_mainEVDisconnectTimeout", 0, 2176512, "EVDisconnectTimeout"},
	{Normal, "OD_mainEVDisconnectAction", 0, 2176768, "EVDisconnectAction"},
	{Extended, "OD_sysIntensity_auto", true, 2121985, "IntensityAuto"},
	{Extended, "OD_sysIntensity_intensity", 100, 2121986, "IntensityIntensity"},
	{Extended, "OD_sysTimeZoneMinutes", 60, 2125312, "TimezoneMinutes"},
	{Extended, "OD_sysLanguage", "nl_NL", 2120960, "Language"},
	{Extended, "OD_commPingPongInterval", 120, 2132480, "PingPongInterval"},
	{Extended, "OD_mainOnlineNFCAuthorization", true, 2178048, "OnlineNFCAuthorization"},
	{Extended, "OD_sysOCPP15SmartCharging_smartChargingType", 0, 2124801, "OCPP15SmartChargingType"},
	{Extended, "OD_gprsSIMpin", "", 2163456, "SimPin"},
	{Extended, "OD_commMeteringAlignment", 1, 2132224, "CentralMeterValueAlignment"},
	{Extended, "OD_commTransactionMessageAttempts", 0, 2135552, "TransactionMessageAttempts"},
	{Extended, "OD_commTransactionMessageRetryInterval", 60, 2135808, "TransactionMessageRetryInterval"},
}

// GetPropertyIds ports ICUBackOffice.GetPropertyIds: the combined
// (propId << 8 | subId) numbers of every non-hidden property, in the C# order.
func GetPropertyIds() []uint32 {
	return []uint32{
		2127616, 2162688, 2162944, 2163200, 2163456, 2128129, 2129921, 2129409, 2129665, 2127873,
		2127874, 2126081, 2126082, 2134784, 2172672, 2130176, 2130432, 2176512, 2176768, 2121985,
		2121986, 2125312, 2120960, 2132480, 2178048, 2124801, 2132224, 2135552, 2135808,
	}
}

// BackOffice ports ICUSettings.ICUBackOffice
// (ACESettings/ICUSettings/ICUBackOffice.cs), including the ICUBaseObject
// dirty tracking it inherits.
type BackOffice struct {
	// Properties is m_lstProperties.
	Properties []*Property
	// OriginalValues is the snapshot taken on load/commit (Normal and
	// Extended properties only — CopyFrom without allProperties).
	OriginalValues *BackOffice

	dirty bool
}

// NewBackOffice ports `new ICUBackOffice()` (base(fDirty: true) +
// InitializeProperties).
func NewBackOffice() *BackOffice {
	return newBackOffice(true)
}

func newBackOffice(dirty bool) *BackOffice {
	b := &BackOffice{dirty: dirty}
	for _, d := range backOfficeDefs {
		b.addPropertyInternal(d.typ, d.name, d.value, d.number, d.propName)
	}
	return b
}

// addPropertyInternal ports ICUBackOffice.AddPropertyInternal: adds only when
// no property with that Name exists yet.
func (b *BackOffice) addPropertyInternal(typ PropertyType, name string, value any, number uint32, propName string) {
	if b.GetProperty(name) != nil {
		return
	}
	b.Properties = append(b.Properties, &Property{
		Type: typ, Name: name, Value: value, PropertyName: propName, PropertyNumber: number,
	})
}

// setProperty ports ICUBackOffice.SetProperty (private in the C#): converts
// value to the type of the current value. Returns false when no property has
// that Name (the key is then ignored).
func (b *BackOffice) setProperty(name string, value any) bool {
	p := b.GetProperty(name)
	if p == nil {
		return false
	}
	if v, ok := convertLike(p.Value, value); ok {
		p.Value = v
	}
	return true
}

// GetProperty ports ICUBackOffice.GetProperty(string name).
func (b *BackOffice) GetProperty(name string) *Property {
	for _, p := range b.Properties {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// GetPropertyByNumber ports ICUBackOffice.GetProperty(uint propnumber).
func (b *BackOffice) GetPropertyByNumber(number uint32) *Property {
	for _, p := range b.Properties {
		if p.PropertyNumber == number {
			return p
		}
	}
	return nil
}

func (b *BackOffice) byPropertyName(propName string) *Property {
	for _, p := range b.Properties {
		if p.PropertyName == propName {
			return p
		}
	}
	return nil
}

// getValue ports ICUBackOffice.getValue: Value.ToString() of the property
// backing propName ("" where the C# returns null).
func (b *BackOffice) getValue(propName string) string {
	if p := b.byPropertyName(propName); p != nil {
		return p.ValueString()
	}
	return ""
}

// getValueInt ports ICUBackOffice.getValueInt (0 when missing).
func (b *BackOffice) getValueInt(propName string) int {
	p := b.byPropertyName(propName)
	if p == nil {
		return 0
	}
	n, _ := csToInt32(p.Value)
	return n
}

// getBool ports Convert.ToBoolean(getValue(..)) used by the bool getters.
func (b *BackOffice) getBool(propName string) bool {
	p := b.byPropertyName(propName)
	if p == nil {
		return false
	}
	v, _ := csParseBool(p.ValueString())
	return v
}

// setValue ports ICUBackOffice.setValue: stores value as-is (no conversion)
// and re-evaluates Dirty.
func (b *BackOffice) setValue(value any, propName string) {
	if p := b.byPropertyName(propName); p != nil {
		p.Value = value
		b.CheckDirty()
	}
}

// Title ports ICUBackOffice.Title.
func (b *BackOffice) Title() string { return b.getValue("Title") }

// SetTitle ports the ICUBackOffice.Title setter.
func (b *BackOffice) SetTitle(v string) { b.setValue(v, "Title") }

// TitleNL ports ICUBackOffice.TitleNL.
func (b *BackOffice) TitleNL() string { return b.getValue("TitleNL") }

// SetTitleNL ports the ICUBackOffice.TitleNL setter.
func (b *BackOffice) SetTitleNL(v string) { b.setValue(v, "TitleNL") }

// TitleDE ports ICUBackOffice.TitleDE.
func (b *BackOffice) TitleDE() string { return b.getValue("TitleDE") }

// SetTitleDE ports the ICUBackOffice.TitleDE setter.
func (b *BackOffice) SetTitleDE(v string) { b.setValue(v, "TitleDE") }

// TitleFR ports ICUBackOffice.TitleFR.
func (b *BackOffice) TitleFR() string { return b.getValue("TitleFR") }

// SetTitleFR ports the ICUBackOffice.TitleFR setter.
func (b *BackOffice) SetTitleFR(v string) { b.setValue(v, "TitleFR") }

// Groups ports ICUBackOffice.Groups ("|"-separated company codes).
func (b *BackOffice) Groups() string { return b.getValue("Groups") }

// SetGroups ports the ICUBackOffice.Groups setter.
func (b *BackOffice) SetGroups(v string) { b.setValue(v, "Groups") }

// ConnectMethod ports ICUBackOffice.ConnectMethod (Convert.ToInt32 of the
// string value; 0 None, 1 Wired, 2 GPRS, 3 Auto).
func (b *BackOffice) ConnectMethod() int {
	n, _ := csToInt32(b.getValue("ConnectMethod"))
	return n
}

// SetConnectMethod ports the ICUBackOffice.ConnectMethod setter.
func (b *BackOffice) SetConnectMethod(v int) { b.setValue(v, "ConnectMethod") }

// APNName ports ICUBackOffice.APNName.
func (b *BackOffice) APNName() string { return b.getValue("APNName") }

// SetAPNName ports the ICUBackOffice.APNName setter.
func (b *BackOffice) SetAPNName(v string) { b.setValue(v, "APNName") }

// APNUser ports ICUBackOffice.APNUser.
func (b *BackOffice) APNUser() string { return b.getValue("APNUser") }

// SetAPNUser ports the ICUBackOffice.APNUser setter.
func (b *BackOffice) SetAPNUser(v string) { b.setValue(v, "APNUser") }

// APNPassword ports ICUBackOffice.APNPassword.
func (b *BackOffice) APNPassword() string { return b.getValue("APNPassword") }

// SetAPNPassword ports the ICUBackOffice.APNPassword setter.
func (b *BackOffice) SetAPNPassword(v string) { b.setValue(v, "APNPassword") }

// SimPin ports ICUBackOffice.SimPin.
func (b *BackOffice) SimPin() string { return b.getValue("SimPin") }

// SetSimPin ports the ICUBackOffice.SimPin setter.
func (b *BackOffice) SetSimPin(v string) { b.setValue(v, "SimPin") }

// DNS1_1 ports ICUBackOffice.DNS1_1.
func (b *BackOffice) DNS1_1() string { return b.getValue("DNS1_1") }

// SetDNS1_1 ports the ICUBackOffice.DNS1_1 setter.
func (b *BackOffice) SetDNS1_1(v string) { b.setValue(v, "DNS1_1") }

// DNS1_2 ports ICUBackOffice.DNS1_2.
func (b *BackOffice) DNS1_2() string { return b.getValue("DNS1_2") }

// SetDNS1_2 ports the ICUBackOffice.DNS1_2 setter.
func (b *BackOffice) SetDNS1_2(v string) { b.setValue(v, "DNS1_2") }

// DNS2_1 ports ICUBackOffice.DNS2_1.
func (b *BackOffice) DNS2_1() string { return b.getValue("DNS2_1") }

// SetDNS2_1 ports the ICUBackOffice.DNS2_1 setter.
func (b *BackOffice) SetDNS2_1(v string) { b.setValue(v, "DNS2_1") }

// DNS2_2 ports ICUBackOffice.DNS2_2.
func (b *BackOffice) DNS2_2() string { return b.getValue("DNS2_2") }

// SetDNS2_2 ports the ICUBackOffice.DNS2_2 setter.
func (b *BackOffice) SetDNS2_2(v string) { b.setValue(v, "DNS2_2") }

// BackOfficeURL_Domain ports ICUBackOffice.BackOfficeURL_Domain (GPRS).
func (b *BackOffice) BackOfficeURL_Domain() string { return b.getValue("BackOfficeURL_Domain") }

// SetBackOfficeURL_Domain ports the ICUBackOffice.BackOfficeURL_Domain setter.
func (b *BackOffice) SetBackOfficeURL_Domain(v string) { b.setValue(v, "BackOfficeURL_Domain") }

// BackOfficeURL_Path ports ICUBackOffice.BackOfficeURL_Path (GPRS).
func (b *BackOffice) BackOfficeURL_Path() string { return b.getValue("BackOfficeURL_Path") }

// SetBackOfficeURL_Path ports the ICUBackOffice.BackOfficeURL_Path setter.
func (b *BackOffice) SetBackOfficeURL_Path(v string) { b.setValue(v, "BackOfficeURL_Path") }

// BackOfficeURLwired_Domain ports ICUBackOffice.BackOfficeURLwired_Domain.
func (b *BackOffice) BackOfficeURLwired_Domain() string {
	return b.getValue("BackOfficeURLwired_Domain")
}

// SetBackOfficeURLwired_Domain ports the ICUBackOffice.BackOfficeURLwired_Domain setter.
func (b *BackOffice) SetBackOfficeURLwired_Domain(v string) {
	b.setValue(v, "BackOfficeURLwired_Domain")
}

// BackOfficeURLwired_Path ports ICUBackOffice.BackOfficeURLwired_Path.
func (b *BackOffice) BackOfficeURLwired_Path() string { return b.getValue("BackOfficeURLwired_Path") }

// SetBackOfficeURLwired_Path ports the ICUBackOffice.BackOfficeURLwired_Path setter.
func (b *BackOffice) SetBackOfficeURLwired_Path(v string) {
	b.setValue(v, "BackOfficeURLwired_Path")
}

// SendStationStatus ports ICUBackOffice.SendStationStatus.
func (b *BackOffice) SendStationStatus() bool { return b.getBool("SendStationStatus") }

// SetSendStationStatus ports the ICUBackOffice.SendStationStatus setter.
func (b *BackOffice) SetSendStationStatus(v bool) { b.setValue(v, "SendStationStatus") }

// OfflineNFCAuthorization ports ICUBackOffice.OfflineNFCAuthorization.
func (b *BackOffice) OfflineNFCAuthorization() int { return b.getValueInt("OfflineNFCAuthorization") }

// SetOfflineNFCAuthorization ports the ICUBackOffice.OfflineNFCAuthorization setter.
func (b *BackOffice) SetOfflineNFCAuthorization(v int) { b.setValue(v, "OfflineNFCAuthorization") }

// ProtocolName ports ICUBackOffice.ProtocolName.
func (b *BackOffice) ProtocolName() string { return b.getValue("ProtocolName") }

// SetProtocolName ports the ICUBackOffice.ProtocolName setter.
func (b *BackOffice) SetProtocolName(v string) { b.setValue(v, "ProtocolName") }

// ProtocolVersion ports ICUBackOffice.ProtocolVersion.
func (b *BackOffice) ProtocolVersion() string { return b.getValue("ProtocolVersion") }

// SetProtocolVersion ports the ICUBackOffice.ProtocolVersion setter.
func (b *BackOffice) SetProtocolVersion(v string) { b.setValue(v, "ProtocolVersion") }

// EVDisconnectTimeout ports ICUBackOffice.EVDisconnectTimeout.
func (b *BackOffice) EVDisconnectTimeout() int { return b.getValueInt("EVDisconnectTimeout") }

// SetEVDisconnectTimeout ports the ICUBackOffice.EVDisconnectTimeout setter.
func (b *BackOffice) SetEVDisconnectTimeout(v int) { b.setValue(v, "EVDisconnectTimeout") }

// EVDisconnectAction ports ICUBackOffice.EVDisconnectAction.
func (b *BackOffice) EVDisconnectAction() int { return b.getValueInt("EVDisconnectAction") }

// SetEVDisconnectAction ports the ICUBackOffice.EVDisconnectAction setter.
func (b *BackOffice) SetEVDisconnectAction(v int) { b.setValue(v, "EVDisconnectAction") }

// IntensityAuto ports ICUBackOffice.IntensityAuto.
func (b *BackOffice) IntensityAuto() bool { return b.getBool("IntensityAuto") }

// SetIntensityAuto ports the ICUBackOffice.IntensityAuto setter.
func (b *BackOffice) SetIntensityAuto(v bool) { b.setValue(v, "IntensityAuto") }

// IntensityIntensity ports ICUBackOffice.IntensityIntensity.
func (b *BackOffice) IntensityIntensity() int { return b.getValueInt("IntensityIntensity") }

// SetIntensityIntensity ports the ICUBackOffice.IntensityIntensity setter.
func (b *BackOffice) SetIntensityIntensity(v int) { b.setValue(v, "IntensityIntensity") }

// TimezoneMinutes ports ICUBackOffice.TimezoneMinutes.
func (b *BackOffice) TimezoneMinutes() int { return b.getValueInt("TimezoneMinutes") }

// SetTimezoneMinutes ports the ICUBackOffice.TimezoneMinutes setter.
func (b *BackOffice) SetTimezoneMinutes(v int) { b.setValue(v, "TimezoneMinutes") }

// Language ports ICUBackOffice.Language.
func (b *BackOffice) Language() string { return b.getValue("Language") }

// SetLanguage ports the ICUBackOffice.Language setter.
func (b *BackOffice) SetLanguage(v string) { b.setValue(v, "Language") }

// PingPongInterval ports ICUBackOffice.PingPongInterval.
func (b *BackOffice) PingPongInterval() int { return b.getValueInt("PingPongInterval") }

// SetPingPongInterval ports the ICUBackOffice.PingPongInterval setter.
func (b *BackOffice) SetPingPongInterval(v int) { b.setValue(v, "PingPongInterval") }

// OnlineNFCAuthorization ports ICUBackOffice.OnlineNFCAuthorization.
func (b *BackOffice) OnlineNFCAuthorization() bool { return b.getBool("OnlineNFCAuthorization") }

// SetOnlineNFCAuthorization ports the ICUBackOffice.OnlineNFCAuthorization setter.
func (b *BackOffice) SetOnlineNFCAuthorization(v bool) { b.setValue(v, "OnlineNFCAuthorization") }

// OCPP15SmartChargingType ports ICUBackOffice.OCPP15SmartChargingType.
func (b *BackOffice) OCPP15SmartChargingType() int { return b.getValueInt("OCPP15SmartChargingType") }

// SetOCPP15SmartChargingType ports the ICUBackOffice.OCPP15SmartChargingType setter.
func (b *BackOffice) SetOCPP15SmartChargingType(v int) { b.setValue(v, "OCPP15SmartChargingType") }

// CentralMeterValueAlignment ports ICUBackOffice.CentralMeterValueAlignment.
func (b *BackOffice) CentralMeterValueAlignment() int {
	return b.getValueInt("CentralMeterValueAlignment")
}

// SetCentralMeterValueAlignment ports the ICUBackOffice.CentralMeterValueAlignment setter.
func (b *BackOffice) SetCentralMeterValueAlignment(v int) {
	b.setValue(v, "CentralMeterValueAlignment")
}

// TransactionMessageAttempts ports ICUBackOffice.TransactionMessageAttempts.
func (b *BackOffice) TransactionMessageAttempts() int {
	return b.getValueInt("TransactionMessageAttempts")
}

// SetTransactionMessageAttempts ports the ICUBackOffice.TransactionMessageAttempts setter.
func (b *BackOffice) SetTransactionMessageAttempts(v int) {
	b.setValue(v, "TransactionMessageAttempts")
}

// TransactionMessageRetryInterval ports ICUBackOffice.TransactionMessageRetryInterval.
func (b *BackOffice) TransactionMessageRetryInterval() int {
	return b.getValueInt("TransactionMessageRetryInterval")
}

// SetTransactionMessageRetryInterval ports the ICUBackOffice.TransactionMessageRetryInterval setter.
func (b *BackOffice) SetTransactionMessageRetryInterval(v int) {
	b.setValue(v, "TransactionMessageRetryInterval")
}

// IsGPRS ports ICUBackOffice.IsGPRS (ConnectMethod >= 2).
func (b *BackOffice) IsGPRS() bool { return b.ConnectMethod() >= 2 }

// IsAuto ports ICUBackOffice.IsAuto (ConnectMethod 1 or 3).
func (b *BackOffice) IsAuto() bool {
	if b.ConnectMethod() != 1 {
		return b.ConnectMethod() == 3
	}
	return true
}

// JSON ports the ICUBackOffice.Json property: the hand-built object exactly as
// ICUConfig.WriteInstallerSettings embeds it (leading "\n\t", no escaping,
// newlines in values replaced by ", ").
func (b *BackOffice) JSON() string {
	var values, valuesEx strings.Builder
	for _, p := range b.Properties {
		if p.Type == Normal {
			addVariable(&values, p.Name, p.Value)
		}
	}
	for _, p := range b.Properties {
		if p.Type == Extended {
			addVariable(&valuesEx, p.Name, p.Value)
		}
	}
	return fmt.Sprintf("\n\t{\"Title\":\"%s\",\"TitleNL\":\"%s\",\"TitleDE\":\"%s\",\"TitleFR\":\"%s\",\"Groups\":\"%s\",\"Values\":[%s\n\t], \"ValuesEx\":[%s\n\t]}",
		b.Title(), b.TitleNL(), b.TitleDE(), b.TitleFR(), b.Groups(), values.String(), valuesEx.String())
}

// addVariable ports ICUBackOffice.AddVariable / ICUPMBackOffice.AddVariable.
func addVariable(sb *strings.Builder, odName string, value any) {
	if sb.Len() > 0 {
		sb.WriteByte(',')
	}
	arg := reNewline.ReplaceAllString(csToString(value), ", ")
	fmt.Fprintf(sb, "\n\t\t{\"Key\":\"%s\",\"Value\":\"%s\"}", odName, arg)
}

// Dirty ports ICUBaseObject.Dirty.
func (b *BackOffice) Dirty() bool { return b.dirty }

// SetDirty ports the ICUBaseObject.Dirty setter.
func (b *BackOffice) SetDirty(v bool) { b.dirty = v }

// CheckDirty ports ICUBaseObject.CheckDirty (Dirty = OnCheckDirty()).
func (b *BackOffice) CheckDirty() { b.dirty = b.onCheckDirty() }

// onCheckDirty ports ICUBackOffice.OnCheckDirty: compares by Name against
// OriginalValues (hidden fields are not in the snapshot, so title edits do not
// make it dirty — as in the C#).
func (b *BackOffice) onCheckDirty() bool {
	if b.OriginalValues == nil {
		return true
	}
	for _, p := range b.Properties {
		o := b.OriginalValues.GetProperty(p.Name)
		if o != nil && o.Value != nil && p.ValueString() != o.ValueString() {
			return true
		}
	}
	return false
}

// CopyFrom ports ICUBackOffice.CopyFrom(other, allProperties). Without
// allProperties only Normal and Extended properties are copied (the Hidden
// title/group fields are dropped, as in the C#).
func (b *BackOffice) CopyFrom(other *BackOffice, allProperties bool) {
	if other == nil {
		return
	}
	b.Properties = nil
	for _, p := range other.Properties {
		switch {
		case allProperties:
			b.addPropertyInternal(p.Type, p.Name, p.Value, p.PropertyNumber, p.PropertyName)
		case p.Type == Normal:
			b.addPropertyInternal(Normal, p.Name, p.Value, p.PropertyNumber, p.PropertyName)
		case p.Type == Extended:
			b.addPropertyInternal(Extended, p.Name, p.Value, p.PropertyNumber, p.PropertyName)
		}
	}
	b.CheckDirty()
}

// snapshot ports the `new ICUBackOffice(); copy.CopyFrom(this); OriginalValues
// = copy` tail of LoadFromDynamic/ParseElement.
func (b *BackOffice) snapshot() {
	o := NewBackOffice()
	o.CopyFrom(b, false)
	b.OriginalValues = o
}

// Commit ports ICUBackOffice.Commit.
func (b *BackOffice) Commit() {
	if b.OriginalValues != nil {
		b.OriginalValues.CopyFrom(b, false)
	}
	b.CheckDirty()
}

// Rollback ports ICUBackOffice.Rollback (CopyFrom(OriginalValues)). Note: as
// in the C#, this drops the Hidden title/group properties.
func (b *BackOffice) Rollback() { b.CopyFrom(b.OriginalValues, false) }

// GetValue ports ICUBackOffice.GetValue(uint combinedPropId).
func (b *BackOffice) GetValue(combinedPropID uint32) any {
	return b.GetValueSub(uint16(combinedPropID>>8), byte(combinedPropID&0xFF))
}

// GetValueSub ports ICUBackOffice.GetValue(ushort usId, byte bSubId).
func (b *BackOffice) GetValueSub(id uint16, sub byte) any {
	if p := b.GetPropertyByNumber(uint32(id)<<8 + uint32(sub)); p != nil {
		return p.Value
	}
	return nil
}

// loadFromMap ports ICUBackOffice.LoadFromDynamic over a JavaScriptSerializer
// dictionary. Missing required keys error out where the C# indexer throws
// KeyNotFoundException (which aborts ICUConfig.ReadInstallerSettings).
func (b *BackOffice) loadFromMap(obj map[string]any) error {
	for _, k := range []string{"Title", "TitleNL", "TitleDE", "TitleFR", "Groups"} {
		v, ok := obj[k]
		if !ok {
			return fmt.Errorf("backoffice: key %q not present", k)
		}
		b.setProperty(k, v)
	}
	vals, ok := obj["Values"]
	if !ok {
		return fmt.Errorf("backoffice: key %q not present", "Values")
	}
	if err := b.loadKeyValues(vals); err != nil {
		return err
	}
	if ex, ok := obj["ValuesEx"]; ok {
		if err := b.loadKeyValues(ex); err != nil {
			return err
		}
	}
	b.snapshot()
	return nil
}

func (b *BackOffice) loadKeyValues(v any) error {
	return eachKeyValue(v, func(key string, value any) { b.setProperty(key, value) })
}

// eachKeyValue iterates a [{"Key":..,"Value":..}] array as the C#
// `foreach (dynamic item in obj["Values"]) SetProperty(item["Key"], item["Value"])`.
func eachKeyValue(v any, fn func(key string, value any)) error {
	arr, ok := v.([]any)
	if !ok {
		return fmt.Errorf("backoffice: Values is %T, not an array", v)
	}
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			return fmt.Errorf("backoffice: Values entry is %T, not an object", it)
		}
		k, ok := m["Key"]
		if !ok {
			return fmt.Errorf("backoffice: Values entry without \"Key\"")
		}
		key, ok := k.(string)
		if !ok {
			return fmt.Errorf("backoffice: Values entry Key is %T, not a string", k)
		}
		val, ok := m["Value"]
		if !ok {
			return fmt.Errorf("backoffice: Values entry %q without \"Value\"", key)
		}
		fn(key, val)
	}
	return nil
}
