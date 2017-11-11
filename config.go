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

type dfcstring string

type Config struct {
	Proto            string `json:"proto"`
	Port             string `json:"port"`
	Cachedir         string `json:"cachedir"`
	Logdir           string `json:"logdir"`
	CloudProvider    string `json"cloudprovider"`
	Maxconcurrdownld uint32 `json"maxconcurrdownld"`
	Maxconcurrupld   uint32 `json"maxconcurrupld"`
	Maxpartsize      uint64 `json"maxpartsize"`
}

// Need to define structure for each cloud vendor like S3 , Azure, Cloud etc
// AWS S3 configurable parameters

// Configurable Parameters for Amazon S3
type S3configparam struct {
	// Concurrent Upload
	maxconupload uint32
	// Concurent Download
	maxcondownload uint32
	// Maximum part size
	maxpartsize uint64
}

// Configurable parameter for LRU cache
type Cacheparam struct {
	// HighwaterMark for free storage before flusher moves it to Cloud
	highwamark uint64
	// TODO
}

// Listner Port and Type for DFC service.It's constant aka non configurable.
type Listnerparam struct {
	// Prototype : tcp
	proto dfcstring
	// Listening port.
	port dfcstring
}

// Configurable parameters for DFC service
type ConfigParam struct {
	logdir        string
	cachedir      string
	lsparam       Listnerparam
	cloudprovider string
	s3config      S3configparam
}

func initconfigparam(configfile string) error {
	var err error
	conf := getConfig(configfile)
	// TODO ASSERT if conf is nil
	for _, config := range conf {

		flag.Lookup("log_dir").Value.Set(config.Logdir)
		ctx.configparam.logdir = config.Logdir
		ctx.configparam.cachedir = config.Cachedir
		ctx.configparam.lsparam.proto = dfcstring(config.Proto)
		ctx.configparam.lsparam.port = dfcstring(config.Port)
		ctx.configparam.cloudprovider = config.CloudProvider
		ctx.configparam.s3config.maxconupload = config.Maxconcurrupld
		ctx.configparam.s3config.maxcondownload = config.Maxconcurrdownld
		ctx.configparam.s3config.maxpartsize = config.Maxpartsize
		glog.Infof("Logdir = %s cachedir = %s proto =%s port = %s \n", config.Logdir,
			config.Cachedir, config.Proto, config.Port)
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
func getConfig(fpath string) []Config {
	raw, err := ioutil.ReadFile(fpath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// Currently supporting only one
	var c []Config
	json.Unmarshal(raw, &c)
	//glog.Infof("GetConfig: The json entry %v \n", c)
	return c
}
