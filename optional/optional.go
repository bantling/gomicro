// SPDX-License-Identifier: Apache-2.0

package optional

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/bantling/gomicro/funcs"
	iter "github.com/bantling/gomicro/iter"
)

// ZeroValueIsPresentFlags is a pair of flags indicating whether or not a zero value should be considered present
type ZeroValueIsPresentFlags bool

const (
	// ZeroValueIsPresent is the default, and indicates a zero value is considered present
	ZeroValueIsPresent ZeroValueIsPresentFlags = false
	// ZeroValueIsEmpty indicates a zero value is considered empty
	ZeroValueIsEmpty
)

// Optional is a mostly immutable generic wrapper for any kind of value with a present flag.
// The only mutable operation is the implementation of the sql.Scanner interface.
// The zero value is ready to use.
type Optional struct {
	value   interface{}
	present bool
}

var (
	errNotPresent = "No value present"
	emptyString   = "Optional"
)

// Of returns an Optional.
// If no value or a nil value is provided, a new empty Optional is returned.
// Otherwise a new Optional that wraps the value is returned.
func Of(value ...interface{}) Optional {
	v := funcs.IndexOf(value, 0)
	return funcs.Ternary(funcs.IsNil(v), Optional{}, Optional{value: v, present: true}).(Optional)
}

// Get returns the wrapped value and whether or not it is present.
// The wrapped value is only valid if the boolean is true.
func (o Optional) Get() (interface{}, bool) {
	return o.value, o.present
}

// MustGet returns the unwrapped value and panics if it is not present.
func (o Optional) MustGet() interface{} {
	return funcs.PanicVBM(o.value, o.present, errNotPresent)
}

// OrElse returns the wrapped value if it is present, else it returns the given value.
func (o Optional) OrElse(value interface{}) interface{} {
	return funcs.Ternary(o.present, o.value, value)
}

// OrElseGet returns the wrapped value if it is present, else it returns the result of the given function.
// supplier must be a func of no args that returns a single value to be wrapped.
func (o Optional) OrElseGet(supplier interface{}) interface{} {
	return funcs.TernaryOf(o.present, o.MustGet, supplier)
}

// OrElsePanic returns the wrapped value if it is present, else it panics with the result of the given function
func (o Optional) OrElsePanic(f func() string) interface{} {
	return funcs.PanicVBM(o.value, o.present, f())
}

// IsEmpty returns true if this Optional is not present
func (o Optional) IsEmpty() bool {
	return !o.present
}

// IsPresent returns true if this Optional is present
func (o Optional) IsPresent() bool {
	return o.present
}

// IfEmpty executes the function only if the value is not present.
func (o Optional) IfEmpty(f func()) {
	if !o.present {
		f()
	}
}

// IfPresent executes the consumer function with the wrapped value only if the value is present.
// consumer must be a func that receives a type the wrapped value can be converted into and has no return values.
func (o Optional) IfPresent(consumer interface{}) {
	if o.present {
		funcs.Consumer(consumer)(o.value)
	}
}

// IfPresentOrElse executes the consumer function with the wrapped value if the value is present, otherwise executes the function of no args.
// consumer must be a func that receives a type the wrapped value can be converted into and has no return values.
func (o Optional) IfPresentOrElse(consumer interface{}, f func()) {
	if o.present {
		funcs.Consumer(consumer)(o.value)
	} else {
		f()
	}
}

// Iter returns an *Iter of one element containing the wrapped value if present, else an empty Iter.
// See Iter for typed methods that return builtin types.
func (o Optional) Iter() *iter.Iter {
	return funcs.Ternary(o.present, iter.Of(o.value), iter.Of()).(*iter.Iter)
}

// Filter applies the predicate to the value of this Optional.
// Returns this Optional only if this Optional is present and the filter returns true for the value.
// Otherwise an empty Optional is returned.
// The predicate must be a func(any) bool, where the arg is compatible with the value of this Optional.
// Use funcs for predicate conjunctions, disjuctions, negations, etc.
func (o Optional) Filter(predicate interface{}) Optional {
	return funcs.Ternary(o.present && funcs.Filter(predicate)(o.value), o, Optional{}).(Optional)
}

// Map the wrapped value with the given mapping function, which may return a different type.
// An empty Optional is returned if any of the following is true:
// - This Optional is not present. In this case, the mapping function is not invoked.
// - The mapping function returns a nil value.
// - The mapping function returns a zero value, and zeroValIsPresent == ZeroValueIsEmpty. By default, zeroValIsPresent == ZeroValueIsPresent.
// Otherwise, an Optional wrapping the mapped value is returned.
// f must be a func that accepts one arg that the wrapped value can be converted into, and returns one value to wrap.
func (o Optional) Map(f interface{}, zeroValIsPresent ...ZeroValueIsPresentFlags) Optional {
	if !o.present {
		return Optional{}
	}

	v := funcs.Map(f)(o.value)
	if funcs.IsNil(v) {
		return Optional{}
	}

	if (len(zeroValIsPresent) > 0) && (zeroValIsPresent[0] == ZeroValueIsEmpty) && reflect.ValueOf(v).IsZero() {
		return Optional{}
	}

	return Of(v)
}

// FlatMap operates like Map, except that the mapping function already returns an Optional, which is returned as is.
func (o Optional) FlatMap(f interface{}) Optional {
	if !o.present {
		return Optional{}
	}

	return funcs.MapTo(f, Optional{}).(func(interface{}) Optional)(o.value)
}

// Scan is database/sql Scanner interface, allowing users to read null query columns into an Optional.
// This is the only method that modifies an Optional.
// The result will be same whether or not the Optional was initially empty.
// The provided value is just stored, so if it is a reference type it must be copied before the next call to Scan.
// Since any value can be stored, the result is always a nil error.
// It is up to the caller to ensure the correct type is being read.
func (o *Optional) Scan(src interface{}) error {
	o.value = src
	o.present = !funcs.IsNil(src)
	return nil
}

// Value is the database/sql/driver/Valuer interface, allowing users to write an Optional into a column.
// If a present optional does not contain an allowed type, the operation will fail.
// It is up to the caller to ensure the correct type is being written.
func (o Optional) Value() (driver.Value, error) {
	if o.present {
		return o.value, nil
	}

	return nil, nil
}

// String returns fmt.Sprintf("Optional (%v)", wrapped value) if present, else "Optional" if it is empty.
func (o Optional) String() string {
	return funcs.Ternary(o.present, fmt.Sprintf("Optional (%v)", o.value), emptyString).(string)
}
