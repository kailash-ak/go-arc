package main

import (
	"bufio"
	"fmt"
	"io"
	"net"

	//"time
	"log"
	"strconv"
	"strings"
)

func handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	line1, err1 := reader.ReadString('\n')
	if err1 != nil {
		log.Println(err1)
		return
	}
	fmt.Println(line1)
	parts := strings.Fields(line1)
	fmt.Println("method1:", parts[0])
	fmt.Println("Path1:", parts[1])
	fmt.Println("Version:", parts[2])
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(line)
		if line == "\r\n" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			log.Println("Invalid header")
			return
		}
		name := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		headers[name] = value

	}

	contentLength, errs := strconv.Atoi(headers["content-length"])
	if errs != nil {
		log.Println("Invalid content length")
		return
	}
	body := make([]byte, contentLength)
	n, e := io.ReadFull(reader, body)
	if e != nil {
		log.Println(e)
		return
	}
	requestBody := string(body[:n])
	response := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
		"Content-Type: text/plain\r\n"+
		"Content-Length:%d\r\n"+
		"\r\n"+
		"%s",
		contentLength,
		requestBody)
	_, err1 = conn.Write([]byte(response))
	if err1 != nil {
		log.Println(err1)
		return
	}
}
func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Error listening:", err.Error())
	}
	defer ln.Close()
	fmt.Println("server Listening on :8080")
	conn, err1 := ln.Accept()
	if err1 != nil {
		fmt.Println("Error accepting connection:", err1.Error())
	}
	fmt.Println("Client connected")
	handle(conn)

}
