package main

import (
	"fmt"
	_ "net/http/pprof"
	"time"

	"github.com/shrirama/dfc"
)

func main() {
	ctx1, pool1, err := dfc.Init()
	if err != nil {
		fmt.Println(" 1A: Error in DFC initialization  %v \n", err)
		return
	}
	go dfc.Run(pool1)
	// stop after 60 seconds
	time.Sleep(6000 * time.Second)
	dfc.Stop(ctx1)
}
