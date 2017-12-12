package dfc

import (
	"container/heap"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
)

func checkfs() {
	glog.Infof("checkfs entering \n")
	if ctx.checkfsrunning {
		glog.Infof("Already running checkfs, returning \n")
		return
	}
	ctx.checkfsrunning = true
	mntcnt := len(ctx.mntpath)
	glog.Infof("Number of mountpath = %d \n", mntcnt)
	for i := 0; i < mntcnt; i++ {
		ctx.fschkwg.Add(1)
		go fsscan(ctx.mntpath[i].Path)
	}
	// Wait for completion of scans on all mountpaths
	ctx.fschkwg.Wait()
	ctx.checkfsrunning = false
	return
}

func fsscan(mntpath string) error {
	defer ctx.fschkwg.Done()
	glog.Infof("fsscan for mntpath = %s \n", mntpath)
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(mntpath, &fs)
	if err != nil {
		glog.Errorf("Failed to statfs on mntpath = %s err = %v \n", mntpath, err)
		return err
	}
	glog.Infof(" Used Block = %v Free blocks = %v for mntpath = %s \n",
		fs.Blocks, fs.Bfree, mntpath)
	// in terms of block
	used := fs.Blocks - fs.Bfree
	hwm := ctx.config.Cache.FSHighWaterMark
	lwm := ctx.config.Cache.FSLowWaterMark

	// FS is used less than HighWaterMark, nothing needs to be done.
	if (used * 100 / fs.Blocks) < uint64(hwm) {
		// Do nothing
		glog.Infof("Mntpath = %s currently used = %d HighWaterMark = %d \n",
			mntpath, used*100/fs.Blocks, hwm)
		return nil
	}

	// if FileSystem's Used block are more than hwm(%), delete files to bring
	// FileSystem's Used block equal to lwm.
	desiredblks := fs.Blocks * uint64(lwm) / 100
	tobedeletedblks := used - desiredblks
	bytestodel := tobedeletedblks * uint64(fs.Bsize)
	glog.Infof("Currently Used blocks = %v Desired Used blocks = %v Tobe freed blocks = %v bytestodel = %v\n",
		fs.Blocks, desiredblks, tobedeletedblks, bytestodel)
	fileList := []string{}

	_ = filepath.Walk(mntpath, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})
	if err != nil {
		glog.Fatalf("Failed to traverse all files in dir = %s err = %v \n", mntpath, err)
		return err
	}
	h := &PriorityQueue{}
	heap.Init(h)

	var evictCurrBytes, evictDesiredBytes int64
	evictDesiredBytes = int64(bytestodel)
	var maxatime time.Time
	var maxfo *FileObject
	var filecnt uint64
	for _, file := range fileList {
		filecnt++

		// Skip special files starting with .
		if strings.HasPrefix(file, ".") {
			continue
		}
		fi, err := os.Stat(file)
		if err != nil {
			glog.Errorf("Failed to do stat on %s error = %v \n", file, err)
			continue
		}
		switch mode := fi.Mode(); {
		case mode.IsRegular():
			// do file stuff
			stat := fi.Sys().(*syscall.Stat_t)
			atime := time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
			item := &FileObject{
				path: file, size: stat.Size, atime: atime, index: 0}

			// Heapsize refers to total size of objects into heap.
			// Insert into heap until evictDesiredBytes
			if evictCurrBytes < evictDesiredBytes {
				heap.Push(h, item)
				evictCurrBytes += stat.Size
				if glog.V(4) {
					glog.Infof(" 1A: evictCurrBytes  %v  currentpath = %s atime = %v \n ", evictCurrBytes, file, atime)
				}
				break
			}
			// Find Maxheap element for comparision with next set of incoming file object.
			maxfo = heap.Pop(h).(*FileObject)
			maxatime = maxfo.atime
			evictCurrBytes -= maxfo.size
			if glog.V(4) {
				glog.Infof("1B: curheapsize = %v len = %v \n", evictCurrBytes, h.Len())
			}

			// Push object into heap only if current fileobject's atime is lower than Maxheap element.
			if atime.Before(maxatime) {
				heap.Push(h, item)
				evictCurrBytes += stat.Size

				if glog.V(4) {
					glog.Infof("1C: curheapsize = %v len = %v \n", evictCurrBytes, h.Len())
				}

				// Get atime of Maxheap fileobject
				maxfo = heap.Pop(h).(*FileObject)
				evictCurrBytes -= maxfo.size
				if glog.V(4) {
					glog.Infof("1D: curheapsize = %v len = %v \n", evictCurrBytes, h.Len())
				}
				maxatime = maxfo.atime
				if glog.V(4) {
					glog.Infof("1C: current path = %s atime = %v maxatime Maxpath = %s maxatime = %v \n",
						file, atime, maxfo.path, maxatime)
				}
			}

		case mode.IsDir():
			if glog.V(4) {
				glog.Infof("Skipping = %s due to being directory \n", file)
			}
			continue
		default:
			continue
		}

	}
	heapelecnt := h.Len()
	if glog.V(4) {
		glog.Infof("No of elements in heap = %v evictCurrBytes = %v  evictDesiredBytes = %v filecnt = %v \n",
			heapelecnt, evictCurrBytes, evictDesiredBytes, filecnt)
	}
	for heapelecnt > 0 && evictCurrBytes > 0 {
		maxfo = heap.Pop(h).(*FileObject)
		evictCurrBytes -= maxfo.size
		if glog.V(4) {
			glog.Infof("1E: curheapsize = %v len = %v \n", evictCurrBytes, h.Len())
		}
		heapelecnt--
		err := os.Remove(maxfo.path)
		if err != nil {
			glog.Errorf("Failed to delete file = %s err = %v \n", maxfo.path, err)
			continue
		}
	}
	return nil
}
