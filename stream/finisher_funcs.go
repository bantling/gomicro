// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"strconv"
	"sync"

	"github.com/bantling/gomicro/iter"
)

// Error constants
const (
	ErrInvalidJSONDocument = "The elements are not a valid JSON array or object"
	ErrInvalidJSONArray    = "The elements are not a valid JSON array"
	ErrInvalidJSONObject   = "The elements are not a valid JSON object"
	ErrNotAnArrayOrSlice   = "The elements must be arrays or slices"
	ErrInvalidBigInt       = "A number couild not be converted to a math/big.Int"
	ErrInvalidBigFloat     = "A number couild not be converted to a math/big.Float"
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

// JSONDocType describes what kind of JSON documents to allow - arrays or objects, only arrays, or only objects
type JSONDocType uint

// JSONDocType constants
const (
	JSONArrayOrObject JSONDocType = iota
	JSONArray
	JSONObject
)

// JSONNumberType describes what kind of Go type a JSON number should be translated to
type JSONNumberType uint

// JSONNumberType constants
const (
	JSONNumAsNumber = iota
	JSONNumAsInt64
	JSONNumAsUint64
	JSONNumAsFloat64
	JSONNumAsBigInt
	JSONNumAsBigFloat
	JSONNumAsString
)

// JSONConfig contains the parameters for JSON parsing
type JSONConfig struct {
	DocType JSONDocType
	NumType JSONNumberType
}

// JSONNumberToNumber converts a json.Number into a json.Number.
// Just returns the value as is.
func JSONNumberToNumber(num json.Number) interface{} {
	return num
}

// JSONNumberToInt64 converts a json.Number into an int64.
// Panics if the Number cannot be converted into an int64.
func JSONNumberToInt64(num json.Number) interface{} {
	var (
		val int64
		err error
	)

	if val, err = strconv.ParseInt(num.String(), 10, 64); err != nil {
		panic(err)
	}

	return val
}

// JSONNumberToUint64 converts a json.Number into a uint64.
// Panics if the Number cannot be converted into a uint64.
func JSONNumberToUint64(num json.Number) interface{} {
	var (
		val uint64
		err error
	)

	if val, err = strconv.ParseUint(num.String(), 10, 64); err != nil {
		panic(err)
	}

	return val
}

// JSONNumberToFloat64 converts a json.Number into a float64.
// Panics if the Number cannot be converted into a float64.
func JSONNumberToFloat64(num json.Number) interface{} {
	var (
		val float64
		err error
	)

	if val, err = strconv.ParseFloat(num.String(), 64); err != nil {
		panic(err)
	}

	return val
}

// JSONNumberToBigInt converts a json.Number into a math/big.Int.
// Panics if the Number cannot be converted into an Int.
func JSONNumberToBigInt(num json.Number) interface{} {
	var (
		val = big.NewInt(0)
		ok  bool
	)

	if val, ok = val.SetString(num.String(), 10); !ok {
		panic(ErrInvalidBigInt)
	}

	return val
}

// JSONNumberToBigFloat converts a json.Number into a math/big.Float.
// Panics if the Number cannot be converted into a Float.
func JSONNumberToBigFloat(num json.Number) interface{} {
	var (
		val = big.NewFloat(0.0)
		ok  bool
	)

	if val, ok = val.SetString(num.String()); !ok {
		panic(ErrInvalidBigFloat)
	}

	return val
}

// JSONNumberToString converts a json.Number into a string.
func JSONNumberToString(num json.Number) interface{} {
	return string(num)
}

// JSONNumberConversion returns a conversion function of json.Number to the specified type.
// Returns nil if typ is JSONNumAsNumber.
func JSONNumberConversion(typ JSONNumberType) func(json.Number) interface{} {
	switch typ {
	case JSONNumAsNumber:
		return JSONNumberToNumber
	case JSONNumAsInt64:
		return JSONNumberToInt64
	case JSONNumAsUint64:
		return JSONNumberToUint64
	case JSONNumAsFloat64:
		return JSONNumberToFloat64
	case JSONNumAsBigInt:
		return JSONNumberToBigInt
	case JSONNumAsBigFloat:
		return JSONNumberToBigFloat
	default:
		return JSONNumberToString
	}
}

// JSONDocumentNumberConversion recurses a JSON document (array or object) looking for array elements or object values
// that are instances of json.Number, and converts them using the given conversion function.
// The document is modified in place.
func JSONDocumentNumberConversion(doc interface{}, conv func(json.Number) interface{}) interface{} {
	handle := func(val interface{}) interface{} {
		if num, isNum := val.(json.Number); isNum {
			return conv(num)
		} else if _, isArray := val.([]interface{}); isArray {
			return JSONDocumentNumberConversion(val, conv)
		} else if _, isObj := val.(map[string]interface{}); isObj {
			return JSONDocumentNumberConversion(val, conv)
		}
		return val
	}

	if array, isArray := doc.([]interface{}); isArray {
		for i, val := range array {
			array[i] = handle(val)
		}

		return array
	}

	obj := doc.(map[string]interface{})
	for k, val := range obj {
		obj[k] = handle(val)
	}

	return obj
}

// ToJSON is a Transform function that maps each JSON array or object from the source bytes into a
// []interface{} or map[string]interface{}, respectively.
//
// The input may have multiple arrays and/or objects, where each one is a single element in the output.
// If the optional config parameter is passed, then the input may be restricted to contain only arrays or only objects,
// and the Go type to use for json numbers can be specified (json.Number, int, uint, float64, math/big.Int, math/big.Float, string).
// The default value for config is the zero value, which allows arrays and objects, and leaves numbers as json.Number.
//
// Panics if the elements are not bytes.
// Panics if the elements do not contain a valid JSON array or object.
// Panics if the expected doc type is restricted to only arrays or only objects, and the elements are not the expected type.
func ToJSON(config ...JSONConfig) func() func(*iter.Iter) *iter.Iter {
	var cfg JSONConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func() func(*iter.Iter) *iter.Iter {
		return func(it *iter.Iter) *iter.Iter {
			return iter.New(func() (interface{}, bool) {
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

				// Check first char according to JSONDocType
				ch = it.Value().(byte)
				switch cfg.DocType {
				case JSONArrayOrObject:
					if !((ch == '[') || (ch == '{')) {
						panic(ErrInvalidJSONDocument)
					}
				case JSONArray:
					if ch != '[' {
						panic(ErrInvalidJSONArray)
					}
				default:
					if ch != '{' {
						panic(ErrInvalidJSONObject)
					}
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
						if lastStack := stack[len(stack)-1]; ch == ']' {
							if lastStack != '[' {
								panic(ErrInvalidJSONDocument)
							}
						} else if lastStack != '{' {
							panic(ErrInvalidJSONDocument)
						}

						// Remove last element from stack, if it is empty, break
						if stack = stack[0 : len(stack)-1]; len(stack) == 0 {
							break
						}
					}
				}

				// The stack must be empty, or the doc ended prematurely
				if len(stack) > 0 {
					panic(ErrInvalidJSONDocument)
				}

				// Use json.Decoder to unmarshal the array or object from the buffer
				// (json.Unmarshal always translates numbers to float64)
				var (
					doc     interface{}
					decoder = json.NewDecoder(bytes.NewBuffer(buf))
				)
				// Decode numbers as json.Number
				decoder.UseNumber()

				if err := decoder.Decode(&doc); err != nil {
					panic(err)
				}

				// If the desired numeric type is not json.Number, then convert all json.Number to the requested type
				if cfg.NumType != JSONNumAsNumber {
					doc = JSONDocumentNumberConversion(doc, JSONNumberConversion(cfg.NumType))
				}

				return doc, true
			})
		}
	}
}

// FromArraySlice is a Transform function that maps each source array or slice into their elements.
// Panics if the elements are not arrays or slices.
func FromArraySlice() func(*iter.Iter) *iter.Iter {
	return func(it *iter.Iter) *iter.Iter {
		var (
			arraySlice reflect.Value
			n          = 0
			sz         = 0
		)

		return iter.New(func() (interface{}, bool) {
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
