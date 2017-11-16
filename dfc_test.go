// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"testing"
	"time"
)

func TestInitRunStop(t *testing.T) {
	Dctx1, pool1, err := Init()
	if err != nil {
		t.Errorf(" 1A: Error in DFC initialization ")
		return
	}
	go Run(pool1)
	// stop after 6000 seconds
	time.Sleep(6000 * time.Second)
	Stop(ctx1)
}

func TestConfig(t *testing.T) {
	// TODO
}

func TestStat(t *testing.T) {
	// TODO
}
