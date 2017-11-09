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
	// statics or histogram for dfc
	stat Stats

	sig chan os.Signal
}

// Global context
var ctx *dctx

// Initialization
func init() {
	ctx = new(dctx)

	ctx.sig = make(chan os.Signal, 1)
	ctx.cancel = make(chan struct{})
	flag.Parse()
	flags := flag.Args()
	if len(flags) != 1 {
		fmt.Fprintf(os.Stderr, "Usage: go run dfc config-filename \n")
		os.Exit(2)
	}
	err := initconfigparam(flags[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to do initialization from config file err = %s \n", err)
		os.Exit(2)
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
