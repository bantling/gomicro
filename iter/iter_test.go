// SPDX-License-Identifier: Apache-2.0

package iter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewIter(t *testing.T) {
	// Empty slice
	iter := NewIter(ArraySliceIterFunc(reflect.ValueOf([]int{})))
	assert.False(t, iter.Next())

	// Slice of 1 element
	iter = NewIter(ArraySliceIterFunc(reflect.ValueOf([]int{1})))

	assert.True(t, iter.Next())
	assert.Equal(t, 1, iter.Value())
	assert.False(t, iter.Next())
}

func TestOf(t *testing.T) {
	// Empty items
	iter := Of()
	assert.False(t, iter.Next())

	// One item
	iter = Of(5)
	assert.True(t, iter.Next())
	assert.Equal(t, 5, iter.Value())
	assert.False(t, iter.Next())

	// Two items
	iter = Of(5, []int{6, 7})
	assert.True(t, iter.Next())
	assert.Equal(t, 5, iter.Value())
	assert.True(t, iter.Next())
	assert.Equal(t, []int{6, 7}, iter.Value())
	assert.False(t, iter.Next())

	// Test semantics of Next and Value
	iter = Of(1, 2)
	{
		defer func() {
			assert.Equal(t, ErrValueNextFirst, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	assert.True(t, iter.Next())
	assert.True(t, iter.Next())
	assert.Equal(t, 1, iter.Value())

	assert.True(t, iter.Next())
	assert.True(t, iter.Next())
	assert.Equal(t, 2, iter.Value())

	assert.False(t, iter.Next())
	assert.False(t, iter.Next())
	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}
}

func TestOfFlatten(t *testing.T) {
	iter := OfFlatten([]interface{}{1, [2]int{2, 3}, [][]string{{"4", "5"}, {"6", "7", "8"}}})
	assert.Equal(t, 1, iter.NextValue())
	assert.Equal(t, 2, iter.NextValue())
	assert.Equal(t, 3, iter.NextValue())
	assert.Equal(t, "4", iter.NextValue())
	assert.Equal(t, "5", iter.NextValue())
	assert.Equal(t, "6", iter.NextValue())
	assert.Equal(t, "7", iter.NextValue())
	assert.Equal(t, "8", iter.NextValue())
	assert.False(t, iter.Next())
}

func TestOfElements(t *testing.T) {
	// ==== Array

	iter := OfElements([2]int{5, 6})
	assert.True(t, iter.Next())
	assert.Equal(t, 5, iter.Value())
	assert.True(t, iter.Next())
	assert.Equal(t, 6, iter.Value())
	assert.False(t, iter.Next())

	// ==== Slice

	iter = OfElements([]int{5, 6})
	assert.True(t, iter.Next())
	assert.Equal(t, 5, iter.Value())
	assert.True(t, iter.Next())
	assert.Equal(t, 6, iter.Value())
	assert.False(t, iter.Next())

	// ==== Map
	iter = OfElements(map[int]int{1: 2})
	assert.True(t, iter.Next())
	assert.Equal(t, KeyValue{1, 2}, iter.Value())
	assert.False(t, iter.Next())

	// ==== Nil

	iter = OfElements(nil)
	assert.False(t, iter.Next())

	// ==== One item

	iter = OfElements(5)
	assert.True(t, iter.Next())
	assert.Equal(t, 5, iter.Value())
	assert.False(t, iter.Next())
}

func TestConcat(t *testing.T) {
	iter := Concat()
	assert.Equal(t, []interface{}{}, iter.ToSlice())

	// 000
	iter = Concat(Of(), Of(), Of())
	assert.Equal(t, []interface{}{}, iter.ToSlice())

	// 001
	iter = Concat(Of(), Of(), Of(3))
	assert.Equal(t, []interface{}{3}, iter.ToSlice())

	// 010
	iter = Concat(Of(), Of(2), Of())
	assert.Equal(t, []interface{}{2}, iter.ToSlice())

	// 011
	iter = Concat(Of(), Of(2), Of(3))
	assert.Equal(t, []interface{}{2, 3}, iter.ToSlice())

	// 100
	iter = Concat(Of(1), Of(), Of())
	assert.Equal(t, []interface{}{1}, iter.ToSlice())

	// 101
	iter = Concat(Of(1), Of(), Of(3))
	assert.Equal(t, []interface{}{1, 3}, iter.ToSlice())

	// 110
	iter = Concat(Of(1), Of(2), Of())
	assert.Equal(t, []interface{}{1, 2}, iter.ToSlice())

	// 111
	iter = Concat(Of(1), Of(2), Of(3))
	assert.Equal(t, []interface{}{1, 2, 3}, iter.ToSlice())

	iter = Concat(Of(1, 2), Of(3), Of(4, 5, 6))
	assert.Equal(t, []interface{}{1, 2, 3, 4, 5, 6}, iter.ToSlice())
}

func TestValueOfType(t *testing.T) {
	var (
		v1   = "1"
		v2   = "2"
		iter = Of(v1, v2)
	)

	next := iter.Next()
	assert.True(t, next)
	var v string = iter.ValueOfType("").(string)
	assert.Equal(t, v1, v)
	v = iter.NextValueOfType("").(string)
	assert.Equal(t, v2, v)
}

func TestBoolValue(t *testing.T) {
	var (
		iter = Of(true, false)
	)

	next := iter.Next()
	assert.True(t, next)
	var v bool = iter.BoolValue()
	assert.True(t, v)

	v = iter.NextBoolValue()
	assert.False(t, v)
}

func TestIntValue(t *testing.T) {
	{
		var (
			v1   = byte(1)
			v2   = byte(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v byte = iter.ByteValue()
		assert.Equal(t, v1, v)
		v = iter.NextByteValue()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = '1'
			v2   = '2'
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v rune = iter.RuneValue()
		assert.Equal(t, v1, v)
		v = iter.NextRuneValue()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = 1
			v2   = 2
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v int = iter.IntValue()
		assert.Equal(t, v1, v)
		v = iter.NextIntValue()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = int8(1)
			v2   = int8(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v int8 = iter.Int8Value()
		assert.Equal(t, v1, v)
		v = iter.NextInt8Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = int16(1)
			v2   = int16(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v int16 = iter.Int16Value()
		assert.Equal(t, v1, v)
		v = iter.NextInt16Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = int32(1)
			v2   = int32(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v int32 = iter.Int32Value()
		assert.Equal(t, v1, v)
		v = iter.NextInt32Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = int64(1)
			v2   = int64(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v int64 = iter.Int64Value()
		assert.Equal(t, v1, v)
		v = iter.NextInt64Value()
		assert.Equal(t, v2, v)
	}
}

func TestUintValue(t *testing.T) {
	{
		var (
			v1   = uint(1)
			v2   = uint(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v uint = iter.UintValue()
		assert.Equal(t, v1, v)
		v = iter.NextUintValue()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = uint8(1)
			v2   = uint8(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v uint8 = iter.Uint8Value()
		assert.Equal(t, v1, v)
		v = iter.NextUint8Value()
		assert.Equal(t, v2, v)
	}
	{
		var (
			v1   = uint16(1)
			v2   = uint16(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v uint16 = iter.Uint16Value()
		assert.Equal(t, v1, v)
		v = iter.NextUint16Value()
		assert.Equal(t, v2, v)
	}
	{
		var (
			v1   = uint32(1)
			v2   = uint32(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v uint32 = iter.Uint32Value()
		assert.Equal(t, v1, v)
		v = iter.NextUint32Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = uint64(1)
			v2   = uint64(2)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v uint64 = iter.Uint64Value()
		assert.Equal(t, v1, v)
		v = iter.NextUint64Value()
		assert.Equal(t, v2, v)
	}
}

func TestFloatValue(t *testing.T) {
	{
		var (
			v1   = float32(1.25)
			v2   = float32(2.5)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v float32 = iter.Float32Value()
		assert.Equal(t, v1, v)
		v = iter.NextFloat32Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = float64(1.25)
			v2   = float64(2.5)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v float64 = iter.Float64Value()
		assert.Equal(t, v1, v)
		v = iter.NextFloat64Value()
		assert.Equal(t, v2, v)
	}
}

func TestComplexValue(t *testing.T) {
	{
		var (
			v1   = complex64(1 + 2i)
			v2   = complex64(3 + 4i)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v complex64 = iter.Complex64Value()
		assert.Equal(t, v1, v)
		v = iter.NextComplex64Value()
		assert.Equal(t, v2, v)
	}

	{
		var (
			v1   = complex128(1 + 2i)
			v2   = complex128(3 + 4i)
			iter = Of(v1, v2)
		)

		assert.True(t, iter.Next())
		var v complex128 = iter.Complex128Value()
		assert.Equal(t, v1, v)
		v = iter.NextComplex128Value()
		assert.Equal(t, v2, v)
	}
}

func TestStringValue(t *testing.T) {
	var (
		v1   = "1"
		v2   = "2"
		iter = Of(v1, v2)
	)

	assert.True(t, iter.Next())
	var v string = iter.StringValue()
	assert.Equal(t, v1, v)
	v = iter.NextStringValue()
	assert.Equal(t, v2, v)
}

func TestUnread(t *testing.T) {
	iter := Of(1, 2, 3)
	iter.Next()
	iter.Unread(1)

	for i := 1; i <= 3; i++ {
		assert.Equal(t, i, iter.NextValue())
	}

	// Unread backwards just to prove it works
	iter.Unread(1)
	iter.Unread(2)
	iter.Unread(3)

	for i := 3; i >= 1; i-- {
		// Test NextValue
		assert.Equal(t, i, iter.NextValue())
	}
	assert.False(t, iter.Next())

	// Test unreading before even reading
	iter = Of(1)
	iter.Unread(2)
	for i := 2; i >= 1; i-- {
		// Test Next/Value
		iter.Next()
		assert.Equal(t, i, iter.Value())
	}
	assert.False(t, iter.Next())

	// Unreading doesn't affect panic on exhausted iter
	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must die")
	}
}

func TestSplitIntoRows(t *testing.T) {
	// Split with n = 5 items per subslice
	var (
		iter  = Of()
		split = iter.SplitIntoRows(5)
	)
	assert.Equal(t, [][]interface{}{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2, 3, 4)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3, 4}}, split)

	iter = Of(1, 2, 3, 4, 5)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3, 4, 5}}, split)

	iter = Of(1, 2, 3, 4, 5, 6)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3, 4, 5}, {6}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	split = iter.SplitIntoRows(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}, {11}}, split)

	// Split with n = 1 items per subslice corner case
	iter = Of()
	split = iter.SplitIntoRows(1)
	assert.Equal(t, [][]interface{}{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoRows(1)
	assert.Equal(t, [][]interface{}{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2)
	split = iter.SplitIntoRows(1)
	assert.Equal(t, [][]interface{}{{1}, {2}}, split)

	// Die if n < 1
	{
		defer func() {
			assert.Equal(t, ErrColsGreaterThanZero, recover())
		}()

		iter.SplitIntoRows(0)
		assert.Fail(t, "Must panic")
	}
}

func TestSplitIntoRowsOf(t *testing.T) {
	// Split with n = 5 items per subslice
	var (
		iter  = Of()
		split = iter.SplitIntoRowsOf(5, 0)
	)
	assert.Equal(t, [][]int{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2, 3, 4)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3, 4}}, split)

	iter = Of(1, 2, 3, 4, 5)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}}, split)

	iter = Of(1, 2, 3, 4, 5, 6)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}, {6}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3, 4, 5}, {6, 7, 8, 9, 10}, {11}}, split)

	// Split into a type that requires conversion
	iter = Of(uint(1), uint(2))
	split = iter.SplitIntoRowsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2}}, split)

	// Split with n = 1 items per subslice corner case
	iter = Of()
	split = iter.SplitIntoRowsOf(1, 0)
	assert.Equal(t, [][]int{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoRowsOf(1, 0)
	assert.Equal(t, [][]int{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2)
	split = iter.SplitIntoRowsOf(1, 0)
	assert.Equal(t, [][]int{{1}, {2}}, split)

	// Die if n < 1
	{
		defer func() {
			assert.Equal(t, ErrColsGreaterThanZero, recover())
		}()

		iter.SplitIntoRowsOf(0, 0)
		assert.Fail(t, "Must panic")
	}

	// Die if value is nil
	{
		defer func() {
			assert.Equal(t, ErrValueCannotBeNil, recover())
		}()

		iter.SplitIntoRowsOf(1, nil)
		assert.Fail(t, "Must panic")
	}
}

func TestSplitIntoColumns(t *testing.T) {
	// Split with n = 5 columns per subslice
	var (
		iter  = Of()
		split = iter.SplitIntoColumns(5)
	)
	assert.Equal(t, [][]interface{}{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2, 3, 4)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1}, {2}, {3}, {4}}, split)

	iter = Of(1, 2, 3, 4, 5)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1}, {2}, {3}, {4}, {5}}, split)

	iter = Of(1, 2, 3, 4, 5, 6)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1, 2}, {3}, {4}, {5}, {6}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9, 10}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	split = iter.SplitIntoColumns(5)
	assert.Equal(t, [][]interface{}{{1, 2, 3}, {4, 5}, {6, 7}, {8, 9}, {10, 11}}, split)

	// Split with n = 1 columns per subslice corner case
	iter = Of()
	split = iter.SplitIntoColumns(1)
	assert.Equal(t, [][]interface{}{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoColumns(1)
	assert.Equal(t, [][]interface{}{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2)
	split = iter.SplitIntoColumns(1)
	assert.Equal(t, [][]interface{}{{1, 2}}, split)

	// Die if n < 1
	{
		defer func() {
			assert.Equal(t, ErrRowsGreaterThanZero, recover())
		}()

		iter.SplitIntoColumns(0)
		assert.Fail(t, "Must panic")
	}
}

func TestSplitIntoColumnsOf(t *testing.T) {
	// Split with n = 5 columns per subslice
	var (
		iter  = Of()
		split = iter.SplitIntoColumnsOf(5, 0)
	)
	assert.Equal(t, [][]int{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2, 3, 4)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1}, {2}, {3}, {4}}, split)

	iter = Of(1, 2, 3, 4, 5)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1}, {2}, {3}, {4}, {5}}, split)

	iter = Of(1, 2, 3, 4, 5, 6)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2}, {3}, {4}, {5}, {6}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5, 6}, {7, 8}, {9, 10}}, split)

	iter = Of(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11)
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1, 2, 3}, {4, 5}, {6, 7}, {8, 9}, {10, 11}}, split)

	// Split into a type that requires conversion
	iter = Of(uint(1), uint(2))
	split = iter.SplitIntoColumnsOf(5, 0)
	assert.Equal(t, [][]int{{1}, {2}}, split)

	// Split with n = 1 columns per subslice corner case
	iter = Of()
	split = iter.SplitIntoColumnsOf(1, 0)
	assert.Equal(t, [][]int{}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1)
	split = iter.SplitIntoColumnsOf(1, 0)
	assert.Equal(t, [][]int{{1}}, split)

	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}

	iter = Of(1, 2)
	split = iter.SplitIntoColumnsOf(1, 0)
	assert.Equal(t, [][]int{{1, 2}}, split)

	// Die if n < 1
	{
		defer func() {
			assert.Equal(t, ErrRowsGreaterThanZero, recover())
		}()

		iter.SplitIntoColumnsOf(0, 0)
		assert.Fail(t, "Must panic")
	}

	// Die if value is nil
	{
		defer func() {
			assert.Equal(t, ErrValueCannotBeNil, recover())
		}()

		iter.SplitIntoColumnsOf(1, nil)
		assert.Fail(t, "Must panic")
	}
}

func TestToSlice(t *testing.T) {
	assert.Equal(t, []interface{}{}, Of().ToSlice())
	assert.Equal(t, []interface{}{1}, Of(1).ToSlice())
	assert.Equal(t, []interface{}{1, 2}, Of(1, 2).ToSlice())

	iter := Of()
	iter.ToSlice()
	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}
}

func TestToSliceOf(t *testing.T) {
	assert.Equal(t, []int{}, Of().ToSliceOf(0))
	assert.Equal(t, []int{1}, Of(1).ToSliceOf(0))
	assert.Equal(t, []int{1, 2}, Of(1, 2).ToSliceOf(0))

	iter := Of()
	iter.ToSliceOf(0)
	{
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}
}

func TestForLoop(t *testing.T) {
	{
		var (
			iter     = Of(5, []int{6, 7})
			idx      = 0
			expected = []interface{}{5, []int{6, 7}}
		)

		for iter.Next() {
			assert.Equal(t, expected[idx], iter.Value())
			idx++
		}

		assert.Equal(t, 2, idx)

		{
			defer func() {
				assert.Equal(t, ErrValueExhaustedIter, recover())
			}()

			iter.Value()
			assert.Fail(t, "Must panic")
		}
	}

	{
		var (
			iter     = OfElements([]int{6, 7})
			idx      = 0
			expected = []int{6, 7}
		)

		for iter.Next() {
			assert.Equal(t, expected[idx], iter.Value())
			idx++
		}

		assert.Equal(t, 2, idx)

		{
			defer func() {
				assert.Equal(t, ErrValueExhaustedIter, recover())
			}()

			iter.Value()
			assert.Fail(t, "Must panic")
		}
	}
}
