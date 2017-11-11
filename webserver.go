// CopyRight Notice: All rights reserved
//
//
package dfc

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

// Start first instance of webserver listening on port 8080
func websrvstart() error {
	// For server it needs to register with Proxy client before it can start
	if ctx.proxy == false {
		err := registerwithproxy()
		if err != nil {
			return err
		}
	}
	server8080 := http.NewServeMux()
	server8080.HandleFunc("/", httphdlr)
	portstring := ":" + ctx.configparam.lsparam.port
	ports := string(portstring)
	return http.ListenAndServe(ports, server8080)

}

// Function for handling request coming on specific port

func httphdlr(w http.ResponseWriter, r *http.Request) {

	glog.Infof("httphdlr Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)

	if ctx.proxy {
		proxyhdlr(w, r)
	} else {
		servhdlr(w, r)
	}
}

func proxyhdlr(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		// Serve the resource.
		// TODO Give proper error if no server is registered and client is requesting data
		// or may be directly get from S3??

	case "POST":
		//Proxy server may will get POST for  registration only
		// Need ParseForm method, to get data from form
		r.ParseForm()
		// attention: If you do not call ParseForm method, the following data can not be obtained form
		glog.Infof("request content %s  \n", r.Form) // print information on server side.
		var sinfo serverinfo
		for str, val := range r.Form {
			if str == "ip" {
				//glog.Infof(" str is = %s val = %s \n", str, val)
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.ip = strings.Join(val, "")
			}
			if str == "port" {
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.port = strings.Join(val, "")
			}

		}
		// Insert into Map based on Port
		ctx.smap[sinfo.port] = sinfo
		glog.Infof(" IP = %s port = %s curlen of map = %d \n", sinfo.ip, sinfo.port, len(ctx.smap))

	case "PUT":
		// Update an existing record.
	case "DELETE":
		// Remove the record.
	default:
		// Give an error message.
	}

	glog.Infof("Proxy Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
	fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))

}

func servhdlr(w http.ResponseWriter, r *http.Request) {

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

var (
	httpClient *http.Client
)

const (
	MaxIdleConnections int = 20
	RequestTimeout     int = 5
)

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

func registerwithproxy() error {
	httpClient = createHTTPClient()
	var err error
	// Proxy well known address
	proxyUrl := "http://localhost:8080"
	resource := "/"
	data := url.Values{}
	ipaddr := getipaddr()
	data.Set("ip", ipaddr)
	data.Add("port", string(ctx.configparam.lsparam.port))

	u, _ := url.ParseRequestURI(proxyUrl)
	u.Path = resource
	urlStr := u.String() // "http://api.com/user/"
	glog.Infof("urlStr : %s \n ", string(urlStr))
	glog.Infof("ipaddr : %s \n ", ipaddr)

	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode())) // <-- URL-encoded payload
	if err != nil {
		glog.Errorf("Error Occured. %+v", err)
		return err
	}
	req.Header.Add("Authorization", "auth_token=\"XXXXXXX\"")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// use httpClient to send request
	response, err := httpClient.Do(req)
	if err != nil && response == nil {
		glog.Errorf("Error sending request to Proxy server %+v \n", err)
		return err
	} else {
		// Close the connection to reuse it
		defer response.Body.Close()

		// Let's check if the work actually is done
		// We have seen inconsistencies even when we get 200 OK response
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			glog.Errorf("Couldn't parse response body. %+v \n", err)
		}

		glog.Infof("Response Body: \n", string(body))
	}
	return err
}
