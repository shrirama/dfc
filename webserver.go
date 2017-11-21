// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"html"
	"io"
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

// Start instance of webserver listening on specific port.
func websrvstart() error {
	// For server it needs to register with Proxy client before it can start
	if !ctx.proxy {
		err := registerwithproxy()
		if err != nil {
			glog.Errorf("Hit Error %q", err)
			return err
		}
	}
	wbsvport := http.NewServeMux()
	wbsvport.HandleFunc("/", httphdlr)
	portstring := ":" + ctx.configparam.lsparam.port
	ports := string(portstring)
	return http.ListenAndServe(ports, wbsvport)

}

// Function for handling request  on specific port
func httphdlr(w http.ResponseWriter, r *http.Request) {

	glog.Infof("httphdlr Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)

	// Stop accepting new http request during Main daemon stop.
	if !ctx.stopinprogress {
		if ctx.proxy {
			proxyhdlr(w, r)
		} else {
			servhdlr(w, r)
		}
	} else {
		glog.Infof(" All daemons and handler are being stopped \n")
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

}

// Servhdlr function serves request coming to listening port of DFC's Storage Server.
// It supports GET method only and return 405 error non supported Methods.
// This function checks wheather key exists locally or not. If key does not exist locally
// it prepares session and download objects from S3 to path on local host.
func servhdlr(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		// Path will have following format
		// /<bucketname>/keypath
		s := strings.SplitN(html.EscapeString(r.URL.Path), "/", 3)
		bktname := s[1]
		keyname := s[2]
		glog.Infof("Bucket name = %s Key Name = %s \n", bktname, keyname)
		fname := ctx.configparam.cachedir + "/" + bktname + "/" + keyname
		glog.Infof("complete file name = %s \n", fname)

		// check wheather filename exists in local directory or not
		_, err := os.Stat(fname)
		if os.IsNotExist(err) {
			// TODO optimization to avoid creating sessions for each request.
			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))

			// Create S3 Downloader
			// TODO Optimize values for downloader options, it currently dowloads with 5MB chunk
			// and 5 concurrent downloads.
			downloader := s3manager.NewDownloader(sess)
			ctx.httprqwg.Add(1)

			err = downloadobject(w, downloader, fname, bktname, keyname)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
			} else {
				glog.Infof("httphdlr Bucket = %s Key =%s download completed \n", bktname, keyname)
				//fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))
			}
		} else {
			glog.Infof("Bucket = %s Key =%s exist \n", bktname, keyname)
			//fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))
		}
		file, err := os.Open(fname)
		if err != nil {
			glog.Errorf("Failed to open file %s err %v \n", fname, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			defer file.Close()
			_, err := io.Copy(w, file)
			if err != nil {
				glog.Errorf("Failed to Copy data to http response for fname %s err %v \n", fname, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				glog.Infof("Succefully copied file = %s to http response \n", fname)
			}
		}

	case "POST":
	case "PUT":
	case "DELETE":
	default:
		glog.Errorf("Invalid  Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)

	}

}

// This function download S3 object into local file.
func downloadobject(w http.ResponseWriter, downloader *s3manager.Downloader,
	fname string, bucket string, kname string) error {

	defer ctx.httprqwg.Done()

	var file *os.File
	var err error
	var bytes int64

	pathname := ctx.configparam.cachedir + "/" + bucket + "/" + kname
	// strips the last part from filepath
	dirname := filepath.Dir(pathname)
	_, err = os.Stat(dirname)
	if err != nil {
		// Create bucket-path directory for non existent paths.
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				glog.Errorf("Failed to create bucket dir = %s err = %q \n", dirname, err)
				return err
			}
		} else {
			glog.Errorf("Failed to do stat = %s err = %q \n", dirname, err)
			return err
		}
	}

	file, err = os.Create(fname)
	if err != nil {
		glog.Errorf("Unable to create file = %s err = %q \n", fname, err)
		return err
	}
	bytes, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(kname),
	})
	if err != nil {
		glog.Errorf("Failed to download key %q from bucket %q, %q",
			kname, bucket, err)
	} else {
		glog.Infof("Successfully downloaded file %q size  = %d bytes \n",
			file.Name(), bytes)
	}
	return err

}

// Stop Http service .It waits for http outstanding requests to be completed
// before returning.
func websrvstop(err error) {
	glog.Infof("The NVWebServer worker was interrupted with: %v\n", err)
	// Wait for completion of all pending HTTP requests
	ctx.httprqwg.Wait()
}
