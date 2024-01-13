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
	"net/http"
	"strings"
)

// Option is a configurable setting for the logger.
type Option func(h *Handler)

// WithRequestBody returns an option that configures request post data logging.
func WithRequestBody(enabled bool) Option {
	return func(h *Handler) {
		h.reqBody = func(*http.Request) bool {
			return enabled
		}
	}
}

// WithRequestBodyByContentTypes returns an option that logs request bodies based
// on opting in to the Content-Type of the request.
func WithRequestBodyByContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.reqBody = func(req *http.Request) bool {
			rct := req.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return true
				}
			}
			return false
		}
	}
}

// WithSkipRequestBodyForContentTypes returns an option that logs request bodies based
// on opting out of the Content-Type of the request.
func WithSkipRequestBodyForContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.reqBody = func(req *http.Request) bool {
			rct := req.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return false
				}
			}
			return true
		}
	}
}

// WithResponseBody returns an option that configures response body logging.
func WithResponseBody(enabled bool) Option {
	return func(h *Handler) {
		h.respBody = func(*http.Response) bool {
			return enabled
		}
	}
}

// WithResponseBodyByContentTypes returns an option that logs response bodies based
// on opting in to the Content-Type of the response.
func WithResponseBodyByContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.respBody = func(res *http.Response) bool {
			rct := res.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return true
				}
			}
			return false
		}
	}
}

// WithSkipResponseBodyForContentTypes returns an option that logs response bodies based
// on opting out of the Content-Type of the response.
func WithSkipResponseBodyForContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.respBody = func(res *http.Response) bool {
			rct := res.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return false
				}
			}
			return true
		}
	}
}

// WithComment .
func WithComment(str string) Option {
	return func(h *Handler) {
		h.comment = str
	}
}

// WithCookie whether http requests carry cookies
func WithCookie(c bool) Option {
	return func(h *Handler) {
		h.cookie = c
	}
}

// WithTransport set http.RoundTripper
func WithTransport(t http.RoundTripper) Option {
	return func(h *Handler) {
		h.transport = t
	}
}
