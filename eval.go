// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
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

func (x *XPath) Eval(n dom.Node) (r interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	return x.expr.eval(&context{n, 0, 1}), nil
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
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
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
}
