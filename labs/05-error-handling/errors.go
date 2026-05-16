package errlab

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrPermissionDenied   = errors.New("permission denied")
)

type ValidationError struct {
	Field string
	Rule  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Rule)
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter)
}

func WrapWithPercentW(err error) error {
	return fmt.Errorf("load user: %w", err)
}

func WrapWithPercentV(err error) error {
	return fmt.Errorf("load user: %v", err)
}

func RepositoryFindUser(found bool) error {
	if !found {
		return fmt.Errorf("query user: %w", sql.ErrNoRows)
	}
	return nil
}

func ServiceGetUser(found bool) error {
	if err := RepositoryFindUser(found); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("get user: %w", ErrUserNotFound)
		}
		return fmt.Errorf("get user: %w", err)
	}
	return nil
}

type ErrorBody struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

func HTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrEmailAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrPermissionDenied):
		return http.StatusForbidden
	case errors.Is(err, contextDeadlineExceeded()):
		return http.StatusGatewayTimeout
	default:
		var ve *ValidationError
		if errors.As(err, &ve) {
			return http.StatusBadRequest
		}
		var re *RateLimitError
		if errors.As(err, &re) {
			return http.StatusTooManyRequests
		}
		return http.StatusInternalServerError
	}
}

func contextDeadlineExceeded() error { return contextDeadlineExceededSentinel }

var contextDeadlineExceededSentinel = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string { return "deadline exceeded" }

func APIErrorFor(err error, requestID string) ErrorBody {
	body := ErrorBody{Error: APIError{RequestID: requestID}}

	var ve *ValidationError
	var re *RateLimitError
	switch {
	case errors.Is(err, ErrUserNotFound):
		body.Error.Code = "USER_NOT_FOUND"
		body.Error.Message = "user not found"
	case errors.Is(err, ErrEmailAlreadyExists):
		body.Error.Code = "EMAIL_ALREADY_EXISTS"
		body.Error.Message = "email already exists"
	case errors.Is(err, ErrPermissionDenied):
		body.Error.Code = "PERMISSION_DENIED"
		body.Error.Message = "permission denied"
	case errors.As(err, &ve):
		body.Error.Code = "VALIDATION_FAILED"
		body.Error.Message = "validation failed"
		body.Error.Details = map[string]string{ve.Field: ve.Rule}
	case errors.As(err, &re):
		body.Error.Code = "RATE_LIMITED"
		body.Error.Message = "rate limited"
		body.Error.Details = map[string]string{"retry_after": re.RetryAfter.String()}
	default:
		body.Error.Code = "INTERNAL"
		body.Error.Message = "internal server error"
	}
	return body
}

func MarshalAPIError(err error, requestID string) string {
	b, _ := json.Marshal(APIErrorFor(err, requestID))
	return string(b)
}

func ValidateUser(email string, age int) error {
	var errs []error
	if email == "" {
		errs = append(errs, &ValidationError{Field: "email", Rule: "required"})
	}
	if age < 0 {
		errs = append(errs, &ValidationError{Field: "age", Rule: "must be non-negative"})
	}
	return errors.Join(errs...)
}

type PanicResult struct {
	Status  int
	Body    string
	Stack   string
	Recovered bool
}

func Recover(handler func()) (result PanicResult) {
	defer func() {
		if r := recover(); r != nil {
			result.Status = http.StatusInternalServerError
			result.Body = "internal server error"
			result.Stack = string(debug.Stack())
			result.Recovered = true
		}
	}()
	handler()
	return PanicResult{Status: http.StatusOK, Body: "ok"}
}
