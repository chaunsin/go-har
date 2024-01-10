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

import "time"

const _format = "2006-01-02T15:04:05.999999999"

type Time time.Time

func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.RFC3339 + `"`), nil
}

func (t *Time) MarshalText() ([]byte, error) {
	return []byte(time.Time(*t).Format(time.RFC3339)), nil
}

func (t *Time) UnmarshalJSON(text []byte) (err error) {
	now, err := time.ParseInLocation(`"`+time.RFC3339+`"`, string(text), time.Local)
	if err != nil {
		now, err = time.ParseInLocation(`"`+time.RFC3339Nano+`"`, string(text), time.Local)
		if err != nil {
			now, err = time.ParseInLocation(`"`+_format+`"`, string(text), time.Local)
			if err != nil {
				return
			}
		}
	}
	*t = Time(now)
	return
}

func (t *Time) UnmarshalText(text []byte) (err error) {
	now, err := time.ParseInLocation(`"`+time.RFC3339+`"`, string(text), time.Local)
	if err != nil {
		now, err = time.ParseInLocation(`"`+time.RFC3339Nano+`"`, string(text), time.Local)
		if err != nil {
			now, err = time.ParseInLocation(`"`+_format+`"`, string(text), time.Local)
			if err != nil {
				return
			}
		}
	}
	*t = Time(now)
	return
}

func (t *Time) String() string {
	return time.Time(*t).String()
}
