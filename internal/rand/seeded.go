package rand

import (
	"math/rand"
	"sync"
	"time"
)

// Seeded is an interface for seeded, concurrency-safe random number generators.
type Seeded interface {
	// Intn returns, as an int, a non-negative pseudo-random number in [0,max). It
	// panics if max <= 0.
	Intn(max int) int
	// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0).
	Float64() float64
}

type seeded struct {
	rand *rand.Rand
	mut  *sync.Mutex
}

// NewSeeded returns a seeded, concurrency-safe random number generator.
func NewSeeded() Seeded {
	// nolint: gosec
	rnd := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	return &seeded{
		rand: rnd,
		mut:  &sync.Mutex{},
	}
}

func (s *seeded) Intn(max int) int {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.rand.Intn(max)
}

func (s *seeded) Float64() float64 {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.rand.Float64()
}
