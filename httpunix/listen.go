// Copyright (c) 2019 Tamás Gulácsi.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package httpunix

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	errors "golang.org/x/xerrors"
)

// ListenAndServe is the same as http.ListenAndServe, except it can listen on unix domain sockets.
func ListenAndServe(ctx context.Context, addr string, hndl http.Handler) error {
	addr = strings.TrimPrefix(addr, "http+")
	if !strings.HasPrefix(addr, "unix:") {
		return http.ListenAndServe(addr, hndl)
	}
	addrU := addr
	addr = strings.TrimPrefix(addr[4:], "://")
	addr = strings.TrimPrefix(addr, ":")
	os.Remove(addr)
	ln, err := net.Listen("unix", addr)
	if err != nil {
		return errors.Errorf("%s: %w", addrU, err)
	}
	defer ln.Close()
	srv := http.Server{
		Addr:              addrU,
		Handler:           hndl,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	return srv.Serve(ln)
}