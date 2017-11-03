// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

// Provides stats for DFC
type Stats struct {
	rdcachehit uint64
	rdtotal    uint64
	rdmaxsize  uint64
}

// Provides stats for DFC service
func Stat() Stats {
	glog.Info("Stat function \n")
	// TODO
	return ctx.stat
}
