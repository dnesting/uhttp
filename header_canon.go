package uhttp

// This is super, super ugly.  Because some HTTP-over-UDP protocols may need tighter control
// over things like headers, we can't completely rely on the stock http package's implementation
// to get things right for all cases.  In particular, for SSDP, the stock http.Request.Write
// method will force the use of a "Host" header, whereas SSDP requires that this be all upper-case.
// We can't fix this without either abandoning the http.Request.Write implementation entirely,
// or tampering with the output stream.  We choose to do the latter.  This may be a questionable
// decision.

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
)

type headerCanon struct {
	io.WriteCloser
	err error
	wg  sync.WaitGroup
}

// newHeaderCanon creates an io.WriteCloser that expects an HTTP request in wire format.
// For each HTTP header observed written to w, fn will be called with the header name.  If
// fn returns a string, the header is kept using the provided header name.
// Otherwise it is dropped.  Callers MUST call Close to ensure all data is flushed to w.
func newHeaderCanon(fn func(name string) string, w io.Writer) io.WriteCloser {
	pr, pw := io.Pipe()
	hs := &headerCanon{
		WriteCloser: pw,
	}
	hs.wg.Add(1)
	go func() {
		_, _, hs.err = canonHeaders(fn, w, pr)
		hs.wg.Done()
	}()
	return hs
}

// Close signals that no more data will be written to hs.  This is necessary to ensure
// all data is flushed.
func (hs *headerCanon) Close() error {
	err := hs.WriteCloser.Close()
	if err != nil {
		return err
	}
	hs.wg.Wait()
	return hs.err
}

// canonHeaders expects an HTTP request in wire format on r.  It then functions like io.Copy and
// will write the same HTTP request to w, omitting any HTTP headers for which fn returns "", and
// changing header names to match the name returned by fn.
// Returns the number of bytes read, written, and any error occurred reading or writing.
func canonHeaders(fn func(name string) string, w io.Writer, r io.Reader) (read, written int64, err error) {
	br := bufio.NewReader(r)
	for {
		var line string
		var nr int

		line, err = br.ReadString('\n')
		read += int64(len(line))
		if err != nil {
			break
		}
		if line == "\r\n" {
			// end of headers
			nr, err = io.WriteString(w, line)
			written += int64(nr)
			if err != nil {
				break
			}
			var nr64 int64
			nr64, err = io.Copy(w, br)
			read += nr64
			written += nr64
			break
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			k := fn(kv[0])
			if k != "" {
				nr, err = io.WriteString(w, fmt.Sprintf("%s:%s", k, kv[1]))
				written += int64(nr)
				if err != nil {
					break
				}
			}
		} else {
			// Probably status line
			nr, err = io.WriteString(w, kv[0])
			written += int64(nr)
			if err != nil {
				break
			}
		}
	}
	return
}
