// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

//go:build !windows

package system

func Service(name string, start, stop func()) error {
	return nil
}
