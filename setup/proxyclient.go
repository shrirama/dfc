// CopyRight Notice: All rights reserved
//
//

// Test Program to request data from Proxy
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
)

var wg sync.WaitGroup

func main() {
	flag.Parse()
	for i := 0; i < 200; i++ {
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
	//glog.Infof(" URL = %s \n", url)
	fmt.Printf(" URL = %s \n", url)
	resp, err := http.Get(url)
	if err != nil {
		//glog.Errorf("Failed to get key = %s err = %q", keyname, err)
		fmt.Printf("Failed to get key = %s err = %q", keyname, err)
		//panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	//glog.Infof(" URL = %s Response  = %s \n", url, body)
	fmt.Printf(" URL = %s Response  = %s \n", url, body)

}
