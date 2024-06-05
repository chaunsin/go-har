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
	"log/slog"
	"os"
	"runtime"
	"time"
)

var ctx = context.Background()

type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

type logger struct {
	log   *slog.Logger
	level *slog.LevelVar
}

func NewLogger(format string, level string, writer ...io.Writer) Logger {
	var (
		h    slog.Handler
		lv   slog.LevelVar
		w    = []io.Writer{os.Stderr}
		opts = slog.HandlerOptions{
			AddSource:   true,
			Level:       &lv,
			ReplaceAttr: nil,
		}
	)

	w = append(w, writer...)

	switch level {
	case "debug":
		lv.Set(slog.LevelDebug)
	case "info":
		lv.Set(slog.LevelInfo)
	case "level":
		lv.Set(slog.LevelWarn)
	case "error":
		lv.Set(slog.LevelError)
	default:
		lv.Set(slog.LevelDebug)
	}

	switch format {
	case "json":
		h = slog.NewJSONHandler(io.MultiWriter(w...), &opts)
	case "text":
		fallthrough
	default:
		h = slog.NewTextHandler(io.MultiWriter(w...), &opts)
	}
	h = h.WithAttrs([]slog.Attr{slog.String("app", "go-har")})

	l := logger{
		log:   slog.New(h),
		level: &lv,
	}
	return &l
}

func (l *logger) logger(ctx context.Context, lv slog.Level, msg string, args ...any) {
	if !l.log.Handler().Enabled(ctx, lv) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, Info]
	r := slog.NewRecord(time.Now(), lv, msg, pcs[0])
	r.Add(args...)
	if err := l.log.Handler().Handle(ctx, r); err != nil {
		fmt.Printf("log handle err: %s\n", err)
	}
}

func (l *logger) Debug(format string, args ...any) {
	l.logger(ctx, slog.LevelDebug, fmt.Sprintf(format, args...))
}

func (l *logger) Info(format string, args ...any) {
	l.logger(ctx, slog.LevelInfo, fmt.Sprintf(format, args...))
}

func (l *logger) Warn(format string, args ...any) {
	l.logger(ctx, slog.LevelWarn, fmt.Sprintf(format, args...))
}

func (l *logger) Error(format string, args ...any) {
	l.logger(ctx, slog.LevelError, fmt.Sprintf(format, args...))
}
