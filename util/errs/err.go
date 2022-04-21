// Copyright Suneido Software Corp. All rights reserved.
// Governed by the MIT license found in the LICENSE file.

package errs

import (
	"errors"
	"fmt"
)

func From(e any) error {
	if err, ok := e.(error); ok {
		return err
	}
	return errors.New(fmt.Sprint(e))
}
