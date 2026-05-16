package httplab

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubmitJobAccepted(t *testing.T) {
	h := NewHandler(&Service{})
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"kind":"email"}`))
	req.Header.Set("X-Request-Id", "req-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
	if rr.Header().Get("X-Request-Id") != "req-1" { t.Fatalf("missing request id") }
}

func TestInvalidJobMapsTo400(t *testing.T) {
	h := NewHandler(&Service{})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"kind":""}`)))
	if rr.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
}

func TestBodyLimit(t *testing.T) {
	h := NewHandler(&Service{})
	body := bytes.NewBuffer(make([]byte, 1<<20+1))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/jobs", body))
	if rr.Code != http.StatusBadRequest { t.Fatalf("status=%d", rr.Code) }
}

func TestServerTimeoutsConfigured(t *testing.T) {
	s := NewServer(":0", NewHandler(&Service{}))
	if s.ReadHeaderTimeout == 0 || s.ReadTimeout == 0 || s.WriteTimeout == 0 || s.IdleTimeout == 0 { t.Fatalf("timeouts must be configured") }
}

func TestGracefulShutdownReturnsContextErrorWhenServerNotRunning(t *testing.T) {
	s := NewServer(":0", NewHandler(&Service{}))
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_ = GracefulShutdown(ctx, s)
}
