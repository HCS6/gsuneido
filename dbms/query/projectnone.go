// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package query

import . "github.com/apmckinlay/gsuneido/runtime"

// ProjectNone produces a single empty row with no columns.
// It results from a Project with no columns.
// It is not generated by the query parser.
type ProjectNone struct {
	cache
	done bool
}

var _ Query = (*ProjectNone)(nil)

func (*ProjectNone) String() string {
	return "PROJECT-NONE"
}

func (pn *ProjectNone) Transform() Query {
	return pn
}

func (*ProjectNone) Columns() []string {
	return nil
}

func (*ProjectNone) Keys() [][]string {
	return [][]string{{}}
}

func (*ProjectNone) fastSingle() bool {
	return true
}

func (*ProjectNone) Indexes() [][]string {
	return [][]string{{}}
}

func (*ProjectNone) Nrows() (int, int) {
	return 1, 1
}

func (*ProjectNone) rowSize() int {
	return 0
}

func (*ProjectNone) Order() []string {
	return nil
}

func (*ProjectNone) Fixed() []Fixed {
	return nil
}

func (*ProjectNone) Updateable() string {
	return "nothing"
}

func (*ProjectNone) SingleTable() bool {
	return true
}

func (*ProjectNone) SetTran(QueryTran) {
}

func (*ProjectNone) optimize(Mode, []string, float64) (Cost, Cost, any) {
	return 0, 0, nil
}

func (*ProjectNone) setApproach([]string, float64, any, QueryTran) {
}

func (*ProjectNone) lookupCost() Cost {
	return 0
}

func (*ProjectNone) Lookup(*Thread, []string, []string) Row {
	return nil
}

var pjHeader = SimpleHeader([]string{})

func (*ProjectNone) Header() *Header {
	return pjHeader
}

func (*ProjectNone) Output(*Thread, Record) {
	panic("can't Output to ProjectNone")
}

func (pn *ProjectNone) Get(*Thread, Dir) Row {
	if pn.done {
		return nil
	}
	pn.done = true
	return Row{DbRec{Record: Record("")}}
}

func (*ProjectNone) Rewind() {
}

func (*ProjectNone) Select([]string, []string) {
}
