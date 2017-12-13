// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

const (
	fslash           = "/"
	s3skipTokenToKey = 3
)

// Start instance of webserver listening on specific port.
func websrvstart() error {
	var err error
	// server must register with the proxy
	if !ctx.proxy {
		// Chanel for stopping filesystem check timer.
		ctx.fschkchan = make(chan bool)
		//
		// FIXME: UNREGISTER is totally missing
		//
		err = registerwithproxy()
		if err != nil {
			glog.Errorf("Failed to parse mounts, err: %v", err)
			return err
		}
		// Local mount points have precedence over cachePath settings.
		ctx.mntpath, err = parseProcMounts(procMountsPath)
		if err != nil {
			glog.Errorf("Failed to register with proxy, err: %v", err)
			return err
		}
		glog.Infof("Num mp-s found %d", len(ctx.mntpath))
		if len(ctx.mntpath) == 0 {
			glog.Infof("Warning: 0 (zero) mp-s")

			// Use CachePath from config file if set.
			if ctx.config.Cache.CachePath == "" || ctx.config.Cache.CachePathCount < 1 {
				errstr := fmt.Sprintf("Invalid configuration: CachePath %q or CachePathCount %d",
					ctx.config.Cache.CachePath, ctx.config.Cache.CachePathCount)
				glog.Error(errstr)
				err := errors.New(errstr)
				return err
			}
			ctx.mntpath = populateCachepathMounts()
		}
		// Start FScheck thread
		go fsCheckTimer(ctx.fschkchan)
	}
	httpmux := http.NewServeMux()
	httpmux.HandleFunc("/", httphdlr)
	portstring := ":" + ctx.config.Listen.Port

	ctx.listener, err = net.Listen("tcp", portstring)
	if err != nil {
		glog.Errorf("Failed to start listening on port %s, err: %v", portstring, err)
		return err
	}
	return http.Serve(ctx.listener, httpmux)

}

// Function for handling request  on specific port
func httphdlr(w http.ResponseWriter, r *http.Request) {
	if glog.V(1) {
		glog.Infof("HTTP request from %s: %s %q", r.RemoteAddr, r.Method, r.URL)
	}

	// Stop accepting new http request during Main daemon stop.
	if !ctx.stopinprogress {
		if ctx.proxy {
			proxyhdlr(w, r)
		} else {
			servhdlr(w, r)
		}
	} else {
		glog.Infof("Stopping...")
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

// Servhdlr function serves request coming to listening port of DFC's Storage Server.
// It supports GET method only and return 405 error for non supported Methods.
// This function checks wheather key exists locally or not. If key does not exist locally
// it prepares session and download objects from S3 to path on local host.
func servhdlr(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		atomic.AddInt64(&stats.numget, 1)
		// Expecting /<bucketname>/keypath
		s := strings.SplitN(html.EscapeString(r.URL.Path), fslash, s3skipTokenToKey)
		bktname := s[1]
		keyname := s[2]
		mpath := doHashfindMountPath(bktname + keyname)
		fname := mpath + fslash + bktname + fslash + keyname

		// Check wheather filename exists in local directory or not
		_, err := os.Stat(fname)
		if os.IsNotExist(err) {
			atomic.AddInt64(&stats.numnotcached, 1)
			glog.Infof("Bucket %s key %s fqn %q is not cached", bktname, keyname, fname)
			// TODO: avoid creating sessions for each request
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			// Create S3 Downloader
			// TODO: Optimize downloader options
			// (currently: 5MB chunks and 5 concurrent downloads)
			downloader := s3manager.NewDownloader(sess)
			ctx.httprqwg.Add(1)

			err = downloadobject(w, downloader, mpath, bktname, keyname)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				glog.Infof("Bucket %s key %s fqn %q downloaded", bktname, keyname, fname)
			}
		} else if glog.V(2) {
			glog.Infof("Bucket %s key %s fqn %q *is* cached", bktname, keyname, fname)
		}
		file, err := os.Open(fname)
		if err != nil {
			glog.Errorf("Failed to open file %q, err: %v", fname, err)
			checksetmounterror(fname)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			defer file.Close()

			// TODO: optimize. Currently the file gets downloaded and stored locally
			//       _prior_ to sending http response back to the requesting client
			_, err := io.Copy(w, file)
			if err != nil {
				glog.Errorf("Failed to copy data to http response for fname %q, err: %v", fname, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				glog.Infof("Copied %q to http response\n", fname)
			}
		}
	case "POST":
	case "PUT":
	case "DELETE":
	default:
		glog.Errorf("Invalid request from %s: %s %q", r.RemoteAddr, r.Method, r.URL)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed)+": "+r.Method,
			http.StatusMethodNotAllowed)

	}
	glog.Flush()
}

// This function download S3 object into local file.
func downloadobject(w http.ResponseWriter, downloader *s3manager.Downloader,
	mpath string, bucket string, kname string) error {

	defer ctx.httprqwg.Done()

	var file *os.File
	var err error
	var bytes int64

	//pathname := ctx.configparam.cachedir + "/" + bucket + "/" + kname
	fname := mpath + fslash + bucket + fslash + kname
	// strips the last part from filepath
	dirname := filepath.Dir(fname)
	_, err = os.Stat(dirname)
	if err != nil {
		// Create bucket-path directory for non existent paths.
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				glog.Errorf("Failed to create bucket dir %s, err: %v", dirname, err)
				return err
			}
		} else {
			glog.Errorf("Failed to fstat dir %q, err: %v", dirname, err)
			return err
		}
	}

	file, err = os.Create(fname)
	if err != nil {
		glog.Errorf("Unable to create file %q, err: %v", fname, err)
		checksetmounterror(fname)
		return err
	}
	bytes, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(kname),
	})
	if err != nil {
		glog.Errorf("Failed to download key %s from bucket %s, err: %v",
			kname, bucket, err)
		checksetmounterror(fname)
	} else {
		atomic.AddInt64(&stats.bytesloaded, bytes)
	}
	return err
}

// Stop Http service .It waits for http outstanding requests to be completed
// before returning.
func websrvstop(err error) {
	if ctx.listener == nil {
		return
	}
	glog.Infof("Stop http worker, err: %v", err)

	// stop listening
	ctx.listener.Close()

	// Wait for the completion of all pending HTTP requests
	ctx.httprqwg.Wait()
	if !ctx.proxy {
		close(ctx.fschkchan)
	}
}
