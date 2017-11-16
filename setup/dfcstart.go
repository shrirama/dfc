package main

import (
	"fmt"
	"time"

	"github.com/shrirama/dfc"
)

func main() {
	//conf1 := os.Getenv("DFC_CONF1")
	Dctx1, pool1, err := dfc.Init()
	if err != nil {
		fmt.Println(" 1A: Error in DFC initialization ")
		return
	}
	go dfc.Run(pool1)
	// stop after 60 seconds
	time.Sleep(6000 * time.Second)
	dfc.Stop(ctx1)
}
