package syncatomic

import (
	"sync"
	"testing"
)

func TestMutexCounterConcurrent(t *testing.T) {
	var c MutexCounter
	runConcurrent(100, func(){ c.Inc() })
	if got := c.Value(); got != 100 { t.Fatalf("got %d want 100", got) }
}

func TestAtomicCounterConcurrent(t *testing.T) {
	var c AtomicCounter
	runConcurrent(100, func(){ c.Inc() })
	if got := c.Value(); got != 100 { t.Fatalf("got %d want 100", got) }
}

func runConcurrent(n int, fn func()) {
	var wg sync.WaitGroup
	wg.Add(n)
	for i:=0; i<n; i++ { go func(){ defer wg.Done(); fn() }() }
	wg.Wait()
}

func TestCacheSetGetDelete(t *testing.T) {
	c := NewCache()
	c.Set("a", "1")
	if v, ok := c.Get("a"); !ok || v != "1" { t.Fatalf("got %q ok=%v", v, ok) }
	c.Delete("a")
	if _, ok := c.Get("a"); ok { t.Fatal("key should be deleted") }
}

func TestLazyConfigOnce(t *testing.T) {
	var c LazyConfig
	runConcurrent(50, func(){ if c.Value() != "initialized" { t.Fatal("bad value") } })
}

func TestParallelSumWaitGroup(t *testing.T) {
	if got := ParallelSum([]int{1,2,3,4}); got != 10 { t.Fatalf("got %d want 10", got) }
}

func TestConfigStorePublishesImmutableSnapshot(t *testing.T) {
	cfg := Config{Version:1, Limits: map[string]int{"qps":100}}
	store := NewConfigStore(cfg)
	cfg.Limits["qps"] = 999
	loaded := store.Load()
	if loaded.Limits["qps"] != 100 { t.Fatalf("snapshot mutated: %v", loaded) }
	loaded.Limits["qps"] = 777
	loaded2 := store.Load()
	if loaded2.Limits["qps"] != 100 { t.Fatalf("load returned mutable internal map: %v", loaded2) }
}

func TestChannelHappensBefore(t *testing.T) {
	if got := ChannelHappensBefore(); got != 42 { t.Fatalf("got %d want 42", got) }
}
