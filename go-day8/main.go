package main

import (
	"bufio"
	"fmt"
	"io"
	"net"

	"log"
	"strconv"
	"strings"
	"time"
)

func writeResponse(conn net.Conn, status string, contentType string, body string) {
	response := fmt.Sprintf("HTTP/1.1 %s\r\n"+
		"Content-Type: %s\r\n"+
		"Content-Length:%d\r\n"+
		"\r\n"+
		"%s",
		status,
		contentType,
		len(body),
		body)

	_, err1 := conn.Write([]byte(response))
	if err1 != nil {
		log.Println(err1)
		return
	}
}
func handle(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// 1. Read request line
	requestLine, err := reader.ReadString('\n')
	if err != nil {
		log.Println(err)
		return
	}

	parts := strings.Fields(requestLine)
	if len(parts) != 3 {
		log.Println("Invalid request line")
		return
	}

	method := parts[0]
	path := parts[1]
	version := parts[2]

	fmt.Println("Method:", method)
	fmt.Println("Path:", path)
	fmt.Println("Version:", version)

	// 2. Read headers
	headers := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		if line == "\r\n" {
			break
		}

		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) != 2 {
			log.Println("Invalid header")
			return
		}

		name := strings.ToLower(strings.TrimSpace(headerParts[0]))
		value := strings.TrimSpace(headerParts[1])

		headers[name] = value
	}

	// 3. Read request body if Content-Length exists
	var requestBody string

	contentLengthValue, exists := headers["content-length"]

	if !exists {
		if method == "POST" {
			writeResponse(conn, "411 Length Required",
				"text/plain", "Content length required")
			return
		}
	} else {
		contentLength, err := strconv.Atoi(contentLengthValue)
		if err != nil || contentLength < 0 || contentLength > 1024*1024 {
			writeResponse(conn, "400 Bad Request",
				"text/plain", "Invalid content length")
			return
		}

		body := make([]byte, contentLength)

		n, err := io.ReadFull(reader, body)
		if err != nil {
			log.Println("Request body read error:", err)
			return
		}

		requestBody = string(body[:n])
	}

	fmt.Println("Request Body:", requestBody)

	// 4. Routing
	switch {
	case path == "/" && method == "GET":
		writeResponse(conn, "200 OK", "text/plain", "home")

	case path == "/health" && method == "GET":
		writeResponse(conn, "200 OK", "application/json", `{"status":"ok"}`)

	case path == "/" || path == "/health":
		writeResponse(conn, "405 Method Not Allowed",
			"text/plain", "method not allowed")
	case method == "GET" && path == "/slow":
		time.Sleep(2 * time.Second)
		writeResponse(conn, "200 OK", "text/plain", "done")
	case method == "POST" && path == "/echo":
		writeResponse(conn, "200 OK", "text/plain", requestBody)
	default:
		writeResponse(conn, "404 Not Found",
			"text/plain", "not found")
	}
}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Error listening:", err.Error())
	}
	defer ln.Close()
	fmt.Println("server Listening on :8080")
	for {
		conn, err1 := ln.Accept()
		if err1 != nil {
			fmt.Println("Error accepting connection:", err1.Error())
		}

		fmt.Println("Client connected")
		go handle(conn) // concurrency
	}

}
