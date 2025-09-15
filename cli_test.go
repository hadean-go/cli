package cli

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestExitCode_String(t *testing.T) {
	tests := []struct {
		code     ExitCode
		expected string
	}{
		{ExitCodeSuccess, "Success"},
		{ExitCodeErrorInternal, "Internal error"},
		{ExitCodeInvalidArgument, "Invalid argument"},
		{ExitCodeNotFound, "Not found"},
		{ExitCodeConfig, "Configuration error"},
		{ExitCodeAuthFailed, "Authentication failed"},
		{ExitCodeValidation, "Validation error"},
		{ExitCodeInterrupted, "Interrupted by user"},
		{ExitCode(999), "Unknown exit code: 999"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ExitCode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExitCode_Category(t *testing.T) {
	tests := []struct {
		code     ExitCode
		expected Category
	}{
		{ExitCodeSuccess, CategorySuccess},
		{ExitCodeError, CategoryGeneral},
		{ExitCodeUsageError, CategoryGeneral},
		{ExitCodeDataError, CategoryUserError},
		{ExitCodeConfig, CategoryUserError},
		{ExitCodeAuthFailed, CategoryCLIExtended},
		{ExitCodeValidation, CategoryCLIExtended},
		{ExitCodeInterrupted, CategorySystemSignal},
		{ExitCodeTerminated, CategorySystemSignal},
		{ExitCode(999), CategorySystemSignal},
	}

	for _, tt := range tests {
		t.Run(string(tt.expected), func(t *testing.T) {
			if got := tt.code.Category(); got != tt.expected {
				t.Errorf("ExitCode.Category() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExitCode_IsRetriable(t *testing.T) {
	retriableCodes := []ExitCode{
		ExitCodeTempFail,
		ExitCodeUnavailable,
		ExitCodeIOError,
		ExitCodeRateLimit,
	}

	nonRetriableCodes := []ExitCode{
		ExitCodeSuccess,
		ExitCodeError,
		ExitCodeUsageError,
		ExitCodeDataError,
		ExitCodeConfig,
		ExitCodeAuthFailed,
		ExitCodeValidation,
		ExitCodeNotFound,
		ExitCodeInterrupted,
	}

	for _, code := range retriableCodes {
		t.Run("retriable_"+code.String(), func(t *testing.T) {
			if !code.IsRetriable() {
				t.Errorf("ExitCode %d should be retriable", code)
			}
		})
	}

	for _, code := range nonRetriableCodes {
		t.Run("non_retriable_"+code.String(), func(t *testing.T) {
			if code.IsRetriable() {
				t.Errorf("ExitCode %d should not be retriable", code)
			}
		})
	}
}

func TestExitCode_IsUserError(t *testing.T) {
	userErrorCodes := []ExitCode{
		ExitCodeUsageError,
		ExitCodeCmdUsage,
		ExitCodeDataError,
		ExitCodeNoInput,
		ExitCodeConfig,
		ExitCodeValidation,
	}

	systemErrorCodes := []ExitCode{
		ExitCodeSuccess,
		ExitCodeError,
		ExitCodeSoftware,
		ExitCodeOSError,
		ExitCodeIOError,
		ExitCodeAuthFailed,
		ExitCodeInterrupted,
		ExitCodeUnavailable,
		ExitCodeTempFail,
	}

	for _, code := range userErrorCodes {
		t.Run("user_error_"+code.String(), func(t *testing.T) {
			if !code.IsUserError() {
				t.Errorf("ExitCode %d should be a user error", code)
			}
		})
	}

	for _, code := range systemErrorCodes {
		t.Run("system_error_"+code.String(), func(t *testing.T) {
			if code.IsUserError() {
				t.Errorf("ExitCode %d should not be a user error", code)
			}
		})
	}
}

func TestExitError(t *testing.T) {
	t.Run("with_message", func(t *testing.T) {
		err := NewExitError(ExitCodeConfig, "custom message", nil)
		if err.Error() != "custom message" {
			t.Errorf("ExitError.Error() = %v, want %v", err.Error(), "custom message")
		}
		if err.Code != ExitCodeConfig {
			t.Errorf("ExitError.Code = %v, want %v", err.Code, ExitCodeConfig)
		}
	})

	t.Run("with_cause", func(t *testing.T) {
		cause := errors.New("root cause")
		err := NewExitError(ExitCodeIOError, "", cause)
		if err.Error() != "root cause" {
			t.Errorf("ExitError.Error() = %v, want %v", err.Error(), "root cause")
		}
		if err.Unwrap() != cause {
			t.Errorf("ExitError.Unwrap() = %v, want %v", err.Unwrap(), cause)
		}
	})

	t.Run("default_string", func(t *testing.T) {
		err := NewExitError(ExitCodeNotFound, "", nil)
		if err.Error() != "Not found" {
			t.Errorf("ExitError.Error() = %v, want %v", err.Error(), "Not found")
		}
	})
}

func TestResolveExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ExitCode
	}{
		{"nil_error", nil, ExitCodeSuccess},
		{"internal_error", ErrInternal, ExitCodeErrorInternal},
		{"invalid_error", ErrInvalid, ExitCodeInvalidArgument},
		{"usage_error", ErrUsage, ExitCodeUsageError},
		{"data_format_error", ErrDataFormat, ExitCodeDataError},
		{"not_found_error", ErrNotFound, ExitCodeNotFound},
		{"permission_error", ErrNoPermission, ExitCodeNoPermission},
		{"config_error", ErrConfig, ExitCodeConfig},
		{"auth_error", ErrAuth, ExitCodeAuthFailed},
		{"forbidden_error", ErrForbidden, ExitCodeForbidden},
		{"validation_error", ErrValidation, ExitCodeValidation},
		{"io_error", ErrIO, ExitCodeIOError},
		{"unavailable_error", ErrUnavailable, ExitCodeUnavailable},
		{"temp_fail_error", ErrTempFail, ExitCodeTempFail},
		{"unknown_error", errors.New("unknown"), ExitCodeErrorInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveExitCode(tt.err); got != tt.expected {
				t.Errorf("ResolveExitCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResolveExitCode_ExitError(t *testing.T) {
	exitErr := NewExitError(ExitCodeValidation, "validation failed", nil)
	wrapped := errors.New("wrapped: " + exitErr.Error())

	// Should return code from ExitError
	if got := ResolveExitCode(exitErr); got != ExitCodeValidation {
		t.Errorf("ResolveExitCode(ExitError) = %v, want %v", got, ExitCodeValidation)
	}

	// For wrapped error should return general code
	if got := ResolveExitCode(wrapped); got != ExitCodeErrorInternal {
		t.Errorf("ResolveExitCode(wrapped) = %v, want %v", got, ExitCodeErrorInternal)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("UsageError", func(t *testing.T) {
		err := UsageError("invalid command")
		var exitErr *ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal("UsageError should return ExitError")
		}
		if exitErr.Code != ExitCodeUsageError {
			t.Errorf("UsageError code = %v, want %v", exitErr.Code, ExitCodeUsageError)
		}
		if err.Error() != "invalid command" {
			t.Errorf("UsageError message = %v, want %v", err.Error(), "invalid command")
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := ValidationError("invalid data")
		var exitErr *ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal("ValidationError should return ExitError")
		}
		if exitErr.Code != ExitCodeValidation {
			t.Errorf("ValidationError code = %v, want %v", exitErr.Code, ExitCodeValidation)
		}
	})

	t.Run("NotFoundError", func(t *testing.T) {
		err := NotFoundError("config file")
		if err.Error() != "config file not found" {
			t.Errorf("NotFoundError message = %v, want %v", err.Error(), "config file not found")
		}
	})

	t.Run("PermissionError", func(t *testing.T) {
		err := PermissionError("write file")
		if err.Error() != "permission denied: write file" {
			t.Errorf("PermissionError message = %v, want %v", err.Error(), "permission denied: write file")
		}
	})

	t.Run("AuthError", func(t *testing.T) {
		err := AuthError("invalid token")
		var exitErr *ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal("AuthError should return ExitError")
		}
		if exitErr.Code != ExitCodeAuthFailed {
			t.Errorf("AuthError code = %v, want %v", exitErr.Code, ExitCodeAuthFailed)
		}
	})

	t.Run("TempFailError", func(t *testing.T) {
		err := TempFailError("network timeout")
		var exitErr *ExitError
		if !errors.As(err, &exitErr) {
			t.Fatal("TempFailError should return ExitError")
		}
		if exitErr.Code != ExitCodeTempFail {
			t.Errorf("TempFailError code = %v, want %v", exitErr.Code, ExitCodeTempFail)
		}
		if !exitErr.Code.IsRetriable() {
			t.Error("TempFailError should be retriable")
		}
	})
}

type dummyNetError struct{ timeout bool }

func (d dummyNetError) Error() string   { return "dummy" }
func (d dummyNetError) Timeout() bool   { return d.timeout }
func (d dummyNetError) Temporary() bool { return !d.timeout }

func TestResolveExitCode_Mappings(t *testing.T) {
	t.Run("context_canceled", func(t *testing.T) {
		if got := ResolveExitCode(context.Canceled); got != ExitCodeInterrupted {
			t.Fatalf("want %v got %v", ExitCodeInterrupted, got)
		}
	})
	t.Run("context_deadline", func(t *testing.T) {
		if got := ResolveExitCode(context.DeadlineExceeded); got != ExitCodeTempFail {
			t.Fatalf("want %v got %v", ExitCodeTempFail, got)
		}
	})
	t.Run("net_timeout", func(t *testing.T) {
		if got := ResolveExitCode(dummyNetError{timeout: true}); got != ExitCodeTempFail {
			t.Fatalf("want %v got %v", ExitCodeTempFail, got)
		}
	})
	t.Run("net_unavailable", func(t *testing.T) {
		if got := ResolveExitCode(dummyNetError{timeout: false}); got != ExitCodeTempFail {
			t.Fatalf("want %v got %v", ExitCodeTempFail, got)
		}
	})
}

func TestHTTPMappings(t *testing.T) {
	cases := []struct {
		status int
		code   ExitCode
	}{
		{200, ExitCodeSuccess},
		{401, ExitCodeAuthFailed},
		{403, ExitCodeForbidden},
		{404, ExitCodeNotFound},
		{422, ExitCodeValidation},
		{429, ExitCodeRateLimit},
		{503, ExitCodeUnavailable},
	}
	for _, c := range cases {
		if got := FromHTTPStatus(c.status); got != c.code {
			t.Fatalf("FromHTTPStatus(%d) want %v got %v", c.status, c.code, got)
		}
	}

	codes := []ExitCode{
		ExitCodeSuccess,
		ExitCodeAuthFailed,
		ExitCodeForbidden,
		ExitCodeNotFound,
		ExitCodeValidation,
		ExitCodeRateLimit,
		ExitCodeQuotaExceeded,
		ExitCodeUnavailable,
		ExitCodeTempFail,
		ExitCodeSoftware,
	}
	for _, code := range codes {
		status := ToHTTPStatus(code)
		if FromHTTPStatus(status) == ExitCodeErrorInternal {
			t.Fatalf("ToHTTPStatus/FromHTTPStatus mapping should not produce internal error for %v", code)
		}
	}
}

func TestQuotaExceededHTTPMapping(t *testing.T) {
	if got := ToHTTPStatus(ExitCodeQuotaExceeded); got != 429 {
		t.Fatalf("ToHTTPStatus(QuotaExceeded) want 429 got %d", got)
	}
}

func TestExitError_JSONMarshaling(t *testing.T) {
	err := NewExitError(ExitCodeValidation, "invalid input", errors.New("root cause"))
	data, mErr := err.MarshalJSON()
	if mErr != nil {
		t.Fatalf("MarshalJSON error: %v", mErr)
	}
	var obj map[string]any
	if uErr := json.Unmarshal(data, &obj); uErr != nil {
		t.Fatalf("json.Unmarshal error: %v", uErr)
	}
	if int(obj["code"].(float64)) != int(ExitCodeValidation) {
		t.Fatalf("json code mismatch: %v", obj["code"])
	}
	if obj["name"].(string) != ExitCodeValidation.String() {
		t.Fatalf("json name mismatch: %v", obj["name"])
	}
	if obj["category"].(string) != string(ExitCodeValidation.Category()) {
		t.Fatalf("json category mismatch: %v", obj["category"])
	}
	if obj["message"].(string) != "invalid input" {
		t.Fatalf("json message mismatch: %v", obj["message"])
	}
}

func TestExitCode_UnmarshalText(t *testing.T) {
	var c ExitCode
	// empty -> success
	if err := c.UnmarshalText([]byte("   \t   ")); err != nil {
		t.Fatalf("UnmarshalText empty returned error: %v", err)
	}
	if c != ExitCodeSuccess {
		t.Fatalf("UnmarshalText empty want %v got %v", ExitCodeSuccess, c)
	}
	// valid number
	if err := c.UnmarshalText([]byte(" 85 \n")); err != nil {
		t.Fatalf("UnmarshalText valid returned error: %v", err)
	}
	if c != ExitCodeValidation {
		t.Fatalf("UnmarshalText valid want %v got %v", ExitCodeValidation, c)
	}
	// invalid
	if err := c.UnmarshalText([]byte("abc")); err == nil {
		t.Fatal("UnmarshalText invalid should return error")
	}
}

func TestNewfAndWithCode(t *testing.T) {
	err := Newf(ExitCodeValidation, "invalid field %s", "email")
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatal("Newf should return ExitError")
	}
	if ee.Code != ExitCodeValidation {
		t.Fatalf("want %v got %v", ExitCodeValidation, ee.Code)
	}

	orig := errors.New("root cause")
	wrapped := WithCode(orig, ExitCodeConfig)
	if !errors.As(wrapped, &ee) {
		t.Fatal("WithCode should return ExitError")
	}
	if ee.Code != ExitCodeConfig || !errors.Is(wrapped, orig) {
		t.Fatalf("WithCode mismatch: code=%v cause matches=%v", ee.Code, errors.Is(wrapped, orig))
	}
}

// Example integration test
func TestIntegrationExample(t *testing.T) {
	// Simulation of various CLI application scenarios
	scenarios := []struct {
		name         string
		operation    func() error
		expectedCode ExitCode
		shouldRetry  bool
	}{
		{
			name: "success",
			operation: func() error {
				return nil
			},
			expectedCode: ExitCodeSuccess,
			shouldRetry:  false,
		},
		{
			name: "file_not_found",
			operation: func() error {
				return NotFoundError("input.txt")
			},
			expectedCode: ExitCodeNotFound,
			shouldRetry:  false,
		},
		{
			name: "invalid_arguments",
			operation: func() error {
				return UsageError("missing required argument")
			},
			expectedCode: ExitCodeUsageError,
			shouldRetry:  false,
		},
		{
			name: "network_failure",
			operation: func() error {
				return TempFailError("connection timeout")
			},
			expectedCode: ExitCodeTempFail,
			shouldRetry:  true,
		},
		{
			name: "permission_denied",
			operation: func() error {
				return PermissionError("access config directory")
			},
			expectedCode: ExitCodeNoPermission,
			shouldRetry:  false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.operation()
			exitCode := ResolveExitCode(err)

			if exitCode != scenario.expectedCode {
				t.Errorf("Expected exit code %d (%s), got %d (%s)",
					scenario.expectedCode, scenario.expectedCode.String(),
					exitCode, exitCode.String())
			}

			if exitCode.IsRetriable() != scenario.shouldRetry {
				t.Errorf("Expected retriable=%v, got %v",
					scenario.shouldRetry, exitCode.IsRetriable())
			}

			t.Logf("Scenario: %s", scenario.name)
			t.Logf("  Exit code: %d (%s)", exitCode, exitCode.String())
			t.Logf("  Category: %s", exitCode.Category())
			t.Logf("  Retriable: %v", exitCode.IsRetriable())
			t.Logf("  User error: %v", exitCode.IsUserError())
		})
	}
}
