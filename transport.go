package uhttp

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/lex/httplex"
)

const defaultPacketSize = 8192

// Transport implements http.RoundTripper and RoundTripMultier and allows for sending HTTP requests
// over UDP.  It supports both unicast and multicast destinations.
type Transport struct {
	// MaxSize is the maximum allowable size of an HTTP Request.  It cannot be larger than 64k (UDP limit).
	// A zero value will use the default of 8k.
	MaxSize int

	// WaitTime is the default time we will spend waiting for HTTP responses before returning.  A zero
	// value means wait forever.
	WaitTime time.Duration

	// HeaderCanon provides the canonical header name for the given header.  If this function returns an
	// empty string, the header will be omitted.  This is used when precise control over the case of the
	// resulting HTTP headers is needed.  Filtering headers such as "Host" may break compatibility with
	// HTTP/1.1 (but let's face it, none of this is standard).
	HeaderCanon func(name string) string

	// Repeat enables requests to be repeated, according to the delays returned by the resulting
	// RepeatFunc.
	Repeat RepeatGenerator

	bufPool sync.Pool
}

func (t *Transport) getMaxSize() int {
	if t.MaxSize > 0 {
		return t.MaxSize
	}
	return defaultPacketSize
}

type RoundTripMultier interface {
	http.RoundTripper

	// RoundTripMulti is responsible for delivering req, and waiting for responses to arrive.  For
	// each response received, fn will be invoked so that the caller can process it.  This method will
	// return when wait is reached (if non-zero), req.Context() expires, or an error occurs.  The fn
	// implementation may return the sentinal error Stop to halt processing without causing
	// RoundTripMulti to return an error.
	RoundTripMulti(req *http.Request, wait time.Duration, fn func(sender net.Addr, res *http.Response) error) error
}

var DefaultTransport RoundTripMultier = &Transport{
	MaxSize:  defaultPacketSize,
	WaitTime: 3 * time.Second,
}

func (t *Transport) newBuf() []byte {
	if b := t.bufPool.Get(); b != nil {
		return b.([]byte)
	}
	// We don't use pool.New since t.MaxSize is a field of Transport and this simplifies
	// initialization.
	return make([]byte, t.getMaxSize())
}

func (t *Transport) releaseBuf(b []byte) {
	t.bufPool.Put(b)
}

type timeoutErr string

func (e timeoutErr) Error() string   { return string(e) }
func (e timeoutErr) Timeout() bool   { return true }
func (e timeoutErr) Temporary() bool { return true }

var ErrTimeout error = timeoutErr("timeout waiting for responses")
var Stop = errors.New("stop processing")

// RoundTrip issues a UDP HTTP request and waits for a single response.  Returns
// when a response was received, when the req.Context() expires, or when
// t.MaxWait is reached (if non-zero).
func (t *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	err = t.RoundTripMulti(req, 0, func(_ net.Addr, r *http.Response) error {
		res = r
		return Stop
	})
	if res == nil && err == nil {
		err = ErrTimeout
	}
	return
}

func closeBody(req *http.Request) {
	if req.Body != nil {
		req.Body.Close()
	}
}

func validateRequest(req *http.Request) error {
	if req.URL == nil {
		return errors.New("uhttp: nil http.Request.URL")
	}
	if req.URL.Host == "" {
		return errors.New("uhttp: missing Host in http.Request.URL")
	}
	if req.Header == nil {
		return errors.New("uhttp: nil http.Request.Header")
	}
	for k, vals := range req.Header {
		if !httplex.ValidHeaderFieldName(k) {
			return fmt.Errorf("uhttp: invalid header field name %q", k)
		}
		for _, v := range vals {
			if !httplex.ValidHeaderFieldValue(v) {
				return fmt.Errorf("uhttp: invalid header field value %q for key %v", v, k)
			}
		}
	}
	return nil
}

// repeat repeats fn for every durFn call that returns a non-nil delay time.  Returns when
// ctx expires, every returns nil, or fn returns an error.
func repeat(ctx context.Context, durFn func(_ time.Duration) *time.Duration, fn func() error) {
	prev := time.Duration(0)
	for next := durFn(prev); next != nil; next = durFn(prev) {
		prev = *next
		select {
		case <-time.After(*next):
			if err := fn(); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (t *Transport) sendDirect(ctx context.Context, address string, data []byte) (n int, conn net.PacketConn, err error) {
	// Listen on a new UDP socket with a system-assigned local port number, "connected" to the
	// remote unicast UDP endpoint.
	var d net.Dialer
	c, err := d.DialContext(ctx, "udp", address)
	if err != nil {
		return 0, nil, fmt.Errorf("uhttp: dial %q: %v", address, err)
	}
	conn = c.(net.PacketConn)

	// Send the request.
	if n, err = c.Write(data); err != nil {
		conn.Close()
		err = fmt.Errorf("uhttp: write request to %q: %v", address, err)
		conn = nil
		return
	}

	if t.Repeat != nil {
		// Send duplicate requests if requested.  This goroutine will continue running based on the behavior of
		// t.Repeat and will automatically exit when ctx expires.
		go repeat(ctx, t.Repeat(), func() error {
			_, err := c.Write(data)
			return err
		})
	}
	return
}

func (t *Transport) sendMulti(ctx context.Context, addr *net.UDPAddr, data []byte) (n int, conn net.PacketConn, err error) {
	// Listen on all addresses with a request-specific system-assigned UDP port number.
	conn, err = net.ListenPacket("udp", "")
	if err != nil {
		err = fmt.Errorf("uhttp: listen: %v", err)
		return
	}

	// Send the request.
	if n, err = conn.WriteTo(data, addr); err != nil {
		conn.Close()
		err = fmt.Errorf("uhttp: write request to %q: %v", addr, err)
		conn = nil
		return
	}

	if t.Repeat != nil {
		// Send duplicate requests if requested.  This goroutine will continue running based on the behavior of
		// t.Repeat and will automatically exit when ctx expires.
		go repeat(ctx, t.Repeat(), func() error {
			_, err := conn.WriteTo(data, addr)
			return err
		})
	}
	return
}

// WriteRequest writes req to w, in wire format.  If req is larger than t.MaxSize, returns
// an error.  This applies header canonicalization per t.HeaderCanon, if it's provided.
func (t *Transport) WriteRequest(w io.Writer, req *http.Request) error {
	w = &limitedWriter{Writer: w, N: t.getMaxSize()}

	var wc io.WriteCloser
	if t.HeaderCanon != nil {
		wc = newHeaderCanon(t.HeaderCanon, w)
		w = wc
	}

	if err := req.Write(w); err != nil {
		if err == io.ErrShortWrite {
			return fmt.Errorf("uhttp: http.Request does not fit in MaxSize of %d", t.MaxSize)
		}
		return err
	}
	if wc != nil {
		wc.Close()
	}
	return nil
}

// RoundTripMulti issues a UDP HTTP request and calls fn for each response received.  Returns when wait
// is reached (no error), req.Context() expires, an error occurs, or when fn returns an error.  The
// sentinal error Stop may be returned by fn to cause this method to return immediately without error.
func (t *Transport) RoundTripMulti(req *http.Request, wait time.Duration, fn func(sender net.Addr, r *http.Response) error) (err error) {
	if err = validateRequest(req); err != nil {
		closeBody(req)
		return err
	}

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	// Grab a []byte buffer and write req into it.
	b := t.newBuf()
	defer func() { t.releaseBuf(b) }()
	buf := bytes.NewBuffer(b[:0])
	t.WriteRequest(buf, req)

	var conn net.PacketConn
	var n int
	raddr, err := net.ResolveUDPAddr("udp", req.URL.Host)
	if err != nil {
		return fmt.Errorf("uhttp: resolve %q: %v", req.URL.Host, err)
	}

	// If the request is intended for a multicast group, we need to explicitly
	// listen and receive packets from arbitrary responders.  Otherwise, we use
	// Dial so that we can get 'connection refused' errors and automatic
	// filtering of responses that don't come from the server.
	if raddr.IP.Equal(net.IPv4bcast) || raddr.IP.IsMulticast() {
		n, conn, err = t.sendMulti(ctx, raddr, buf.Bytes())
	} else {
		n, conn, err = t.sendDirect(ctx, req.URL.Host, buf.Bytes())
	}
	if err != nil {
		return fmt.Errorf("uhttp send request: %v", err)
	}
	defer conn.Close()

	if n != buf.Len() {
		// Shouldn't normally happen.
		panic(fmt.Sprintf("udp attempted to write %d bytes, wrote %d", buf.Len(), n))
	}

	type packet struct {
		addr net.Addr
		data []byte
		err  error
	}

	// Read from conn in a goroutine, until conn is closed.
	ch := make(chan *packet)
	go func() {
		for {
			n, addr, err := conn.ReadFrom(b)
			ch <- &packet{addr, b[:n], err}
			if err != nil {
				break
			}
		}
		close(ch)
	}()

	if wait == 0 {
		wait = t.WaitTime
	}
	var waitCh <-chan time.Time
	if wait > 0 {
		waitCh = time.After(wait)
	}

forloop:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break forloop
		case <-waitCh:
			break forloop
		case p := <-ch:
			if p == nil {
				// Channel was closed.  We shouldn't normally get here since the next case is
				// the only time the channel should be closed and we'll have already broken out.
				break forloop
			}
			if p.err != nil {
				// This will be the last message we receive.
				err = p.err
				break forloop
			}

			r, er := http.ReadResponse(bufio.NewReader(bytes.NewReader(p.data)), req)
			if er != nil {
				err = fmt.Errorf("uhttp: parse response: %v", err)
				// Discard this packet and wait to see if more arrive.  If none do, this error will stand.
				continue
			}
			if err = fn(p.addr, r); err != nil {
				break forloop
			}
		}
	}

	if err == Stop {
		err = nil
	}
	return
}
