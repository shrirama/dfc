package dfc_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof" // profile
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"testing"
)

const (
	remroot = "/shri-new"
	locroot = "/iocopy"
)

func Test_ten(t *testing.T) {
	var wg = &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		keyname := "/dir" + strconv.Itoa(i%3+1) + "/a" + strconv.Itoa(i)
		go getAndCopyTmp(keyname, t, wg)
	}
	wg.Wait()
}

func getAndCopyTmp(keyname string, t *testing.T, wg *sync.WaitGroup) {
	defer wg.Done()
	url := "http://localhost:" + "8080" + remroot + keyname
	fname := "/tmp" + locroot + keyname
	dirname := filepath.Dir(fname)
	_, err := os.Stat(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
			if err != nil {
				t.Logf("Failed to create bucket dir = %s err = %q \n", dirname, err)
				return
			}
		} else {
			t.Logf("Failed to fstat, dir = %s err = %q \n", dirname, err)
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
	// write file locally
	defer resp.Body.Close()
	numBytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		t.Errorf("Failed to write to file err %q \n", err)
	} else {
		t.Logf("Succesfully downloaded = %s and written = %d bytes \n",
			fname, numBytesWritten)
	}
}

func Benchmark_one(b *testing.B) {
	var wg = &sync.WaitGroup{}
	for i := 0; i < 40; i++ {
		wg.Add(1)
		keyname := "/dir" + strconv.Itoa(i%3+1) + "/a" + strconv.Itoa(i)
		go get(keyname, b, wg)
	}
	wg.Wait()
}

func get(keyname string, b *testing.B, wg *sync.WaitGroup) {
	defer wg.Done()
	url := "http://localhost:" + "8080" + remroot + keyname
	resp, err := http.Get(url)
	if err != nil {
		if match, _ := regexp.MatchString("connection refused", err.Error()); match {
			fmt.Println("http connection refused - terminating")
			os.Exit(1)
		}
		fmt.Printf("Failed to get key = %s err = %q\n", keyname, err)
	}
	if resp == nil {
		return
	}
	ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}
