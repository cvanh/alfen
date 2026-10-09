package isah

// This file ports the parts of Newtonsoft.Json (the version in
// msi_work/files3/Newtonsoft.Json.dll) that
// JsonConvert.DeserializeObject<IWSObject>(string) uses with default settings:
//   - Member names match ordinal first, then case-insensitively
//     (JsonPropertyCollection.GetClosestMatchProperty). Unknown members are
//     ignored. Duplicate members are applied in document order.
//   - Primitive coercions per JsonTextReader.ReadAsString, ReadAsInt32 and
//     ReadAsBoolean, and per JsonSerializerInternalReader.EnsureType
//     (Convert.ChangeType) for ushort.
//   - JSON null into int/bool/ushort/byte is an error ("Error converting value
//     {null} ..."). Null into string, object or list gives "" or nil.
//   - A list or object member that already has a value is populated
//     (ObjectCreationHandling.Auto): a duplicate "Properties" key appends.
//   - Read-only members (IsLanguageDefined, DebuggerDisplay) are still read
//     with their type's reader, so a bad value is an error, and the result
//     is discarded.
//   - Article has no default constructor, so it is built through
//     Article(partCode, row, quantity, desc) and the remaining writable
//     members are set after it (CreateObjectUsingCreatorWithParameters).
//   - Empty or white-space input gives nil without an error. Content after
//     the root value is an error.
//
// Not replicated (the Go parser accepts strict JSON only): comments, single
// quotes, unquoted names, hex/octal literals, NaN/Infinity, $ref/$type
// metadata, and DateParseHandling.DateTime, which turns ISO-8601 strings in
// object-typed members (IWSPropertyValue.Value) into DateTime. Here they stay
// strings.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

type jsonKind int

const (
	jNull jsonKind = iota
	jBool
	jNumber
	jString
	jObject
	jArray
)

// jsonNode is a JSON value that keeps the member order and the literal
// text of numbers, as Newtonsoft's streaming reader sees them.
type jsonNode struct {
	kind  jsonKind
	b     bool
	s     string      // string value, or the number literal
	keys  []string    // object member names, in document order
	items []*jsonNode // object member values, or array elements
}

// DecodeIWSObject ports JsonConvert.DeserializeObject<IWSObject>(body) with
// the default settings that IWSConnection.GetObjectDataAsync uses. It returns
// (nil, nil) for empty/white-space input or a JSON null. A UTF-8 BOM is
// removed first, as HttpContent.ReadAsStringAsync does.
func DecodeIWSObject(body []byte) (*IWSObject, error) {
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	root, err := parseJSONTree(body)
	if err != nil || root == nil {
		return nil, err
	}
	switch root.kind {
	case jNull:
		return nil, nil
	case jObject:
		o := &IWSObject{}
		if err := populateIWSObject(o, root); err != nil {
			return nil, err
		}
		return o, nil
	default:
		return nil, fmt.Errorf("Error converting value %s to type 'ICUIWSConnection.IWSObject'.", root.describe())
	}
}

// parseJSONTree parses one JSON value. Empty or white-space input gives nil.
func parseJSONTree(data []byte) (*jsonNode, error) {
	if len(strings.TrimFunc(string(data), unicode.IsSpace)) == 0 {
		return nil, nil // "No JSON content found" is not an error for a nullable type
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	n, err := readJSONNode(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("Additional text encountered after finished reading JSON content.")
	}
	return n, nil
}

func readJSONNode(dec *json.Decoder) (*jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case nil:
		return &jsonNode{kind: jNull}, nil
	case bool:
		return &jsonNode{kind: jBool, b: t}, nil
	case json.Number:
		return &jsonNode{kind: jNumber, s: t.String()}, nil
	case string:
		return &jsonNode{kind: jString, s: t}, nil
	case json.Delim:
		n := &jsonNode{kind: jObject}
		if t == '[' {
			n.kind = jArray
		}
		for dec.More() {
			if n.kind == jObject {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("invalid object key %v", kt)
				}
				n.keys = append(n.keys, key)
			}
			v, err := readJSONNode(dec)
			if err != nil {
				return nil, err
			}
			n.items = append(n.items, v)
		}
		if _, err := dec.Token(); err != nil { // closing delimiter
			return nil, err
		}
		return n, nil
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}

// describe renders a value the way Newtonsoft's error messages do.
func (n *jsonNode) describe() string {
	switch n.kind {
	case jNull:
		return "{null}"
	case jBool:
		if n.b {
			return "True"
		}
		return "False"
	case jNumber, jString:
		return n.s
	case jObject:
		return "{"
	default:
		return "["
	}
}

// compact re-serializes an object/array node (for JObject/JArray values).
func (n *jsonNode) compact() json.RawMessage {
	var b bytes.Buffer
	n.writeCompact(&b)
	return json.RawMessage(b.Bytes())
}

func (n *jsonNode) writeCompact(b *bytes.Buffer) {
	switch n.kind {
	case jNull:
		b.WriteString("null")
	case jBool:
		b.WriteString(strconv.FormatBool(n.b))
	case jNumber:
		b.WriteString(n.s)
	case jString:
		enc, _ := json.Marshal(n.s)
		b.Write(enc)
	case jObject, jArray:
		open, closing := byte('{'), byte('}')
		if n.kind == jArray {
			open, closing = '[', ']'
		}
		b.WriteByte(open)
		for i, it := range n.items {
			if i > 0 {
				b.WriteByte(',')
			}
			if n.kind == jObject {
				enc, _ := json.Marshal(n.keys[i])
				b.Write(enc)
				b.WriteByte(':')
			}
			it.writeCompact(b)
		}
		b.WriteByte(closing)
	}
}

// unexpectedChar is JsonTextReader.CreateUnexpectedCharacterException for a
// value that a typed ReadAsXxx cannot start with.
func unexpectedChar(n *jsonNode) error {
	c := "{"
	switch n.kind {
	case jArray:
		c = "["
	case jBool:
		c = "f"
		if n.b {
			c = "t"
		}
	}
	return fmt.Errorf("Unexpected character encountered while parsing value: %s.", c)
}

func nullConversion(target string) error {
	return fmt.Errorf("Error converting value {null} to type '%s'.", target)
}

// parseFloatLiteral ports double.TryParse(literal, NumberStyles.Float,
// InvariantCulture) on .NET Framework, where an overflow fails.
func parseFloatLiteral(lit string) (float64, error) {
	f, err := strconv.ParseFloat(lit, 64)
	if err != nil || math.IsInf(f, 0) {
		return 0, fmt.Errorf("Input string '%s' is not a valid number.", lit)
	}
	return f, nil
}

// readPlain ports JsonTextReader.Read() for a number/primitive
// (ParseReadNumber with ReadType.Read): an integer literal gives int64, or
// *big.Int on overflow (an error past 380 chars). A literal with '.', 'e' or
// 'E' gives float64. Objects and arrays give their compact JSON text.
func readPlain(n *jsonNode) (any, error) {
	switch n.kind {
	case jNull:
		return nil, nil
	case jBool:
		return n.b, nil
	case jString:
		return n.s, nil
	case jObject, jArray:
		return n.compact(), nil
	}
	lit := n.s
	if !strings.ContainsAny(lit, ".eE") {
		if v, err := strconv.ParseInt(lit, 10, 64); err == nil {
			return v, nil
		}
		if len(lit) > 380 {
			return nil, fmt.Errorf("JSON integer %s is too large to parse.", lit)
		}
		bi, ok := new(big.Int).SetString(lit, 10)
		if !ok {
			return nil, fmt.Errorf("Input string '%s' is not a valid number.", lit)
		}
		return bi, nil
	}
	return parseFloatLiteral(lit)
}

// readAsString ports JsonTextReader.ReadAsString: a number gives its literal
// text (validated as a double), true/false give "true"/"false", and an object
// or array is an error. isNull reports a JSON null.
func readAsString(n *jsonNode) (s string, isNull bool, err error) {
	switch n.kind {
	case jNull:
		return "", true, nil
	case jString:
		return n.s, false, nil
	case jNumber:
		if _, err := parseFloatLiteral(n.s); err != nil {
			return "", false, err
		}
		return n.s, false, nil
	case jBool:
		return strconv.FormatBool(n.b), false, nil
	}
	return "", false, unexpectedChar(n)
}

// readAsInt32 ports JsonTextReader.ReadAsInt32: integer literals only, with an
// Int32 range check. A string is parsed with Int32.TryParse(Integer,
// Invariant), and "" counts as null.
func readAsInt32(n *jsonNode) (v int32, isNull bool, err error) {
	switch n.kind {
	case jNull:
		return 0, true, nil
	case jString:
		if n.s == "" {
			return 0, true, nil
		}
		i, ok, _ := parseDotNetInt(n.s, 32)
		if !ok {
			return 0, false, fmt.Errorf("Could not convert string to integer: %s.", n.s)
		}
		return int32(i), false, nil
	case jNumber:
		if strings.ContainsAny(n.s, ".eE") {
			return 0, false, fmt.Errorf("Input string '%s' is not a valid integer.", n.s)
		}
		i, err := strconv.ParseInt(n.s, 10, 32)
		if err != nil {
			return 0, false, fmt.Errorf("JSON integer %s is too large or small for an Int32.", n.s)
		}
		return int32(i), false, nil
	}
	return 0, false, unexpectedChar(n)
}

// dotNetBoolTryParse ports Boolean.TryParse: "True"/"False" in any case,
// also after trimming white space and NUL characters.
func dotNetBoolTryParse(s string) (bool, bool) {
	try := func(s string) (bool, bool) {
		switch {
		case strings.EqualFold(s, "True"):
			return true, true
		case strings.EqualFold(s, "False"):
			return false, true
		}
		return false, false
	}
	if b, ok := try(s); ok {
		return b, true
	}
	return try(strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == 0 }))
}

// readAsBoolean ports JsonTextReader.ReadAsBoolean: a number is true when it is
// not zero, a string goes through Boolean.TryParse ("" counts as null), and an
// object or array is an error.
func readAsBoolean(n *jsonNode) (v bool, isNull bool, err error) {
	switch n.kind {
	case jNull:
		return false, true, nil
	case jBool:
		return n.b, false, nil
	case jString:
		if n.s == "" {
			return false, true, nil
		}
		b, ok := dotNetBoolTryParse(n.s)
		if !ok {
			return false, false, fmt.Errorf("Could not convert string to boolean: %s.", n.s)
		}
		return b, false, nil
	case jNumber:
		x, err := readPlain(n)
		if err != nil {
			return false, false, err
		}
		switch t := x.(type) {
		case int64:
			return t != 0, false, nil
		case *big.Int:
			return t.Sign() != 0, false, nil
		case float64:
			return t != 0, false, nil // Convert.ToBoolean(double); NaN cannot occur here
		}
	}
	return false, false, unexpectedChar(n)
}

func readInt(n *jsonNode) (int, error) {
	v, isNull, err := readAsInt32(n)
	if err != nil {
		return 0, err
	}
	if isNull {
		return 0, nullConversion("System.Int32")
	}
	return int(v), nil
}

func readBool(n *jsonNode) (bool, error) {
	v, isNull, err := readAsBoolean(n)
	if err != nil {
		return false, err
	}
	if isNull {
		return false, nullConversion("System.Boolean")
	}
	return v, nil
}

func readString(n *jsonNode) (string, error) {
	s, _, err := readAsString(n)
	return s, err
}

// readByte ports a byte member: ReadAsInt32, then Convert.ChangeType(int, byte).
func readByte(n *jsonNode) (byte, error) {
	v, isNull, err := readAsInt32(n)
	if err != nil {
		return 0, err
	}
	if isNull {
		return 0, nullConversion("System.Byte")
	}
	if v < 0 || v > math.MaxUint8 {
		return 0, fmt.Errorf("Error converting value %d to type 'System.Byte'.", v)
	}
	return byte(v), nil
}

// readUInt16 ports a ushort member: ushort is not in Newtonsoft's ReadTypeMap,
// so the token is read plainly and converted by Convert.ChangeType(value,
// typeof(ushort), InvariantCulture):
//   - an integer must be in range;
//   - a double is rounded half-to-even (Convert.ToUInt16(double));
//   - a string goes through UInt16.Parse(Integer, Invariant);
//   - a bool gives 1 or 0.
func readUInt16(n *jsonNode) (uint16, error) {
	const target = "System.UInt16"
	if n.kind == jObject || n.kind == jArray {
		return 0, fmt.Errorf("Cannot deserialize the current JSON %s into type '%s'.", map[jsonKind]string{jObject: "object", jArray: "array"}[n.kind], target)
	}
	x, err := readPlain(n)
	if err != nil {
		return 0, err
	}
	fail := func() (uint16, error) {
		return 0, fmt.Errorf("Error converting value %s to type '%s'.", n.describe(), target)
	}
	switch t := x.(type) {
	case nil:
		return 0, nullConversion(target)
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case int64:
		if t < 0 || t > math.MaxUint16 {
			return fail()
		}
		return uint16(t), nil
	case *big.Int:
		return fail()
	case float64:
		if !(t >= -0.5 && t < 65535.5) {
			return fail()
		}
		r := int(t)
		dif := t - float64(r)
		if dif > 0.5 || (dif == 0.5 && r&1 != 0) {
			r++
		}
		return uint16(r), nil
	case string:
		u, ok, _ := parseDotNetUint(t, 16)
		if !ok {
			return fail()
		}
		return uint16(u), nil
	}
	return fail()
}

// matchMember ports GetClosestMatchProperty: ordinal match first, then
// ordinal-ignore-case. names are the C# member names.
func matchMember(key string, names []string) string {
	for _, n := range names {
		if n == key {
			return n
		}
	}
	for _, n := range names {
		if strings.EqualFold(n, key) {
			return n
		}
	}
	return ""
}

var iwsObjectMembers = []string{
	"Version", "ObjectId", "PartCode", "Description", "OrderHeader", "OrderNumber",
	"Backoffice", "BackofficeSettingsVersion", "IsEichrechtOrder", "PublicKeyBaseURL",
	"Language", "IsLanguageDefined", "LoadBalancing", "IsPersonalizedDisplay",
	"LogoFileName", "Logo", "IsPartManagementEnabled", "FeaturesUnlockedText",
	"IsThreePhases", "IsSimNeeded", "IsSSAEnabled", "Communication", "Interface",
	"Properties", "ArticleCollection", "InternalInfo",
}

func populateIWSObject(o *IWSObject, n *jsonNode) error {
	for i, key := range n.keys {
		v := n.items[i]
		var err error
		switch matchMember(key, iwsObjectMembers) {
		case "Version":
			o.Version, err = readInt(v)
		case "ObjectId":
			o.ObjectID, err = readString(v)
		case "PartCode":
			o.PartCode, err = readString(v)
		case "Description":
			o.Description, err = readString(v)
		case "OrderHeader":
			o.OrderHeader, err = readString(v)
		case "OrderNumber":
			o.OrderNumber, err = readString(v)
		case "Backoffice":
			o.Backoffice, err = readString(v)
		case "BackofficeSettingsVersion":
			o.BackofficeSettingsVersion, err = readString(v)
		case "IsEichrechtOrder":
			o.IsEichrechtOrder, err = readBool(v)
		case "PublicKeyBaseURL":
			o.PublicKeyBaseURL, err = readString(v)
		case "Language":
			o.Language, err = readString(v)
		case "IsLanguageDefined": // get-only: read as bool, value discarded
			_, _, err = readAsBoolean(v)
		case "LoadBalancing":
			o.LoadBalancing, err = readInt(v)
		case "IsPersonalizedDisplay":
			o.IsPersonalizedDisplay, err = readBool(v)
		case "LogoFileName":
			o.LogoFileName, err = readString(v)
		case "Logo":
			o.Logo, err = readString(v)
		case "IsPartManagementEnabled":
			o.IsPartManagementEnabled, err = readBool(v)
		case "FeaturesUnlockedText":
			o.FeaturesUnlockedText, err = readString(v)
		case "IsThreePhases":
			o.IsThreePhases, err = readBool(v)
		case "IsSimNeeded":
			o.IsSimNeeded, err = readBool(v)
		case "IsSSAEnabled":
			o.IsSSAEnabled, err = readBool(v)
		case "Communication":
			o.Communication, err = readCommunication(o.Communication, v)
		case "Interface":
			o.Interface, err = readInterface(o.Interface, v)
		case "Properties":
			o.Properties, err = readPropertyList(o.Properties, v)
		case "ArticleCollection":
			o.ArticleCollection, err = readArticleList(o.ArticleCollection, v)
		case "InternalInfo":
			o.InternalInfo, err = readString(v)
		}
		if err != nil {
			return fmt.Errorf("%w Path '%s'.", err, key)
		}
	}
	return nil
}

// objectOrNull handles a class-typed member: JSON null gives nil, an object is
// populated, and anything else is an error.
func objectOrNull(n *jsonNode, target string) (isNull bool, err error) {
	switch n.kind {
	case jNull:
		return true, nil
	case jObject:
		return false, nil
	case jArray:
		return false, fmt.Errorf("Cannot deserialize the current JSON array (e.g. [1,2,3]) into type '%s'.", target)
	}
	return false, fmt.Errorf("Error converting value %s to type '%s'.", n.describe(), target)
}

var communicationMembers = []string{"SmallModule", "LargeModule", "GPRS", "UTP"}

func readCommunication(existing *CSCommunication, n *jsonNode) (*CSCommunication, error) {
	if isNull, err := objectOrNull(n, "ICUIWSConnection.CSCommunication"); isNull || err != nil {
		return nil, err
	}
	c := existing
	if c == nil {
		c = &CSCommunication{}
	}
	for i, key := range n.keys {
		var err error
		switch matchMember(key, communicationMembers) {
		case "SmallModule":
			c.SmallModule, err = readBool(n.items[i])
		case "LargeModule":
			c.LargeModule, err = readBool(n.items[i])
		case "GPRS":
			c.GPRS, err = readBool(n.items[i])
		case "UTP":
			c.UTP, err = readBool(n.items[i])
		}
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

var interfaceMembers = []string{"PlugCharge", "Display", "RFID", "OrangeLed", "HeartbeatLed"}

func readInterface(existing *CSInterface, n *jsonNode) (*CSInterface, error) {
	if isNull, err := objectOrNull(n, "ICUIWSConnection.CSInterface"); isNull || err != nil {
		return nil, err
	}
	c := existing
	if c == nil {
		c = &CSInterface{}
	}
	for i, key := range n.keys {
		var err error
		switch matchMember(key, interfaceMembers) {
		case "PlugCharge":
			c.PlugCharge, err = readBool(n.items[i])
		case "Display":
			c.Display, err = readBool(n.items[i])
		case "RFID":
			c.RFID, err = readBool(n.items[i])
		case "OrangeLed":
			c.OrangeLed, err = readBool(n.items[i])
		case "HeartbeatLed":
			c.HeartbeatLed, err = readBool(n.items[i])
		}
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// arrayOrNull handles a list-typed member: JSON null gives nil, an array is
// read, and anything else is an error.
func arrayOrNull(n *jsonNode, target string) (isNull bool, err error) {
	switch n.kind {
	case jNull:
		return true, nil
	case jArray:
		return false, nil
	case jObject:
		return false, fmt.Errorf("Cannot deserialize the current JSON object (e.g. {\"name\":\"value\"}) into type '%s' because the type requires a JSON array (e.g. [1,2,3]) to deserialize correctly.", target)
	}
	return false, fmt.Errorf("Error converting value %s to type '%s'.", n.describe(), target)
}

var propertyValueMembers = []string{"DebuggerDisplay", "Id", "SubId", "Value"}

func readPropertyList(existing []*IWSPropertyValue, n *jsonNode) ([]*IWSPropertyValue, error) {
	const target = "System.Collections.Generic.List`1[ICUIWSConnection.IWSPropertyValue]"
	if isNull, err := arrayOrNull(n, target); isNull || err != nil {
		return nil, err
	}
	list := existing
	if list == nil {
		list = []*IWSPropertyValue{}
	}
	for idx, el := range n.items {
		if el.kind == jNull {
			list = append(list, nil)
			continue
		}
		if isNull, err := objectOrNull(el, "ICUIWSConnection.IWSPropertyValue"); isNull || err != nil {
			return nil, fmt.Errorf("[%d]: %w", idx, err)
		}
		p := &IWSPropertyValue{}
		for i, key := range el.keys {
			v := el.items[i]
			var err error
			switch matchMember(key, propertyValueMembers) {
			case "DebuggerDisplay": // get-only: read as string, value discarded
				_, _, err = readAsString(v)
			case "Id":
				p.ID, err = readUInt16(v)
			case "SubId":
				p.SubID, err = readByte(v)
			case "Value":
				p.Value, err = readPlain(v)
			}
			if err != nil {
				return nil, fmt.Errorf("[%d].%s: %w", idx, key, err)
			}
		}
		list = append(list, p)
	}
	return list, nil
}

// Article creator parameters as Newtonsoft names them: a parameter that has a
// member of the same name and type takes the member's name (partCode becomes
// PartCode and quantity becomes Quantity), and row and desc keep their names.
var (
	articleCreatorParams = []string{"PartCode", "row", "Quantity", "desc"}
	articleMembers       = []string{"Name", "PartCode", "DetailCode", "Quantity", "Description"}
)

// readArticle ports CreateObjectUsingCreatorWithParameters for Article.
func readArticle(n *jsonNode) (*Article, error) {
	var (
		partCode, row, desc string
		quantity            int
	)
	type pending struct {
		member string
		value  string
	}
	var after []pending
	for i, key := range n.keys {
		v := n.items[i]
		if param := matchMember(key, articleCreatorParams); param != "" {
			var err error
			switch param {
			case "PartCode":
				partCode, err = readString(v)
			case "row":
				row, err = readString(v)
			case "Quantity":
				quantity, err = readInt(v)
			case "desc":
				desc, err = readString(v)
			}
			if err != nil {
				return nil, err
			}
			continue
		}
		switch member := matchMember(key, articleMembers); member {
		case "Name", "DetailCode", "Description":
			s, err := readString(v)
			if err != nil {
				return nil, err
			}
			after = append(after, pending{member, s})
		}
	}
	a := NewArticle(partCode, row, quantity, desc)
	for _, p := range after {
		switch p.member {
		case "Name":
			a.Name = p.value
		case "DetailCode":
			a.DetailCode = p.value
		case "Description":
			a.Description = p.value
		}
	}
	return a, nil
}

func readArticleList(existing []*Article, n *jsonNode) ([]*Article, error) {
	const target = "System.Collections.ObjectModel.Collection`1[ICUIWSConnection.Article]"
	if isNull, err := arrayOrNull(n, target); isNull || err != nil {
		return nil, err
	}
	list := existing
	if list == nil {
		list = []*Article{}
	}
	for idx, el := range n.items {
		if el.kind == jNull {
			list = append(list, nil)
			continue
		}
		if isNull, err := objectOrNull(el, "ICUIWSConnection.Article"); isNull || err != nil {
			return nil, fmt.Errorf("[%d]: %w", idx, err)
		}
		a, err := readArticle(el)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", idx, err)
		}
		list = append(list, a)
	}
	return list, nil
}
