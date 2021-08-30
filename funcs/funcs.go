// SPDX-License-Identifier: Apache-2.0

package gofuncs

import (
	"fmt"
	"math/big"
	"math/cmplx"
	"reflect"
)

const (
	indexOfErrorMsg    = "slc must be a slice"
	valueOfKeyErrorMsg = "mp must be a map"
	filterErrorMsg     = "fn must be a non-nil function of one argument of any type that returns bool"
	lessThanErrorMsg   = "val must be a lessable type"
	mapErrorMsg        = "fn must be a non-nil function of one argument of any type that returns one value of any type"
	mapToErrorMsg      = "fn must be a non-nil function of one argument of any type that returns one value convertible to type %s"
	supplierErrorMsg   = "fn must be a non-nil function of no arguments or a single variadic argument that returns one value of any type"
	supplierOfErrorMsg = "fn must be a non-nil function of no arguments or a single variadic argument that returns one value convertible to type %s"
	consumerErrorMsg   = "fn must be a non-nil funciton of one argument of any type and no return values"
	sortErrorMsg       = "fn must be a non-nil function of two arguments of the same type and return bool"
)

// IndexOf returns the first of the following given an array or slice, index, and optional default value:
// 1. slice[index] if the array or slice length > index
// 2. default value if provided, converted to array or slice element type
// 3. zero value of array or slice element type
// Panics if arrslc is not an array or slice.
// Panics if the default value is not convertible to the array or slice element type, even if it is not needed.
func IndexOf(arrslc interface{}, index uint, defalt ...interface{}) interface{} {
	rv := reflect.ValueOf(arrslc)
	switch rv.Kind() {
	case reflect.Array:
	case reflect.Slice:
	default:
		panic(indexOfErrorMsg)
	}

	elementTyp := rv.Type().Elem()

	// Always ensure if default is provided that it is convertible to slice element type
	var rdf reflect.Value
	if len(defalt) > 0 {
		rdf = reflect.ValueOf(defalt[0]).Convert(elementTyp)
	}

	// Return index if it exists
	idx := int(index)
	if rv.Len() > idx {
		return rv.Index(idx).Interface()
	}

	// Else return default if provided
	if rdf.IsValid() {
		return rdf.Interface()
	}

	// Else return zero value of array or slice element type
	return reflect.Zero(elementTyp).Interface()
}

// ValueOfKey returns the first of the following:
// 1. map[key] if the key exists in the map
// 2. default if provided
// 3. zero value of map value type
// Panics if mp is not a map.
// Panics if the default value is not convertible to map value type, even if it is not needed.
func ValueOfKey(mp interface{}, key interface{}, defalt ...interface{}) interface{} {
	rv := reflect.ValueOf(mp)
	if rv.Kind() != reflect.Map {
		panic(valueOfKeyErrorMsg)
	}

	elementTyp := rv.Type().Elem()

	// Always ensure if default is provided that it is convertible to map value type
	var rdf reflect.Value
	if len(defalt) > 0 {
		rdf = reflect.ValueOf(defalt[0]).Convert(elementTyp)
	}

	// Return key value if it exists
	for mr := rv.MapRange(); mr.Next(); {
		if mr.Key().Interface() == key {
			return mr.Value().Interface()
		}
	}

	// Else return default if provided
	if rdf.IsValid() {
		return rdf.Interface()
	}

	// Else return zero value of map value type
	return reflect.Zero(elementTyp).Interface()
}

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

// Map (fn) adapts a func(any) any into a func(interface{}) interface{}.
// If fn happens to be a func(interface{}) interface{}, it is returned as is.
// Otherwise, each invocation converts the arg passed to the type the func receives.
func Map(fn interface{}) func(interface{}) interface{} {
	// Return fn as is if it is desired type
	if res, isa := fn.(func(interface{}) interface{}); isa {
		return res
	}

	vfn := reflect.ValueOf(fn)
	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(mapErrorMsg)
	}

	typ := vfn.Type()
	if (typ.NumIn() != 1) || (typ.NumOut() != 1) {
		panic(mapErrorMsg)
	}

	argTyp := typ.In(0)

	return func(arg interface{}) interface{} {
		var (
			argVal = reflect.ValueOf(arg).Convert(argTyp)
			resVal = vfn.Call([]reflect.Value{argVal})[0].Interface()
		)

		return resVal
	}
}

// MapTo (fn, X) adapts a func(any) X' into a func(interface{}) X.
// If fn happens to be a func(interface{}) X, it is returned as is.
// Otherwise, each invocation converts the arg passed to the type the func receives, and type X' must be convertible to X.
// The result will have to be type asserted by the caller.
func MapTo(fn interface{}, val interface{}) interface{} {
	// val cannot be nil
	if IsNil(val) {
		panic("val cannot be nil")
	}

	// Verify val is a non-interface type
	var (
		xval = reflect.ValueOf(val)
		xtyp = xval.Type()
	)
	if xval.Kind() == reflect.Interface {
		panic("val cannot be an interface{} value")
	}

	// Verify fn has is a non-nil func of 1 parameter and 1 result
	var (
		vfn    = reflect.ValueOf(fn)
		errMsg = fmt.Sprintf(mapToErrorMsg, xtyp)
	)

	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(errMsg)
	}

	// The func has to accept 1 arg and return 1 type
	typ := vfn.Type()
	if (typ.NumIn() != 1) || (typ.NumOut() != 1) {
		panic(errMsg)
	}

	var (
		argTyp = typ.In(0)
		resTyp = typ.Out(0)
	)

	// Return fn as is if it is desired type
	if (argTyp.Kind() == reflect.Interface) && (resTyp == xtyp) {
		return fn
	}

	// If fn returns any type convertible to X, then generate a function of interface{} to exactly X
	if !resTyp.ConvertibleTo(xtyp) {
		panic(errMsg)
	}

	return reflect.MakeFunc(
		reflect.FuncOf(
			[]reflect.Type{reflect.TypeOf((*interface{})(nil)).Elem()},
			[]reflect.Type{xtyp},
			false,
		),
		func(args []reflect.Value) []reflect.Value {
			var (
				argVal = reflect.ValueOf(args[0].Interface()).Convert(argTyp)
				resVal = vfn.Call([]reflect.Value{argVal})[0].Convert(xtyp)
			)

			return []reflect.Value{resVal}
		},
	).Interface()
}

// ConvertTo generates a func(interface{}) interface{} that converts a value into the same type as the value passed.
// Eg, ConvertTo(int8(0)) converts a func that converts a value into an int8.
func ConvertTo(out interface{}) func(interface{}) interface{} {
	outTyp := reflect.TypeOf(out)

	return func(in interface{}) interface{} {
		return reflect.ValueOf(in).Convert(outTyp).Interface()
	}
}

// Supplier (fn) adapts a func() any into a func() interface{}.
// If fn happens to be a func() interface{}, it is returned as is.
// fn may have a single variadic argument.
func Supplier(fn interface{}) func() interface{} {
	// Return fn as is if it is desired type
	if res, isa := fn.(func() interface{}); isa {
		return res
	}

	// Verify fn has is a non-nil func of 0 parameters and 1 result
	vfn := reflect.ValueOf(fn)

	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(supplierErrorMsg)
	}

	// The func has to accept no args or a single variadic arg and return 1 type
	typ := vfn.Type()
	if !(((typ.NumIn() == 0) || ((typ.NumIn() == 1) && (typ.IsVariadic()))) &&
		(typ.NumOut() == 1)) {
		panic(supplierErrorMsg)
	}

	return func() interface{} {
		resVal := vfn.Call([]reflect.Value{})[0].Interface()

		return resVal
	}
}

// SupplierOf (fn, X) adapts a func() X' into a func() X.
// If fn happens to be a func() X, it is returned as is.
// Otherwise, type X' must be convertible to X.
// The result will have to be type asserted by the caller.
// fn may have a single variadic argument.
func SupplierOf(fn interface{}, val interface{}) interface{} {
	// val cannot be nil
	if IsNil(val) {
		panic("val cannot be nil")
	}

	// Verify val is a non-interface type
	var (
		xval = reflect.ValueOf(val)
		xtyp = xval.Type()
	)
	if xval.Kind() == reflect.Interface {
		panic("val cannot be an interface{} value")
	}

	// Verify fn has is a non-nil func of 0 parameters and 1 result
	var (
		vfn    = reflect.ValueOf(fn)
		errMsg = fmt.Sprintf(supplierOfErrorMsg, xtyp)
	)

	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(errMsg)
	}

	// The func has to accept no args or a single variadic arg and return 1 type
	typ := vfn.Type()
	if !(((typ.NumIn() == 0) || ((typ.NumIn() == 1) && (typ.IsVariadic()))) &&
		(typ.NumOut() == 1)) {
		panic(errMsg)
	}

	resTyp := typ.Out(0)

	// Return fn as is if it is desired type
	if resTyp == xtyp {
		return fn
	}

	// If fn returns any type convertible to X, then generate a function that returns exactly X
	if !resTyp.ConvertibleTo(xtyp) {
		panic(errMsg)
	}

	return reflect.MakeFunc(
		reflect.FuncOf(
			[]reflect.Type{},
			[]reflect.Type{xtyp},
			false,
		),
		func(args []reflect.Value) []reflect.Value {
			resVal := vfn.Call([]reflect.Value{})[0].Convert(xtyp)

			return []reflect.Value{resVal}
		},
	).Interface()
}

// Consumer (fn) adapts a func(any) into a func(interface{})
// If fn happens to be a func(interface{}), it is returned as is.
// Otherwise, each invocation converts the arg passed to the type the func receives.
func Consumer(fn interface{}) func(interface{}) {
	// Return fn as is if it is desired type
	if res, isa := fn.(func(interface{})); isa {
		return res
	}

	// Verify fn has is a non-nil func of 1 parameters and no result
	vfn := reflect.ValueOf(fn)

	if (vfn.Kind() != reflect.Func) || vfn.IsNil() {
		panic(consumerErrorMsg)
	}

	// The func has to accept one arg and return nothing
	typ := vfn.Type()
	if (typ.NumIn() != 1) || (typ.NumOut() != 0) {
		panic(consumerErrorMsg)
	}

	argTyp := typ.In(0)

	return func(arg interface{}) {
		argVal := reflect.ValueOf(arg).Convert(argTyp)
		vfn.Call([]reflect.Value{argVal})
	}
}

// Ternary returns trueVal if expr is true, else it returns falseVal
func Ternary(expr bool, trueVal, falseVal interface{}) interface{} {
	if expr {
		return trueVal
	}

	return falseVal
}

// TernaryOf returns trueVal() if expr is true, else it returns falseVal()
// trueVal and falseVal must be func() any.
func TernaryOf(expr bool, trueVal, falseVal interface{}) interface{} {
	if expr {
		return Supplier(trueVal)()
	}

	return Supplier(falseVal)()
}

// PanicE panics if err is non-nil
func PanicE(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// PanicVE panics if err is non-nil, otherwise returns val
func PanicVE(val interface{}, err error) interface{} {
	if err != nil {
		panic(err.Error())
	}

	return val
}

// PanicBM panics with msg if valid is false
func PanicBM(valid bool, msg string) {
	if !valid {
		panic(msg)
	}
}

// PanicVBM panics with msg if valid is false, else returns val
func PanicVBM(val interface{}, valid bool, msg string) interface{} {
	if !valid {
		panic(msg)
	}

	return val
}

// SortFunc adapts a func(val1, val2 any) bool into a func(val1, val2 interface{}) bool.
// If fn is already a func(val1, val2 interface{}) bool, it is returned as is.
// The passed func must return true if and only if val1 < val2.
// Panics if fn is nil, not a func, does not accept two args of the same type, or does not return a single bool value.
func SortFunc(fn interface{}) func(val1, val2 interface{}) bool {
	PanicBM(!IsNil(fn), sortErrorMsg)

	// If fn is already the right signature, return it as is
	if f, isa := fn.(func(val1, val2 interface{}) bool); isa {
		return f
	}

	var (
		vfn   = reflect.ValueOf(fn)
		fnTyp = vfn.Type()
	)

	if !((fnTyp.Kind() == reflect.Func) &&
		(fnTyp.NumIn() == 2) &&
		(fnTyp.NumOut() == 1) &&
		(fnTyp.In(0) == fnTyp.In(1)) &&
		(fnTyp.Out(0).Kind() == reflect.Bool)) {
		panic(sortErrorMsg)
	}

	valTyp := fnTyp.In(0)

	return func(val1, val2 interface{}) bool {
		return vfn.Call([]reflect.Value{
			reflect.ValueOf(val1).Convert(valTyp),
			reflect.ValueOf(val2).Convert(valTyp),
		})[0].Bool()
	}
}

var (
	// IntSortFunc returns true if int64 val1 < val2
	IntSortFunc = SortFunc(func(val1, val2 int64) bool {
		return val1 < val2
	})

	// UintSortFunc returns true if uint64 val1 < val2
	UintSortFunc = SortFunc(func(val1, val2 uint64) bool {
		return val1 < val2
	})

	// FloatSortFunc returns true if float64 val1 < val2
	FloatSortFunc = SortFunc(func(val1, val2 float64) bool {
		return val1 < val2
	})

	// ComplexSortFunc returns true if abs(complex128 val1) < abs(complex128 val2)
	ComplexSortFunc = SortFunc(func(val1, val2 complex128) bool {
		return cmplx.Abs(val1) < cmplx.Abs(val2)
	})

	// StringSortFunc returns true if string val1 < val2
	StringSortFunc = SortFunc(func(val1, val2 string) bool {
		return val1 < val2
	})

	// BigIntSortFunc returns true if big.Int val1 < val2
	BigIntSortFunc = SortFunc(func(val1, val2 *big.Int) bool {
		return val1.Cmp(val2) == -1
	})

	// BigRatSortFunc returns true if big.Rat val1 < val2
	BigRatSortFunc = SortFunc(func(val1, val2 *big.Rat) bool {
		return val1.Cmp(val2) == -1
	})

	// BigFloatSortFunc returns true if big.Float val1 < val2
	BigFloatSortFunc = SortFunc(func(val1, val2 *big.Float) bool {
		return val1.Cmp(val2) == -1
	})
)
