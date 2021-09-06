// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"encoding/json"
	"testing"

	"github.com/bantling/gomicro/iter"
	"github.com/stretchr/testify/assert"
)

// ==== Compose

func TestComposeGenerators(t *testing.T) {
	var (
		f1 = func() func(it *iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); val.(int) < 5 {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		}

		f2 = func() func(it *iter.Iter) *iter.Iter {
			return func(it *iter.Iter) *iter.Iter {
				return iter.NewIter(
					func() (interface{}, bool) {
						for it.Next() {
							if val := it.Value(); val.(int) > 0 {
								return val, true
							}
						}

						return nil, false
					},
				)
			}
		}

		c  = composeGenerators(f1, f2)
		it = iter.Of(0, 1, 2, 3, 4, 5)
	)

	assert.Equal(t, []int{1, 2, 3, 4}, c()(it).ToSliceOf(0))
}

// ==== ToJSON

func TestToJSON(t *testing.T) {
	{
		goodDocs := []interface{}{
			[]byte(`[]a`), []interface{}{},
			[]byte(`[1]a`), []interface{}{json.Number("1")},
			[]byte(`{"foo": true, "bar": ["baz"]}a`), map[string]interface{}{"foo": true, "bar": []interface{}{"baz"}},
		}

		for i := 0; i < len(goodDocs); i += 2 {
			input := goodDocs[i]
			it1 := iter.OfElements(input)
			it2 := iter.NewIter(ToJSON(it1))

			assert.Equal(t, goodDocs[i+1], it2.NextValue())
			assert.Equal(t, byte('a'), it1.NextValue())
			assert.False(t, it1.Next())
		}
	}

	{
		goodDocs := []interface{}{
			[]byte(`[][1]a`), []interface{}{
				[]interface{}{},
				[]interface{}{json.Number("1")},
			},
			[]byte(`[1][2]a`), []interface{}{
				[]interface{}{json.Number("1")},
				[]interface{}{json.Number("2")},
			},
			[]byte(`[1,2]{"foo": null, "bar": {"baz": "taz"}}[4]a`), []interface{}{
				[]interface{}{json.Number("1"), json.Number("2")},
				map[string]interface{}{
					"foo": nil,
					"bar": map[string]interface{}{"baz": "taz"},
				},
				[]interface{}{json.Number("4")},
			},
		}

		for i := 0; i < len(goodDocs); i += 2 {
			var (
				input = goodDocs[i]
				it1   = iter.OfElements(input)
				it2   = iter.NewIter(ToJSON(it1))
			)

			for _, expected := range goodDocs[i+1].([]interface{}) {
				assert.Equal(t, expected, it2.NextValue())
			}
			assert.Equal(t, byte('a'), it1.NextValue())
			assert.False(t, it1.Next())
		}
	}

	{
		badDocs := []interface{}{
			[]byte(`[`),
			[]byte(`]`),
			[]byte(`{`),
			[]byte(`}`),
			[]byte(`[{]`),
			[]byte(`{]}`),
			[]byte(`[[}`),
			[]byte(`{{]`),
		}

		for _, input := range badDocs {
			var (
				it1 = iter.OfElements(input)
				it2 = iter.NewIter(ToJSON(it1))
			)

			{
				defer func() {
					assert.Equal(t, ErrInvalidJSONDocument, recover())
				}()

				it2.NextValue()
				assert.Fail(t, "Must panic")
			}
		}
	}
}

// ==== FromArraySlice

func TestFromArraySlice(t *testing.T) {
	{
		// Empty
		var (
			it1 = iter.Of()
			fn  = FromArraySlice(it1)
			it2 = iter.NewIter(fn)
		)
		assert.Equal(t, []interface{}{}, it2.ToSlice())
	}

	{
		// array of 1 element
		var (
			it1 = iter.Of([1]int{1})
			fn  = FromArraySlice(it1)
			it2 = iter.NewIter(fn)
		)
		assert.Equal(t, []interface{}{1}, it2.ToSlice())
	}

	{
		// slice of 2 elements
		var (
			it1 = iter.Of([]int{1, 2})
			fn  = FromArraySlice(it1)
			it2 = iter.NewIter(fn)
		)
		assert.Equal(t, []interface{}{1, 2}, it2.ToSlice())
	}

	{
		// array of 1 element and slice of 2 elements
		var (
			it1 = iter.Of([1]int{1}, []int{2, 3})
			fn  = FromArraySlice(it1)
			it2 = iter.NewIter(fn)
		)
		assert.Equal(t, []interface{}{1, 2, 3}, it2.ToSlice())
	}
}

// ==== SetMap

//func TestSetMap(t *testing.T) {
//	var (
//		fin = New().AndThen().SetMap(ToJSON)
//		it  = iter.OfElements([]byte(`[1,2,3]`))
//	)
//
//	assert.Equal(
//		t,
//		[]interface{}{
//			[]interface{}{float64(1), float64(2), float64(3)},
//		},
//		fin.ToSlice(it),
//	)
//	assert.False(t, it.Next())
//
//	it = iter.OfElements([]byte(`[1,2,3][4,5,6]`))
//	finit := fin.Iter(it)
//	assert.True(t, finit.Next())
//	assert.Equal(t, []interface{}{float64(1), float64(2), float64(3)}, finit.Value())
//	assert.Equal(t, byte('['), it.NextValue())
//
//	assert.Equal(
//		t,
//		[]interface{}{
//			[]interface{}{float64(1), float64(2), float64(3)},
//			[]interface{}{float64(4), float64(5), float64(6)},
//		},
//		fin.ToSlice(),
//	)
//
//	fin = fin.SetMap(FromArraySlice)
//
//	assert.Equal(
//		t,
//		[]interface{}{float64(4), float64(5), float64(6)},
//		fin.ToSlice(iter.OfElements([]byte(`[4,5,6]`))),
//	)
//
//	assert.Equal(
//		t,
//		[]interface{}{float64(1), float64(2), float64(3), float64(4), float64(5), float64(6)},
//		fin.ToSlice(iter.OfElements([]byte(`[1,2,3][4,5,6]`))),
//	)
//}
