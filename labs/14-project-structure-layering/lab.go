package lab

import (
    "context"
    "errors"
    "sync"
    "sync/atomic"
    "time"
)

type Event struct {
    ID string
    Kind string
    Attempts int
}

type Recorder struct {
    mu sync.Mutex
    events []Event
    processed atomic.Int64
}

func (r *Recorder) Record(e Event) {
    r.mu.Lock()
    r.events = append(r.events, e)
    r.mu.Unlock()
    r.processed.Add(1)
}

func (r *Recorder) Events() []Event {
    r.mu.Lock()
    defer r.mu.Unlock()
    out := make([]Event, len(r.events))
    copy(out, r.events)
    return out
}

func (r *Recorder) Count() int64 { return r.processed.Load() }

var ErrInvalidEvent = errors.New("invalid event")

func Process(ctx context.Context, r *Recorder, events []Event) error {
    for _, e := range events {
        if e.ID == "" { return ErrInvalidEvent }
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            r.Record(e)
        }
    }
    return nil
}

func ProcessWithTimeout(parent context.Context, r *Recorder, events []Event, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(parent, timeout)
    defer cancel()
    return Process(ctx, r, events)
}

func SampleEvents() []Event {
    return []Event{
        {ID:"job-1", Kind:"project-structure-layering", Attempts:1},
        {ID:"job-2", Kind:"project-structure-layering", Attempts:2},
    }
}

func Topic() string { return "Lesson 14: 项目结构与分层设计" }
