// Package limiter implementa un límite de ventana fija en memoria.
package limiter

import (
	"sync"
	"time"
)

const maxEntries = 10000

type entry struct {
	count int
	reset time.Time
}

// Limiter permite max eventos por ventana y clave.
type Limiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]*entry
}

// New crea un limiter con max eventos por ventana.
func New(max int, window time.Duration) *Limiter {
	if max <= 0 {
		max = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{max: max, window: window, now: time.Now, entries: make(map[string]*entry)}
}

// Allow registra un evento para key. Si se excede el límite devuelve false y
// el tiempo restante hasta el reinicio de la ventana.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if len(l.entries) > maxEntries {
		for k, e := range l.entries {
			if now.After(e.reset) {
				delete(l.entries, k)
			}
		}
	}

	e, ok := l.entries[key]
	if !ok || now.After(e.reset) {
		l.entries[key] = &entry{count: 1, reset: now.Add(l.window)}
		return true, 0
	}
	if e.count >= l.max {
		return false, time.Until(e.reset)
	}
	e.count++
	return true, 0
}
