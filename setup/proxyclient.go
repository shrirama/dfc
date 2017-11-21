// CopyRight Notice: All rights reserved
//
//

// Test Program to request data from Proxy
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var wg sync.WaitGroup

func main() {
	flag.Parse()
	for i := 0; i < 1; i++ {
		wg.Add(1)
		if i%3 == 0 {
			keyname := "/dir1/a" + strconv.Itoa(i)
			go getkey(keyname)
		} else if i%3 == 1 {
			keyname := "/dir2/a" + strconv.Itoa(i)
			go getkey(keyname)
		} else {
			keyname := "/dir3/a" + strconv.Itoa(i)
			go getkey(keyname)
		}
	}
	wg.Wait()
	//glog.Info("Completed main exiting \n")
	fmt.Printf("Completed main exiting \n")
}

func getkey(keyname string) {
	defer wg.Done()
	url := "http://localhost:" + "8080" + "/shri-new" + keyname
	fname := "/tmp/shri-new" + keyname
	// strips the last part from filepath
	dirname := filepath.Dir(fname)
	_, err := os.Stat(dirname)
	if err != nil {
		// Create bucket-path directory for non existent paths.
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				fmt.Printf("Failed to create bucket dir = %s err = %q \n", dirname, err)
				return
			}
		} else {
			fmt.Printf("Failed to do stat = %s err = %q \n", dirname, err)
			return
		}
	}

	file, err := os.Create(fname)
	if err != nil {
		fmt.Printf("Unable to create file = %s err = %q \n", fname, err)
		return
	}
	//glog.Infof(" URL = %s \n", url)
	fmt.Printf(" URL = %s \n", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to get key = %s err = %q", keyname, err)
	}
	defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	// io.Copy writes 32k at a time
	numBytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Failed to write to file err %q \n", err)
		panic(err)
	} else {
		fmt.Printf("Succesfully downloaded = %s and written = %d bytes \n",
			fname, numBytesWritten)
	}
}
