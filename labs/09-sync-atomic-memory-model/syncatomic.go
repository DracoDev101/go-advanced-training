package syncatomic

import (
	"sync"
	"sync/atomic"
)

type UnsafeCounter struct{ n int64 }
func (c *UnsafeCounter) Inc() { c.n++ }
func (c *UnsafeCounter) Value() int64 { return c.n }

type MutexCounter struct {
	mu sync.Mutex
	n  int64
}
func (c *MutexCounter) Inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *MutexCounter) Value() int64 { c.mu.Lock(); defer c.mu.Unlock(); return c.n }

type AtomicCounter struct{ n atomic.Int64 }
func (c *AtomicCounter) Inc() { c.n.Add(1) }
func (c *AtomicCounter) Value() int64 { return c.n.Load() }

type ChannelCounter struct{ ops chan counterOp }
type counterOp struct{ kind string; resp chan int64 }
func NewChannelCounter() *ChannelCounter {
	c := &ChannelCounter{ops: make(chan counterOp)}
	go func(){
		var n int64
		for op := range c.ops {
			switch op.kind {
			case "inc": n++
			case "value": op.resp <- n
			}
		}
	}()
	return c
}
func (c *ChannelCounter) Inc() { c.ops <- counterOp{kind:"inc"} }
func (c *ChannelCounter) Value() int64 { resp:=make(chan int64); c.ops<-counterOp{kind:"value", resp:resp}; return <-resp }
func (c *ChannelCounter) Close() { close(c.ops) }

type Cache struct {
	mu sync.RWMutex
	m  map[string]string
}
func NewCache() *Cache { return &Cache{m: map[string]string{}} }
func (c *Cache) Set(k, v string) { c.mu.Lock(); c.m[k]=v; c.mu.Unlock() }
func (c *Cache) Get(k string) (string, bool) { c.mu.RLock(); defer c.mu.RUnlock(); v, ok := c.m[k]; return v, ok }
func (c *Cache) Delete(k string) { c.mu.Lock(); delete(c.m, k); c.mu.Unlock() }

type LazyConfig struct {
	once sync.Once
	value string
}
func (c *LazyConfig) Value() string {
	c.once.Do(func(){ c.value = "initialized" })
	return c.value
}

func ParallelSum(nums []int) int {
	var wg sync.WaitGroup
	results := make(chan int, len(nums))
	for _, n := range nums {
		n := n
		wg.Add(1)
		go func(){ defer wg.Done(); results <- n }()
	}
	wg.Wait()
	close(results)
	sum := 0
	for n := range results { sum += n }
	return sum
}

type Config struct {
	Version int
	Limits map[string]int
}

type ConfigStore struct{ v atomic.Value }
func NewConfigStore(cfg Config) *ConfigStore { s:=&ConfigStore{}; s.Store(cfg); return s }
func cloneConfig(cfg Config) Config {
	limits := make(map[string]int, len(cfg.Limits))
	for k, v := range cfg.Limits { limits[k]=v }
	return Config{Version: cfg.Version, Limits: limits}
}
func (s *ConfigStore) Store(cfg Config) { s.v.Store(cloneConfig(cfg)) }
func (s *ConfigStore) Load() Config { return cloneConfig(s.v.Load().(Config)) }

func ChannelHappensBefore() int {
	ch := make(chan struct{})
	value := 0
	go func(){ value = 42; close(ch) }()
	<-ch
	return value
}
