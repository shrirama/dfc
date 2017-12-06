package main

import (
	"fmt"
	"time"

	"github.com/shrirama/dfc"
)

func main() {
	ctx1, pool1, err := dfc.Init()
	if err != nil {
		fmt.Println("Error in DFC initialization %v \n", err)
		return
	}
	go dfc.Run(pool1)
	// stop after 10 min
	time.Sleep(6000 * time.Second)
	dfc.Stop(ctx1)
}
