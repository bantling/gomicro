// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"io"
	"reflect"
	"sort"
	"sync"

	"github.com/bantling/gomicro/iter"
	"github.com/bantling/gomicro/optional"
)

// composeTransforms composes two func(*Iter) *Iter f1, f2 and returns a composition func(x *Iter) *Iter of f2(f1(x)).
// If f1 is nil, the composition degenerates to f2(x).
// Panics if f2 is nil.
func composeTransforms(f1, f2 func(*iter.Iter) *iter.Iter) func(*iter.Iter) *iter.Iter {
	if f2 == nil {
		panic("composeTransforms: f2 cannot be nil")
	}

	composition := f2
	if f1 != nil {
		composition = func(it *iter.Iter) *iter.Iter {
			return f2(f1(it))
		}
	}

	return composition
}

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

// IterateFunc adapts any func that accepts and returns the exact same type into a func(interface{}) interface{} suitable for the Iterate function.
// Panics if f is not a func that accepts and returns one type that are exactly the same.
// If f happens to already be a func(interface{}) interface{}, it is returned as is.
func IterateFunc(f interface{}) func(interface{}) interface{} {
	// If the types happen to already be a func(interface{}) interface{}, return f as is
	if iterFunc, isa := f.(func(interface{}) interface{}); isa {
		return iterFunc
	}

	var (
		val = reflect.ValueOf(f)
		typ = val.Type()
	)

	if typ.Kind() != reflect.Func {
		panic("f must be a function")
	}

	if (typ.NumIn() != 1) || (typ.NumOut() != 1) {
		panic("f must be a function that accepts and returns a single value of the exact same type")
	}

	argType, retType := typ.In(0), typ.Out(0)
	if argType != retType {
		panic("f must be a function that accepts and returns a single value of the exact same type")
	}

	return func(arg interface{}) interface{} {
		return val.Call([]reflect.Value{reflect.ValueOf(arg)})[0].Interface()
	}
}

// Iterate takes an initial seed value and an iterative func that is applied to the seed to generate a series of values.
// The result is an infinite series of seed, f(seed), f(f(seed)), ...
// The infinite series is represented as an iterator.
func Iterate(seed interface{}, f func(interface{}) interface{}) *iter.Iter {
	nextValue := seed

	return iter.NewIter(
		func() (interface{}, bool) {
			retValue := nextValue
			nextValue = f(nextValue)
			return retValue, true
		},
	)
}

// Stream is based on a composed transform, and provides a streaming facility where items can be transformed one by one as they are iterated into a new set, and possibly apply further transforms on the new set.
// A Stream is effectively a kind of builder pattern, building up a set of transforms from an input data set to an output data set.
//
// The idea is to compose a set of transforms, then call a terminal method that will invoke the composed transforms and produce a new result.
// All single element transforms are handled by Stream (eg, filter to retain elements > 5)
// All multi element transforms are handled by Finisher (eg, distinct elements only).
//
// The Stream.Transform method allows for arbitrary transforms, for cases where the transforms provided are not sufficient.
// When calling the transform methods, the transforms are composed using function composition so that there is only one transform function in the Stream.
// Each transform is a function that accepts a *iter.Iter and returns a new *iter.Iter.
//
// The Finisher works the same way, the only difference from Stream is that Finisher transforms may track state
// information across elements (eg, distinct requires tracking all unique elements that have occurred in past reads).
//
// This distinction is important for parallel processing, which works as follows:
// - The composed Stream transforms are applied in parallel, where each thread operates on a separate subset of the data set.
// - The transformed subsets are flattened into a single dataset.
// - The composed Finisher transforms are applied serially to the flattened data set into a new data set.
//
// As an example, suppose the following sequence is executed:
//
// New().
//   Filter(FilterFunc(func(i int) bool { return i < 5 })).
//   Map(MapFunc(func(i int) int { return i * 2 })).
//   AndThen().
//   Distinct().
//   Sort(funcs.IntSortFunc).
//   ToSliceOf(giter.Of(1,3,1,2,9,7,2,4,7,5,8,6,8), 0)
//
// The order of operations is exactly as indicated - filter then map each element one by one into a new set, finally remove duplicates, sort the set, and collect the result into a slice of int.
// The result will be []int{2,4,6,8}.
//
// Since the data to be processed is passed to Finisher methods that accept a iter.*Iter, the stream can be reused any number of times with different data sets.
// There are three general usage patterns for Stream:
// - One off: Create a Stream/Finisher for a given data set, get the results, and discard the Stream/Finisher (like the example above).
// - Reuse: Create a Stream/Finisher, keeping a reference to the Finisher.
//          As needed, call a terminal method of the Finisher with different data sets.
//          This allows a separation of constructing the set of transforms from applying the transforms to a particular data set.
// - Threaded Reuse: As above, but different threads can use the same Stream/Finisher to process different data sets in parallel.
//                   An example would be an http service where multiple users can simultaneously access the same endpoint to process parallel queries.
//
// The zero value is ready to use.
type Stream struct {
	transform func(*iter.Iter) *iter.Iter
}

// New constructs a new Stream
func New() *Stream {
	return &Stream{}
}

// === Transforms

// Transform composes the current transform with a new one
func (s *Stream) Transform(t func(*iter.Iter) *iter.Iter) *Stream {
	s.transform = composeTransforms(s.transform, t)
	return s
}

// Filter returns a new stream of all elements that pass the given predicate
func (s *Stream) Filter(f func(element interface{}) bool) *Stream {
	return s.Transform(
		func(it *iter.Iter) *iter.Iter {
			return iter.NewIter(
				func() (interface{}, bool) {
					for it.Next() {
						if val := it.Value(); f(val) {
							return val, true
						}
					}

					return nil, false
				},
			)
		},
	)
}

// FilterNot returns a new stream of all elements that do not pass the given predicate
func (s *Stream) FilterNot(f func(element interface{}) bool) *Stream {
	return s.Filter(
		func(element interface{}) bool {
			return !f(element)
		},
	)
}

// Map maps each element to a new element, possibly of a different type
func (s *Stream) Map(f func(element interface{}) interface{}) *Stream {
	return s.Transform(
		func(it *iter.Iter) *iter.Iter {
			return iter.NewIter(
				func() (interface{}, bool) {
					if it.Next() {
						return f(it.Value()), true
					}

					return nil, false
				},
			)
		},
	)
}

// Peek returns a stream that calls a function that examines each value and performs an additional operation
func (s *Stream) Peek(f func(interface{})) *Stream {
	return s.Transform(
		func(it *iter.Iter) *iter.Iter {
			return iter.NewIter(
				func() (interface{}, bool) {
					if it.Next() {
						val := it.Value()
						f(val)
						return val, true
					}

					return nil, false
				},
			)
		},
	)
}

//
// ==== Terminals
//

// Iter returns an iterator of the elements in this Stream.
func (s Stream) Iter(source *iter.Iter) *iter.Iter {
	it := source
	if s.transform != nil {
		it = s.transform(it)
	}

	return it
}

//
// ==== Continuation
//

// AndThen returns a Finisher, which performs additional post processing on the results of the transforms in this Stream.
func (s *Stream) AndThen() *Finisher {
	return &Finisher{
		stream:    s,
		generator: nil,
	}
}

// ==== Finisher

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

// Finisher does two things:
// 1. Apply zero or more transforms that operate across multiple elements after any Stream transforms have been applied to each individual element of the Stream source
// 2. Provide terminal methods that return the final result of applying the Stream and Finisher trasforms to the Stream source
//
// The purpose of separating Finisher from Stream is twofold:
// 1. Make the chaining method calls accurately represent that all multi-element transforms are applied after all single element tranforms.
// 2. Simplify paralell execution of transforms by breaking it into two phases:
//    a. Execute single element transforms on the Stream source in parallel
//    b. Execute multi element transforms on the result of the parallel execution
//
// Guaranteeing the mutli element transforms occur after parallel execution of single element transforms greatly simplifies the parallel algorithm:
// - Only one parallel algorithm is needed
// - No need for multiple passes or buffering
//
// The Finisher transform is actually composed of generators, functions that return a transform.
// This allows the generated multi-element transform to generate new initial state every time a terminal method is called,
// so that the same Finisher instance can be reused any number of times.
// If the caller maintains a reference to the Iterable provided to the Stream this Finisher came from,
// then the caller can change the data the Iterable provides, so that each call to Finisher terminal methods processes a new set of data.
type Finisher struct {
	stream    *Stream
	generator func() func(*iter.Iter) *iter.Iter
	finite    bool
}

//
// ==== Transforms
//

// Transform composes the current generator with a new one
func (fin *Finisher) Transform(g func() func(*iter.Iter) *iter.Iter) *Finisher {
	fin.generator = composeGenerators(fin.generator, g)
	return fin
}

// Distinct composes the current generator with a generator of distinct elements only.
// The order of the result is the first occurence of each distinct element.
// Elements must be a type compatible with a map key.
func (fin *Finisher) Distinct() *Finisher {
	return fin.Filter(
		func() func(element interface{}) bool {
			alreadyRead := map[interface{}]bool{}

			return func(element interface{}) bool {
				if !alreadyRead[element] {
					alreadyRead[element] = true
					return true
				}

				return false
			}
		},
	)
}

// Duplicate composes the current generator with a generator of duplicate elements only.
// The order of the result is the second occurence of each duplicate element.
// Elements must be a type compatible with a map key.
func (fin *Finisher) Duplicate() *Finisher {
	return fin.Filter(
		func() func(element interface{}) bool {
			alreadyRead := map[interface{}]bool{}

			return func(element interface{}) bool {
				if !alreadyRead[element] {
					alreadyRead[element] = true
					return false
				}

				return true
			}
		},
	)
}

// Filter composes the current generator with a filter of all elements that pass the given predicate generator
func (fin *Finisher) Filter(g func() func(element interface{}) bool) *Finisher {
	return fin.Transform(
		func() func(it *iter.Iter) *iter.Iter {
			f := g()

			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); f(val) {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		},
	)
}

// FilterNot composes the current generator with a filter of all elements that do not pass the given predicate generator
func (fin *Finisher) FilterNot(g func() func(element interface{}) bool) *Finisher {
	return fin.Transform(
		func() func(it *iter.Iter) *iter.Iter {
			f := g()

			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); !f(val) {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		},
	)
}

// Limit composes the current generator with a generator that only iterates the first n elements, ignoring the rest
func (fin *Finisher) Limit(n uint) *Finisher {
	fin.Transform(
		func() func(it *iter.Iter) *iter.Iter {
			var (
				elementsRead uint
			)

			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						if (elementsRead == n) || (!it.Next()) {
							return nil, false
						}

						elementsRead++
						return it.Value(), true
					},
				)
			}
		},
	)

	return fin
}

// ReverseSort composes the current generator with a generator that sorts the values by the provided comparator in reverse order.
// The provided function must compare elements in increasing order, same as for Sorted.
func (fin *Finisher) ReverseSort(less func(element1, element2 interface{}) bool) *Finisher {
	return fin.Sort(func(element1, element2 interface{}) bool {
		return !less(element1, element2)
	})
}

// SetReduce uses a generated function to reduce the set of input elements to a smaller set of output elements by
// iteratively executing a function with the current accumulated value and the next stream element.
//
// The generator is executed at the beginning of each reduction, so that each reduction begins with a
// consistent initial state, including the initial value.
//
// The generated function returns both the accumulated value so far and a boolean flag that is true to indicate more
// elements can be added to the current reduction. When the flag is false, it means the current reduction is complete,
// and the current accumulated value is added to the output stream.
//
// If there are no elements in the input stream, no reductions occur and no values are added to the output stream.
// Otherwise, the result is one or more reductions provided by f(f(element1), element2)...
//
// Note that there is a corner case to consider: it is possible that the final reduction runs out of input elements
// before the iterative function indicates the accumulated value is complete. The incomplete accumulated value will be
// the final output stream result.
//
// It is up to the caller to perform validation of each reduction, which may be accomplished by using AndThen to
// contine to a Stream that uses some combination of Filter, Map, and Peek to validate results.
func (fin *Finisher) SetReduce(
	generator func() func(*iter.Iter) (interface{}, bool),
) *Finisher {
	return fin.Transform(
		func() func(*iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						return generator()(it)
					},
				)
			}
		},
	)
}

// Skip composes the current generator with a generator that skips the first n elements
func (fin *Finisher) Skip(n int) *Finisher {
	return fin.Transform(
		func() func(it *iter.Iter) *iter.Iter {
			skipped := false

			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						// Skip n elements only once
						if !skipped {
							skipped = true

							for i := 1; i <= n; i++ {
								if it.Next() {
									it.Value()
									continue
								}

								// We don't have n elements to skip
								return nil, false
							}
						}

						if it.Next() {
							// Return next element
							return it.Value(), true
						}

						return nil, false
					},
				)
			}
		},
	)
}

// Sort composes the current generator with a generator that sorts the values by the provided comparator.
func (fin *Finisher) Sort(less func(element1, element2 interface{}) bool) *Finisher {
	return fin.Transform(
		func() func(it *iter.Iter) *iter.Iter {
			var sortedIter *iter.Iter
			done := false

			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						if !done {
							// Sort all stream elements
							sorted := it.ToSlice()
							sort.Slice(sorted, func(i, j int) bool {
								return less(sorted[i], sorted[j])
							})

							sortedIter = iter.OfElements(sorted)
							done = true
						}

						// Return next sorted element
						if sortedIter.Next() {
							return sortedIter.Value(), true
						}

						return nil, false
					},
				)
			}
		},
	)
}

//
// ==== Terminals
//

// Iter returns an iterator of the elements in the given source after applying the transforms in this Finisher.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before returning the Iter.
func (fin Finisher) Iter(source *iter.Iter, pc ...ParallelConfig) *iter.Iter {
	var it *iter.Iter

	if len(pc) > 0 {
		// Parallel execution
		pconf := pc[0]

		data := doParallel(
			source,
			fin.stream.transform,
			fin.generator,
			pconf.NumberOfItems,
			pconf.Flags,
		)

		it = iter.Of(data...)
	} else {
		// Serial execution
		it = source

		if fin.stream.transform != nil {
			it = fin.stream.transform(it)
		}

		if fin.generator != nil {
			it = fin.generator()(it)
		}
	}

	return it
}

// AllMatch is true if the predicate matches all elements with short-circuit logic.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before applying the predicate.
func (fin Finisher) AllMatch(f func(element interface{}) bool, source *iter.Iter, pc ...ParallelConfig) bool {
	allMatch := true
	for it := fin.Iter(source, pc...); it.Next(); {
		if allMatch = f(it.Value()); !allMatch {
			break
		}
	}

	return allMatch
}

// AnyMatch is true if the predicate matches any element with short-circuit logic.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before applying the predicate.
func (fin Finisher) AnyMatch(f func(element interface{}) bool, source *iter.Iter, pc ...ParallelConfig) bool {
	anyMatch := false
	for it := fin.Iter(source, pc...); it.Next(); {
		if anyMatch = f(it.Value()); anyMatch {
			break
		}
	}

	return anyMatch
}

// Average returns an optional average value.
// The slice elements must be convertible to a float64.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before the calculation.
func (fin Finisher) Average(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var (
		sum   float64
		count int
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		sum += it.Float64Value()
		count++
	}

	if count == 0 {
		return optional.Of()
	}

	avg := sum / float64(count)
	return optional.Of(avg)
}

// Count returns the count of all elements.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before counting.
func (fin Finisher) Count(source *iter.Iter, pc ...ParallelConfig) int {
	count := 0
	for it := fin.Iter(source, pc...); it.Next(); {
		it.Value()
		count++
	}

	return count
}

// First returns the optional first element of applying any tranforms to the stream source.
// Note that an empty Optional means either the first element is nil, or the stream is empty.
func (fin Finisher) First(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var val interface{}

	if it := fin.Iter(source, pc...); it.Next() {
		val = it.Value()
	}

	return optional.Of(val)
}

// ForEach invokes a consumer with each element of the stream.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before invoking the consumer.
func (fin Finisher) ForEach(f func(element interface{}), source *iter.Iter, pc ...ParallelConfig) {
	for it := fin.Iter(source, pc...); it.Next(); {
		f(it.Value())
	}
}

// GroupBy groups elements by executing the given function on each value to get a key,
// and appending the element to the end of a slice associated with the key in the resulting map.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before grouping.
func (fin Finisher) GroupBy(
	f func(element interface{}) (key interface{}),
	source *iter.Iter,
	pc ...ParallelConfig,
) map[interface{}][]interface{} {
	m := map[interface{}][]interface{}{}

	fin.Reduce(
		m,
		func(accumulator interface{}, element interface{}) interface{} {
			k := f(element)
			m[k] = append(m[k], element)
			return m
		},
		source,
		pc...,
	)

	return m
}

// Last returns the optional last element.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before finding the last element.
func (fin Finisher) Last(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var last interface{}
	for it := fin.Iter(source, pc...); it.Next(); {
		last = it.Value()
	}

	return optional.Of(last)
}

// Max returns an optional maximum value according to the provided comparator.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before finding the maximum.
func (fin Finisher) Max(less func(element1, element2 interface{}) bool, source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var max interface{}
	if it := fin.Iter(source, pc...); it.Next() {
		max = it.Value()

		for it.Next() {
			element := it.Value()

			if less(max, element) {
				max = element
			}
		}
	}

	return optional.Of(max)
}

// Min returns an optional minimum value according to the provided comparator.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before finding the minimum.
func (fin Finisher) Min(less func(element1, element2 interface{}) bool, source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var min interface{}
	if it := fin.Iter(source, pc...); it.Next() {
		min = it.Value()

		for it.Next() {
			element := it.Value()

			if less(element, min) {
				min = element
			}
		}
	}

	return optional.Of(min)
}

// NoneMatch is true if the predicate matches none of the elements with short-circuit logic.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before applying the predicate.
func (fin Finisher) NoneMatch(f func(element interface{}) bool, source *iter.Iter, pc ...ParallelConfig) bool {
	noneMatch := true
	for it := fin.Iter(source, pc...); it.Next(); {
		if noneMatch = !f(it.Value()); !noneMatch {
			break
		}
	}

	return noneMatch
}

// Reduce uses a function to reduce the stream to a single value by iteratively executing a function
// with the current accumulated value and the next stream element.
// The identity provided is the initial accumulated value, which means the result type is the
// same type as the initial value, which can be any type.
// If there are no elements in the stream, the result is the identity.
// Otherwise, the result is f(f(identity, element1), element2)...
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before reducing.
func (fin Finisher) Reduce(
	identity interface{},
	f func(accumulator interface{}, element interface{}) interface{},
	source *iter.Iter,
	pc ...ParallelConfig,
) interface{} {
	result := identity
	for it := fin.Iter(source, pc...); it.Next(); {
		result = f(result, it.Value())
	}

	return result
}

// Sum returns an optional sum value.
// The slice elements must be convertible to a float64.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before the calculation.
func (fin Finisher) Sum(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var (
		sum    float64
		hasSum bool
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		sum += it.Float64Value()
		hasSum = true
	}

	if !hasSum {
		return optional.Of()
	}

	return optional.Of(sum)
}

// SumAsInt returns an optional sum value.
// The slice elements must be convertible to an int.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before the calculation.
func (fin Finisher) SumAsInt(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var (
		sum    int
		hasSum bool
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		sum += it.IntValue()
		hasSum = true
	}

	if !hasSum {
		return optional.Of()
	}

	return optional.Of(sum)
}

// SumAsUint returns an optional sum value.
// The slice elements must be convertible to a uint.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before the calculation.
func (fin Finisher) SumAsUint(source *iter.Iter, pc ...ParallelConfig) optional.Optional {
	var (
		sum    uint
		hasSum bool
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		sum += it.UintValue()
		hasSum = true
	}

	if !hasSum {
		return optional.Of()
	}

	return optional.Of(sum)
}

// ToMap returns a map of all elements by invoking the given function to get a key/value pair for the map.
// It is up to the function to generate unique keys to prevent values from being overwritten.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before mapping.
func (fin Finisher) ToMap(
	f func(interface{}) (key interface{}, value interface{}),
	source *iter.Iter,
	pc ...ParallelConfig,
) map[interface{}]interface{} {
	m := map[interface{}]interface{}{}

	for it := fin.Iter(source, pc...); it.Next(); {
		k, v := f(it.Value())
		m[k] = v
	}

	return m
}

// ToMapOf returns a map of all elements, where the map key and value types are the same as the types of aKey and aValue.
// EG, if aKey is an int and aVaue is a string, then a map[int]string is returned.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before mapping.
// Panics if keys are not convertible to the key type or values are not convertible to the value type.
func (fin Finisher) ToMapOf(
	f func(interface{}) (key interface{}, value interface{}),
	aKey, aValue interface{},
	source *iter.Iter,
	pc ...ParallelConfig,
) interface{} {
	var (
		ktyp = reflect.TypeOf(aKey)
		vtyp = reflect.TypeOf(aValue)
		m    = reflect.MakeMap(reflect.MapOf(ktyp, vtyp))
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		k, v := f(it.Value())
		m.SetMapIndex(
			reflect.ValueOf(k).Convert(ktyp),
			reflect.ValueOf(v).Convert(vtyp),
		)
	}

	return m.Interface()
}

// ToSlice returns a slice of all elements.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before collecting.
func (fin Finisher) ToSlice(source *iter.Iter, pc ...ParallelConfig) []interface{} {
	array := []interface{}{}

	it := fin.Iter(source, pc...)
	for it.Next() {
		array = append(array, it.Value())
	}

	return array
}

// ToSliceOf returns a slice of all elements, where the slice elements are the same type as the type of elementVal.
// EG, if elementVal is an int, an []int is returned.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before collecting.
// Panics if elements are not convertible to the type of elementVal.
func (fin Finisher) ToSliceOf(elementVal interface{}, source *iter.Iter, pc ...ParallelConfig) interface{} {
	var (
		elementTyp = reflect.TypeOf(elementVal)
		array      = reflect.MakeSlice(reflect.SliceOf(elementTyp), 0, 0)
	)

	for it := fin.Iter(source, pc...); it.Next(); {
		array = reflect.Append(array, reflect.ValueOf(it.Value()).Convert(elementTyp))
	}

	return array.Interface()
}

const (
	toWriterBufSize int = 64 * 1024
)

// ToByteWriter writes the source to the Writer after applying any transformations.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before writing it.
// Panics if elements are not convertible to byte.
func (fin Finisher) ToByteWriter(w io.Writer, source *iter.Iter, pc ...ParallelConfig) (int, error) {
	var (
		buf        = make([]byte, toWriterBufSize)
		count      = 0
		totalCount = 0
	)

	writeOp := func() (int, error) {
		// Write buffer contents - could be a full buffer or remainder left at end
		n, err := w.Write(buf[0:count])

		// Track total number of bytes written so far - if an error occurs, n is probably < count
		totalCount += n

		// If an error occurred, return (totalCount, error)
		if err != nil {
			return totalCount, err
		}

		// Reset count in case there are further writes
		count = 0

		// Return success values
		return totalCount, nil
	}

	// Read transformed data as bytes to write
	for it := fin.Iter(source, pc...); it.Next(); {
		// Convert each element to a byte and write them one at a time
		buf[count] = it.ByteValue()
		count++

		// When the buffer is full, write it to the writer, then continue in case there is more data
		if count == toWriterBufSize {
			if n, err := writeOp(); err != nil {
				return n, err
			}
		}
	}

	// If iter ran out with a partially filled buffer, write the remainder and return (totalCount, nil)
	if count > 0 {
		return writeOp()
	}

	// If iter is an exact multiple of the buffer size, return (totalCount, nil)
	return totalCount, nil
}

// ToRuneWriter writes the source to the Writer after applying any transformations.
// If the optional ParallelConfig is provided, the transformed data set is collected via parallel execution before writing it.
// Panics if elements are not convertible to rune.
func (fin Finisher) ToRuneWriter(w io.Writer, source *iter.Iter, pc ...ParallelConfig) (int, error) {
	var (
		buf        = make([]byte, toWriterBufSize)
		count      = 0
		totalCount = 0
	)

	writeOp := func() (int, error) {
		// Write buffer contents - could be a full buffer or remainder left at end
		n, err := w.Write(buf[0:count])

		// Track total number of bytes written so far - if an error occurs, n is probably < count
		totalCount += n

		// If an error occurred, return (totalCount, error)
		if err != nil {
			return totalCount, err
		}

		// Reset count in case there are further writes
		count = 0

		// Return success values
		return totalCount, nil
	}

	// Read transformed data as runes to write
	for it := fin.Iter(source, pc...); it.Next(); {
		// Convert each rune element to one or more bytes and write them one at a time
		for _, runeByte := range []byte(string(it.RuneValue())) {
			buf[count] = runeByte
			count++

			// When the buffer is full, write it to the writer, then continue in case there is more data
			if count == toWriterBufSize {
				if n, err := writeOp(); err != nil {
					return n, err
				}
			}
		}
	}

	// If iter ran out with a partially filled buffer, write the remainder and return (totalCount, nil)
	if count > 0 {
		return writeOp()
	}

	// If iter is an exact multiple of the buffer size, return (totalCount, nil)
	return totalCount, nil
}

//
// ==== Continuation
//

// AndThen returns a stream such that when iterated, it will begin with all elements produced by ToSlice.
// If the optional ParallelConfig is provided, when the stream is iterated the given ParallelConfig is passed to ToSlice.
func (fin Finisher) AndThen(pc ...ParallelConfig) *Stream {
	return New().Transform(
		func(source *iter.Iter) *iter.Iter {
			return fin.Iter(source, pc...)
		},
	)
}
