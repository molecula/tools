package loader

import "sync"

type stat struct {
	m  map[string]int
	mu sync.RWMutex
}

func (s *stat) inc(k string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k]++
}

func newStat() *stat {
	return &stat{
		m: make(map[string]int),
	}
}

type stats struct {
	e *stat
	s *stat
}

func newStats() *stats {
	return &stats{
		e: newStat(),
		s: newStat(),
	}
}

func (s *stats) success(k string) {
	s.s.inc(k)
}

func (s *stats) error(k string) {
	s.e.inc(k)
}

func (s *stats) counts() map[string][2]int {
	m := make(map[string][2]int)

	s.s.mu.RLock()
	for k, v := range s.s.m {
		a := m[k]
		a[0] = v
		m[k] = a
	}
	s.s.mu.RUnlock()

	s.e.mu.RLock()
	for k, v := range s.e.m {
		a := m[k]
		a[0] = v
		m[k] = a

	}
	s.e.mu.RUnlock()
	return m
}
