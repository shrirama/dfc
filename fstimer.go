package dfc

import (
	"time"

	"github.com/golang/glog"
)

func fsCheckTimer(quit chan bool) {
	wttime := ctx.config.Cache.FSCheckfreq
	freq := time.Duration(wttime * 60)
	glog.Info("fsChecktimer entering \n")
	ticker := time.NewTicker(freq * time.Second)
	for {
		select {
		case <-ticker.C:
			if glog.V(2) {
				glog.Infof(" Going to Run FSCheck for purging old data \n")
			}
			if !ctx.checkfsrunning {
				checkfs()
			} else {
				glog.Infof(" Already Running FSCheck for purging old data \n")
			}
		case <-quit:
			glog.Infof("Received stop signal for timer thread \n")
			ticker.Stop()
		}
	}
}
