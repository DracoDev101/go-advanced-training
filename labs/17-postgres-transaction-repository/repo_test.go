package pglab

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestClaimAtomicOnlyOneWorkerWins(t *testing.T) {
	r := NewRepository(Job{ID:"j1", Status:"pending"})
	var wg sync.WaitGroup
	wins := make(chan string, 2)
	for _, w := range []string{"w1", "w2"} {
		wg.Add(1)
		go func(worker string){ defer wg.Done(); if _, err := r.ClaimAtomic(context.Background(), "j1", worker); err == nil { wins <- worker } }(w)
	}
	wg.Wait(); close(wins)
	count := 0
	for range wins { count++ }
	if count != 1 { t.Fatalf("expected exactly one winner, got %d", count) }
}

func TestUnsafeClaimCanOverwriteOwner(t *testing.T) {
	r := NewRepository(Job{ID:"j1", Status:"pending"})
	barrier := make(chan struct{})
	resume := make(chan struct{})
	go func(){ _, _ = r.UnsafeClaim(context.Background(), "j1", "w1", func(){ close(barrier); <-resume }) }()
	<-barrier
	if _, err := r.UnsafeClaim(context.Background(), "j1", "w2", nil); err != nil { t.Fatal(err) }
	close(resume)
	time.Sleep(10*time.Millisecond)
	j, _ := r.Get("j1")
	if j.LockedBy != "w1" { t.Fatalf("expected stale first claimant to overwrite w2, got %+v", j) }
}

func TestClaimAtomicReturnsNoJob(t *testing.T) {
	r := NewRepository(Job{ID:"j1", Status:"running"})
	_, err := r.ClaimAtomic(context.Background(), "j1", "w1")
	if !errors.Is(err, ErrNoJobClaimed) { t.Fatalf("err=%v", err) }
}

func TestRequeueStale(t *testing.T) {
	old := time.Now().Add(-time.Hour)
	r := NewRepository(Job{ID:"j1", Status:"running", LockedBy:"w1", LockedAt:old})
	if n := r.RequeueStale(time.Now(), time.Minute); n != 1 { t.Fatalf("requeued=%d", n) }
	j, _ := r.Get("j1")
	if j.Status != "pending" || j.LockedBy != "" { t.Fatalf("job=%+v", j) }
}

func TestCompleteWithOutbox(t *testing.T) {
	tx := CompleteWithOutbox(Job{ID:"j1", Status:"running"})
	if tx.Job.Status != "succeeded" || tx.Event.JobID != "j1" { t.Fatalf("tx=%+v", tx) }
}
