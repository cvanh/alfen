package discovery

import (
	"sync"
	"time"
)

// netDict reproduces the enumeration order of a .NET Framework
// Dictionary<TKey,TValue>, which Tmds.MDns iterates when it builds queries
// and applies packets: entries are enumerated by slot index, a removed slot is
// pushed on a LIFO free list and reused by the next Add, and Clear resets all
// slots. Keys are the case-folded dnsName keys.
type netDict[V any] struct {
	slots []dictSlot[V]
	free  []int
	index map[string]int
}

type dictSlot[V any] struct {
	key  string
	val  V
	used bool
}

func newNetDict[V any]() *netDict[V] {
	return &netDict[V]{index: map[string]int{}}
}

func (d *netDict[V]) get(key string) (V, bool) {
	if i, ok := d.index[key]; ok {
		return d.slots[i].val, true
	}
	var zero V
	return zero, false
}

func (d *netDict[V]) has(key string) bool {
	_, ok := d.index[key]
	return ok
}

// add ports Dictionary.Add for a key known to be absent (callers check
// first); for an existing key it overwrites in place.
func (d *netDict[V]) add(key string, v V) {
	if i, ok := d.index[key]; ok {
		d.slots[i].val = v
		return
	}
	var i int
	if n := len(d.free); n > 0 {
		i = d.free[n-1]
		d.free = d.free[:n-1]
		d.slots[i] = dictSlot[V]{key: key, val: v, used: true}
	} else {
		i = len(d.slots)
		d.slots = append(d.slots, dictSlot[V]{key: key, val: v, used: true})
	}
	d.index[key] = i
}

func (d *netDict[V]) remove(key string) bool {
	i, ok := d.index[key]
	if !ok {
		return false
	}
	delete(d.index, key)
	d.slots[i] = dictSlot[V]{}
	d.free = append(d.free, i)
	return true
}

func (d *netDict[V]) clear() {
	d.slots = d.slots[:0]
	d.free = d.free[:0]
	clear(d.index)
}

func (d *netDict[V]) len() int { return len(d.index) }

// each enumerates in .NET order. The callback must not add to or remove from
// d itself (the C# would throw InvalidOperationException).
func (d *netDict[V]) each(fn func(key string, v V) bool) {
	for i := range d.slots {
		if d.slots[i].used {
			if !fn(d.slots[i].key, d.slots[i].val) {
				return
			}
		}
	}
}

// dispatcher serialises event delivery the way the installer's
// SynchronizationContext.Post / UI thread does: callbacks run one at a time,
// in posting order, on a goroutine that never holds a package lock.
type dispatcher struct {
	mu      sync.Mutex
	queue   []func()
	running bool
	idle    *sync.Cond
}

func newDispatcher() *dispatcher {
	d := &dispatcher{}
	d.idle = sync.NewCond(&d.mu)
	return d
}

func (d *dispatcher) post(f func()) {
	d.mu.Lock()
	d.queue = append(d.queue, f)
	if !d.running {
		d.running = true
		go d.run()
	}
	d.mu.Unlock()
}

func (d *dispatcher) run() {
	for {
		d.mu.Lock()
		if len(d.queue) == 0 {
			d.running = false
			d.idle.Broadcast()
			d.mu.Unlock()
			return
		}
		f := d.queue[0]
		d.queue[0] = nil
		d.queue = d.queue[1:]
		d.mu.Unlock()
		f()
	}
}

// wait blocks until every posted callback has run (tests and shutdown).
func (d *dispatcher) wait() {
	d.mu.Lock()
	for d.running || len(d.queue) > 0 {
		d.idle.Wait()
	}
	d.mu.Unlock()
}

// subscribers is a C#-event-like (+= / -=) list of callbacks. The list is
// read at delivery time, so a callback removed before a queued event is
// delivered is not invoked (as with Tmds' posted closures that re-check the
// event field).
type subscribers[E any] struct {
	mu   sync.Mutex
	next int
	fns  []subscriber[E]
}

type subscriber[E any] struct {
	id int
	fn func(E)
}

func (s *subscribers[E]) add(fn func(E)) (remove func()) {
	s.mu.Lock()
	s.next++
	id := s.next
	s.fns = append(s.fns, subscriber[E]{id: id, fn: fn})
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, f := range s.fns {
			if f.id == id {
				s.fns = append(s.fns[:i:i], s.fns[i+1:]...)
				return
			}
		}
	}
}

func (s *subscribers[E]) snapshot() []func(E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]func(E), len(s.fns))
	for i, f := range s.fns {
		out[i] = f.fn
	}
	return out
}

func (s *subscribers[E]) deliver(ev E) {
	for _, fn := range s.snapshot() {
		fn(ev)
	}
}

// clock abstracts DateTime.Now and System.Threading.Timer so the browser's
// timing can be driven deterministically in tests.
type clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) stopper
}

type stopper interface{ Stop() bool }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) AfterFunc(d time.Duration, f func()) stopper { return time.AfterFunc(d, f) }
