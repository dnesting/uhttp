package uhttp_test

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnesting/uhttp"
)

func ExampleClient_sSDP() {
	// This example performs an SSDP M-SEARCH to the local Multicast SSDP address.
	// It uses the uhttp.Client so as to receive multiple responses.
	client := uhttp.Client{
		Transport: &uhttp.Transport{
			// SSDP requires upper-case header names
			HeaderCanon: func(n string) string { return strings.ToUpper(n) },
		},
	}

	// Build M-SEARCH request
	req, _ := http.NewRequest("M-SEARCH", "", nil)
	req.URL = &url.URL{
		Host: "239.255.255.250:1900",
		Path: "*", // We specify req.URL.Path explicitly so that we get "*" instead of "/*"
	}
	req.Header.Add("MAN", `"ssdp:discover"`)
	req.Header.Add("MX", "1")
	req.Header.Add("ST", "upnp:rootdevice")
	req.Header.Add("CPFN.UPNP.ORG", "Test")

	err := client.Do(req, 3*time.Second, func(resp *uhttp.Response) error {
		fmt.Printf("From %s:\n", resp.Addr)
		resp.Response.Write(os.Stdout)
		fmt.Println("---")
		return nil
	})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}
