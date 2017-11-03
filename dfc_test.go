package dfc

import (
	"testing"
	"time"
)

func TestInitRunStop(t *testing.T) {
	err, ctx, pool := Init()
	if err != nil {
		t.Errorf(" error in DFC initialization ")
	}
	go Run(pool)
	// stop after 60 seconds
	time.Sleep(60 * time.Second)
	Stop(ctx)
}

func TestConfig(t *testing.T) {
	// TODO
}

func TestStat(t *testing.T) {
	// TODO
}
