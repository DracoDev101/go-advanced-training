package pglab

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNoJobClaimed = errors.New("no job claimed")

type Job struct {
	ID       string
	Status   string
	LockedBy string
	LockedAt time.Time
	Version  int
}

// Repository is an in-memory model of the DB invariants we want from PostgreSQL.
// The SQL equivalent is documented in README.md.
type Repository struct {
	mu   sync.Mutex
	jobs map[string]Job
}

func NewRepository(jobs ...Job) *Repository {
	r := &Repository{jobs: make(map[string]Job)}
	for _, j := range jobs { r.jobs[j.ID] = j }
	return r
}

func (r *Repository) Get(id string) (Job, bool) {
	r.mu.Lock(); defer r.mu.Unlock()
	j, ok := r.jobs[id]
	return j, ok
}

// UnsafeClaim models the broken SELECT then UPDATE pattern. It is intentionally split
// by a hook so tests can demonstrate duplicate claims.
func (r *Repository) UnsafeClaim(ctx context.Context, id, worker string, beforeUpdate func()) (Job, error) {
	r.mu.Lock()
	j, ok := r.jobs[id]
	if !ok || j.Status != "pending" { r.mu.Unlock(); return Job{}, ErrNoJobClaimed }
	r.mu.Unlock()
	if beforeUpdate != nil { beforeUpdate() }
	select { case <-ctx.Done(): return Job{}, ctx.Err(); default: }
	r.mu.Lock(); defer r.mu.Unlock()
	j.LockedBy = worker; j.LockedAt = time.Now(); j.Status = "running"; j.Version++
	r.jobs[id] = j
	return j, nil
}

// ClaimAtomic models UPDATE ... WHERE status='pending' RETURNING id.
func (r *Repository) ClaimAtomic(ctx context.Context, id, worker string) (Job, error) {
	select { case <-ctx.Done(): return Job{}, ctx.Err(); default: }
	r.mu.Lock(); defer r.mu.Unlock()
	j, ok := r.jobs[id]
	if !ok || j.Status != "pending" { return Job{}, ErrNoJobClaimed }
	j.Status = "running"; j.LockedBy = worker; j.LockedAt = time.Now(); j.Version++
	r.jobs[id] = j
	return j, nil
}

func (r *Repository) RequeueStale(now time.Time, maxAge time.Duration) int {
	r.mu.Lock(); defer r.mu.Unlock()
	count := 0
	for id, j := range r.jobs {
		if j.Status == "running" && now.Sub(j.LockedAt) > maxAge {
			j.Status = "pending"; j.LockedBy = ""; j.Version++
			r.jobs[id] = j; count++
		}
	}
	return count
}

type OutboxEvent struct { ID, JobID, Kind string }

type Tx struct { Job Job; Event OutboxEvent }

func CompleteWithOutbox(j Job) Tx {
	j.Status = "succeeded"; j.Version++
	return Tx{Job: j, Event: OutboxEvent{ID: "evt-"+j.ID, JobID: j.ID, Kind: "job.succeeded"}}
}
