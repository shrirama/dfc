package dfc_test

import (
	"io"
	"net/http"
	_ "net/http/pprof" // profile
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
)

var wg sync.WaitGroup

func Test_3dirs(t *testing.T) {
	// profile go func() { t.Log(http.ListenAndServe("localhost:6060", nil)) }()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		if i%3 == 0 {
			keyname := "/dir1/a" + strconv.Itoa(i)
			go getkey(keyname, t)
		} else if i%3 == 1 {
			keyname := "/dir2/a" + strconv.Itoa(i)
			go getkey(keyname, t)
		} else {
			keyname := "/dir3/a" + strconv.Itoa(i)
			go getkey(keyname, t)
		}
	}
	wg.Wait()
}

func getkey(keyname string, t *testing.T) {
	defer wg.Done()
	url := "http://localhost:" + "8080" + "/shri-new" + keyname
	fname := "/tmp/shri-new" + keyname
	// strips the last part from the filepath
	dirname := filepath.Dir(fname)
	_, err := os.Stat(dirname)
	if err != nil {
		// Create bucket-path directory for non existent paths.
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				t.Logf("Failed to create bucket dir = %s err = %q \n", dirname, err)
				return
			}
		} else {
			t.Logf("Failed to do stat = %s err = %q \n", dirname, err)
			return
		}
	}

	file, err := os.Create(fname)
	if err != nil {
		t.Logf("Unable to create file = %s err = %q \n", fname, err)
		return
	}
	t.Logf(" URL = %s \n", url)
	resp, err := http.Get(url)
	if err != nil {
		if match, _ := regexp.MatchString("connection refused", err.Error()); match {
			t.Fatalf("http connection refused - terminating")
		}
		t.Logf("Failed to get key = %s err = %q", keyname, err)
	}
	if resp == nil {
		return
	}
	defer resp.Body.Close()
	// body, err := ioutil.ReadAll(resp.Body)
	// io.Copy writes 32k at a time
	numBytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		t.Errorf("Failed to write to file err %q \n", err)
	} else {
		t.Logf("Succesfully downloaded = %s and written = %d bytes \n",
			fname, numBytesWritten)
	}
}
