// Code generated by "stringer -type=Type"; DO NOT EDIT.

package types

import "strconv"

const _Type_name = "BooleanNumberStringDateObjectRecordFunctionBlockBuiltinFunctionClassMethodExceptInstanceIteratorTransactionQueryCursor"

var _Type_index = [...]uint8{0, 7, 13, 19, 23, 29, 35, 43, 48, 63, 68, 74, 80, 88, 96, 107, 112, 118}

func (i Type) String() string {
	if i < 0 || i >= Type(len(_Type_index)-1) {
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Type_name[_Type_index[i]:_Type_index[i+1]]
}
