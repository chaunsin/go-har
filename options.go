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
	"regexp"
	"strings"
)

// Option represents the optional function
type Option func(h *Handler)

// WithRequestBody returns an option that configures request post data logging.
func WithRequestBody(enabled bool) Option {
	return func(h *Handler) {
		h.reqBody = append(h.reqBody, func(*http.Request) bool {
			return enabled
		})
	}
}

// WithRequestBodyByContentTypes returns an option that logs request bodies based
// on opting in to the Content-Type of the request.
func WithRequestBodyByContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.reqBody = append(h.reqBody, func(req *http.Request) bool {
			rct := req.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return true
				}
			}
			return false
		})
	}
}

// WithSkipRequestBodyForContentTypes returns an option that logs request bodies
// based on opting out of the Content-Type of the request.
func WithSkipRequestBodyForContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.reqBody = append(h.reqBody, func(req *http.Request) bool {
			rct := req.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return false
				}
			}
			return true
		})
	}
}

// WithResponseBody returns an option that configures response body logging.
func WithResponseBody(enabled bool) Option {
	return func(h *Handler) {
		h.respBody = append(h.respBody, func(*http.Response) bool {
			return enabled
		})
	}
}

// WithResponseBodyByContentTypes returns an option that logs response bodies based
// on opting in to the Content-Type of the response.
func WithResponseBodyByContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.respBody = append(h.respBody, func(res *http.Response) bool {
			rct := res.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return true
				}
			}
			return false
		})
	}
}

// WithSkipResponseBodyForContentTypes returns an option that logs response bodies
// based on opting out of the Content-Type of the response.
func WithSkipResponseBodyForContentTypes(cts ...string) Option {
	return func(h *Handler) {
		h.respBody = append(h.respBody, func(res *http.Response) bool {
			rct := res.Header.Get("Content-Type")
			for _, ct := range cts {
				if strings.HasPrefix(strings.ToLower(rct), strings.ToLower(ct)) {
					return false
				}
			}
			return true
		})
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

// WithRequestConcurrency Number of concurrent http requests allowed. 0 indicates no limit.
func WithRequestConcurrency(num uint64) Option {
	return func(ctx *Handler) {
		ctx.concurrency.Store(int64(num))
	}
}

// WithLogger Register a logger object interface
func WithLogger(l Logger) Option {
	return func(ctx *Handler) {
		ctx.log = l
	}
}

// RequestOption SyncExecute() or Execute() represents the optional function
// TODO: The current condition is a OR condition, which needs to be changed into an AND policy
type RequestOption func(ctx *Handler, e *Entry) bool

// WithRequestUrlIs .
func WithRequestUrlIs(urls ...string) RequestOption {
	var urlSet = make(map[string]struct{}, len(urls))
	for _, u := range urls {
		urlSet[u] = struct{}{}
	}
	return func(ctx *Handler, e *Entry) bool {
		_, ok := urlSet[e.Request.URL]
		return ok
	}
}

// WithRequestUrlPrefix .
func WithRequestUrlPrefix(prefix string) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		return strings.HasPrefix(e.Request.URL, prefix)
	}
}

// WithRequestUrlRegexp .
func WithRequestUrlRegexp(regexps *regexp.Regexp) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		if regexps == nil {
			return false
		}
		return regexps.MatchString(e.Request.URL)
	}
}

// WithRequestHostIs .
func WithRequestHostIs(hosts ...string) RequestOption {
	var hostSet = make(map[string]struct{}, len(hosts))
	for _, u := range hosts {
		hostSet[u] = struct{}{}
	}
	return func(ctx *Handler, e *Entry) bool {
		_, ok := hostSet[e.Request.URL]
		return ok
	}
}

// WithRequestHostRegexp .
func WithRequestHostRegexp(regexps ...*regexp.Regexp) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		for _, r := range regexps {
			if r.MatchString(e.Request.URL) {
				return true
			}
		}
		return false
	}
}

// WithRequestMethod .
func WithRequestMethod(methods ...string) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		for _, r := range methods {
			if e.Request.Method == strings.ToUpper(r) {
				return true
			}
		}
		return false
	}
}

// WithSkipRequestMethod .
func WithSkipRequestMethod(methods ...string) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		for _, r := range methods {
			if e.Request.Method != strings.ToUpper(r) {
				return false
			}
		}
		return true
	}
}

// WithRequestHandler .
func WithRequestHandler(handler EntityHandler) RequestOption {
	return func(ctx *Handler, e *Entry) bool {
		return handler(e)
	}
}

// // WithResponseStatusCode .
// func WithResponseStatusCode(codes ...int) RequestOption {
// 	var codeSet = make(map[int]struct{}, len(codes))
// 	for _, c := range codes {
// 		codeSet[c] = struct{}{}
// 	}
// 	return func(ctx *Handler, e *Entry) bool {
// 		if e == nil || e.Response == nil {
// 			return false
// 		}
// 		_, ok := codeSet[e.Response.Status]
// 		return ok
// 	}
// }

// // WithResponseHandler .
// func WithResponseHandler(handler EntityHandler) RequestOption {
// 	return func(h *Handler) {
// 		h.append(nil, handler)
// 	}
// }
