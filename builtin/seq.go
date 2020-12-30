// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package builtin

import (
	"math"

	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
)

var _ = builtin1("Seq?(value)", func(val Value) Value {
	_, ok := val.(*SuSequence)
	return SuBool(ok)
})

var _ = builtin3("Seq(from=false, to=false, by=1)",
	func(from, to, by Value) Value {
		if from == False {
			from = Zero
			to = MaxInt
		} else if to == False {
			to = from
			from = Zero
		}
		f := ToInt(from)
		return NewSuSequence(
			&seqIter{from: f, to: ToInt(to), by: ToInt(by), i: f})
	})

type seqIter struct {
	MayLock
	from int
	to   int
	by   int
	i    int
}

func (seq *seqIter) Next() Value {
	if seq.Lock() {
		defer seq.Unlock()
	}
	assert.That(seq.by != 0)
	if seq.i >= seq.to {
		return nil
	}
	i := seq.i
	seq.i += seq.by
	return IntVal(i)
}

func (seq *seqIter) Dup() Iter {
	if seq.Lock() {
		defer seq.Unlock()
	}
	return &seqIter{from: seq.from, to: seq.to, by: seq.by, i: seq.from}
}

func (seq *seqIter) Infinite() bool {
	// to is read-only so no locking required
	return seq.to == math.MaxInt32 // has to match MaxInt opcode
}

func (seq *seqIter) SetConcurrent() {
	seq.SetConcurrentFlag()
}
