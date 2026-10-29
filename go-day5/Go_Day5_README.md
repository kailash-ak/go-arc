# Go Day 5 — Channels, Select, Context, and Goroutine Leaks

## Learning Goals

- Communicate between goroutines using channels.
- Understand buffered and unbuffered channels.
- Close channels and receive values safely.
- Use `select`, `time.After`, and `time.Ticker`.
- Cancel work with `context.WithCancel` and `context.WithTimeout`.
- Recognize and prevent common goroutine leaks.

## 1. Channels

A channel lets goroutines communicate by sending and receiving values.

```go
ch := make(chan int)

ch <- 10      // Send a value.
value := <-ch // Receive a value.
```

An **unbuffered channel** requires a sender and receiver to meet:

- `ch <- value` blocks until a receiver is ready.
- `value := <-ch` blocks until a sender is ready.

If the only goroutine tries to send without a receiver, the program can deadlock.

### Buffered channels

A buffered channel can queue a limited number of values without an immediate receiver.

```go
ch := make(chan int, 2)
ch <- 10
ch <- 20

fmt.Println(len(ch)) // 2: values currently buffered
fmt.Println(cap(ch)) // 2: total buffer capacity
```

When the buffer is full, another send blocks until a receiver makes room. A receive blocks when the channel is empty and still open.

| Channel | Send blocks when | Receive blocks when |
|---|---|---|
| Unbuffered | No receiver is ready | No sender is ready |
| Buffered | Buffer is full | Buffer is empty and channel is open |

A buffer is a queue, not a replacement for coordinating goroutines.

## 2. Closing a channel

Closing a channel tells receivers that no more values will be sent.

```go
ch := make(chan int, 3)
ch <- 1
ch <- 2
ch <- 3
close(ch)

for value := range ch {
    fmt.Println(value)
}
```

The loop prints `1`, `2`, and `3`, then finishes. Closing a buffered channel does not discard its buffered values.

Important rules:

- Buffered values remain readable after close.
- After a closed channel is drained, a receive returns the element type's zero value.
- Use the two-value receive to check whether a value was received:

```go
value, ok := <-ch
if !ok {
    fmt.Println("Channel is closed and drained")
}
```

- Sending to a closed channel panics.
- The sender that knows no more values will be sent is normally responsible for closing the channel.
- `for value := range ch` ends when the channel is closed and drained. If the channel remains open with no future sender, the loop can block forever.

## 3. Channel directions

```go
func sendData(ch chan<- int) {
    ch <- 100 // Send-only channel
}

func receiveData(ch <-chan int) {
    value := <-ch // Receive-only channel
    fmt.Println(value)
}
```

- `chan int`: bidirectional channel.
- `chan<- int`: send-only channel.
- `<-chan int`: receive-only channel.

Directional channel types clarify a function's responsibilities and prevent unintended operations.

## 4. The `select` statement

`select` waits for channel operations and executes one ready case.

```go
select {
case value := <-results:
    fmt.Println("Result:", value)
case <-time.After(time.Second):
    fmt.Println("Timed out")
}
```

This waits for either a result or a one-second timeout.

- If multiple cases are ready, Go chooses one pseudo-randomly; there is no guaranteed priority.
- Without `default`, `select` waits if no case is ready.
- With `default`, it executes immediately when no communication case is ready.

A non-blocking receive:

```go
select {
case value := <-ch:
    fmt.Println("Received:", value)
default:
    fmt.Println("No value ready")
}
```

Avoid a tight loop that repeatedly executes `default`; it can waste CPU.

## 5. `time.Sleep` versus `time.After`

### `time.Sleep`

Pauses the current goroutine for at least the specified duration.

```go
time.Sleep(700 * time.Millisecond)
cancel()
```

Here, `cancel()` runs after the sleep finishes.

### `time.After`

Returns a channel that becomes ready after a duration.

```go
select {
case result := <-results:
    fmt.Println(result)
case <-time.After(time.Second):
    fmt.Println("Timed out")
}
```

Calling `time.After(...)` by itself does **not** pause the current goroutine:

```go
time.After(700 * time.Millisecond)
cancel() // Runs immediately; the timer was not awaited.
```

Use `time.Sleep` to intentionally pause, or use the timer channel in a receive or `select` when waiting for an event or timeout.

## 6. Context cancellation

A context communicates cancellation and deadlines across function calls and goroutines. It does not forcibly kill a goroutine; the work must cooperate by checking the context.

### `context.WithCancel`

Use this when another part of the program decides when to stop work.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Later, when you want to request cancellation:
cancel()
```

Calling `cancel()` closes `ctx.Done()`. The context error becomes `context.Canceled`.

### `context.WithTimeout`

Use this when work should stop waiting after a time limit.

```go
ctx, cancel := context.WithTimeout(
    context.Background(),
    2*time.Second,
)
defer cancel()

<-ctx.Done()
fmt.Println(ctx.Err())
```

If the deadline expires first, the error is `context.DeadlineExceeded`. You can also call `cancel()` manually before the deadline.

| Feature | `WithCancel` | `WithTimeout` |
|---|---|---|
| Automatic cancellation | No deadline of its own | Yes, when the timeout expires |
| Manual cancellation | Yes | Yes |
| Typical use | Stop when told | Stop waiting after a time limit |

### `ctx.Done()` and `ctx.Err()`

- `ctx.Done()` returns a channel that closes when the context is cancelled or its deadline expires.
- Receiving from `ctx.Done()` waits until the signal is available.
- `ctx.Err()` reports the reason, usually `context.Canceled` or `context.DeadlineExceeded`.

### A worker that responds to cancellation

```go
func worker(ctx context.Context) {
    ticker := time.NewTicker(300 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            fmt.Println("Stopped:", ctx.Err())
            return
        case <-ticker.C:
            fmt.Println("Working")
        }
    }
}
```

Run it with a timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

worker(ctx)
```

The worker periodically prints `Working`, then stops when the context deadline is reached. The exact number of ticks can vary slightly.

If you start the worker with `go worker(ctx)`, coordinate with it using a `sync.WaitGroup` or a completion channel if you need to ensure it has finished. Do not rely on an arbitrary `time.Sleep` as synchronization.

## 7. Tickers

A ticker sends time events repeatedly at an approximate interval.

```go
ticker := time.NewTicker(300 * time.Millisecond)
defer ticker.Stop()

for range ticker.C {
    fmt.Println("Tick")
}
```

- `ticker.C` is a channel that delivers tick events.
- Receiving from `ticker.C` waits for a tick.
- `ticker.Stop()` stops future ticks.
- Use `defer ticker.Stop()` when the ticker should stop when the function returns.

Tickers are useful for periodic tasks such as polling device status. For real network automation, use context-aware network operations and suitable timeouts rather than relying only on simulated delays.

## 8. Goroutine leaks

A goroutine leak occurs when a goroutine remains alive without making useful progress, often because it is blocked forever on a channel operation that nobody can complete.

### Example: blocked send

```go
ch := make(chan int)

go func() {
    fmt.Println("Worker started")
    ch <- 10 // Blocks if nobody receives.
    fmt.Println("Worker finished")
}()
```

Because the channel is unbuffered, the send needs a receiver. If no receiver ever arrives, the worker cannot reach `Worker finished`.

A buffered channel can also block:

```go
ch := make(chan int, 1)

ch <- 10 // Buffer is now full.
ch <- 20 // Blocks until a value is received.
```

The buffer holds only one value, so the second send blocks if nobody receives.

### Make a send cancellation-aware

```go
select {
case ch <- value:
    // Value was sent.
case <-ctx.Done():
    // Stop waiting because the context was cancelled.
    return
}
```

This lets the goroutine stop waiting for the send when cancellation occurs.

Cancellation is cooperative: a goroutine must listen for cancellation and return. A goroutine blocked on an unrelated channel operation will not be released by context cancellation unless that operation is made cancellation-aware.

## 9. Quick revision

- **Channel:** communicates values between goroutines.
- **Unbuffered channel:** send and receive must rendezvous.
- **Buffered channel:** holds up to its capacity in queued values.
- **`close(ch)`:** signals that no more values will be sent; buffered values remain readable.
- **`select`:** waits for channel operations and runs one ready case.
- **`time.Sleep`:** pauses the current goroutine.
- **`time.After`:** provides a timer channel.
- **`context.WithCancel`:** cancellation is triggered manually or by a parent context.
- **`context.WithTimeout`:** cancellation occurs at the deadline or earlier if manually cancelled.
- **`ctx.Done()`:** cancellation/deadline signal channel.
- **`ctx.Err()`:** explains why the context ended.
- **`time.Ticker`:** produces recurring tick events.
- **Goroutine leak:** a goroutine gets stuck without a path to finish.

## 10. Practice checklist

- [ ] Create an unbuffered channel and demonstrate why a send blocks without a receiver.
- [ ] Create a buffered channel and print `len(ch)` and `cap(ch)`.
- [ ] Close a buffered channel and range over its remaining values.
- [ ] Use `select` to wait for a result or timeout.
- [ ] Write a worker that stops when `ctx.Done()` is signalled.
- [ ] Use a ticker and stop it with `defer ticker.Stop()`.
- [ ] Demonstrate a blocked send, then make the send cancellation-aware.

## Commands

Run these commands from the `go-day5` directory:

```powershell
go run .
go test ./...
```

From the repository root, stage and commit the Day 5 work:

```powershell
git status
git add go-day5
git commit -m "day 5: channels, select, and context"
git push
```
