// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"net"
	"os"
)

// Returns first IP address of host.
func getipaddr() string {
	var ipaddr string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}
	// Returns first IP address
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				//os.Stdout.WriteString(ipnet.IP.String() + "\n")
				ipaddr = ipnet.IP.String()
				break
			}
		}
	}
	return ipaddr

}
