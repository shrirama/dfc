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
		glog.Info("hungup")
		return errors.New("Received SIGHUP, Cancelling")

	// kill -SIGINT XXXX or Ctrl+c
	case syscall.SIGINT:
		glog.Info("GOT SIGINT")
		return errors.New("Received SIGINT, Cancelling")
	// kill -SIGTERM XXXX
	case syscall.SIGTERM:
		glog.Info("Force Stop")
		return errors.New("Received SIGTERM, Cancelling")

	// kill -SIGQUIT XXXX
	case syscall.SIGQUIT:
		glog.Info("Stop and Core dump")
		return errors.New("Received SIGQUIT, Cancelling")
	default:
		glog.Info("Unknown Signal")
		return errors.New("Received Unknown signal, Cancelling")
	}
}

// Exit function in context of signal
func sigexit(err error) {
	glog.Infof("The sighandler worker was interrupted with: %v\n", err)
	//lbderegister(err)
	os.Exit(2)
}
