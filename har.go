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
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chaunsin/go-har/messageview"

	"golang.org/x/net/http/httpguts"
	"golang.org/x/sync/semaphore"
)

type (
	ReqHandler    func(req *http.Request) bool
	RespHandler   func(req *http.Response) bool
	EntityHandler func(e *Entry) bool
)

type Handler struct {
	har         *Har
	mu          sync.Mutex
	entries     map[string]*Entry
	transport   http.RoundTripper
	log         Logger
	cookie      bool
	comment     string
	reqBody     []ReqHandler
	respBody    []RespHandler
	concurrency atomic.Int64 // default runtime.NumCPU()
	// reqHandler  []EntityHandler
	// respHandler []EntityHandler
}

func Parse(path string, opts ...Option) (*Handler, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewReader(bytes.NewReader(data), opts...)
}

func NewReader(r io.Reader, opts ...Option) (*Handler, error) {
	var (
		har    Har
		decode = json.NewDecoder(r)
	)
	if err := decode.Decode(&har); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	h, err := NewHandler(&har, opts...)
	if err != nil {
		return nil, err
	}
	for _, e := range har.Log.Entries {
		// Since PageRef is an optional field, there will be many duplicates,
		// so we only record records that are not empty
		if e.PageRef != "" {
			h.entries[e.PageRef] = e
		}
	}
	return h, nil
}

func NewHandler(har *Har, opts ...Option) (*Handler, error) {
	var h = &Handler{
		har: &Har{
			Log: &Log{
				Version: "1.2",
				Creator: &Creator{
					Name:    "go-har",
					Version: "0.0.1",
				},
			},
		},
		mu:      sync.Mutex{},
		entries: make(map[string]*Entry),
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second}).DialContext,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
	if har != nil {
		h.har = har
	}
	if err := h.har.Validate(); err != nil {
		return nil, err
	}
	h.setOption(WithRequestBody(true))
	h.setOption(WithResponseBody(true))
	h.setOption(WithCookie(true))
	h.setOption(WithRequestConcurrency(uint64(runtime.NumCPU())))
	h.setOption(WithLogger(NewLogger("text", "info")))
	h.setOption(opts...)
	return h, nil
}

// setOption sets configurable options.
func (h *Handler) setOption(opts ...Option) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, opt := range opts {
		opt(h)
	}
}

// Export Har structure data
// Note: The exported data is a copy object, and modifying the Har does not affect the original value
func (h *Handler) Export() *Har {
	h.mu.Lock()
	defer h.mu.Unlock()
	har := *h.har
	return &har
}

// Reset the Har structure data
func (h *Handler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make(map[string]*Entry)
	h.har = &Har{Log: &Log{
		Version: "1.2",
		Creator: &Creator{
			Name:    "go-har",
			Version: "0.0.1",
		},
	}}
}

// EntryTotal Returns the total number of entries
func (h *Handler) EntryTotal() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.har == nil || h.har.Log == nil {
		return 0
	}
	return int64(len(h.har.Log.Entries))
}

// Write writes the Har data to the writer
func (h *Handler) Write(w io.Writer) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	// SetEscapeHTML(true) ?
	if err := json.NewEncoder(w).Encode(h.har); err != nil {
		return err
	}
	return nil
}

// AddRequest Add an http.Request to Har
func (h *Handler) AddRequest(id string, r *http.Request) error {
	req, err := NewRequest(r, h.isReqBody(r))
	if err != nil {
		return err
	}

	var entry = Entry{
		PageRef:         id,
		StartedDateTime: time.Now().UTC().Format(time.RFC3339Nano),
		Time:            -1,
		Request:         req,
		Response:        &Response{},
		Cache:           &Cache{},
		Timings:         &Timings{},
		Comment:         h.comment,
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.entries[id]; exists {
		return fmt.Errorf("har: duplicate request id: %s", id)
	}
	h.entries[id] = &entry
	h.har.Log.Entries = append(h.har.Log.Entries, &entry)
	return nil
}

// AddResponse Add an http.Response to Har if the data exists,
// overwrite it, otherwise, no operation is performed
func (h *Handler) AddResponse(id string, resp *http.Response) error {
	nr, err := NewResponse(resp, h.isRespBody(resp))
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if e, ok := h.entries[id]; ok {
		nr.Comment = h.comment
		e.Response = nr
		t, _ := ParseISO8601(e.StartedDateTime)
		e.Time = float64(time.Since(t).Microseconds())
		for _, e := range h.har.Log.Entries {
			if e.PageRef != "" && e.PageRef == id {
				e.Response = nr
			}
		}
	}
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Add("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		h.log.Error("ServeHTTP: method not allowed: %s", r.Method)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	h.mu.Lock()
	defer h.mu.Unlock()
	if err := json.NewEncoder(w).Encode(h.har); err != nil {
		h.log.Error("ServeHTTP: encode: %s", err)
	}
}

// SyncExecute concurrent execution http request.
// Note: The order of execution is not guaranteed
func (h *Handler) SyncExecute(ctx context.Context, filter ...RequestOption) (<-chan Receipt, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var entries = make(map[int]*Entry)
	h.mu.Lock()
	for i, entry := range h.har.Log.Entries {
		for _, f := range filter {
			if !f(h, entry) {
				continue
			}
			entries[i] = entry
		}
	}
	h.mu.Unlock()

	if len(entries) <= 0 {
		var receipt = make(chan Receipt, 1)
		close(receipt)
		return receipt, nil
	}

	var (
		receipt     = make(chan Receipt, len(entries))
		concurrency = h.concurrency.Load()
		client      = &http.Client{
			Transport: h.transport,
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}
	)

	// if concurrency <= 0 then no limit
	if concurrency <= 0 {
		concurrency = int64(len(entries))
	}
	var sema = semaphore.NewWeighted(concurrency)

	go func() {
		for i, e := range entries {
			var (
				index = i
				entry = e
			)
			if err := sema.Acquire(ctx, 1); err != nil {
				h.log.Error("go-har: semaphore acquire failed: %s", err)
			}
			go func() {
				defer sema.Release(1)
				// reset Entry.Time time ?
				var body []byte
				response, err := h.run(ctx, client, entry, true)
				if err == nil && response != nil {
					defer response.Body.Close()
					body, err = io.ReadAll(response.Body)
					if err != nil {
						h.log.Error("go-har: read body filed: %s", err)
					} else {
						response.Body = io.NopCloser(bytes.NewReader(body))
					}
				}
				receipt <- Receipt{h: h, index: index, Entry: entry, Response: response, body: body, err: err}
			}()
		}
		if err := sema.Acquire(ctx, concurrency); err != nil {
			h.log.Error("go-har: semaphore wait failed: %s", err)
		}
		close(receipt)
	}()
	return receipt, nil
}

// Execute sequential synchronous http execution
func (h *Handler) Execute(ctx context.Context, filter ...RequestOption) ([]Receipt, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var entries = make(map[int]*Entry)
	h.mu.Lock()
	for index, entry := range h.har.Log.Entries {
		for _, f := range filter {
			if !f(h, entry) {
				continue
			}
			entries[index] = entry
		}
	}
	h.mu.Unlock()

	if len(entries) <= 0 {
		return make([]Receipt, 0), nil
	}

	var (
		receipt = make([]Receipt, 0, len(entries))
		client  = &http.Client{
			Transport: h.transport,
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}
	)

	for index, entry := range entries {
		// reset Entry.Time time ?
		var body []byte
		response, err := h.run(ctx, client, entry, h.cookie)
		if err == nil && response != nil {
			body, err = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				h.log.Error("go-har: read body failed: %s", err)
			} else {
				response.Body = io.NopCloser(bytes.NewReader(body))
			}
		}
		receipt = append(receipt, Receipt{h: h, index: index, Entry: entry, Response: response, body: body, err: err})
	}
	return receipt, nil
}

func (h *Handler) run(ctx context.Context, cli *http.Client, entry *Entry, withCookie bool) (resp *http.Response, err error) {
	defer func() {
		if x := recover(); x != nil {
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf, false)]
			h.log.Error("go-har: execute request failed: %s\n%s", x, buf)
			err = fmt.Errorf("go-har: execute request failed: %s", x)
			return
		}
	}()

	request, err := EntryToRequest(entry, withCookie)
	if err != nil {
		return nil, fmt.Errorf("EntryToRequest: %w", err)
	}
	request.WithContext(ctx)
	resp, err = cli.Do(request)
	return
}

func (h *Handler) isReqBody(r *http.Request) bool {
	for _, f := range h.reqBody {
		if f(r) {
			return true
		}
	}
	return false
}

func (h *Handler) isRespBody(r *http.Response) bool {
	for _, f := range h.respBody {
		if f(r) {
			return true
		}
	}
	return false
}

// NewRequest .
func NewRequest(req *http.Request, withBody bool) (*Request, error) {
	if req == nil {
		return nil, errors.New("go-har: request is empty")
	}
	r := &Request{
		Method:      req.Method,
		URL:         req.URL.String(),
		HTTPVersion: req.Proto,
		Cookies:     cookies(req.Cookies()),
		Headers:     headers(req.Header),
		HeaderSize:  -1,
		BodySize:    req.ContentLength,
		Comment:     "",
	}

	for n, vs := range req.URL.Query() {
		for _, v := range vs {
			r.QueryString = append(r.QueryString, &NVP{
				Name:    n,
				Value:   v,
				Comment: "",
			})
		}
	}

	pd, err := postData(req, withBody)
	if err != nil {
		return nil, err
	}
	r.PostData = pd
	return r, nil
}

// NewResponse .
func NewResponse(res *http.Response, withBody bool) (*Response, error) {
	if res == nil {
		return nil, errors.New("go-har: response is empty")
	}
	r := &Response{
		HTTPVersion: res.Proto,
		Status:      res.StatusCode,
		StatusText:  http.StatusText(res.StatusCode),
		HeadersSize: -1,
		BodySize:    res.ContentLength,
		Headers:     headers(res.Header),
		Cookies:     cookies(res.Cookies()),
		Comment:     "",
		Content: &Content{
			Encoding: "base64",
			MimeType: res.Header.Get("Content-Type"),
			Comment:  "",
		},
	}

	if res.StatusCode >= 300 && res.StatusCode < 400 {
		r.RedirectURL = res.Header.Get("Location")
	}

	if withBody {
		var mv = messageview.New()
		if err := mv.SnapshotResponse(res); err != nil {
			return nil, err
		}

		reader, err := mv.BodyReader(messageview.Decode())
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		r.Content.Text = body
		r.Content.Size = int64(len(body))
	}
	return r, nil
}

// EntryToRequest .
func EntryToRequest(e *Entry, withCookie bool) (*http.Request, error) {
	if e == nil || e.Request == nil {
		return nil, errors.New("go-har: entry or request is empty")
	}

	var req = e.Request
	_url, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}
	for _, v := range req.QueryString {
		_url.Query().Add(v.Name, v.Value)
	}

	var body string
	if req.PostData != nil {
		if len(req.PostData.Params) == 0 {
			body = req.PostData.Text
		} else {
			var form = make(url.Values)
			for _, p := range req.PostData.Params {
				form.Add(p.Name, p.Value)
			}
			body = form.Encode()
		}
	}

	request, err := http.NewRequest(req.Method, _url.String(), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	for _, h := range req.Headers {
		if httpguts.ValidHeaderFieldName(h.Name) &&
			httpguts.ValidHeaderFieldValue(h.Value) &&
			h.Name != "Cookie" {
			request.Header.Add(h.Name, h.Name)
		}
	}

	if !withCookie {
		return request, nil
	}

	for _, c := range req.Cookies {
		var expires time.Time
		if c.Expires != "" {
			expires, _ = time.Parse(c.Expires, time.RFC3339Nano)
		}
		request.AddCookie(&http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			Expires:  expires,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}
	return request, nil
}

func cookies(cs []*http.Cookie) []*Cookie {
	var hcs = make([]*Cookie, 0, len(cs))
	for _, c := range cs {
		var expires string
		if !c.Expires.IsZero() {
			expires = c.Expires.Format(time.RFC3339Nano)
		}
		hcs = append(hcs, &Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			Expires:  expires,
			HTTPOnly: c.HttpOnly,
			Secure:   c.Secure,
			Comment:  "",
		})
	}
	return hcs
}

func headers(header http.Header) []*NVP {
	var hs = make([]*NVP, 0, len(header))
	for n, vs := range header {
		for _, v := range vs {
			hs = append(hs, &NVP{
				Name:  n,
				Value: v,
			})
		}
	}
	return hs
}

func postData(req *http.Request, withBody bool) (*PostData, error) {
	// If the request has No body (no Content-Length and Transfer-Encoding isn't
	// chunked), skip the post-data.
	if req.ContentLength <= 0 && len(req.TransferEncoding) == 0 {
		return nil, nil
	}

	var ct = req.Header.Get("Content-Type")
	mt, ps, err := mime.ParseMediaType(ct)
	if err != nil {
		// log.Printf("go-har: cannot parse Content-Type header %q: %v", ct, err)
		// mt = ct
		return nil, fmt.Errorf("ParseMediaType: %w", err)
	}

	pd := &PostData{
		MimeType: mt,
		Params:   []*PostParam{},
		Comment:  "",
	}

	if !withBody {
		return pd, nil
	}

	var mv = messageview.New()
	if err := mv.SnapshotRequest(req); err != nil {
		return nil, fmt.Errorf("SnapshotRequest: %w", err)
	}
	br, err := mv.BodyReader()
	if err != nil {
		return nil, fmt.Errorf("BodyReader: %w", err)
	}

	switch mt {
	case "multipart/form-data":
		var mpr = multipart.NewReader(br, ps["boundary"])
		for {
			p, err := mpr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("NextPart: %w", err)
			}

			body, err := io.ReadAll(p)
			if err != nil {
				_ = p.Close()
				return nil, fmt.Errorf("ReadAll: %w", err)
			}
			_ = p.Close()

			pd.Params = append(pd.Params, &PostParam{
				Name:        p.FormName(),
				Value:       string(body),
				FileName:    p.FileName(),
				ContentType: p.Header.Get("Content-Type"),
				Comment:     "",
			})
		}
	case "application/x-www-form-urlencoded":
		body, err := io.ReadAll(br)
		if err != nil {
			return nil, fmt.Errorf("ReadAll: %w", err)
		}

		vs, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("ParseQuery: %w", err)
		}

		for n, vs := range vs {
			for _, v := range vs {
				pd.Params = append(pd.Params, &PostParam{
					Name:    n,
					Value:   v,
					Comment: "",
				})
			}
		}
	default:
		body, err := io.ReadAll(br)
		if err != nil {
			return nil, err
		}
		pd.Text = string(body)
	}
	return pd, nil
}

type Receipt struct {
	h        *Handler
	index    int
	Entry    *Entry
	Response *http.Response
	body     []byte
	err      error
}

func (r *Receipt) Error() error {
	return r.err
}

func (r *Receipt) Body() []byte {
	return r.body
}

func (r *Receipt) FillInResponse(withBody ...bool) error {
	if r.Entry == nil {
		return errors.New("go-har: entry is nil")
	}
	if r.err != nil {
		return r.err
	}

	var wb bool
	if withBody == nil || len(withBody) <= 0 {
		wb = r.h.isRespBody(r.Response)
	} else {
		wb = withBody[0]
	}
	if r.Response.Body == nil {
		return errors.New("go-har: response body is nil")
	}
	if wb {
		r.Response.Body = io.NopCloser(bytes.NewReader(r.body))
	}
	resp, err := NewResponse(r.Response, wb)
	if err != nil {
		return err
	}

	r.h.mu.Lock()
	defer r.h.mu.Unlock()
	r.Entry.Response = resp
	r.h.har.Log.Entries[r.index].Response = resp
	return nil
}
