// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"encoding/json"
	"math/big"
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

func TestJSONNumberConversions(t *testing.T) {
	assert.Equal(t, json.Number("0"), JSONNumberToNumber(json.Number("0")))
	assert.Equal(t, int64(1), JSONNumberToInt64(json.Number("1")))
	assert.Equal(t, uint64(2), JSONNumberToUint64(json.Number("2")))
	assert.Equal(t, float64(3.5), JSONNumberToFloat64(json.Number("3.5")))
	assert.Equal(t, big.NewInt(4), JSONNumberToBigInt(json.Number("4")))
	assert.Equal(t, big.NewFloat(5.25), JSONNumberToBigFloat(json.Number("5.25")))
	assert.Equal(t, "6", JSONNumberToString(json.Number("6")))

	assert.Equal(t, json.Number("0"), JSONNumberConversion(JSONNumAsNumber)(json.Number("0")))
	assert.Equal(t, int64(1), JSONNumberConversion(JSONNumAsInt64)(json.Number("1")))
	assert.Equal(t, uint64(2), JSONNumberConversion(JSONNumAsUint64)(json.Number("2")))
	assert.Equal(t, float64(3.5), JSONNumberConversion(JSONNumAsFloat64)(json.Number("3.5")))
	assert.Equal(t, big.NewInt(4), JSONNumberConversion(JSONNumAsBigInt)(json.Number("4")))
	assert.Equal(t, big.NewFloat(5.25), JSONNumberConversion(JSONNumAsBigFloat)(json.Number("5.25")))
	assert.Equal(t, "6", JSONNumberConversion(JSONNumAsString)(json.Number("6")))
}

func TestToJSON(t *testing.T) {
	// single document, arrays or objects
	{
		goodDocs := []interface{}{
			[]byte(`[]a`), []interface{}{},
			[]byte(`[1]a`), []interface{}{json.Number("1")},
			[]byte(`{"foo": true, "bar": ["baz"]}a`), map[string]interface{}{"foo": true, "bar": []interface{}{"baz"}},
		}

		for i := 0; i < len(goodDocs); i += 2 {
			input := goodDocs[i]
			it1 := iter.OfElements(input)
			it2 := ToJSON()()(it1)

			assert.Equal(t, goodDocs[i+1], it2.NextValue())
			assert.Equal(t, byte('a'), it1.NextValue())
			assert.False(t, it1.Next())
		}
	}

	// two documents, arrays or objects
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
				it2   = ToJSON(JSONConfig{DocType: JSONArrayOrObject})()(it1)
			)

			for _, expected := range goodDocs[i+1].([]interface{}) {
				assert.Equal(t, expected, it2.NextValue())
			}
			assert.Equal(t, byte('a'), it1.NextValue())
			assert.False(t, it1.Next())
		}
	}

	// array only
	{
		var (
			input = []byte(`[1]a`)
			it1   = iter.OfElements(input)
			it2   = ToJSON(JSONConfig{DocType: JSONArray})()(it1)
		)

		assert.Equal(t, []interface{}{json.Number("1")}, it2.NextValue())
		assert.Equal(t, byte('a'), it1.NextValue())
		assert.False(t, it1.Next())
	}

	// object only
	{
		var (
			input = []byte(`{"foo": "bar"}a`)
			it1   = iter.OfElements(input)
			it2   = ToJSON(JSONConfig{DocType: JSONObject})()(it1)
		)

		assert.Equal(t, map[string]interface{}{"foo": "bar"}, it2.NextValue())
		assert.Equal(t, byte('a'), it1.NextValue())
		assert.False(t, it1.Next())
	}

	// Badly formed JSON fails
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
				it2 = ToJSON()()(it1)
			)

			func() {
				defer func() {
					assert.Equal(t, ErrInvalidJSONDocument, recover())
				}()

				it2.NextValue()
				assert.Fail(t, "Must panic")
			}()
		}
	}

	// Arrays only fails on objects
	{
		var (
			input = []byte(`{"foo":"bar"}`)
			it1   = iter.OfElements(input)
			it2   = ToJSON(JSONConfig{DocType: JSONArray})()(it1)
		)

		func() {
			defer func() {
				assert.Equal(t, ErrInvalidJSONArray, recover())
			}()

			it2.NextValue()
			assert.Fail(t, "Must panic")
		}()
	}

	// Objects only fails on arrays
	{
		var (
			input = []byte(`[1]`)
			it1   = iter.OfElements(input)
			it2   = ToJSON(JSONConfig{DocType: JSONObject})()(it1)
		)

		func() {
			defer func() {
				assert.Equal(t, ErrInvalidJSONObject, recover())
			}()

			it2.NextValue()
			assert.Fail(t, "Must panic")
		}()
	}
}

// ==== FromArraySlice

func TestFromArraySlice(t *testing.T) {
	{
		// Empty
		var (
			it1 = iter.Of()
			it2 = FromArraySlice()(it1)
		)
		assert.Equal(t, []interface{}{}, it2.ToSlice())
	}

	{
		// array of 1 element
		var (
			it1 = iter.Of([1]int{1})
			it2 = FromArraySlice()(it1)
		)
		assert.Equal(t, []interface{}{1}, it2.ToSlice())
	}

	{
		// slice of 2 elements
		var (
			it1 = iter.Of([]int{1, 2})
			it2 = FromArraySlice()(it1)
		)
		assert.Equal(t, []interface{}{1, 2}, it2.ToSlice())
	}

	{
		// array of 1 element and slice of 2 elements
		var (
			it1 = iter.Of([1]int{1}, []int{2, 3})
			it2 = FromArraySlice()(it1)
		)
		assert.Equal(t, []interface{}{1, 2, 3}, it2.ToSlice())
	}
}
