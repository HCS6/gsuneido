// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package meta

import "math/bits"

type InfoHamt struct {
	root       *nodeInfo
	mutable    bool
	generation uint32 // if mutable, nodes with this generation are mutable
}

type nodeInfo struct {
	generation uint32
	bmVal      uint32
	bmPtr      uint32
	vals       []Info
	ptrs       []*nodeInfo
}

const bitsPerInfoNode = 5
const maskInfo = 1<<bitsPerInfoNode - 1

func (ht InfoHamt) IsNil() bool {
	return ht.root == nil
}

func (ht InfoHamt) MustGet(key string) Info {
	it, ok := ht.Get(key)
	if !ok {
		panic("Hamt MustGet failed")
	}
	return it
}

func (ht InfoHamt) GetPtr(key string) *Info {
	if !ht.mutable {
		panic("can't modify an immutable Hamt")
	}
	return ht.get(key)
}

func (ht InfoHamt) Get(key string) (Info, bool) {
	it := ht.get(key)
	if it == nil {
		var zero Info
		return zero, false
	}
	return *it, true
}

func (ht InfoHamt) get(key string) *Info {
	nd := ht.root
	if nd == nil {
		return nil
	}
	hash := InfoHash(key)
	for shift := 0; shift < 32; shift += bitsPerInfoNode { // iterative
		bit := nd.bit(hash, shift)
		iv := bits.OnesCount32(nd.bmVal & (bit - 1))
		if (nd.bmVal & bit) != 0 {
			if nd.vals[iv].Key() != key {
				return nil
			}
			return &nd.vals[iv]
		}
		if (nd.bmPtr & bit) == 0 {
			return nil
		}
		ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
		nd = nd.ptrs[ip]
	}
	// overflow node, linear search
	for i := range nd.vals {
		if nd.vals[i].Key() == key {
			return &nd.vals[i]
		}
	}
	return nil // not found
}

func (*nodeInfo) bit(hash uint32, shift int) uint32 {
	return 1 << ((hash >> shift) & maskInfo)
}

//-------------------------------------------------------------------

func (ht InfoHamt) Mutable() InfoHamt {
	gen := ht.generation + 1
	nd := ht.root
	if nd == nil {
		nd = &nodeInfo{generation: gen}
	}
	nd = nd.dup()
	nd.generation = gen
	return InfoHamt{root: nd, mutable: true, generation: gen}
}

func (ht InfoHamt) Put(item *Info) {
	if !ht.mutable {
		panic("can't modify an immutable Hamt")
	}
	key := item.Key()
	hash := InfoHash(key)
	ht.root.with(ht.generation, item, key, hash, 0)
}

func (nd *nodeInfo) with(gen uint32, item *Info, key string, hash uint32, shift int) *nodeInfo {
	// recursive
	if nd.generation != gen {
		// path copy on the way down the tree
		nd = nd.dup()
		nd.generation = gen // now mutable in this generation
	}
	if shift >= 32 {
		// overflow node
		for i := range nd.vals { // linear search
			if nd.vals[i].Key() == key {
				nd.vals[i] = *item // update if found
				return nd
			}
		}
		nd.vals = append(nd.vals, *item) // not found, add it
		return nd
	}
	bit := nd.bit(hash, shift)
	ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
	if (nd.bmPtr & bit) != 0 {
		// recurse to child node
		nd.ptrs[ip] = nd.ptrs[ip].with(gen, item, key, hash, shift+bitsPerInfoNode)
		return nd
	}
	iv := bits.OnesCount32(nd.bmVal & (bit - 1))
	if (nd.bmVal & bit) == 0 {
		// slot is empty, insert new value
		nd.bmVal |= bit
		var zero Info
		nd.vals = append(nd.vals, zero)
		copy(nd.vals[iv+1:], nd.vals[iv:])
		nd.vals[iv] = *item
		return nd
	}
	if nd.vals[iv].Key() == key {
		// already exists, update it
		nd.vals[iv] = *item
		return nd
	}
	// collision, create new child node
	nu := &nodeInfo{generation: gen}
	if shift+bitsPerInfoNode < 32 {
		oldval := &nd.vals[iv]
		oldkey := oldval.Key()
		nu = nu.with(gen, oldval, oldkey, InfoHash(oldkey), shift+bitsPerInfoNode)
		nu = nu.with(gen, item, key, hash, shift+bitsPerInfoNode)
	} else {
		// overflow node, no bitmaps, just list values
		nu.vals = append(nu.vals, nd.vals[iv], *item)
	}

	// remove old colliding value from node
	nd.bmVal &^= bit
	copy(nd.vals[iv:], nd.vals[iv+1:])
	nd.vals = nd.vals[:len(nd.vals)-1]

	// point to new child node instead
	nd.ptrs = append(nd.ptrs, nil)
	copy(nd.ptrs[ip+1:], nd.ptrs[ip:])
	nd.ptrs[ip] = nu
	nd.bmPtr |= bit

	return nd
}

func (nd *nodeInfo) dup() *nodeInfo {
	dup := *nd // shallow copy
	dup.vals = append(nd.vals[0:0:0], nd.vals...)
	dup.ptrs = append(nd.ptrs[0:0:0], nd.ptrs...)
	return &dup
}

func (ht InfoHamt) Freeze() InfoHamt {
	return InfoHamt{root: ht.root, generation: ht.generation}
}

//-------------------------------------------------------------------

// Delete removes an item.
// Empty nodes are removed, but single entry nodes are node pulled up.
func (ht InfoHamt) Delete(key string) bool {
	if !ht.mutable {
		panic("can't modify an immutable Hamt")
	}
	hash := InfoHash(key)
	_, ok := ht.root.without(ht.generation, key, hash, 0)
	return ok
}

func (nd *nodeInfo) without(gen uint32, key string, hash uint32, shift int) (*nodeInfo, bool) {
	// recursive
	if nd.generation != gen {
		// path copy on the way down the tree
		nd = nd.dup()
		nd.generation = gen // now mutable in this generation
	}
	if shift >= 32 {
		// overflow node
		for i := range nd.vals { // linear search
			if nd.vals[i].Key() == key {
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
		if nd.vals[iv].Key() == key {
			nd.bmVal &^= bit
			nd.vals = append(nd.vals[:iv], nd.vals[iv+1:]...) // preserve order
			if nd.bmVal == 0 && nd.bmPtr == 0 {               // node emptied
				nd = nil
			}
			return nd, true
		}
		return nd, false
	}
	if (nd.bmPtr & bit) == 0 {
		return nd, false
	}
	ip := bits.OnesCount32(nd.bmPtr & (bit - 1))
	nu, ok := nd.ptrs[ip].without(gen, key, hash, shift+bitsPerInfoNode) // recurse
	if nu != nil {
		nd.ptrs[ip] = nu
	} else { // child emptied
		nd.bmPtr &^= bit
		nd.ptrs = append(nd.ptrs[:ip], nd.ptrs[ip+1:]...) // preserve order
		if nd.bmPtr == 0 && nd.bmVal == 0 {               // this node emptied
			nd = nil
		}
	}
	return nd, ok
}

//-------------------------------------------------------------------

func (ht InfoHamt) ForEach(fn func(*Info)) {
	if ht.root != nil {
		ht.root.forEach(fn)
	}
}

func (nd *nodeInfo) forEach(fn func(*Info)) {
	for i := range nd.vals {
		fn(&nd.vals[i])
	}
	for _, p := range nd.ptrs {
		p.forEach(fn)
	}
}
