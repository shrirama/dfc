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
		err = registerwithproxy()
		if err != nil {
			glog.Errorf("Failed to parse mounts, err %v", err)
			return err
		}
		// Local mount points have precedence over cachePath settings.
		ctx.mntpath, err = parseProcMounts(procMountsPath)
		if err != nil {
			glog.Errorf("Failed to register with proxy, err %v", err)
			return err
		}

		glog.Infof("Num mp-s found %d", len(ctx.mntpath))
		if len(ctx.mntpath) == 0 {
			glog.Infof("Warning: zero mp-s", len(ctx.mntpath))

			// Use CachePath from config file if set.
			if ctx.config.Cache.CachePath == "" || ctx.config.Cache.CachePathCount < 1 {
				errstr := fmt.Sprintf("Invalid CachePath %q Insufficient CachePathCount %d",
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
		glog.Errorf("Failed to listen, portstring %s err %v", portstring, err)
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

		// Path will have following format
		// /<bucketname>/keypath
		s := strings.SplitN(html.EscapeString(r.URL.Path), fslash, s3skipTokenToKey)
		bktname := s[1]
		keyname := s[2]
		mpath := doHashfindMountPath(bktname + keyname)
		fname := mpath + fslash + bktname + fslash + keyname
		glog.Infof("Bucket %s key %s fqn %q", bktname, keyname, fname)

		// Check wheather filename exists in local directory or not
		_, err := os.Stat(fname)
		if os.IsNotExist(err) {
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
			}
		} else {
			glog.Infof("Bucket %s key %s already exists", bktname, keyname)
		}
		file, err := os.Open(fname)
		if err != nil {
			glog.Errorf("Failed to open file %q err %v", fname, err)
			checksetmounterror(fname)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			defer file.Close()

			// TODO Currently entire file is being downloaded before sending streaming
			// response to  http response. It's possible to do chunking, concurrency of
			// object without using downloader to stream chunks as soon as it lands on
			// local storage to http response(without waiting for entire file to download)
			// It would require multipart and concurrency implementation in DFC itself.
			_, err := io.Copy(w, file)
			if err != nil {
				glog.Errorf("Failed to copy data to http response for fname %q err %v\n", fname, err)
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
				glog.Errorf("Failed to create bucket dir %s err %v\n", dirname, err)
				return err
			}
		} else {
			glog.Errorf("Failed to fstat dir %q err %v\n", dirname, err)
			return err
		}
	}

	file, err = os.Create(fname)
	if err != nil {
		glog.Errorf("Unable to create file %q err %v\n", fname, err)
		checksetmounterror(fname)
		return err
	}
	bytes, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(kname),
	})
	if err != nil {
		glog.Errorf("Failed to download key %s from bucket %s err %v\n",
			kname, bucket, err)
		checksetmounterror(fname)
	} else {
		glog.Infof("Downloaded %q size = %d from bucket %s by key %s\n",
			file.Name(), bytes, bucket, kname)
	}
	return err
}

// Stop Http service .It waits for http outstanding requests to be completed
// before returning.
func websrvstop(err error) {
	glog.Infof("Stop http worker, err %v", err)

	// stop listening
	ctx.listener.Close()

	// Wait for the completion of all pending HTTP requests
	ctx.httprqwg.Wait()
	if !ctx.proxy {
		close(ctx.fschkchan)
	}
}
