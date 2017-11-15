// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

// Provides stats for DFC
type Stats struct {

	// Total count of key/object found locally
	rdcachehit uint64

	//Total count of key/object access
	rdtotal uint64

	//Maximum size of Object
	rdmaxsize uint64
}

// Provides stats for DFC instance
func Stat() Stats {
	glog.Info("Stat function \n")
	// TODO
	return ctx.stat
}
