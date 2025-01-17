// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package builtin

import (
	"github.com/apmckinlay/gsuneido/options"
	. "github.com/apmckinlay/gsuneido/runtime"
)

var _ = builtin(ClientQ, "()")

func ClientQ() Value {
	return SuBool(options.Action == "client")
}
