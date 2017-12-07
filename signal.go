// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

// Custom sighandler for trapping signals to DFC process.
func sighandler() error {
	signal.Notify(ctx.sig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	s := <-ctx.sig
	switch s {
	// kill -SIGHUP XXXX
	case syscall.SIGHUP:
		return errors.New("Signal SIGHUP")
	// kill -SIGINT XXXX or Ctrl+c
	case syscall.SIGINT:
		return errors.New("Signal SIGINT")
	// kill -SIGTERM XXXX
	case syscall.SIGTERM:
		return errors.New("Signal SIGTERM")

	// kill -SIGQUIT XXXX
	case syscall.SIGQUIT:
		return errors.New("Signal SIGQUIT")
	default:
		glog.Errorln("Unknown Signal:", s)
	}
	return nil
}

// Exit function in context of signal
func sigexit(err error) {
	glog.Infof("The sighandler worker was interrupted with: %v\n", err)
	//lbderegister(err)
	os.Exit(2)
}
