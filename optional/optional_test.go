// SPDX-License-Identifier: Apache-2.0

package optional

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionalOfEmptyPresentGet(t *testing.T) {
	var (
		c chan bool
		f func()
		m map[string]int
		p *int
		s []int
		e error

		val   interface{}
		valid bool
	)

	for _, opt := range []Optional{Of(), Of(c), Of(f), Of(m), Of(p), Of(s), Of(e)} {
		assert.Nil(t, opt.value)
		assert.False(t, opt.present)
		assert.True(t, opt.IsEmpty())
		assert.False(t, opt.IsPresent())
		called := false
		opt.IfPresent(func(interface{}) { called = true })
		assert.False(t, called)
		opt.IfEmpty(func() { called = true })
		assert.True(t, called)
		called = false
		opt.IfPresentOrElse(func(interface{}) {}, func() { called = true })
		assert.True(t, called)

		func() {
			defer func() {
				assert.True(t, errNotPresent == recover())
			}()

			opt.MustGet()
			assert.Fail(t, "Expected Panic")
		}()
	}

	opt := Of(0)
	assert.Equal(t, 0, opt.value)
	assert.True(t, opt.present)
	assert.False(t, opt.IsEmpty())
	assert.True(t, opt.IsPresent())
	val = 1
	opt.IfPresent(func(v interface{}) { val = v })
	assert.Equal(t, 0, val)
	val = 1
	opt.IfPresent(func(v int) { val = v })
	assert.Equal(t, 0, val)
	val = 1
	opt.IfEmpty(func() { val = 0 })
	assert.Equal(t, 1, val)
	val = 1
	opt.IfPresentOrElse(func(v interface{}) { val = v.(int) + 2 }, func() { val = 3 })
	assert.Equal(t, 2, val)
	val = 1
	opt.IfPresentOrElse(func(v int) { val = v + 2 }, func() { val = 3 })
	assert.Equal(t, 2, val)

	val, valid = opt.Get()
	assert.Equal(t, 0, val)
	assert.True(t, valid)
	assert.Equal(t, 0, opt.MustGet())

	opt = Of("")
	assert.Equal(t, "", opt.value)
	assert.True(t, opt.present)

	// Test zero value
	var zval Optional
	assert.Nil(t, zval.value)
	assert.False(t, zval.present)
	assert.True(t, zval.IsEmpty())
	assert.False(t, zval.IsPresent())
	called := false
	zval.IfPresent(func(interface{}) { called = true })
	assert.False(t, called)
	zval.IfEmpty(func() { called = true })
	assert.True(t, called)
	called = false
	zval.IfPresentOrElse(func(interface{}) {}, func() { called = true })
	assert.True(t, called)
	func() {
		defer func() {
			assert.True(t, errNotPresent == recover())
		}()

		zval.MustGet()
		assert.Fail(t, "Expected Panic")
	}()
}

func TestOptionalFilter(t *testing.T) {
	opt := Of(1)
	assert.True(t, opt == opt.Filter(func(val interface{}) bool { return true }))
	assert.True(t, opt.Filter(func(interface{}) bool { return false }).IsEmpty())
	assert.True(t, opt == opt.Filter(func(val int) bool { return true }))

	assert.True(t, Of().Filter(func(interface{}) bool { return true }).IsEmpty())
}

func TestOptionalIter(t *testing.T) {
	var (
		opt Optional = Of(1)
		it           = opt.Iter()
	)
	assert.True(t, it.Next())
	assert.Equal(t, 1, it.Value())
	assert.False(t, it.Next())

	opt = Of()
	it = opt.Iter()
	assert.False(t, it.Next())
}

func TestOptionalMap(t *testing.T) {
	too := func(val interface{}) interface{} {
		return val.(int) + 1
	}
	assert.True(t, Of().Map(too).IsEmpty())
	assert.Equal(t, 2, Of(1).Map(too).MustGet())

	tooint := func(val int) int {
		return val + 1
	}
	assert.True(t, Of().Map(tooint).IsEmpty())
	assert.Equal(t, 2, Of(1).Map(tooint).MustGet())

	tonp := func(val interface{}) interface{} {
		return nil
	}
	assert.True(t, Of(1).Map(tonp).IsEmpty())

	toz := func(val interface{}) interface{} {
		return 0
	}
	assert.False(t, Of(1).Map(toz).IsEmpty())
	assert.True(t, Of(1).Map(toz, ZeroValueIsEmpty).IsEmpty())
}

func TestOptionalFlatMap(t *testing.T) {
	too := func(val interface{}) Optional {
		return Of(val.(int) + 1)
	}
	assert.True(t, Of().FlatMap(too).IsEmpty())
	assert.Equal(t, 2, Of(1).FlatMap(too).MustGet())

	tooopt := func(val interface{}) Optional {
		return Of(val.(int) + 1)
	}
	assert.True(t, Of().FlatMap(tooopt).IsEmpty())
	assert.Equal(t, 2, Of(1).FlatMap(tooopt).MustGet())

	tonp := func(val interface{}) Optional {
		return Of()
	}
	assert.True(t, Of(1).FlatMap(tonp).IsEmpty())

	toz := func(val interface{}) Optional {
		return Of()
	}
	assert.True(t, Of(1).FlatMap(toz).IsEmpty())
}

func TestOptionalOrElseGetPanic(t *testing.T) {
	f := func() interface{} { return 2 }
	assert.Equal(t, 1, Of().OrElse(1))
	assert.Equal(t, 2, Of().OrElseGet(f))

	ft := func() int { return 2 }
	assert.Equal(t, 1, Of().OrElse(1))
	assert.Equal(t, 2, Of().OrElseGet(ft))

	errf := func() string { return "" }
	func() {
		defer func() {
			assert.True(t, "" == recover())
		}()
		Of().OrElsePanic(errf)
		assert.Fail(t, "Expected Panic")
	}()

	assert.Equal(t, 3, Of(3).OrElse(1))
	assert.Equal(t, 3, Of(3).OrElseGet(f))
	assert.Equal(t, 3, Of(3).OrElsePanic(errf))
}

func TestOptionalScan(t *testing.T) {
	var opt Optional
	assert.Nil(t, opt.Scan(0))
	assert.Equal(t, 0, opt.MustGet())
	assert.Nil(t, opt.Scan(1))
	assert.Equal(t, 1, opt.MustGet())

	sc := (sql.Scanner)(&opt)
	assert.NotNil(t, &sc)
}

func TestOptionalValue(t *testing.T) {
	val, err := Of().Value()
	assert.Nil(t, val)
	assert.Nil(t, err)

	val, err = Of(0).Value()
	assert.Equal(t, 0, val)
	assert.Nil(t, err)
}

type OptionalT int

func (t OptionalT) String() string {
	return fmt.Sprintf("%d", t+1)
}

func TestOptionalString(t *testing.T) {
	assert.Equal(t, emptyString, fmt.Sprintf("%s", Of()))
	assert.Equal(t, "Optional (1)", fmt.Sprintf("%s", Of(1)))
	assert.Equal(t, "Optional (2)", fmt.Sprintf("%s", Of(OptionalT(1))))
}
