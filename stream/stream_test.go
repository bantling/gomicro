// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/bantling/gomicro/funcs"
	"github.com/bantling/gomicro/iter"
	"github.com/stretchr/testify/assert"
)

// ==== Functions

func TestComposeTransforms(t *testing.T) {
	var (
		f1 = func(it *iter.Iter) *iter.Iter {
			return iter.New(
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

		f2 = func(it *iter.Iter) *iter.Iter {
			return iter.New(
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

		c  = composeTransforms(f1, f2)
		it = iter.Of(0, 1, 2, 3, 4, 5)
	)

	assert.Equal(t, []int{1, 2, 3, 4}, c(it).ToSliceOf(0))
}

func TestIterateFunc(t *testing.T) {
	// f is already a func(interface{}) interface{}, gets returned as is
	{
		f := func(arg interface{}) interface{} {
			return arg
		}

		iterFunc := IterateFunc(f)
		assert.Equal(t, fmt.Sprintf("%p", f), fmt.Sprintf("%p", iterFunc))
		assert.Equal(t, 0, iterFunc(0))
		assert.Equal(t, 1, iterFunc(1))

		it := Iterate(2, iterFunc)
		assert.Equal(t, 2, it.NextIntValue())
		assert.Equal(t, 2, it.NextIntValue())

		fin := New().AndThen().Limit(3)
		assert.Equal(t, []int{2, 2, 2}, fin.ToSliceOf(0, it))
	}

	// f can be adapted to a func(interface{}) interface{}
	{
		f := func(arg int) int {
			return arg
		}

		iterFunc := IterateFunc(f)
		assert.NotEqual(t, fmt.Sprintf("%p", f), fmt.Sprintf("%p", iterFunc))
		assert.Equal(t, 0, iterFunc(0))
		assert.Equal(t, 1, iterFunc(1))

		it := Iterate(2, iterFunc)
		assert.Equal(t, 2, it.NextIntValue())
		assert.Equal(t, 2, it.NextIntValue())

		fin := New().AndThen().Limit(3)
		assert.Equal(t, []int{2, 2, 2}, fin.ToSliceOf(0, it))
	}

	// Series of all ints starting at seed
	{
		f := func(arg int) int {
			return arg + 1
		}

		iterFunc := IterateFunc(f)
		assert.NotEqual(t, fmt.Sprintf("%p", f), fmt.Sprintf("%p", iterFunc))
		assert.Equal(t, 1, iterFunc(0))
		assert.Equal(t, 2, iterFunc(1))
		assert.Equal(t, 3, iterFunc(2))
		assert.Equal(t, 4, iterFunc(3))

		it := Iterate(0, iterFunc)
		assert.Equal(t, 0, it.NextIntValue())
		assert.Equal(t, 1, it.NextIntValue())
		assert.Equal(t, 2, it.NextIntValue())
		assert.Equal(t, 3, it.NextIntValue())

		it = Iterate(0, iterFunc)
		fin := New().AndThen().Limit(3)
		assert.Equal(t, []int{0, 1, 2}, fin.ToSliceOf(0, it))
	}

	// Fibonacci func, ignores args. Assumes seed of 0.
	{
		g := func() func(int) int {
			var (
				prev1 = 0
				prev2 = 1
				first = true
			)

			return func(int) int {
				if first {
					first = false
					return 1
				}

				result := prev1 + prev2
				prev1 = prev2
				prev2 = result
				return result
			}
		}

		f := g()
		iterFunc := IterateFunc(f)
		assert.NotEqual(t, fmt.Sprintf("%p", f), fmt.Sprintf("%p", iterFunc))
		assert.Equal(t, 1, iterFunc(0))
		assert.Equal(t, 1, iterFunc(0))
		assert.Equal(t, 2, iterFunc(0))
		assert.Equal(t, 3, iterFunc(0))
		assert.Equal(t, 5, iterFunc(0))

		f = g()
		iterFunc = IterateFunc(f)
		it := Iterate(0, iterFunc)
		assert.Equal(t, 0, it.NextIntValue())
		assert.Equal(t, 1, it.NextIntValue())
		assert.Equal(t, 1, it.NextIntValue())
		assert.Equal(t, 2, it.NextIntValue())
		assert.Equal(t, 3, it.NextIntValue())
		assert.Equal(t, 5, it.NextIntValue())

		f = g()
		iterFunc = IterateFunc(f)
		it = Iterate(0, iterFunc)
		fin := New().AndThen().Limit(6)
		assert.Equal(t, []int{0, 1, 1, 2, 3, 5}, fin.ToSliceOf(0, it))

		f = g()
		iterFunc = IterateFunc(f)
		it = Iterate(0, iterFunc)
		fin = New().AndThen().Skip(2).Limit(6)
		assert.Equal(t, []int{1, 2, 3, 5, 8, 13}, fin.ToSliceOf(0, it))
	}

	// f is not a function
	{
		f := 0

		defer func() {
			assert.Equal(t, "f must be a function", recover())
		}()

		IterateFunc(f)
		assert.Fail(t, "Must panic")
	}

	// f does not have one arg
	{
		f := func() uint {
			return 0
		}

		defer func() {
			assert.Equal(t, "f must accept and return a single value of the exact same type", recover())
		}()

		IterateFunc(f)
		assert.Fail(t, "Must panic")
	}

	// f does not have one return type
	{
		f := func(int) {
			//
		}

		defer func() {
			assert.Equal(t, "f must accept and return a single value of the exact same type", recover())
		}()

		IterateFunc(f)
		assert.Fail(t, "Must panic")
	}

	// f has one arg and return type, but they are not the same
	{
		f := func(arg int) uint {
			return 0
		}

		defer func() {
			assert.Equal(t, "f must accept and return a single value of the exact same type", recover())
		}()

		IterateFunc(f)
		assert.Fail(t, "Must panic")
	}
}

func TestIterate(t *testing.T) {
	iter := Iterate(1, IterateFunc(func(val int) int { return val * 2 }))
	assert.Equal(t, 1, iter.NextIntValue())
	assert.Equal(t, 2, iter.NextIntValue())
	assert.Equal(t, 4, iter.NextIntValue())
	assert.Equal(t, 8, iter.NextIntValue())
}

// ==== Constructors

func TestStreamZeroValue(t *testing.T) {
	s := &Stream{}
	assert.Equal(t, []interface{}{1, 2, 3}, s.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestStreamNew(t *testing.T) {
	s := New()
	assert.Equal(t, []interface{}{1, 2, 3}, s.Iter(iter.Of(1, 2, 3)).ToSlice())
}

// ==== Transforms

func TestStreamTransform(t *testing.T) {
	s := New().
		Transform(func(it *iter.Iter) *iter.Iter {
			return iter.New(func() (interface{}, bool) {
				if it.Next() {
					return it.IntValue() * 2, true
				}

				return nil, false
			})
		})

	assert.Equal(t, []int{2, 4, 6}, s.Iter(iter.Of(1, 2, 3)).ToSliceOf(0))
}

func TestStreamFilter(t *testing.T) {
	fn := func(element interface{}) bool { return element.(int) < 3 }
	s := New().Filter(fn)
	assert.Equal(t, []interface{}{}, s.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, s.Iter(iter.Of(1, 2, 3)).ToSlice())

	fn2 := funcs.Filter(func(element int) bool { return element < 3 })
	s = New().Filter(fn2)
	assert.Equal(t, []int{1, 2}, s.Iter(iter.Of(1, 2, 3)).ToSliceOf(0))
}

func TestStreamFilterNot(t *testing.T) {
	fn := func(element interface{}) bool { return element.(int) < 3 }
	s := New().FilterNot(fn)
	assert.Equal(t, []interface{}{}, s.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{3}, s.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestStreamMap(t *testing.T) {
	fn := func(element interface{}) interface{} {
		return strconv.Itoa(element.(int) * 2)
	}
	s := New().Map(fn)
	assert.Equal(t, []interface{}{}, s.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{"2"}, s.Iter(iter.Of(1)).ToSlice())
	assert.Equal(t, []interface{}{"2", "4"}, s.Iter(iter.Of(1, 2)).ToSlice())

	fn2 := funcs.Map(func(element int) string { return strconv.Itoa(element * 2) })
	s = New().Map(fn2)
	assert.Equal(t, []string{"2", "4"}, s.Iter(iter.Of(1, 2)).ToSliceOf(""))
}

func TestStreamPeek(t *testing.T) {
	var elements []interface{}
	fn := func(element interface{}) {
		elements = append(elements, element)
	}
	s := New().Peek(fn)
	s.Iter(iter.Of()).ToSlice()
	assert.Equal(t, []interface{}(nil), elements)

	elements = nil
	s.Iter(iter.Of(1)).ToSlice()
	assert.Equal(t, []interface{}{1}, elements)

	elements = nil
	s.Iter(iter.Of(1, 2)).ToSlice()
	assert.Equal(t, elements, []interface{}{1, 2})

	var elements2 []int
	fn2 := funcs.Consumer(func(element int) { elements2 = append(elements2, element) })
	s = New().Peek(fn2)
	s.Iter(iter.Of(1, 2)).ToSlice()
	assert.Equal(t, elements2, []int{1, 2})
}

// ==== Continuation

func TestStreamIter(t *testing.T) {
	s := New()
	assert.Equal(t, []interface{}{1}, s.Iter(iter.Of(1)).ToSlice())
}

func TestStreamAndThen(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1}, f.Iter(iter.Of(1)).ToSlice())
}
