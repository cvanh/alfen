package isah

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

// Device property IDs (object-dictionary index, sub 0) that the IWS code reads.
const (
	// PropSoftwareVersion (0x100A, "Manufacturer software version") is the
	// source of ICULanDevice.FirmwareVersionNumber.
	PropSoftwareVersion uint16 = 4106
	// PropObjectID (0x2051, sysChargePointSerialNumber, "Object Number") is
	// the objectId that LoadSettingsFromIWS sends, lower-cased and trimmed.
	PropObjectID uint16 = 8273
	// PropProcessorID (0x21A0) is sent as ProcessorId (GetPropertyUInt64(8608, 0, 0)).
	PropProcessorID uint16 = 8608
	// PropLicenseKey (0x21A1, "Feature license key") is the IWS property that
	// PanelInformation stores first, before a reboot.
	PropLicenseKey uint16 = 8609
	// PropFeatureFlags (0x21A2) holds the unlocked Features bitmask.
	PropFeatureFlags uint16 = 8610
)

// CSCommunication ports ICUIWSConnection.CSCommunication
// (ACEISAHConnection/ICUIWSConnection/CSCommunication.cs).
type CSCommunication struct {
	SmallModule bool
	LargeModule bool
	GPRS        bool
	UTP         bool
}

// CSInterface ports ICUIWSConnection.CSInterface
// (ACEISAHConnection/ICUIWSConnection/CSInterface.cs).
type CSInterface struct {
	PlugCharge   bool
	Display      bool
	RFID         bool
	OrangeLed    bool
	HeartbeatLed bool
}

// Article ports ICUIWSConnection.Article
// (ACEISAHConnection/ICUIWSConnection/Article.cs).
type Article struct {
	Name        string
	PartCode    string
	DetailCode  string
	Quantity    int // C# int (Int32)
	Description string
}

// NewArticle ports the only constructor, Article(partCode, row, quantity,
// desc): Name and PartCode both get partCode, and DetailCode gets row.
func NewArticle(partCode, row string, quantity int, desc string) *Article {
	return &Article{
		Name:        partCode,
		PartCode:    partCode,
		DetailCode:  row,
		Quantity:    quantity,
		Description: desc,
	}
}

// IWSPropertyValue ports ICUIWSConnection.IWSPropertyValue
// (ACEISAHConnection/ICUIWSConnection/IWSPropertyValue.cs): one device
// setting (object-dictionary index/subindex) with its factory value.
//
// Value is C# `object`, filled the way Newtonsoft.Json fills an object-typed
// member. The Go types are: nil (JSON null), bool, int64 (JSON integer),
// *big.Int (an integer outside Int64, which Newtonsoft reads as BigInteger),
// float64 (a JSON number with '.', 'e' or 'E'), string, or json.RawMessage
// (compact JSON text for a JObject or JArray).
type IWSPropertyValue struct {
	ID    uint16 // Id (ushort)
	SubID byte   // SubId (byte)
	Value any    // Value (object)
}

// ValueString ports Value.ToString() as the device layer uses it
// (ICUProperty.SetValue converts newValue.ToString() to the property type, and
// StoreProperties writes Value.ToString()). It uses .NET Framework formatting:
// bool gives "True"/"False", and float64 gives Double.ToString() ("G", 15
// significant digits). nil gives "" (C# storeProperty ignores a null value).
// A json.RawMessage gives its compact text (Newtonsoft's JToken.ToString()
// would indent it).
func (p IWSPropertyValue) ValueString() string {
	return dotNetToString(p.Value)
}

// DebuggerDisplay ports IWSPropertyValue.DebuggerDisplay:
// $"{Id:X4}_{SubId:X2}: {Value}".
func (p IWSPropertyValue) DebuggerDisplay() string {
	return fmt.Sprintf("%04X_%02X: %s", p.ID, p.SubID, p.ValueString())
}

// String returns DebuggerDisplay.
func (p IWSPropertyValue) String() string { return p.DebuggerDisplay() }

// IWSObject ports ICUIWSConnection.IWSObject
// (ACEISAHConnection/ICUIWSConnection/IWSObject.cs): the factory/order data
// that IWS returns for one charger object number. C# null strings are "".
// C# ints (Int32) are Go int, range-checked by the decoder.
type IWSObject struct {
	Version                   int
	ObjectID                  string // ObjectId
	PartCode                  string
	Description               string
	OrderHeader               string
	OrderNumber               string
	Backoffice                string
	BackofficeSettingsVersion string
	IsEichrechtOrder          bool
	PublicKeyBaseURL          string
	Language                  string
	LoadBalancing             int
	IsPersonalizedDisplay     bool
	LogoFileName              string
	Logo                      string // base64 image (see LogoData)
	IsPartManagementEnabled   bool
	FeaturesUnlockedText      string
	IsThreePhases             bool
	IsSimNeeded               bool
	IsSSAEnabled              bool
	Communication             *CSCommunication
	Interface                 *CSInterface
	// Properties is List<IWSPropertyValue>. A JSON null element stays a nil
	// entry, as in C#.
	Properties []*IWSPropertyValue
	// ArticleCollection is Collection<Article>. A JSON null element stays a
	// nil entry.
	ArticleCollection []*Article
	InternalInfo      string
}

// IsLanguageDefined ports IWSObject.IsLanguageDefined
// (!string.IsNullOrEmpty(Language)).
func (o *IWSObject) IsLanguageDefined() bool { return o.Language != "" }

// FindProperty ports the lookup that PanelInformation does for the license
// key: objData.Properties.FirstOrDefault(a => a.Id == id). It returns nil if
// there is no match. It skips nil entries, where C# would throw a
// NullReferenceException.
func (o *IWSObject) FindProperty(id uint16) *IWSPropertyValue {
	for _, p := range o.Properties {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

// HasPersonalizedLogo ports the condition that
// PanelInformation.ResetToFactorySettings uses before it uploads the logo:
// objData.Logo != null && objData.Logo.Length > 0 && objData.IsPersonalizedDisplay.
func (o *IWSObject) HasPersonalizedLogo() bool {
	return o.Logo != "" && o.IsPersonalizedDisplay
}

// LogoData ports Convert.FromBase64String(objData.Logo) from
// PanelInformation.ResetToFactorySettings. Like .NET, it ignores the white
// space characters ' ', '\t', '\r' and '\n' and requires correct padding.
func (o *IWSObject) LogoData() ([]byte, error) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n':
			return -1
		}
		return r
	}, o.Logo)
	return base64.StdEncoding.DecodeString(s)
}

// HTTPResponse is the part of System.Net.Http.HttpResponseMessage that the C#
// callers read. Body is the buffered content (HttpClient.GetAsync reads the
// whole body before it returns).
type HTTPResponse struct {
	StatusCode int
	Status     string // reason phrase, e.g. "Request Timeout"
	Header     http.Header
	Body       []byte
}

// IsSuccessStatusCode ports HttpResponseMessage.IsSuccessStatusCode (200..299).
func (r *HTTPResponse) IsSuccessStatusCode() bool {
	return r != nil && r.StatusCode >= 200 && r.StatusCode <= 299
}

// IsahResponse ports ICUIWSConnection.IsahResponse
// (ACEISAHConnection/ICUIWSConnection/IsahResponse.cs).
//
// Err is the exception that GetObjectDataAsync caught and logged ("Failed to
// connect to IWS"). When Err is set, HTTPResponse is the synthesized
// 408 RequestTimeout response, as in C#.
type IsahResponse struct {
	HTTPResponse *HTTPResponse
	IsahObject   *IWSObject
	Err          error
}

// requestTimeoutResponse is new HttpResponseMessage(HttpStatusCode.RequestTimeout).
func requestTimeoutResponse() *HTTPResponse {
	return &HTTPResponse{StatusCode: http.StatusRequestTimeout, Status: "Request Timeout", Header: http.Header{}}
}

// dotNetToString ports object.ToString() for the values that Newtonsoft puts
// in an object-typed member (.NET Framework, invariant formats).
func dotNetToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case int64:
		return strconv.FormatInt(x, 10)
	case *big.Int:
		return x.String()
	case float64:
		return dotNetDoubleToString(x)
	case json.RawMessage:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}

// dotNetDoubleToString ports .NET Framework Double.ToString() (format "G",
// precision 15): scientific notation if the exponent is < -4 or >= 15, the
// exponent is written "E+XX"/"E-XX" with at least two digits, and trailing
// zeros are removed. NaN and ±Infinity use the invariant names. Negative
// zero is "0" on .NET Framework.
func dotNetDoubleToString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	return strings.ToUpper(strconv.FormatFloat(f, 'g', 15, 64))
}
