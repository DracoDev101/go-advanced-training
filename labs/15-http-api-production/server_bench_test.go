package httplab

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkSubmitJobHandler(b *testing.B) {
	h := NewHandler(&Service{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"kind":"email"}`))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusAccepted { b.Fatalf("status=%d", rr.Code) }
	}
}
