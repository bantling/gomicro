// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/bantling/gomicro/funcs"
	"github.com/bantling/gomicro/iter"
	"github.com/stretchr/testify/assert"
)

// ==== Functions

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

// ==== Finisher Transforms

func TestFinisherTransform(t *testing.T) {
	f := New().AndThen().Transform(func() func(*iter.Iter) *iter.Iter {
		return func(it *iter.Iter) *iter.Iter {
			return iter.NewIter(func() (interface{}, bool) {
				if it.Next() {
					return it.IntValue() * 2, true
				}

				return nil, false
			})
		}
	})

	assert.Equal(t, []int{2, 4, 6}, f.Iter(iter.Of(1, 2, 3)).ToSliceOf(0))
}

func TestFinisherDistinct(t *testing.T) {
	f := New().AndThen().Distinct()
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1}, f.Iter(iter.Of(1)).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, f.Iter(iter.Of(1, 1, 2)).ToSlice())
	assert.Equal(t, []interface{}{1, 2, 3}, f.Iter(iter.Of(1, 2, 2, 1, 3)).ToSlice())
}

func TestFinisherDuplicate(t *testing.T) {
	f := New().AndThen().Duplicate()
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of(1)).ToSlice())
	assert.Equal(t, []interface{}{1}, f.Iter(iter.Of(1, 1, 2)).ToSlice())
	assert.Equal(t, []interface{}{2, 1}, f.Iter(iter.Of(1, 2, 2, 1, 3)).ToSlice())
}

func TestFinisherFilter(t *testing.T) {
	f := New().AndThen().Filter(func() func(element interface{}) bool {
		return func(element interface{}) bool {
			return element.(int) < 3
		}
	})
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, f.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestFinisherFilterNot(t *testing.T) {
	f := New().AndThen().FilterNot(func() func(element interface{}) bool {
		return func(element interface{}) bool {
			return element.(int) < 3
		}
	})
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{3}, f.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestFinisherLimit(t *testing.T) {
	f := New().AndThen().Limit(2)
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, f.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestFinisherReverseSort(t *testing.T) {
	f := New().AndThen().ReverseSort(funcs.IntSortFunc)
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{3, 2, 1}, f.Iter(iter.Of(2, 3, 1)).ToSlice())
}

func TestFinisherSetReduce(t *testing.T) {
	// Add pairs of ints to produce a new set of ints that is half the size.
	// If the source set is an odd length, the last int is returned as is.
	f := New().AndThen().SetReduce(
		func() func(*iter.Iter) (interface{}, bool) {
			return func(it *iter.Iter) (interface{}, bool) {
				var val1, val2 int

				if it.Next() {
					val1 = it.IntValue()

					if it.Next() {
						val2 = it.IntValue()
					}

					return val1 + val2, true
				}

				return nil, false
			}
		},
	)

	assert.Equal(t, []interface{}{}, f.ToSlice(iter.Of()))
	assert.Equal(t, []interface{}{1}, f.ToSlice(iter.Of(1)))
	assert.Equal(t, []interface{}{3}, f.ToSlice(iter.Of(1, 2)))
	assert.Equal(t, []interface{}{3, 3}, f.ToSlice(iter.Of(1, 2, 3)))
	assert.Equal(t, []interface{}{3, 7}, f.ToSlice(iter.Of(1, 2, 3, 4)))

	// Reader of bytes that are a json array of ints, where each int is returned as is.
	// EG, bytes must be of the form [1,20,300].
	// Returns an int[]
	f = New().AndThen().SetReduce(
		func() func(*iter.Iter) (interface{}, bool) {
			var (
				state      = 0
				currentVal = 0
				array      = []int{}
			)

			return func(it *iter.Iter) (interface{}, bool) {
				// Test next, unreading the value if we have one to simplify the following for loop
				if !it.Next() {
					return nil, false
				}
				it.Unread(it.Value())

			LOOP:
				for it.Next() {
					bval := it.IntValue()

					switch state {
					// State 0 expects [
					case 0:
						if bval != '[' {
							break LOOP
						}
						state = 1

					// State 1 expects first digit of next int
					case 1:
						if (bval < '0') || (bval > '9') {
							break LOOP
						}
						currentVal = bval - '0'
						state = 2

					// State 2 expects another digit, a comma, or ]
					case 2:
						switch {
						case (bval >= '0') && (bval <= '9'):
							currentVal = currentVal*10 + (bval - '0')

						case bval == ',':
							array = append(array, currentVal)
							state = 1

						case bval == ']':
							array = append(array, currentVal)
							state = 3
							break LOOP

						default:
							break LOOP
						}
					}
				}

				// State 3 indicates read complete array
				if state != 3 {
					panic("Invalid JSON array")
				}

				return array, true
			}
		},
	)

	assert.Equal(
		t,
		[]interface{}{[]int{1, 2, 3}},
		f.ToSlice(iter.OfReader(strings.NewReader("[1,2,3]"))),
	)

	assert.Equal(
		t,
		[]interface{}{[]int{1, 2, 3}, []int{10, 200, 3000}},
		f.ToSlice(iter.OfReader(strings.NewReader("[1,2,3][10,200,3000]"))),
	)

	for _, v := range []string{"{", "[", "[a", "[0a", "[0,"} {
		defer func() {
			assert.Equal(t, "Invalid JSON array", recover())
		}()

		f.ToSlice(iter.OfReader(strings.NewReader(v)))
		assert.Fail(t, "Must panic")
	}
}

func TestFinisherSkip(t *testing.T) {
	f := New().AndThen().Skip(2)
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of(1)).ToSlice())
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of(1, 2)).ToSlice())
	assert.Equal(t, []interface{}{3}, f.Iter(iter.Of(1, 2, 3)).ToSlice())
	assert.Equal(t, []interface{}{3, 4}, f.Iter(iter.Of(1, 2, 3, 4)).ToSlice())
}

func TestFinisherSort(t *testing.T) {
	f := New().AndThen().Sort(funcs.IntSortFunc)
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2, 3}, f.Iter(iter.Of(2, 3, 1)).ToSlice())
}

//
// ==== Finisher Terminals
//

func TestFinisherIter(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, []interface{}{}, f.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2, 3}, f.Iter(iter.Of(1, 2, 3)).ToSlice())
}

func TestFinisherAllMatch(t *testing.T) {
	fn := func(element interface{}) bool { return element.(int) < 3 }
	f := New().AndThen()
	assert.True(t, f.AllMatch(fn, iter.Of()))
	assert.True(t, f.AllMatch(fn, iter.Of(1, 2)))
	assert.False(t, f.AllMatch(fn, iter.Of(1, 2, 3)))
}

func TestFinisherAnyMatch(t *testing.T) {
	fn := func(element interface{}) bool { return element.(int) < 3 }
	f := New().AndThen()
	assert.False(t, f.AnyMatch(fn, iter.Of()))
	assert.True(t, f.AnyMatch(fn, iter.Of(1, 2)))
	assert.False(t, f.AnyMatch(fn, iter.Of(3)))
}

func TestFinisherAverage(t *testing.T) {
	f := New().AndThen()
	assert.True(t, f.Average(iter.Of()).IsEmpty())
	assert.Equal(t, 1.5, f.Average(iter.Of(1, 2)).MustGet())
	assert.Equal(t, 3.0, f.Average(iter.Of(3)).MustGet())
}

func TestFinisherCount(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, 0, f.Count(iter.Of()))
	assert.Equal(t, 2, f.Count(iter.Of(1, 2)))
}

func TestFinisherFirst(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, 1, f.First(iter.Of(1, 2, 3)).MustGet())

	f = New().Filter(func(v interface{}) bool { return v.(int) > 2 }).AndThen()
	assert.Equal(t, 3, f.First(iter.Of(1, 2, 3)).MustGet())
}

func TestFinisherForEach(t *testing.T) {
	var elements []interface{}
	fn := func(element interface{}) {
		elements = append(elements, element)
	}
	f := New().AndThen()
	f.ForEach(fn, iter.Of())
	assert.Equal(t, []interface{}(nil), elements)

	elements = nil
	f.ForEach(fn, iter.Of(1))
	assert.Equal(t, []interface{}{1}, elements)

	elements = nil
	f.ForEach(fn, iter.Of(1, 2, 3))
	assert.Equal(t, []interface{}{1, 2, 3}, elements)
}

func TestFinisherGroupBy(t *testing.T) {
	fn := func(element interface{}) (key interface{}) {
		return element.(int) % 3
	}
	f := New().AndThen()
	assert.Equal(t, map[interface{}][]interface{}{}, f.GroupBy(fn, iter.Of()))
	assert.Equal(t, map[interface{}][]interface{}{0: {0}}, f.GroupBy(fn, iter.Of(0)))
	assert.Equal(t, map[interface{}][]interface{}{0: {0}, 1: {1, 4}}, f.GroupBy(fn, iter.Of(0, 1, 4)))
}

func TestFinisherLast(t *testing.T) {
	f := New().AndThen()
	assert.True(t, f.Last(iter.Of()).IsEmpty())
	assert.Equal(t, 1, f.Last(iter.Of(1)).MustGet())
	assert.Equal(t, 2, f.Last(iter.Of(1, 2)).MustGet())
}

func TestFinisherMax(t *testing.T) {
	f := New().AndThen()
	assert.True(t, f.Max(funcs.IntSortFunc, iter.Of()).IsEmpty())
	assert.Equal(t, 1, f.Max(funcs.IntSortFunc, iter.Of(1)).MustGet())
	assert.Equal(t, 2, f.Max(funcs.IntSortFunc, iter.Of(1, 2)).MustGet())
	assert.Equal(t, 3, f.Max(funcs.IntSortFunc, iter.Of(1, 3, 2)).MustGet())
}

func TestFinisherMin(t *testing.T) {
	f := New().AndThen()
	assert.True(t, f.Min(funcs.IntSortFunc, iter.Of()).IsEmpty())
	assert.Equal(t, 1, f.Min(funcs.IntSortFunc, iter.Of(1)).MustGet())
	assert.Equal(t, 2, f.Min(funcs.IntSortFunc, iter.Of(2, 3)).MustGet())
	assert.Equal(t, 3, f.Min(funcs.IntSortFunc, iter.Of(4, 3, 5)).MustGet())
}

func TestFinisherNoneMatch(t *testing.T) {
	fn := func(element interface{}) bool { return element.(int) < 3 }
	f := New().AndThen()
	assert.True(t, f.NoneMatch(fn, iter.Of()))
	assert.True(t, f.NoneMatch(fn, iter.Of(3, 4)))
	assert.False(t, f.NoneMatch(fn, iter.Of(1, 2, 3)))
}

func TestFinisherReduce(t *testing.T) {
	fn := func(accumulator, element2 interface{}) interface{} {
		return accumulator.(int) + element2.(int)
	}
	f := New().AndThen()
	assert.Equal(t, 0, f.Reduce(0, fn, iter.Of()))
	assert.Equal(t, 7, f.Reduce(1, fn, iter.Of(1, 2, 3)))
}

func TestFinisherSum(t *testing.T) {
	f := New().AndThen()

	// Float64
	assert.True(t, f.Sum(iter.Of()).IsEmpty())
	assert.Equal(t, 3.25, f.Sum(iter.Of(1, 2.25)).Iter().NextFloat64Value())

	// Int
	assert.True(t, f.SumAsInt(iter.Of()).IsEmpty())
	assert.Equal(t, math.MaxInt, f.SumAsInt(iter.Of(1, math.MaxInt-1)).Iter().NextIntValue())

	// Uint
	assert.True(t, f.SumAsUint(iter.Of()).IsEmpty())
	assert.True(t, math.MaxUint == f.SumAsUint(iter.Of(1, math.MaxUint-uint(1))).Iter().NextUintValue())
}

func TestFinisherToMap(t *testing.T) {
	fn := func(element interface{}) (k interface{}, v interface{}) {
		return element, strconv.Itoa(element.(int))
	}
	f := New().AndThen()
	assert.Equal(t, map[interface{}]interface{}{}, f.ToMap(fn, iter.Of()))
	assert.Equal(t, map[interface{}]interface{}{1: "1"}, f.ToMap(fn, iter.Of(1)))
	assert.Equal(t, map[interface{}]interface{}{1: "1", 2: "2", 3: "3"}, f.ToMap(fn, iter.Of(1, 2, 3)))
}

func TestFinisherToMapOf(t *testing.T) {
	fn := func(element interface{}) (k interface{}, v interface{}) {
		return element, strconv.Itoa(element.(int))
	}
	f := New().AndThen()
	assert.Equal(t, map[int]string{}, f.ToMapOf(fn, 0, "0", iter.Of()))
	assert.Equal(t, map[int]string{1: "1"}, f.ToMapOf(fn, 0, "0", iter.Of(1)))
	assert.Equal(t, map[int]string{1: "1", 2: "2", 3: "3"}, f.ToMapOf(fn, 0, "0", iter.Of(1, 2, 3)))
}

func TestFinisherToSlice(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, []interface{}{}, f.ToSlice(iter.Of()))
	assert.Equal(t, []interface{}{1, 2}, f.ToSlice(iter.Of(1, 2)))
}

func TestFinisherToSliceOf(t *testing.T) {
	f := New().AndThen()
	assert.Equal(t, []int{}, f.ToSliceOf(0, iter.Of()))
	assert.Equal(t, []int{1, 2}, f.ToSliceOf(0, iter.Of(1, 2)))
}

func TestToByteWriter(t *testing.T) {
	f := New().AndThen()
	buf := &bytes.Buffer{}

	buf.Reset()
	f.ToByteWriter(buf, iter.Of())
	assert.Equal(t, []byte(nil), buf.Bytes())

	buf.Reset()
	f.ToByteWriter(buf, iter.Of(1))
	assert.Equal(t, []byte{1}, buf.Bytes())

	// Generate a buffer of exactly toWriterBufSize of a repeating cycle of values 0x00 thru 0xff
	data := make([]byte, toWriterBufSize)
	for i, j := 0, byte(0x00); i < toWriterBufSize; i++ {
		data[i] = j
		j++
		if j > math.MaxUint8 {
			j = 0
		}
	}
	assert.Equal(t, toWriterBufSize, len(data))
	assert.Equal(t, byte(0xff), data[len(data)-1])

	buf.Reset()
	f.ToByteWriter(buf, iter.OfElements(data))
	assert.Equal(t, data, buf.Bytes())

	// Try buffer size + 1
	dataPlus1 := append(data, 0x66)
	assert.Equal(t, toWriterBufSize+1, len(dataPlus1))

	buf.Reset()
	f.ToByteWriter(buf, iter.OfElements(dataPlus1))
	assert.Equal(t, dataPlus1, buf.Bytes())

	// Try exactly twice the buffer size
	dataTwice := append(data, data...)
	assert.Equal(t, toWriterBufSize*2, len(dataTwice))

	buf.Reset()
	f.ToByteWriter(buf, iter.OfElements(dataTwice))
	assert.Equal(t, dataTwice, buf.Bytes())

	// Try exactly twice the buffer size plus 1
	dataTwicePlus1 := append(dataTwice, 0x66)
	assert.Equal(t, toWriterBufSize*2+1, len(dataTwicePlus1))

	buf.Reset()
	f.ToByteWriter(buf, iter.OfElements(dataTwicePlus1))
	assert.Equal(t, dataTwicePlus1, buf.Bytes())
}

func TestToRuneWriter(t *testing.T) {
	f := New().AndThen()
	buf := &bytes.Buffer{}

	buf.Reset()
	f.ToRuneWriter(buf, iter.Of())
	assert.Equal(t, []byte(nil), buf.Bytes())

	buf.Reset()
	f.ToRuneWriter(buf, iter.Of('1'))
	assert.Equal(t, []byte(string('1')), buf.Bytes())

	// Generate a buffer of exactly toWriterBufSize of a repeating cycle of values '0' thru '9'
	data := make([]byte, toWriterBufSize)
	for i, j := 0, '0'; i < toWriterBufSize; i++ {
		data[i] = byte(j)
		j++
		if j > '9' {
			j = '0'
		}
	}
	assert.Equal(t, toWriterBufSize, len(data))

	buf.Reset()
	f.ToRuneWriter(buf, iter.OfElements(data))
	assert.Equal(t, data, buf.Bytes())

	// Try buffer size + 1
	dataPlus1 := append(data, 'A')
	assert.Equal(t, toWriterBufSize+1, len(dataPlus1))

	buf.Reset()
	f.ToRuneWriter(buf, iter.OfElements(dataPlus1))
	assert.Equal(t, dataPlus1, buf.Bytes())

	// Try exactly twice the buffer size
	dataTwice := append(data, data...)
	assert.Equal(t, toWriterBufSize*2, len(dataTwice))

	buf.Reset()
	f.ToRuneWriter(buf, iter.OfElements(dataTwice))
	assert.Equal(t, dataTwice, buf.Bytes())

	// Try exactly twice the buffer size plus 1
	dataTwicePlus1 := append(dataTwice, 'A')
	assert.Equal(t, toWriterBufSize*2+1, len(dataTwicePlus1))

	buf.Reset()
	f.ToRuneWriter(buf, iter.OfElements(dataTwicePlus1))
	assert.Equal(t, dataTwicePlus1, buf.Bytes())

	// Try 2 byte char à, 3 byte char ḁ, 4 byte char 𝆑
	buf.Reset()
	f.ToRuneWriter(buf, iter.Of('à', 'ḁ', '𝆑'))
	assert.Equal(t, []byte(string("àḁ𝆑")), buf.Bytes())
}

//
// ==== Finisher Continuation
//

func TestFinisherStream(t *testing.T) {
	s := New().AndThen().AndThen()
	assert.Equal(t, []interface{}{}, s.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, s.Iter(iter.Of(1, 2)).ToSlice())

	s = New().AndThen().AndThen(ParallelConfig{})
	assert.Equal(t, []interface{}{}, s.Iter(iter.Of()).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, s.Iter(iter.Of(1, 2)).ToSlice())
}

// ==== Sequence

func TestSequence(t *testing.T) {
	//      1,   2,   1,   3,   4,   3,   5,   6,   7,   7,   8,   9,  10
	f := New().
		Map(funcs.Map(func(i int) int { return i * 2 })).
		//  2,   4,   2,   6,   8,   6,  10,  12,  14,  14,  16,  18,  20
		Map(funcs.Map(func(i int) int { return i - 3 })).
		// -1,   1,  -1,   3,   5,   3,   7,   9,  11,  11,  13,  15,  17
		Filter(funcs.Filter(func(i int) bool { return i <= 7 })).
		// -1,   1,  -1,   3,   5,   3,   7
		AndThen().
		Skip(2).
		// -1,   3,   5,   3,   7
		Distinct().
		// -1,   3,   5,   7
		ReverseSort(funcs.IntSortFunc).
		//  7,   5,   3,  -1
		Limit(3)
		//  7,   5,   3

	itgen := func() *iter.Iter {
		return iter.Of(1, 2, 1, 3, 4, 3, 5, 6, 7, 7, 8, 9, 10)
	}
	// 7, 5, 3
	assert.Equal(t, 7, f.First(itgen()).MustGet())
	// 7, 5, 3
	assert.Equal(t, []int{7, 5, 3}, f.ToSliceOf(0, itgen()))
}

func TestParallel(t *testing.T) {
	var (
		input           = []interface{}{1, 2, 1, 3, 4, 3, 5, 6, 7, 7, 8, 9, 10}
		itgen           = func() *iter.Iter { return iter.Of(input...) }
		doubler         = funcs.Map(func(i int) int { return i * 2 })
		all             = []int{1, 2, 1, 3, 4, 3, 5, 6, 7, 7, 8, 9, 10}
		distinct        = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		doubled         = []int{2, 4, 2, 6, 8, 6, 10, 12, 14, 14, 16, 18, 20}
		doubledDistinct = []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
	)

	// a series of tests for all 4 combinations
	// transform?, finisher?

	// 00
	f := New().AndThen()
	assert.Equal(t, all, f.ToSliceOf(0, itgen(), ParallelConfig{}))

	// 01
	f = New().AndThen().Distinct()
	assert.Equal(t, distinct, f.ToSliceOf(0, itgen(), ParallelConfig{}))

	// 10
	f = New().Map(doubler).AndThen()
	assert.Equal(t, doubled, f.ToSliceOf(0, itgen(), ParallelConfig{}))

	// 11
	f = New().Map(doubler).AndThen().Distinct()
	assert.Equal(t, doubledDistinct, f.ToSliceOf(0, itgen(), ParallelConfig{}))
}

func TestThreadedReuse(t *testing.T) {
	var (
		f     = New().Filter(func(v interface{}) bool { return v.(int) > 5 }).AndThen().Sort(funcs.IntSortFunc)
		itgen = func() *iter.Iter { return iter.Of(3, 7, 8, 6) }
	)

	// Case 1: all routines use serial processing
	{
		wg := &sync.WaitGroup{}

		for i := 1; i <= 10; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				assert.Equal(t, []interface{}{6, 7, 8}, f.ToSlice(itgen()))
			}()
		}

		wg.Wait()
	}

	// Case 2: all goroutines use parallel processing
	{
		wg := &sync.WaitGroup{}

		for i := 1; i <= 10; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()

				assert.Equal(t, []interface{}{6, 7, 8}, f.ToSlice(itgen(), ParallelConfig{}))
			}()
		}

		wg.Wait()
	}

	// Case 3: half of goroutines use serial processing, half use parallel processing
	{
		wg := &sync.WaitGroup{}

		for i := 1; i <= 10; i++ {
			wg.Add(1)

			go func(row int) {
				defer wg.Done()

				if row <= 5 {
					assert.Equal(t, []interface{}{6, 7, 8}, f.ToSlice(itgen()))
				} else {
					assert.Equal(t, []interface{}{6, 7, 8}, f.ToSlice(itgen(), ParallelConfig{}))
				}
			}(i)
		}

		wg.Wait()
	}
}

func TestConcat(t *testing.T) {
	var (
		f1 = New().
			Map(func(element interface{}) interface{} { return element.(int) * 2 }).
			AndThen().
			Distinct()
		f2 = New().
			Map(func(element interface{}) interface{} { return element.(int) * 3 }).
			AndThen().
			Distinct()
		f3 = New().
			Map(func(element interface{}) interface{} { return element.(int) * 4 }).
			AndThen().
			Distinct()
		c = iter.Concat(
			f1.Iter(iter.Of(1, 2)),
			f2.Iter(iter.Of(3, 4, 5)),
			f3.Iter(iter.Of(6, 7, 8, 9)),
		)
	)

	assert.Equal(t,
		[]interface{}{
			2, 4,
			9, 12, 15,
			24, 28, 32, 36,
		},
		c.ToSlice(),
	)
}
