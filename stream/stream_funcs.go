// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// Error constants
const (
	ErrExampleValueIsNotAStruct = "The value provided is not a struct or a pointer to a struct"
	ErrElementIsNotAMap         = "The stream elements passed to MapToStruct must all be map[string]interface{}"
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
		structPtr := reflect.New(rtyp)
		mapstructure.Decode(mapVal, structPtr.Interface())

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
