# Go Day 6 — What Happens When You Open a URL?

## Goals

Understand the network path behind a browser request and practise measuring parts of it with Go.

By the end of this day, you should be able to explain:
- How DNS resolves a domain name to an IP address.
- How DNS caches and TTL work.
- How TCP establishes a connection using the three-way handshake.
- How HTTP requests and responses are structured.
- Why HTTP keep-alive can make later requests faster.
- How latency and RTT affect performance.
- How to make rough capacity and storage estimates.

## 1. DNS

DNS (Domain Name System) translates a hostname such as `example.com` into one or more IP addresses.

A simplified lookup path is:

1. The browser/application checks its DNS cache.
2. The operating system may check its DNS cache.
3. A recursive DNS resolver checks its cache.
4. If the answer is not cached, the resolver follows the hierarchy: root server → TLD server (such as `.com`) → authoritative DNS server.
5. The answer is returned and may be cached for its TTL.

**TTL (time to live)** tells a DNS cache how long it may reuse a DNS answer.

- Longer TTL: usually fewer DNS lookups and less resolver traffic.
- Shorter TTL: DNS changes can be picked up sooner, but lookup traffic may increase.
- TTL expiry does not delete the authoritative DNS record; it means the cached answer should be revalidated.

Go's `net.LookupHost()` resolves a hostname but does not expose the DNS record's TTL.

## 2. TCP connection

TCP is a reliable, ordered byte-stream transport protocol.

The three-way handshake:

1. Client → Server: `SYN`
2. Server → Client: `SYN-ACK`
3. Client → Server: `ACK`

The handshake synchronizes sequence numbers and confirms that both endpoints can communicate. It generally costs about one RTT before application data can be sent.

A TCP connection is identified by a four-tuple:
- Source IP address
- Source port
- Destination IP address
- Destination port

The client typically uses a temporary **ephemeral source port**. The server listens on a destination port, such as 80 for HTTP or 443 for HTTPS.

Useful Go APIs:
- `net.Dial("tcp", "example.com:80")` opens a TCP connection.
- `net.Conn` represents the connection.
- `conn.LocalAddr()` reports the local endpoint.
- `conn.RemoteAddr()` reports the remote endpoint.
- `conn.Close()` closes the connection.

**Important:** `net.Dial("tcp", "example.com:443")` creates a TCP connection; it does not perform TLS encryption. HTTPS also requires a TLS handshake. HTTP/3 uses QUIC over UDP rather than TCP.

## 3. HTTP request and response

HTTP is an application-layer protocol. It defines methods, paths, headers, status codes and message bodies. TCP transports bytes; it does not understand the meaning of `GET` or `200 OK`.

Example HTTP/1.1 request:

```http
GET / HTTP/1.1
Host: example.com
Connection: close

```

The blank line marks the end of the headers. A request may also contain a body, such as a `POST` request.

Example response:

```http
HTTP/1.1 200 OK
Content-Type: text/html
Content-Length: 125

<html>...</html>
```

Common status codes:

| Code | Meaning |
|---|---|
| 200 | OK |
| 301 | Moved Permanently |
| 302 | Found / temporary redirect |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 500 | Internal Server Error |

A `301` or `302` response commonly includes a `Location` header. A raw TCP client does not automatically follow redirects.

## 4. Keep-alive and HTTP versions

HTTP/1.1 connections are persistent by default unless closed or otherwise made unavailable. Keep-alive allows multiple HTTP requests to reuse one TCP connection.

This can avoid another TCP handshake and save roughly one RTT for a later request. It does not eliminate the latency of sending the request and receiving its response.

- **HTTP/1.1:** Text-based message syntax; persistent connections are supported.
- **HTTP/2:** Binary framing and multiple streams multiplexed over one TCP connection.
- **HTTP/3:** Uses QUIC over UDP.

HTTP/2 multiplexing does not remove TCP-level head-of-line blocking: packet loss affecting the TCP stream can delay data for multiple streams.

## 5. Latency and RTT

RTT (round-trip time) is the time for a signal to travel from one endpoint to another and for a response to return.

Illustrative first-request example:

| Step | Time |
|---|---:|
| DNS lookup | 40 ms |
| TCP handshake | 60 ms |
| Server processing | 20 ms |
| Response travel | 60 ms |
| **Total** | **180 ms** |

These are illustrative values, not fixed timings. Actual latency depends on distance, routing, congestion, caching and server load.

A later request may be faster when DNS is cached and an existing keep-alive connection is reusable.

## Exercise 1 — Measure DNS and make a raw HTTP request

Build a Go program that:

1. Calls `net.LookupHost()` and measures the lookup duration.
2. Repeats the lookup and compares timings.
3. Opens a TCP connection to the host on port 80 using `net.Dial()`.
4. Writes a basic HTTP/1.1 request.
5. Reads and prints the response status line.
6. Reads headers until the blank line.
7. Reads the remaining response body.

Run from this folder:

```powershell
go run .
```

Implementation notes:
- Use `time.Now()` and `time.Since(start)` for elapsed time.
- `net.LookupHost()` returns `([]string, error)`.
- `net.Dial()` returns `(net.Conn, error)`.
- `conn.Write()` accepts bytes, so convert the request with `[]byte(request)`.
- HTTP/1.x lines use CRLF (`\r\n`) endings.
- `bufio.NewReader(conn)` makes it convenient to read one line at a time.
- `ReadString('\n')` returns the line including its delimiter.
- After reading headers, continue reading the body from the **same** `bufio.Reader`, because it may already have buffered body bytes.
- Put `defer conn.Close()` immediately after a successful connection so every later return path closes it.

### Example observations

One DNS timing run produced:

```text
[142.251.43.142]
70.6461ms
```

A later pair of lookups produced approximately:

```text
Attempt 0: 35 ms
Attempt 1: 1 ms
```

IP addresses and timings vary. A faster second lookup may be due to caching in the Go process, operating system or recursive resolver; timing alone does not identify which cache was responsible.

A request to `google.com:80` returned `301 Moved Permanently` and a `Location` header. This demonstrates that a redirect is an HTTP response, not an action performed by TCP.

## Exercise 2 — URL shortener estimation

See [`design/day-06-network-and-estimation.md`](design/day-06-network-and-estimation.md) for the full capacity and storage calculation.

## Reflection questions

Answer in your own words:

1. Why does TCP perform a handshake before application data is exchanged?
2. What trade-off does DNS TTL create?
3. Why can HTTP keep-alive make a second request faster?
4. Why should a raw HTTP client not assume every response body can be read until the TCP connection closes?
5. What storage should be considered beyond raw record size?

## Key takeaway

Opening a URL is a chain of operations: name resolution, connection establishment, application-protocol exchange and response processing. Understanding each layer helps diagnose latency and design reliable backend services.

## References

[HTTP versions](https://medium.com/%40rajeevprasanna/from-http-1-1-to-http-2-a-new-era-of-web-performance-c49f6657f45a)

[after entering URL ](https://medium.com/%40thesamventure/from-browser-to-server-what-really-happens-when-you-type-a-url-93187c1e9454)
