// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
)

type XPath struct {
	str  string
	expr Expr
}

func (x *XPath) String() string {
	return x.str
}

func (x *XPath) Eval(n dom.Node, vars Variables) (r interface{}, err error) {
	defer func() {
		panic2error(recover(), &err)
	}()
	return x.expr.Eval(&Context{n, 0, 1, vars}), nil
}

type Compiler struct {
	NS        map[string]string
	Functions Functions
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
		panic2error(recover(), &err)
	}()
	expr, err := xpath.Parse(str)
	if err != nil {
		return nil, err
	}
	return &XPath{str, Simplify(c.compile(expr))}, nil
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
