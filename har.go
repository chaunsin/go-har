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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/chaunsin/go-har/messageview"
)

type Handler struct {
	har      *Har
	mu       sync.Mutex
	entries  map[string]*Entry
	comment  string
	reqBody  func(*http.Request) bool
	respBody func(*http.Response) bool
}

func Parse(path string) (*Handler, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewReader(bytes.NewReader(data))
}

func NewReader(r io.Reader) (*Handler, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	h := NewHandler()
	if err := h.har.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	for _, e := range h.har.Log.Entries {
		// Since PageRef is an optional field, there will be many duplicates,
		// so we only record records that are not empty
		if e.PageRef != "" {
			h.entries[e.PageRef] = e
		}
	}
	return h, nil
}

func NewHandler() *Handler {
	h := &Handler{
		har: &Har{},
		mu:  sync.Mutex{},
	}
	h.SetOption(WithRequestBody(true))
	h.SetOption(WithResponseBody(true))
	return h
}

func (h *Handler) HAR() *Har {
	return h.har
}

// SetOption sets configurable options on the logger.
func (h *Handler) SetOption(opts ...HandlerOption) {
	for _, opt := range opts {
		opt(h)
	}
}

func (h *Handler) AddRequest(id string, r *http.Request) error {
	req, err := NewRequest(r, h.reqBody(r))
	if err != nil {
		return err
	}

	var entry = Entry{
		PageRef:         id,
		StartedDateTime: time.Now().UTC(),
		Time:            0,
		Request:         req,
		Response:        &Response{},
		Cache:           Cache{},
		Timings:         Timings{},
		ServerIPAddress: "", // todo:
		Connection:      "", // todo:
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

func (h *Handler) AddResponse(id string, resp *http.Response) error {
	nr, err := NewResponse(resp, h.respBody(resp))
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if e, ok := h.entries[id]; ok {
		nr.Comment = h.comment
		e.Response = nr
		e.Time = time.Since(e.StartedDateTime).Microseconds()
		for _, e := range h.har.Log.Entries {
			if e.PageRef != "" && e.PageRef == id {
				e.Response = nr
			}
		}
	}
	return nil
}

func (h *Handler) Handler() http.Handler {
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Add("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Printf("har.ServeHTTP: method not allowed: %s", r.Method)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(h.har); err != nil {
		log.Printf("Encode:%s", err)
	}
}

// NewRequest . todo: options
func NewRequest(req *http.Request, withBody bool) (*Request, error) {
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
			r.QueryString = append(r.QueryString, NVP{
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

func cookies(cs []*http.Cookie) []Cookie {
	var hcs = make([]Cookie, 0, len(cs))
	for _, c := range cs {
		var expires string
		if !c.Expires.IsZero() {
			expires = c.Expires.Format(time.RFC3339)
		}
		hcs = append(hcs, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			Expires:  expires,
			HTTPOnly: c.HttpOnly,
			Secure:   c.Secure,
			Comment:  false,
		})
	}
	return hcs
}

func headers(header http.Header) []NVP {
	var hs = make([]NVP, 0, len(header))
	for n, vs := range header {
		for _, v := range vs {
			hs = append(hs, NVP{
				Name:  n,
				Value: v,
			})
		}
	}
	return hs
}

func postData(req *http.Request, logBody bool) (*PostData, error) {
	// If the request has no body (no Content-Length and Transfer-Encoding isn't
	// chunked), skip the post data.
	if req.ContentLength <= 0 && len(req.TransferEncoding) == 0 {
		return nil, nil
	}

	ct := req.Header.Get("Content-Type")
	mt, ps, err := mime.ParseMediaType(ct)
	if err != nil {
		log.Printf("har: cannot parse Content-Type header %q: %v", ct, err)
		mt = ct
	}

	pd := &PostData{
		MimeType: mt,
		Params:   []PostParam{},
		Comment:  "",
	}

	if !logBody {
		return pd, nil
	}

	mv := messageview.New()
	if err := mv.SnapshotRequest(req); err != nil {
		return nil, err
	}
	br, err := mv.BodyReader()
	if err != nil {
		return nil, err
	}

	switch mt {
	case "multipart/form-data":
		mpr := multipart.NewReader(br, ps["boundary"])
		for {
			p, err := mpr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			body, err := io.ReadAll(p)
			if err != nil {
				_ = p.Close()
				return nil, err
			}
			_ = p.Close()

			pd.Params = append(pd.Params, PostParam{
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
			return nil, err
		}

		vs, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}

		for n, vs := range vs {
			for _, v := range vs {
				pd.Params = append(pd.Params, PostParam{
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

// NewResponse todo: options
func NewResponse(res *http.Response, withBody bool) (*Response, error) {
	r := &Response{
		HTTPVersion: res.Proto,
		Status:      res.StatusCode,
		StatusText:  http.StatusText(res.StatusCode),
		HeadersSize: -1,
		BodySize:    res.ContentLength,
		Headers:     headers(res.Header),
		Cookies:     cookies(res.Cookies()),
		Comment:     "",
		Content: Content{
			Encoding: "base64",
			MimeType: res.Header.Get("Content-Type"),
			Comment:  "",
		},
	}

	if res.StatusCode >= 300 && res.StatusCode < 400 {
		r.RedirectURL = res.Header.Get("Location")
	}

	if withBody {
		mv := messageview.New()
		if err := mv.SnapshotResponse(res); err != nil {
			return nil, err
		}

		br, err := mv.BodyReader(messageview.Decode())
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(br)
		if err != nil {
			return nil, err
		}

		r.Content.Text = body
		r.Content.Size = int64(len(body))
	}
	return r, nil
}

func ToHTTPRequest(r Request) (*http.Request, error) {
	return nil, nil
}

func ToHTTPResponse(r Response) (*http.Response, error) {
	return nil, nil
}