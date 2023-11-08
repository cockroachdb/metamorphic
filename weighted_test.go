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

import (
	"math/rand"
	"testing"
	"time"
)

func TestWeighted_RandomDeck(t *testing.T) {
	weighted := Weighted[int]{
		{Item: 1, Weight: 1},
		{Item: 2, Weight: 2},
		{Item: 3, Weight: 3},
		{Item: 4, Weight: 4},
		{Item: 5, Weight: 5},
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	next := weighted.RandomDeck(rng)

	// Pull weighted.total() elements from `next`.
	total := weighted.total()
	const passesThroughTheDeck = 5
	dist := make([]int, len(weighted))
	for p := 0; p < passesThroughTheDeck; p++ {
		for i := 0; i < total; i++ {
			it := next()
			j := it - 1
			dist[j]++
			if dist[j] > (p+1)*weighted[j].Weight {
				t.Errorf("Item %d (weight = %d) has already appeared %d times in %d passes",
					it, weighted[j].Weight, dist[j], p)
			}
		}
	}
	for i := range dist {
		if dist[i] != passesThroughTheDeck*weighted[i].Weight {
			t.Errorf("Item %d appeared %d times in %d passes, but its weight is %d",
				weighted[i].Item, dist[i], passesThroughTheDeck, weighted[i].Weight)
		}
	}
}
