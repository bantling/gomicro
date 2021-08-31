// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	"fmt"
	"math/big"
	"math/cmplx"
	"reflect"
)

const (
	indexOfErrorMsg    = "slc must be a slice"
	valueOfKeyErrorMsg = "mp must be a map"
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
