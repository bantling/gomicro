// SPDX-License-Identifier: Apache-2.0

package stream

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapToStruct(t *testing.T) {
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
