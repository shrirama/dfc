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
	// WaitGroup for completing Http Requests.
	httprqwg sync.WaitGroup

	// Channel for listening cancellation request.
	cancel chan struct{}

	// ConfigParameter for DFC instance.
	configparam ConfigParam

	// DFC can run as Proxy or Storage Server Instance..
	// True will imply running as Proxy and
	// False will imply running as Storage Server.
	proxy bool

	// Map of Registered storage servers with Proxy Instance. It will be NIL for Storage
	// Server Instance.
	smap map[string]serverinfo

	// Statistics or Histogram for DFC. It's  currently designed as in Memory Non Persistent
	// data structure to maintain histogram/statistic with respect to running DFC instance.
	stat Stats

	// Channel for  cancellation/termination signal.
	sig chan os.Signal

	// stopinprogress is set during main daemon thread stopping. DFC instance cannot
	// accept new http requests once stopinprogress is set.
	stopinprogess bool
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
	// TODO Need to expand
}

// Global context
var ctx *Dctx

// Initialization
func init() {
	var stype string
	var conffile string

	flag.StringVar(&stype, "type", "", "a string var")
	flag.StringVar(&conffile, "configfile", "", "a string var")

	flag.Parse()

	if conffile == "" || stype == "" {
		fmt.Fprintf(os.Stderr, "Usage: go run dfc type=[proxy][server] configfile=file.json \n")
		os.Exit(2)
	}
	if stype != "proxy" && stype != "server" {
		fmt.Fprintf(os.Stderr, "Invalid type = %s \n", stype)
		fmt.Fprintf(os.Stderr, "Usage: go run dfc type=[proxy][server] configfile=name.json \n")
		os.Exit(2)
	}

	ctx = new(Dctx)
	ctx.sig = make(chan os.Signal, 1)
	ctx.cancel = make(chan struct{})
	err := initconfigparam(conffile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to do initialization from config file = %s err = %s \n", conffile, err)
		os.Exit(2)
	}
	if stype == "proxy" {
		ctx.proxy = true
		ctx.smap = make(map[string]serverinfo)
	} else {
		ctx.proxy = false
	}

}

// Init function initialize DFC Instance's Process Group.
func Init() (*Dctx, *group.Group, error) {
	var pool *group.Group
	var err error
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

	glog.Infof("Run \n")
	pool.Run()
}

// Stop DFC instance. similar to user pressing CTL-C or interrupt
func Stop(ctx *Dctx) {
	glog.Infof(" Sending stop signal to DFC Main worker \n")
	close(ctx.cancel)
}

// Daemon thread running in for loop until receives cancel signal.
func dstart() error {
	// This worker keep running until cancel is called

	for {
		// Using select to have extendability for other cases
		select {
		case <-ctx.cancel:
			glog.Info("The Mainworker was canceled\n")
			return nil
		}
	}
}

// Daemon exit function.
func dstop(err error) {
	glog.Infof("The Mainworker was interrupted with: %v\n", err)

	// Not protecting it through mutex or atomic update for performance reason.
	// It will not cause any correctness issue.
	// Atmost some http new request may get submitted during stop process.
	ctx.stopinprogress = true
	glog.Flush()
	close(ctx.cancel)
}
