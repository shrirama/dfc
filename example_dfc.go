package dfc_test

import (
	"time"

	"github.com/golang/glog"
	"github.com/shrirama/dfc"
)

func ExampleDfc_start_stop() {
	glog.Infof(" Going to init DFC \n")

	ctx, pool := dfc.Init()

	glog.Infof(" Going to Run DFC \n")
	go pool.Run()

	glog.Infof("Running dfc service, waiting for 30 seconds \n")
	time.Sleep(60 * time.Second)
	glog.Infof("Attempting stopping dfc service \n")
	dfc.Stop(ctx)
	glog.Infof("Stopped dfc service \n")
}
