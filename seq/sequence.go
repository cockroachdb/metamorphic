// Copyright 2023 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

// Package seq provides facilities for creating and managing sequences of data.
package seq

import "math/rand"

// A Sequence represents an ordered sequence of elements.
type Sequence[I any] interface {
	// Next returns the next item in the sequence. If the sequence is beginning,
	// including for the first time, Next returns restarted = true.
	Next() (next I, restarted bool)
}

// RandomFilter returns a sequence formed by randomly filtering inner, using
// randomness from rng, returning any individual element with probability p.
func RandomFilter[I any](inner Sequence[I], rng *rand.Rand, p float64) Sequence[I] {
	return &randomFilter[I]{sequence: inner, prng: rng, probability: p}
}

type randomFilter[I any] struct {
	sequence    Sequence[I]
	prng        *rand.Rand
	probability float64
}

// Next implements Sequence.
func (s *randomFilter[I]) Next() (I, bool) {
	var restarted bool
	for {
		curr, currRestarted := s.sequence.Next()
		restarted = restarted || currRestarted
		if s.prng.Float64() < s.probability {
			return curr, restarted
		}
	}
}

// Slice is a sequence that pulls from a slice.
type Slice[I any] struct {
	Elems []I

	index IntsAscending[int]
}

// Next implements Sequence.
func (s *Slice[I]) Next() (I, bool) {
	if s.index.Max != len(s.Elems) {
		s.index = IntsAscending[int]{Min: 0, Max: len(s.Elems), v: 0}
	}
	i, restarted := s.index.Next()
	return s.Elems[i], restarted
}

// IntsAscending is a sequence of ascending integers in the range [Min,Max).
type IntsAscending[I ints] struct {
	Min, Max I
	v        I
}

// Next implements Sequence.
func (s *IntsAscending[I]) Next() (I, bool) {
	s.v += 1
	// If we're more than max, restart. We also restart if v <= min to
	// detect overflow.
	if s.v >= s.Max || s.v <= s.Min {
		s.v = s.Min
		return s.v, true
	}
	return s.v, false
}

// IntsDescending constructs a sequence of ascending integers in the range
// [min,max).
type IntsDescending[I ints] struct {
	Min, Max I
	v        I
}

// Next implements Sequence.
func (s *IntsDescending[I]) Next() (I, bool) {
	// If we're less than min, restart. We also restart if v >= max to
	// detect underflow.
	s.v -= 1
	if s.v < s.Min || s.v >= s.Max-1 {
		s.v = s.Max - 1
		return s.v, true
	}
	return s.v, false
}

type ints interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Func wraps a function with the signature func(I,bool), implementing the
// Sequence[I] interface.
type Func[I any] func() (I, bool)

// Next implements the Sequence[I] interface.
func (f Func[I]) Next() (I, bool) {
	return f()
}
