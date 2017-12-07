// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

// Stats structure returns in-memory statistics information of a DFC instance.
type stats struct {

	// Total count of key/object found locally
	rdcachehit uint64

	//Total count of key/object access
	rdtotal uint64

	//Maximum size of Object
	rdmaxsize uint64
}

// Getstat function will provide in-memory statistics for local DFC instance.
// TODO Yet to be implemented.
func Getstat() stats {
	if glog.V(4) {
		glog.Info("Stat function \n")
	}
	stat := ctx.stat
	// TODO
	return stat
}
