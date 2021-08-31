// SPDX-License-Identifier: Apache-2.0

package iter

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArraySliceIterFunc(t *testing.T) {
	// Empty array
	iterFunc := ArraySliceIterFunc(reflect.ValueOf([0]int{}))

	_, next := iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element array
	iterFunc = ArraySliceIterFunc(reflect.ValueOf([1]int{1}))

	val, next := iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Two element array
	iterFunc = ArraySliceIterFunc(reflect.ValueOf([2]int{1, 2}))

	val, next = iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	val, next = iterFunc()
	assert.Equal(t, 2, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Empty slice
	iterFunc = ArraySliceIterFunc(reflect.ValueOf([]int{}))

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element slice
	iterFunc = ArraySliceIterFunc(reflect.ValueOf([]int{1}))

	val, next = iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Two element slice
	iterFunc = ArraySliceIterFunc(reflect.ValueOf([]int{1, 2}))

	val, next = iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	val, next = iterFunc()
	assert.Equal(t, 2, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	iterFunc = ArraySliceIterFunc(reflect.ValueOf([]interface{}{3, 4}))

	val, next = iterFunc()
	assert.Equal(t, 3, val)
	assert.True(t, next)

	val, next = iterFunc()
	assert.Equal(t, 4, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Non-array/slice
	{
		defer func() {
			assert.Equal(t, ErrArraySliceIterFuncArg, recover())
		}()

		ArraySliceIterFunc(reflect.ValueOf(1))

		assert.Fail(t, "Must panic on non-array/slice")
	}
}

func TestMapIterFunc(t *testing.T) {
	// Empty map
	iterFunc := MapIterFunc(reflect.ValueOf(map[int]int{}))

	_, next := iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element map
	iterFunc = MapIterFunc(reflect.ValueOf(map[int]int{1: 2}))

	val, next := iterFunc()
	assert.Equal(t, KeyValue{Key: 1, Value: 2}, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Two element map
	expected := map[int]int{1: 2, 3: 4}
	iterFunc = MapIterFunc(reflect.ValueOf(expected))
	m := map[int]int{}

	val, next = iterFunc()
	kv := val.(KeyValue)
	m[kv.Key.(int)] = kv.Value.(int)
	assert.True(t, next)

	val, next = iterFunc()
	kv = val.(KeyValue)
	m[kv.Key.(int)] = kv.Value.(int)
	assert.True(t, next)

	assert.Equal(t, expected, m)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// Non-map
	{
		defer func() {
			assert.Equal(t, ErrMapIterFuncArg, recover())
		}()

		MapIterFunc(reflect.ValueOf(1))

		assert.Fail(t, "Must panic on non-map")
	}
}

func TestNoValueIterFunc(t *testing.T) {
	iterFunc := NoValueIterFunc

	_, next := iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)
}

func TestSingleValueIterFunc(t *testing.T) {
	// One element
	iterFunc := SingleValueIterFunc(reflect.ValueOf(5))

	val, next := iterFunc()
	assert.Equal(t, 5, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)
}

func TestElementsIterFunc(t *testing.T) {
	// ==== Array

	// Empty
	iterFunc := ElementsIterFunc(reflect.ValueOf([0]int{}))

	_, next := iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element
	iterFunc = ElementsIterFunc(reflect.ValueOf([1]int{1}))

	val, next := iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// ==== Slice

	// Empty
	iterFunc = ElementsIterFunc(reflect.ValueOf([]int{}))

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element
	iterFunc = ElementsIterFunc(reflect.ValueOf([]int{1}))

	val, next = iterFunc()
	assert.Equal(t, 1, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// ==== Map

	// Empty
	iterFunc = ElementsIterFunc(reflect.ValueOf(map[int]int{}))

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// One element
	iterFunc = ElementsIterFunc(reflect.ValueOf(map[int]int{1: 2}))

	val, next = iterFunc()
	assert.Equal(t, KeyValue{Key: 1, Value: 2}, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// ==== Nil ptr

	iterFunc = ElementsIterFunc(reflect.ValueOf((*int)(nil)))

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	// ==== Single value

	iterFunc = ElementsIterFunc(reflect.ValueOf(5))

	val, next = iterFunc()
	assert.Equal(t, 5, val)
	assert.True(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)
}

func TestReaderIterFuncAndOfReader(t *testing.T) {
	var (
		str      = "t2"
		iterFunc = ReaderIterFunc(strings.NewReader(str))
		iter     = OfReader(strings.NewReader(str))
		raw      = []byte(str)
		val      interface{}
		next     bool
	)

	for _, abyte := range raw {
		val, next = iterFunc()
		assert.Equal(t, abyte, val)
		assert.True(t, next)

		assert.Equal(t, abyte, iter.NextValue())
	}

	_, next = iterFunc()
	assert.False(t, next)

	_, next = iterFunc()
	assert.False(t, next)

	assert.False(t, iter.Next())
}

func TestReaderToRunesIterFuncAndOfReaderRunes(t *testing.T) {
	inputs := []string{
		"",
		// 1 byte UTF8
		"a",
		"ab",
		"abc",
		"abcd",
		"abcde",
		"abcdef",
		"abcdefg",
		"abcdefgh",
		"abcdefghi",
		// 2 byte UTF8
		"à",
		"àà",
		"ààa",
		"ààaa",
		// 3 byte UTF8
		"ḁ",
		"ḁḁ",
		"ḁḁḁ",
		"ḁḁḁḁ",
		// 4 bytes UTF8
		"𝆑",
		"𝆑𝆑",
		"𝆑𝆑𝆑",
		"𝆑𝆑𝆑𝆑",
	}

	for _, input := range inputs {
		var (
			iterFunc = ReaderToRunesIterFunc(strings.NewReader(input))
			iter     = OfReaderRunes(strings.NewReader(input))
			val      interface{}
			next     bool
		)

		for _, char := range []rune(input) {
			val, next = iterFunc()
			assert.Equal(t, char, val)
			assert.True(t, next)

			assert.Equal(t, char, iter.NextValue())
		}

		val, next = iterFunc()
		assert.Equal(t, 0, val)
		assert.False(t, next)

		val, next = iterFunc()
		assert.Equal(t, 0, val)
		assert.False(t, next)

		assert.False(t, iter.Next())
	}
}

func TestReaderToLinesIterFuncAndOfReaderLines(t *testing.T) {
	var (
		inputs = []string{
			"",
			"oneline",
			"two\rline cr",
			"two\nline lf",
			"two\r\nline crlf",
		}
		linesRegex, _ = regexp.Compile("\r\n|\r|\n")
	)

	for _, input := range inputs {
		var (
			iterFunc = ReaderToLinesIterFunc(strings.NewReader(input))
			iter     = OfReaderLines(strings.NewReader(input))
			lines    = linesRegex.Split(input, -1)
			val      interface{}
			next     bool
		)

		for _, line := range lines {
			val, next = iterFunc()
			assert.Equal(t, line, val)
			assert.Equal(t, input != "", next)

			if input == "" {
				assert.False(t, iter.Next())
			} else {
				assert.Equal(t, line, iter.NextValue())
			}
		}

		val, next = iterFunc()
		assert.Equal(t, "", val)
		assert.False(t, next)

		val, next = iterFunc()
		assert.Equal(t, "", val)
		assert.False(t, next)

		if input != "" {
			assert.False(t, iter.Next())
		}
	}
}

func TestFlattenArraySlice(t *testing.T) {
	f := FlattenArraySlice([2]int{1, 2})
	assert.Equal(t, []interface{}{1, 2}, f)

	f = FlattenArraySlice([]int{1, 3, 4})
	assert.Equal(t, []interface{}{1, 3, 4}, f)

	f = FlattenArraySlice([][]int{{1, 2}, {3, 4, 5}})
	assert.Equal(t, []interface{}{1, 2, 3, 4, 5}, f)

	f = FlattenArraySlice([]interface{}{1, [2]int{2, 3}, [][]string{{"4", "5"}, {"6", "7", "8"}}})
	assert.Equal(t, []interface{}{1, 2, 3, "4", "5", "6", "7", "8"}, f)
}

func TestFlattenArraySliceAsType(t *testing.T) {
	f := FlattenArraySliceAsType([2]int{1, 2}, 0)
	assert.Equal(t, []int{1, 2}, f)

	f = FlattenArraySliceAsType([]int{1, 3, 4}, 0)
	assert.Equal(t, []int{1, 3, 4}, f)

	f = FlattenArraySliceAsType([][]int{{1, 2}, {3, 4, 5}}, 0)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, f)

	f = FlattenArraySliceAsType([]interface{}{1, [2]int{2, 3}, [][]uint{{4, 5}, {6, 7, 8}}}, 0)
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8}, f)
}
