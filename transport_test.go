package uhttp_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dnesting/uhttp"
)

func ExampleTransport_sSDP() {
	// This example performs an SSDP M-SEARCH to the local Multicast SSDP address.
	// It leverages the stock Go http.Client with uhttp.Transport.  Only the first
	// response will be returned.
	client := http.Client{
		Transport: &uhttp.Transport{
			// SSDP requires upper-case header names
			HeaderCanon: func(n string) string { return strings.ToUpper(n) },
			WaitTime:    2 * time.Second,
			Repeat:      uhttp.RepeatAfter(50*time.Millisecond, 1),
		},
	}

	// Build M-SEARCH request
	req, _ := http.NewRequest("M-SEARCH", "", nil)
	req.URL.Host = "239.255.255.250:1900"
	req.URL.Path = "*"
	req.Header.Add("MAN", `"ssdp:discover"`)
	req.Header.Add("MX", "1")
	req.Header.Add("ST", "upnp:rootdevice")
	req.Header.Add("CPFN.UPNP.ORG", "Test")

	// Leverage stock http.Client to actually do the work.
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	} else {
		resp.Write(os.Stdout)
	}
}
