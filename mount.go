package dfc

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

type MountPoint struct {
	Device string
	Path   string
	Type   string
	Opts   []string
}

const (
	//
	dfcStoreMntPrefix    = "/mnt/dfcstore"
	dfcSignatureFileName = "/dfc.txt"
	// Number of fields per line in /proc/mounts as per the fstab man page.
	expectedNumFieldsPerLine = 6
	// Location of the mount file to use
	procMountsPath = "/proc/mounts"
)

func parseProcMounts(filename string) ([]MountPoint, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		glog.Fatalf("Failed to read from file %s err = %v \n", filename, err)
	}
	out := []MountPoint{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != expectedNumFieldsPerLine {
			glog.Errorf("Wrong number of fields (expected %d, got %d): %s \n",
				expectedNumFieldsPerLine, len(fields), line)
			continue
		}
		if checkdfcmntpath(fields[1]) {
			mp := MountPoint{
				Device: fields[0],
				Path:   fields[1],
				Type:   fields[2],
				Opts:   strings.Split(fields[3], ","),
			}

			out = append(out, mp)
		}
	}
	return out, nil
}

//dfcmntpath
func checkdfcmntpath(path string) bool {

	if strings.HasPrefix(path, dfcStoreMntPrefix) && checkdfcsignature(path) {
		return true
	} else {
		return false
	}
}

func checkdfcsignature(path string) bool {
	filename := path + dfcSignatureFileName
	_, err := os.Stat(filename)
	if err != nil {
		return false
	} else {
		return true
	}
}
