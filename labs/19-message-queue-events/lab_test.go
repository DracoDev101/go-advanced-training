package lab

import (
    "context"
    "errors"
    "testing"
    "time"
)

func TestProcessRecordsEvents(t *testing.T) {
    var r Recorder
    if err := Process(context.Background(), &r, SampleEvents()); err != nil { t.Fatalf("Process error: %v", err) }
    if r.Count() != int64(len(SampleEvents())) { t.Fatalf("count got %d", r.Count()) }
    if len(r.Events()) != len(SampleEvents()) { t.Fatalf("events not recorded") }
}
func TestProcessRejectsInvalidEvent(t *testing.T) {
    var r Recorder
    err := Process(context.Background(), &r, []Event{{Kind:"bad"}})
    if !errors.Is(err, ErrInvalidEvent) { t.Fatalf("got %v want ErrInvalidEvent", err) }
}
func TestProcessRespectsCanceledContext(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background()); cancel()
    var r Recorder
    err := Process(ctx, &r, SampleEvents())
    if !errors.Is(err, context.Canceled) { t.Fatalf("got %v want context.Canceled", err) }
}
func TestProcessWithTimeout(t *testing.T) {
    var r Recorder
    if err := ProcessWithTimeout(context.Background(), &r, SampleEvents(), time.Second); err != nil { t.Fatalf("ProcessWithTimeout error: %v", err) }
}
func TestTopic(t *testing.T) { if Topic()=="" { t.Fatal("empty topic") } }
