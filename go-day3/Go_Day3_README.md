# Go Day 3 — Errors, Packages, and Testing

## What I learned

### 1. Error handling in Go
- Go commonly returns errors as values instead of using `try/catch`.
- A function can return a result and an `error`, for example: `func divide(a, b int) (int, error)`.
- Check errors with `if err != nil` before relying on the result.
- `nil` means the function reported no error; it does not automatically mean the operation achieved every business goal.
- Avoid ignoring errors with `_` unless there is a clear reason.

### 2. The `error` interface
Go's built-in `error` interface is equivalent to:

```go
type error interface {
    Error() string
}
```

A type implements `error` by providing an `Error() string` method.

### 3. Creating and formatting errors
- `errors.New("message")` creates a simple error value.
- `fmt.Errorf("message: %v", err)` formats an error message, but `%v` does not preserve the wrapped error chain.
- `fmt.Errorf("operation failed: %w", err)` wraps an error and preserves it for inspection.
- Two separate calls to `errors.New("same message")` create distinct error values; matching text does not make them the same error.

### 4. Sentinel errors and `errors.Is`
A sentinel error is a reusable error value declared once:

```go
var ErrNotFound = errors.New("user not found")
```

Return and compare the same sentinel:

```go
return "", ErrNotFound

if errors.Is(err, ErrNotFound) {
    // Handle the not-found case.
}
```

`errors.Is` checks whether an error matches a target through its wrapping chain (and also supports custom `Is` methods).

### 5. Custom error types and `errors.As`
Custom error types can carry extra information:

```go
type UserNotFoundError struct {
    UserID int
}

func (e UserNotFoundError) Error() string {
    return fmt.Sprintf("user with %d id not found", e.UserID)
}
```

Use `errors.As` when I need to extract a particular error type and access its fields:

```go
var userErr UserNotFoundError
if errors.As(err, &userErr) {
    fmt.Println(userErr.UserID)
}
```

- `errors.Is` asks: “Does this error match this error value?”
- `errors.As` asks: “Can I find an error of this type in the error chain?”

### 6. Wrapping errors from other packages
For example, `strconv.Atoi` can fail when parsing invalid text:

```go
age, err := strconv.Atoi(s)
if err != nil {
    return 0, fmt.Errorf("parse age %q: %w", s, err)
}
```

Using `%w` allows callers to inspect the original cause, such as with `errors.Is(err, strconv.ErrSyntax)`.

### 7. Packages and modules
- A **package** is made up of Go source files in the same directory that declare the same package.
- Files in the same package can use each other's package-level declarations without importing that package.
- A **module** is a collection of packages managed from a module root, usually identified by `go.mod`.
- An import path is the module path plus the package's subdirectory path.
- `package main` with a `func main()` entry point builds an executable.
- Identifiers starting with an uppercase letter are exported and can be accessed from another package. Lowercase identifiers are unexported to other packages.

### 8. Testing basics
- Test files end in `_test.go`.
- Test functions use the form `func TestSomething(t *testing.T)`.
- Table-driven tests store multiple inputs and expected results in a slice and test each case, often using `t.Run`.
- `t.Errorf` records a failure and continues; `t.Fatalf` records a failure and stops the current test.
- Run tests with `go test ./...`.

### 9. Typed nil and interfaces
A nil pointer stored inside an interface does **not** make the interface itself nil:

```go
var p *UserNotFoundError = nil
var err error = p

fmt.Println(p == nil)   // true
fmt.Println(err == nil) // false
fmt.Println(err != nil) // true
```

The interface holds a dynamic type (`*UserNotFoundError`) and a nil dynamic value. An interface compares equal to `nil` only when it has no dynamic type and no dynamic value. When returning “no error,” return a genuine `nil` error interface rather than a typed nil pointer converted to `error`.

### 10. Unwrapping an error chain
`errors.Unwrap(err)` returns the next error in a standard `%w` wrapping chain. A basic manual check can walk the chain:

```go
func has(err, target error) bool {
    for err != nil {
        if err == target {
            return true
        }
        err = errors.Unwrap(err)
    }
    return false
}
```

This is only a simplified learning exercise. In production, prefer `errors.Is`, which also handles custom `Is` methods and multi-error chains.

---

## Practice work / exercises

Use this checklist to mark what is completed in the repository.

- [ ] Write a function that returns an error when division by zero is attempted.
- [ ] Create a reusable `ErrDivideByZero` sentinel and check it with `errors.Is`.
- [ ] Create a separate `calc` package and call its exported `Divide` function from `main`.
- [ ] Confirm that a lowercase function such as `calc.divide` cannot be accessed from another package.
- [ ] Implement `ParseAge(s string) (int, error)` using `strconv.Atoi`.
- [ ] Wrap parse errors with `%w` and preserve `strconv.ErrSyntax`.
- [ ] Return a sentinel `ErrNegativeAge` for negative values.
- [ ] Create a custom `UserNotFoundError` carrying a user ID and retrieve it with `errors.As`.
- [ ] Implement a simple error-chain traversal using `errors.Unwrap`.
- [ ] Explain why `errors.New("x") == errors.New("x")` is false.
- [ ] Explain why an interface containing a typed nil pointer may not compare equal to `nil`.
- [ ] Add tests in `_test.go` files and run `go test ./...`.

> Only mark an exercise as completed if I actually implemented and ran it. Some items above may be planned practice rather than finished code.

## Commands to remember

```powershell
go run .
go test ./...
go test
```

Run these commands from the appropriate Go module/package directory.

## Day 3 takeaway

Go errors are values. I should return meaningful errors, wrap them with `%w` when the original cause matters, use `errors.Is` for sentinel matching, and use `errors.As` to retrieve typed errors. I also learned how packages and exported names work, and why typed nil values inside interfaces can be surprising.


