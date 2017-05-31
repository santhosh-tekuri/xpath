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

// String returns the source xpath expression
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

// Eval evaluates the compiled XPath expression in the given context and return the result.
//
// The vars argument can be nil. The DataType of returned value will be *XPath.Returns()
func (x *XPath) Eval(n dom.Node, vars Variables) (r interface{}, err error) {
	defer func() {
		panic2error(recover(), &err)
	}()
	return x.expr.Eval(&Context{n, 0, 1, vars}), nil
}

// EvalNodeSet evaluates the compiled XPath expression in given context and returns []dom.Node value.
// if the result cannot be converted to []dom.Node, returns ConversionError
//
// The vars argument can be nil.
func (x *XPath) EvalNodeSet(n dom.Node, vars Variables) ([]dom.Node, error) {
	switch x.Returns() {
	case NodeSet, Any:
		r, err := x.Eval(n, vars)
		if err != nil {
			return nil, err
		}
		if ns, ok := r.([]dom.Node); ok {
			return ns, nil
		}
		return nil, ConversionError{TypeOf(r), NodeSet}
	default:
		return nil, ConversionError{x.Returns(), NodeSet}
	}
}

// EvalString evaluates the compiled XPath expression in given context and returns string value.
//
// The vars argument can be nil.
func (x *XPath) EvalString(n dom.Node, vars Variables) (string, error) {
	r, err := x.Eval(n, vars)
	if err != nil {
		return "", err
	}
	return Value2String(r), nil
}

// EvalNumber evaluates the compiled XPath expression in given context and returns float64 value.
//
// The vars argument can be nil.
func (x *XPath) EvalNumber(n dom.Node, vars Variables) (float64, error) {
	r, err := x.Eval(n, vars)
	if err != nil {
		return 0, err
	}
	return Value2Number(r), nil
}

// EvalBoolean evaluates the compiled XPath expression in given context and returns bool value.
//
// The vars argument can be nil.
func (x *XPath) EvalBoolean(n dom.Node, vars Variables) (bool, error) {
	r, err := x.Eval(n, vars)
	if err != nil {
		return false, err
	}
	return Value2Boolean(r), nil
}

// Context represents the evaluation context of xpath engine.
type Context struct {
	// Node is the current node in context-set
	Node dom.Node

	// Pos is the position of current node in context-set
	Pos int

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

// Eval returns the value bound to given variable.
// It returns nil if no value is bound.
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

// Resolve returns the *Function bound to given function name.
// It returns nil if no function is bound.
func (fm FunctionMap) Resolve(function string) *Function {
	return fm[function]
}
