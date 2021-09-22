// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"math/big"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// Error constants
const (
	ErrExampleValueIsNotAStruct = "The value provided is not a struct or a pointer to a struct"
	ErrElementIsNotAMap         = "The stream elements passed to MapToStruct must all be map[string]interface{}"
)

// BoolString represents a union of bool and string, to allow bool fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type BoolString struct {
	IsMsg bool
	Value bool
	Msg   string
}

// IntString represents a union of int and string, to allow int fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type IntString struct {
	IsMsg bool
	Value int
	Msg   string
}

// UintString represents a union of uint and string, to allow uint fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type UintString struct {
	IsMsg bool
	Value uint
	Msg   string
}

// FloatString represents a union of float64 and string, to allow float fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type FloatString struct {
	IsMsg bool
	Value float64
	Msg   string
}

// BigIntString represents a union of math.big/Int and string, to allow Int fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type BigIntString struct {
	IsMsg bool
	Value *big.Int
	Msg   string
}

// BigFloatString represents a union of math.big/Float and string, to allow Float fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type BigFloatString struct {
	IsMsg bool
	Value *big.Float
	Msg   string
}

// StructString represents a union of a struct and string, to allow struct fields to be redacted.
// IsMsg is false if the Value field is selected, true if the Msg field is selected.
type StructString struct {
	IsMsg bool
	Value interface{}
	Msg   string
}

// BoolStringHookFunc returns a DecodeHookFunc that converts values into BoolString.
// The values are not bools or strings, they are ignored.
func BoolStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(BoolString{}) {
			switch f.Kind() {
			case reflect.Bool:
				return BoolString{IsMsg: false, Value: data.(bool)}, nil
			case reflect.String:
				return BoolString{IsMsg: true, Msg: data.(string)}, nil
			}
		}

		// Ignore everything except conversions from bool or string to BoolString
		return data, nil
	}
}

// IntStringHookFunc returns a DecodeHookFunc that converts values into IntString.
// The values are not any kind of int or uint or strings, they are ignored.
func IntStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(IntString{}) {
			switch f.Kind() {
			case reflect.Int8:
				return IntString{IsMsg: false, Value: int(data.(int8))}, nil
			case reflect.Int16:
				return IntString{IsMsg: false, Value: int(data.(int16))}, nil
			case reflect.Int32:
				return IntString{IsMsg: false, Value: int(data.(int32))}, nil
			case reflect.Int64:
				return IntString{IsMsg: false, Value: int(data.(int64))}, nil
			case reflect.Int:
				return IntString{IsMsg: false, Value: data.(int)}, nil

			case reflect.Uint8:
				return IntString{IsMsg: false, Value: int(data.(uint8))}, nil
			case reflect.Uint16:
				return IntString{IsMsg: false, Value: int(data.(uint16))}, nil
			case reflect.Uint32:
				return IntString{IsMsg: false, Value: int(data.(uint32))}, nil
			case reflect.Uint64:
				return IntString{IsMsg: false, Value: int(data.(uint64))}, nil
			case reflect.Uint:
				return IntString{IsMsg: false, Value: int(data.(uint))}, nil

			case reflect.String:
				return IntString{IsMsg: true, Msg: data.(string)}, nil
			}
		}

		// Ignore everything except conversions from any kind of int or uint or string to IntString
		return data, nil
	}
}

// UintStringHookFunc returns a DecodeHookFunc that converts values into UintString.
// The values are not any kind of int or uint or strings, they are ignored.
func UintStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(UintString{}) {
			switch f.Kind() {
			case reflect.Int8:
				return UintString{IsMsg: false, Value: uint(data.(int8))}, nil
			case reflect.Int16:
				return UintString{IsMsg: false, Value: uint(data.(int16))}, nil
			case reflect.Int32:
				return UintString{IsMsg: false, Value: uint(data.(int32))}, nil
			case reflect.Int64:
				return UintString{IsMsg: false, Value: uint(data.(int64))}, nil
			case reflect.Int:
				return UintString{IsMsg: false, Value: uint(data.(int))}, nil

			case reflect.Uint8:
				return UintString{IsMsg: false, Value: uint(data.(uint8))}, nil
			case reflect.Uint16:
				return UintString{IsMsg: false, Value: uint(data.(uint16))}, nil
			case reflect.Uint32:
				return UintString{IsMsg: false, Value: uint(data.(uint32))}, nil
			case reflect.Uint64:
				return UintString{IsMsg: false, Value: uint(data.(uint64))}, nil
			case reflect.Uint:
				return UintString{IsMsg: false, Value: data.(uint)}, nil

			case reflect.String:
				return UintString{IsMsg: true, Msg: data.(string)}, nil
			}
		}

		// Ignore everything except conversions from any kind of int or uint or string to UintString
		return data, nil
	}
}

// FloatStringHookFunc returns a DecodeHookFunc that converts values into FloatString.
// The values are not any kind of int or uint or float or strings, they are ignored.
func FloatStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(FloatString{}) {
			switch f.Kind() {
			case reflect.Int8:
				return FloatString{IsMsg: false, Value: float64(data.(int8))}, nil
			case reflect.Int16:
				return FloatString{IsMsg: false, Value: float64(data.(int16))}, nil
			case reflect.Int32:
				return FloatString{IsMsg: false, Value: float64(data.(int32))}, nil
			case reflect.Int64:
				return FloatString{IsMsg: false, Value: float64(data.(int64))}, nil
			case reflect.Int:
				return FloatString{IsMsg: false, Value: float64(data.(int))}, nil

			case reflect.Uint8:
				return FloatString{IsMsg: false, Value: float64(data.(uint8))}, nil
			case reflect.Uint16:
				return FloatString{IsMsg: false, Value: float64(data.(uint16))}, nil
			case reflect.Uint32:
				return FloatString{IsMsg: false, Value: float64(data.(uint32))}, nil
			case reflect.Uint64:
				return FloatString{IsMsg: false, Value: float64(data.(uint64))}, nil
			case reflect.Uint:
				return FloatString{IsMsg: false, Value: float64(data.(uint))}, nil

			case reflect.Float32:
				return FloatString{IsMsg: false, Value: float64(data.(float32))}, nil
			case reflect.Float64:
				return FloatString{IsMsg: false, Value: data.(float64)}, nil

			case reflect.String:
				return FloatString{IsMsg: true, Msg: data.(string)}, nil
			}
		}

		// Ignore everything except conversions from any kind of int or uint or string to FloatString
		return data, nil
	}
}

// BigIntStringHookFunc returns a DecodeHookFunc that converts values into *math/big.Int.
// The values are not any kind of int or uint or *math/big.Int or strings, they are ignored.
func BigIntStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(BigIntString{}) {
			switch f.Kind() {
			case reflect.Int8:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(int8)))}, nil
			case reflect.Int16:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(int16)))}, nil
			case reflect.Int32:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(int32)))}, nil
			case reflect.Int64:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(int64)))}, nil
			case reflect.Int:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(int)))}, nil

			case reflect.Uint8:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(uint8)))}, nil
			case reflect.Uint16:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(uint16)))}, nil
			case reflect.Uint32:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(uint32)))}, nil
			case reflect.Uint64:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(uint64)))}, nil
			case reflect.Uint:
				return BigIntString{IsMsg: false, Value: big.NewInt(int64(data.(uint)))}, nil

			case reflect.String:
				return BigIntString{IsMsg: true, Msg: data.(string)}, nil
			}

			if f == reflect.TypeOf((*big.Int)(nil)) {
				return BigIntString{IsMsg: false, Value: data.(*big.Int)}, nil
			}
		}

		// Ignore everything except conversions from any kind of int or uint or *Int or string to BigIntString
		return data, nil
	}
}

// BigFloatStringHookFunc returns a DecodeHookFunc that converts values into *math/big.Int.
// The values are not any kind of int or uint or *math/big.Int or *math.Big/Float or strings, they are ignored.
func BigFloatStringHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if t == reflect.TypeOf(BigFloatString{}) {
			switch f.Kind() {
			case reflect.Int8:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(int8)))}, nil
			case reflect.Int16:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(int16)))}, nil
			case reflect.Int32:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(int32)))}, nil
			case reflect.Int64:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(int64)))}, nil
			case reflect.Int:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(int)))}, nil

			case reflect.Uint8:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(uint8)))}, nil
			case reflect.Uint16:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(uint16)))}, nil
			case reflect.Uint32:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(uint32)))}, nil
			case reflect.Uint64:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(uint64)))}, nil
			case reflect.Uint:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(uint)))}, nil

			case reflect.Float32:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(float64(data.(float32)))}, nil
			case reflect.Float64:
				return BigFloatString{IsMsg: false, Value: big.NewFloat(data.(float64))}, nil

			case reflect.String:
				return BigFloatString{IsMsg: true, Msg: data.(string)}, nil
			}

			if f == reflect.TypeOf((*big.Int)(nil)) {
				return BigFloatString{IsMsg: false, Value: big.NewFloat(0).SetInt(data.(*big.Int))}, nil
			}

			if f == reflect.TypeOf((*big.Float)(nil)) {
				return BigFloatString{IsMsg: false, Value: data.(*big.Float)}, nil
			}
		}

		// Ignore everything except conversions from any kind of int or uint or float or *Int or *Float or string to BigFloatString
		return data, nil
	}
}

// ComposedValueStringHookFunc is DecodeHookFunc that is a composition of all the above XStringHookFuncs.
func ComposedValueStringHookFunc() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		BoolStringHookFunc(),
		IntStringHookFunc(),
		UintStringHookFunc(),
		FloatStringHookFunc(),
		BigIntStringHookFunc(),
		BigFloatStringHookFunc(),
	)
}

var (
	mapstructureDecoderConfig = mapstructure.DecoderConfig{DecodeHook: ComposedValueStringHookFunc(), Squash: true}
)

// MapToStruct is a Stream.Map function that maps each map[string]interface{} element into a struct of the given example value.
// Panics if the given example value is not zero or more pointers to a struct or a reflect.Type instance of the same.
// Panics if the stream elements are not map[string]interface{}.
func MapToStruct(typ interface{}) func(element interface{}) interface{} {
	// Get type of struct and count of pointer indirects, if any
	var (
		rtyp  reflect.Type
		nptrs = 0
	)

	if refTyp, isa := typ.(reflect.Type); isa {
		rtyp = refTyp
	} else {
		rtyp = reflect.ValueOf(typ).Type()
	}

	for rtyp.Kind() == reflect.Ptr {
		rtyp = rtyp.Elem()
		nptrs++
	}

	// Ensure it is a struct
	if rtyp.Kind() != reflect.Struct {
		panic(ErrExampleValueIsNotAStruct)
	}

	return func(element interface{}) interface{} {
		mapVal, isa := element.(map[string]interface{})
		if !isa {
			panic(ErrElementIsNotAMap)
		}

		// Create a new instance of the struct for each decode, to guarantee each element of new stream is a separate value
		var (
			structPtr     = reflect.New(rtyp)
			decoderConfig = mapstructureDecoderConfig
			decoder       *mapstructure.Decoder
			err           error
		)
		decoderConfig.Result = structPtr.Interface()
		if decoder, err = mapstructure.NewDecoder(&decoderConfig); err != nil {
			panic(err)
		}
		if err = decoder.Decode(mapVal); err != nil {
			panic(err)
		}

		// Return a value of the correct number of pointers
		switch nptrs {
		case 0:
			return structPtr.Elem().Interface()
		case 1:
			return structPtr.Interface()
		default:
			for ; nptrs > 1; nptrs-- {
				nextStructPtr := reflect.New(structPtr.Type())
				nextStructPtr.Elem().Set(structPtr)
				structPtr = nextStructPtr
			}

			return structPtr.Interface()
		}
	}
}
