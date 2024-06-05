# go-har

[![GoDoc](https://godoc.org/github.com/chaunsin/go-har?status.svg)](https://godoc.org/github.com/chaunsin/go-har) [![Go Report Card](https://goreportcard.com/badge/github.com/chaunsin/go-har)](https://goreportcard.com/report/github.com/chaunsin/go-har)

golang parses HAR files

## What is Har?

https://w3c.github.io/web-performance/specs/HAR/Overview.html

https://toolbox.googleapps.com/apps/har_analyzer/

## Feature

- supports standard HAR-1.2 content parsing
- replay HTTP request based on har content stub content
- supports HTTP synchronous requests and asynchronous concurrent requests
- .har file import and export
- can be embedded in HTTP services to present data
- build HAR based on http.Request and http.Response

## Use restriction

- golang version >= 1.21

## Example

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
	var path = "./testdata/zh.wikipedia.org.har"
	// parse har file
	h, err := Parse(path, WithCookie(true))
	if err != nil {
		log.Fatalf("Parse: %s", err)
	}
	har := h.Export().Log
	fmt.Printf("version: %s create: %+v entries: %v\n", har.Version, har.Creator, h.EntryTotal())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// construct request filter
	filter := []RequestOption{
		WithRequestUrlIs("https://zh.wikipedia.org/wiki/.har"),
		WithRequestMethod("GET"),
		// add another filter
	}

	// concurrent execution http request
	receipt, err := h.SyncExecute(ctx, filter...)
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

		// Anonymous functions avoid body resource leakage
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

	// add a new golang standard http request
	uniqueId := "1"
	request, err := http.NewRequest(http.MethodGet, "https://www.baidu.com", nil)
	if err != nil {
		panic(err)
	}
	if err := h.AddRequest(uniqueId, request); err != nil {
		// err maybe unique id is repeated
		log.Fatalf("add request failed: %s", err)
	}

	// exclude other requests, ready for execution https://www.baidu.com
	filter = []RequestOption{WithRequestUrlIs("https://www.baidu.com")}

	// sequential execution http request
	execReceipt, err := h.Execute(context.TODO(), filter...)
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

	// clean har
	h.Reset()
	fmt.Println("entries:", h.EntryTotal())

	// Output:
	// version: 1.2 create: &{Name:WebInspector Version:537.36 Comment:} entries: 3
	// url: https://zh.wikipedia.org/wiki/.har status: 200 OK
	// url: https://www.baidu.com status: 200 OK
	// entries: 0
}
```

Please refer to [example_test.go](./example_test.go)