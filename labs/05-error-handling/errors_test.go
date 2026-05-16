package errlab

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPercentWWrapPreservesErrorChain(t *testing.T) {
	err := WrapWithPercentW(sql.ErrNoRows)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("%w wrapping should preserve sql.ErrNoRows")
	}
}

func TestPercentVWrapBreaksErrorChain(t *testing.T) {
	err := WrapWithPercentV(sql.ErrNoRows)
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatal("percent-v wrapping should not preserve sql.ErrNoRows")
	}
}

func TestRepositoryErrorMappedToDomainError(t *testing.T) {
	err := ServiceGetUser(false)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound in chain, got %v", err)
	}
	if HTTPStatus(err) != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", HTTPStatus(err))
	}
}

func TestValidationErrorAsAndHTTPMapping(t *testing.T) {
	err := fmtValidationError()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("expected ValidationError in chain")
	}
	if ve.Field != "email" {
		t.Fatalf("field: got %q", ve.Field)
	}
	if HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", HTTPStatus(err))
	}
}

func fmtValidationError() error {
	return WrapWithPercentW(&ValidationError{Field: "email", Rule: "required"})
}

func TestRateLimitErrorAsAndHTTPMapping(t *testing.T) {
	err := WrapWithPercentW(&RateLimitError{RetryAfter: time.Second})
	var re *RateLimitError
	if !errors.As(err, &re) {
		t.Fatal("expected RateLimitError in chain")
	}
	if re.RetryAfter != time.Second {
		t.Fatalf("retry after: got %s", re.RetryAfter)
	}
	if HTTPStatus(err) != http.StatusTooManyRequests {
		t.Fatalf("status: got %d, want 429", HTTPStatus(err))
	}
}

func TestAPIErrorDoesNotLeakInternalError(t *testing.T) {
	err := errors.New("pq: password=secret relation internal_user_shadow does not exist")
	body := MarshalAPIError(err, "req-1")

	if strings.Contains(body, "password") || strings.Contains(body, "internal_user_shadow") {
		t.Fatalf("response leaked internal error: %s", body)
	}
	if !strings.Contains(body, "INTERNAL") || !strings.Contains(body, "req-1") {
		t.Fatalf("response missing stable code/request id: %s", body)
	}
}

func TestErrorsJoinStillSupportsErrorsAs(t *testing.T) {
	err := ValidateUser("", -1)
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("joined validation errors should support errors.As")
	}
}

func TestPermissionDeniedMapsTo403(t *testing.T) {
	if HTTPStatus(WrapWithPercentW(ErrPermissionDenied)) != http.StatusForbidden {
		t.Fatal("permission denied should map to 403")
	}
}

func TestRecoverCapturesPanic(t *testing.T) {
	result := Recover(func() { panic("boom") })
	if !result.Recovered {
		t.Fatal("panic should be recovered")
	}
	if result.Status != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", result.Status)
	}
	if result.Body != "internal server error" {
		t.Fatalf("body: got %q", result.Body)
	}
	if !strings.Contains(result.Stack, "panic") {
		t.Fatal("stack should contain panic")
	}
}
