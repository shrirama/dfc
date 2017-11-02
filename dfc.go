// TODO Make it dfc package.
package dfc

import (
	"errors"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
	"github.com/oklog/oklog/pkg/group"
)

// Configurable Parameters for Amazon S3
type S3configparam struct {
	// Concurrent Upload
	conupload int32
	// Concurent Download
	condownload int32
	// Maximum part size
	maxpartsize uint64
}

// Configurable parameter for LRU cache
type Cacheparam struct {
	// HighwaterMark for free storage before flusher moves it to Cloud
	highwamark uint64
	// TODO
}

// Configurable parameters for DFC service
type Configparam struct {
	s3config    S3configparam
	cacheconfig Cacheparam
}

// Provides stats for DFC/
type Stats struct {
	rdcachehit uint64
	rdtotal    uint64
	rdmaxsize  uint64
}

// Need to define structure for each cloud vendor like S3 , Azure, Cloud etc
// AWS S3 configurable parameters
type S3 struct {
	localdir string
}

// Listner Port and Type
type listner struct {
	proto string
	// Multiple ports are defined to test webserver listening to Multiple Ports for Testing.
	port1 string
	port2 string
}

// Global Context
type dctx struct {
	wg     sync.WaitGroup
	cancel chan struct{}
	//TODO Make it generic to support Multiple cloud vendors
	s3param S3
	lsparam listner
	// statics or histogram for dfc
	stat Stats

	sig chan os.Signal
}

// Global context
var ctx *dctx

// Initialization
func init() {
	ctx = new(dctx)

	ctx.sig = make(chan os.Signal, 1)
	// TODO  Get type and port from config
	ctx.lsparam.proto = "tcp"

	ctx.lsparam.port1 = "8080"
	ctx.lsparam.port2 = "8081"

	// localdir is scratch space to download
	// It will be destination path for DFC use case.
	ctx.s3param.localdir = "/tmp/nvidia/"

	flag.Parse()
	ctx.cancel = make(chan struct{})

}

// Init DFC on Node.
func Init() (error, *dctx, *group.Group) {
	var pool *group.Group
	var err error
	pool = new(group.Group)
	// TODO Registration with load balancer
	// pool.Add(lbregister, noopfunc)

	pool.Add(dstart, dstop)

	// Two HTTP server listening on different ports[8080 and 8081]
	// Its possible to do single one with multiple handler.
	pool.Add(websrv1start, websrvstop)
	pool.Add(websrv2start, websrvstop)

	// Signal handler runnning as third worker
	pool.Add(sighandler, sigexit)
	return err, ctx, pool

}

// Start DFC Main worker thread
func Run(pool *group.Group) {
	//panic(pool == nil)

	glog.Infof("Run \n")
	pool.Run()
}

// It stops DFC service, similar to user pressing CTL-C or interrupt
func Stop(ctx *dctx) {
	//panic(ctx == nil)
	glog.Infof(" Sending stop signal to DFC Main worker \n")
	close(ctx.cancel)
}

// To enable configurable parameter for DFC
func Config(config Configparam) {
	// TODO
}

// Provides stats for DFC service
func Stat() Stats {

	// TODO
	return ctx.stat
}

func dstart() error {
	// This worker keep running until cancel is called

	for {
		// Using select to have extendability for other cases
		select {
		case <-ctx.cancel:
			glog.Info("The Mainworker was canceled\n")
			return nil
		}
	}
}

func dstop(err error) {
	glog.Infof("The Mainworker was interrupted with: %v\n", err)
	glog.Flush()
	close(ctx.cancel)
}

// Function for Registering to Load Balancer .
func lbregister() error {
	// TODO
	glog.Infof("Registering to the Load Balancer \n")
	return nil
}

// Dummy function for No Operations.
func noopfunc(err error) {
	// TODO
	glog.Infof("UnRegistering \n")
}

// Function for De-Registering
func lbderegister(err error) {
	// TODO
	glog.Infof("UnRegistering error = %s\n", err)
}

func websrv1start() error {

	server8080 := http.NewServeMux()
	server8080.HandleFunc("/", httphdr1)
	portstring := ":" + ctx.lsparam.port1
	return http.ListenAndServe(portstring, server8080)

}

func websrv2start() error {
	server8081 := http.NewServeMux()
	server8081.HandleFunc("/", httphdr1)
	portstring := ":" + ctx.lsparam.port2
	// nil will use Default ServeMux
	return http.ListenAndServe(portstring, server8081)

}

func httphdr1(w http.ResponseWriter, r *http.Request) {

	glog.Infof("httphdr1 Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)

	// Path will have following format
	// /<bucketname>/keypath
	s := strings.SplitN(html.EscapeString(r.URL.Path), "/", 3)
	bktname := s[1]
	keyname := s[2]
	glog.Infof("Bucket name = %s Key Name = %s \n", bktname, keyname)
	fname := ctx.s3param.localdir + bktname + "/" + keyname
	glog.Infof("complete file name = %s \n", fname)
	//check wheather filename exists in local directory or not
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
		glog.Infof("httphdr1 Bucket = %s Key =%s download completed \n", bktname, keyname)
	} else {
		glog.Infof("Bucket = %s Key =%s exist \n", bktname, keyname)
	}
	fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))

}

func downloadkey(w http.ResponseWriter, downloader *s3manager.Downloader,
	fname string, bucket string, kname string, donechan chan bool) {

	defer ctx.wg.Done()

	var file *os.File
	var err error
	var bytes int64

	dirname := ctx.s3param.localdir + bucket
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

func sighandler() error {
	signal.Notify(ctx.sig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	s := <-ctx.sig
	switch s {
	// kill -SIGHUP XXXX
	case syscall.SIGHUP:
		glog.Info("hungup")
		return errors.New("Received SIGHUP, Cancelling")

	// kill -SIGINT XXXX or Ctrl+c
	case syscall.SIGINT:
		glog.Info("GOT SIGINT")
		return errors.New("Received SIGINT, Cancelling")
	// kill -SIGTERM XXXX
	case syscall.SIGTERM:
		glog.Info("Force Stop")
		return errors.New("Received SIGTERM, Cancelling")

	// kill -SIGQUIT XXXX
	case syscall.SIGQUIT:
		glog.Info("Stop and Core dump")
		return errors.New("Received SIGQUIT, Cancelling")
	default:
		glog.Info("Unknown Signal")
		return errors.New("Received Unknown signal, Cancelling")
	}
}

func sigexit(err error) {
	glog.Infof("The sighandler worker was interrupted with: %v\n", err)
	//lbderegister(err)
	os.Exit(2)
}
