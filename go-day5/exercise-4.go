package main

import (
	"fmt"
	"time"
)

func exercise() {
	res := make(chan string)
	go func() {
		time.Sleep(500 * time.Millisecond)
		res <- "done"
	}()

	select { // usually takes 1 seond by default
	case r := <-res:
		fmt.Println(r)

	case <-time.After(time.Second):
		fmt.Println("timeout")
	}

}
