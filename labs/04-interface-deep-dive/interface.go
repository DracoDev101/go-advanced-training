package iface

import (
	"context"
	"fmt"
	"time"
)

type Notifier interface {
	Notify(ctx context.Context, msg string) error
}

type EmailNotifier struct {
	Sent []string
}

func (n *EmailNotifier) Notify(ctx context.Context, msg string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		n.Sent = append(n.Sent, msg)
		return nil
	}
}

var _ Notifier = (*EmailNotifier)(nil)

type Counter struct{ n int }

func (c *Counter) Inc() { c.n++ }
func (c *Counter) Value() int { return c.n }

type Incrementer interface{ Inc() }

var _ Incrementer = (*Counter)(nil)

type ValidationError struct{ Field string }

func (e *ValidationError) Error() string {
	if e == nil {
		return "<nil ValidationError>"
	}
	return fmt.Sprintf("invalid field %s", e.Field)
}

func ReturnTypedNilError() error {
	var err *ValidationError = nil
	return err
}

func ReturnNilErrorCorrectly() error {
	var err *ValidationError = nil
	if err != nil {
		return err
	}
	return nil
}

type User struct {
	ID   string
	Name string
}

type UserFinder interface {
	FindByID(ctx context.Context, id string) (User, error)
}

type MemoryUsers struct {
	Users map[string]User
}

func (m MemoryUsers) FindByID(ctx context.Context, id string) (User, error) {
	select {
	case <-ctx.Done():
		return User{}, ctx.Err()
	default:
	}
	u, ok := m.Users[id]
	if !ok {
		return User{}, fmt.Errorf("user %s not found", id)
	}
	return u, nil
}

type Greeter struct {
	Users UserFinder
}

func (g Greeter) Greet(ctx context.Context, id string) (string, error) {
	u, err := g.Users.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	return "hello " + u.Name, nil
}

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

type FakeClock struct{ T time.Time }

func (c FakeClock) Now() time.Time { return c.T }

func Expired(c Clock, deadline time.Time) bool {
	return !c.Now().Before(deadline)
}

type Small struct{ A, B int }

func SumConcrete(s Small) int { return s.A + s.B }

type Summer interface{ Sum() int }

func (s Small) Sum() int { return s.A + s.B }

func SumInterface(s Summer) int { return s.Sum() }

func BoxAny(s Small) any { return s }
