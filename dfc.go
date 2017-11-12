// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/oklog/oklog/pkg/group"
)

type dctx struct {
	wg          sync.WaitGroup
	cancel      chan struct{}
	configparam ConfigParam
	// True will imply running as Proxy and False will imply Server
	proxy bool
	// Will be only populate for server.
	smap map[string]serverinfo

	// statics or histogram for dfc
	stat Stats

	sig chan os.Signal
}

// Server Registration info
type serverinfo struct {
	port string
	ip   string
	id   string // Should be Unique among all Nodes
	// TODO Need to expand
}

// Global context
var ctx *dctx

// Initialization
func init() {
	var stype string
	var conffile string

	flag.StringVar(&stype, "type", "", "a string var")
	flag.StringVar(&conffile, "configfile", "", "a string var")

	flag.Parse()
	// TODO Type should be either Proxy or Server.

	if conffile == "" || (stype != "proxy" && stype != "server") {
		fmt.Fprintf(os.Stderr, "Usage: go run dfc type=[proxy][server] configfile=name.json stype = %s \n", stype)
		os.Exit(2)
	}

	ctx = new(dctx)
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

// Initialize DFC
func Init() (error, *dctx, *group.Group) {
	var pool *group.Group
	var err error
	pool = new(group.Group)
	// TODO Registration with load balancer
	// pool.Add(lbregister, noopfunc)

	//		err = initconfigparam(conffile)
	//if err != nil {
	//	glog.Errorf("Failed to do initialization from config file err = %s \n", err)
	//	return err, nil, nil
	//}

	// Main daemon thread waiting in for loop for signal
	pool.Add(dstart, dstop)

	// Start webserver
	pool.Add(websrvstart, websrvstop)

	// Signal handler runnning as third worker
	pool.Add(sighandler, sigexit)
	return err, ctx, pool

}

// Start DFC Main worker thread
func Run(pool *group.Group) {

	glog.Infof("Run \n")
	pool.Run()
}

// It stops DFC service, similar to user pressing CTL-C or interrupt
func Stop(ctx *dctx) {
	glog.Infof(" Sending stop signal to DFC Main worker \n")
	close(ctx.cancel)
}

// Daemon thread running in for loop
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
	glog.Flush()
	close(ctx.cancel)
}
