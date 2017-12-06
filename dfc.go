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
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/oklog/oklog/pkg/group"
)

// Dctx : DFC context is context for each DFC instance(Proxy or Storage Servers).
type Dctx struct {

	// Map of Registered storage servers with Proxy Instance. It will be NIL for Storage
	// Server Instance.
	smap map[string]serverinfo

	// Configuration
	config dfconfig

	// Statistics or Histogram for DFC. It's  currently designed as in Memory Non Persistent
	// data structure to maintain histogram/statistic with respect to running DFC instance.
	stat stats

	// Channel for  cancellation/termination signal.
	sig chan os.Signal

	// Channel for listening cancellation request.
	cancel chan struct{}

	// stopinprogress is set during main daemon thread stopping. DFC instance cannot
	// accept new http requests once stopinprogress is set.
	stopinprogress bool

	// DFC can run as Proxy or Storage Server Instance..
	// True will imply running as Proxy and
	// False will imply running as Storage Server.
	proxy bool

	// WaitGroup for completing Http Requests.
	httprqwg sync.WaitGroup
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
var ctx *Dctx

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
	flag.StringVar(&loglevel, "loglevel", "0", "glog loglevel")

	flag.Parse()
	if conffile == "" || role == "" {
		fmt.Fprintf(os.Stderr, "Usage: go run dfc role=<proxy|server> configfile=<somefile.json> \n")
		os.Exit(2)
	}
	if role != "proxy" && role != "server" {
		fmt.Fprintf(os.Stderr, "Invalid role = %s \n", role)
		fmt.Fprintf(os.Stderr, "Usage: go run dfc role=<proxy|server> configfile=<somefile.json> \n")
		os.Exit(2)
	}
	ctx = new(Dctx)
	ctx.sig = make(chan os.Signal, 1)
	ctx.cancel = make(chan struct{})
	err := initconfigparam(conffile, loglevel, role)
	if err != nil {
		// Will exit process and  dump the stack
		glog.Fatalf("Failed to initialize, config = %s err = %v \n", conffile, err)
	}
	if role == "proxy" {
		ctx.proxy = true
		ctx.smap = make(map[string]serverinfo)
	}
}

// Init function initialize DFC Instance's Process Group.
func Init() (*Dctx, *group.Group, error) {
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

// Stop DFC instance. similar to user pressing CTL-C or interrupt
func Stop(ctx *Dctx) {
	if glog.V(2) {
		glog.Infof(" Sending stop signal to DFC Main worker \n")
	}
	close(ctx.cancel)
}

// Daemon thread running in for loop until receives cancel signal.
func dstart() error {
	// This worker keep running until cancel is called
	for {
		// Using select to have extendability for other cases
		select {
		case <-ctx.cancel:
			if glog.V(2) {
				glog.Info("The Mainworker was canceled\n")
			}
			return nil
		}
	}
}

// Daemon exit function.
func dstop(err error) {
	if glog.V(2) {
		glog.Infof("The Mainworker was interrupted with: %v\n", err)
	}

	// Not protecting it through mutex or atomic update for performance reason.
	// It will not cause any correctness issue.
	// Atmost some http new request may get submitted during stop process.
	ctx.stopinprogress = true
	glog.Flush()
	close(ctx.cancel)
}
