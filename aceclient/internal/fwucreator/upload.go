package fwucreator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"alfen/aceclient/internal/api"
)

// Messages returned by StartUpload (ACENetwork/ICUNetwork/ICULanDevice.cs).
const (
	ErrMsgUploadInProgress  = "There is still a firmware upload in progress"
	ErrMsgCouldNotCommunate = "Couldn't communicate with the device, please reboot the device."
)

// Property ids used by the resource upload (ICULanDevice.StartUpload/SetDate).
const (
	propTimeSyncID  = 0x3600 // 13824: UpdateProperties(3538945u) = "3600_1"; value 3 skips SetDate
	propTimeSyncSub = 1
	propDateTimeID  = 0x2059 // 8281/0: SetDate stores Unix milliseconds here
	propDateTimeSub = 0
)

// Request parameters of StartUpload's ExecuteWebRequest calls.
const (
	statusRequestTimeout = 10000 * time.Millisecond // GetFirmwareUploadStatus: ("firmware", "", null, 10000, 1)
	uploadAttempts       = 3                        // int num = 3; while (num-- > 0)
	// The POST itself is ("firmware", "", fileData, 900000, 1): api.FirmwareUploadTimeout.
)

// propTimeSyncCombined is the uint StartUpload passes to UpdateProperties
// (3538945u = 0x360001 -> "3600_1").
const propTimeSyncCombined uint32 = 3538945

// PropertyLookup returns a property from the caller's cache, standing in for
// ICULanDevice.GetProperty on the in-memory PropertyDictionary.
type PropertyLookup func(id uint16, sub byte) (api.Property, bool)

// ProgressHelper ports ICUNetwork.Helpers.ProgressHelper
// (ACENetwork/ICUNetwork.Helpers/ProgressHelper.cs): the fake progress curve
// StartUpload reports while the HTTP POST is running.
type ProgressHelper struct {
	steps        []float64
	tresholds    []float64
	currentStage int
}

// NewProgressHelper ports the ProgressHelper(isAhp) constructor.
func NewProgressHelper(isAHP bool) *ProgressHelper {
	if isAHP {
		return &ProgressHelper{steps: []float64{0.1, 0.1, 0.25, 0.3}, tresholds: []float64{4.0, 8.0, 97.0, 100.0}}
	}
	return &ProgressHelper{steps: []float64{0.25, 0.5, 0.5}, tresholds: []float64{50.0, 97.0, 100.0}}
}

// GetNextStage ports ProgressHelper.GetNextStage.
func (p *ProgressHelper) GetNextStage() float64 {
	if p.currentStage < len(p.steps)-1 {
		v := p.tresholds[p.currentStage]
		p.currentStage++
		return v
	}
	return 0
}

// GetProgress ports ProgressHelper.GetProgress.
func (p *ProgressHelper) GetProgress(progress float64) float64 {
	num := progress + p.steps[p.currentStage]
	if num >= p.tresholds[p.currentStage] {
		return progress
	}
	return num
}

// UploadOptions carries what StartUpload reads from the ICULanDevice and the
// BackgroundWorker.
type UploadOptions struct {
	// IsAHP is ICULanDevice.isAHP (see IsAHPModel).
	IsAHP bool
	// Progress is BackgroundWorker.ReportProgress; nil means bgw == null
	// (no fake-progress task).
	Progress func(percent int)
	// Property looks up the caller's property cache (ICULanDevice.GetProperty
	// on its PropertyDictionary). SetDate needs 0x2059/0 from it; when nil or
	// not cached, SetDate skips the property store like the C# does for a
	// missing property.
	Property PropertyLookup
	// Relogin stands in for ICULanDevice.Login() after a 401/403 and returns
	// its HTTP status code. Default: c.Authenticate(ctx, c.Creds).
	Relogin func(ctx context.Context) int
	// Now and Sleep are DateTime.UtcNow and Thread.Sleep (for tests).
	Now   func() time.Time
	Sleep func(time.Duration)
	// ProgressInterval is the fake-progress Task.Delay (default 1000 ms).
	ProgressInterval time.Duration
}

func (o *UploadOptions) defaults(c *api.Client) {
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.Sleep == nil {
		o.Sleep = time.Sleep
	}
	if o.ProgressInterval == 0 {
		o.ProgressInterval = time.Second
	}
	if o.Relogin == nil {
		o.Relogin = func(ctx context.Context) int {
			res, _ := c.Authenticate(ctx, c.Creds)
			return res.StatusCode
		}
	}
}

// netToString mirrors .NET object.ToString() on a JavaScriptSerializer value.
func netToString(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "", errors.New("Cannot perform runtime binding on a null reference")
	case string:
		return x, nil
	case bool:
		if x {
			return "True", nil
		}
		return "False", nil
	case json.Number:
		if i, err := strconv.ParseInt(string(x), 10, 64); err == nil {
			return strconv.FormatInt(i, 10), nil
		}
		return string(x), nil
	default:
		return fmt.Sprint(x), nil
	}
}

// netIntTryParse ports int.TryParse(string) (NumberStyles.Integer).
func netIntTryParse(s string) (int, bool) {
	t := strings.Trim(s, "\t\n\v\f\r ")
	n, err := strconv.ParseInt(t, 10, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// orderedKV is one top-level member of a JSON object, in document order
// (JavaScriptSerializer enumerates its Dictionary in insertion order).
type orderedKV struct {
	Key   string
	Value any
}

func decodeOrderedObject(body string) ([]orderedKV, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("Unable to cast object of type '%T' to a dictionary", tok)
	}
	var out []orderedKV
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		out = append(out, orderedKV{Key: kt.(string), Value: v})
	}
	return out, nil
}

// GetFirmwareUploadStatus ports ICULanDevice.GetFirmwareUploadStatus
// (ACENetwork/ICUNetwork/ICULanDevice.cs:2426-2471):
// ExecuteWebRequest("firmware", "", null, 10000, 1, suppressPopups: true)
// and read "uploadInProgress" and "OD_fileFirmwareUpdateStatus".value. ok is
// false whenever the C# returns false; err carries what the C# would throw
// (e.g. a missing "value" key).
//
// Faithful quirk: inProgress is `item.Value.ToString() == "true"`, so only the
// JSON string "true" counts — a JSON boolean true stringifies to "True".
func GetFirmwareUploadStatus(ctx context.Context, c *api.Client) (ok, inProgress bool, status FirmwareUpdateStatus, err error) {
	status = FwNoActiveUpdate
	state, resp, _ := c.ExecuteWebRequest(ctx, "firmware", "", nil, api.ExecOptions{Timeout: statusRequestTimeout, MaxRetries: 1})
	if state != api.ValidResponse {
		return false, false, status, nil
	}
	body := resp.Body
	if strings.HasSuffix(body, ",}") {
		body = strings.ReplaceAll(body, ",}", "}")
	}
	if body == "" {
		return false, false, status, nil
	}
	items, derr := decodeOrderedObject(body)
	if derr != nil {
		return false, false, status, derr
	}
	for _, item := range items {
		if item.Key == "uploadInProgress" {
			s, e := netToString(item.Value)
			if e != nil {
				return false, inProgress, status, e
			}
			inProgress = s == "true"
		}
		if item.Key == "OD_fileFirmwareUpdateStatus" {
			val3, isObj := item.Value.(map[string]any)
			if !isObj {
				return false, inProgress, status, fmt.Errorf("'%T' does not contain a definition for 'Count'", item.Value)
			}
			if len(val3) == 0 {
				return false, inProgress, status, nil
			}
			raw, has := val3["value"]
			if !has {
				return false, inProgress, status, errors.New("The given key was not present in the dictionary.")
			}
			s, e := netToString(raw)
			if e != nil {
				return false, inProgress, status, e
			}
			num, parsed := netIntTryParse(s)
			if !parsed || !FirmwareUpdateStatus(num).IsDefined() {
				return false, inProgress, status, nil
			}
			status = FirmwareUpdateStatus(num)
		}
	}
	return true, inProgress, status, nil
}

// sendCommand ports ICULanDevice.SendCommand (ICULanDevice.cs:2605-2608): POST /api/cmd with the
// hand-built body {"command":"<command>"} (5000 ms, 2 attempts).
func sendCommand(ctx context.Context, c *api.Client, command string) bool {
	state, _, _ := c.ExecuteWebRequest(ctx, "cmd", "", "{\"command\":\""+command+"\"}", api.ExecOptions{Timeout: 5000 * time.Millisecond, MaxRetries: 2})
	return state == api.ValidResponse
}

// sendDatetime ports ICULanDevice.SendDatetime (ICULanDevice.cs:2633-2640): AHP POSTs the quoted
// "yyyy-MM-dd HH:mm:ss" string to /api/datetime (5000 ms, 1 attempt); other
// models SendCommand("date yyyy-MM-dd HH:mm:ss").
func sendDatetime(ctx context.Context, c *api.Client, isAHP bool, dt time.Time) bool {
	ts := dt.Format("2006-01-02 15:04:05")
	if isAHP {
		state, _, _ := c.ExecuteWebRequest(ctx, "datetime", "", "\""+ts+"\"", api.ExecOptions{Timeout: 5000 * time.Millisecond, MaxRetries: 1})
		return state == api.ValidResponse
	}
	return sendCommand(ctx, c, "date "+ts)
}

// unixMillisString is Math.Round((dt - UnixEpoch).TotalMilliseconds) as the
// double ICUProperty.SetValue receives and ToString()s.
func unixMillisString(dt time.Time) string {
	ticks := dt.UTC().Sub(time.Unix(0, 0).UTC()).Nanoseconds() / 100
	ms := math.RoundToEven(float64(ticks) * (1.0 / 10000))
	return strconv.FormatFloat(ms, 'f', -1, 64)
}

// setDate ports ICULanDevice.SetDate(DateTime.UtcNow) (ICULanDevice.cs:2584-2594): when 0x2059/0 is
// known, store the Unix milliseconds in it, then SendDatetime.
func setDate(ctx context.Context, c *api.Client, opt *UploadOptions, dt time.Time) bool {
	if opt.Property != nil {
		if p, ok := opt.Property(propDateTimeID, propDateTimeSub); ok {
			p.Value = unixMillisString(dt)
			_ = c.StorePropertiesContext(ctx, p)
		}
	}
	return sendDatetime(ctx, c, opt.IsAHP, dt)
}

// UploadResource ports ICULanDevice.UploadResource(bgw, resourceData)
// (ACENetwork/ICUNetwork/ICULanDevice.cs:2405-2424), i.e. StartUpload(bgw, data,
// isFirmwareFile: false) (ICULanDevice.cs:2090-2172 for that path):
//
//  1. UpdateProperties("3600_1"); unless 0x3600/1 reads 3, SetDate(UtcNow)
//     (store 2059_0 = Unix ms when cached, then /api/datetime or /api/cmd "date …").
//  2. GetFirmwareUploadStatus; IsUploading, a failed status read or
//     uploadInProgress returns ErrMsgUploadInProgress.
//  3. IsUploading = true; up to 3 ExecuteWebRequest("firmware", "", data,
//     900000, 1) multipart POSTs while a fake-progress task reports
//     ProgressHelper steps every second. 401/403 calls Login(); a login
//     answered 401 sleeps 1 s and retries. Any other non-200 status returns
//     ErrMsgCouldNotCommunate.
//  4. Done: for a non-firmware file there is no completion polling or reboot
//     tracking ("Not a firmware update, so no reboot to track").
//
// Faithful quirks: after a 401/403 whose re-login succeeds, the loop breaks
// WITHOUT re-sending (the C# falls through to `break`), and three 401s with
// failing re-logins also end as success. The returned error's text is what
// the C# stores in LastUploadError and DlgUploadResources shows.
func UploadResource(ctx context.Context, c *api.Client, resourceData []byte, opt UploadOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	opt.defaults(c)
	progressHelper := NewProgressHelper(opt.IsAHP)
	progress := progressHelper.GetProgress(0.9)

	// UpdateProperties(3538945u); GetProperty(13824, 1) == null || GetPropertyInt(13824, 1) != 3.
	timeSync, haveTimeSync := api.Property{}, false
	if props, err := c.UpdateProperties(ctx, api.CombinedIDs(propTimeSyncCombined)); err == nil {
		for _, p := range props {
			if p.ID == propTimeSyncID && p.Sub == propTimeSyncSub {
				timeSync, haveTimeSync = p, true
			}
		}
	}
	if !haveTimeSync && opt.Property != nil {
		timeSync, haveTimeSync = opt.Property(propTimeSyncID, propTimeSyncSub)
	}
	if !haveTimeSync || timeSync.Int(0) != 3 {
		setDate(ctx, c, &opt, opt.Now().UTC())
	}

	statusOK, inProgress, _, err := GetFirmwareUploadStatus(ctx, c)
	if err != nil {
		return err
	}
	if c.IsUploading() || !statusOK || inProgress {
		return errors.New(ErrMsgUploadInProgress)
	}
	c.SetUploading(true)
	defer c.SetUploading(false)

	var stop chan struct{}
	var wg sync.WaitGroup
	if opt.Progress != nil {
		stop = make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.NewTicker(opt.ProgressInterval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					progress = progressHelper.GetProgress(progress)
					opt.Progress(int(progress))
				}
			}
		}()
	}
	stopProgress := func() {
		if stop != nil {
			close(stop)
			wg.Wait()
			stop = nil
		}
	}
	defer stopProgress()

	for num := uploadAttempts; num > 0; num-- {
		_, resp, _ := c.ExecuteWebRequest(ctx, "firmware", "", resourceData, api.ExecOptions{Timeout: api.FirmwareUploadTimeout, MaxRetries: 1})
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			if opt.Relogin(ctx) == http.StatusUnauthorized {
				opt.Sleep(time.Second)
				continue
			}
		} else if resp.StatusCode != http.StatusOK {
			return errors.New(ErrMsgCouldNotCommunate)
		}
		break
	}
	return nil
}
