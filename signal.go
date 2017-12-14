// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

type signalError struct {
	sig syscall.Signal
}

func (se *signalError) Error() string {
	return fmt.Sprintf("Signal %d", se.sig)
}

//===========================================================================
//
// sig runner
//
//===========================================================================
type sigrunner struct {
	chsig chan os.Signal
}

// signal handler
func (r *sigrunner) run() error {
	r.chsig = make(chan os.Signal, 1)
	signal.Notify(r.chsig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	s := <-r.chsig
	signal.Stop(r.chsig) // stop immediately
	switch s {
	case syscall.SIGHUP: // kill -SIGHUP XXXX
		return &signalError{sig: syscall.SIGHUP}
	case syscall.SIGINT: // kill -SIGINT XXXX or Ctrl+c
		return &signalError{sig: syscall.SIGINT}
	case syscall.SIGTERM: // kill -SIGTERM XXXX
		return &signalError{sig: syscall.SIGTERM}
	case syscall.SIGQUIT: // kill -SIGQUIT XXXX
		return &signalError{sig: syscall.SIGQUIT}
	}
	return nil
}

func (r *sigrunner) stop(err error) {
	glog.Infof("Stopping sigrunner, err: %v", err)
	glog.Flush()
	signal.Stop(r.chsig)
	close(r.chsig)
}
