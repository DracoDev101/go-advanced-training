package iface

import (
	"context"
	"testing"
	"time"
)

func TestImplicitInterfaceImplementation(t *testing.T) {
	n := &EmailNotifier{}
	var notifier Notifier = n

	if err := notifier.Notify(context.Background(), "hello"); err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if len(n.Sent) != 1 || n.Sent[0] != "hello" {
		t.Fatalf("sent messages: got %v", n.Sent)
	}
}

func TestPointerReceiverSatisfiesInterface(t *testing.T) {
	var inc Incrementer = &Counter{}
	inc.Inc()

	c, ok := inc.(*Counter)
	if !ok {
		t.Fatal("Incrementer should hold *Counter")
	}
	if c.Value() != 1 {
		t.Fatalf("counter value: got %d, want 1", c.Value())
	}
}

func TestTypedNilErrorIsNotNilInterface(t *testing.T) {
	err := ReturnTypedNilError()

	if err == nil {
		t.Fatal("typed nil error stored in error interface should not equal nil")
	}
}

func TestReturnNilErrorCorrectly(t *testing.T) {
	err := ReturnNilErrorCorrectly()

	if err != nil {
		t.Fatalf("expected real nil error, got %T %v", err, err)
	}
}

func TestSmallInterfaceSupportsFocusedFake(t *testing.T) {
	g := Greeter{Users: MemoryUsers{Users: map[string]User{
		"u1": {ID: "u1", Name: "Draco"},
	}}}

	got, err := g.Greet(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Greet returned error: %v", err)
	}
	if got != "hello Draco" {
		t.Fatalf("Greet: got %q", got)
	}
}

func TestClockInterfaceMakesTimeTestable(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	clock := FakeClock{T: now}

	if !Expired(clock, now) {
		t.Fatal("deadline equal to now should be expired")
	}
	if Expired(clock, now.Add(time.Second)) {
		t.Fatal("future deadline should not be expired")
	}
}
