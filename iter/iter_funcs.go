package iter

import (
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Error constants
const (
	ErrArraySliceIterFuncArg = "ArraySliceIterFunc argument must be an array or slice"
	ErrInvalidUTF8Encoding   = "Invalid UTF 8 encoding"
	ErrMapIterFuncArg        = "MapIterFunc argument must be a map"
)

// ArraySliceIterFunc iterates an array or slice outermost dimension.
// EG, if an [][]int is passed, the iterator returns []int values.
// Panics if the value is not an array or slice.
func ArraySliceIterFunc(arraySlice reflect.Value) func() (interface{}, bool) {
	if (arraySlice.Kind() != reflect.Array) && (arraySlice.Kind() != reflect.Slice) {
		panic(ErrArraySliceIterFuncArg)
	}

	var (
		num = arraySlice.Len()
		idx int
	)

	return func() (interface{}, bool) {
		if idx == num {
			// Exhausted all values - don't care how many calls are made once exhausted
			return nil, false
		}

		// Return value at current index, and increment index for next time
		val := arraySlice.Index(idx).Interface()
		idx++
		return val, true
	}
}

// KeyValue contains a key value pair from a map
type KeyValue struct {
	Key   interface{}
	Value interface{}
}

// MapIterFunc iterates a map
func MapIterFunc(aMap reflect.Value) func() (interface{}, bool) {
	if aMap.Kind() != reflect.Map {
		panic(ErrMapIterFuncArg)
	}

	var (
		mapIter = aMap.MapRange()
		done    bool
	)

	return func() (interface{}, bool) {
		// Return immediately if further calls are made after last key was iterated
		if done {
			return nil, false
		}

		// Advance MapIter to next key/value pair, if any
		if !mapIter.Next() {
			// Exhausted all values
			done = true
			return nil, false
		}

		// Return next key/value pair
		val := KeyValue{
			Key:   mapIter.Key().Interface(),
			Value: mapIter.Value().Interface(),
		}
		return val, true
	}
}

// NoValueIterFunc always returns (nil, false)
func NoValueIterFunc() (interface{}, bool) {
	return nil, false
}

// SingleValueIterFunc iterates a single value
func SingleValueIterFunc(aVal reflect.Value) func() (interface{}, bool) {
	done := false

	return func() (interface{}, bool) {
		if done {
			return nil, false
		}

		// First call returns the wrapped value given
		done = true
		return aVal.Interface(), true
	}
}

// ElementsIterFunc returns an iterator function that iterates the elements of the item passed non-recursively.
// The item is handled as follows:
// - Array or Slice: returns ArraySliceOuterIterFunc(item)
// - Map: returns MapIterFunc(item)
// - Nil ptr: returns NoValueIterFunc
// - Otherwise returns SingleValueIterFunc(item)
func ElementsIterFunc(item reflect.Value) func() (interface{}, bool) {
	switch item.Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		return ArraySliceIterFunc(item)
	case reflect.Map:
		return MapIterFunc(item)
	default:
		if (item.Kind() == reflect.Ptr) && item.IsNil() {
			return NoValueIterFunc
		}

		return SingleValueIterFunc(item)
	}
}

// ReaderIterFunc iterates the bytes of an io.Reader.
// For each byte in the Reader, returns (byte, true).
// When eof read, returns (0, false).
// When any other error occurs, panics with the error.
func ReaderIterFunc(src io.Reader) func() (interface{}, bool) {
	buf := make([]byte, 1)

	return func() (interface{}, bool) {
		if _, err := src.Read(buf); err != nil {
			if err != io.EOF {
				panic(err)
			}

			return 0, false
		}

		return buf[0], true
	}
}

// ReaderToRunesIterFunc iterates the bytes of an io.Reader, and interprets them as UTF-8 runes.
// For each valid rune contained in the Reader, returns (rune, true).
// When EOF read, returns (0, false).
// When any other error occurs (including invalid UTF-8 encoding), panics with the error.
func ReaderToRunesIterFunc(src io.Reader) func() (interface{}, bool) {
	// UTF-8 requires at most 4 bytes for a code point
	var (
		buf    = make([]byte, 4)
		bufPos int
	)

	return func() (interface{}, bool) {
		// Read next up to 4 bytes from reader into subslice of buffer, after any remaining bytes from last read
		if _, err := src.Read(buf[bufPos:]); (err != nil) && (err != io.EOF) {
			panic(err)
		}

		// If first byte is 0 after reading, must have emptied source and returned all runes
		if buf[0] == 0 {
			return 0, false
		}

		// Decode up to 4 bytes for next code point
		r, rl := utf8.DecodeRune(buf)
		if r == utf8.RuneError {
			panic(ErrInvalidUTF8Encoding)
		}

		// Shift any remaining unused bytes back to the begining of the buffer
		copy(buf, buf[rl:])

		// Next time read up to as many bytes as were shifted from source, overwriting remaining bytes
		bufPos = 4 - rl

		// Clear out the unused bytes at the end, in case we don't have enough bytes left to fill them
		copy(buf[bufPos:], zeroUTF8Buffer)

		return r, true
	}
}

// ReaderToLinesIterFunc iterates the bytes of an io.Reader, and interprets them as runes.
// Runes are read until an EOL sequence occurs (CR, LF, CRLF) or EOF occurs.
// For each line contained in the Reader, returns (string, true), where the string does not contain an EOL sequence.
// After the last line has been returned, all further calls return ("", false).
// When any other error occurs (including invalid UTF-8 encoding), panics with the error.
func ReaderToLinesIterFunc(src io.Reader) func() (interface{}, bool) {
	// Use ReaderToRunesIterFunc to read individual runes until a line is read
	var (
		runesIter = ReaderToRunesIterFunc(src)
		str       strings.Builder
		lastCR    bool
	)

	return func() (interface{}, bool) {
		str.Reset()

		for {
			codePoint, haveIt := runesIter()

			if !haveIt {
				if str.Len() > 0 {
					return str.String(), true
				}

				return "", false
			}

			if codePoint == '\r' {
				lastCR = true
				return str.String(), true
			}

			if codePoint == '\n' {
				if lastCR {
					lastCR = false
					continue
				}

				return str.String(), true
			}

			str.WriteRune(codePoint.(rune))
		}
	}
}

// FlattenArraySlice flattens an array or slice of any number of dimensions into a new slice of one dimension.
// EG, an [][]int{{1, 2}, {3, 4, 5}} is flattened into an []interface{}{1,2,3,4,5}.
// Note that in case where the element type is interface{}, a mixture of values and arrays/slices could be used.
// EG, an []interface{}{1, [2]int{2, 3}, [][]string{{"4", "5"}, {"6", "7", "8"}}} is flattened into []interface{}{1, 2, 3, "4", "5", "6", "7", "8"}.
// Panics if the value is not an array or slice.
func FlattenArraySlice(value interface{}) []interface{} {
	arraySlice := reflect.ValueOf(value)
	if (arraySlice.Kind() != reflect.Array) && (arraySlice.Kind() != reflect.Slice) {
		panic("FlattenArraySlice argument must be an array or slice")
	}

	// Make a one dimensional slice
	result := []interface{}{}

	// Recursive function
	var f func(reflect.Value)
	f = func(currentArraySlice reflect.Value) {
		// Iterate current array or slice
		for i, num := 0, currentArraySlice.Len(); i < num; i++ {
			val := reflect.ValueOf(currentArraySlice.Index(i).Interface())

			// Recurse sub-arrays/slices
			if (val.Kind() == reflect.Array) || (val.Kind() == reflect.Slice) {
				f(val)
			} else {
				result = append(result, val.Interface())
			}
		}
	}
	f(arraySlice)

	return result
}

// FlattenArraySliceAsType flattens an array or slice of any number of dimensions into a new slice of one dimension,
// where the slice type is the same as the given element.
// EG, an [][]int{{1, 2}, {3, 4, 5}} can be flattened into an []int{}{1,2,3,4,5}.
// Note that in case where the element type is interface{}, a mixture of values and arrays/slices could be used.
// EG, an []interface{}{1, [2]int{2, 3}, [][]uint{{4, 5}, {6, 7, 8}}} can be flattened into []int{}{1, 2, 3, 4, 5, 6, 7, 8}.
// Panics if the value is not an array or slice.
func FlattenArraySliceAsType(value interface{}, elementVal interface{}) interface{} {
	arraySlice := reflect.ValueOf(value)
	if (arraySlice.Kind() != reflect.Array) && (arraySlice.Kind() != reflect.Slice) {
		panic("FlattenArraySliceAs value must be an array or slice")
	}

	// Make a one dimensional slice that has the same type as the type of elementVal
	var (
		typ    = reflect.TypeOf(elementVal)
		result = reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
	)

	// Recursive function
	var f func(reflect.Value)
	f = func(currentArraySlice reflect.Value) {
		// Iterate current array or slice
		for i, num := 0, currentArraySlice.Len(); i < num; i++ {
			val := reflect.ValueOf(currentArraySlice.Index(i).Interface())

			// Recurse sub-arrays/slices
			if (val.Kind() == reflect.Array) || (val.Kind() == reflect.Slice) {
				f(val)
			} else {
				result = reflect.Append(result, val.Convert(typ))
			}
		}
	}
	f(arraySlice)

	return result.Interface()
}
