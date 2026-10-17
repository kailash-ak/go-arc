# Go Day 2 --- Concurrency, Goroutines, and Mutexes

This folder contains hands-on experiments for understanding how Go runs
concurrent tasks and how to protect shared data.

## Learning goals

By the end of Day 2, you should understand:

-   What goroutines are and how to start them with `go`.
-   How `sync.WaitGroup` waits for goroutines to finish.
-   Why shared variables can cause data races.
-   Why checking and updating shared state must be protected as one
    operation.
-   How `sync.Mutex` protects a critical section.
-   Why a program can appear to work correctly and still contain a
    concurrency bug.

## Project structure

``` text
go-arc/
├── go-day1/
└── go-day2/
    ├── go.mod
    ├── main.go
    ├── booking.go
    └── README.md
```

Your exact files may differ depending on which experiments you keep.

## 1. Run the project

Open a terminal in the `go-day2` directory and run:

``` powershell
go run .
```

The command runs the current Go package. If you have multiple `.go`
files in the same directory, they can belong to the same package and be
compiled together.

To build an executable instead of immediately running the program:

``` powershell
go build -o booking.exe .
.\booking.exe
```

After changing the source code, run `go build` again to update the
executable.

## 2. Goroutines

A goroutine is a function executing concurrently with other goroutines.

``` go
go doWork()
```

The `go` keyword starts the function in a new goroutine. It does not
guarantee that the function will run on a separate CPU core. Go's
runtime schedules goroutines onto operating-system threads.

**Concurrency** means multiple tasks can make progress during
overlapping periods. **Parallelism** means tasks are actually executing
at the same time, such as on different CPU cores.

## 3. Waiting for goroutines with `sync.WaitGroup`

A program's `main` function does not automatically wait for goroutines
it starts. When `main` returns, the process exits.

Use a `WaitGroup` when you need to wait for a group of goroutines to
finish:

``` go
var wg sync.WaitGroup

for i := 0; i < 5; i++ {
    wg.Add(1)

    go func() {
        defer wg.Done()
        // Perform work here.
    }()
}

wg.Wait()
```

-   `wg.Add(1)` registers one unit of work. Do this before starting the
    goroutine.
-   `wg.Done()` marks one unit of work as finished.
-   `wg.Wait()` blocks the calling goroutine until the counter reaches
    zero.
-   `defer wg.Done()` ensures `Done` is called when the goroutine's
    function returns.

A `WaitGroup` coordinates completion. It **does not protect shared
variables**.

## 4. The seat-booking example

Imagine 100 customers are trying to book the same seat. The program
uses:

``` go
booked := false
successes := 0
```

Each goroutine tries to:

1.  Check whether `booked` is false.
2.  Mark the seat as booked.
3.  Increase `successes`.

The unsafe pattern is:

``` go
if !booked {
    booked = true
    successes++
}
```

The check and the updates are separate operations. Two goroutines may
both observe `booked == false` before either marks it true. Both may
then count themselves as successful.

This is a **check-then-act race**. The variables are shared mutable
state, and accessing them concurrently without suitable synchronization
is unsafe.

The bug may not show up in every run. An output of `successes = 1` does
not prove the code is safe.

## 5. Why use `time.Sleep` in the demonstration?

``` go
time.Sleep(time.Millisecond)
```

This pauses the current goroutine for at least approximately one
millisecond. In the unsafe example, placing it between the availability
check and the update widens the opportunity for another goroutine to run
in between.

Sleep is only a demonstration technique. It does not coordinate
goroutines or make shared data safe, and it does not guarantee that a
race will occur.

A start gate can also make many goroutines begin competing around the
same time. Even then, the exact output is not guaranteed.

## 6. Fix the booking operation with a mutex

A mutex protects the complete check-and-update operation:

``` go
var mu sync.Mutex

mu.Lock()

if !booked {
    booked = true
    successes++
}

mu.Unlock()
```

The lock must be acquired **before reading `booked`** and released after
the related updates are complete.

If the first goroutine books the seat, the next goroutine must wait for
the lock. Once it acquires the lock, it sees `booked == true` and does
not book the seat again.

For a single seat, the expected final result is:

``` text
Successful bookings: 1
```

Use `defer mu.Unlock()` when appropriate to help ensure the mutex is
released when a function returns. Keep the critical section as small as
practical.

## 7. `WaitGroup` versus `Mutex`

  -----------------------------------------------------------------------
  Tool                                Purpose
  ----------------------------------- -----------------------------------
  `sync.WaitGroup`                    Waits for goroutines to finish.

  `sync.Mutex`                        Allows only one goroutine at a time
                                      into a protected critical section.

  `time.Sleep`                        Pauses a goroutine; it is not a
                                      synchronization tool.
  -----------------------------------------------------------------------

You may need both a `WaitGroup` and a `Mutex`: one to wait for
completion and the other to protect shared data.

## 8. Run the race detector (optional)

If your Go installation and C compiler support it, run:

``` powershell
go run -race .
```

The race detector instruments the program and reports data races that
occur during that run. It does not add locks or fix the code
automatically, and it cannot guarantee detection of races in execution
paths that were not exercised.

On some Windows setups, the race detector requires cgo and a compatible
C compiler. If you see an error such as `-race requires cgo`, you can
return to this step after configuring the toolchain.

## Key takeaways

-   Starting a goroutine does not mean it runs on a separate CPU core.
-   `main` returning ends the process, even if other goroutines are
    still running.
-   A `WaitGroup` waits; it does not make shared data safe.
-   A data race can exist even when the output looks correct.
-   Protect the entire check-and-update operation with a mutex---not
    just the assignment.
-   `time.Sleep` can help demonstrate a race but cannot fix one.

## Next steps

-   [ ] Run the unsafe booking experiment several times.
-   [ ] Explain why an output of `1` does not prove the code is safe.
-   [ ] Add a mutex around the complete booking check and update.
-   [ ] Compare the unsafe and protected versions.
-   [ ] Revisit `go run -race .` once cgo is configured.
