// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

// Function for Registering to Load Balancer .
func lbregister() error {
	// TODO
	glog.Infof("Registering to the Load Balancer \n")
	return nil
}

// Dummy function for No Operations.
func noopfunc(err error) {
	// TODO
	glog.Infof("UnRegistering \n")
}

// Function for De-Registering from Load Balancer
func lbderegister(err error) {
	// TODO
	glog.Infof("UnRegistering error = %s\n", err)
}
