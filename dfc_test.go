package dfc

import (
	"os"
	"testing"
	"time"
)

func TestInitRunStop(t *testing.T) {
	os.Args = []string{"/dfc_conf1.json"}
	err, ctx1, pool1 := Init()
	if err != nil {
		t.Errorf(" 1A: Error in DFC initialization ")
		return
	}
	go Run(pool1)
	// stop after 60 seconds
	time.Sleep(60 * time.Second)
	Stop(ctx1)
}

func TestConfig(t *testing.T) {
	// TODO
}

func TestStat(t *testing.T) {
	// TODO
}
