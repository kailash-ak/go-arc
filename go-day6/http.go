package main

import (
	"fmt"
	"net"

	//"time"
	"bufio"
	"io"
	"strings"
)

func dummy() {
	con, err2 := net.Dial("tcp", "google.com:80")
	if err2 != nil {
		fmt.Println("tcp connection failed", err2)
		return
	}
	fmt.Println("connection established")
	// here we use existing connection to send http request
	// i have known the diffrence between  http1  vs hhtp2
	//http1 ->it uses text like one way one request at a time if that request failed then subsequent requests also be blocked
	//http2-->it uses binary which is multiplexing which allows multiple requests and responses to be sent simultaneously over a single connection.

	// we will send this request
	//GET / HTTP/1.1
	//Host: example.com
	//Connection: close

	request := "GET / HTTP/1.1\r\n" +
		"Host: google.com\r\n" +
		"Connection: close\r\n\r\n"
	_, e := con.Write([]byte(request))
	if e != nil {
		fmt.Println("write failed", e)
		return
	}
	reader := bufio.NewReader(con)
	for {
		line, er := reader.ReadString('\n')
		if er != nil {
			fmt.Println("read failed", er)
			return
		}
		if line == "\r\n" {
			break
		}
		fmt.Println("HTTP Status:", strings.TrimRight(line, "\r\n"))
	}
	body, erb := io.ReadAll(reader)
	if erb != nil {
		fmt.Println("reading body failed", erb)
		return
	}
	fmt.Println(string(body))
	defer con.Close()

}
