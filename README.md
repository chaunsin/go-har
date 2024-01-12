# go-har

golang parses HAR files

# What is Har?

https://w3c.github.io/web-performance/specs/HAR/Overview.html

# Feature

- supports standard HAR-1.2 content parsing
- replay HTTP request based on har content stub content
- supports HTTP synchronous requests and asynchronous concurrent requests
- file import and export
- can be embedded in HTTP services to present data
- other conversion help functions, etc

# Use restriction

- golang version >= 1.9

# Tutorial

```go
package main

import (
	"context"
	"errors"
	"fmt
	"io"
	"log"
	"net/http"
	"time"

	har "github.com/chaunsin/go-har"
)

func Example() {
	path := "./testdata/zh.wikipedia.org.har"
	h, err := har.Parse(path)
	if err != nil {
		log.Fatalf("Parse: %s", err)
	}
	har := h.Export().Log
	fmt.Printf("version: %s create: %+v entries: %v\n", har.Version, har.Creator, h.EntryTotal())

	// add request filter
	filter := func(e *har.Entry) bool {
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
		case errors.Is(r.Error(), context.DeadlineExceeded):
			log.Printf("%s request is timeout!", r.Entry.Request.URL)
			continue
		case r.Error() != nil:
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
	filter = func(e *har.Entry) bool {
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

```