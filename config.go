// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
)

// dfconfig structure specifies Configuration Parameters for DFC Instance
// (Proxy Client or Storage Server) in JSON format.
// Config Parameters are specified during DFC service instantiation.
//  These Parameter overrides default paramemters.
// TODO Get and Set Config Parameter functionality/interface(s).
type dfconfig struct {
	ID            string       `json:"id"`
	Cachedir      string       `json:"cachedir"`
	Logdir        string       `json:"logdir"`
	Loglevel      string       `json:"loglevel"`
	CloudProvider string       `json:"cloudprovider"`
	Listen        listenconfig `json:"listen"`
	Proxy         proxyconfig  `json:"proxy"`
	S3            s3config     `json:"s3"`
}

// Need to define structure for each cloud vendor like S3 , Azure, Cloud etc
// AWS S3 configurable parameters

// s3config specifies  Amazon S3 specific configuration parameters.
type s3config struct {
	Maxconcurrdownld uint32 `json:"maxconcurrdownld"` // Concurent Download for a session.
	Maxconcurrupld   uint32 `json:"maxconcurrupld"`   // Concurrent Upload for a session.
	Maxpartsize      uint64 `json:"maxpartsize"`      // Maximum part size for Upload and Download used for buffering.
}

// listenconfig specifies listner Parameter for DFC instance.
// User can specify Port and Protocol(TCP/UDP) as part of listenconfig.
type listenconfig struct {
	Proto string `json:"proto"` // Prototype : tcp, udp
	Port  string `json:"port"`  // Listening port.
}

// proxyconfig specifies well-known address for http proxy as http://<ipaddress>:<portnumber>
type proxyconfig struct {
	URL      string `json:"url"`      // used to register caching servers
	Passthru bool   `json:"passthru"` // false: get then redirect, true (default): redirect right away
}

// Read JSON Config file and populate DFC Instance's config parameters.
// We currently support only one configuration per JSON file.
func initconfigparam(configfile, loglevel, role string) error {
	getConfig(configfile)

	err := flag.Lookup("log_dir").Value.Set(ctx.config.Logdir)
	if err != nil {
		// Not fatal as it will use default logfile under /tmp/
		glog.Errorf("Failed to set glog file name = %v \n", err)
	}

	if glog.V(3) {
		glog.Infof("Logdir = %s Cachedir = %s Proto =%s Port = %s ID = %s loglevel = %s \n",
			ctx.config.Logdir, ctx.config.Cachedir, ctx.config.Listen.Proto,
			ctx.config.Listen.Port, ctx.config.ID, ctx.config.Loglevel)
	}
	err = createdir(ctx.config.Logdir)
	if err != nil {
		glog.Errorf("Failed to create Logdir = %s err = %s \n", ctx.config.Logdir, err)
		return err
	}
	err = createdir(ctx.config.Cachedir)
	if err != nil {
		glog.Errorf("Failed to create Cachedir = %s err = %s \n", ctx.config.Cachedir, err)
		return err
	}
	// Argument specified at commandline or through flags has highest precedence.
	if loglevel != "" {
		err = flag.Lookup("v").Value.Set(loglevel)
	} else {
		err = flag.Lookup("v").Value.Set(ctx.config.Loglevel)
	}
	if err != nil {
		//  Not fatal as it will use default logging level
		glog.Errorf("Failed to set loglevel = %v \n", err)
	}

	glog.Infof("============== Log level: %s Config: %s Role: %s ==============\n",
		flag.Lookup("v").Value.String(), configfile, role)
	glog.Flush()
	return err
}

// Helper function to Create specified directory. It will also create complete path, not
// just short path.(similar to mkdir -p)
func createdir(dirname string) error {
	var err error
	_, err = os.Stat(dirname)
	if err != nil {
		// Create bucket-path directory for non existent paths.
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				glog.Errorf("Failed to create dir = %s err = %q \n", dirname, err)
			}
		} else {
			glog.Errorf("Failed to do stat = %s err = %q \n", dirname, err)
		}
	}
	return err
}

// Read JSON config file and unmarshal json content into config struct.
func getConfig(fpath string) {
	raw, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	err = json.Unmarshal(raw, &ctx.config)
	if err != nil {
		glog.Errorf("Failed to unmarshal JSON file = %s err = %v \n", fpath, err)
		os.Exit(1)
	}
}
