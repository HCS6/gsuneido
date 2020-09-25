// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package btree

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apmckinlay/gsuneido/db19/ixspec"
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/cksum"
)

// fbtree is an immutable btree designed to be stored in a file.
// Only an unshared copy with mutable=true can be updated.
type fbtree struct {
	// treeLevels is how many levels of tree nodes there are (initially 0)
	treeLevels int
	// root is the offset of the root node
	root uint64
	// store is where the btree is stored
	store *stor.Stor
	// redirs maps offsets to updated in-memory nodes (not persisted)
	redirs redirs
	// ixspec is an opaque value passed to GetLeafKey
	// normally it specifies which fields make up the key, based on the schema
	ixspec *ixspec.T
	// redirsOff is the offset of the saved redirections
	redirsOff uint64
	// mutable is true during updates
	mutable bool
}

// MaxNodeSize is the maximum node size in bytes, split if larger.
// Overridden by tests.
var MaxNodeSize = 1536 // * .75 ~ 1k

// GetLeafKey is used to get the key for a data offset.
// It is a dependency that must be injected
var GetLeafKey func(st *stor.Stor, is *ixspec.T, off uint64) string

func CreateFbtree(store *stor.Stor, is *ixspec.T) *fbtree {
	re := newRedirs()
	re.tbl = re.tbl.Mutable()
	root := re.add(fNode{})
	re.tbl = re.tbl.Freeze()
	re.generation++ // so root isn't mutable
	return &fbtree{root: root, redirs: re, store: store, ixspec: is}
}

func OpenFbtree(store *stor.Stor, root uint64, treeLevels int, redirsOff uint64) *fbtree {
	re := loadRedirs(store, redirsOff)
	return &fbtree{root: root, treeLevels: treeLevels, redirs: re, store: store,
		redirsOff: redirsOff}
}

func (fb *fbtree) getLeafKey(off uint64) string {
	return GetLeafKey(fb.store, fb.ixspec, off)
}

func (fb *fbtree) Search(key string) uint64 {
	off := fb.root
	for i := 0; i <= fb.treeLevels; i++ {
		node := fb.getNode(off)
		off, _, _ = node.search(key)
	}
	return off
}

//-------------------------------------------------------------------

// Save writes the btree (changes) to the stor
// and returns a new fbtree with no in-memory nodes (but still redirections)
func (fb *fbtree) Save() *fbtree {
	return fb.Update(func(mfb *fbtree) { mfb.save() })
}

const redirMax = 100 // ???

func (fb *fbtree) save() {
	fb.keep()
	if fb.redirCount() < redirMax {
		fb.saveRedirs()
	} else {
		trace("FLATTEN")
		fb.flatten()
	}
}

func (fb *fbtree) printPaths(s string) {
	fb.redirs.paths.ForEach(func(p path) {
		s += " " + OffStr(p)
	})
	fmt.Println(s)
}

func (fb *fbtree) redirCount() int {
	n := 0
	fb.redirs.tbl.ForEach(func(*redir) { n++ })
	return n
}

func (fb *fbtree) pathsCount() int {
	n := 0
	fb.redirs.paths.ForEach(func(uint64) { n++ })
	return n
}

// keep saves the in-memory nodes (like flatten) but keeps the redirects.
func (fb *fbtree) keep() {
	root := fb.keep2(0, fb.root)
	fb.redirs.tbl.Delete(fb.root)
	fb.root = root
}

// keep2 recursively traverses the modified branches of the fbtree.
// It visits in-memory mnodes and nodes in fb.redirs.paths.
func (fb *fbtree) keep2(depth int, nodeOff uint64) uint64 {
	r, ok := fb.redirs.tbl.Get(nodeOff)
	traced(depth, "keep", OffStr(nodeOff), "|", r)
	inPaths := false
	var mnode fNode
	if depth < fb.treeLevels {
		// tree node
		if ok && r.mnode != nil {
			mnode = r.mnode
		}
		inPaths = fb.pathNode(nodeOff)
		if mnode == nil && !inPaths {
			return nodeOff
		}
		traced(depth, "tree node")
		node := fb.getNode(nodeOff) // also handles redir
		// copy lazily, if current generation it's already a copy
		copied := mnode != nil && r.generation  == fb.redirs.generation
		for it := node.iter(); it.next(); {
			off := it.offset
			off2 := fb.keep2(depth+1, off) // RECURSE
			if off2 == off {
				continue
			}
			// child offset changed
			if mnode == nil {
				// can't update so just redirect
				fb.redirs.tbl.Put(&redir{offset: off, newOffset: off2})
			} else {
				// If we're on an mnode
				// then we can update it and delete the redir (flatten)
				traced(depth, "update tree", OffStr(off), "=>", off2)
				if !copied {
					traced(depth, "copy")
					copied = true
					mnode = append(mnode[:0:0], mnode...) // copy
				}
				mnode.setOffset(it.fi, off2)
				assert.That(fb.redirs.tbl.Delete(off)) // remove flattened redirect
				if fb.pathNode(off) {
					if !fb.redirs.isFake(off) {
						assert.That(fb.redirs.paths.Delete(off))
					}
					if depth+1 < fb.treeLevels {
						fb.addPath(off2)
					}
				}
			}
		}
		if mnode == nil {
			return nodeOff
		}
	} else {
		// leaf node
		if !ok { // not redirected
			return nodeOff
		}
		if r.mnode == nil {
			traced(depth, "leaf newOffset")
			return r.newOffset
		}
		mnode = r.mnode
		traced(depth, "leaf mnode")
	}
	// save mnode
	newOffset := mnode.putNode(fb.store)
	traced(depth, "putNode", OffStr(nodeOff), "=>", newOffset)
	return newOffset
}

func (fb *fbtree) pathNode(off uint64) bool {
	if fb.redirs.isFake(off) || off == fb.root {
		return true
	}
	_, ok := fb.redirs.paths.Get(off)
	return ok
}

//-------------------------------------------------------------------

// flatten saves in-memory nodes (like keep)
// and in addition it applies the redirects and then clears them
func (fb *fbtree) flatten() {
	fb.root = fb.flatten2(0, fb.root)
	fb.redirs = newRedirs()
	fb.redirsOff = 0
}

func (fb *fbtree) flatten2(depth int, nodeOff uint64) uint64 {
	traced(depth, "flatten", OffStr(nodeOff))
	var rwNode fNode
	if depth < fb.treeLevels {
		// tree node, need to update any redirected offsets
		roNode := fb.getNode(nodeOff)
		// delay making mutable copy until we need to update
		for it := roNode.iter(); it.next(); {
			off := it.offset
			// only traverse modified paths, not entire (possibly huge) tree
			if depth+1 == fb.treeLevels || fb.pathNode(off) {
				off2 := fb.flatten2(depth+1, off) // RECURSE
				// bottom up
				if off2 != off {
					if rwNode == nil {
						rwNode = fb.getMutableNode(nodeOff)
					}
					rwNode.setOffset(it.fi, off2)
				}
			} else {
				traced(depth+1, "skipped tree node", off)
			}
		}
		if rwNode == nil {
			if r, ok := fb.redirs.tbl.Get(nodeOff); ok && r.mnode != nil {
				traced(depth, "tree mnode")
				rwNode = r.mnode
			} else {
				if ok {
					traced(depth, "tree new offset", OffStr(r.newOffset))
					return r.newOffset
				}
				traced(depth, "tree NO SAVE")
				return nodeOff // nothing modified, don't need to save
			}
		} else {
			traced(depth, "tree node modified")
		}
	} else {
		// leaf node
		r, ok := fb.redirs.tbl.Get(nodeOff)
		if !ok {
			return nodeOff
		}
		if r.mnode == nil {
			traced(depth, "leaf newOffset")
			return r.newOffset
		}
		rwNode = r.mnode
		traced(depth, "leaf mnode")
	}
	off := rwNode.putNode(fb.store)
	traced(depth, "putNode", OffStr(nodeOff), "=>", off)
	return off
}

func trace(args ...interface{}) {
	// fmt.Println(args...)
}

func traced(depth int, args ...interface{}) {
	// fmt.Print(strings.Repeat("    ", depth))
	// fmt.Println(args...)
}

func (fb *fbtree) saveRedirs() {
	nr := 0
	fb.redirs.tbl.ForEach(func(*redir) { nr++ })
	np := 0
	fb.redirs.paths.ForEach(func(p uint64) { np++ })
	size := 2 + 5 + 2 + nr*10 + 2 + np*5 + cksum.Len
	off, buf := fb.store.Alloc(size)
	w := stor.NewWriter(buf)
	w.Put2(size)
	w.Put5(fb.redirs.nextOff)
	w.Put2(nr)
	fb.redirs.tbl.ForEach(func(r *redir) {
		assert.That(!fb.redirs.isFake(r.offset))
		assert.That(r.mnode == nil)
		assert.That(r.offset != 0 && r.newOffset != 0)
		w.Put5(r.offset).Put5(r.newOffset)
	})
	w.Put2(np)
	fb.redirs.paths.ForEach(func(p uint64) {
		assert.That(!fb.redirs.isFake(p))
		w.Put5(p)
	})
	assert.That(w.Len()+cksum.Len == size)
	cksum.Update(buf)
	fb.redirsOff = off
}

func loadRedirs(store *stor.Stor, redirsOff uint64) redirs {
	re := newRedirs()
	if redirsOff == 0 {
		return re
	}
	buf := store.Data(redirsOff)
	rdr := stor.NewReader(buf)
	size := rdr.Get2()
	cksum.MustCheck(buf[:size])
	re.nextOff = rdr.Get5()
	re.tbl = re.tbl.Mutable()
	for n := rdr.Get2(); n > 0; n-- {
		r := &redir{}
		r.offset = rdr.Get5()
		r.newOffset = rdr.Get5()
		re.tbl.Put(r)
	}
	re.tbl = re.tbl.Freeze()
	re.paths = re.paths.Mutable()
	for n := rdr.Get2(); n > 0; n-- {
		off := rdr.Get5()
		re.paths.Put(off)
	}
	re.paths = re.paths.Freeze()
	return re
}

// func init() {
// 	rand.Seed(time.Now().UnixNano())
// }

// putNode stores the node
func (node fNode) putNode(store *stor.Stor) uint64 {
	off := store.SaveSized(node)
	// if len(node) > 0 && rand.Intn(500) == 42 {
	// 	// corrupt some nodes to test checking
	// 	fmt.Println("ZAP")
	// 	buf := store.Data(off)
	// 	buf[3 + rand.Intn(len(node))] = byte(rand.Intn(256))
	// }
	return off
}

// getNode returns the node for a given offset using the redirects
func (fb *fbtree) getNode(off uint64) fNode {
	if r, ok := fb.redirs.tbl.Get(off); ok {
		assert.That((r.mnode == nil) != (r.newOffset == 0))
		if r.mnode != nil {
			return r.mnode
		}
		off = r.newOffset
	}
	return fb.readNode(off)
}

func (fb *fbtree) getCkNode(off uint64) fNode {
	if r, ok := fb.redirs.tbl.Get(off); ok {
		assert.That((r.mnode == nil) != (r.newOffset == 0))
		if r.mnode != nil {
			return r.mnode
		}
		off = r.newOffset
	}
	buf := fb.store.Data(off)
	rn := runtime.RecLen(buf)
	cksum.MustCheck(buf[:rn+cksum.Len])
	return fb.readNode(off)
}

func (fb *fbtree) readNode(off uint64) fNode {
	assert.That(!fb.redirs.isFake(off))
	buf := fb.store.DataSized(off)
	return fNode(buf)
}

//-------------------------------------------------------------------

const recentSize = 32 * 1024 * 1024 // ???

func (fb *fbtree) quickCheck(fn func(uint64)) {
	recent := int64(fb.store.Size()) - recentSize
	fb.quickCheck1(0, fb.root, recent, fn)
}

func (fb *fbtree) quickCheck1(depth int, offset uint64, recent int64,
	fn func(uint64)) {
	node := fb.getCkNode(offset)
	if depth < fb.treeLevels {
		// tree node
		for it := node.iter(); it.next(); {
			if int64(offset) < recent || fb.pathNode(offset) {
				fb.quickCheck1(depth+1, it.offset, recent, fn)
			}
		}
	} else {
		// leaf node
		for it := node.iter(); it.next(); {
			if int64(it.offset) > recent {
				fn(it.offset)
			}
		}
	}
}

// check verifies that the keys are in order and returns the number of keys.
// The supplied fn is applied to each leaf offset.
func (fb *fbtree) check(fn func(uint64)) (count, size, nnodes int) {
	key := ""
	return fb.check1(0, fb.root, &key, true, fn)
}

func (fb *fbtree) check1(depth int, offset uint64, key *string, path bool,
	fn func(uint64)) (count, size, nnodes int) {
	node := fb.getCkNode(offset)
	size += len(node)
	nnodes++
	for it := node.iter(); it.next(); {
		offset := it.offset
		if depth < fb.treeLevels {
			// tree
			path2 := fb.pathNode(offset)
			if path2 && !path {
				panic("orphaned path node")
			}
			if it.fi > 0 && *key > it.known {
				panic("keys out of order")
			}
			*key = it.known
			c, s, n := fb.check1(depth+1, offset, key, path2, fn) // recurse
			count += c
			size += s
			nnodes += n
		} else {
			// leaf
			count++
			if fn != nil {
				fn(offset)
			}
			itkey := fb.getLeafKey(offset)
			if !strings.HasPrefix(itkey, it.known) {
				panic("index key does not match data")
			}
			if *key > itkey {
				panic("keys out of order")
			}
			*key = itkey
		}
	}
	return
}

// ------------------------------------------------------------------

type fbIter = func() (string, uint64, bool)

// Iter returns a function that can be called to return consecutive entries.
// NOTE: The returned key is only the known prefix.
// (unlike mbtree which returns the actual key)
func (fb *fbtree) Iter() fbIter {
	var stack [maxlevels]*fnIter

	// traverse down the tree to the leftmost leaf, making a stack of iterators
	nodeOff := fb.root
	for i := 0; i < fb.treeLevels; i++ {
		stack[i] = fb.getNode(nodeOff).iter()
		stack[i].next()
		nodeOff = stack[i].offset
	}
	iter := fb.getNode(nodeOff).iter()

	return func() (string, uint64, bool) {
		for {
			if iter.next() {
				return iter.known, iter.offset, true // most common path
			}
			// end of leaf, go up the tree
			i := fb.treeLevels - 1
			for ; i >= 0; i-- {
				if stack[i].next() {
					nodeOff = stack[i].offset
					break
				}
			}
			if i == -1 {
				return "", 0, false // eof
			}
			// and then back down to the next leaf
			for i++; i < fb.treeLevels; i++ {
				stack[i] = fb.getNode(nodeOff).iter()
				stack[i].next()
				nodeOff = stack[i].offset
			}
			iter = fb.getNode(nodeOff).iter()
		}
	}
}

// ------------------------------------------------------------------

func (fb *fbtree) print() {
	fmt.Println("---------------------------------")
	fb.print1(0, fb.root)
	fmt.Println("---------------------------------")
}

func (fb *fbtree) print1(depth int, offset uint64) {
	explan := ""
	r, ok := fb.redirs.tbl.Get(offset)
	if ok && r.newOffset != 0 {
		explan += " -> " + OffStr(r.newOffset)
	} else if ok && r.mnode != nil {
		explan += " mnode"
	} else if ok {
		panic("neither mnode nor newOffset")
	}
	if _, pathNode := fb.redirs.paths.Get(offset); pathNode {
		explan += " PATH"
	}
	if depth >= fb.treeLevels {
		explan += " LEAF"
	}
	print(strings.Repeat(" . ", depth)+"offset", OffStr(offset)+explan)
	node := fb.getNode(offset)
	for it := node.iter(); it.next(); {
		offset := it.offset
		if depth < fb.treeLevels {
			// tree
			print(strings.Repeat(" . ", depth)+strconv.Itoa(it.fi)+":",
				it.npre, it.diff, "=", it.known)
			fb.print1(depth+1, offset) // recurse
		} else {
			// leaf
			// print(strings.Repeat(" . ", depth)+strconv.Itoa(it.fi)+":",
			// 	OffStr(offset)+",", it.npre, it.diff, "=", it.known,
			// 	"("+fb.getLeafKey(offset)+")")
		}
	}
}

// ------------------------------------------------------------------

// fbtreeBuilder is used to bulk load an fbtree.
// Keys must be added in order.
// The fbtree is built bottom up with no splitting or inserting.
// All nodes will be "full" except for the right hand edge.
type fbtreeBuilder struct {
	levels   []*level // leaf is [0]
	prev     string
	notFirst bool
	store    *stor.Stor
}

type level struct {
	first   string
	builder fNodeBuilder
}

func NewFbtreeBuilder(store *stor.Stor) *fbtreeBuilder {
	return &fbtreeBuilder{store: store, levels: []*level{{}}}
}

func (fb *fbtreeBuilder) Add(key string, off uint64) {
	if fb.notFirst {
		if key == fb.prev {
			panic("fbtreeBuilder keys must not have duplicates")
		}
		if key < fb.prev {
			panic("fbtreeBuilder keys must be inserted in order")
		}
	} else {
		fb.notFirst = true
	}
	fb.insert(0, key, off)
	fb.prev = key
}

func (fb *fbtreeBuilder) insert(li int, key string, off uint64) {
	if li >= len(fb.levels) {
		fb.levels = append(fb.levels, &level{})
	}
	lev := fb.levels[li]
	if len(lev.builder.fe) > (MaxNodeSize * 2 / 3) {
		// flush full node to stor
		offNode := lev.builder.fe.putNode(fb.store)
		fb.insert(li+1, lev.first, offNode) // recurse
		*lev = level{}
	}
	if len(lev.builder.fe) == 0 {
		lev.first = key
	}
	embedLen := 1
	if li > 0 {
		embedLen = 255
	}
	lev.builder.Add(key, off, embedLen)
}

func (fb *fbtreeBuilder) Finish() *Overlay {
	var key string
	var off uint64
	for li := 0; li < len(fb.levels); li++ {
		if li > 0 {
			// allow node to slightly exceed max size
			fb.levels[li].builder.Add(key, off, 255)
		}
		key = fb.levels[li].first
		off = fb.levels[li].builder.fe.putNode(fb.store)
	}
	treeLevels := len(fb.levels) - 1
	bt := OpenFbtree(fb.store, off, treeLevels, 0)
	return &Overlay{under: []tree{bt}}
}