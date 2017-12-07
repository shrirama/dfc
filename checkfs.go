package dfc

import (
	"container/heap"
	"os"
	"path/filepath"
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

	desired := lwm - 15
	// FS is used less than LowWaterMark, nothing needs to be done.
	if (used * 100 / fs.Blocks) < uint64(lwm) {
		// Do nothing
		glog.Infof("Mntpath = %s currently used = %d LowWaterMark = %d \n",
			mntpath, used*100/fs.Blocks, lwm)
		return nil
	}
	// currently incoming rate of I/O request is not maintained.
	// It will delete more storage at higher usage , rather than fixed percentage.

	// Used blocks are even more than HighWater marks. It should not reach here
	// under normal scenario, need to be aggressive in deleting content.
	if used*100/fs.Blocks > uint64(hwm) {

		// Delete content until Used Filesystem becomes half of HighWaterMark.
		// if HighWaterMark was 80% , bring filesystem usage to be 40%
		desired = hwm / 2
	} else {
		// Delete 15% below LowWaterMark aka if LowWaterMark was 65%, make it
		// 50% and so on.

		desired = lwm - 15
	}
	desiredblks := fs.Blocks * uint64(desired) / 100
	tobedeletedblks := used - desiredblks
	bytestodel := tobedeletedblks * uint64(fs.Bsize)
	glog.Infof("Tobe freed blocks = %v  used blocks = %v desired used blocks = %v bytestodel = %v\n",
		tobedeletedblks, fs.Blocks, desiredblks, bytestodel)
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

	var curhsize, desiredhsize int64
	desiredhsize = int64(bytestodel)
	var maxatime time.Time
	var maxfo *FileObject
	var filecnt uint64
	for _, file := range fileList {
		filecnt++
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
			// Insert into heap until desiredheapsize
			if curhsize < desiredhsize {
				heap.Push(h, item)
				curhsize += stat.Size
				if glog.V(4) {
					glog.Infof(" 1A: curhsize  %v  currentpath = %s atime = %v \n ", curhsize, file, atime)
				}
				break
			}
			// Find Maxheap element for comparision with next set of incoming file object.
			maxfo = heap.Pop(h).(*FileObject)
			maxatime = maxfo.atime
			curhsize -= maxfo.size
			if glog.V(4) {
				glog.Infof("1B: curheapsize = %v len = %v \n", curhsize, h.Len())
			}

			// Push object into heap only if current fileobject's atime is lower than Maxheap element.
			if atime.Before(maxatime) {
				heap.Push(h, item)
				curhsize += stat.Size

				if glog.V(4) {
					glog.Infof("1C: curheapsize = %v len = %v \n", curhsize, h.Len())
				}

				// Get atime of Maxheap fileobject
				maxfo = heap.Pop(h).(*FileObject)
				curhsize -= maxfo.size
				if glog.V(4) {
					glog.Infof("1D: curheapsize = %v len = %v \n", curhsize, h.Len())
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
		glog.Infof("No of elements in heap = %v curhsize = %v  desiredhsize = %v filecnt = %v \n",
			heapelecnt, curhsize, desiredhsize, filecnt)
	}
	for heapelecnt > 0 && curhsize > 0 {
		maxfo = heap.Pop(h).(*FileObject)
		curhsize -= maxfo.size
		if glog.V(4) {
			glog.Infof("1E: curheapsize = %v len = %v \n", curhsize, h.Len())
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
