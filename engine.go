// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
)

// XPath is the representation of a compiled xpath 1.0 expression.
// A XPath is safe for concurrent use by multiple goroutines.
type XPath struct {
	str  string
	expr Expr
}

func (x *XPath) String() string {
	return x.str
}

// Returns tells the DataType of value that this expression evaluates to.
func (x *XPath) Returns() DataType {
	return x.expr.Returns()
}

// IsStatic tells whether this xpath is static,
// i.e, it evaluates to same value every time.
//
// a static expression can be evaluated by passing
// nil arguments to Eval method.
func (x *XPath) IsStatic() bool {
	return Literals(x.expr)
}

// Eval evaluates the compiled XPath expression in the specified context and return the result.
//
// The vars argument can be nil. The returned value will be the raw result of the expression.
// To convert the result, use Value2String, Value2Number, Value2Boolean functions.
func (x *XPath) Eval(n dom.Node, vars Variables) (r interface{}, err error) {
	defer func() {
		panic2error(recover(), &err)
	}()
	return x.expr.Eval(&Context{n, 0, 1, vars}), nil
}

// Context represents the evaluation context of xpath engine.
type Context struct {
	// Node is the current node in context-set
	Node dom.Node

	// Pos is the position of current node in context-set
	Pos  int

	// Size is the size of the context-set
	Size int

	// Vars is the set of variable bindings
	Vars Variables
}

// Document returns the Document of current node in context-set
func (ctx *Context) Document() *dom.Document {
	n := ctx.Node
	for {
		if d, ok := n.(*dom.Document); ok {
			return d
		}
		n = Parent(n)
	}
}

// Variables is interface that is used to evaluate variable references.
//
// In the course of evaluating any single XPath expression, a variable's value must not change.
type Variables interface {
	// Eval returns the value of the variable. If there is no such variable,
	// it should return nil. The argument is the clark-name of the variable.
	//
	// The returned value must be nil, []dom.Node, string, float64 or bool.
	Eval(variable string) interface{}
}

// VariableMap implements Variables interface using map.
//
// Key must be clark-name of variable.
// Value must be []dom.Node, string, float64 or bool.
type VariableMap map[string]interface{}

func (vm VariableMap) Eval(variable string) interface{} {
	return vm[variable]
}

// Functions is interface that provides access to the set of
// user defined functions during xpath expression compilation.
//
// This cannot be used to override XPath built-in functions.
// In the course of evaluating any single XPath expression, a function must not change.
type Functions interface {
	// Resolve find a function bound to the given name in the set of available functions.
	// If there is no such function, it should return nil.
	// The argument is the clark-name of the function
	Resolve(function string) *Function
}

// FunctionMap implements Functions interface using map.
//
// Key must be clark-name of function.
type FunctionMap map[string]*Function

func (fm FunctionMap) Resolve(function string) *Function {
	return fm[function]
}
