// SPDX-License-Identifier: Apache-2.0

package gofuncs

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexOf(t *testing.T) {
	// Slice
	// Index exists
	assert.Equal(t, 1, IndexOf([]int{1}, 0))
	// Index does not exist, have default
	assert.Equal(t, 2, IndexOf([]int{1}, 1, 2))
	// Index does not exist, no default
	assert.Equal(t, 0, IndexOf([]int{1}, 1))

	// Array
	// Index exists
	assert.Equal(t, 1, IndexOf([1]int{1}, 0))
	// Index does not exist, have default
	assert.Equal(t, 2, IndexOf([1]int{1}, 1, 2))
	// Index does not exist, no default
	assert.Equal(t, 0, IndexOf([1]int{1}, 1))

	func() {
		defer func() {
			assert.Equal(t, indexOfErrorMsg, recover())
		}()

		IndexOf(nil, 0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer func() {
			assert.Equal(t, indexOfErrorMsg, recover())
		}()

		IndexOf(5, 0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer func() {
			assert.Equal(t, indexOfErrorMsg, recover())
		}()

		IndexOf(5, 0)
		assert.Fail(t, "must panic")
	}()
}

func TestValueOfKey(t *testing.T) {
	// Key exists
	assert.Equal(t, 1, ValueOfKey(map[string]int{"1": 1}, "1"))
	// Key does not exist, with default
	assert.Equal(t, 2, ValueOfKey(map[string]int{"1": 1}, "", 2))
	// Key does not exist, no default
	assert.Equal(t, 0, ValueOfKey(map[string]int{"1": 1}, ""))

	func() {
		defer func() {
			assert.Equal(t, valueOfKeyErrorMsg, recover())
		}()

		ValueOfKey(nil, 0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer func() {
			assert.Equal(t, valueOfKeyErrorMsg, recover())
		}()

		ValueOfKey(5, 0)
		assert.Fail(t, "must panic")
	}()
}

func TestFilter(t *testing.T) {
	// Exact match
	filterFn := Filter(func(i interface{}) bool { return i.(int) < 3 })
	assert.True(t, filterFn(1))
	assert.False(t, filterFn(5))

	// Inexact match
	filterFn = Filter(func(i int) bool { return i < 3 })
	assert.True(t, filterFn(uint8(1)))
	assert.False(t, filterFn(5))

	// Exact and Inexact match all
	filterFns := FilterAll(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i >= 0 },
	)

	assert.True(t, filterFns[0](1))
	assert.False(t, filterFns[1](int8(-1)))

	// Exact and Inexact match And
	filterFn = And(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i >= 0 },
	)

	assert.True(t, filterFn(1))
	assert.False(t, filterFn(-1))

	// Exact and Inexact match Or
	filterFn = Or(
		func(i interface{}) bool { return i.(int) < 3 },
		func(i int) bool { return i%2 == 0 },
	)

	assert.True(t, filterFn(1))
	assert.False(t, filterFn(5))

	// Exact and Inexact match Not
	filterFn = Not(func(i interface{}) bool { return i.(int) < 3 })

	assert.False(t, filterFn(1))
	assert.True(t, filterFn(5))

	// EqualTo/DeepEqualTo
	filterFn = EqualTo(nil)
	filterFn2 := DeepEqualTo(nil)

	assert.True(t, filterFn(nil))
	assert.True(t, filterFn2(nil))
	assert.False(t, filterFn(0))
	assert.False(t, filterFn2(0))

	filterFn = EqualTo(([]int)(nil))
	filterFn2 = DeepEqualTo(([]int)(nil))

	assert.True(t, filterFn(([]int)(nil)))
	assert.True(t, filterFn2(([]int)(nil)))
	assert.False(t, filterFn(nil))
	assert.False(t, filterFn2(nil))
	assert.False(t, filterFn(([]string)(nil)))
	assert.False(t, filterFn2(([]string)(nil)))

	theVal := []int{1, 2}
	filterFn = EqualTo(theVal)
	filterFn2 = DeepEqualTo([]int{1, 2})

	assert.False(t, filterFn(([]int)(nil)))
	assert.False(t, filterFn2(([]int)(nil)))
	assert.False(t, filterFn(nil))
	assert.False(t, filterFn2(nil))
	assert.False(t, filterFn([]int{1}))
	assert.False(t, filterFn2([]int{1}))
	assert.False(t, filterFn([]int{1, 2}))
	assert.True(t, filterFn2([]int{1, 2}))
	assert.True(t, filterFn(theVal))
	assert.True(t, filterFn2(theVal))

	filterFn = EqualTo(1)
	filterFn2 = DeepEqualTo(1)

	assert.True(t, filterFn(int8(1)))
	assert.True(t, filterFn2(int8(1)))
	assert.False(t, filterFn(5))
	assert.False(t, filterFn2(5))

	// LessThan
	filterFn = IsLessThan(5)
	assert.True(t, filterFn(int8(3)))
	assert.False(t, filterFn(5))

	// LessThanEquals
	filterFn = IsLessThanEquals(5)
	assert.True(t, filterFn(int8(5)))
	assert.False(t, filterFn(6))

	// GreaterThan
	filterFn = IsGreaterThan(5)
	assert.True(t, filterFn(int8(6)))
	assert.False(t, filterFn(5))

	// GreaterThanEquals
	filterFn = IsGreaterThanEquals(5)
	assert.True(t, filterFn(int8(5)))
	assert.False(t, filterFn(4))

	// IsNegative
	filterFn = IsNegative
	assert.True(t, filterFn(int8(-1)))
	assert.False(t, filterFn(0))

	// IsNonNegative
	filterFn = IsNonNegative
	assert.True(t, filterFn(int8(0)))
	assert.False(t, filterFn(-1))

	// IsPositive
	filterFn = IsPositive
	assert.True(t, filterFn(int8(1)))
	assert.False(t, filterFn(0))

	// Nil
	filterFn = IsNil

	assert.True(t, filterFn(nil))
	assert.True(t, filterFn([]int(nil)))
	var f func()
	assert.True(t, filterFn(f))
	assert.False(t, filterFn(""))

	// IsNilable
	filterFn = IsNilable
	assert.True(t, filterFn(nil))
	assert.True(t, filterFn([]int(nil)))
	assert.True(t, filterFn(f))
	assert.False(t, filterFn(""))

	deferFunc := func() {
		assert.Equal(t, filterErrorMsg, recover())
	}

	func() {
		defer deferFunc()

		// Not a func
		Filter(0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil
		Filter(nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil func
		var fn func()
		Filter(fn)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No arg
		Filter(func() {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No result
		Filter(func(int) {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Wrong result type
		Filter(func(int) int { return 0 })
		assert.Fail(t, "must panic")
	}()
}

func TestMap(t *testing.T) {
	// Exact match
	mapFn := Map(func(i interface{}) interface{} { return i.(int) * 2 })
	assert.Equal(t, 2, mapFn(1))

	// Inexact match
	mapFn = Map(func(i int) int { return i * 2 })
	assert.Equal(t, 4, mapFn(uint8(2)))
	assert.Equal(t, 6, mapFn(3))

	deferFunc := func() {
		assert.Equal(t, mapErrorMsg, recover())
	}

	func() {
		defer deferFunc()

		// Not a func
		Map(0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil
		Map(nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil func
		var fn func(int) int
		Map(fn)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No args
		Map(func() {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No result
		Map(func(int) {})
		assert.Fail(t, "must panic")
	}()
}

func TestMapTo(t *testing.T) {
	// Exact match
	mapFn := MapTo(func(i interface{}) int { return i.(int) * 2 }, 0).(func(interface{}) int)
	assert.Equal(t, 2, mapFn(1))

	// Inexact match
	mapFn = MapTo(func(i int) int { return i * 2 }, 0).(func(interface{}) int)
	assert.Equal(t, 4, mapFn(2))

	// Conversion match
	mapFn = MapTo(func(i int8) int8 { return i * 2 }, 0).(func(interface{}) int)
	assert.Equal(t, 4, mapFn(2))

	// Arg of different type
	mapFn = MapTo(func(s string) int { str, _ := strconv.Atoi(s); return str }, 0).(func(interface{}) int)
	assert.Equal(t, 2, mapFn("2"))

	deferGen := func(errMsg string) func() {
		return func() {
			assert.Equal(t, errMsg, recover())
		}
	}

	func() {
		defer deferGen("val cannot be nil")()
		MapTo(nil, nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferGen("val cannot be nil")()
		var p *int
		MapTo(p, p)
		assert.Fail(t, "must panic")
	}()

	// Not a function
	func() {
		defer deferGen(fmt.Sprintf(mapToErrorMsg, "int"))()
		MapTo("", 0)
		assert.Fail(t, "must panic")
	}()

	// Wrong signature
	func() {
		defer deferGen(fmt.Sprintf(mapToErrorMsg, "int"))()
		MapTo(func() {}, 0)
		assert.Fail(t, "must panic")
	}()

	// Returns uncovertible type
	func() {
		defer deferGen(fmt.Sprintf(mapToErrorMsg, "int"))()
		MapTo(func(string) string { return "" }, 0)
		assert.Fail(t, "must panic")
	}()
}

func TestConvertTo(t *testing.T) {
	convertFn := ConvertTo(int8(0))
	assert.Equal(t, int8(1), convertFn(1))
}

func TestSupplier(t *testing.T) {
	// Exact match
	supplierFn := Supplier(func() interface{} { return 2 })
	assert.Equal(t, 2, supplierFn())

	// Inexact match
	supplierFn = Supplier(func() int { return 4 })
	assert.Equal(t, 4, supplierFn())

	// Variadic match
	supplierFn = Supplier(func(...int) int { return 6 })
	assert.Equal(t, 6, supplierFn())

	deferFunc := func() {
		assert.Equal(t, supplierErrorMsg, recover())
	}

	func() {
		defer deferFunc()

		// Not a func
		Supplier(0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil
		Supplier(nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil func
		var fn func() int
		Supplier(fn)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Has args
		Supplier(func(int) {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// variadic but not only arg
		Supplier(func(int, ...int) int { return 0 })
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No result
		Supplier(func() {})
		assert.Fail(t, "must panic")
	}()
}

func TestSupplierOf(t *testing.T) {
	// Exact match
	supplierFn := SupplierOf(func() int { return 2 }, 0).(func() int)
	assert.Equal(t, 2, supplierFn())

	// Conversion match
	supplierFn = SupplierOf(func() int8 { return 4 }, 0).(func() int)
	assert.Equal(t, 4, supplierFn())

	// Variadic match
	supplierFn = SupplierOf(func(...int) int8 { return 6 }, 0).(func() int)
	assert.Equal(t, 6, supplierFn())

	deferGen := func(errMsg string) func() {
		return func() {
			assert.Equal(t, errMsg, recover())
		}
	}

	func() {
		defer deferGen("val cannot be nil")()
		SupplierOf(nil, nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferGen("val cannot be nil")()
		var p *int
		SupplierOf(p, p)
		assert.Fail(t, "must panic")
	}()

	// Not a function
	func() {
		defer deferGen(fmt.Sprintf(supplierOfErrorMsg, "int"))()
		SupplierOf("", 0)
		assert.Fail(t, "must panic")
	}()

	// Wrong signature
	func() {
		defer deferGen(fmt.Sprintf(supplierOfErrorMsg, "int"))()
		SupplierOf(func() {}, 0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferGen(fmt.Sprintf(supplierOfErrorMsg, "int"))()

		// variadic but not only arg
		SupplierOf(func(int, ...int) int { return 0 }, 0)
		assert.Fail(t, "must panic")
	}()

	// Returns uncovertible type
	func() {
		defer deferGen(fmt.Sprintf(supplierOfErrorMsg, "int"))()
		SupplierOf(func() string { return "" }, 0)
		assert.Fail(t, "must panic")
	}()
}

func TestConsumer(t *testing.T) {
	// Exact match
	var (
		val        interface{}
		consumerFn = Consumer(func(i interface{}) { val = i })
	)
	consumerFn(2)
	assert.Equal(t, 2, val)

	// Inexact match
	consumerFn = Consumer(func(i int) { val = i })
	consumerFn(uint8(3))
	assert.Equal(t, 3, val)
	consumerFn(4)
	assert.Equal(t, 4, val)

	deferFunc := func() {
		assert.Equal(t, consumerErrorMsg, recover())
	}

	func() {
		defer deferFunc()

		// Not a func
		Consumer(0)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil
		Consumer(nil)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Nil func
		var fn func()
		Consumer(fn)
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// No arg
		Consumer(func() {})
		assert.Fail(t, "must panic")
	}()

	func() {
		defer deferFunc()

		// Has result
		Consumer(func() int { return 0 })
		assert.Fail(t, "must panic")
	}()
}

func TestTernary(t *testing.T) {
	assert.Equal(t, 1, Ternary(true, 1, 2))
	assert.Equal(t, 2, Ternary(false, 1, 2))

	assert.Equal(t, 1, TernaryOf(true, func() interface{} { return 1 }, func() interface{} { return 2 }))
	assert.Equal(t, 2, TernaryOf(false, func() int { return 1 }, func() int { return 2 }))
}

func TestPanic(t *testing.T) {
	var str string
	PanicE(json.Unmarshal([]byte(`"abc"`), &str))
	assert.Equal(t, "abc", str)

	func() {
		defer func() {
			assert.Equal(t, "unexpected end of JSON input", recover())
		}()

		PanicE(json.Unmarshal([]byte("{"), &str))
		assert.Fail(t, "json.Unmarshal must fail")
	}()

	i := PanicVE(strconv.Atoi("1")).(int)
	assert.Equal(t, 1, i)

	func() {
		defer func() {
			assert.Equal(t, `strconv.Atoi: parsing "a": invalid syntax`, recover())
		}()

		PanicVE(strconv.Atoi("a"))
		assert.Fail(t, "strconv must fail")
	}()

	PanicBM(big.NewRat(2, 1).IsInt(), "must be int")

	func() {
		defer func() {
			assert.Equal(t, "must be int", recover())
		}()

		PanicBM(big.NewRat(2, 3).IsInt(), "must be int")
		assert.Fail(t, "IsInt must fail")
	}()

	f, ok := big.NewFloat(1.0).SetString("2")
	PanicVBM(f, ok, "must be float64")
	assert.Equal(t, "2", f.String())

	func() {
		defer func() {
			assert.Equal(t, "must be float64", recover())
		}()

		f, ok = big.NewFloat(1.0).SetString("a")
		PanicVBM(f, ok, "must be float64")
		assert.Fail(t, "Float64 must fail")
	}()
}

func TestSortFunc(t *testing.T) {
	sf := SortFunc(func(val1, val2 int) bool { return val1 < val2 })
	assert.True(t, sf(1, 2))
	assert.False(t, sf(2, 1))
	assert.True(t, sf(int8(1), int8(2)))
	assert.False(t, sf(int8(2), int8(1)))

	sf = IntSortFunc
	assert.True(t, sf(1, 2))
	assert.False(t, sf(2, 1))
	assert.True(t, sf(int8(1), int8(2)))
	assert.False(t, sf(int8(2), int8(1)))

	sf = UintSortFunc
	assert.True(t, sf(uint(1), uint(2)))
	assert.False(t, sf(uint(2), uint(1)))
	assert.True(t, sf(uint8(1), uint8(2)))
	assert.False(t, sf(uint8(2), uint8(1)))

	sf = FloatSortFunc
	assert.True(t, sf(1.0, 2.0))
	assert.False(t, sf(2.0, 1.0))
	assert.True(t, sf(int8(1), int8(2)))
	assert.False(t, sf(int8(2), int8(1)))

	sf = ComplexSortFunc
	assert.True(t, sf((1+2i), (2+3i)))
	assert.False(t, sf((2+3i), (1+2i)))
	assert.True(t, sf(complex64(1+2i), complex64(2+3i)))
	assert.False(t, sf(complex64(2+3i), complex64(1+2i)))

	sf = StringSortFunc
	assert.True(t, sf("a", "b"))
	assert.False(t, sf("b", "a"))
	assert.True(t, sf('1', '2'))
	assert.False(t, sf('2', '1'))

	sf = BigIntSortFunc
	assert.True(t, sf(big.NewInt(1), big.NewInt(2)))
	assert.False(t, sf(big.NewInt(2), big.NewInt(1)))

	sf = BigRatSortFunc
	assert.True(t, sf(big.NewRat(1, 2), big.NewRat(2, 1)))
	assert.False(t, sf(big.NewRat(2, 1), big.NewRat(1, 2)))

	sf = BigFloatSortFunc
	assert.True(t, sf(big.NewFloat(1.0), big.NewFloat(2.0)))
	assert.False(t, sf(big.NewFloat(2.0), big.NewFloat(1.0)))
}
