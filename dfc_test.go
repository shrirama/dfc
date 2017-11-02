package dfc

import (
	"errors"
	"testing"
	"time"
)

func TestInitRunStop(t *testing.T) {
	err, ctx, pool := dfc.Init()
	if err != nil {
		t.Errorf(" error in DFC initialization ")
	}
	go pool.Run()
	// stop after 60 seconds
	time.Sleep(60 * time.Second)
	dfc.Stop(ctx)
}

func TestConfig(t *testing.T) {
	// TODO
}

func TestStat(t *testing.T) {
	// TODO
}
