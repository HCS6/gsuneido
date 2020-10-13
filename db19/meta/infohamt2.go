// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package meta

import (
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/cksum"
)

// list returns a list of the keys in the table
func (ht InfoHamt) list() []string {
	keys := make([]string, 0, 16)
	ht.ForEach(func(it *Info) {
		keys = append(keys, InfoKey(it))
	})
	return keys
}

func (ht InfoHamt) Write(st *stor.Stor) uint64 {
	size := 0
	ht.ForEach(func(it *Info) {
		size += it.storSize()
	})
	if size == 0 {
		return 0
	}
	size += 3 + cksum.Len
	off, buf := st.Alloc(size)
	w := stor.NewWriter(buf)
	w.Put3(size)
	ht.ForEach(func(it *Info) {
		it.Write(w)
	})
	assert.That(w.Len() == size-cksum.Len)
	cksum.Update(buf)
	return off
}

func ReadInfoHamt(st *stor.Stor, off uint64) InfoHamt {
	if off == 0 {
		return InfoHamt{}
	}
	buf := st.Data(off)
	size := stor.NewReader(buf).Get3()
	cksum.MustCheck(buf[:size])
	r := stor.NewReader(buf[3 : size-cksum.Len])
	ht := InfoHamt{}.Mutable()
	for r.Remaining() > 0 {
		ht.Put(ReadInfo(st, r))
	}
	return ht.Freeze()
}
