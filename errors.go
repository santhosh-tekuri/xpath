// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"errors"
	"fmt"
	"runtime"
)

// UnresolvedPrefixError is the error type returned by *Compiler#Compile function.
// It tells that no URI is bound for that prefix.
type UnresolvedPrefixError string

func (e UnresolvedPrefixError) Error() string {
	return fmt.Sprintf("unresolved prefix: %s", string(e))
}

// UnresolvedVariableError is the error type returned by *XPath.Eval function.
// It tells that no variable is bound for that clarkName.
type UnresolvedVariableError string

func (e UnresolvedVariableError) Error() string {
	return fmt.Sprintf("unresolved variable: %s", string(e))
}

// UnresolvedFunctionError is the error type returned by *Compiler#Compile function.
// It tells that no function is bound for that clarkName.
type UnresolvedFunctionError string

func (e UnresolvedFunctionError) Error() string {
	return fmt.Sprintf("unresolved function: %s", string(e))
}

// SignatureError is the error type returned by *Compiler#Compile function.
// It tells that function registered for that clarkName has invalid signature.
//
// the signature is valid only if:
// - variadic argument can appear only as last argument.
// - all mandatory arguments must precede optional and variadic arguments.
type SignatureError string

func (e SignatureError) Error() string {
	return fmt.Sprintf("function %s has invalid argument signature", string(e))
}

// ArgCountError is the error type returned by *Compiler#Compile function.
// It tells that function registered for that clarkName does not accept the
// number of args specified in xpath expression.
type ArgCountError string

func (e ArgCountError) Error() string {
	return fmt.Sprintf("wrong number of args to function %s", string(e))
}

// InvalidValueError is the error type returned by *XPath#Eval function.
// It tells that function registered returned value other than
// []dom.Node, string, float64 or boolean
type InvalidValueError struct {
	val interface{}
}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf("%T is not valid xpath data-type", e.val)
}

// VarMustBeNodeSet is the error type returned by *XPath#Eval function.
// It tells that variable or function that is expected to evaluate to
// []dom.Node results in value that is not []dom.Node.
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
