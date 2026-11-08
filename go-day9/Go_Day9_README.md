# Go Day 9 — HTTP Servers with `net/http`

> Goal: Replace the manual HTTP implementation from Day 8 with Go's standard `net/http` package, understand its abstractions, and learn how routing and HTTP responses work.

## Learning checklist

- [x] `http.Handler` and the `ServeHTTP` contract
- [x] `http.HandlerFunc`
- [x] `http.ServeMux` and route registration
- [x] Relationship between `http.Server`, `ServeMux`, and handlers
- [x] Method-aware routing
- [x] Dynamic path values with `r.PathValue`
- [x] JSON decoding with `json.NewDecoder`
- [x] JSON encoding with `json.NewEncoder`
- [x] `http.ResponseWriter`
- [x] Response headers, status codes, and response commitment
- [x] Exact-root versus subtree route patterns
- [ ] Follow-up: request size limits, server timeouts, and automated HTTP tests

---

## 1. Day 8 vs Day 9: the big picture

In Day 8, we built a basic HTTP server using lower-level TCP primitives. We manually accepted connections, read bytes, parsed request lines and headers, handled request bodies, selected routes, and constructed HTTP responses.

In Day 9, we use Go's standard library:

```go
net/http
```

The library handles much of the HTTP protocol machinery so we can focus on application behaviour.

| Responsibility | Day 8: manual TCP server | Day 9: `net/http` |
|---|---|---|
| Listen for TCP connections | `net.Listen` | `http.Server.ListenAndServe` |
| Accept connections | `Listener.Accept` | Managed by HTTP server |
| Read HTTP bytes | `conn.Read` / `bufio.Reader` | Managed by `net/http` |
| Parse request line and headers | Manual parsing | `http.Request` |
| Read request body | Manual framing / `Content-Length` | `r.Body` |
| Route requests | Manual conditions | `http.ServeMux` |
| Handle concurrent connections | We launched goroutines | HTTP server manages connection handling |
| Build response | Manual status/header/body bytes | `http.ResponseWriter` |

Day 8 was an educational HTTP subset, not a complete HTTP implementation. `net/http` provides mature protocol handling, including parsing, framing, connection management, and many edge cases that are easy to miss when implementing HTTP manually.

## 2. `http.Handler`: the handler contract

The central abstraction is the `http.Handler` interface:

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Any type that implements this method satisfies `http.Handler`. Go interfaces are implemented implicitly: a type does not need to declare that it implements the interface.

Example:

```go
type UserHandler struct{}

func (h UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "user handler")
}
```

`UserHandler` satisfies `http.Handler` because it has the required `ServeHTTP` method.

- `http.ResponseWriter` lets the handler construct the response: headers, status, and body.
- `*http.Request` represents the incoming request: method, URL, headers, body, path values, and other metadata.

Mental model:
- `r` tells the handler what the client sent.
- `w` lets the handler decide what the server sends back.

**Analogy:** Imagine every department in an organization must follow a standard contract called `HandleRequest`. The organization can send work to any department that follows the contract without knowing its internal implementation. The HTTP server similarly invokes any value satisfying `http.Handler`.

## 3. `http.HandlerFunc`: adapting functions into handlers

A normal function does not automatically have methods, so it does not itself satisfy `http.Handler`. Go provides `http.HandlerFunc`, an adapter type that lets a function act as a handler.

Conceptually:

```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```

This is why we can conveniently register a function:

```go
mux.HandleFunc("GET /health", healthHandler)
```

Use a custom type implementing `http.Handler` when the handler needs dependencies or state. Use `HandleFunc` for straightforward handler functions.

## 4. `http.ServeMux`: routing requests

A `ServeMux` is Go's HTTP request multiplexer, commonly called a router.

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /health", healthHandler)
mux.HandleFunc("POST /echo", echoHandler)
mux.HandleFunc("GET /users/{id}", userHandler)
```

When a request arrives, the mux matches its method and path against registered patterns, then invokes the matching handler.

### Go 1.22+ method-aware patterns

Go 1.22 added method-aware patterns and wildcard path segments to the standard `ServeMux`.

```go
mux.HandleFunc("GET /health", healthHandler)
mux.HandleFunc("POST /echo", echoHandler)
mux.HandleFunc("GET /users/{id}", userHandler)
```

A `GET /health` request matches the GET pattern. A `DELETE /health` request does not match that GET-only pattern. When the path is registered for other methods, the mux can return `405 Method Not Allowed` and an `Allow` header. `HEAD` is also treated as matching a GET pattern.

### Dynamic path values

This route:

```go
mux.HandleFunc("GET /users/{id}", userHandler)
```

matches:

```text
/users/42
/users/100
/users/alex
```

Handler:

```go
func userHandler(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    fmt.Fprintf(w, "user id: %s", id)
}
```

`r.PathValue("id")` retrieves the value captured by `{id}`. It returns a string. Convert it with `strconv.Atoi` if the application needs an integer.

One handler can serve many IDs instead of registering a separate route for every user.

### Important: `/` is a subtree pattern

This registration:

```go
mux.HandleFunc("GET /", homeHandler)
```

does **not** mean only the exact root path. Because the pattern ends in `/`, it matches the root and paths beneath it, such as `/test` and `/about`.

That explains why `/test` invoked `homeHandler` even though `/test` had not been registered separately.

For an exact root route in Go 1.22+:

```go
mux.HandleFunc("GET /{$}", homeHandler)
```

Now `/` matches, while `/test` and `/about` do not match this route.

| Pattern | Meaning |
|---|---|
| `GET /{$}` | Exact root path `/` |
| `GET /health` | Exact `/health` path |
| `GET /users/{id}` | One wildcard path segment |
| `GET /` | Root subtree, including paths below `/` |

## 5. How `http.Server`, `ServeMux`, and handlers work together

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /health", healthHandler)

srv := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}

log.Fatal(srv.ListenAndServe())
```

Request flow:

```text
HTTP client
    |
    | GET /health
    v
http.Server
    | parses HTTP request
    | creates ResponseWriter and Request
    v
ServeMux (srv.Handler)
    | matches method + path
    v
healthHandler
    | writes status, headers, and body
    v
HTTP response sent to client
```

- `http.Server` listens for connections and serves HTTP requests.
- `Handler` tells the server which handler receives requests. Here, it is the mux.
- `ServeMux` matches the request to a route.
- The selected handler executes application logic.

The client connects to the server, not directly to the mux.

`http.ListenAndServe(":8080", mux)` is a convenience shortcut. An explicit `http.Server` is useful when configuring timeouts and other server settings.

For example:

```go
srv := &http.Server{
    Addr:              ":8080",
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
}
```

`ReadHeaderTimeout` limits how long the server waits to read request headers. Production servers should choose appropriate timeouts for their workload.

## 6. `http.ResponseWriter`: constructing the response

A handler receives:

```go
func handler(w http.ResponseWriter, r *http.Request)
```

### Writing a body

```go
fmt.Fprintln(w, "home")
```

This writes `home` plus a newline to the HTTP response body.

Compare:

```go
fmt.Println("home")
```

That writes to the server's terminal, not to the HTTP client.

### Setting headers

```go
w.Header().Set("Content-Type", "application/json")
```

For JSON responses, this tells the client how to interpret the response body. Set headers before writing the body.

### Setting a status

```go
w.WriteHeader(http.StatusNotFound)
w.Write([]byte("not found"))
```

`WriteHeader` commits the response status and headers. If no status is explicitly written, the first call to `Write` implicitly commits `200 OK`.

## 7. Response commitment: why order matters

An HTTP response is sent in this order:

```text
Status line
Headers
Blank line
Body
```

Example:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 5

Hello
```

The status and headers must be sent before the body. Once committed, they cannot be changed for that response.

### Case A: body first

```go
func testHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello"))
    w.WriteHeader(http.StatusNotFound)
}
```

What happens:

1. `w.Write` is called.
2. If no status has been set, Go commits the default `200 OK`.
3. The body is written.
4. `WriteHeader(404)` is too late and is ignored.
5. Go logs a warning similar to:

```text
http: superfluous response.WriteHeader call
```

The client receives `200 OK`.

### Case B: status first

```go
func testHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Hello"))
}
```

The client receives `404 Not Found` with `Hello` as the body.

### Why can't the status be changed later?

The server sends response bytes over the connection in order. Once the status line and headers have been sent, the client may already have read and acted on them. TCP is a byte stream; it does not provide a way to retract bytes already sent.

**Rule: Set headers and status before writing the response body.**

For common error responses, `http.Error` handles this ordering:

```go
http.Error(w, "invalid JSON", http.StatusBadRequest)
```

## 8. Reading request bodies

The incoming request body is:

```go
r.Body
```

Its type is `io.ReadCloser`. It is a stream, not a string or a byte slice.

For a simple echo endpoint:

```go
body, err := io.ReadAll(r.Body)
if err != nil {
    http.Error(w, "could not read request body", http.StatusBadRequest)
    return
}

w.Write(body)
```

`io.ReadAll` consumes the stream and returns its contents as `[]byte`.

For a real service, do not blindly accept unlimited request bodies. You can impose a limit:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
```

Then handle errors from reading or decoding. The appropriate limit depends on the endpoint.

## 9. JSON request decoding and response encoding

Day 4 introduced `json.Marshal` and `json.Unmarshal`. In HTTP handlers, streaming APIs are often convenient.

### Define request and response types

```go
type GreetRequest struct {
    Name string `json:"name"`
}

type GreetResponse struct {
    Message string `json:"message"`
}
```

Struct tags define the JSON field names.

### Decode JSON from the request

```go
var input GreetRequest

err := json.NewDecoder(r.Body).Decode(&input)
if err != nil {
    http.Error(w, "invalid JSON", http.StatusBadRequest)
    return
}
```

`json.NewDecoder` accepts an `io.Reader`. `r.Body` already implements `io.Reader`, so the decoder reads directly from the request stream.

`Decode(&input)` receives a pointer because it must populate the original variable.

- `&input` means the memory address of `input`.
- The decoder uses that address to fill the struct.
- Passing only `input` would pass a copy, not provide the destination for decoding.

### Encode JSON into the response

```go
response := GreetResponse{
    Message: "Hello, " + input.Name + "!",
}

w.Header().Set("Content-Type", "application/json")

if err := json.NewEncoder(w).Encode(response); err != nil {
    log.Printf("encode response: %v", err)
}
```

`json.NewEncoder(w)` accepts an `io.Writer`. `http.ResponseWriter` implements `io.Writer`, so the encoder writes JSON directly to the HTTP response. `Encode` appends a newline.

### Decoder vs `io.ReadAll` + `Unmarshal`

Both approaches are valid:

```go
// Streaming decoder
err := json.NewDecoder(r.Body).Decode(&input)
```

Or:

```go
// Read all bytes, then unmarshal
body, err := io.ReadAll(r.Body)
if err != nil {
    // handle error
}

err = json.Unmarshal(body, &input)
```

Do not read all of `r.Body` and then try to decode from the same body stream: reading it consumes the stream. Choose one approach.

Example request:

```http
POST /greet
Content-Type: application/json

{"name":"kailash"}
```

Example response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"message":"Hello, kailash!"}
```

Invalid JSON should produce `400 Bad Request`, not `500 Internal Server Error`, because the client sent a malformed request. A production endpoint should also validate required fields, such as rejecting an empty name, and consider rejecting unexpected trailing JSON values if exactly one JSON object is required.

## 10. Day 9 application routes

| Method | Path | Handler | Purpose |
|---|---|---|---|
| GET | `/` (registered as `GET /{$}` for exact root) | `homeHandler` | Home response |
| GET | `/health` | `healthHandler` | JSON health response |
| POST | `/echo` | `echoHandler` | Reads and echoes request body |
| GET | `/users/{id}` | `userHandler` | Returns dynamic user ID |
| POST | `/greet` | `greetHandler` | Decodes JSON and returns JSON greeting |

Compact routing and server setup:

```go
mux := http.NewServeMux()

mux.HandleFunc("GET /{$}", homeHandler)
mux.HandleFunc("GET /health", healthHandler)
mux.HandleFunc("POST /echo", echoHandler)
mux.HandleFunc("GET /users/{id}", userHandler)
mux.HandleFunc("POST /greet", greetHandler)

srv := &http.Server{
    Addr:              ":8080",
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
}

log.Fatal(srv.ListenAndServe())
```

Remember to import `time` if you add `ReadHeaderTimeout`.

## 11. Testing endpoints in PowerShell

Use `curl.exe` explicitly in PowerShell when you want the actual curl executable.

### Home

```powershell
curl.exe -i http://localhost:8080/
```

Expected body:

```text
home
```

### Health

```powershell
curl.exe -i http://localhost:8080/health
```

Expected JSON:

```json
{"status":"ok"}
```

### Echo

```powershell
curl.exe -i -X POST --data-raw "x=1" http://localhost:8080/echo
```

Expected body:

```text
x=1
```

### Dynamic user path

```powershell
curl.exe -i http://localhost:8080/users/42
```

Expected body:

```text
user id: 42
```

### Greeting JSON endpoint

PowerShell's native HTTP client avoids some shell-quoting confusion:

```powershell
$body = '{"name":"kailash"}'

Invoke-RestMethod `
    -Uri "http://localhost:8080/greet" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

Expected result:

```text
message
-------
Hello, kailash!
```

You can also use `curl.exe` with a PowerShell variable:

```powershell
$body = '{"name":"kailash"}'

curl.exe -i -X POST `
    -H "Content-Type: application/json" `
    --data-raw $body `
    http://localhost:8080/greet
```

For malformed JSON:

```powershell
$body = '{bad'

Invoke-RestMethod `
    -Uri "http://localhost:8080/greet" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

The endpoint should return `400 Bad Request`.

## 12. Key takeaways

1. `http.Handler` is an interface. Any type implementing `ServeHTTP` satisfies it.
2. `http.HandlerFunc` adapts a function into an HTTP handler.
3. `http.ServeMux` matches incoming requests to registered handlers.
4. Go 1.22+ supports method-aware patterns and wildcard path segments.
5. `GET /` is a subtree pattern; use `GET /{$}` for the exact root path.
6. `r.PathValue("id")` retrieves a wildcard path value as a string.
7. `r.Body` is a stream (`io.ReadCloser`), not a string or byte slice.
8. `json.NewDecoder(r.Body)` decodes JSON directly from the request stream.
9. `json.NewEncoder(w)` writes JSON directly to the response writer.
10. `&v` passes a variable's address so the decoder can populate it.
11. `ResponseWriter` sets headers, commits a status, and writes the body.
12. The first `Write` implicitly commits `200 OK` if no status was set.
13. Once headers are committed, a later `WriteHeader` cannot change the status.
14. Set headers and status before writing the response body.
15. `http.Server` owns the HTTP serving lifecycle; `ServeMux` handles routing; handlers implement endpoint behaviour.
16. Prefer the standard library over manually parsing HTTP in ordinary Go web applications.

## 13. Day 8 → Day 9: the mental-model change

In Day 8, you learned the mechanics underneath:

```text
TCP connection
  -> bytes
  -> HTTP request parsing
  -> routing
  -> handler logic
  -> HTTP response bytes
```

In Day 9, `net/http` provides the HTTP layer:

```text
HTTP client
  -> http.Server
  -> http.Request + http.ResponseWriter
  -> ServeMux
  -> handler
  -> response
```

Day 8 helped you understand the mechanics. Day 9 teaches you how Go applications use a standard, reusable HTTP abstraction.

## 14. Git commands

From the repository root:

```powershell
cd C:\Users\kaila\code\go-arc-1

git status
git add go-day9
git commit -m "day 9: net/http"
git push
```

If you have not initialized the Day 9 module, run this from the `go-day9` directory:

```powershell
go mod init go-day9
```

Then format, test, and build:

```powershell
gofmt -w .
go test ./...
go build ./...
```


