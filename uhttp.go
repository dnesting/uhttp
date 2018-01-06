// Package uhttp provides a transport for HTTP over UDP.  It implements http.RoundTripper so that
// it can plug in to stock Go HTTP libraries.
//
// Caveat: No real standard for HTTP over UDP exists.  This may not work well for all protocols
// that look like HTTP over UDP.
package uhttp

import (
	"net"
	"net/http"
)

/*
type Message struct {
	Response
	Request *http.Request
}
*/

type Response struct {
	Response *http.Response
	Addr     net.Addr
}
