// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

// Stats structure returns in-memory statistics information of a DFC instance.
type Stats struct {

	// Total count of key/object found locally
	rdcachehit uint64

	//Total count of key/object access
	rdtotal uint64

	//Maximum size of Object
	rdmaxsize uint64
}

// Stat function will provide in-memory statistics for local DFC instance.
// TODO Yet to be implemented.
func Stat() Stats {
	glog.Info("Stat function \n")
	// TODO
	return ctx.stat
}
