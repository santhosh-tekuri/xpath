// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"errors"
	"fmt"
	"runtime"
)

type UnresolvedPrefixError string

func (e UnresolvedPrefixError) Error() string {
	return fmt.Sprintf("unresolved prefix: %s", string(e))
}

type UnresolvedVariableError string

func (e UnresolvedVariableError) Error() string {
	return fmt.Sprintf("unresolved variable: %s", string(e))
}

type UnresolvedFunctionError string

func (e UnresolvedFunctionError) Error() string {
	return fmt.Sprintf("unresolved function: %s", string(e))
}

type ArgCountError string

func (e ArgCountError) Error() string {
	return fmt.Sprintf("wrong number of args to function %s", string(e))
}

type InvalidValueError struct {
	val interface{}
}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf("%T is not valid xpath data-type", e.val)
}

type VarMustBeNodeSet string

func (e VarMustBeNodeSet) Error() string {
	return fmt.Sprintf("variable %s must evaluate to node-set", string(e))
}

func panic2error(r interface{}, errRef *error) {
	if r != nil {
		if _, ok := r.(runtime.Error); ok {
			panic(r)
		}
		if err, ok := r.(error); ok {
			*errRef = err
		} else {
			*errRef = errors.New(fmt.Sprint(r))
		}
	}
}
