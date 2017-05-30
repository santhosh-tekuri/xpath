// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
)

type XPath struct {
	str  string
	expr Expr
}

func (x *XPath) String() string {
	return x.str
}

func (x *XPath) Returns() DataType {
	return x.expr.Returns()
}

func (x *XPath) IsStatic() bool {
	return Literals(x.expr)
}

func (x *XPath) Eval(n dom.Node, vars Variables) (r interface{}, err error) {
	defer func() {
		panic2error(recover(), &err)
	}()
	return x.expr.Eval(&Context{n, 0, 1, vars}), nil
}

type Context struct {
	Node dom.Node
	Pos  int
	Size int
	Vars Variables
}

func (ctx *Context) Document() *dom.Document {
	n := ctx.Node
	for {
		if d, ok := n.(*dom.Document); ok {
			return d
		}
		n = Parent(n)
	}
}

type Variables interface {
	eval(variable string) interface{}
}

type VariableMap map[string]interface{}

func (vm VariableMap) eval(variable string) interface{} {
	return vm[variable]
}

type Functions interface {
	resolve(function string) *Function
}

type FunctionMap map[string]*Function

func (fm FunctionMap) resolve(function string) *Function {
	return fm[function]
}
