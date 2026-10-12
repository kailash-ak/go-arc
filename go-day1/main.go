package main

import (
	"fmt"
	"sync"
	"time"
)

func task(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Starting task", name)
	time.Sleep(2 * time.Second)
	fmt.Println("Completed task", name)
}
func main() {
	start := time.Now()
	var wg sync.WaitGroup
	//fmt.Println("hello, this is kailash")
	wg.Add(3)
	go task("Task 1", &wg)
	go task("Task 2", &wg)
	go task("Task 3", &wg)
	wg.Wait() // wait for all tasks to complete(go routines) starts the tasks and reaches the end of the main  funtion so it exits before the go routines complete their execution
	end := time.Now()
	fmt.Println("Total execution time:", end.Sub(start))
}
