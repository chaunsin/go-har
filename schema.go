// MIT License
//
// Copyright (c) 2024 chaunsin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

package go_har

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

// Har HTTP Archive (HAR) format
// https://w3c.github.io/web-performance/specs/HAR/Overview.html
// This specification defines an archival format for HTTP transactions that can be
// used by a web browser to export detailed performance data about web pages it loads.
type Har struct {
	Log Log `json:"log"`
}

func (h *Har) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, h)
}

func (h *Har) MarshalJSON() ([]byte, error) {
	return json.Marshal(h)
}

// Log This object represents the root of the exported data.
// This object MUST be present and its name MUST be "log". The object contains the following name/value pairs:
type Log struct {
	// Required. Version number of the format.
	Version string `json:"version"`
	// Required. An object of type creator that contains the name and version
	// information of the log creator application.
	Creator Creator `json:"creator"`
	// [optional]. An object of type browser that contains the name and version
	// information of the user agent.
	Browser Browser `json:"browser,omitempty"`
	// [optional]. An array of objects of type page, each representing one exported
	// (tracked) page. Leave out this field if the application does not support
	// grouping by pages.
	Pages []Page `json:"pages,omitempty"`
	// Required. An array of objects of type entry, each representing one
	// exported (tracked) HTTP request.
	Entries []*Entry `json:"entries"`
	// [optional]. A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Creator This object contains information about the log creator application and contains the following name/value pairs:
type Creator struct {
	// Required. The name of the application that created the log.
	Name string `json:"name"`
	// Required. The version number of the application that created the log.
	Version string `json:"version"`
	// [optional]. A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Browser This object contains information about the browser that
// created the log and contains the following name/value pairs:
type Browser struct {
	// Required. The name of the browser that created the log.
	Name string `json:"name"`
	// Required. The version number of the browser that created the log.
	Version string `json:"version"`
	// [optional]. A comment provided by the user or the browser.
	Comment string `json:"comment,omitempty"`
}

// Page There is one <page> object for every exported web page and
// one <entry> object for every HTTP request. In case when an HTTP
// trace tool isn't able to group requests by a page, the <pages>
// object is empty and individual requests doesn't have a parent page.
type Page struct {
	// Date and time stamp for the beginning of the page load
	// (ISO 8601 YYYY-MM-DDThh:mm:ss.sTZD, e.g. 2009-07-24T19:20:30.45+01:00).
	StartedDateTime string `json:"startedDateTime"`
	// Unique identifier of a page within the . Entries use it to refer the parent page.
	ID string `json:"id"`
	// Page title.
	Title string `json:"title"`
	// Detailed timing info about page load.
	PageTimings PageTimings `json:"pageTimings"`
	// (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// PageTimings describes timings for various events (states) fired during the page load.
// All times are specified in milliseconds. If a time info is not available appropriate field is set to -1.
type PageTimings struct {
	// Content of the page loaded. Number of milliseconds since page load started
	// (page.startedDateTime). Use -1 if the timing does not apply to the current
	// request.
	// Depending on the browser, onContentLoad property represents DOMContentLoad
	// event or document.readyState == interactive.
	OnContentLoad int64 `json:"onContentLoad"`
	// Page is loaded (onLoad event fired). Number of milliseconds since page
	// load started (page.startedDateTime). Use -1 if the timing does not apply
	// to the current request.
	OnLoad int64 `json:"onLoad"`
	// (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment"`
}

// Entry This object represents an array with all exported HTTP requests.
// Sorting entries by startedDateTime (starting from the oldest) is
// preferred way how to export data since it can make importing faster.
// However, the reader application should always make sure the array is sorted (if required for the import)
type Entry struct {
	// [string, unique, optional] Reference to the parent page.
	// Leave out this field if the application does not support grouping by pages.
	PageRef string `json:"pageref,omitempty"`
	// Date and time stamp of the request start
	// (ISO 8601 YYYY-MM-DDThh:mm:ss.sTZD).
	StartedDateTime time.Time `json:"startedDateTime"`
	// Total elapsed time of the request in milliseconds. This is the sum of all
	// timings available in the timings object (i.e. not including -1 values) .
	Time int64 `json:"time"`
	// Detailed info about the request.
	Request *Request `json:"request"`
	// Detailed info about the response.
	Response *Response `json:"response"`
	// Info about cache usage.
	Cache Cache `json:"cache"`
	// Detailed timing info about request/response round trip.
	Timings Timings `json:"timings"`
	// [optional] (new in 1.2) IP address of the server that was connected
	// (result of DNS resolution).
	ServerIPAddress string `json:"serverIPAddress,omitempty"`
	// [optional] (new in 1.2) Unique ID of the parent TCP/IP connection, can be
	// the client port number. Note that a port number doesn't have to be unique
	// identifier in cases where the port is shared for more connections. If the
	// port isn't available for the application, any other unique connection ID
	// can be used instead (e.g. connection index). Leave out this field if the
	// application doesn't support this info.
	Connection string `json:"connection,omitempty"`
	// (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Request contains detailed info about performed request.
type Request struct {
	// Request method (GET, POST, ...).
	Method string `json:"method"`
	// Absolute URL of the request (fragments are not included).
	URL string `json:"url"`
	// Request HTTP Version.
	HTTPVersion string `json:"httpVersion"`
	// List of cookie objects.
	Cookies []Cookie `json:"cookies"`
	// List of header objects.
	Headers []NVP `json:"headers"`
	// List of query parameter objects.
	QueryString []NVP `json:"queryString"`
	// Posted data info.
	PostData *PostData `json:"postData"`
	// Total number of bytes from the start of the HTTP request message until
	// (and including) the double CRLF before the body. Set to -1 if the info
	// is not available.
	HeaderSize int64 `json:"headerSize"`
	// Size of the request body (POST data payload) in bytes. Set to -1 if the
	// info is not available.
	BodySize int64 `json:"bodySize"`
	// (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment"`
}

// Response contains detailed info about the response.
//
// The total response size received can be computed as follows (if both values are available):
// var totalSize = entry.response.headersSize + entry.response.bodySize;
type Response struct {
	// Response status.
	Status int `json:"status"`
	// Response status description.
	StatusText string `json:"statusText"`
	// Response HTTP Version.
	HTTPVersion string `json:"httpVersion"`
	// List of cookie objects.
	Cookies []Cookie `json:"cookies"`
	// List of header objects.
	Headers []NVP `json:"headers"`
	// Details about the response body.
	Content Content `json:"content"`
	// Redirection target URL from the Location response header.
	RedirectURL string `json:"redirectURL"`
	// Total number of bytes from the start of the HTTP response message until
	// (and including) the double CRLF before the body. Set to -1 if the info is
	// not available.
	// The size of received response-headers is computed only from headers that
	// are really received from the server. Additional headers appended by the
	// browser are not included in this number, but they appear in the list of
	// header objects.
	HeadersSize int64 `json:"headersSize"`
	// Size of the received response body in bytes. Set to zero in case of
	// responses coming from the cache (304). Set to -1 if the info is not
	// available.
	BodySize int64 `json:"bodySize"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Cookie contains list of all cookies (used in <request> and <response> objects).
type Cookie struct {
	// The name of the cookie.
	Name string `json:"name"`
	// The cookie value.
	Value string `json:"value"`
	// [optional] The path pertaining to the cookie.
	Path string `json:"path,omitempty"`
	// [optional] The host of the cookie.
	Domain string `json:"domain,omitempty"`
	// [optional] Cookie expiration time.
	// (ISO 8601 YYYY-MM-DDThh:mm:ss.sTZD, e.g. 2009-07-24T19:20:30.123+02:00).
	Expires string `json:"expires,omitempty"`
	// [optional] Set to true if the cookie is HTTP only, false otherwise.
	HTTPOnly bool `json:"httpOnly,omitempty"`
	// [optional] (new in 1.2) True if the cookie was transmitted over ssl, false
	// otherwise.
	Secure bool `json:"secure,omitempty"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment bool `json:"comment,omitempty"`
}

// NVP is simply a name/value pair with a comment
type NVP struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	Comment string `json:"comment,omitempty"`
}

// PostData describes posted data, if any (embedded in <request> object).
type PostData struct {
	//  Mime type of posted data.
	MimeType string `json:"mimeType"`
	//  List of posted parameters (in case of URL encoded parameters).
	Params []PostParam `json:"params"`
	//  Plain text posted data
	Text string `json:"text"`
	// [optional] (new in 1.2) A comment provided by the user or the
	// application.
	Comment string `json:"comment,omitempty"`
}

// pdBinary is the JSON representation of binary PostData.
type pdBinary struct {
	MimeType string `json:"mimeType"`
	// Params is a list of posted parameters (in case of URL encoded parameters).
	Params   []PostParam `json:"params"`
	Text     []byte      `json:"text"`
	Encoding string      `json:"encoding"`
}

// MarshalJSON returns a JSON representation of binary PostData.
func (p *PostData) MarshalJSON() ([]byte, error) {
	if utf8.ValidString(p.Text) {
		type noMethod PostData // avoid infinite recursion
		return json.Marshal((*noMethod)(p))
	}
	return json.Marshal(pdBinary{
		MimeType: p.MimeType,
		Params:   p.Params,
		Text:     []byte(p.Text),
		Encoding: "base64",
	})
}

// UnmarshalJSON populates PostData based on the []byte representation of
// the binary PostData.
func (p *PostData) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) { // conform to json.Unmarshaler spec
		return nil
	}
	var enc struct {
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &enc); err != nil {
		return err
	}
	if enc.Encoding != "base64" {
		type noMethod PostData // avoid infinite recursion
		return json.Unmarshal(data, (*noMethod)(p))
	}
	var pb pdBinary
	if err := json.Unmarshal(data, &pb); err != nil {
		return err
	}
	p.MimeType = pb.MimeType
	p.Params = pb.Params
	p.Text = string(pb.Text)
	return nil
}

// PostParam is a list of posted parameters, if any (embedded in <postData> object).
type PostParam struct {
	// name of a posted parameter.
	Name string `json:"name"`
	// [optional] value of a posted parameter or content of a posted file.
	Value string `json:"value,omitempty"`
	// [optional] name of a posted file.
	FileName string `json:"fileName,omitempty"`
	// [optional] content type of a posted file.
	ContentType string `json:"contentType,omitempty"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Content describes details about response content (embedded in <response> object).
type Content struct {
	// Length of the returned content in bytes. Should be equal to
	// response.bodySize if there is no compression and bigger when the content
	// has been compressed.
	Size int64 `json:"size"`
	// [optional] Number of bytes saved. Leave out this field if the information
	// is not available.
	Compression int64 `json:"compression,omitempty"`
	// MIME type of the response text (value of the Content-Type response
	// header). The charset attribute of the MIME type is included (if
	// available).
	MimeType string `json:"mimeType"`
	// [optional] Response body sent from the server or loaded from the browser
	// cache. This field is populated with textual content only. The text field
	// is either HTTP decoded text or a encoded (e.g. "base64") representation of
	// the response body. Leave out this field if the information is not
	// available.
	Text []byte `json:"text,omitempty"`
	// [optional] (new in 1.2) Encoding used for response text field e.g
	// "base64". Leave out this field if the text field is HTTP decoded
	// (decompressed & unchunked), than trans-coded from its original character
	// set into UTF-8.
	Encoding string `json:"encoding,omitempty"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// For marshaling Content to and from json. This works around the json library's
// default conversion of []byte to base64 encoded string.
type contentJSON struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`

	// Text contains the response body sent from the server or loaded from the
	// browser cache. This field is populated with textual content only. The text
	// field is either HTTP decoded text or a encoded (e.g. "base64")
	// representation of the response body. Leave out this field if the
	// information is not available.
	Text string `json:"text,omitempty"`

	// Encoding used for response text field e.g "base64". Leave out this field
	// if the text field is HTTP decoded (decompressed & unchunked), than
	// trans-coded from its original character set into UTF-8.
	Encoding string `json:"encoding,omitempty"`
}

// MarshalJSON marshals the byte slice into json after encoding based on c.Encoding.
func (c Content) MarshalJSON() ([]byte, error) {
	var txt string
	switch c.Encoding {
	case "base64":
		txt = base64.StdEncoding.EncodeToString(c.Text)
	case "":
		txt = string(c.Text)
	default:
		return nil, fmt.Errorf("unsupported encoding for Content.Text: %s", c.Encoding)
	}

	cj := contentJSON{
		Size:     c.Size,
		MimeType: c.MimeType,
		Text:     txt,
		Encoding: c.Encoding,
	}
	return json.Marshal(cj)
}

// Cache contains info about a request coming from browser cache.
type Cache struct {
	// [optional] State of a cache entry before the request. Leave out this field
	// if the information is not available.
	BeforeRequest CacheObject `json:"beforeRequest,omitempty"`
	// [optional] State of a cache entry after the request. Leave out this field if
	// the information is not available.
	AfterRequest CacheObject `json:"afterRequest,omitempty"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// CacheObject is used by both beforeRequest and afterRequest
type CacheObject struct {
	// [optional] - Expiration time of the cache entry.
	Expires string `json:"expires,omitempty"`
	// The last time the cache entry was opened.
	LastAccess string `json:"lastAccess"`
	// Etag
	ETag string `json:"eTag"`
	// The number of times the cache entry has been opened.
	HitCount int64 `json:"hitCount"`
	// [optional] (new in 1.2) A comment provided by the user or the application.
	Comment string `json:"comment,omitempty"`
}

// Timings describes various phases within request-response round trip.
// All times are specified in milliseconds.
type Timings struct {
	Blocked int64 `json:"blocked,omitempty"`
	// [optional]  Time spent in a queue waiting for a network connection. Use -1
	// if the timing does not apply to the current request.
	DNS int64 `json:"dns,omitempty"`
	// [optional] - DNS resolution time. The time required to resolve a host name.
	// Use -1 if the timing does not apply to the current request.
	Connect int64 `json:"connect,omitempty"`
	// [optional] - Time required to create TCP connection. Use -1 if the timing
	// does not apply to the current request.
	Send int64 `json:"send"`
	// Time required to send HTTP request to the server.
	Wait int64 `json:"wait"`
	// Waiting for a response from the server.
	Receive int64 `json:"receive"`
	// Time required to read entire response from the server (or cache).
	Ssl int64 `json:"ssl,omitempty"`
	// [optional] (new in 1.2) - Time required for SSL/TLS negotiation. If this
	// field is defined then the time is also included in the connect field (to
	// ensure backward compatibility with HAR 1.1). Use -1 if the timing does not
	// apply to the current request.
	Comment string `json:"comment,omitempty"`
	// [optional] (new in 1.2) - A comment provided by the user or the application.
}
