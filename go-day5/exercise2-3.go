package main

import (
	"fmt"

)

func dummy(){
	ch:=make(chan int ,3)
	ch<-1
	ch<-2
	fmt.Println(len(ch),cap(ch))
	ch<-3
	close(ch)
	for v:= range ch{
		fmt.Println(v)
	}
	a,ok:=<-ch
	fmt.Println(a,ok)
}