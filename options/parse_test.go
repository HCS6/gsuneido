// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package options

import (
	"strings"
	"testing"

	"github.com/apmckinlay/gsuneido/util/assert"
)

func TestParse(t *testing.T) {
	test := func(args ...string) func(string) {
		Action, Arg, Port, CmdLine = "", "", "", ""
		Parse(args)
		s := Action
		if Arg != "" {
			s += " " + Arg
		}
		if Port != "3147" && Port != "" {
			s += " port " + Port
		}
		if CmdLine != "" {
			s += " | " + CmdLine
		}
		s = strings.TrimPrefix(s, " ")
		if Action == "error" {
			s = "error"
		}
		return func(expected string) {
			t.Helper()
			assert.T(t).This(s).Is(expected)
		}
	}
	test()("")
	test("-r")("repl")
	test("-repl")("repl")
	test("-c")("client 127.0.0.1")
	test("-client")("client 127.0.0.1")
	test("-c", "--")("client 127.0.0.1")
	test("-c", "1.2.3.4")("client 1.2.3.4")
	test("-client", "1.2.3.4")("client 1.2.3.4")
	test("-c", "-p", "1234")("client 127.0.0.1 port 1234")
	test("-c", "localhost", "-p", "1234")("client localhost port 1234")
	test("-c1.2.3.4")("error")
	test("-c", "--", "foo", "bar")("client 127.0.0.1 | foo bar")
	test("-client", "--", "foo", "bar")("client 127.0.0.1 | foo bar")
	test("-client1.2.3.4")("error")
	test("-load", "-client")("error")
	test("-p", "1234", "-repl")("error")
	test("-p1234")("error")
	test("-port", "1234", "-repl")("error")
	test("-port1234")("error")
	test("-p")("error")
	test("-port")("error")
	test("-c", "1.2.3.4", "foo", "bar")("client 1.2.3.4 | foo bar")
	test("-client", "1.2.3.4", "foo", "bar")("client 1.2.3.4 | foo bar")
	test("-load")("load")
	test("-load", "stdlib")("load stdlib")
	test("-dump")("dump")
	test("-dump", "stdlib")("dump stdlib")
	test("-server")("server")
	test("-repair")("repair")
	test("-xyz")("error")
}

func TestEscapeArg(t *testing.T) {
	test := func(s, expected string) {
		t.Helper()
		assert.T(t).This(EscapeArg(s)).Is(expected)
	}
	test(`foo`, `foo`)
	test(`foo bar`, `"foo bar"`)
	test(`ab"c`, `ab\"c`)
	test(`\`, `\`)
	test(`a\\\b`, `a\\\b`)
	test(`a\"b`, `a\\\"b`)
}
