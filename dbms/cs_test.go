// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package dbms

import (
	"net"
	"testing"
	"time"

	"github.com/apmckinlay/gsuneido/db19"
	"github.com/apmckinlay/gsuneido/db19/stor"
	"github.com/apmckinlay/gsuneido/dbms/mux"
	"github.com/apmckinlay/gsuneido/options"
	"github.com/apmckinlay/gsuneido/runtime"
	"github.com/apmckinlay/gsuneido/util/assert"
)

func TestClientServer(*testing.T) {
	// trace.Set(int(trace.ClientServer))
	options.BuiltDate = "Dec 29 2020 12:34"
	db, _ := db19.CreateDb(stor.HeapStor(8192))
	dbmsLocal := NewDbmsLocal(db)
	p1, p2 := net.Pipe()
	workers = mux.NewWorkers(doRequest)
	go newServerConn(dbmsLocal, p1)
	jserver, errmsg := checkHello(p2)
	assert.False(jserver)
	assert.This(errmsg).Is("")
	c := NewMuxClient(p2)
	ses := c.NewSession()
	ses.Get(nil, "tables", runtime.Next, nil)

	ses2 := c.NewSession()
	ses2.Get(nil, "tables", runtime.Prev, nil)
	ses2.Close()

	time.Sleep(25 * time.Millisecond)
}
