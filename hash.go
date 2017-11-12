package dfc

import (
	"hash/crc32"

	"github.com/golang/glog"
)

// It will do hash on Path+Port+ID and will pick storage server with Max Hash value.

func doHashfindServer(url string) string {
	var sid string
	var max uint32
	for _, smap := range ctx.smap {
		glog.Infof("Id = %s Port = %s \n", smap.id, smap.port)
		cs := crc32.Checksum([]byte(url+smap.id+smap.port), crc32.IEEETable)
		if cs > max {
			max = cs
			sid = smap.id
		}
	}
	return sid
}
