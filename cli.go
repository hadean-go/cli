package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// ExitCode represents a semantic program exit code.
// The concept is based on POSIX standards, BSD sysexits.h.
type ExitCode int

// Category is a type-safe category for exit codes
type Category string

const (
	CategorySuccess      Category = "success"
	CategoryGeneral      Category = "general"
	CategoryUserError    Category = "user_error"
	CategoryCLIExtended  Category = "cli_extended"
	CategorySystemSignal Category = "system_signal"
	CategoryUnknown      Category = "unknown"
)

// Main exit code categories
const (
	// ===== SUCCESS CODES =====
	// ExitCodeSuccess successful program completion (compatibility with existing code)
	ExitCodeSuccess ExitCode = 0

	// ===== GENERAL ERROR CODES (1-63) =====
	// ExitCodeErrorInternal general internal error (compatibility)
	ExitCodeErrorInternal ExitCode = 1
	// ExitCodeError alternative name for general error
	ExitCodeError ExitCode = 1

	// ExitCodeInvalidArgument incorrect command or argument usage
	ExitCodeInvalidArgument ExitCode = 2
	// ExitCodeUsageError alternative name for usage error
	ExitCodeUsageError ExitCode = 2

	// ===== USER ERROR CODES (64-79) - sysexits.h based =====
	// ExitCodeCmdUsage incorrect command usage (arguments, flags, syntax)
	ExitCodeCmdUsage ExitCode = 64

	// ExitCodeDataError incorrect user input data
	ExitCodeDataError ExitCode = 65

	// ExitCodeNoInput file does not exist or is not accessible for reading
	ExitCodeNoInput ExitCode = 66

	// ExitCodeNoUser specified user does not exist
	ExitCodeNoUser ExitCode = 67

	// ExitCodeNoHost specified host is unavailable
	ExitCodeNoHost ExitCode = 68

	// ExitCodeUnavailable service unavailable
	ExitCodeUnavailable ExitCode = 69

	// ExitCodeSoftware internal software error
	ExitCodeSoftware ExitCode = 70

	// ExitCodeOSError operating system error
	ExitCodeOSError ExitCode = 71

	// ExitCodeOSFile system file unavailable or corrupted
	ExitCodeOSFile ExitCode = 72

	// ExitCodeCantCreate cannot create output file
	ExitCodeCantCreate ExitCode = 73

	// ExitCodeIOError input/output error
	ExitCodeIOError ExitCode = 74

	// ExitCodeTempFail temporary failure (should retry later)
	ExitCodeTempFail ExitCode = 75

	// ExitCodeProtocol network protocol error
	ExitCodeProtocol ExitCode = 76

	// ExitCodeNoPermission insufficient permissions to perform operation
	ExitCodeNoPermission ExitCode = 77

	// ExitCodeConfig configuration error
	ExitCodeConfig ExitCode = 78

	// ===== EXTENDED CLI CODES (80-99) =====
	// ExitCodeAuthRequired authentication required
	ExitCodeAuthRequired ExitCode = 80

	// ExitCodeAuthFailed authentication failed
	ExitCodeAuthFailed ExitCode = 81

	// ExitCodeForbidden operation forbidden (authorization failed)
	ExitCodeForbidden ExitCode = 82

	// ExitCodeNotFound resource not found
	ExitCodeNotFound ExitCode = 83

	// ExitCodeConflict resource conflict
	ExitCodeConflict ExitCode = 84

	// ExitCodeValidation data validation error
	ExitCodeValidation ExitCode = 85

	// ExitCodeRateLimit request rate limit exceeded
	ExitCodeRateLimit ExitCode = 86

	// ExitCodeQuotaExceeded quota exceeded
	ExitCodeQuotaExceeded ExitCode = 87

	// ===== SYSTEM/SIGNAL CODES (128+) =====
	// ExitCodeInterrupted process interrupted by user (Ctrl+C, SIGINT)
	ExitCodeInterrupted ExitCode = 130

	// ExitCodeTerminated process terminated by system (SIGTERM)
	ExitCodeTerminated ExitCode = 143
)

// String returns a human-readable description of the exit code
func (c ExitCode) String() string {
	switch c {
	case ExitCodeSuccess:
		return "Success"
	case ExitCodeErrorInternal:
		return "Internal error"
	case ExitCodeInvalidArgument:
		return "Invalid argument"
	case ExitCodeCmdUsage:
		return "Command usage error"
	case ExitCodeDataError:
		return "Data format error"
	case ExitCodeNoInput:
		return "Input file not found"
	case ExitCodeNoUser:
		return "User not found"
	case ExitCodeNoHost:
		return "Host not found"
	case ExitCodeUnavailable:
		return "Service unavailable"
	case ExitCodeSoftware:
		return "Internal software error"
	case ExitCodeOSError:
		return "Operating system error"
	case ExitCodeOSFile:
		return "System file error"
	case ExitCodeCantCreate:
		return "Cannot create output file"
	case ExitCodeIOError:
		return "I/O error"
	case ExitCodeTempFail:
		return "Temporary failure"
	case ExitCodeProtocol:
		return "Protocol error"
	case ExitCodeNoPermission:
		return "Permission denied"
	case ExitCodeConfig:
		return "Configuration error"
	case ExitCodeAuthRequired:
		return "Authentication required"
	case ExitCodeAuthFailed:
		return "Authentication failed"
	case ExitCodeForbidden:
		return "Forbidden"
	case ExitCodeNotFound:
		return "Not found"
	case ExitCodeConflict:
		return "Conflict"
	case ExitCodeValidation:
		return "Validation error"
	case ExitCodeRateLimit:
		return "Rate limit exceeded"
	case ExitCodeQuotaExceeded:
		return "Quota exceeded"
	case ExitCodeInterrupted:
		return "Interrupted by user"
	case ExitCodeTerminated:
		return "Terminated by system"
	default:
		return fmt.Sprintf("Unknown exit code: %d", int(c))
	}
}

// Category returns the category of the exit code
func (c ExitCode) Category() Category {
	switch {
	case c == 0:
		return CategorySuccess
	case c >= 1 && c <= 63:
		return CategoryGeneral
	case c >= 64 && c <= 79:
		return CategoryUserError
	case c >= 80 && c <= 99:
		return CategoryCLIExtended
	case c >= 128:
		return CategorySystemSignal
	default:
		return CategoryUnknown
	}
}

// IsRetriable indicates whether the operation should be retried for this code
func (c ExitCode) IsRetriable() bool {
	switch c {
	case ExitCodeTempFail, ExitCodeUnavailable, ExitCodeIOError, ExitCodeRateLimit:
		return true
	default:
		return false
	}
}

// IsUserError indicates whether the error is a result of incorrect user actions
func (c ExitCode) IsUserError() bool {
	switch c {
	case ExitCodeInvalidArgument, // == ExitCodeUsageError
		ExitCodeCmdUsage,
		ExitCodeDataError,
		ExitCodeNoInput,
		ExitCodeNoUser,
		ExitCodeNoHost,
		ExitCodeNoPermission,
		ExitCodeConfig,
		ExitCodeNotFound,
		ExitCodeValidation:
		return true
	default:
		return false
	}
}

// MarshalText implements encoding.TextMarshaler (stable numeric string)
func (c ExitCode) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(c))), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (c *ExitCode) UnmarshalText(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "" {
		*c = ExitCodeSuccess
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid exit code text: %q", s)
	}
	*c = ExitCode(n)
	return nil
}

// ===== PREDEFINED ERRORS =====

var (
	// ErrInternal general internal error (compatibility)
	ErrInternal = errors.New("internal error")
	// ErrInvalid invalid argument passed (compatibility)
	ErrInvalid = errors.New("invalid argument")

	// Extended errors
	// ErrUsage command usage error
	ErrUsage = errors.New("usage error")
	// ErrDataFormat data format error
	ErrDataFormat = errors.New("data format error")
	// ErrNotFound resource not found
	ErrNotFound = errors.New("not found")
	// ErrNoPermission insufficient permissions
	ErrNoPermission = errors.New("permission denied")
	// ErrConfig configuration error
	ErrConfig = errors.New("configuration error")
	// ErrAuth authentication error
	ErrAuth = errors.New("authentication error")
	// ErrForbidden operation forbidden
	ErrForbidden = errors.New("forbidden")
	// ErrValidation validation error
	ErrValidation = errors.New("validation error")
	// ErrIO input/output error
	ErrIO = errors.New("I/O error")
	// ErrUnavailable service unavailable
	ErrUnavailable = errors.New("service unavailable")
	// ErrTempFail temporary failure
	ErrTempFail = errors.New("temporary failure")
)

// ExitError represents an error with an associated exit code
type ExitError struct {
	Code    ExitCode
	Message string
	Cause   error
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Code.String()
}

func (e *ExitError) Unwrap() error {
	return e.Cause
}

// NewExitError creates a new error with an exit code
func NewExitError(code ExitCode, message string, cause error) *ExitError {
	return &ExitError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Newf creates a new error with a formatted message and code
func Newf(code ExitCode, format string, args ...any) *ExitError {
	return NewExitError(code, fmt.Sprintf(format, args...), nil)
}

// WithCode wraps an existing error and assigns it a code
func WithCode(err error, code ExitCode) *ExitError {
	if err == nil {
		return nil
	}
	// Preserve original error text in Message and the error itself in Cause
	return NewExitError(code, err.Error(), err)
}

// MarshalJSON implements json.Marshaler for structured logging/transport
func (e *ExitError) MarshalJSON() ([]byte, error) {
	type alias struct {
		Code     int      `json:"code"`
		Name     string   `json:"name"`
		Category Category `json:"category"`
		Message  string   `json:"message"`
		Cause    string   `json:"cause,omitempty"`
	}
	var cause string
	if e.Cause != nil {
		cause = e.Cause.Error()
	}
	return json.Marshal(alias{
		Code:     int(e.Code),
		Name:     e.Code.String(),
		Category: e.Code.Category(),
		Message:  e.Error(),
		Cause:    cause,
	})
}

// ResolveExitCode determines the exit code based on an error
func ResolveExitCode(err error) ExitCode {
	if err == nil {
		return ExitCodeSuccess
	}

	// Check for ExitError
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}

	// Mapping of common standard library errors
	if errors.Is(err, context.Canceled) {
		return ExitCodeInterrupted
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ExitCodeTempFail
	}
	// os errors
	if os.IsNotExist(err) {
		// For local resources return NoInput (sysexits: 66)
		return ExitCodeNoInput
	}
	if os.IsPermission(err) {
		return ExitCodeNoPermission
	}
	// net errors
	var ne net.Error
	if errors.As(err, &ne) {
		if ne.Timeout() {
			return ExitCodeTempFail
		}
		// If error is marked as temporary - also TempFail
		if te, ok := any(ne).(interface{ Temporary() bool }); ok && te.Temporary() {
			return ExitCodeTempFail
		}
		// Otherwise consider service unavailable
		return ExitCodeUnavailable
	}

	// Check predefined errors (compatibility with existing code)
	switch {
	case errors.Is(err, ErrInternal):
		return ExitCodeErrorInternal
	case errors.Is(err, ErrInvalid):
		return ExitCodeInvalidArgument
	// New checks
	case errors.Is(err, ErrUsage):
		return ExitCodeUsageError
	case errors.Is(err, ErrDataFormat):
		return ExitCodeDataError
	case errors.Is(err, ErrNotFound):
		return ExitCodeNotFound
	case errors.Is(err, ErrNoPermission):
		return ExitCodeNoPermission
	case errors.Is(err, ErrConfig):
		return ExitCodeConfig
	case errors.Is(err, ErrAuth):
		return ExitCodeAuthFailed
	case errors.Is(err, ErrForbidden):
		return ExitCodeForbidden
	case errors.Is(err, ErrValidation):
		return ExitCodeValidation
	case errors.Is(err, ErrIO):
		return ExitCodeIOError
	case errors.Is(err, ErrUnavailable):
		return ExitCodeUnavailable
	case errors.Is(err, ErrTempFail):
		return ExitCodeTempFail
	default:
		return ExitCodeErrorInternal
	}
}

// ===== HELPER FUNCTIONS =====

// UsageError creates an incorrect usage error
func UsageError(message string) *ExitError {
	return NewExitError(ExitCodeUsageError, message, ErrUsage)
}

// ValidationError creates a validation error
func ValidationError(message string) *ExitError {
	return NewExitError(ExitCodeValidation, message, ErrValidation)
}

// ConfigError creates a configuration error
func ConfigError(message string) *ExitError {
	return NewExitError(ExitCodeConfig, message, ErrConfig)
}

// NotFoundError creates a "not found" error
func NotFoundError(resource string) *ExitError {
	return NewExitError(ExitCodeNotFound, fmt.Sprintf("%s not found", resource), ErrNotFound)
}

// PermissionError creates a permission denied error
func PermissionError(action string) *ExitError {
	return NewExitError(ExitCodeNoPermission, fmt.Sprintf("permission denied: %s", action), ErrNoPermission)
}

// AuthError creates an authentication error
func AuthError(message string) *ExitError {
	return NewExitError(ExitCodeAuthFailed, message, ErrAuth)
}

// TempFailError creates a temporary failure error
func TempFailError(message string) *ExitError {
	return NewExitError(ExitCodeTempFail, message, ErrTempFail)
}

// OSExitCode returns an integer code for use with os.Exit
func OSExitCode(err error) int {
	return int(ResolveExitCode(err))
}

// FromHTTPStatus maps HTTP status to ExitCode
func FromHTTPStatus(status int) ExitCode {
	switch status {
	case 200, 201, 202, 204:
		return ExitCodeSuccess
	case 400:
		return ExitCodeDataError
	case 401:
		return ExitCodeAuthFailed
	case 403:
		return ExitCodeForbidden
	case 404:
		return ExitCodeNotFound
	case 409:
		return ExitCodeConflict
	case 412, 422:
		return ExitCodeValidation
	case 423:
		return ExitCodeForbidden
	case 429:
		return ExitCodeRateLimit
	case 500:
		return ExitCodeSoftware
	case 501, 502, 503, 504:
		return ExitCodeUnavailable
	default:
		if status >= 200 && status < 300 {
			return ExitCodeSuccess
		}
		if status >= 400 && status < 500 {
			return ExitCodeDataError
		}
		if status >= 500 {
			return ExitCodeUnavailable
		}
		return ExitCodeErrorInternal
	}
}

// ToHTTPStatus maps ExitCode to recommended HTTP status
func ToHTTPStatus(code ExitCode) int {
	switch code {
	case ExitCodeSuccess:
		return 200
	case ExitCodeInvalidArgument, ExitCodeCmdUsage, ExitCodeDataError, ExitCodeValidation:
		return 400
	case ExitCodeAuthRequired, ExitCodeAuthFailed:
		return 401
	case ExitCodeForbidden, ExitCodeNoPermission:
		return 403
	case ExitCodeNotFound, ExitCodeNoInput:
		return 404
	case ExitCodeConflict:
		return 409
	case ExitCodeRateLimit, ExitCodeQuotaExceeded:
		return 429
	case ExitCodeUnavailable, ExitCodeTempFail:
		return 503
	case ExitCodeSoftware, ExitCodeOSError, ExitCodeIOError, ExitCodeErrorInternal:
		return 500
	default:
		return 500
	}
}
