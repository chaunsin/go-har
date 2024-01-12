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
	"context"
	"fmt"
	"io"
	"testing"
)

func TestParse(t *testing.T) {
	path := "./testdata/sample.har"
	h, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%+v\n", h.Export())
}

func TestSyncExecute(t *testing.T) {
	path := "./testdata/sample.har"
	h, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}

	filter := func(e *Entry) bool {
		if e.Request.URL == "https://music.163.com/eapi/batch" {
			return true
		}
		return false
	}

	receipt, err := h.SyncExecute(context.TODO(), filter)
	if err != nil {
		t.Fatalf("SyncExecute: %s", err)
	}
	for r := range receipt {
		if r.Error() != nil {
			t.Errorf("execute %s err: %s", r.Entry.Request.URL, r.Error())
			continue
		}
		func() {
			defer r.Response.Body.Close()
			body, err := io.ReadAll(r.Response.Body)
			if err != nil {
				t.Errorf("readall err:%s", err)
				return
			}
			t.Logf("url:%s status:%s body:%s\n", r.Entry.Request.URL, r.Response.Status, string(body))
		}()
	}
}

func TestExecute(t *testing.T) {
	path := "./testdata/sample.har"
	h, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}

	filter := func(e *Entry) bool {
		if e.Request.URL == "https://music.163.com/eapi/batch" {
			return true
		}
		return false
	}

	receipt, err := h.Execute(context.TODO(), filter)
	if err != nil {
		t.Fatalf("SyncExecute: %s", err)
	}
	for _, r := range receipt {
		if r.Error() != nil {
			t.Errorf("execute %s err: %s", r.Entry.Request.URL, r.Error())
			continue
		}
		func() {
			defer r.Response.Body.Close()
			body, err := io.ReadAll(r.Response.Body)
			if err != nil {
				t.Errorf("readall err:%s", err)
			}
			t.Logf("url:%s status:%s body:%s\n", r.Entry.Request.URL, r.Response.Status, string(body))
		}()
	}
}
