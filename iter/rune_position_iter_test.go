// SPDX-License-Identifier: Apache-2.0

package iter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunePositionIter(t *testing.T) {
	// Test position and line
	var (
		text        = "line 1\rline 2\nline3\r\nline44"
		lines       = []string{"line 1", "line 2", "line3", "line44"}
		iter        = NewRunePositionIter(strings.NewReader(text))
		char        rune
		lineNum     = 1
		lastCharPos = 1
	)

	var lineText strings.Builder
	for iter.Next() {
		if char = iter.Value(); char == '\n' {
			assert.Equal(t, lines[lineNum-1], lineText.String())
			assert.Equal(t, len(lines[lineNum-1])+1, lastCharPos)
			lineNum++
			assert.Equal(t, lineNum, iter.Line())

			lineText.Reset()
		} else {
			lineText.WriteRune(char)
			lastCharPos = iter.Position()
		}
	}

	assert.Equal(t, len(lines), lineNum)
	assert.Equal(t, len(lines), iter.Line())
	assert.Equal(t, len(lines[len(lines)-1])+1, iter.Position())

	// Test unread
	iter = NewRunePositionIter(strings.NewReader("a"))
	assert.True(t, iter.Next())
	assert.Equal(t, 'a', iter.Value())

	iter.Unread('a')
	assert.True(t, iter.Next())
	assert.Equal(t, 'a', iter.Value())

	assert.False(t, iter.Next())

	// Test just one space and cr
	iter = NewRunePositionIter(strings.NewReader(" \r"))
	assert.True(t, iter.Next())
	assert.Equal(t, ' ', iter.Value())
	assert.True(t, iter.Next())
	assert.Equal(t, '\n', iter.Value())
	assert.Equal(t, 2, iter.Line())
	assert.Equal(t, 1, iter.Position())

	assert.False(t, iter.Next())

	// Panics if we call value again
	func() {
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}()

	assert.False(t, iter.Next())

	// Corner case of ending with a CR
	iter = NewRunePositionIter(strings.NewReader("\r"))
	assert.True(t, iter.Next())
	assert.Equal(t, '\n', iter.Value())
	assert.Equal(t, 2, iter.Line())
	assert.Equal(t, 1, iter.Position())

	assert.False(t, iter.Next())

	// Panics if we call value again
	func() {
		defer func() {
			assert.Equal(t, ErrValueExhaustedIter, recover())
		}()

		iter.Value()
		assert.Fail(t, "Must panic")
	}()

	assert.False(t, iter.Next())
}
