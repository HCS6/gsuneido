// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package runtime

import "sync"

type Sviews struct {
	defs map[string]string
	lock sync.Mutex
}

func (sv *Sviews) AddSview(name, def string) {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	if sv.defs == nil {
		sv.defs = make(map[string]string)
	}
	sv.defs[name] = def
}

func (sv *Sviews) GetSview(name string) string {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	return sv.defs[name]
}

func (sv *Sviews) DropSview(name string) bool {
	sv.lock.Lock()
	defer sv.lock.Unlock()
	if _, ok := sv.defs[name]; ok {
		delete(sv.defs, name)
		return true
	}
	return false
}
