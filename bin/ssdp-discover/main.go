// This is just a test binary to validate that the package is working correctly
// by trying to do something semi-useful.  It is excluded from the build so that
// it does not get installed by users of this package.
package main

// +build ignore

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dnesting/uhttp"
)

var (
	address  = flag.String("address", "239.255.255.250:1900", "send requests to this address in host:port form (or [addr%iface]:port for ipv6)")
	waitSecs = flag.Int("wait_secs", 1, "number of seconds to wait for a response")
	target   = flag.String("target", "ssdp:all", "search target (e.g. upnp:rootdevice)")
)

func main() {
	flag.Parse()
	client := uhttp.Client{
		Transport: &uhttp.Transport{
			HeaderCanon: func(n string) string { return strings.ToUpper(n) },
		},
	}
	req, _ := http.NewRequest("M-SEARCH", "", nil)
	req.URL.Host = *address
	req.URL.Path = "*"
	req.Header.Add("MAN", `"ssdp:discover"`)
	req.Header.Add("MX", strconv.Itoa(*waitSecs))
	req.Header.Add("ST", *target)
	req.Header.Add("CPFN.UPNP.ORG", "Test")

	// Add 100ms to waitSecs to account for any network or device delays.
	wait := time.Duration(*waitSecs)*time.Second + 100*time.Millisecond

	err := client.Do(req, wait, func(sender net.Addr, resp *http.Response) error {
		fmt.Printf("--- From %s:\n", sender)
		resp.Write(os.Stdout)
		fmt.Println("---")
		return nil
	})
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}
