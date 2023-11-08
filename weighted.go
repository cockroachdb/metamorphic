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

package metamorphic

import "math/rand"

// ItemWeight holds an item and its corresponding weight.
type ItemWeight[I any] struct {
	Item   I
	Weight int
}

// Weighted is a slice of items and their weights.
type Weighted[I any] []ItemWeight[I]

func (w Weighted[I]) total() int {
	var total int
	for i := 0; i < len(w); i++ {
		total += w[i].Weight
	}
	return total
}

// Random returns a function that returns one item at random, using the
// distribution indicated by item weights and the provided pseudorandom number
// generator for randomness.
func (w Weighted[I]) Random(rng *rand.Rand) func() I {
	total := w.total()
	return func() I {
		t := rng.Intn(total)
		for i := 0; i < len(w); i++ {
			t -= w[i].Weight
			if t < 0 {
				return w[i].Item
			}
		}
		panic("unreachable")
	}
}

// RandomDeck returns a function that returns one item at random, using a
// deck-style distribution to ensure each item is returned with the desired
// frequency. Randomness is taken from the provided pseudorandom number
// generator.
func (w Weighted[I]) RandomDeck(rng *rand.Rand) func() I {
	total := w.total()
	deck := make([]int, 0, total)
	for i := range w {
		for j := 0; j < w[i].Weight; j++ {
			deck = append(deck, i)
		}
	}
	index := len(deck)
	return func() I {
		if index == len(deck) {
			rng.Shuffle(len(deck), func(i, j int) {
				deck[i], deck[j] = deck[j], deck[i]
			})
			index = 0
		}
		it := w[deck[index]].Item
		index++
		return it
	}
}
