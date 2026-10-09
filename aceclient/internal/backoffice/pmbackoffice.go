package backoffice

import (
	"fmt"
	"strings"
)

// pmDefs ports ICUPMBackOffice.InitializeProperties
// (ACESettings/ICUSettings/ICUPMBackOffice.cs). The order is the JSON order.
// Note it has no EVDisconnectAction, Language, OfflineNFCAuthorization,
// OnlineNFCAuthorization, PingPongInterval or CentralMeterValueAlignment
// entries: those C# properties exist on ICUPMBackOffice but read 0/false/""
// and their setters are no-ops.
var pmDefs = []propDef{
	{Normal, "Title", "", 0, "Title"},
	{Normal, "OD_gprsAPNname", "", 2162688, "APNName"},
	{Normal, "OD_gprsAPNuser", "", 2162944, "APNUser"},
	{Normal, "OD_gprsAPNpassword", "", 2163200, "APNPassword"},
	{Normal, "OD_gprsSIMpin", "", 2163456, "SimPin"},
	{Normal, "OD_commDNS1_1_value", "", 2128129, "DNS1_1"},
	{Normal, "OD_commDNS1_2_value", "", 2129921, "DNS1_2"},
	{Normal, "OD_commDNS2_1_value", "", 2129409, "DNS2_1"},
	{Normal, "OD_commDNS2_2_value", "", 2129665, "DNS2_2"},
	{Normal, "OD_commBackOfficeURL_serverDomainAndPort", "", 2127873, "BackOfficeURL_Domain"},
	{Normal, "OD_commBackOfficeURL_serverPath", "", 2127874, "BackOfficeURL_Path"},
	{Normal, "OD_commBackOfficeURLwired_serverDomainAndPort", "", 2126081, "BackOfficeURLwired_Domain"},
	{Normal, "OD_commBackOfficeURLwired_serverPath", "", 2126082, "BackOfficeURLwired_Path"},
	{Normal, "OD_commProtocolName", "ocpp/json", 2130176, "ProtocolName"},
	{Normal, "OD_commProtocolVersion", "1.6", 2130432, "ProtocolVersion"},
	{Normal, "OD_mainEVDisconnectTimeout", 10, 2176512, "EVDisconnectTimeout"},
	{Normal, "OD_sysIntensity_auto", true, 2121985, "IntensityAuto"},
	{Normal, "OD_sysIntensity_intensity", 100, 2121986, "IntensityIntensity"},
	{Normal, "OD_sysTimeZoneMinutes", 60, 2125312, "TimezoneMinutes"},
	{Normal, "OD_sysOCPP15SmartCharging_smartChargingType", 0, 2124801, "OCPP15SmartChargingType"},
	{Normal, "OD_commSendStationStatus", true, 2134784, "SendStationStatus"},
	{Normal, "OD_commTransactionMessageAttempts", 0, 2135552, "TransactionMessageAttempts"},
	{Normal, "OD_commTransactionMessageRetryInterval", 60, 2135808, "TransactionMessageRetryInterval"},
}

// PMBackOffice ports ICUSettings.ICUPMBackOffice (part-management backoffice,
// one per customer, merging the GPRS and wired variants of an ICUBackOffice).
type PMBackOffice struct {
	// Properties is m_lstProperties (the Properties property).
	Properties []*PMProperty

	original    []*PMProperty // m_lstOriginalProperties (never null)
	lanEnabled  bool          // m_fLANEnabled (default true)
	origLAN     bool          // m_fOriginalLANEnabled
	gprsEnabled bool          // m_fGPRSEnabled (default true)
	origGPRS    bool          // m_fOriginalGPRSEnabled
	dirty       bool
}

// NewPMBackOffice ports `new ICUPMBackOffice()` (base(fDirty: true)).
func NewPMBackOffice() *PMBackOffice {
	return newPMBackOffice(true)
}

func newPMBackOffice(dirty bool) *PMBackOffice {
	p := &PMBackOffice{lanEnabled: true, gprsEnabled: true, dirty: dirty, original: []*PMProperty{}}
	for _, d := range pmDefs {
		p.addProperty(d.name, d.value, d.number, d.propName)
	}
	return p
}

// addProperty ports ICUPMBackOffice.AddProperty (find-or-create by Name, then
// overwrite every field).
func (p *PMBackOffice) addProperty(name string, value any, number uint32, propName string) {
	q := p.GetProperty(name)
	if q == nil {
		q = &PMProperty{}
		p.Properties = append(p.Properties, q)
	}
	q.Name, q.Value, q.PropertyName, q.PropertyNumber = name, value, propName, number
}

// setProperty ports ICUPMBackOffice.SetProperty (private in the C#).
func (p *PMBackOffice) setProperty(name string, value any) bool {
	q := p.GetProperty(name)
	if q == nil {
		return false
	}
	if v, ok := convertLike(q.Value, value); ok {
		q.Value = v
	}
	return true
}

// GetProperty ports ICUPMBackOffice.GetProperty(string name).
func (p *PMBackOffice) GetProperty(name string) *PMProperty {
	for _, q := range p.Properties {
		if q.Name == name {
			return q
		}
	}
	return nil
}

// GetPropertyByNumber ports ICUPMBackOffice.GetProperty(uint propnumber).
func (p *PMBackOffice) GetPropertyByNumber(number uint32) *PMProperty {
	for _, q := range p.Properties {
		if q.PropertyNumber == number {
			return q
		}
	}
	return nil
}

func (p *PMBackOffice) byPropertyName(propName string) *PMProperty {
	for _, q := range p.Properties {
		if q.PropertyName == propName {
			return q
		}
	}
	return nil
}

func (p *PMBackOffice) getValue(propName string) string {
	if q := p.byPropertyName(propName); q != nil {
		return q.ValueString()
	}
	return ""
}

func (p *PMBackOffice) getValueInt(propName string) int {
	q := p.byPropertyName(propName)
	if q == nil {
		return 0
	}
	n, _ := csToInt32(q.Value)
	return n
}

func (p *PMBackOffice) getBool(propName string) bool {
	q := p.byPropertyName(propName)
	if q == nil {
		return false
	}
	v, _ := csParseBool(q.ValueString())
	return v
}

// setValue ports ICUPMBackOffice.setValue (no conversion; no-op when the
// C# property has no backing entry).
func (p *PMBackOffice) setValue(value any, propName string) {
	if q := p.byPropertyName(propName); q != nil {
		q.Value = value
		p.CheckDirty()
	}
}

// Title ports ICUPMBackOffice.Title.
func (p *PMBackOffice) Title() string { return p.getValue("Title") }

// SetTitle ports the ICUPMBackOffice.Title setter.
func (p *PMBackOffice) SetTitle(v string) { p.setValue(v, "Title") }

// APNName ports ICUPMBackOffice.APNName.
func (p *PMBackOffice) APNName() string { return p.getValue("APNName") }

// SetAPNName ports the ICUPMBackOffice.APNName setter.
func (p *PMBackOffice) SetAPNName(v string) { p.setValue(v, "APNName") }

// APNUser ports ICUPMBackOffice.APNUser.
func (p *PMBackOffice) APNUser() string { return p.getValue("APNUser") }

// SetAPNUser ports the ICUPMBackOffice.APNUser setter.
func (p *PMBackOffice) SetAPNUser(v string) { p.setValue(v, "APNUser") }

// APNPassword ports ICUPMBackOffice.APNPassword.
func (p *PMBackOffice) APNPassword() string { return p.getValue("APNPassword") }

// SetAPNPassword ports the ICUPMBackOffice.APNPassword setter.
func (p *PMBackOffice) SetAPNPassword(v string) { p.setValue(v, "APNPassword") }

// SimPin ports ICUPMBackOffice.SimPin.
func (p *PMBackOffice) SimPin() string { return p.getValue("SimPin") }

// SetSimPin ports the ICUPMBackOffice.SimPin setter.
func (p *PMBackOffice) SetSimPin(v string) { p.setValue(v, "SimPin") }

// DNS1_1 ports ICUPMBackOffice.DNS1_1.
func (p *PMBackOffice) DNS1_1() string { return p.getValue("DNS1_1") }

// SetDNS1_1 ports the ICUPMBackOffice.DNS1_1 setter.
func (p *PMBackOffice) SetDNS1_1(v string) { p.setValue(v, "DNS1_1") }

// DNS1_2 ports ICUPMBackOffice.DNS1_2.
func (p *PMBackOffice) DNS1_2() string { return p.getValue("DNS1_2") }

// SetDNS1_2 ports the ICUPMBackOffice.DNS1_2 setter.
func (p *PMBackOffice) SetDNS1_2(v string) { p.setValue(v, "DNS1_2") }

// DNS2_1 ports ICUPMBackOffice.DNS2_1.
func (p *PMBackOffice) DNS2_1() string { return p.getValue("DNS2_1") }

// SetDNS2_1 ports the ICUPMBackOffice.DNS2_1 setter.
func (p *PMBackOffice) SetDNS2_1(v string) { p.setValue(v, "DNS2_1") }

// DNS2_2 ports ICUPMBackOffice.DNS2_2.
func (p *PMBackOffice) DNS2_2() string { return p.getValue("DNS2_2") }

// SetDNS2_2 ports the ICUPMBackOffice.DNS2_2 setter.
func (p *PMBackOffice) SetDNS2_2(v string) { p.setValue(v, "DNS2_2") }

// BackOfficeURL_Domain ports ICUPMBackOffice.BackOfficeURL_Domain.
func (p *PMBackOffice) BackOfficeURL_Domain() string { return p.getValue("BackOfficeURL_Domain") }

// SetBackOfficeURL_Domain ports the ICUPMBackOffice.BackOfficeURL_Domain setter.
func (p *PMBackOffice) SetBackOfficeURL_Domain(v string) { p.setValue(v, "BackOfficeURL_Domain") }

// BackOfficeURL_Path ports ICUPMBackOffice.BackOfficeURL_Path.
func (p *PMBackOffice) BackOfficeURL_Path() string { return p.getValue("BackOfficeURL_Path") }

// SetBackOfficeURL_Path ports the ICUPMBackOffice.BackOfficeURL_Path setter.
func (p *PMBackOffice) SetBackOfficeURL_Path(v string) { p.setValue(v, "BackOfficeURL_Path") }

// BackOfficeURLwired_Domain ports ICUPMBackOffice.BackOfficeURLwired_Domain.
func (p *PMBackOffice) BackOfficeURLwired_Domain() string {
	return p.getValue("BackOfficeURLwired_Domain")
}

// SetBackOfficeURLwired_Domain ports the ICUPMBackOffice.BackOfficeURLwired_Domain setter.
func (p *PMBackOffice) SetBackOfficeURLwired_Domain(v string) {
	p.setValue(v, "BackOfficeURLwired_Domain")
}

// BackOfficeURLwired_Path ports ICUPMBackOffice.BackOfficeURLwired_Path.
func (p *PMBackOffice) BackOfficeURLwired_Path() string {
	return p.getValue("BackOfficeURLwired_Path")
}

// SetBackOfficeURLwired_Path ports the ICUPMBackOffice.BackOfficeURLwired_Path setter.
func (p *PMBackOffice) SetBackOfficeURLwired_Path(v string) {
	p.setValue(v, "BackOfficeURLwired_Path")
}

// SendStationStatus ports ICUPMBackOffice.SendStationStatus.
func (p *PMBackOffice) SendStationStatus() bool { return p.getBool("SendStationStatus") }

// SetSendStationStatus ports the ICUPMBackOffice.SendStationStatus setter.
func (p *PMBackOffice) SetSendStationStatus(v bool) { p.setValue(v, "SendStationStatus") }

// OfflineNFCAuthorization ports ICUPMBackOffice.OfflineNFCAuthorization
// (no backing entry: always 0).
func (p *PMBackOffice) OfflineNFCAuthorization() int {
	return p.getValueInt("OfflineNFCAuthorization")
}

// SetOfflineNFCAuthorization ports the ICUPMBackOffice.OfflineNFCAuthorization
// setter (a no-op).
func (p *PMBackOffice) SetOfflineNFCAuthorization(v int) { p.setValue(v, "OfflineNFCAuthorization") }

// ProtocolName ports ICUPMBackOffice.ProtocolName.
func (p *PMBackOffice) ProtocolName() string { return p.getValue("ProtocolName") }

// SetProtocolName ports the ICUPMBackOffice.ProtocolName setter.
func (p *PMBackOffice) SetProtocolName(v string) { p.setValue(v, "ProtocolName") }

// ProtocolVersion ports ICUPMBackOffice.ProtocolVersion.
func (p *PMBackOffice) ProtocolVersion() string { return p.getValue("ProtocolVersion") }

// SetProtocolVersion ports the ICUPMBackOffice.ProtocolVersion setter.
func (p *PMBackOffice) SetProtocolVersion(v string) { p.setValue(v, "ProtocolVersion") }

// EVDisconnectTimeout ports ICUPMBackOffice.EVDisconnectTimeout.
func (p *PMBackOffice) EVDisconnectTimeout() int { return p.getValueInt("EVDisconnectTimeout") }

// SetEVDisconnectTimeout ports the ICUPMBackOffice.EVDisconnectTimeout setter.
func (p *PMBackOffice) SetEVDisconnectTimeout(v int) { p.setValue(v, "EVDisconnectTimeout") }

// EVDisconnectAction ports ICUPMBackOffice.EVDisconnectAction (no backing
// entry: always 0).
func (p *PMBackOffice) EVDisconnectAction() int { return p.getValueInt("EVDisconnectAction") }

// SetEVDisconnectAction ports the ICUPMBackOffice.EVDisconnectAction setter (a no-op).
func (p *PMBackOffice) SetEVDisconnectAction(v int) { p.setValue(v, "EVDisconnectAction") }

// IntensityAuto ports ICUPMBackOffice.IntensityAuto.
func (p *PMBackOffice) IntensityAuto() bool { return p.getBool("IntensityAuto") }

// SetIntensityAuto ports the ICUPMBackOffice.IntensityAuto setter.
func (p *PMBackOffice) SetIntensityAuto(v bool) { p.setValue(v, "IntensityAuto") }

// IntensityIntensity ports ICUPMBackOffice.IntensityIntensity.
func (p *PMBackOffice) IntensityIntensity() int { return p.getValueInt("IntensityIntensity") }

// SetIntensityIntensity ports the ICUPMBackOffice.IntensityIntensity setter.
func (p *PMBackOffice) SetIntensityIntensity(v int) { p.setValue(v, "IntensityIntensity") }

// TimezoneMinutes ports ICUPMBackOffice.TimezoneMinutes.
func (p *PMBackOffice) TimezoneMinutes() int { return p.getValueInt("TimezoneMinutes") }

// SetTimezoneMinutes ports the ICUPMBackOffice.TimezoneMinutes setter.
func (p *PMBackOffice) SetTimezoneMinutes(v int) { p.setValue(v, "TimezoneMinutes") }

// Language ports ICUPMBackOffice.Language (no backing entry: always "").
func (p *PMBackOffice) Language() string { return p.getValue("Language") }

// SetLanguage ports the ICUPMBackOffice.Language setter (a no-op).
func (p *PMBackOffice) SetLanguage(v string) { p.setValue(v, "Language") }

// PingPongInterval ports ICUPMBackOffice.PingPongInterval (no backing entry: always 0).
func (p *PMBackOffice) PingPongInterval() int { return p.getValueInt("PingPongInterval") }

// SetPingPongInterval ports the ICUPMBackOffice.PingPongInterval setter (a no-op).
func (p *PMBackOffice) SetPingPongInterval(v int) { p.setValue(v, "PingPongInterval") }

// OnlineNFCAuthorization ports ICUPMBackOffice.OnlineNFCAuthorization (no
// backing entry: always false).
func (p *PMBackOffice) OnlineNFCAuthorization() bool { return p.getBool("OnlineNFCAuthorization") }

// SetOnlineNFCAuthorization ports the ICUPMBackOffice.OnlineNFCAuthorization setter (a no-op).
func (p *PMBackOffice) SetOnlineNFCAuthorization(v bool) { p.setValue(v, "OnlineNFCAuthorization") }

// OCPP15SmartChargingType ports ICUPMBackOffice.OCPP15SmartChargingType.
func (p *PMBackOffice) OCPP15SmartChargingType() int {
	return p.getValueInt("OCPP15SmartChargingType")
}

// SetOCPP15SmartChargingType ports the ICUPMBackOffice.OCPP15SmartChargingType setter.
func (p *PMBackOffice) SetOCPP15SmartChargingType(v int) { p.setValue(v, "OCPP15SmartChargingType") }

// CentralMeterValueAlignment ports ICUPMBackOffice.CentralMeterValueAlignment
// (no backing entry: always 0).
func (p *PMBackOffice) CentralMeterValueAlignment() int {
	return p.getValueInt("CentralMeterValueAlignment")
}

// SetCentralMeterValueAlignment ports the ICUPMBackOffice.CentralMeterValueAlignment
// setter (a no-op).
func (p *PMBackOffice) SetCentralMeterValueAlignment(v int) {
	p.setValue(v, "CentralMeterValueAlignment")
}

// TransactionMessageAttempts ports ICUPMBackOffice.TransactionMessageAttempts.
func (p *PMBackOffice) TransactionMessageAttempts() int {
	return p.getValueInt("TransactionMessageAttempts")
}

// SetTransactionMessageAttempts ports the ICUPMBackOffice.TransactionMessageAttempts setter.
func (p *PMBackOffice) SetTransactionMessageAttempts(v int) {
	p.setValue(v, "TransactionMessageAttempts")
}

// TransactionMessageRetryInterval ports ICUPMBackOffice.TransactionMessageRetryInterval.
func (p *PMBackOffice) TransactionMessageRetryInterval() int {
	return p.getValueInt("TransactionMessageRetryInterval")
}

// SetTransactionMessageRetryInterval ports the
// ICUPMBackOffice.TransactionMessageRetryInterval setter.
func (p *PMBackOffice) SetTransactionMessageRetryInterval(v int) {
	p.setValue(v, "TransactionMessageRetryInterval")
}

// IsLANEnabled ports ICUPMBackOffice.IsLANEnabled.
func (p *PMBackOffice) IsLANEnabled() bool { return p.lanEnabled }

// SetLANEnabled ports the ICUPMBackOffice.IsLANEnabled setter (SetPropertyField).
func (p *PMBackOffice) SetLANEnabled(v bool) {
	if p.lanEnabled != v {
		p.lanEnabled = v
		p.CheckDirty()
	}
}

// IsGPRSEnabled ports ICUPMBackOffice.IsGPRSEnabled.
func (p *PMBackOffice) IsGPRSEnabled() bool { return p.gprsEnabled }

// SetGPRSEnabled ports the ICUPMBackOffice.IsGPRSEnabled setter (SetPropertyField).
func (p *PMBackOffice) SetGPRSEnabled(v bool) {
	if p.gprsEnabled != v {
		p.gprsEnabled = v
		p.CheckDirty()
	}
}

// JSON ports the ICUPMBackOffice.Json property (hand-built, no escaping).
func (p *PMBackOffice) JSON() string {
	var values strings.Builder
	for _, q := range p.Properties {
		addVariable(&values, q.Name, q.Value)
	}
	return fmt.Sprintf("\n\t{\"Title\":\"%s\",\"LANEnabled\":%s,\"GPRSEnabled\":%s,\"Values\":[%s\n\t] }",
		p.Title(), strings.ToLower(csToString(p.lanEnabled)), strings.ToLower(csToString(p.gprsEnabled)), values.String())
}

// Dirty ports ICUBaseObject.Dirty.
func (p *PMBackOffice) Dirty() bool { return p.dirty }

// SetDirty ports the ICUBaseObject.Dirty setter.
func (p *PMBackOffice) SetDirty(v bool) { p.dirty = v }

// CheckDirty ports ICUBaseObject.CheckDirty.
func (p *PMBackOffice) CheckDirty() { p.dirty = p.onCheckDirty() }

// onCheckDirty ports ICUPMBackOffice.OnCheckDirty (matches originals by
// PropertyNumber, then compares the LAN/GPRS flags).
func (p *PMBackOffice) onCheckDirty() bool {
	if p.original == nil {
		return true
	}
	for _, q := range p.Properties {
		var o *PMProperty
		for _, c := range p.original {
			if c.PropertyNumber == q.PropertyNumber {
				o = c
				break
			}
		}
		if o != nil && o.Value != nil && q.ValueString() != o.ValueString() {
			return true
		}
	}
	return p.lanEnabled != p.origLAN || p.gprsEnabled != p.origGPRS
}

// CopyFrom ports ICUPMBackOffice.CopyFrom (allProperties is ignored by the C#).
func (p *PMBackOffice) CopyFrom(other *PMBackOffice, allProperties bool) {
	_ = allProperties
	if other == nil {
		return
	}
	p.Properties = nil
	for _, q := range other.Properties {
		p.addProperty(q.Name, q.Value, q.PropertyNumber, q.PropertyName)
	}
	p.lanEnabled = other.IsLANEnabled()
	p.gprsEnabled = other.IsGPRSEnabled()
	p.CheckDirty()
}

func cloneAll(in []*PMProperty) []*PMProperty {
	out := make([]*PMProperty, len(in))
	for i, q := range in {
		out[i] = q.Clone()
	}
	return out
}

// Commit ports ICUPMBackOffice.Commit.
func (p *PMBackOffice) Commit() {
	p.original = cloneAll(p.Properties)
	p.origLAN = p.lanEnabled
	p.origGPRS = p.gprsEnabled
	p.CheckDirty()
}

// Rollback ports ICUPMBackOffice.Rollback (restores the snapshot; like the C#
// it does not re-evaluate Dirty, and on a never-loaded/committed instance the
// snapshot is empty).
func (p *PMBackOffice) Rollback() {
	p.Properties = cloneAll(p.original)
	p.lanEnabled = p.origLAN
	p.gprsEnabled = p.origGPRS
}

// GetValue ports ICUPMBackOffice.GetValue(uint combinedPropId).
func (p *PMBackOffice) GetValue(combinedPropID uint32) any {
	return p.GetValueSub(uint16(combinedPropID>>8), byte(combinedPropID&0xFF))
}

// GetValueSub ports ICUPMBackOffice.GetValue(ushort usId, byte bSubId).
func (p *PMBackOffice) GetValueSub(id uint16, sub byte) any {
	if q := p.GetPropertyByNumber(uint32(id)<<8 + uint32(sub)); q != nil {
		return q.Value
	}
	return nil
}

// loadFromMap ports ICUPMBackOffice.LoadFromDynamic.
func (p *PMBackOffice) loadFromMap(obj map[string]any) error {
	t, ok := obj["Title"]
	if !ok {
		return fmt.Errorf("pmbackoffice: key %q not present", "Title")
	}
	p.setProperty("Title", t)
	if v, ok := obj["GPRSEnabled"]; ok {
		b, err := csToBool(v)
		if err != nil {
			return fmt.Errorf("pmbackoffice: GPRSEnabled: %w", err)
		}
		p.gprsEnabled = b
	}
	if v, ok := obj["LANEnabled"]; ok {
		b, err := csToBool(v)
		if err != nil {
			return fmt.Errorf("pmbackoffice: LANEnabled: %w", err)
		}
		p.lanEnabled = b
	}
	p.origGPRS = p.gprsEnabled
	p.origLAN = p.lanEnabled
	vals, ok := obj["Values"]
	if !ok {
		return fmt.Errorf("pmbackoffice: key %q not present", "Values")
	}
	if err := eachKeyValue(vals, func(k string, v any) { p.setProperty(k, v) }); err != nil {
		return err
	}
	p.original = cloneAll(p.Properties)
	return nil
}
