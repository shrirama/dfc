// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
)

type dfcstring string

// Config structure specifies Configuration Parameters for DFC Instance
// (Proxy Client or Storage Server) in JSON format.
// Config Parameters are specified during DFC service instantiation.
//  These Parameter overrides default paramemters.
// TODO Get and Set Config Parameter functionality/interface(s).
type Config struct {
	Proto            string `json:"proto"`
	Port             string `json:"port"`
	ID               string `json:id`
	ProxyClientURL   string `json.proxyclienturl`
	Cachedir         string `json:"cachedir"`
	Logdir           string `json:"logdir"`
	CloudProvider    string `json"cloudprovider"`
	Maxconcurrdownld uint32 `json"maxconcurrdownld"`
	Maxconcurrupld   uint32 `json"maxconcurrupld"`
	Maxpartsize      uint64 `json"maxpartsize"`
}

// Need to define structure for each cloud vendor like S3 , Azure, Cloud etc
// AWS S3 configurable parameters

// S3configparam specifies  Amazon S3 specific configuration parameters.
type S3configparam struct {

	// Concurrent Upload for a session.
	maxconupload uint32

	// Concurent Download for a session.
	maxcondownload uint32

	// Maximum part size for Upload and Download. This size is used for buffering.
	maxpartsize uint64
}

// Cacheparam specifies parameters for LRU cache.
// TODO DFC can support different caching algorithm such as LRU, Most Frequently Used.
type Cacheparam struct {

	// HighwaterMark for free storage before flusher moves it to Cloud
	highwamark uint64

	// TODO
}

// Listnerparam specifies listner Parameter for DFC instance.
// User can specify Port and Protocol(TCP/UDP) as part of Listnerparam.
type Listnerparam struct {

	// Prototype : tcp, udp
	proto dfcstring

	// Listening port.
	port dfcstring
}

// Proxyclientparam specifies well known http address for Proxy Client.
// It is specified as http://<ipaddress>:<portnumber>
type Proxyclientparam struct {

	// ProxyClientURL is used by DFC' Storage Server Instances to
	// register with Proxy Client. It is specified as
	// http://[<ipaddr>][localhost]:<portnum>
	pclienturl dfcstring
}

// ConfigParam specifies configurable parameters for DFC instance.
// User specified configparams override default parameters.
type ConfigParam struct {

	// Logdir refers to Logdirectory for GLOG package to Print logs.
	logdir string

	// Cachedir refers to path on local host on which objects are cached as local file.
	cachedir string

	//ID need to be unique across all DFC instance.
	// Default ID will be MAC ID
	ID string

	// Pcparam refers to ProxyClientURL.DFC's storage instance uses this URL to register
	// with DFC's ProxyClient. DFC can support multiple ProxyClientURL across
	// DFC Cluster to do load balancing but we are currently supporting only one.
	pcparam Proxyclientparam

	//Lsparam refers to Listening Parametrs (Port and Protocol)
	lsparam Listnerparam

	// Cloudprovider refers to different Cloud Providers/services.
	// DFC currently supports only amazon S3. It's possible to do authentication, optimization
	// based on backend cloud provider. (Currently not Used)
	cloudprovider string

	// S3config refers to S3 Configurable Parameters.
	s3config S3configparam
}

// Read JSON Config file and populate DFC Instance's config parameters.
// We currently support only one configuration per JSON file.
func initconfigparam(configfile string) error {
	var err error
	conf := getConfig(configfile)
	if len(conf) != 1 {
		errstr := fmt.Sprintf("Configuration data length is %d, Needed 1 \n", len(conf))
		glog.Errorf(errstr)
		return errors.New(errstr)
	}
	for _, config := range conf {

		flag.Lookup("log_dir").Value.Set(config.Logdir)
		ctx.configparam.logdir = config.Logdir
		ctx.configparam.cachedir = config.Cachedir
		ctx.configparam.pcparam.pclienturl = dfcstring(config.ProxyClientURL)
		ctx.configparam.lsparam.proto = dfcstring(config.Proto)
		ctx.configparam.lsparam.port = dfcstring(config.Port)
		ctx.configparam.ID = config.ID
		ctx.configparam.cloudprovider = config.CloudProvider
		ctx.configparam.s3config.maxconupload = config.Maxconcurrupld
		ctx.configparam.s3config.maxcondownload = config.Maxconcurrdownld
		ctx.configparam.s3config.maxpartsize = config.Maxpartsize
		glog.Infof("Logdir = %s Cachedir = %s Proto =%s Port = %s ID = %s \n", config.Logdir,
			config.Cachedir, config.Proto, config.Port, config.ID)
		err = createdir(config.Logdir)
		if err != nil {
			glog.Errorf("Failed to create Logdir = %s err = %s \n", config.Logdir, err)
			return err
		}
		err := createdir(config.Cachedir)
		if err != nil {
			glog.Errorf("Failed to create Cachedir = %s err = %s \n", config.Cachedir, err)
			return err
		}
	}
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
func getConfig(fpath string) []Config {
	raw, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var c []Config
	json.Unmarshal(raw, &c)
	//glog.Infof("GetConfig: The json entry %v \n", c)
	return c
}
