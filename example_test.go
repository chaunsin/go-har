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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func Example() {
	path := "./testdata/zh.wikipedia.org.har"
	// parse har file
	h, err := Parse(path)
	if err != nil {
		log.Fatalf("Parse: %s", err)
	}
	har := h.Export().Log
	fmt.Printf("version: %s create: %+v entries: %v\n", har.Version, har.Creator, h.EntryTotal())

	// add request filter
	filter := func(e *Entry) bool {
		if e.Request.URL == "https://zh.wikipedia.org/wiki/.har" {
			return true
		}
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// concurrent execution http request
	receipt, err := h.SyncExecute(ctx, filter)
	if err != nil {
		log.Fatalf("SyncExecute: %s", err)
	}
	for r := range receipt {
		switch {
		case errors.Is(r.err, context.DeadlineExceeded):
			log.Printf("%s request is timeout!", r.Entry.Request.URL)
			continue
		case r.err != nil:
			log.Printf("%s request failed: %s\n", r.Entry.Request.URL, r.Error())
			continue
		}
		func() {
			defer r.Response.Body.Close()
			_, err := io.ReadAll(r.Response.Body)
			if err != nil {
				log.Fatalf("readall err:%s", err)
				return
			}
			fmt.Printf("url: %s status: %s\n", r.Entry.Request.URL, r.Response.Status)
		}()
	}

	// add a new request
	uniqueId := "1"
	request, err := http.NewRequest(http.MethodGet, "https://www.baidu.com", nil)
	if err != nil {
		panic(err)
	}
	if err := h.AddRequest(uniqueId, request); err != nil {
		// err maybe unique id is repeated
		log.Fatalf("add request failed: %s", err)
	}

	// exclude other requests
	filter = func(e *Entry) bool {
		if e.Request.URL == "https://www.baidu.com" {
			return true
		}
		return false
	}
	// sequential execution http request
	execReceipt, err := h.Execute(context.TODO(), filter)
	if err != nil {
		log.Fatalf("Execute: %s", err)
	}
	for _, r := range execReceipt {
		if r.Error() != nil {
			log.Printf("%s request failed: %s\n", r.Entry.Request.URL, r.Error())
			continue
		}
		func() {
			defer r.Response.Body.Close()
			// read body do something's
			_, err := io.ReadAll(r.Response.Body)
			if err != nil {
				log.Fatalf("readall err:%s", err)
				return
			}

			// fill in response to current entry.Response
			if err := r.FillInResponse(); err != nil {
				log.Printf("FillInResponse: %e", err)
			}

			fmt.Printf("url: %s status: %s\n", r.Entry.Request.URL, r.Response.Status)
		}()
	}

	// Writes the contents of the internal har json object to IO
	if err := h.Write(io.Discard); err != nil {
		log.Fatalf("write err:%s", err)
	}

	// clean
	h.Reset()
	fmt.Println("entries:", h.EntryTotal())

	// Output:
	// version: 1.2 create: &{Name:WebInspector Version:537.36 Comment:} entries: 3
	// url: https://zh.wikipedia.org/wiki/.har status: 200 OK
	// url: https://www.baidu.com status: 200 OK
	// entries: 0
}

func ExampleParse() {
	path := "./testdata/zh.wikipedia.org.har"
	h, err := Parse(path)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", h.Export())

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	h, err = NewReader(file)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", h.Export())

	// user default har
	h, err = NewHandler(nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", h.Export())

	// Output:
	// &{Log:{Version:1.2 Creator:0xc00010af90 Browser:<nil> Pages:[] Entries:[0xc00015a080] Comment:}}
	// &{Log:{Version:1.2 Creator:0xc0001ae5d0 Browser:<nil> Pages:[] Entries:[0xc00015a180] Comment:}}
	// &{Log:{Version:1.2 Creator:0xc0001aecc0 Browser:<nil> Pages:[] Entries:[] Comment:}}
}
