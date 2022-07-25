// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package query

import (
	"github.com/apmckinlay/gsuneido/db19/index/ixkey"
	. "github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
	"github.com/apmckinlay/gsuneido/util/generic/ord"
	"github.com/apmckinlay/gsuneido/util/generic/set"
	"github.com/apmckinlay/gsuneido/util/generic/slc"
	"golang.org/x/exp/slices"
)

type Union struct {
	Compatible
	fixed    []Fixed // lazy, calculated by Fixed()
	strategy unionStrategy
	rewound  bool
	empty1   Row
	empty2   Row
	src1     bool
	src2     bool
	key1     string
	key2     string
	row1     Row
	row2     Row
	curKey   string
}

type unionApproach struct {
	keyIndex   []string
	strategy   unionStrategy
	idx1, idx2 []string
	reverse    bool
}

type unionStrategy int

const (
	// unionMerge is a merge of source and source2
	unionMerge unionStrategy = iota + 2
	// unionLookup is source not in source2, followed by source2 (unordered)
	unionLookup
)

func NewUnion(src, src2 Query) *Union {
	u := &Union{Compatible: Compatible{
		Query2: Query2{Query1: Query1{source: src}, source2: src2}}}
	u.init()
	return u
}

func (u *Union) String() string {
	strategy := ""
	switch u.strategy {
	case unionMerge:
		strategy += "-MERGE"
	case unionLookup:
		if u.disjoint == "" {
			strategy += "-LOOKUP"
			// if u.keyIndex != nil {
			// 	strategy += "^" + str.Join("(,)", u.keyIndex)
			// }
		}
	}
	return u.String2("UNION", strategy)
}

func (u *Union) Columns() []string {
	return u.allCols
}

func (u *Union) Keys() [][]string {
	if u.disjoint == "" {
		return [][]string{u.allCols}
	}
	keys := u.keypairs()
	for i := range keys {
		// keypairs must ensure that appending is valid
		keys[i] = set.AddUnique(keys[i], u.disjoint)
	}
	return withoutDupsOrSupersets(keys)
}

func (u *Union) Indexes() [][]string {
	// lookup can read via any index
	return set.UnionFn(u.source.Indexes(), u.source2.Indexes(), slices.Equal[string])
}

func (u *Union) Nrows() int {
	return u.nrowsCalc(u.source.Nrows(), u.source2.Nrows())
}

func (u *Union) nrowsCalc(n1, n2 int) int {
	if u.disjoint != "" {
		return n1 + n2
	}
	min := ord.Max(n1, n2) // smaller could be all duplicates
	max := n1 + n2         // could be no duplicates
	return (min + max) / 2 // estimate half way between
}

func (u *Union) Transform() Query {
	u.source = u.source.Transform()
	u.source2 = u.source2.Transform()
	// propagate Nothing
	if _, ok := u.source.(*Nothing); ok {
		return u.source2
	}
	if _, ok := u.source2.(*Nothing); ok {
		return u.source
	}
	return u
}

func (u *Union) Fixed() []Fixed {
	if u.fixed != nil { // once only
		return u.fixed
	}
	fixed1 := u.source.Fixed()
	fixed2 := u.source2.Fixed()
	for _, f1 := range fixed1 {
		for _, f2 := range fixed2 {
			if f1.col == f2.col {
				u.fixed = append(u.fixed,
					Fixed{f1.col, set.Union(f1.values, f2.values)})
				break
			}
		}
	}
	cols2 := u.source2.Columns()
	emptyStr := []string{""}
	for _, f1 := range fixed1 {
		if !slices.Contains(cols2, f1.col) {
			u.fixed = append(u.fixed,
				Fixed{f1.col, set.Union(f1.values, emptyStr)})
		}
	}
	cols1 := u.source.Columns()
	for _, f2 := range fixed2 {
		if !slices.Contains(cols1, f2.col) {
			u.fixed = append(u.fixed,
				Fixed{f2.col, set.Union(f2.values, emptyStr)})
		}
	}
	return u.fixed
}

func (u *Union) optimize(mode Mode, index []string) (Cost, any) {
	// if there is a required index, use Merge
	if index != nil {
		// if not disjoint then index must also be a key
		if u.disjoint == "" &&
			(!slc.ContainsFn(u.source.Keys(), index, set.Equal[string]) ||
				!slc.ContainsFn(u.source2.Keys(), index, set.Equal[string])) {
			return impossible, nil
		}
		cost := Optimize(u.source, mode, index) + Optimize(u.source2, mode, index)
		approach := &unionApproach{keyIndex: index, strategy: unionMerge,
			idx1: index, idx2: index}
		return cost, approach
	}
	// else no required index
	if u.disjoint != "" {
		cost := Optimize(u.source, mode, nil) + Optimize(u.source2, mode, nil)
		approach := &unionApproach{strategy: unionLookup}
		return cost, approach
	}
	// else not disjoint
	mergeCost, mergeApp := u.optMerge(u.source, u.source2, mode)
	lookupCost, lookupApp := u.optLookup(u.source, u.source2, mode)
	lookupCostRev, lookupAppRev := u.optLookup(u.source2, u.source, mode)
	cost, approach := min3(mergeCost, mergeApp,
		lookupCost, lookupApp, lookupCostRev, lookupAppRev)
	if cost >= impossible {
		return impossible, nil
	}
	return cost, approach
}

func (*Union) optMerge(source, source2 Query, mode Mode) (Cost, any) {
	// need key (unique) index to eliminate duplicates
	keys := set.IntersectFn(source.Keys(), source2.Keys(), set.Equal[string])
	var bestKey, bestIdx1, bestIdx2 []string
	bestCost := impossible
	for _, key := range keys {
		for _, idx1 := range source.Indexes() {
			if !set.Subset(idx1, key) {
				continue
			}
			ik1 := set.Intersect(idx1, key)
			for _, idx2 := range source2.Indexes() {
				ik2 := set.Intersect(idx2, key)
				if slices.Equal(ik1, ik2) {
					cost := Optimize(source, mode, idx1) +
						Optimize(source2, mode, idx2)
					if cost < bestCost {
						bestKey = key
						bestCost = cost
						bestIdx1, bestIdx2 = idx1, idx2
					}
				}
			}
		}
	}
	approach := &unionApproach{keyIndex: bestKey, strategy: unionMerge,
		idx1: bestIdx1, idx2: bestIdx2}
	return bestCost, approach
}

func (u *Union) optLookup(source, source2 Query, mode Mode) (Cost, any) {
	var bestKey []string
	bestCost := impossible
	for _, key := range source2.Keys() {
		cost := Optimize(source, mode, nil) +
			LookupCost(source2, mode, key, source.Nrows())
		if cost < bestCost {
			bestKey = key
			bestCost = cost
		}
	}
	approach := &unionApproach{keyIndex: bestKey, strategy: unionLookup,
		idx1: nil, idx2: bestKey}
	if source == u.source2 {
		approach.reverse = true
		bestCost += outOfOrder
	}
	return bestCost, approach
}

func (u *Union) setApproach(_ []string, approach any, tran QueryTran) {
	app := approach.(*unionApproach)
	u.strategy = app.strategy
	u.keyIndex = app.keyIndex
	if app.reverse {
		u.source, u.source2 = u.source2, u.source
	}
	u.source = SetApproach(u.source, app.idx1, tran)
	u.source2 = SetApproach(u.source2, app.idx2, tran)

	u.empty1 = make(Row, len(u.source.Header().Fields))
	u.empty2 = make(Row, len(u.source2.Header().Fields))

	u.rewound = true
}

// execution --------------------------------------------------------

func (u *Union) Rewind() {
	u.source.Rewind()
	u.source2.Rewind()
	u.rewound = true
}

func (u *Union) Get(th *Thread, dir Dir) Row {
	defer func() { u.rewound = false }()
	switch u.strategy {
	case unionLookup:
		return u.getLookup(th, dir)
	case unionMerge:
		return u.getMerge(th, dir)
	}
	assert.ShouldNotReachHere()
	return nil
}

func (u *Union) getLookup(th *Thread, dir Dir) Row {
	if u.rewound {
		u.src1 = (dir == Next)
	}
	var row Row
	for {
		if u.src1 {
			for {
				row = u.source.Get(th, dir)
				if row == nil {
					break
				}
				if !u.source2Has(th, row) {
					return JoinRows(row, u.empty2)
				}
			}
			if dir == Prev {
				return nil
			}
			u.src1 = false
			u.source2.Rewind()
		} else { // source2
			row = u.source2.Get(th, dir)
			if row != nil {
				return JoinRows(u.empty1, row)
			}
			if dir == Next {
				return nil
			}
			u.src1 = true
			// continue
		}
	}
}

func (u *Union) getMerge(th *Thread, dir Dir) Row {
	if u.hdr1 == nil {
		u.hdr1 = u.source.Header()
		u.hdr2 = u.source2.Header()
	}

	// read from the appropriate source(s)
	if u.rewound {
		u.fetch1(th, dir)
		u.fetch2(th, dir)
	} else {
		// curkey is required for changing direction
		if u.src1 || u.before(dir, u.key1, u.curKey, true) {
			u.fetch1(th, dir)
		}
		if u.src2 || u.before(dir, u.key2, u.curKey, false) {
			u.fetch2(th, dir)
		}
	}

	u.src1, u.src2 = false, false
	if u.row1 == nil && u.row2 == nil {
		u.curKey = u.key1
		u.src1 = true
		return nil
	} else if u.row1 != nil && u.row2 != nil && u.equal(u.row1, u.row2, th) {
		// rows same so return either one
		u.curKey = u.key1
		u.src1, u.src2 = true, true
		return JoinRows(u.row1, u.empty2)
	} else if u.row1 != nil &&
		(u.row2 == nil || u.before(dir, u.key1, u.key2, true)) {
		u.curKey = u.key1
		u.src1 = true
		return JoinRows(u.row1, u.empty2)
	} else {
		u.curKey = u.key2
		u.src2 = true
		return JoinRows(u.empty1, u.row2)
	}
}

func (u *Union) fetch1(th *Thread, dir Dir) {
	u.row1 = u.source.Get(th, dir)
	if u.row1 == nil {
		u.key1 = endKey(dir)
	} else {
		u.key1 = projectKey(u.row1, u.hdr1, u.keyIndex, th, u.st)
	}
}

func (u *Union) fetch2(th *Thread, dir Dir) {
	u.row2 = u.source2.Get(th, dir)
	if u.row2 == nil {
		u.key2 = endKey(dir)
	} else {
		u.key2 = projectKey(u.row2, u.hdr2, u.keyIndex, th, u.st)
	}
}

func (*Union) before(dir Dir, key1, key2 string, x bool) bool {
	if key1 == key2 {
		if dir == Next {
			return x
		}
		return !x
	}
	if dir == Next {
		return key1 < key2
	}
	return key1 > key2
}

func endKey(dir Dir) string {
	if dir == Next {
		return ixkey.Max
	} // else Prev
	return ixkey.Min
}

func (u *Union) Select(cols, vals []string) {
	u.source.Select(cols, vals)
	u.source2.Select(cols, vals)
	u.rewound = true
}

func (u *Union) Lookup(th *Thread, cols, vals []string) Row {
	u.Select(cols, vals)
	row := u.Get(th, Next)
	u.Select(nil, nil) // clear select
	return row
}

// unpack is for debugging output
func unpack(packed []string) []Value {
	vals := make([]Value, len(packed))
	for i, p := range packed {
		if p == ixkey.Max {
			vals[i] = SuStr("<max>")
		} else {
			vals[i] = Unpack(p)
		}
	}
	return vals
}
