// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"

	"github.com/bantling/gomicro/iter"
)

// Error constants
const (
	ErrInvalidJSONDocument = "The elements are not a valid JSON array or object"
	ErrNotAnArrayOrSlice   = "The elements must be arrays or slices"
)

// ==== Compose

// composeGenerators composes two func() func(*Iter) *Iter f1, f2 and returns a composition func() func(x *Iter) *Iter that returns f2()(f1()(x)).
// If f1 is nil, the composition degenerates to f2.
// Panics if f2 is nil.
func composeGenerators(f1, f2 func() func(*iter.Iter) *iter.Iter) func() func(*iter.Iter) *iter.Iter {
	if f2 == nil {
		panic("composeGenerators: f2 cannot be nil")
	}

	composition := f2
	if f1 != nil {
		composition = func() func(*iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter { return f2()(f1()(it)) }
		}
	}

	return composition
}

// ==== Parallel

// ParallelFlags is a pair of flags indicating whether to interpret the number as the number of goroutines or the number of items each goroutine processes
type ParallelFlags uint

const (
	// NumberOfGoroutines is the default, and indicates the number of goroutines
	NumberOfGoroutines ParallelFlags = iota
	// NumberOfItemsPerGoroutine indicates the number of items each goroutine processes
	NumberOfItemsPerGoroutine
)

const (
	// DefaultNumberOfParallelItems is the default number of items when executing transforms in parallel
	DefaultNumberOfParallelItems uint = 50
)

// ParallelConfig contains a configuration for parallel execution.
// NumberOfItems defaults to DefaultNumberOfParallelItems.
// Flags defaults to NumberOfGoroutines.
// The zero value is ready to use.
type ParallelConfig struct {
	NumberOfItems uint
	Flags         ParallelFlags
}

// doParallel does the grunt work of parallel processing, returning a slice of results.
// If numItems is 0, the default value is DefaultNumberOfParallelItems.
func doParallel(
	source *iter.Iter,
	transform func(*iter.Iter) *iter.Iter,
	generator func() func(*iter.Iter) *iter.Iter,
	numItems uint,
	flag ParallelFlags,
) []interface{} {
	n := DefaultNumberOfParallelItems
	if numItems > 0 {
		n = numItems
	}

	var flatData []interface{}
	if transform == nil {
		// If the transform is nil, there is no transform, just use source values as is
		flatData = source.ToSlice()
	} else {
		var splitData [][]interface{}
		if flag == NumberOfGoroutines {
			// numItems = desired number of rows; number of colums to be determined
			splitData = source.SplitIntoColumns(n)
		} else {
			// numItems = desired number of columns; number of rows to be determined
			splitData = source.SplitIntoRows(n)
		}

		// Execute goroutines, one per row of splitData.
		// Each goroutine applies the queued operations to each item in its row.
		wg := &sync.WaitGroup{}

		for i, row := range splitData {
			wg.Add(1)

			go func(i int, row []interface{}) {
				defer wg.Done()

				splitData[i] = transform(iter.OfElements(row)).ToSlice()
			}(i, row)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Combine rows into a single flat slice
		flatData = iter.FlattenArraySlice(splitData)
	}

	// If the generator is non-nil, apply it afterwards - it cannot be done in parallel
	if generator != nil {
		flatData = generator()(iter.Of(flatData...)).ToSlice()
	}

	// Return transformed rows
	return flatData
}

// ==== Transform

// ToJSON is a SetMap function that maps each JSON array or object from the source bytes into a
// []interface{} or map[string]interface{}, respectively.
// The input may have multiple arrays and/or objects, where each one is a single element in the output.
//
// Panics if the elements are not bytes.
// Panics if the elements do not contain a valid JSON array or object.
func ToJSON() func(*iter.Iter) *iter.Iter {
	return func(it *iter.Iter) *iter.Iter {
		return iter.NewIter(func() (interface{}, bool) {
			if !it.Next() {
				return nil, false
			}

			// The json.Decoder documentation says it may read more bytes then necessary from the reader.
			// In practice, it seems to exhaust the entire reader and only decode the first array or object.
			// So we read the input into a buffer of only one array or object, tracking brackets and braces.
			var (
				stack []byte
				buf   []byte
				ch    byte
			)

			// Must start with [ or {
			if ch = it.Value().(byte); !((ch == '[') || (ch == '{')) {
				panic(ErrInvalidJSONDocument)
			}
			stack = append(stack, ch)
			buf = append(buf, ch)

			for it.Next() {
				ch = it.Value().(byte)
				buf = append(buf, ch)

				// Stack up [ and {
				if (ch == '[') || (ch == '{') {
					stack = append(stack, ch)
				} else if (ch == ']') || (ch == '}') {
					// Match ] and } with last element in stack
					if ch == ']' {
						if stack[len(stack)-1] != '[' {
							panic(ErrInvalidJSONDocument)
						}
					} else if stack[len(stack)-1] != '{' {
						panic(ErrInvalidJSONDocument)
					}

					if stack = stack[0 : len(stack)-1]; len(stack) == 0 {
						break
					}
				}
			}

			// The stack must be empty, or the doc ended prematurely
			if len(stack) > 0 {
				panic(ErrInvalidJSONDocument)
			}

			// Use json.Unmarshal to unmarshal the array or object
			var (
				doc     interface{}
				decoder = json.NewDecoder(bytes.NewBuffer(buf))
			)
			decoder.UseNumber()

			if err := decoder.Decode(&doc); err != nil {
				panic(err)
			}

			return doc, true
		})
	}
}

// FromArraySlice is a SetMap function that maps each source array or slice into their elements.
// Panics if the elements are npot arrays or slices.
func FromArraySlice() func(*iter.Iter) *iter.Iter {
	return func(it *iter.Iter) *iter.Iter {
		var (
			arraySlice reflect.Value
			n          = 0
			sz         = 0
		)

		return iter.NewIter(func() (interface{}, bool) {
			// Search for next non-empty array or slice, if we need to
			for (n == sz) && it.Next() {
				arraySlice = reflect.ValueOf(it.Value())
				if kind := arraySlice.Kind(); !((kind == reflect.Array) || (kind == reflect.Slice)) {
					panic(ErrNotAnArrayOrSlice)
				}

				n = 0
				sz = arraySlice.Len()
			}

			if n == sz {
				return nil, false
			}

			value := arraySlice.Index(n).Interface()
			n++
			return value, true
		})
	}
}
