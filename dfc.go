// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"flag"
	"os"
	"sync"

	"github.com/golang/glog"
	"github.com/oklog/oklog/pkg/group"
)

type dctx struct {
	wg     sync.WaitGroup
	cancel chan struct{}
	//TODO Make it generic to support Multiple cloud vendors
	s3param S3
	lsparam listner
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
	flag.Parse()
	ctx.cancel = make(chan struct{})
	initconfigparam(ctx)

}

// Initialize DFC
func Init() (error, *dctx, *group.Group) {
	var pool *group.Group
	var err error
	pool = new(group.Group)
	// TODO Registration with load balancer
	// pool.Add(lbregister, noopfunc)

	pool.Add(dstart, dstop)

	// Two HTTP server listening on different ports[8080 and 8081]
	// Its possible to do single one with multiple handler.
	pool.Add(websrv1start, websrvstop)
	pool.Add(websrv2start, websrvstop)

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
