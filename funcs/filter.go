// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	"fmt"
	"reflect"
)

const (
	filterErrorMsg   = "fn must be a non-nil function of one argument of any type that returns bool"
	lessThanErrorMsg = "val must be a lessable type"
)

// Filter (fn) adapts a func(any) bool into a func(interface{}) bool.
// If fn happens to be a func(interface{}) bool, it is returned as is.
// Otherwise, each invocation converts the arg passed to the type the func receives.
func Filter(fn interface{}) func(interface{}) bool {
	// Return fn as is if it is desired type
	if res, isa := fn.(func(interface{}) bool); isa {
		return res
	}

	vfn := reflect.ValueOf(fn)
	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(filterErrorMsg)
	}

	typ := vfn.Type()
	if (typ.NumIn() != 1) ||
		(typ.NumOut() != 1) ||
		(typ.Out(0).Kind() != reflect.Bool) {
		panic(filterErrorMsg)
	}

	argTyp := typ.In(0)

	return func(arg interface{}) bool {
		var (
			argVal = reflect.ValueOf(arg).Convert(argTyp)
			resVal = vfn.Call([]reflect.Value{argVal})[0].Bool()
		)

		return resVal
	}
}

// FilterAll (fns) adapts any number of func(any) bool into a slice of func(interface{}) bool.
// Each func passed is separately adapted using Filter into the corresponding slice element of the result.
// FIlterAll is the basis for composing multiple logic functions into a single logic function.
// Note that when calling the provided set of logic functions, the argument type must be compatible with all of them.
// The most likely failure case is mixing funcs that accept interface{} that type assert the argument with funcs that accept a specific type.
func FilterAll(fns ...interface{}) []func(interface{}) bool {
	// Create adapters
	adaptedFns := make([]func(interface{}) bool, len(fns))
	for i, fn := range fns {
		adaptedFns[i] = Filter(fn)
	}

	return adaptedFns
}

// And (fns) any number of func(any)bool into the conjunction of all the funcs.
// Short-circuit logic will return false on the first function that returns false.
func And(fns ...interface{}) func(interface{}) bool {
	adaptedFns := FilterAll(fns...)

	return func(val interface{}) bool {
		for _, fn := range adaptedFns {
			if !fn(val) {
				return false
			}
		}

		return true
	}
}

// Or (fns) any number of func(any)bool into the disjunction of all the funcs.
// Short-circuit logic will return true on the first function that returns true.
func Or(fns ...interface{}) func(interface{}) bool {
	adaptedFns := FilterAll(fns...)

	return func(val interface{}) bool {
		for _, fn := range adaptedFns {
			if fn(val) {
				return true
			}
		}

		return false
	}
}

// Not (fn) adapts a func(any) bool to the negation of the func.
func Not(fn interface{}) func(interface{}) bool {
	adaptedFn := Filter(fn)

	return func(val interface{}) bool {
		return !adaptedFn(val)
	}
}

// EqualTo (val) returns a func(interface{}) bool that returns true if the func arg is equal to val.
// The arg is converted to the type of val first, then compared.
// If val is nil, then the arg type must be convertible to the type of val.
// If val is an untyped nil, then the arg must be an untyped nil.
// Comparison is made using == operator.
// If val is not comparable using == (eg, slices are not comparable), the result will be true if val and arg have the same address.
func EqualTo(val interface{}) func(interface{}) bool {
	var (
		valIsNil = IsNil(val)
		valTyp   = reflect.TypeOf(val)
	)

	return func(arg interface{}) bool {
		argTyp := reflect.TypeOf(arg)

		if valTyp == nil {
			// val is an untyped nil
			return argTyp == nil
		}

		// Remaining comparisons require arg to be convertible to val type
		if (argTyp == nil) || (!argTyp.ConvertibleTo(valTyp)) {
			return false
		}

		if valIsNil {
			// val is a typed nil, and arg is a convertible type
			return IsNil(arg)
		}

		if !valTyp.Comparable() {
			// val cannot be compared using ==
			return fmt.Sprintf("%p", val) == fmt.Sprintf("%p", arg)
		}

		// val is non-nil, and arg is a possibly nil value of a convertible type
		return (!IsNil(arg)) && (val == reflect.ValueOf(arg).Convert(valTyp).Interface())
	}
}

// DeepEqualTo (val) returns a func(interface{}) bool that returns true if the func arg is deep equal to val.
// The arg is converted to the type of val first, then compared.
// If val is nil, then the arg type must be convertible to the type of val.
// If val is an untyped nil, then the arg must be an untyped nil.
// Comparison is made using reflect.DeepEqual.
func DeepEqualTo(val interface{}) func(interface{}) bool {
	var (
		valIsNil = IsNil(val)
		valTyp   = reflect.TypeOf(val)
	)

	return func(arg interface{}) bool {
		argTyp := reflect.TypeOf(arg)

		if valTyp == nil {
			// val is an untyped nil
			return argTyp == nil
		}

		// Remaining comparisons require arg to be convertible to val type
		if (argTyp == nil) || (!argTyp.ConvertibleTo(valTyp)) {
			return false
		}

		if valIsNil {
			// val is a typed nil, and arg is a convertible type
			return IsNil(arg)
		}

		// val is non-nil, and arg is a possibly nil value of a convertible type
		return (!IsNil(arg)) && reflect.DeepEqual(val, reflect.ValueOf(arg).Convert(valTyp).Interface())
	}
}

// IsLessableKind returns if if kind represents any numeric type or string
func IsLessableKind(kind reflect.Kind) bool {
	return ((kind >= reflect.Int) && (kind <= reflect.Float64) ||
		(kind == reflect.String))
}

// LessThan (val) returns a func(val1, val2 interface{}) bool that returns true if val1 < val2.
// The args are converted to the type of val first, then compared.
// Panics if val is nil or IsLessableKind(kind of val) is false.
func LessThan(val interface{}) func(val1, val2 interface{}) bool {
	if IsNil(val) {
		panic(lessThanErrorMsg)
	}

	kind := reflect.ValueOf(val).Kind()
	if !IsLessableKind(kind) {
		panic(lessThanErrorMsg)
	}

	switch kind {
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		typ := reflect.TypeOf(int64(0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Int() < reflect.ValueOf(val2).Convert(typ).Int()
		}

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		typ := reflect.TypeOf(uint64(0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Uint() < reflect.ValueOf(val2).Convert(typ).Uint()
		}

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		typ := reflect.TypeOf(float64(0.0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Float() < reflect.ValueOf(val2).Convert(typ).Float()
		}

	// Must be string
	default:
		typ := reflect.TypeOf("")
		return func(val1, val2 interface{}) bool {
			return fmt.Sprintf("%s", reflect.ValueOf(val1).Convert(typ)) < fmt.Sprintf("%s", reflect.ValueOf(val2).Convert(typ))
		}
	}
}

// IsLessThan returns a func(arg interface{}) bool that returns true if arg < val
func IsLessThan(val interface{}) func(interface{}) bool {
	lt := LessThan(val)

	return func(arg interface{}) bool {
		return lt(arg, val)
	}
}

// LessThanEquals (val) returns a func(val1, val2 interface{}) bool that returns true if val1 <= val2.
// The args are converted to the type of val first, then compared.
// Panics if val is nil or IsLessableKind(kind of val) is false.
func LessThanEquals(val interface{}) func(val1, val2 interface{}) bool {
	if IsNil(val) {
		panic(lessThanErrorMsg)
	}

	kind := reflect.ValueOf(val).Kind()
	if !IsLessableKind(kind) {
		panic(lessThanErrorMsg)
	}

	switch kind {
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		typ := reflect.TypeOf(int64(0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Int() <= reflect.ValueOf(val2).Convert(typ).Int()
		}

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		typ := reflect.TypeOf(uint64(0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Uint() <= reflect.ValueOf(val2).Convert(typ).Uint()
		}

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		typ := reflect.TypeOf(float64(0.0))
		return func(val1, val2 interface{}) bool {
			return reflect.ValueOf(val1).Convert(typ).Float() <= reflect.ValueOf(val2).Convert(typ).Float()
		}

	// Must be string
	default:
		typ := reflect.TypeOf("")
		return func(val1, val2 interface{}) bool {
			return fmt.Sprintf("%s", reflect.ValueOf(val1).Convert(typ)) <= fmt.Sprintf("%s", reflect.ValueOf(val2).Convert(typ))
		}
	}
}

// IsLessThanEquals returns a func(arg interface{}) bool that returns true if arg <= val
func IsLessThanEquals(val interface{}) func(interface{}) bool {
	lte := LessThanEquals(val)

	return func(arg interface{}) bool {
		return lte(arg, val)
	}
}

// GreaterThan (val) returns a func(val1, val2 interface{}) bool that returns true if val1 > val2.
// The args are converted to the type of val first, then compared.
// Panics if val is nil or IsLessableKind(kind of val) is false.
func GreaterThan(val interface{}) func(val1, val2 interface{}) bool {
	lte := LessThanEquals(val)
	return func(val1, val2 interface{}) bool {
		return !lte(val1, val2)
	}
}

// IsGreaterThan returns a func(arg interface{}) bool that returns true if arg > val
func IsGreaterThan(val interface{}) func(interface{}) bool {
	gt := GreaterThan(val)

	return func(arg interface{}) bool {
		return gt(arg, val)
	}
}

// GreaterThanEquals (val) returns a func(val1, val2 interface{}) bool that returns true if val1 >= val2.
// The args are converted to the type of val first, then compared.
// Panics if val is nil or IsLessableKind(kind of val) is false.
func GreaterThanEquals(val interface{}) func(val1, val2 interface{}) bool {
	lt := LessThan(val)
	return func(val1, val2 interface{}) bool {
		return !lt(val1, val2)
	}
}

// IsGreaterThanEquals returns a func(arg interface{}) bool that returns true if arg >= val
func IsGreaterThanEquals(val interface{}) func(interface{}) bool {
	gte := GreaterThanEquals(val)

	return func(arg interface{}) bool {
		return gte(arg, val)
	}
}

// IsNegative (val) returns true if the val < 0
func IsNegative(val interface{}) bool {
	return LessThan(val)(val, 0)
}

// IsNonNegative (val) returns true if val >= 0
func IsNonNegative(val interface{}) bool {
	return GreaterThanEquals(val)(val, 0)
}

// IsPositive (val) returns true if val > 0
func IsPositive(val interface{}) bool {
	return GreaterThan(val)(val, 0)
}

// IsNil is a func(interface{}) bool that returns true if val is nil
func IsNil(val interface{}) bool {
	if IsNilable(val) {
		rv := reflect.ValueOf(val)
		return (!rv.IsValid()) || rv.IsNil()
	}

	return false
}

// IsNilable is a func(interface{}) bool that returns true if val is nil or the type of val is a nilable type.
// Returns true of the reflect.Kind of val is Chan, Func, Interface, Map, Ptr, or Slice.
func IsNilable(val interface{}) bool {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() {
		return true
	}

	k := rv.Type().Kind()
	return (k >= reflect.Chan) && (k <= reflect.Slice)
}
