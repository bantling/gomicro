// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"sync"

	"github.com/bantling/gomicro/iter"
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

// ==== SetReduce

// UnmarshalJSON is a SetReduce function to unmarshals a JSON array or object into a []interface{} or a recursive map[interface{}]interface{}, respectively.
// If the document is an array of objects, then []interface{} contains resursive map[interface{}]interface{} elements.
// The input may contain multiple arrays and/or objects, each call will read a single array or object.
func UnmarshalJSON() func(*iter.Iter) (interface{}, bool) {
	return nil
}
