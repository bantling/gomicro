// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	// Exact match
	filterFn := Filter(func(i interface{}) bool { return i.(int) < 3 })
	assert.True(t, filterFn(1))
	assert.False(t, filterFn(5))

	// Inexact match
	filterFn = Filter(func(i int) bool { return i < 3 })
	assert.True(t, filterFn(uint8(1)))
	assert.False(t, filterFn(5))

	// Exact and Inexact match all
	filterFns := FilterAll(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i >= 0 },
	)

	assert.True(t, filterFns[0](1))
	assert.False(t, filterFns[1](int8(-1)))

	// Exact and Inexact match And
	filterFn = And(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i >= 0 },
	)

	assert.True(t, filterFn(1))
	assert.False(t, filterFn(-1))

	// Exact and Inexact match Or
	filterFn = Or(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i%2 == 0 },
	)

	assert.True(t, filterFn(1))
	assert.False(t, filterFn(5))

	// Exact and Inexact match Not
	filterFn = Not(func(i interface{}) bool { return i.(int) < 3 })

	assert.False(t, filterFn(1))
	assert.True(t, filterFn(5))

	// EqualTo/DeepEqualTo
	filterFn = EqualTo(nil)
	filterFn2 := DeepEqualTo(nil)

	assert.True(t, filterFn(nil))
	assert.True(t, filterFn2(nil))
	assert.False(t, filterFn(0))
	assert.False(t, filterFn2(0))

	filterFn = EqualTo(([]int)(nil))
	filterFn2 = DeepEqualTo(([]int)(nil))

	assert.True(t, filterFn(([]int)(nil)))
	assert.True(t, filterFn2(([]int)(nil)))
	assert.False(t, filterFn(nil))
	assert.False(t, filterFn2(nil))
	assert.False(t, filterFn(([]string)(nil)))
	assert.False(t, filterFn2(([]string)(nil)))

	theVal := []int{1, 2}
	filterFn = EqualTo(theVal)
	filterFn2 = DeepEqualTo([]int{1, 2})

	assert.False(t, filterFn(([]int)(nil)))
	assert.False(t, filterFn2(([]int)(nil)))
	assert.False(t, filterFn(nil))
	assert.False(t, filterFn2(nil))
	assert.False(t, filterFn([]int{1}))
	assert.False(t, filterFn2([]int{1}))
	assert.False(t, filterFn([]int{1, 2}))
	assert.True(t, filterFn2([]int{1, 2}))
	assert.True(t, filterFn(theVal))
	assert.True(t, filterFn2(theVal))

	filterFn = EqualTo(1)
	filterFn2 = DeepEqualTo(1)

	assert.True(t, filterFn(int8(1)))
	assert.True(t, filterFn2(int8(1)))
	assert.False(t, filterFn(5))
	assert.False(t, filterFn2(5))

	// LessThan
	filterFn = IsLessThan(5)
	assert.True(t, filterFn(int8(3)))
	assert.False(t, filterFn(5))

	// LessThanEquals
	filterFn = IsLessThanEquals(5)
	assert.True(t, filterFn(int8(5)))
	assert.False(t, filterFn(6))

	// GreaterThan
	filterFn = IsGreaterThan(5)
	assert.True(t, filterFn(int8(6)))
	assert.False(t, filterFn(5))

	// GreaterThanEquals
	filterFn = IsGreaterThanEquals(5)
	assert.True(t, filterFn(int8(5)))
	assert.False(t, filterFn(4))

	// IsNegative
	filterFn = IsNegative
	assert.True(t, filterFn(int8(-1)))
	assert.False(t, filterFn(0))

	// IsNonNegative
	filterFn = IsNonNegative
	assert.True(t, filterFn(int8(0)))
	assert.False(t, filterFn(-1))

	// IsPositive
	filterFn = IsPositive
	assert.True(t, filterFn(int8(1)))
	assert.False(t, filterFn(0))

	// Nil
	filterFn = IsNil

	assert.True(t, filterFn(nil))
	assert.True(t, filterFn([]int(nil)))
	var f func()
	assert.True(t, filterFn(f))
	assert.False(t, filterFn(""))

	// IsNilable
	filterFn = IsNilable
	assert.True(t, filterFn(nil))
	assert.True(t, filterFn([]int(nil)))
	assert.True(t, filterFn(f))
	assert.False(t, filterFn(""))

	deferFunc := func() {
		assert.Equal(t, filterErrorMsg, recover())
	}

	func() {
		defer deferFunc()

		// Not a func
		Filter(0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil
		Filter(nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil func
		var fn func()
		Filter(fn)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No arg
		Filter(func() {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No result
		Filter(func(int) {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Wrong result type
		Filter(func(int) int { return 0 })
		assert.Fail(t, "must panic")
	}()
}
