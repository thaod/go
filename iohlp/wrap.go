/*
Copyright 2016 Tamás Gulácsi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iohlp

import (
	"bufio"
	"io"

	"github.com/dgryski/go-linebreak"
)

// WrappingReader returns an io.Reader which will wrap lines longer than the given width.
// All other lines (LF chars) will be preserved.
func WrappingReader(r io.Reader, width uint) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(r)
		ew := &ErrWriter{Writer: pw}
		for scanner.Scan() { // split lines
			if uint(len(scanner.Bytes())) <= width {
				ew.Write(scanner.Bytes())
				if _, err := ew.Write([]byte{'\n'}); err != nil {
					break
				}
				continue
			}
			io.WriteString(ew, linebreak.Wrap(scanner.Text(), int(width)-5, int(width)))
			if _, err := ew.Write([]byte{'\n'}); err != nil {
				break
			}
		}
		err := scanner.Err()
		if err == nil {
			err = ew.Err()
		}
		pw.CloseWithError(err)
	}()

	return pr
}

// ErrWriter is a writer with a "stuck-in" error policy: writes normally,
// until the underlying io.Writer returns error; then after it always returns
// that error.
type ErrWriter struct {
	io.Writer
	err error
}

func (w *ErrWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	var n int
	n, w.err = w.Writer.Write(p)
	return n, w.err
}

// Err returns the first error the underlying io.Writer returned.
func (w *ErrWriter) Err() error { return w.err }
