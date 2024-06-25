
<div align="center">
    <img src="logo.png" width="40%"/>
    <br/>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
    <a href="https://codecov.io/gh/leaanthony/adfer"><img src="https://codecov.io/gh/leaanthony/adfer/branch/main/graph/badge.svg" alt="codecov"></a>
    <a href="https://goreportcard.com/report/github.com/leaanthony/adfer"><img src="https://goreportcard.com/badge/github.com/leaanthony/adfer" alt="Go Report Card"></a>
    <a href="https://godoc.org/github.com/leaanthony/adfer"><img src="https://godoc.org/github.com/leaanthony/adfer?status.svg" alt="GoDoc"></a>
    <a href="https://GitHub.com/leaanthony/adfer/releases/"><img src="https://img.shields.io/github/release/leaanthony/adfer.svg" alt="GitHub release"></a>
</div>

This Go library provides a flexible way to handle panics across your application, including in goroutines. 
It allows for custom error handling, dumping errors to a file, optionally exiting the program after a panic occurs, 
including system information in crash reports, and managing crash reports.

## Features

- Custom error handling
- Panic recovery in goroutines
- Option to dump errors to a JSON file
- Option to exit the program after handling a panic
- Option to include system information in crash reports
- Retrieve last N crash reports
- Wipe crash file on startup or initialization
- Add custom metadata to crash reports
- Easy integration with existing Go applications

## Installation

```bash
go get github.com/leaanthony/adfer
```

## Usage

Here are some examples of how to use adfer in your applications.

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/leaanthony/adfer"
)

func main() {
    // Initialize the panic handler
    ph := adfer.New()

    // Use SafeGo for goroutines
    ph.SafeGo(func() {
        // This panic will be caught and handled
        panic("oops in goroutine")
    })

    // For the main thread, use defer
    defer ph.Recover()

    // This panic will also be caught and handled
    panic("oops in main")
}
```

### Custom Error Handler

```go
ph := panichandler.New(panichandler.WithErrorHandler(func(err error, stack []byte) {
    fmt.Printf("Custom handler: %v\n", err)
}))
```

### Dump to File

```go
ph := panichandler.New(panichandler.WithDumpToFile("/path/to/panic.log"))
```

### Exit on Panic

```go
ph := panichandler.New(panichandler.WithExitOnPanic())
```

### Include System Information

```go
ph := panichandler.New(panichandler.WithSystemInfo())
```

### Wipe Crash File on Initialization

```go
ph := panichandler.New(panichandler.WithWipeFileOnInit())
```

### Add Custom Metadata

```go
ph := panichandler.New(panichandler.WithMetadata(map[string]string{
    "version": "1.2.3",
    "environment": "production",
}))
```

### Combining Options

```go
ph := panichandler.New(
    panichandler.WithErrorHandler(customHandler),
    panichandler.WithDumpToFile("/path/to/panic.log"),
    panichandler.WithExitOnPanic(),
    panichandler.WithSystemInfo(),
    panichandler.WithWipeFileOnInit(),
    panichandler.WithMetadata(map[string]string{
        "version": "1.2.3",
        "environment": "production",
    }),
)
```

### Retrieving Last N Crash Reports

```go
reports, err := ph.GetLastNCrashReports(5)
if err != nil {
    fmt.Printf("Error retrieving crash reports: %v\n", err)
} else {
    for _, report := range reports {
        fmt.Printf("Crash at %v: %s\n", report.Timestamp, report.Error)
    }
}
```

### Wiping Crash File

```go
err := ph.WipeCrashFile()
if err != nil {
    fmt.Printf("Error wiping crash file: %v\n", err)
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
