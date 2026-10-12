Go Day 1 — Getting Started, Building, and Concurrency
Today I set up my first Go project and learned how Go turns source code into an executable, and how goroutines run concurrently.
1. Project setup
My project folder is go-day1.
go-day1/
├── main.go
├── go.mod
└── hello.exe       # created after building on Windows

2. What is go.mod?
Create it once for a new module:
go mod init go-day1
go.mod identifies the Go module and records its Go version and dependency requirements.
Example:
module go-day1

go 1.24.0
The version in my file may be different.
A module is a collection of Go packages managed together. As the project grows, go.mod helps Go identify and manage external dependencies. The fmt package is part of Go's standard library, so I do not need to install it separately.
3. Commands I learned
Run these commands in the terminal opened inside the go-day1 folder.
Run the program
go run .
This builds and runs the package in the current directory. It is convenient while developing; it does not normally create a named executable in my project folder.
Build a named executable
go build -o hello.exe .
Command breakdown:
- go build: compile the package and produce an executable.
- -o hello.exe: choose the output file name.
- .: build the package in the current directory.
If the build succeeds, it usually prints nothing. It creates or updates hello.exe; it does not run it.
Run the executable
.\hello.exe
This runs the existing executable on Windows.
Inspect Go build information
go version -m .\hello.exe
This can show embedded build information, including the Go version, module information, and build settings.
Check the executable's size
Get-Item .\hello.exe | Select-Object Name, Length
Length is the file size in bytes.
4. What happens inside go build?
A simplified build pipeline is:
1. Load packages and check the program — Go resolves imports and checks syntax and types.
2. Compile — the compiler transforms Go source through internal representations into machine code for the target architecture.
3. Link — the linker resolves references and combines compiled code with the required library and runtime code.
4. Write the executable — on Windows, the output is normally a PE-format executable such as hello.exe.
The Go runtime supports language features such as goroutine scheduling and garbage collection. A small program does not necessarily exercise every runtime feature.
Important: source code vs executable
Changing main.go does not automatically change an already-built hello.exe.
After editing the source, rebuild it:
go build -o hello.exe .
Then run the new executable:
.\hello.exe
Think of source code as a recipe and the executable as a cake: changing the recipe does not change a cake already baked.
5. Sequential execution
This example runs three tasks one after another:
package main

import (
    "fmt"
    "time"
)

func task(name string) {
    fmt.Println(name, "started")
    time.Sleep(2 * time.Second)
    fmt.Println(name, "finished")
}

func main() {
    start := time.Now()

    task("Task 1")
    task("Task 2")
    task("Task 3")

    fmt.Println("Total time:", time.Since(start))
}
Each call waits for its function to return before the next call begins. Because each task sleeps for two seconds, the total is about six seconds, plus overhead.
time.Sleep is being used to demonstrate waiting. It does not represent useful computation.
6. Concurrency with goroutines
A goroutine is a lightweight unit of execution managed by the Go runtime. The go keyword starts a function call in a new goroutine so the caller can continue without waiting for that function to finish.
package main

import (
    "fmt"
    "sync"
    "time"
)

func task(name string, wg *sync.WaitGroup) {
    defer wg.Done()

    fmt.Println(name, "started")
    time.Sleep(2 * time.Second)
    fmt.Println(name, "finished")
}

func main() {
    start := time.Now()

    var wg sync.WaitGroup
    wg.Add(3)

    go task("Task 1", &wg)
    go task("Task 2", &wg)
    go task("Task 3", &wg)

    wg.Wait()

    fmt.Println("Total time:", time.Since(start))
}
The total time is typically about two seconds plus overhead, rather than six seconds, because the three waits overlap. The order of the printed messages can vary.
sync.WaitGroup explained
- var wg sync.WaitGroup creates a wait group.
- wg.Add(3) says three task completions are being tracked.
- defer wg.Done() decrements the counter when each task function returns.
- wg.Wait() blocks the calling goroutine until the counter reaches zero.
defer schedules wg.Done() to run when task returns, including if the function returns early.
7. Why does the program exit without wg.Wait()?
Consider:
package main

import (
    "fmt"
    "time"
)

func main() {
    go func() {
        time.Sleep(2 * time.Second)
        fmt.Println("Worker finished!")
    }()

    fmt.Println("Main finished!")
}
The main goroutine starts the worker and then reaches the end of main(). When main() returns, the whole program exits, even if the worker goroutine has not finished.
So the output will typically only be:
Main finished!
The worker's message is not guaranteed to print.
If I add this at the end of main():
time.Sleep(3 * time.Second)
the worker will usually get enough time to print its message. This is only a demonstration; use proper coordination such as a sync.WaitGroup when you need to wait for tasks.
Key point: wg.Wait() blocks the goroutine that calls it—not the entire Go runtime. While main is waiting, worker goroutines can continue executing.
8. Concurrency vs parallelism
- Concurrency: multiple tasks can make progress during overlapping periods.
- Parallelism: multiple tasks execute at the same instant, typically on different CPU cores.
Goroutines make concurrency convenient. The Go runtime scheduler decides when goroutines run and on which operating-system threads. Whether work runs in parallel depends on available CPU resources and runtime configuration.
Concurrency does not automatically make every program faster. It is especially useful when independent operations spend time waiting, such as database queries or network requests. Real programs must also handle errors, shared state, cancellation, and coordination safely.
9. Quick reference
Command or concept	What it does
go mod init go-day1	Creates module metadata
go run .	Builds and runs the current package
go build -o hello.exe .	Builds a named executable
.\hello.exe	Runs the executable already on disk
go version -m .\hello.exe	Shows embedded Go build information when available
go keyword	Starts a function call in a new goroutine
sync.WaitGroup	Coordinates waiting for a group of tasks
wg.Wait()	Waits until the wait-group counter reaches zero
time.Sleep(...)	Pauses the current goroutine for a duration


10. Questions to review
1. Does changing main.go automatically update hello.exe?
2. What do -o and . mean in go build -o hello.exe .?
3. What information does go.mod keep?
4. Why can three concurrent two-second waits finish in roughly two seconds?
5. What does wg.Add(3) do?
6. Why can the program exit before a worker goroutine finishes?
7. What is the difference between concurrency and parallelism?

Today's main takeaway
I learned how to create and run a basic Go module, build a Windows executable, distinguish source code from the built executable, and use goroutines with a sync.WaitGroup to coordinate concurrent tasks.