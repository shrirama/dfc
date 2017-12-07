package main

import (
	"fmt"
	"time"

	"github.com/shrirama/dfc"
)

func main() {
	ctx1, pool1, err := dfc.Init()
	if err != nil {
		fmt.Printf("Failed to initialize DFC service, err %v\n", err)
		return
	}
	go dfc.Run(pool1)
	//
	// FIXME: stop after 10 min
	//
	time.Sleep(6000 * time.Second)
	dfc.Stop(ctx1)
}
