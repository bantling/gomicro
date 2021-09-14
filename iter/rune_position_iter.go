// SPDX-License-Identifier: Apache-2.0

package iter

import (
	"io"
)

// RunePositionIter tracks the line number and rune position while reading UTF8 runes of an io.Reader.
// Tracks the first byte of multi byte runes.
// LineNumberIter is an Iterable but not an Iter, since it only iterates runes.
// When a CR, LF, or CRLF sequence is read, it is returned as a single LF to simplify EOL handling.
type RunePositionIter struct {
	iter           *Iter
	lastChar       rune
	lastCR         bool
	lastReadWasEOF bool
	line           int
	position       int
}

// NewRunePositionIter constructs a new RunePositionIter from an io.Reader
func NewRunePositionIter(src io.Reader) *RunePositionIter {
	return &RunePositionIter{
		iter:           OfReaderRunes(src),
		lastChar:       0,
		lastReadWasEOF: false,
		line:           1,
		position:       1,
	}
}

// Next returns true if there is another rune to be read by Value.
// Once Next returns false, all further calls to Next return false.
func (rp *RunePositionIter) Next() bool {
	if rp.iter == nil {
		return false
	}

	var next bool

	if rp.lastReadWasEOF {
		// Last time we read a CR, peeked ahead for an LF and encountered EOF.
		// We can't call next or a panic will occur.
		// Nullify iter and clear flag so that if caller calls next again, we call next and panic as iters should.
		rp.iter = nil
		rp.lastReadWasEOF = false
		return false
	}

	if next = rp.iter.Next(); next {
		// Get next char and handle EOL any sequence, if present
		rp.lastChar = rp.iter.RuneValue()

		switch rp.lastChar {
		case '\r':
			// Increase line and flag it
			rp.line++
			rp.position = 1

			// If it is a CRLF, consume the LF
			if rp.iter.Next() {
				if peek := rp.iter.RuneValue(); peek != '\n' {
					// Just a CR, unread this second char
					rp.iter.Unread(peek)
				}
			} else {
				// Unable to peek at next char because there is no next char.
				// Flag this condition for next call.
				rp.lastReadWasEOF = true
			}

			// Change char to an LF to collapse CR and CRLF into LF to simplify EOL handling for caller
			rp.lastChar = '\n'

		case '\n':
			rp.line++
			rp.position = 1

		default:
			// Increment position in line - since EOLs reset to 0, it will always be >= 1 for non-EOL chars
			rp.position++
		}
	}

	return next
}

// Value returns the rune retrieved by the prior call to Next.
// All EOL sequences are translated into a single newline for simplicity.
func (rp *RunePositionIter) Value() rune {
	if rp.iter == nil {
		panic(ErrValueExhaustedIter)
	}

	if rp.lastChar == 0 {
		panic(ErrValueNextFirst)
	}

	result := rp.lastChar
	rp.lastChar = 0
	return result
}

// Unread unreads the given character
func (rp *RunePositionIter) Unread(char rune) {
	rp.iter.Unread(char)
}

// Line returns the current line number, starting at 1
func (rp *RunePositionIter) Line() int {
	return rp.line
}

// Position returns the position on the current line, starting at 1
func (rp *RunePositionIter) Position() int {
	return rp.position
}

// Iter is Iterable interface
func (rp *RunePositionIter) Iter() *Iter {
	return New(
		func() (interface{}, bool) {
			if rp.Next() {
				return rp.Value(), true
			}

			return nil, false
		},
	)
}
