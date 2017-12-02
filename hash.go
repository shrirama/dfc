// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"hash/crc32"
	"math"

	"github.com/golang/glog"
)

// It will do hash on Normalized Path +Port+ ID and will pick storage server with Max Hash value.
func doHashfindServer(url string) string {
	var sid string
	var max uint32
	for _, smap := range ctx.smap {
		if glog.V(3) {
			glog.Infof("Id = %s Port = %s \n", smap.id, smap.port)
		}
		cs := crc32.Checksum([]byte(url+smap.id+smap.port), crc32.IEEETable)
		if cs > max {
			max = cs
			sid = smap.id
		}
	}
	return sid
}

// It will do hash on MountPath + bucket+ keypath and will pick mountpath with Min Hash value.
func doHashfindMountPath(key string) string {
	var mpath string
	var min uint32 = math.MaxUint32
	for _, minfo := range ctx.smap[ctx.configparam.ID].mntpath {
		if glog.V(3) {
			glog.Infof("mntpath = %s keypath = %s \n", minfo.Path, key)
		}
		cs := crc32.Checksum([]byte(key+minfo.Path), crc32.IEEETable)
		if cs < min {
			min = cs
			mpath = minfo.Path
		}
	}
	return mpath
}
