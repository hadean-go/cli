# CLI Exit Codes

Semantic exit code system for Go CLI applications, based on POSIX standards and BSD `sysexits.h`.

## Concept

The system provides a high-level abstraction for working with exit codes that:

- Compatible with existing standards (POSIX, BSD sysexits.h)
- Provides semantically meaningful error codes
- Supports error categorization
- Includes metadata (operation retryability, error type)

## Architecture

### Exit Code Categories

| Range | Category | Description |
|-------|----------|-------------|
| `0` | Success | Successful completion |
| `1-63` | General | General errors |
| `64-79` | User Error | User errors (sysexits.h) |
| `80-99` | CLI Extended | Extended CLI errors |
| `128+` | System/Signal | System signals |

### Main Components

1. `ExitCode` - type for representing exit codes
2. `ExitError` - error with associated exit code
3. Predefined constants - standard error codes
4. Helper functions - for creating typical errors
5. `Category` - type-safe category of codes

## Usage

### Basic Usage

```go
package main

import (
    "fmt"
    "os"
    "github.com/hadean-go/cli"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(int(cli.ResolveExitCode(err)))
    }
}

func run() error {
    // Examples of different error types
    
    // Data validation error
    if !isValidInput(input) {
        return cli.ValidationError("invalid input format")
    }
    
    // Resource not found
    if !fileExists(filename) {
        return cli.NotFoundError("config file")
    }
    
    // Insufficient permissions
    if !hasPermissions() {
        return cli.PermissionError("write to directory")
    }
    
    // Temporary error
    if networkUnavailable() {
        return cli.TempFailError("network connection failed")
    }
    
    return nil
}
```

### Working with ExitError

```go
// Creating error with specific exit code
err := cli.NewExitError(cli.ExitCodeConfig, "invalid configuration", nil)

// Getting error information
var exitErr *cli.ExitError
if errors.As(err, &exitErr) {
    fmt.Printf("Exit code: %d\n", exitErr.Code)
    fmt.Printf("Category: %s\n", exitErr.Code.Category())
    fmt.Printf("Is retriable: %v\n", exitErr.Code.IsRetriable())
    fmt.Printf("Is user error: %v\n", exitErr.Code.IsUserError())
}
```

### Integration with Existing Errors

```go
// Automatic exit code determination
func processFile(filename string) error {
    _, err := os.Open(filename)
    if err != nil {
        if os.IsNotExist(err) {
            return cli.NotFoundError("input file")
        }
        if os.IsPermission(err) {
            return cli.PermissionError("read file")
        }
        return fmt.Errorf("failed to open file: %w", err)
    }
    return nil
}

// In main function
func main() {
    err := processFile("config.json")
    exitCode := cli.ResolveExitCode(err)
    
    if exitCode != cli.ExitCodeSuccess {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(int(exitCode))
    }
}
```

### Retry Logic

```go
func withRetry(operation func() error, maxRetries int) error {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        exitCode := cli.ResolveExitCode(err)
        if !exitCode.IsRetriable() {
            return err // Don't retry operation
        }
        
        lastErr = err
        time.Sleep(time.Second * time.Duration(i+1))
    }
    
    return lastErr
}
```

## Code Reference

### Successful Completion (0)

| Code | Constant | Description |
|------|----------|-------------|
| `0` | `ExitCodeSuccess` | Successful completion |

### General Errors (1-63)

| Code | Constant | Description |
|------|----------|-------------|
| `1` | `ExitCodeError`, `ExitCodeErrorInternal` | General error |
| `2` | `ExitCodeUsageError`, `ExitCodeInvalidArgument` | Usage error |

### User Errors (64-79) - sysexits.h

| Code | Constant | Description |
|------|----------|-------------|
| `64` | `ExitCodeCmdUsage` | Incorrect command usage |
| `65` | `ExitCodeDataError` | Incorrect input data |
| `66` | `ExitCodeNoInput` | Input file not found |
| `67` | `ExitCodeNoUser` | User does not exist |
| `68` | `ExitCodeNoHost` | Host unavailable |
| `69` | `ExitCodeUnavailable` | Service unavailable |
| `70` | `ExitCodeSoftware` | Internal software error |
| `71` | `ExitCodeOSError` | OS error |
| `72` | `ExitCodeOSFile` | System file unavailable |
| `73` | `ExitCodeCantCreate` | Cannot create file |
| `74` | `ExitCodeIOError` | I/O error |
| `75` | `ExitCodeTempFail` | Temporary failure |
| `76` | `ExitCodeProtocol` | Protocol error |
| `77` | `ExitCodeNoPermission` | Insufficient permissions |
| `78` | `ExitCodeConfig` | Configuration error |

### Extended CLI Codes (80-99)

| Code | Constant | Description |
|------|----------|-------------|
| `80` | `ExitCodeAuthRequired` | Authentication required |
| `81` | `ExitCodeAuthFailed` | Authentication failed |
| `82` | `ExitCodeForbidden` | Operation forbidden |
| `83` | `ExitCodeNotFound` | Resource not found |
| `84` | `ExitCodeConflict` | Resource conflict |
| `85` | `ExitCodeValidation` | Validation error |
| `86` | `ExitCodeRateLimit` | Request rate limit exceeded |
| `87` | `ExitCodeQuotaExceeded` | Quota exceeded |

### System Signals (128+)

| Code | Constant | Description |
|------|----------|-------------|
| `130` | `ExitCodeInterrupted` | Interrupted by user (SIGINT) |
| `143` | `ExitCodeTerminated` | Terminated by system (SIGTERM) |

## ExitCode Methods

### `String() string`

Returns a human-readable description of the code.

```go
code := cli.ExitCodeNotFound
fmt.Println(code.String()) // "Not found"
```

### `Category() Category`

Returns the exit code category.

```go
code := cli.ExitCodeValidation
fmt.Println(code.Category()) // "cli_extended"
```

### `IsRetriable() bool`

Indicates whether the operation should be retried. Retries are recommended for: `TempFail`, `Unavailable`, `IOError`, `RateLimit`.

```go
code := cli.ExitCodeTempFail
fmt.Println(code.IsRetriable()) // true
```

### `IsUserError() bool`

Indicates whether the error is a result of user actions. System/temporary errors are excluded.

```go
code := cli.ExitCodeValidation
fmt.Println(code.IsUserError()) // true
```

### Marshaling

```go
// JSON representation of ExitError
var ee *cli.ExitError = cli.ValidationError("invalid input")
data, _ := json.Marshal(ee)
// {"code":85,"name":"Validation error","category":"cli_extended","message":"invalid input"}

// TextMarshaler for ExitCode
code := cli.ExitCodeNotFound
text, _ := code.MarshalText() // []byte("83")
```

### HTTP Mapping

```go
status := 404
code := cli.FromHTTPStatus(status) // ExitCodeNotFound

respStatus := cli.ToHTTPStatus(code) // 404
```

## Backward Compatibility

The API is simplified and does not guarantee backward compatibility with earlier versions; use current constants and the `Category` type.

## Best Practices

1. Use semantic codes: Choose the most appropriate code for each situation
2. Check retryability: Use `IsRetriable()` for retry decisions
3. Distinguish user and system errors: Use `IsUserError()`
4. Provide context: Add descriptive messages to errors
5. Follow standards: Use codes according to their purpose

## Integration with Other Tools

### CI/CD Systems

```bash
# Check error type in scripts
exit_code=$?
if [ $exit_code -ge 64 ] && [ $exit_code -le 79 ]; then
    echo "User error occurred"
elif [ $exit_code -ge 80 ] && [ $exit_code -le 99 ]; then
    echo "CLI extended error occurred"
fi
```

### Monitoring

```go
// Metrics for different error categories
func reportError(err error) {
    exitCode := cli.ResolveExitCode(err)
    category := exitCode.Category()
    
    metrics.Counter("cli_errors_total").
        WithLabelValues(category).
        Inc()
}
```
