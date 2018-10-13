package compile

import (
	"bytes"
	"strings"
	"testing"

	"github.com/apmckinlay/gsuneido/interp"
	. "github.com/apmckinlay/gsuneido/util/hamcrest"
)

func TestCodegen(t *testing.T) {
	test := func(src, expected string) {
		ast := ParseFunction("function () {\n" + src + "\n}")
		fn := codegen(ast)
		// fmt.Println(src)
		// fmt.Println(ast)
		// fmt.Println(fn.Code)
		da := []string{}
		var s string
		for i := 0; i < len(fn.Code); {
			i, s = interp.Disasm1(fn, i)
			da = append(da, s)
		}
		actual := strings.Join(da, ", ")
		if actual != expected {
			t.Errorf("%s expected: %s but got: %s", src, expected, actual)
		}
	}
	test("", "")
	test("return", "")
	test("return true", "true")
	test("true", "true")
	test("123", "int 123")
	test("a", "load a")
	test("_a", "dyload _a")
	test("G", "global G")
	test("this", "this")

	test("-a", "load a, uminus")
	test("a + b", "load a, load b, add")
	test("a - b", "load a, load b, sub")
	test("a + b + c", "load a, load b, add, load c, add")
	test("a + b - c", "load a, load b, add, load c, sub")
	test("a - b - c", "load a, load b, sub, load c, sub")

	test("a * b", "load a, load b, mul")
	test("a / b", "load a, load b, div")
	test("a * b * c", "load a, load b, mul, load c, mul")
	test("a * b / c", "load a, load b, mul, load c, div")
	test("a / b / c", "load a, load b, div, load c, div")

	test("a % b", "load a, load b, mod")
	test("a % b % c", "load a, load b, mod, load c, mod")

	test("a | b | c", "load a, load b, bitor, load c, bitor")

	test("a is true", "load a, true, is")
	test("s = 'hello'", "value 'hello', store s")
	test("_dyn = 123", "int 123, store _dyn")
	test("a = b = c", "load c, store b, store a")
	test("a = true; not a", "true, store a, pop, load a, not")
	test("n += 5", "load n, int 5, add, store n")
	test("++n", "load n, one, add, store n")
	test("n--", "load n, dup, one, sub, store n, pop")
	test("a.b", "load a, value 'b', get")
	test("a[2]", "load a, int 2, get")
	test("a.b = 123", "load a, value 'b', int 123, put")
	test("a[2] = false", "load a, int 2, false, put")
	test("a.b += 5", "load a, value 'b', dup2, get, int 5, add, put")
	test("++a.b", "load a, value 'b', dup2, get, one, add, put")
	test("a.b++", "load a, value 'b', dup2, get, dupx2, one, add, put, pop")
	test("a[..]", "load a, zero, value 2147483647, rangeto")
	test("a[..3]", "load a, zero, int 3, rangeto")
	test("a[2..]", "load a, int 2, value 2147483647, rangeto")
	test("a[2..3]", "load a, int 2, int 3, rangeto")
	test("a[::]", "load a, zero, value 2147483647, rangelen")
	test("a[::3]", "load a, zero, int 3, rangelen")
	test("a[2::]", "load a, int 2, value 2147483647, rangelen")
	test("a[2::3]", "load a, int 2, int 3, rangelen")

	test("return", "")
	test("return 123", "int 123")

	test("throw 'fubar'", "value 'fubar', throw")

	test("f()", "load f, call()")
	test("F()", "global F, call()")
	test("f(a, b)", "load a, load b, load f, call(?, ?)")
	test("f(a, b, c:, d: 0)", "load a, load b, true, zero, load f, call(?, ?, c:, d:)")
	test("a().Add(123)", "int 123, load a, call(), value 'Add', get, call(?)")
	test("a().Add(123).Size()",
		"int 123, load a, call(), value 'Add', get, call(?), value 'Size', get, call()")
	test("f(@args)", "load f, call(@)")
	test("f(@+1args)", "load f, call(@+1)")
}

func TestControl(t *testing.T) {
	test := func(src, expected string) {
		t.Helper()
		ast := ParseFunction("function () {\n" + src + "\n}")
		fn := codegen(ast)
		buf := new(bytes.Buffer)
		interp.Disasm(buf, fn)
		s := buf.String()
		Assert(t).That(s, Like(expected).Comment(src))
	}

	test("a and b", `
		0: load a
		2: and 8
		5: load b
		7: bool
		8:`)
	test("a or b", `
		0: load a
		2: or 8
		5: load b
		7: bool
		8:`)
	test("a or b or c", `
		0: load a
		2: or 13
		5: load b
		7: or 13
		10: load c
		12: bool
		13:`)

	test("a ? b : c", `
		0: load a
		2: qmark 10
		5: load b
		7: jump 12
		10: load c
		12:`)

	test("a in (4,5,6)", `
		0: load a
        2: int 4
        5: in 18
        8: int 5
        11: in 18
        14: int 6
        17: is
        18:`)

	test("while (a) b", `
		0: jump 6
		3: load b
		5: pop
		6: load a
		8: tjump 3
		11:`)
	test("while a\n;", `
		0: jump 3
		3: load a
		5: tjump 3
		8:`)

	test("if (a) b", `
		0: load a
		2: fjump 8
		5: load b
		7: pop
		8:`)
	test("if (a) b else c", `
		0: load a
		2: fjump 11
		5: load b
		7: pop
		8: jump 14
		11: load c
		13: pop
		14:`)

	test("switch { case 1: b }", `
		0: true
        1: one
        2: nejump 11
        5: load b
        7: pop
        8: jump 16
        11: pop
        12: value 'unhandled switch value'
        15: throw
        16:`)
	test("switch a { case 1,2: b case 3: c default: d }", `
		0: load a
        2: one
        3: eqjump 12
        6: int 2
        9: nejump 18
        12: load b
        14: pop
        15: jump 34
        18: int 3
        21: nejump 30
        24: load c
        26: pop
        27: jump 34
        30: pop
        31: load d
        33: pop
        34:`)

	test("forever { break }", `
		0: jump 6
		3: jump 0
		6:`)

	test("while a { b; break; continue }", `
		0: jump 12
		3: load b
		5: pop
		6: jump 17
		9: jump 0
		12: load a
		14: tjump 3
		17:`)

	test("do a while b", `
		0: load a
		2: pop
		3: load b
		5: tjump 0
		8:`)

	test("for (i = 0; i < 9; ++i) body", `
		0: zero
        1: store i
        3: pop
        4: jump 17
        7: load body
        9: pop
        10: load i
        12: one
        13: add
        14: store i
        16: pop
        17: load i
        19: int 9
        22: lt
        23: tjump 7
        26:`)
}

func TestParams(t *testing.T) {
	test := func(s string) {
		ast := ParseFunction("function (" + s + ") { }")
		fn := codegen(ast)
		Assert(t).That(fn.String(), Equals("function ("+s+")"))
	}
	test("")
	test("a, b, c")
	test("a, .b, .C, ._d, ._E")
	test("a, b=1, c='x'")
}
