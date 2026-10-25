# Go Day 4 --- JSON, `io.Reader`, and `io.Writer`

## Overview

Day 4 covered JSON encoding/decoding, exported and unexported fields,
struct tags, and Go's reader/writer interfaces.

## 1. `json.Marshal`

-   `json.Marshal(v)` converts a Go value into JSON and returns
    `([]byte, error)`.
-   Use `string(data)` to print the JSON bytes as text.
-   Check the error before using the result.

``` go
data, err := json.Marshal(u)
if err != nil {
    fmt.Println("marshal error:", err)
    return
}
fmt.Println(string(data))
```

## 2. Exported fields, struct tags, and `omitempty`

-   Identifiers starting with uppercase letters are exported; lowercase
    identifiers are unexported.
-   `encoding/json` encodes exported fields and ignores unexported
    fields.
-   Struct tags customize JSON keys and options.
-   JSON keys may be uppercase or lowercase; JSON does not require
    lowercase keys.
-   `omitempty` omits a field when its value is empty/zero according to
    its type.

``` go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email,omitempty"`
    password string
}
```

For `User{1, "Kailash", "", "secret"}`, JSON is:

``` json
{"id":1,"name":"Kailash"}
```

The empty email is omitted because of `omitempty`; `password` is omitted
because it is unexported. A tag cannot make an unexported field
exportable. Without `omitempty`, an empty email appears as `"email":""`.

## 3. `json.Unmarshal`

-   `json.Unmarshal(data, &destination)` decodes JSON bytes into a Go
    value.
-   It needs a pointer to the destination so it can populate the
    variable.
-   Unknown JSON keys are ignored by default when decoding into a
    struct.
-   Incompatible types, such as JSON string `"abc"` for an `int` field,
    return an error.
-   Unexported fields are ignored during unmarshalling too.

``` go
data := []byte(`{"id":2,"name":"Mahesh","extra":true}`)
var u User
err := json.Unmarshal(data, &u)
```

The unknown `extra` key is ignored by default. Passing `u` instead of
`&u` returns an error because the destination is not a pointer.

## 4. `io.Reader`

``` go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

-   `p` is the buffer, `n` is the number of bytes read, and `err`
    reports an error or end-of-input.
-   A read does not have to fill the whole buffer.
-   `io.EOF` signals end-of-input.
-   Process `n > 0` before handling `err`, because a reader may return
    data and an error together.
-   Use `buf[:n]` to process only the bytes read.

### Custom `readAll` exercise

``` go
func readAll(r io.Reader) (int, error) {
    buf := make([]byte, 4)
    total := 0

    for {
        n, err := r.Read(buf)
        if n > 0 {
            fmt.Println("read", n, "bytes", string(buf[:n]))
            total += n
        }
        if err == io.EOF {
            return total, nil
        }
        if err != nil {
            return total, err
        }
    }
}
```

Calling `readAll(strings.NewReader("hello world"))` prints chunks
`hell`, `o wo`, and `rld`, then reports `total: 11 <nil>`. In
production, use `io.ReadAll` when you need to collect all input into a
byte slice.

## 5. `io.Writer`

``` go
type Writer interface {
    Write(p []byte) (n int, err error)
}
```

A writer is a destination for data. `os.Stdout` writes to the terminal;
`*bytes.Buffer` stores data in memory. Both implement `io.Writer`.

``` go
func greet(w io.Writer, name string) {
    fmt.Fprintf(w, "hello %s\n", name)
}
```

To write to the terminal:

``` go
greet(os.Stdout, "Kailash")
```

To write into memory:

``` go
var b bytes.Buffer
greet(&b, "Kailash")
fmt.Printf("%q\n", b.String())
```

Expected quoted output:

``` text
"hello Kailash\n"
```

One function supports multiple destinations because it accepts the
interface rather than a concrete type. `fmt.Fprintf` returns a byte
count and an error; production code should check or propagate that
error.

## 6. `json.NewEncoder`

`json.NewEncoder(w).Encode(v)` encodes a Go value as JSON and writes it
directly to an `io.Writer`.

``` go
err := json.NewEncoder(os.Stdout).Encode(u)
if err != nil {
    fmt.Println("encode error:", err)
    return
}
```

-   `json.Marshal` returns JSON bytes (`[]byte`).
-   `json.NewEncoder(w).Encode(v)` writes directly to the writer.
-   `Encode` appends a trailing newline.
-   Use `&bytes.Buffer` when you want to keep the encoded output in
    memory.

## Key takeaways

-   `Marshal`: Go value → JSON bytes.
-   `Unmarshal`: JSON bytes → Go value, using a pointer destination.
-   Struct tags customize JSON representation; they do not export
    fields.
-   `io.Reader` abstracts data sources.
-   `io.Writer` abstracts data destinations.
-   Interfaces let one function work with different concrete types.
-   `json.Encoder` writes JSON to an `io.Writer` and adds a newline.

## Day 4 completion checklist

-   [ ] Practised `json.Marshal`.
-   [ ] Understood exported/unexported fields, tags, and `omitempty`.
-   [ ] Practised `json.Unmarshal` with a pointer destination.
-   [ ] Tested unknown JSON keys and incompatible field types.
-   [ ] Implemented `readAll(io.Reader)` and counted bytes.
-   [ ] Implemented a writer-based greeting function.
-   [ ] Practised `json.NewEncoder` with `os.Stdout` and/or
    `bytes.Buffer`.
-   [ ] Ran the Day 4 code successfully.
-   [ ] Committed the work with message `day 4: json, io`.

## Useful commands

From the repository root:

``` powershell
go run ./go-day4
git status
git add go-day4
git commit -m "day 4: json, io"
git push
```

If `go-day4` has its own nested `go.mod`, run Go commands from inside
that directory as appropriate. Nested modules are not included in the
root module's `go test ./...` traversal.
