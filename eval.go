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
	expr expr
}

func (x *XPath) String() string {
	return x.str
}

func (x *XPath) Eval(n dom.Node) interface{} {
	return x.expr.eval(n)
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

func (c *Compiler) Compile(str string) (*XPath, error) {
	expr, err := xpath.Parse(str)
	if err != nil {
		return nil, err
	}
	return &XPath{str, c.compile(expr)}, nil
}
