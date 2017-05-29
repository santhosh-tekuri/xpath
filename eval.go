// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
)

type XPath struct {
	str  string
	expr expr
}

func (x *XPath) String() string {
	return x.str
}

func (x *XPath) Eval(n dom.Node, vars Variables) (r interface{}, err error) {
	defer func() {
		err = panic2error(recover())
	}()
	return x.expr.eval(&context{n, 0, 1, vars}), nil
}

type Compiler struct {
	NS map[string]string
}

func NewCompiler(ns map[string]string) *Compiler {
	return &Compiler{ns}
}

func (c *Compiler) resolvePrefix(prefix string) (string, bool) {
	if prefix == "" {
		return "", true
	}
	uri, ok := c.NS[prefix]
	return uri, ok
}

func (c *Compiler) Compile(str string) (x *XPath, err error) {
	defer func() {
		err = panic2error(recover())
	}()
	expr, err := xpath.Parse(str)
	if err != nil {
		return nil, err
	}
	return &XPath{str, c.compile(expr)}, nil
}

type context struct {
	node dom.Node
	pos  int
	size int
	vars Variables
}

type Variables interface {
	eval(variable string) interface{}
}

type VariableMap map[string]interface{}

func (vm VariableMap) eval(variable string) interface{} {
	return vm[variable]
}

func panic2error(r interface{}) error {
	if r == nil {
		return nil
	}
	if _, ok := r.(runtime.Error); ok {
		panic(r)
	}
	if err, ok := r.(error); ok {
		return err
	}
	return errors.New(fmt.Sprint(r))
}

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

type InvalidValueError interface{}

func (e InvalidValueError) Error() string {
	return fmt.Sprintf("%T is not valid xpath data-type", interface{}(e))
}
