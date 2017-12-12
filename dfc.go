// CopyRight Notice: All rights reserved
//
//

// Package dfc refers to Distributed File Cache.It serves as Write Through Pesistent
// Cache for S3 Objects. It's designed to support large number of caching(storage) servers handling
// MultiPetaByte workload. DFC can be run as Proxy Client or Storage Server Instance.
// Proxy Client need to be started first and it can work without any DFC's storage server instance
// aka serving directly through backend storage(S3).
package dfc

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/oklog/oklog/pkg/group"
)

const (
	rolesetrver = "server"
	roleproxy   = "proxy"
)

// global context for each DFC instance (Proxy or Storage Server)
type ctxDfc struct {

	// Map of Registered storage servers with Proxy Instance. It will be NIL for Storage
	// Server Instance.
	smap map[string]serverinfo

	// Configuration
	config dfconfig

	// Channel for  cancellation/termination signal.
	sig chan os.Signal

	// Channel for listening cancellation request.
	cancel chan struct{}

	// Channel for cancelling FSCheck timer
	fschkchan chan bool

	// List of Usable mountpoints on storage server
	mntpath []MountPoint

	// stopinprogress is set during main daemon thread stopping. DFC instance cannot
	// accept new http requests once stopinprogress is set.
	stopinprogress bool

	// DFC can run as Proxy or Storage Server Instance..
	// True will imply running as Proxy and
	// False will imply running as Storage Server.
	proxy bool

	// CheckFS is running or not.
	checkfsrunning bool

	// WaitGroup for completing Http Requests.
	httprqwg sync.WaitGroup

	// http listener
	listener net.Listener

	// WaitGroup for completing fscheck on all mountpaths
	fschkwg sync.WaitGroup
}

// Server Registration info
type serverinfo struct {
	// PORT refers to http listening port id of DFC instance.
	port string

	// IP refers to first I/P address of DFC Node. DFC instance node may have mulitple I/P
	// address but DFC will select first one.
	ip string

	// ID uniquely identifies a Proxy Client or Storage Server in DFC Cluster. It needs to
	// be unique . We currently use MAC id.
	id string

	// List of Usable mountpoints on storage server
	mntpath []MountPoint

	// TODO Need to expand

}

// Global context
var ctx = &ctxDfc{}

// Initialization
func dfcinit() {
	// CLI to override dfc JSON config
	var (
		role     string
		conffile string
		loglevel string
	)
	flag.StringVar(&role, "role", "", "role: proxy OR server")
	flag.StringVar(&conffile, "configfile", "", "config filename")
	flag.StringVar(&loglevel, "loglevel", "", "glog loglevel")

	flag.Parse()
	if conffile == "" {
		fmt.Fprintf(os.Stderr, "Usage: go run dfc role=<proxy|server> configfile=<somefile.json>\n")
		os.Exit(2)
	}
	if role != roleproxy && role != rolesetrver {
		fmt.Fprintf(os.Stderr, "Invalid role %q\n", role)
		fmt.Fprintf(os.Stderr, "Usage: go run dfc role=<proxy|server> configfile=<somefile.json>\n")
		os.Exit(2)
	}
	ctx.sig = make(chan os.Signal, 1)
	ctx.cancel = make(chan struct{})
	err := initconfigparam(conffile, loglevel, role)
	if err != nil {
		// Will exit process and  dump the stack
		glog.Fatalf("Failed to initialize, config %q err %v", conffile, err)
	}
	if role == roleproxy {
		ctx.proxy = true
		ctx.smap = make(map[string]serverinfo)
	}
}

// Init function initialize DFC Instance's Process Group.
func Init() (*ctxDfc, *group.Group, error) {
	var pool *group.Group
	var err error

	dfcinit()

	pool = new(group.Group)

	// Main daemon thread waiting in for loop for signal
	pool.Add(dstart, dstop)

	// Start webserver
	pool.Add(websrvstart, websrvstop)

	// Signal handler runnning as third worker
	pool.Add(sighandler, sigexit)
	return ctx, pool, err
}

// Run each process of DFC's instance pool.
func Run(pool *group.Group) {
	if glog.V(2) {
		glog.Infof("Run \n")
	}
	err := pool.Run()
	if err != nil {
		// Will exit process and dump Stack.
		glog.Fatalf("Failed to Run %v \n", err)
	}
}

// Stop DFC service (similar to user pressing CTRL-C)
func Stop(ctx *ctxDfc) {
	glog.Infof("Terminating on timeout")
	glog.Flush()
	close(ctx.cancel)
}

// TODO: empty
func dstart() error {
	for {
		select {
		case <-ctx.cancel:
			if glog.V(2) {
				glog.Info("dstart: got Cancel")
			}
			return nil
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
}

// Daemon exit function.
func dstop(err error) {
	glog.Infof("dstop: %v", err)

	// Not protecting it through mutex or atomic update for performance reason.
	// It will not cause any correctness issue.
	// Atmost some http new request may get submitted during stop process.
	ctx.stopinprogress = true
	glog.Flush()
	close(ctx.cancel)
}
