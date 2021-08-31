// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"testing"

	"github.com/bantling/gomicro/iter"
	"github.com/stretchr/testify/assert"
)

// ==== Compose

func TestComposeGenerators(t *testing.T) {
	var (
		f1 = func() func(it *iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); val.(int) < 5 {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		}

		f2 = func() func(it *iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); val.(int) > 0 {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		}

		c  = composeGenerators(f1, f2)
		it = iter.Of(0, 1, 2, 3, 4, 5)
	)

	assert.Equal(t, []int{1, 2, 3, 4}, c()(it).ToSliceOf(0))
}

// ==== SetReduce
