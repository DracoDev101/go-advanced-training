package httplab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var ErrInvalidJob = errors.New("invalid job")

type JobRequest struct { Kind string `json:"kind"` }
type JobResponse struct { ID string `json:"id"`; Kind string `json:"kind"` }

type Service struct{ next atomic.Int64 }

func (s *Service) Submit(ctx context.Context, req JobRequest) (JobResponse, error) {
	if strings.TrimSpace(req.Kind) == "" { return JobResponse{}, ErrInvalidJob }
	select {
	case <-ctx.Done(): return JobResponse{}, ctx.Err()
	case <-time.After(2 * time.Millisecond):
	}
	id := s.next.Add(1)
	return JobResponse{ID: "job-" + time.Now().Format("150405") + "-" + string(rune('0'+id%10)), Kind: req.Kind}, nil
}

type statusWriter struct { http.ResponseWriter; status int }
func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" { id = "generated-request-id" }
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

type requestIDKey struct{}

func BodyLimit(max int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, max)
		next.ServeHTTP(w, r)
	})
}

func Timeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func(){ if rec := recover(); rec != nil { writeError(w, r, http.StatusInternalServerError, "internal") } }()
		next.ServeHTTP(w, r)
	})
}

func NewHandler(s *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /jobs", func(w http.ResponseWriter, r *http.Request) {
		var req JobRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeError(w, r, http.StatusBadRequest, "invalid_json"); return }
		resp, err := s.Submit(r.Context(), req)
		if err != nil { mapError(w, r, err); return }
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(resp)
	})
	var h http.Handler = mux
	h = Timeout(50*time.Millisecond, h)
	h = BodyLimit(1<<20, h)
	h = Recover(h)
	h = RequestID(h)
	return h
}

func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr: addr, Handler: h,
		ReadHeaderTimeout: 2*time.Second,
		ReadTimeout: 10*time.Second,
		WriteTimeout: 30*time.Second,
		IdleTimeout: 60*time.Second,
	}
}

func GracefulShutdown(ctx context.Context, srv *http.Server) error {
	err := srv.Shutdown(ctx)
	if err != nil { _ = srv.Close() }
	return err
}

func mapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidJob): writeError(w, r, http.StatusBadRequest, "invalid_argument")
	case errors.Is(err, context.DeadlineExceeded): writeError(w, r, http.StatusGatewayTimeout, "timeout")
	case errors.Is(err, context.Canceled): writeError(w, r, 499, "canceled")
	default: writeError(w, r, http.StatusInternalServerError, "internal")
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "request_id": r.Context().Value(requestIDKey{})}})
}
