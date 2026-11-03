# Go Day 8 — Building a Raw TCP HTTP Server

## Goal

Build a small HTTP-like server directly on top of Go's `net` package. The purpose is to understand what happens underneath `net/http`: TCP listeners, client connections, byte streams, HTTP request parsing, response formatting, routing, request bodies, and concurrent handlers.

> This is an educational server, not a production-ready HTTP implementation. Real HTTP servers should generally use Go's `net/http` package, which handles protocol details and edge cases correctly.

## 1. Starting a TCP server

```go
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    log.Fatal(err)
}
defer ln.Close()
```

- `net.Listen("tcp", ":8080")` creates a TCP listener on port `8080`.
- `:8080` means listen on port 8080 on the available local network interfaces.
- The returned `net.Listener` waits for incoming TCP connections.
- `Listen` does not accept a client connection. `Accept()` does that.
- `defer ln.Close()` schedules the listener to close when the function returns. It stops new connections from being accepted.

A listener and a connection are different resources:

| Resource | Go type | Responsibility |
|---|---|---|
| Listening socket | `net.Listener` | Waits for new connections |
| Client connection | `net.Conn` | Communicates with one connected client |

## 2. Accepting a client connection

```go
conn, err := ln.Accept()
if err != nil {
    log.Println("accept error:", err)
    return
}
defer conn.Close()
```

- `Accept()` blocks until a client establishes a TCP connection or an error occurs.
- It returns a `net.Conn` representing that particular client connection.
- A client can establish a TCP connection without immediately sending an HTTP request.
- `conn.Close()` closes that one client connection. It does not close the listener.
- `defer conn.Close()` schedules cleanup when the handler returns; it does not close the connection immediately.

The listener can accept many connections over its lifetime. Each accepted connection should be handled and eventually closed.

## 3. TCP is a byte stream

TCP delivers an ordered stream of bytes. It does not understand HTTP requests, headers, or messages.

For example, the client might send:

```text
GET /hello HTTP/1.1\r\n
Host: localhost:8080\r\n
\r\n
```

The server is not guaranteed to receive all these bytes in one `Read` call. Data may arrive in several pieces, or multiple pieces may already be buffered together.

```go
n, err := conn.Read(buf)
// n is not necessarily the size of one HTTP request.
```

`Read` reports how many bytes were read in that particular call. HTTP parsing must use the protocol's framing rules.

## 4. Buffering and reading HTTP lines

```go
reader := bufio.NewReader(conn)

line, err := reader.ReadString('\n')
if err != nil {
    return
}
```

- `bufio.NewReader` wraps the connection with a buffered reader.
- `ReadString('\n')` reads through the next newline character.
- HTTP/1.x lines end in CRLF: carriage return plus line feed (`\r\n`).
- The returned string includes the delimiter, so the blank line separating headers from the body is `"\r\n"`.

HTTP headers end at the first empty line. In byte form, the separator is:

```text
\r\n\r\n
```

**Important:** Reuse the same `bufio.Reader` when reading the body. It may already have read body bytes from the connection into its internal buffer while reading the headers. Reading directly from `conn` afterward can skip bytes still buffered in `reader`.

## 5. Parsing the request line

An HTTP/1.x request line has this structure:

```text
METHOD SP REQUEST-TARGET SP HTTP-VERSION CRLF
```

Example:

```text
GET /hello HTTP/1.1
```

```go
parts := strings.Fields(line)
if len(parts) != 3 {
    // malformed request line
    return
}

method := parts[0]
path := parts[1]
version := parts[2]
```

`strings.Fields` splits on runs of whitespace and avoids empty elements caused by repeated spaces. Validate that there are exactly three fields before using them.

Keep `method`, `path`, and `version` in separate variables. Do not reuse the request-line `parts` variable for header parsing if you still need those values.

## 6. Parsing headers

A header has this general form:

```text
Header-Name: Header-Value
```

Example:

```text
Host: localhost:8080
Content-Length: 12
User-Agent: curl/...
```

```go
parts := strings.SplitN(headerLine, ":", 2)
if len(parts) != 2 {
    // malformed header
    return
}

name := strings.ToLower(strings.TrimSpace(parts[0]))
value := strings.TrimSpace(parts[1])
headers[name] = value
```

- `SplitN(..., 2)` splits at the first colon only. Header values can themselves contain colons.
- Header names are case-insensitive, so normalizing names to lowercase makes lookups predictable.
- `strings.TrimSpace` removes surrounding whitespace.

Use a comma-ok map lookup:

```go
value, exists := headers["content-length"]
```

- `exists == true`: the header is present.
- `exists == false`: the header is absent.

This differs from checking whether `value` is an empty string.

## 7. Understanding request-body framing

The blank line ends the headers; it does not necessarily end the entire HTTP request.

Common HTTP/1.1 framing mechanisms include:

- `Content-Length`: the body contains the declared number of bytes.
- `Transfer-Encoding: chunked`: the body is sent in chunks with chunk framing.
- Some message types, particularly certain responses, can be delimited by closing the connection.

This exercise implements the simple `Content-Length` case. It does not implement chunked transfer encoding or all HTTP message-framing rules.

## 8. Reading a body with Content-Length

```go
contentLength, err := strconv.Atoi(contentLengthValue)
if err != nil || contentLength < 0 {
    // invalid length
    return
}

body := make([]byte, contentLength)
n, err := io.ReadFull(reader, body)
if err != nil {
    // body was incomplete or the connection ended
    return
}

requestBody := string(body[:n])
```

- `strconv.Atoi` converts decimal text into an integer.
- `make([]byte, contentLength)` allocates a byte slice of the requested size.
- `io.ReadFull` keeps reading until the requested number of bytes has been read or an error occurs.
- `n` is the number of bytes actually read.
- `body[:n]` contains only the bytes received.

If the client declares 12 bytes but sends only 5 and closes the connection, `io.ReadFull` returns an error (typically `io.ErrUnexpectedEOF`) and reports how many bytes it received.

A real server must impose a maximum body size. Otherwise, a client could declare an enormous `Content-Length` and cause excessive memory allocation. The exercise used a 1 MiB limit as a simple safeguard.

### Missing Content-Length in this exercise

The exercise used a simplified rule:

- A `GET` request without `Content-Length` is treated as having no request body.
- A `POST` request without `Content-Length` receives `411 Length Required`.

This is an exercise-specific simplification, not a complete implementation of HTTP framing.

## 9. Building an HTTP response

An HTTP/1.x response has this structure:

```text
HTTP/VERSION STATUS-CODE REASON-PHRASE\r\n
Header-Name: Header-Value\r\n
\r\n
BODY
```

Example:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 5

hello
```

A helper used in the exercise:

```go
func writeResponse(conn net.Conn, status string, contentType string, body string) {
    response := fmt.Sprintf(
        "HTTP/1.1 %s\r\n"+
            "Content-Type: %s\r\n"+
            "Content-Length: %d\r\n"+
            "\r\n"+
            "%s",
        status,
        contentType,
        len(body),
        body,
    )

    _, err := conn.Write([]byte(response))
    if err != nil {
        log.Println("write response:", err)
    }
}
```

Important details:

- Every HTTP/1.x line ends with `\r\n`.
- The blank line separates headers from the response body.
- `Content-Length` describes the response body's size in **bytes**, not characters and not the TCP buffer size.
- In Go, `len(string)` returns the number of bytes in the string, which is appropriate for this header.
- A robust implementation must handle possible partial writes. For real applications, prefer `net/http` over hand-written protocol handling.

## 10. Routing requests

Routing chooses application behavior based on the request method and path.

| Method | Path | Expected response |
|---|---|---|
| `GET` | `/` | `200 OK`, body `home` |
| `GET` | `/health` | `200 OK`, JSON health response |
| `GET` | `/slow` | Wait 2 seconds, then `200 OK`, body `done` |
| Any | Unknown path | `404 Not Found` |
| Unsupported method on a known path | Known path | `405 Method Not Allowed` |

Status-code distinctions:

- `200 OK`: request succeeded.
- `400 Bad Request`: request syntax or input is invalid.
- `404 Not Found`: the requested route does not exist.
- `405 Method Not Allowed`: the route exists, but that HTTP method is not supported. A real response should include an `Allow` header, such as `Allow: GET`.
- `411 Length Required`: the exercise's response when a POST request has no Content-Length.
- `500 Internal Server Error`: an unexpected server-side failure.
- `503 Service Unavailable`: the server cannot currently handle the request.

Distinguish whether a path exists from whether its method is supported. Otherwise, an unsupported method on an unknown path can accidentally be reported as `405` instead of `404`.

## 11. Concurrency with goroutines

A sequential server that accepts one connection and handles it before accepting the next can make other clients wait.

The concurrent accept loop:

```go
for {
    conn, err := ln.Accept()
    if err != nil {
        log.Println("accept error:", err)
        continue
    }

    go handle(conn)
}
```

The handler should close its own connection:

```go
func handle(conn net.Conn) {
    defer conn.Close()

    // Read and process this client's request.
}
```

### What happens?

1. The main goroutine waits in `Accept()`.
2. A client connects; `Accept()` returns its connection.
3. `go handle(conn)` starts a handler goroutine.
4. The main goroutine goes back to `Accept()` instead of waiting for that handler to finish.
5. Other clients can connect and be handled by their own goroutines.

### Concurrency versus parallelism

- **Concurrency** means multiple tasks can make progress during overlapping periods.
- **Parallelism** means tasks are literally executing at the same instant on different CPU cores.

Go's scheduler manages goroutines on operating-system threads. Concurrency does not guarantee that every goroutine runs simultaneously on a separate CPU core.

A `time.Sleep(2 * time.Second)` in one handler pauses that handler; it does not pause the whole server. Other goroutines and the accept loop can continue.

Each handler owns one accepted connection. `defer conn.Close()` ensures the connection is closed when that handler returns, including on early-return error paths.

### Production considerations

A goroutine-per-connection design is a useful learning pattern, but a production server also needs:

- Read and write deadlines or timeouts.
- Maximum request-line, header, and body sizes.
- Limits on concurrent connections or handlers.
- Correct HTTP framing and malformed-request handling.
- Graceful shutdown and resource cleanup.
- Logging and observability.

## 12. Day 8 test record

The following are the terminal outputs shared during the exercise. Some earlier tests were described without their complete raw output; those are identified as expected behavior rather than captured logs.

### Test A — Build and start

Command:

```powershell
go build -o day8.exe .
.\day8.exe
```

Captured server output:

```text
server Listening on :8080
```

### Test B — Client connection

Captured server output:

```text
Client connected
```

### Test C — Parse request line and headers

Request used:

```http
GET /hello HTTP/1.1
Host: localhost:8080
User-Agent: curl/...
Accept: */*
```

Parsed request details shown during the test:

```text
Method: GET
Path: /hello
Version: HTTP/1.1
```

### Test D — Send a response

Captured curl response:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length:5

hello
```

### Test E — GET `/`

Captured server output:

```text
server Listening on :8080
Client connected
Method: GET
Path: /
Version: HTTP/1.1
Request Body:
```

Captured curl response:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length:4

home
```

### Test F — Routing

The routing endpoints were tested and marked complete. The exact terminal output for each route was not preserved in the shared conversation.

Expected behavior:

- `GET /` → `200 OK`, body `home`.
- `GET /health` → `200 OK`, JSON health response.
- `GET /slow` → wait 2 seconds, then `200 OK`, body `done`.
- Unknown path → `404 Not Found`.
- Unsupported method on a known path → `405 Method Not Allowed`.

These are expected results, not verbatim captured terminal output.

### Test G — Three concurrent `/slow` requests

Captured server output:

```text
server Listening on :8080
Client connected
Method: GET
Path: /slow
Version: HTTP/1.1
Request Body:
Client connected
Method: GET
Path: /slow
Version: HTTP/1.1
Request Body:
Client connected
Method: GET
Path: /slow
Version: HTTP/1.1
Request Body:
```

PowerShell timing command:

```powershell
Measure-Command {
    1..3 | ForEach-Object {
        Start-Job {
            curl.exe -s http://localhost:8080/slow
        }
    } | Wait-Job | Receive-Job
}
```

Captured timing output:

```text
Days              : 0
Hours             : 0
Minutes           : 0
Seconds           : 2
Milliseconds      : 920
Ticks             : 29207643
TotalDays         : 3.38051423611111E-05
TotalHours        : 0.000811323416666667
TotalMinutes      : 0.048679405
TotalSeconds      : 2.9207643
TotalMilliseconds : 2920.7643
```

Each `/slow` handler sleeps for approximately 2 seconds. Three sequential requests would take approximately 6 seconds of sleep time. The measured time of about 2.92 seconds is consistent with the requests overlapping.

PowerShell job startup and scheduling add overhead, so the measurement need not be exactly 2 seconds.

## 13. Windows executable blocked by Code Integrity

During the exercise, Go successfully built `day8.exe`, but Windows blocked it from running.

The relevant Code Integrity log reported:

```text
Event ID: 3077
Code Integrity determined that a process
(...\WindowsPowerShell\v1.0\powershell.exe) attempted to load
(...\go-day8\day8.exe) that did not meet the Enterprise
signing level requirements or violated code integrity policy.
```

Another event reported:

```text
Event ID: 3033
Code Integrity determined that a process attempted to load
day8.exe that did not meet the Enterprise signing level requirements.
```

The Device Guard query returned:

```text
CodeIntegrityPolicyEnforcementStatus         : 2
UsermodeCodeIntegrityPolicyEnforcementStatus : 2
VirtualizationBasedSecurityStatus            : 2
```

The machine reported:

```text
WindowsProductName : Windows 10 Home Single Language
WindowsVersion     : 2009
OsBuildNumber      : 26200
```

The device-join checks reported `NO` for Azure AD, domain, and workplace join.

### Interpretation

The logs establish that Windows Code Integrity enforcement blocked the executable because it did not meet the active signing requirements. They do not, by themselves, identify who installed the policy or the exact policy configuration.

Do not delete Code Integrity policy files or disable Windows security features just to run the executable. Continue diagnosis using Windows security logs and supported Windows settings, or seek help from a qualified Windows administrator.

## 14. Useful commands

Run from the `go-day8` directory:

```powershell
# Compile the current package into an executable
go build -o day8.exe .

# Run the executable
.\day8.exe

# Check whether the executable exists
Test-Path .\day8.exe

# Inspect executable metadata
Get-Item .\day8.exe | Select-Object Name, Length, LastWriteTime
```

If Windows blocks the executable, a successful `go build` only proves compilation succeeded. It does not override Windows Code Integrity policy. Do not attempt to bypass the policy.

## 15. Day 8 learning checklist

- [x] Understand `net.Listen` and `net.Listener`.
- [x] Understand `Accept()` and `net.Conn`.
- [x] Distinguish a listener from an accepted client connection.
- [x] Understand that TCP is a byte stream, not an HTTP message protocol.
- [x] Read HTTP lines with `bufio.Reader`.
- [x] Parse the request line.
- [x] Parse headers and normalize header names.
- [x] Understand the `\r\n\r\n` header/body separator.
- [x] Read a `Content-Length` request body with `io.ReadFull`.
- [x] Construct an HTTP response with the correct status line, headers, separator, and body.
- [x] Implement basic routing and status codes.
- [x] Handle clients concurrently with goroutines.
- [x] Measure overlapping `/slow` requests.

## Key takeaway

A basic HTTP server is an application protocol built on top of a transport connection. TCP provides a reliable byte stream; the HTTP implementation must interpret those bytes according to HTTP's message format. Go's `net` package gives direct access to the connection, while `net/http` provides a complete, robust HTTP implementation for real applications.
