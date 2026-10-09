package api

import (
	"bytes"
	"fmt"
	"net/http"
)

// chunkSize mirrors SendMultipartFormDataContent's `int num = 4096`.
const chunkSize = 4096

// UploadFirmware ports UploadFirmware -> StartUpload -> ExecuteWebRequest(
// "firmware", "", fileData) with SendMultipartFormDataContent.
//
// Over HTTPS the byte array is split into 4096-byte parts, each added as a
// SEPARATE form field named "firmwarefile" with filename "filename" (this is
// what the installer does; the device reassembles the parts in order). Over
// HTTP a single part is sent. The .fwi/.tfw is uploaded VERBATIM — the host
// never decrypts it; the device decrypts internally.
func (c *Client) UploadFirmware(fileData []byte) (Response, error) {
	uri, err := c.buildURI("firmware", "")
	if err != nil {
		return Response{}, err
	}
	return c.execMultipart(uri, fileData)
}

// execMultipart builds the exact multipart body and POSTs it.
func (c *Client) execMultipart(uri string, fileData []byte) (Response, error) {
	boundary := "----aceclientboundary" + randBoundary()
	body := buildFirmwareMultipart(fileData, boundary, c.isHTTPS())

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	return c.do(req)
}

// buildFirmwareMultipart emits the body. .NET's ByteArrayContent carries no
// per-part Content-Type, and MultipartFormDataContent emits the name/filename
// verbatim from the (already-quoted) strings passed in StartUpload, so each
// part header is exactly: Content-Disposition: form-data; name="firmwarefile";
// filename="filename".
func buildFirmwareMultipart(data []byte, boundary string, https bool) []byte {
	var b bytes.Buffer
	writePart := func(chunk []byte) {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString(`Content-Disposition: form-data; name="firmwarefile"; filename="filename"` + "\r\n")
		b.WriteString("\r\n")
		b.Write(chunk)
		b.WriteString("\r\n")
	}
	if https {
		// Mirrors the while(num2 > 0) loop: zero parts for empty input.
		for off := 0; off < len(data); off += chunkSize {
			end := off + chunkSize
			if end > len(data) {
				end = len(data)
			}
			writePart(data[off:end])
		}
	} else {
		writePart(data)
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}
