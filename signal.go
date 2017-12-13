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

// signal handler
func sighandler() error {
	chsig := make(chan os.Signal, 8)
	signal.Notify(chsig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	s := <-chsig
	switch s {
	case syscall.SIGHUP: // kill -SIGHUP XXXX
		return &signalError{sig: syscall.SIGHUP}
	case syscall.SIGINT: // kill -SIGINT XXXX or Ctrl+c
		return &signalError{sig: syscall.SIGINT}
	case syscall.SIGTERM: // kill -SIGTERM XXXX
		return &signalError{sig: syscall.SIGTERM}
	case syscall.SIGQUIT: // kill -SIGQUIT XXXX
		return &signalError{sig: syscall.SIGQUIT}
	default:
		glog.Errorln("Unknown Signal:", s)
	}
	return nil
}

// Exit function in context of signal
func sigexit(err error) {
	glog.Infof("sigexit called, err: %v", err)
}
