# Go Day 10 — Building a URL Shortener REST API

## Goal

Build an in-memory URL shortener REST API using Go's standard `net/http` package.

This day connects several concepts learned earlier:

- HTTP methods, routes, status codes, headers, and JSON
- `http.Handler`, `http.ServeMux`, and `http.ResponseWriter`
- Structs, JSON encoding/decoding, and validation
- Maps and concurrency safety with `sync.RWMutex`
- Cryptographically secure random values
- Request body limits and content-type validation
- Reusable response helpers
- API testing with `curl`

The application stores shortened URLs in memory. Data is lost when the server restarts.

---

# 1. Project structure

```text
go-day10/
├── main.go
├── store.go
├── generator.go
├── handlers.go
└── go.mod
```

All `.go` files use `package main`, so they can directly access declarations in the other files in the same package.

Typical responsibilities:

| File | Responsibility |
|---|---|
| `main.go` | Create the store, register routes, start the HTTP server |
| `store.go` | In-memory data store and thread-safe CRUD operations |
| `generator.go` | Generate random Base62 short codes |
| `handlers.go` | HTTP handlers, validation, JSON responses |

---

# 2. REST API design

A REST API models resources using nouns. HTTP methods express the operation.

The resource in this project is a shortened URL.

| Method | Endpoint | Purpose | Success status |
|---|---|---|---:|
| `POST` | `/urls` | Create a short URL | 201 Created |
| `GET` | `/urls/{code}` | Retrieve one URL | 200 OK |
| `GET` | `/urls` | List all URLs | 200 OK |
| `DELETE` | `/urls/{code}` | Delete a URL | 204 No Content |

We do not provide `PUT` because the short code is immutable in this version of the application.

## Why these status codes?

- **201 Created:** A new resource was successfully created.
- **200 OK:** A read operation succeeded.
- **204 No Content:** The delete operation succeeded and intentionally returns no response body.
- **400 Bad Request:** The request body or supplied URL is invalid.
- **404 Not Found:** The requested short code does not exist.
- **413 Content Too Large:** The request body exceeds the configured limit.
- **415 Unsupported Media Type:** The client did not send JSON.
- **500 Internal Server Error:** An unexpected server-side operation failed.
- **409 Conflict:** A useful status when a requested operation conflicts with existing state, such as a duplicate resource identifier.

---

# 3. HTTP routing

Go's `http.ServeMux` matches incoming requests to handlers.

Example route registrations:

```go
mux := http.NewServeMux()

mux.HandleFunc("POST /urls", handler.CreateUrlHandler)
mux.HandleFunc("GET /urls", handler.ListURLsHandler)
mux.HandleFunc("GET /urls/{code}", handler.GetURLHandler)
mux.HandleFunc("DELETE /urls/{code}", handler.DeleteURLHandler)
```

Go 1.22 and later support method-aware patterns and path wildcards.

Inside a handler:

```go
code := r.PathValue("code")
```

This retrieves the value matched by `{code}`.

Remember the earlier routing lesson:

- `GET /` is a subtree pattern.
- `GET /{$}` matches only the exact root path.

---

# 4. In-memory store and concurrency safety

The store uses a Go map:

```go
map[string]string
```

The key is the short code and the value is the original URL.

A simplified store structure:

```go
type Store struct {
    mu   sync.RWMutex
    urls map[string]string
}
```

The constructor initializes the map:

```go
func NewStore() *Store {
    return &Store{
        urls: make(map[string]string),
    }
}
```

## Why do we need a mutex?

The HTTP server can handle multiple requests concurrently. Multiple handler goroutines may access the same map at the same time.

Go maps are not safe for concurrent read/write access.

`sync.RWMutex` provides two kinds of locking:

- `RLock()` / `RUnlock()` for reads. Multiple readers can hold the read lock simultaneously.
- `Lock()` / `Unlock()` for writes. A writer gets exclusive access.

Typical operation choices:

| Operation | Lock |
|---|---|
| Get | `RLock` |
| List | `RLock` |
| Create | `Lock` |
| Delete | `Lock` |

Example read:

```go
s.mu.RLock()
defer s.mu.RUnlock()

value, ok := s.urls[code]
```

Example write:

```go
s.mu.Lock()
defer s.mu.Unlock()

s.urls[code] = longURL
```

## The check-and-insert race

Creating a short code involves two related operations:

1. Check whether the generated code already exists.
2. Insert the code if it does not exist.

These operations must be protected by the same exclusive lock. Otherwise, two goroutines could both check the map before either inserts the code.

The important principle is:

> Protect the entire critical section, not just individual map operations.

When retrying after a collision, release the lock before continuing the loop. Avoid putting `defer Unlock()` inside a retry loop because the deferred calls execute only when the surrounding function returns.

## Map iteration order

Go map iteration order is unspecified. A list endpoint should not promise a stable ordering unless the application explicitly sorts the results.

---

# 5. URL model and JSON tags

A URL resource can be represented as:

```go
type URL struct {
    Code    string `json:"code"`
    LongURL string `json:"longUrl"`
}
```

JSON struct tags control the names exposed in JSON.

For example, `LongURL` becomes `longUrl` in the JSON representation.

Exported Go fields are important because `encoding/json` works with exported struct fields.

For an empty list, initialize the slice with:

```go
result := make([]URL, 0, len(s.urls))
```

This ensures an empty list encodes as:

```json
[]
```

rather than:

```json
null
```

---

# 6. Generating short codes with Base62

The short code uses these 62 characters:

```text
0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz
```

This is Base62:

- 10 digits
- 26 uppercase letters
- 26 lowercase letters

A seven-character code has:

\[
62^7 = 3,521,614,606,208
\]

possible combinations (about 3.52 trillion).

## Why use `crypto/rand`?

A short code should be difficult to predict.

Go provides:

```go
crypto/rand
```

It obtains cryptographically secure random bytes. This is preferable to `math/rand` when identifiers should be unpredictable.

## Modulo bias

A byte can represent values from 0 through 255. The Base62 alphabet has 62 characters.

But:

\[
256 = 62 \times 4 + 8
\]

Using `randomByte % 62` directly gives the first eight indexes more possible byte values than the remaining indexes.

That is modulo bias.

Rejection sampling avoids this:

- Accept byte values from 0 through 247.
- Reject values from 248 through 255.
- Map accepted values with `% 62`.

There are 248 accepted values, and each of the 62 indexes has exactly four possible accepted byte values.

The generator can retry rejected bytes until it has enough characters.

## Collisions

Random codes can collide. The store must check whether a generated code already exists and retry if necessary.

Randomness makes collisions unlikely; checking the store makes the creation operation safe.

---

# 7. HTTP request body and JSON decoding

A request body is a stream. In `net/http`, `r.Body` implements `io.ReadCloser`.

A typical request struct:

```go
type CreateUrlRequest struct {
    LongUrl string `json:"longUrl"`
}
```

Decode JSON:

```go
var req CreateUrlRequest

err := json.NewDecoder(r.Body).Decode(&req)
if err != nil {
    // Return a bad request response.
}
```

`json.NewDecoder` reads from the stream and `Decode` decodes one JSON value.

Important distinction:

- `json.Unmarshal` works with a byte slice that is already in memory.
- `json.Decoder` reads from an `io.Reader`, making it useful for HTTP request bodies.

## Important follow-up: strict JSON decoding

A single call to `Decode()` does not prove that the entire request body contains exactly one JSON value.

For example, this body contains two JSON values:

```json
{"longUrl":"https://google.com"}
{"admin":true}
```

The first `Decode()` can successfully decode the first object and return `nil`, leaving trailing content unread.

A stricter handler should:

1. Decode the expected JSON object.
2. Attempt a second decode.
3. Accept only if the second decode returns `io.EOF`.
4. Reject a second JSON value or invalid trailing content with HTTP 400.

This strict-decoding exercise was identified as the next learning task. It should be implemented and tested before treating strict JSON validation as complete.

Also remember: if a decoder stops after the first JSON value, it may not consume all trailing bytes. A body-size limit alone does not guarantee that trailing content has been inspected. Strict decoding must consume/check the remaining input.

---

# 8. Content-Type validation

The create endpoint expects JSON.

Read the header:

```go
contentType := r.Header.Get("Content-Type")
```

A simple equality check against `application/json` is too strict because clients may send:

```http
Content-Type: application/json; charset=utf-8
```

Use `mime.ParseMediaType`:

```go
mediaType, _, err := mime.ParseMediaType(contentType)

if err != nil || mediaType != "application/json" {
    // Return 415 Unsupported Media Type.
}
```

This extracts the media type separately from parameters such as `charset`.

Expected behavior:

| Content-Type | Result |
|---|---|
| `application/json` | Accept |
| `application/json; charset=utf-8` | Accept |
| `text/plain` | Reject with 415 |
| Missing header | Reject with 415 |
| `application/xml` | Reject with 415 |

---

# 9. Limiting request body size

Clients should not be able to send arbitrarily large request bodies.

Use `http.MaxBytesReader`:

```go
maxUploadSize := int64(1 << 20) // 1 MiB
r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
```

`1 << 20` equals 1,048,576 bytes, or 1 MiB.

The size is expressed in bytes, not characters.

When the limit is exceeded, reading the body returns an error that can be identified with `errors.As`:

```go
var maxByteError *http.MaxBytesError

if errors.As(err, &maxByteError) {
    // Return 413 Request Entity Too Large.
}
```

For other JSON decoding errors, return 400.

## `100 Continue` and `413`

During the oversized-body test, the response included:

```http
HTTP/1.1 100 Continue

HTTP/1.1 413 Request Entity Too Large
```

These are not contradictory:

- `100 Continue` is an interim response indicating that the client may proceed with sending the request body.
- `413` is the final response after the server detects that the body exceeds the configured limit.

---

# 10. URL validation

The create endpoint should not accept arbitrary strings as destination URLs.

The validation implemented in this project checks:

1. The URL is not empty or whitespace-only.
2. The URL length does not exceed 2048 characters.
3. `url.Parse` can parse the input.
4. The scheme is `http` or `https`.
5. A hostname is present.

Useful standard-library functions:

```go
strings.TrimSpace(input)
url.Parse(input)
strings.ToLower(parsedURL.Scheme)
parsedURL.Hostname()
```

Use `errors.New` for a new simple error:

```go
return errors.New("invalid scheme")
```

Use `fmt.Errorf` with `%w` to wrap an existing error:

```go
return fmt.Errorf("url parsing failed: %w", err)
```

A useful cleanup is to normalize the scheme once:

```go
scheme := strings.ToLower(parsedURL.Scheme)

if scheme != "http" && scheme != "https" {
    return errors.New("invalid scheme")
}
```

The validator should return an error; the handler decides which HTTP status and response body to send.

### Validation is not the same as URL safety

Parsing and checking a URL does not prove that it is safe to fetch. This project stores URLs but does not fetch them.

If a future version fetches user-supplied URLs, it will need additional protections against SSRF, private-network access, DNS rebinding, and unsafe redirects.

---

# 11. Reusable JSON response helpers

Instead of repeating response headers and encoding code in every handler, use shared helpers.

Response model:

```go
type ErrorResponse struct {
    Error string `json:"error"`
}
```

Success response helper:

```go
func writeJson(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    if err := json.NewEncoder(w).Encode(v); err != nil {
        log.Printf("failed to encode the JSON response: %v", err)
    }
}
```

Error response helper:

```go
func writeError(w http.ResponseWriter, status int, message string) {
    errResponse := ErrorResponse{
        Error: message,
    }

    writeJson(w, status, errResponse)
}
```

`any` is an alias for `interface{}`. It lets the helper accept different response types.

Examples:

```go
writeJson(w, http.StatusOK, urls)
writeJson(w, http.StatusCreated, response)
writeError(w, http.StatusBadRequest, "Invalid request body")
writeError(w, http.StatusNotFound, "URL not found")
```

## HTTP response commitment

Calling:

```go
w.WriteHeader(status)
```

commits the response status.

If the first body write happens before `WriteHeader`, Go implicitly commits status 200.

After the status and headers have been sent, the handler cannot replace them with a different status.

Therefore, if JSON encoding or writing fails after `WriteHeader`, logging the error is more accurate than attempting to send a second HTTP error response.

A more robust approach is to encode the JSON into a temporary buffer first, then commit the status and write the encoded bytes. This can catch encoding failures before the response is committed. A network write can still fail after commitment.

---

# 12. `Location` header and resource creation

A successful POST returns:

```http
HTTP/1.1 201 Created
Content-Type: application/json
Location: /urls/Ab12Cd3
```

The response body can contain:

```json
{
  "code": "Ab12Cd3",
  "longUrl": "https://example.com"
}
```

The `Location` header identifies the newly created resource.

For a server running at `http://localhost:8080`, the relative location:

```text
/urls/Ab12Cd3
```

resolves to:

```text
http://localhost:8080/urls/Ab12Cd3
```

Set the header before writing the response:

```go
w.Header().Set("Location", "/urls/"+shortCode)
writeJson(w, http.StatusCreated, response)
```

Headers must be set before the response is committed.

---

# 13. DELETE and 204 No Content

When deletion succeeds:

```go
w.WriteHeader(http.StatusNoContent)
```

Do not encode JSON after sending 204.

A 204 response intentionally has no response body.

When the code does not exist, return a 404 error response instead.

---

# 14. Testing with curl

Start the server in one Ubuntu/WSL terminal:

```bash
go run .
```

Run requests in another terminal.

## Create a URL

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: application/json" \
  --data-raw '{"longUrl":"https://example.com/hello"}'
```

Expected: `201 Created`, a JSON body, and a `Location` header.

## Get a URL

Replace `YOUR_CODE` with the code returned by POST:

```bash
curl -i http://localhost:8080/urls/YOUR_CODE
```

Expected: `200 OK`.

## Get a nonexistent URL

```bash
curl -i http://localhost:8080/urls/doesNotExist
```

Expected: `404 Not Found`.

## List all URLs

```bash
curl -i http://localhost:8080/urls
```

Expected: `200 OK` and a JSON array.

## Delete a URL

```bash
curl -i -X DELETE http://localhost:8080/urls/YOUR_CODE
```

Expected: `204 No Content`.

## Test JSON with charset

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: application/json; charset=utf-8" \
  --data-raw '{"longUrl":"https://example.org"}'
```

Expected: `201 Created`.

## Test unsupported media type

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: text/plain" \
  --data-raw '{"longUrl":"https://example.com"}'
```

Expected: `415 Unsupported Media Type`.

## Test malformed JSON

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: application/json" \
  --data-raw '{"longUrl":'
```

Expected: `400 Bad Request`.

## Test invalid URL scheme

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: application/json" \
  --data-raw '{"longUrl":"ftp://example.com"}'
```

Expected: `400 Bad Request`.

## Test oversized request body

Create a JSON body larger than 1 MiB:

```bash
python3 -c 'import json; print(json.dumps({"longUrl":"https://example.com","extra":"A"*1100000}))' > /tmp/large.json
```

Send it:

```bash
curl -i -X POST http://localhost:8080/urls \
  -H "Content-Type: application/json" \
  --data-binary @/tmp/large.json
```

Expected: `413 Request Entity Too Large`.

---

# 15. Test results from this implementation

The smoke-test run produced the following results:

| Scenario | Actual result |
|---|---:|
| Root route without a registered handler | 404 |
| Create URL | 201 |
| Get created URL | 200 |
| Get unknown code | 404 |
| List URLs | 200 |
| JSON with charset | 201 |
| Wrong Content-Type | 415 |
| Missing Content-Type | 415 |
| Malformed JSON | 400 |
| Invalid URL scheme | 400 |
| Empty URL | 400 |
| Request body over 1 MiB | 413 |
| Delete existing URL | 204 |
| Get deleted URL | 404 |
| Delete same URL again | 404 |

All tested status codes matched the expected results.

The create response included a valid `Location` header. The list endpoint returned a JSON array. The delete endpoint returned 204 without a body.

The root route's 404 was expected because no root handler was registered.

### Note about consistency

The smoke-test output showed that some 404 responses still had an empty body. The next cleanup is to route missing-resource responses through `writeError()` so all API errors consistently return JSON.

---

# 16. What this project does not implement yet

This is an educational in-memory implementation, not a production URL shortener.

Not yet included:

- Persistent database storage
- User accounts and authentication
- Rate limiting
- URL expiration
- Analytics and click tracking
- Custom aliases
- Distributed ID generation
- Redis caching
- Pagination for large lists
- Production-grade observability
- Strict single-JSON-value validation (next exercise)
- Comprehensive automated unit and integration tests

Because storage is in memory, restarting the server clears all URLs.

---

# 17. Day 10 learning checklist

- [x] Design REST endpoints around a resource.
- [x] Choose HTTP methods and status codes.
- [x] Register method-aware Go routes.
- [x] Read path parameters using `r.PathValue`.
- [x] Represent resources with Go structs and JSON tags.
- [x] Build an in-memory map-backed store.
- [x] Protect shared map access with `sync.RWMutex`.
- [x] Understand why check-and-insert must be atomic.
- [x] Generate Base62 codes using `crypto/rand`.
- [x] Understand modulo bias and rejection sampling.
- [x] Handle random-code collisions.
- [x] Validate URLs.
- [x] Validate the request Content-Type.
- [x] Limit request body size with `http.MaxBytesReader`.
- [x] Detect `http.MaxBytesError` using `errors.As`.
- [x] Build reusable JSON success and error helpers.
- [x] Understand when an HTTP response is committed.
- [x] Return a `Location` header after resource creation.
- [x] Return 204 without a body after successful deletion.
- [x] Test CRUD and error scenarios using `curl`.
- [ ] Make all missing-resource errors return consistent JSON.
- [ ] Reject multiple JSON values and invalid trailing data.
- [ ] Verify that the full request body is checked against the size limit.
- [ ] Add automated tests for store and handler behavior.

---

# 18. Suggested commands before committing

Format the project:

```bash
gofmt -w .
```

Build all packages:

```bash
go build ./...
```

Run tests:

```bash
go test ./...
```

Run the race detector:

```bash
go test -race ./...
```

The race detector helps identify unsynchronized concurrent memory access. It does not replace correct locking or thorough concurrency tests.

---

# Final takeaway

Day 10 brought together HTTP, JSON, validation, concurrency, random identifiers, and API design in one practical project.

The most important engineering lessons:

1. A REST API should have predictable resources, methods, status codes, and response formats.
2. Shared maps need synchronization when handlers run concurrently.
3. A check followed by a write must be protected as one critical section.
4. Request bodies are streams; decoding one JSON value does not necessarily validate the entire body.
5. Set headers before committing the response.
6. Enforce request-size limits and validate media types at the API boundary.
7. Test both successful operations and failure cases.

**Day 10 project: In-memory URL shortener REST API.**
