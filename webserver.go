// CopyRight Notice: All rights reserved
//
//
package dfc

import (
	"fmt"
	"html"
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

// Start first instance of webserver listening on port 8080
func websrvstart() error {

	server8080 := http.NewServeMux()
	server8080.HandleFunc("/", httphdlr)
	portstring := ":" + ctx.configparam.lsparam.port
	ports := string(portstring)
	return http.ListenAndServe(ports, server8080)

}

// Function for handling request coming on specific port

func httphdlr(w http.ResponseWriter, r *http.Request) {

	glog.Infof("httphdlr Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)

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
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		// Create S3 Downloader
		downloader := s3manager.NewDownloader(sess)
		ctx.wg.Add(1)
		// Channel to wait for completion of download
		cmpltchan := make(chan bool)
		go downloadkey(w, downloader, fname, bktname, keyname, cmpltchan)
		//wait for it to complete
		<-cmpltchan
		glog.Infof("httphdlr Bucket = %s Key =%s download completed \n", bktname, keyname)
	} else {
		glog.Infof("Bucket = %s Key =%s exist \n", bktname, keyname)
	}
	fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))

}

// Download  key from S3

func downloadkey(w http.ResponseWriter, downloader *s3manager.Downloader,
	fname string, bucket string, kname string, donechan chan bool) {

	defer ctx.wg.Done()

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
				goto err
			}
		} else {
			glog.Errorf("Failed to do stat = %s err = %q \n", dirname, err)
			goto err
		}
	}

	file, err = os.Create(fname)
	if err != nil {
		glog.Errorf("Unable to create file = %s err = %q \n", fname, err)
		goto err
	}
	//sleep for testing purpose only
	//time.Sleep(30 * time.Second)
	bytes, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(kname),
	})
	if err != nil {
		glog.Errorf("Failed to download key %q from bucket %q, %q",
			kname, bucket, err)
		goto err
	} else {
		donechan <- true
		glog.Infof("Successfully downloaded file %q size  = %d bytes \n",
			file.Name(), bytes)
		return
	}
err:
	http.Error(w, http.StatusText(http.StatusInternalServerError),
		http.StatusInternalServerError)
	donechan <- true

}

func websrvstop(err error) {
	glog.Infof("The NVWebServer worker was interrupted with: %v\n", err)
	// Wait for completion of all pending HTTP requests
	ctx.wg.Wait()
	//ctx.ln.Close()
}
