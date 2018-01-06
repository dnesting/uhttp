// This is just a test binary to validate that the package is working correctly
// by trying to do something semi-useful.  It is excluded from the build so that
// it does not get installed by users of this package.
package main

// +build ignore

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnesting/uhttp"
)

func main() {
	client := uhttp.Client{
		Transport: &uhttp.Transport{
			HeaderCanon: func(n string) string { return strings.ToUpper(n) },
		},
	}
	req, _ := http.NewRequest("M-SEARCH", "", nil)
	req.URL = &url.URL{
		Host: "239.255.255.250:1900",
		Path: "*",
	}
	req.Header.Add("MAN", `"ssdp:discover"`)
	req.Header.Add("MX", "1")
	req.Header.Add("ST", "ssdp:all")
	req.Header.Add("CPFN.UPNP.ORG", "Test")

	err := client.Do(req, 2*time.Second, func(resp *uhttp.Response) error {
		fmt.Printf("From %s:\n", resp.Addr)
		resp.Response.Write(os.Stdout)
		fmt.Println("---")
		return nil
	})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}
