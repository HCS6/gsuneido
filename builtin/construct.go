package builtin

import (
	"strings"

	. "github.com/apmckinlay/gsuneido/runtime"
)

var _ = builtin("Construct(what, suffix='')",
	func(t *Thread, args []Value) Value {
		what := args[0]
		suffix := ToStr(args[1])
		as := ArgSpec0
		var arg Value
		if c, ok := what.ToContainer(); ok {
			what = c.ListGet(0)
			if what == nil {
				panic("Construct: object requires member 0")
			}
			arg = c
			as = ArgSpecEach1
		}
		if s, ok := what.ToStr(); ok {
			if !strings.HasSuffix(s, suffix) {
				s += suffix
			}
			what = Global.GetName(t, s)
		}
		t.Push(what)
		if arg != nil {
			t.Push(arg)
		}
		return t.CallMethod("*new*", as)
	})