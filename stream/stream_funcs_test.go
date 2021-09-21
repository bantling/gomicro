// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapToStruct(t *testing.T) {
	// No decode hooks
	{
		type Person struct {
			FirstName string
			LastName  string
			Age       int
		}

		var (
			doc        = map[string]interface{}{"firstName": "John", "lastName": "Doe", "age": 56}
			person     = Person{FirstName: "John", LastName: "Doe", Age: 56}
			personPtr1 = &person
			personPtr2 = &personPtr1
			personPtr3 = &personPtr2
		)

		// Value
		assert.Equal(t, MapToStruct(Person{})(doc), person)

		// 1 Pointer
		assert.Equal(t, MapToStruct(&Person{})(doc), personPtr1)

		// 2 Pointers
		assert.Equal(t, MapToStruct(reflect.TypeOf((**Person)(nil)))(doc), personPtr2)

		// 3 Pointers
		assert.Equal(t, MapToStruct(reflect.TypeOf((***Person)(nil)))(doc), personPtr3)
	}

	// BoolString decode hook
	{
		type Person struct {
			FirstName string
			LastName  string
			Other     BoolString
		}

		var (
			docs = []map[string]interface{}{
				{"firstName": "John", "lastName": "Doe", "other": true},
				{"firstName": "John", "lastName": "Doe", "other": "REDACTED"},
				{"firstName": "John", "lastName": "Doe", "other": nil},
				{"firstName": "John", "lastName": "Doe"},
			}
			persons = []Person{
				{FirstName: "John", LastName: "Doe", Other: BoolString{IsMsg: false, Value: true, Msg: ""}},
				{FirstName: "John", LastName: "Doe", Other: BoolString{IsMsg: true, Msg: "REDACTED"}},
				{FirstName: "John", LastName: "Doe"},
				{FirstName: "John", LastName: "Doe"},
			}
		)

		for i, doc := range docs {
			assert.Equal(t, MapToStruct(Person{})(doc), persons[i])
		}
	}

	// IntString decode hook
	{
		type Person struct {
			FirstName string
			LastName  string
			Other     IntString
		}

		var (
			docs = []map[string]interface{}{
				{"firstName": "John", "lastName": "Doe", "other": 1},
				{"firstName": "John", "lastName": "Doe", "other": int8(2)},
				{"firstName": "John", "lastName": "Doe", "other": "REDACTED"},
				{"firstName": "John", "lastName": "Doe", "other": nil},
				{"firstName": "John", "lastName": "Doe"},
			}
			persons = []Person{
				{FirstName: "John", LastName: "Doe", Other: IntString{IsMsg: false, Value: 1, Msg: ""}},
				{FirstName: "John", LastName: "Doe", Other: IntString{IsMsg: false, Value: 2, Msg: ""}},
				{FirstName: "John", LastName: "Doe", Other: IntString{IsMsg: true, Msg: "REDACTED"}},
				{FirstName: "John", LastName: "Doe"},
				{FirstName: "John", LastName: "Doe"},
			}
		)

		for i, doc := range docs {
			assert.Equal(t, MapToStruct(Person{})(doc), persons[i])
		}
	}

	// UintString decode hook
	{
		type Person struct {
			FirstName string
			LastName  string
			Other     UintString
		}

		var (
			docs = []map[string]interface{}{
				{"firstName": "John", "lastName": "Doe", "other": uint(1)},
				{"firstName": "John", "lastName": "Doe", "other": uint8(2)},
				{"firstName": "John", "lastName": "Doe", "other": "REDACTED"},
				{"firstName": "John", "lastName": "Doe", "other": nil},
				{"firstName": "John", "lastName": "Doe"},
			}
			persons = []Person{
				{FirstName: "John", LastName: "Doe", Other: UintString{IsMsg: false, Value: 1, Msg: ""}},
				{FirstName: "John", LastName: "Doe", Other: UintString{IsMsg: false, Value: 2, Msg: ""}},
				{FirstName: "John", LastName: "Doe", Other: UintString{IsMsg: true, Msg: "REDACTED"}},
				{FirstName: "John", LastName: "Doe"},
				{FirstName: "John", LastName: "Doe"},
			}
		)

		for i, doc := range docs {
			assert.Equal(t, MapToStruct(Person{})(doc), persons[i])
		}
	}
}
