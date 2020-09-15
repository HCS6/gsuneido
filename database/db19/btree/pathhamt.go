// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package btree

import (
	"math/bits"

	"github.com/apmckinlay/gsuneido/util/assert"
)

type PathHamt struct {
	root       *nodePath
	mutable    bool
	generation uint32 // if mutable, nodes with this generation are mutable
}

type nodePath struct {
	generation uint32
	bmVal      uint32
	bmPtr      uint32
	vals       []path
	ptrs       []*nodePath
}

const bitsPerPathNode = 5
const maskPath = 1<<bitsPerPathNode - 1

func (ht PathHamt) IsNil() bool {
	return ht.root == nil
}

func (ht PathHamt) MustGet(key uint64) path {
	it, ok := ht.Get(key)
	if !ok {
		panic("Hamt MustGet failed")
	}
	return it
}

func (ht PathHamt) Get(key uint64) (path, bool) {
	it := ht.get(key)
	if it == nil {
		var zero path
		return zero, false
	}
	return *it, true
}

func (ht PathHamt) get(key uint64) *path {
	nd := ht.root
	if nd == nil {
		return nil
	}
	hash := PathHash(key)
	for shift := 0; shift < 32; shift += bitsPerPathNode { // iterative
		bit := nd.bit(hash, shift)
		iv := bits.OnesCount32(nd.bmVal & (bit - 1))
		if (nd.bmVal & bit) != 0 {
			if PathKey(nd.vals[iv]) == key {
				return &nd.vals[iv]
			}
		}
		if (nd.bmPtr & bit) == 0 {
			return nil
		}
		ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
		nd = nd.ptrs[ip]
	}
	// overflow node, linear search
	for i := range nd.vals {
		if PathKey(nd.vals[i]) == key {
			return &nd.vals[i]
		}
	}
	return nil // not found
}

func (*nodePath) bit(hash uint32, shift int) uint32 {
	return 1 << ((hash >> shift) & maskPath)
}

//-------------------------------------------------------------------

func (ht PathHamt) Mutable() PathHamt {
	gen := ht.generation + 1
	nd := ht.root
	if nd == nil {
		nd = &nodePath{generation: gen}
	}
	nd = nd.dup()
	nd.generation = gen
	return PathHamt{root: nd, mutable: true, generation: gen}
}

func (ht PathHamt) Put(item path) {
	if !ht.mutable {
		panic("can't modify an immutable Hamt")
	}
	key := PathKey(item)
	hash := PathHash(key)
	ht.root.with(ht.generation, item, key, hash, 0)
}

func (nd *nodePath) with(gen uint32, item path, key uint64, hash uint32, shift int) *nodePath {
	// recursive
	if nd.generation != gen {
		// path copy on the way down the tree
		nd = nd.dup()
		nd.generation = gen // now mutable in this generation
	}
	if shift >= 32 {
		// overflow node
		for i := range nd.vals { // linear search
			if PathKey(nd.vals[i]) == key {
				nd.vals[i] = item // update if found
				return nd
			}
		}
		nd.vals = append(nd.vals, item) // not found, add it
		return nd
	}
	bit := nd.bit(hash, shift)
	iv := bits.OnesCount32(nd.bmVal & (bit - 1))
	if (nd.bmVal & bit) == 0 {
		// slot is empty, insert new value
		nd.bmVal |= bit
		var zero path
		nd.vals = append(nd.vals, zero)
		copy(nd.vals[iv+1:], nd.vals[iv:])
		nd.vals[iv] = item
		return nd
	}
	if PathKey(nd.vals[iv]) == key {
		// already exists, update it
		nd.vals[iv] = item
		return nd
	}

	ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
	if (nd.bmPtr & bit) != 0 {
		// recurse to child node
		nd.ptrs[ip] = nd.ptrs[ip].with(gen, item, key, hash, shift+bitsPerPathNode)
		return nd
	}
	// collision, push new value down to new child node
	child := &nodePath{generation: gen}
	child = child.with(gen, item, key, hash, shift+bitsPerPathNode)

	// point to new child node
	nd.ptrs = append(nd.ptrs, nil)
	copy(nd.ptrs[ip+1:], nd.ptrs[ip:])
	nd.ptrs[ip] = child
	nd.bmPtr |= bit

	return nd
}

func (nd *nodePath) dup() *nodePath {
	dup := *nd // shallow copy
	dup.vals = append(nd.vals[0:0:0], nd.vals...)
	dup.ptrs = append(nd.ptrs[0:0:0], nd.ptrs...)
	return &dup
}

func (ht PathHamt) Freeze() PathHamt {
	return PathHamt{root: ht.root, generation: ht.generation}
}

//-------------------------------------------------------------------

// Delete removes an item.
func (ht PathHamt) Delete(key uint64) bool {
	if !ht.mutable {
		panic("can't modify an immutable Hamt")
	}
	hash := PathHash(key)
	_, ok := ht.root.without(ht.generation, key, hash, 0)
	return ok
}

func (nd *nodePath) without(gen uint32, key uint64, hash uint32, shift int) (*nodePath, bool) {
	// recursive
	if nd.generation != gen {
		// path copy on the way down the tree
		nd = nd.dup()
		nd.generation = gen // now mutable in this generation
	}
	if shift >= 32 {
		// overflow node
		for i := range nd.vals { // linear search
			if PathKey(nd.vals[i]) == key {
				nd.vals[i] = nd.vals[len(nd.vals)-1]
				nd.vals = nd.vals[:len(nd.vals)-1]
				if len(nd.vals) == 0 { // node emptied
					nd = nil
				}
				return nd, true
			}
		}
		return nd, false
	}
	bit := nd.bit(hash, shift)
	iv := bits.OnesCount32(nd.bmVal & (bit - 1))
	if (nd.bmVal & bit) != 0 {
		if PathKey(nd.vals[iv]) == key {
			// found it
			if (nd.bmPtr & bit) == 0 { // no child
				nd.bmVal &^= bit
				nd.vals = append(nd.vals[:iv], nd.vals[iv+1:]...) // preserve order
				if nd.bmVal == 0 && nd.bmPtr == 0 {               // node emptied
					nd = nil
				}
			} else {
				// pull up child value
				ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
				child, item := nd.ptrs[ip].pullUp(gen)
				nd.vals[iv] = item // replace the item
				if child != nil {
					nd.ptrs[ip] = child
				} else { // child emptied
					nd.bmPtr &^= bit
					nd.ptrs = append(nd.ptrs[:ip], nd.ptrs[ip+1:]...) // preserve order
				}
			}
			return nd, true
		}
	}
	if (nd.bmPtr & bit) == 0 {
		return nd, false
	}
	ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
	child, ok := nd.ptrs[ip].without(gen, key, hash, shift+bitsPerPathNode) // RECURSE
	if child != nil {
		nd.ptrs[ip] = child
	} else { // child emptied
		nd.bmPtr &^= bit
		nd.ptrs = append(nd.ptrs[:ip], nd.ptrs[ip+1:]...) // preserve order
	}
	return nd, ok
}

func (nd *nodePath) pullUp(gen uint32) (*nodePath, path) {
	// recursive
	if nd.generation != gen {
		// path copy on the way down the tree
		nd = nd.dup()
		nd.generation = gen // now mutable in this generation
	}
	if nd.bmPtr != 0 { // have children
		assert.That(nd.bmVal != 0)
		ip := len(nd.ptrs) - 1
		child, item := nd.ptrs[ip].pullUp(gen) // RECURSE
		if child != nil {
			nd.ptrs[ip] = child
		} else {
			nd.ptrs = nd.ptrs[:ip] // drop empty child node
			// clear highest one bit
			nd.bmPtr = nd.clearHighestOneBit(nd.bmPtr)
		}
		return nd, item
	}
	// no children
	iv := len(nd.vals) - 1
	item := nd.vals[iv]
	if iv == 0 { // last value in node
		return nil, item
	}
	nd.vals = nd.vals[:iv]
	if nd.bmVal != 0 { // not an overflow node
		// clear highest one bit
		nd.bmVal = nd.clearHighestOneBit(nd.bmVal)
	}
	return nd, item
}

func (*nodePath) clearHighestOneBit(n uint32) uint32 {
	return n &^ (1 << (31 - bits.LeadingZeros32(n)))
}

//-------------------------------------------------------------------

func (ht PathHamt) ForEach(fn func(path)) {
	if ht.root != nil {
		ht.root.forEach(fn)
	}
}

func (nd *nodePath) forEach(fn func(path)) {
	for i := range nd.vals {
		fn(nd.vals[i])
	}
	for _, p := range nd.ptrs {
		p.forEach(fn)
	}
}
