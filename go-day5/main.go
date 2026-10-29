package main

import (
	"fmt"
	"time"
	"context"
)

func main(){
	ch:=make(chan int)// here we are creating a channel of type int and assigning it to the variable ch
	go func(){
		//val:=<-ch 
		 ch<-1// we are sending value 1 to the channel ch
		//fmt.Println(val)
	}()
	//ch<-1
		val:=<-ch // we are receiving value from the channel ch and assigning it to the variable val
		fmt.Println(val)
	
	// buffered channel 
	chane:=make(chan int,1)
	chane<-1
	va:=<-chane
	fmt.Println(va)

	//closed buffer we can recieve from closed channel but send to closed channel will cause panic
	dumchan:=make(chan int,1)
	close(dumchan)
	v,ok:=<-dumchan
	fmt.Println(v,ok)// here v is the value received from the channel and ok is a boolean value which indicates whether the channel is closed or not
	dummy()
	exercise()

	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()

	worker(ctx)

	ctx1, canc:=context.WithCancel(context.Background())
	go worker(ctx1) // here using go keyword it starts another go routine 
	time.Sleep(700*time.Millisecond)
	 canc()


}