package uhttp

import (
	"net/http"
	"time"
)

// Client is an UDP over HTTP client.  It is largely a simplistic wrapper over RoundTripMultier,
// meant to ease use for people familiar with http.Client.
type Client struct {
	Transport RoundTripMultier
}

// Get makes a GET request for url, waits up to wait, and delivers any responses received to fn.
// If an error occurs, or fn returns an error, execution will return immediately.  The sentinal
// error Stop may be returned to cause Get to return immediately with no error.
func (c *Client) Get(url string, wait time.Duration, fn func(resp *Response) error) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	return c.Do(req, wait, fn)
}

// Do issues r, waits up to wait, and delivers any responses received to fn.
// If an error occurs, or fn returns an error, execution will return immediately.  The sentinal
// error Stop may be returned to cause Get to return immediately with no error.
func (c *Client) Do(r *http.Request, wait time.Duration, fn func(resp *Response) error) error {
	return c.Transport.RoundTripMulti(r, wait, fn)
}

// DefaultClient is the Client used by the top-level Do and Get functions.
var DefaultClient = &Client{
	Transport: DefaultTransport,
}

// Do issues r via DefaultClient, waits up to wait, and delivers any responses received to fn.
// If an error occurs, or fn returns an error, execution will return immediately.  The sentinal
// error Stop may be returned to cause Get to return immediately with no error.
func Do(r *http.Request, wait time.Duration, fn func(resp *Response) error) error {
	return DefaultClient.Do(r, wait, fn)
}

// Get makes a GET request for url via DefaultClient, waits up to wait, and delivers any responses received to fn.
// If an error occurs, or fn returns an error, execution will return immediately.  The sentinal
// error Stop may be returned to cause Get to return immediately with no error.
func Get(url string, wait time.Duration, fn func(resp *Response) error) error {
	return DefaultClient.Get(url, wait, fn)
}
