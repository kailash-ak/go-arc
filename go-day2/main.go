package main

import (
	"fmt"
	"sync"
	"time"
)

func task(id int, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Task", id, "started")
	time.Sleep(1 * time.Second)
	fmt.Println("Task", id, "completed")
}
func main() {
	var wg sync.WaitGroup
	//var mu sync.Mutex
	start := time.Now()
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go task(i, &wg)
	}
	wg.Wait()
	end := time.Now()
	fmt.Println("All tasks completed in", end.Sub(start))
	counter := 0
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			//mu.Lock()
			counter++
			//mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Println("Counter:", counter)
	booking()
}
