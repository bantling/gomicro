// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"reflect"

	"github.com/bantling/gomicro/iter"
)

// ==== Functions

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

	return iter.New(
		func() (interface{}, bool) {
			retValue := nextValue
			nextValue = f(nextValue)
			return retValue, true
		},
	)
}

// ==== Stream

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
			return iter.New(
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
			return iter.New(
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
			return iter.New(
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
