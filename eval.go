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

func Compile(str string) (*XPath, error) {
	expr, err := xpath.Parse(str)
	if err != nil {
		return nil, err
	}
	return &XPath{str, compile(expr)}, nil
}
