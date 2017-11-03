// CopyRight Notice: All rights reserved
//
//

package dfc

import "github.com/golang/glog"

type dfcstring string

// Need to define structure for each cloud vendor like S3 , Azure, Cloud etc
// AWS S3 configurable parameters

// Configurable Parameters for Amazon S3
type S3configparam struct {
	// Concurrent Upload
	conupload int32
	// Concurent Download
	condownload int32
	// Maximum part size
	maxpartsize uint64
}

// Configurable parameter for LRU cache
type Cacheparam struct {
	// HighwaterMark for free storage before flusher moves it to Cloud
	highwamark uint64
	// TODO
}

// Configurable parameters for DFC service
type Configparam struct {
	s3config    S3configparam
	cacheconfig Cacheparam
}

// Destination Path for key(s) on host.It's configurable.
type S3 struct {
	localdir string
}

// Listner Port and Type for DFC service.It's constant aka non configurable.
type listner struct {
	proto dfcstring
	// Multiple ports are defined to test webserver listening to Multiple Ports for Testing.
	port1 dfcstring
	port2 dfcstring
}

func initconfigparam(ctx *dctx) {

	ctx.lsparam.proto = "tcp"
	ctx.lsparam.port1 = "8080"
	ctx.lsparam.port2 = "8081"

	// localdir is scratch space to download
	// It will be destination path for DFC use case.
	ctx.s3param.localdir = "/tmp/nvidia/"
}

// To enable configurable parameter for DFC
func Config(config Configparam) {
	// TODO
	glog.Info("Config function \n")
}
