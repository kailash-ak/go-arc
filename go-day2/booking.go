package main

import (
	"fmt"
	"sync"
	"time"
)

func booking() {
	var wg sync.WaitGroup
	//var mu sync.Mutex
	booked := false
	successes := 0
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			//mu.Lock()
			if !booked {
				//mu.Lock()
				time.Sleep(time.Millisecond)
				booked = true
				successes++
				//mu.Unlock()
			}
			//mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println("Successful bookings:", successes)
}
