// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"fmt"
	"math"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
)

// A Compiler represents a xpath 1.0 expression compiler.
type Compiler struct {
	// NS gives bindings of prefix to uri
	NS map[string]string

	// Functions gives access to set of user defined functions.
	Functions Functions
}

// Compile compiles given xpath 1.0 expression, if successful
// return a XPath object.
//
// Namespace prefixes and functions are resolved during compilation.
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

func (c *Compiler) compile(e xpath.Expr) Expr {
	switch e := e.(type) {
	case xpath.Number:
		return numberVal(e)
	case xpath.String:
		return stringVal(e)
	case *xpath.VarRef:
		uri, ok := c.resolvePrefix(e.Prefix)
		if !ok {
			panic(UnresolvedPrefixError(e.Prefix))
		}
		return &variable{ClarkName(uri, e.Local), Unknown}
	case *xpath.NegateExpr:
		return &negateExpr{asNumber(c.compile(e.Expr))}
	case *xpath.BinaryExpr:
		lhs := c.compile(e.LHS)
		rhs := c.compile(e.RHS)
		switch e.Op {
		case xpath.Add, xpath.Subtract, xpath.Multiply, xpath.Div, xpath.Mod:
			return &arithmeticExpr{asNumber(lhs), asNumber(rhs), arithmeticOp[e.Op-xpath.Add]}
		case xpath.And:
			return &logicalExpr{asBoolean(lhs), asBoolean(rhs), false}
		case xpath.Or:
			return &logicalExpr{asBoolean(lhs), asBoolean(rhs), true}
		case xpath.EQ, xpath.NEQ:
			return &equalityExpr{lhs, rhs, equalityOp[e.Op]}
		case xpath.LT, xpath.LTE, xpath.GT, xpath.GTE:
			return &relationalExpr{lhs, rhs, relationalOp[e.Op-xpath.LT]}
		case xpath.Union:
			return &unionExpr{asNodeSet(lhs), asNodeSet(rhs)}
		default:
			panic(fmt.Sprintf("unknown binaryOp %v", e.Op))
		}
	case *xpath.LocationPath:
		lp := new(locationPath)
		lp.abs = e.Abs
		for _, estep := range e.Steps {
			s := new(step)
			lp.steps = append(lp.steps, s)
			s.iter = iterators[estep.Axis]
			switch estep.Axis {
			case xpath.Preceding, xpath.PrecedingSibling, xpath.Ancestor, xpath.AncestorOrSelf:
				s.reverse = true
			}
			s.test = c.nodeTest(estep.Axis, estep.NodeTest)
			s.predicates = c.compilePredicates(estep.Predicates)
		}
		return lp
	case *xpath.FilterExpr:
		return &filterExpr{c.compile(e.Expr), c.compilePredicates(e.Predicates)}
	case *xpath.PathExpr:
		return &pathExpr{c.compile(e.Filter), c.compile(e.LocationPath).(*locationPath)}
	case *xpath.FuncCall:
		uri, ok := c.resolvePrefix(e.Prefix)
		if !ok {
			panic(UnresolvedPrefixError(e.Prefix))
		}
		fname := ClarkName(uri, e.Local)
		function := coreFunctions[fname]
		if function == nil && c.Functions != nil {
			function = c.Functions.Resolve(fname)
		}
		if function == nil {
			panic(UnresolvedFunctionError(fname))
		}

		if !function.Args.valid() {
			panic(SignatureError(fname))
		}
		if !function.Args.canAccept(len(e.Args)) {
			panic(ArgCountError(fname))
		}
		var args []Expr
		if len(e.Args) > 0 {
			args = make([]Expr, len(e.Args))
			for i, arg := range e.Args {
				arg := c.compile(arg)
				switch function.Args.typeOf(i) {
				case Unknown:
					args[i] = arg
				case NodeSet:
					args[i] = asNodeSet(arg)
				case String:
					args[i] = asString(arg)
				case Number:
					args[i] = asNumber(arg)
				case Boolean:
					args[i] = asBoolean(arg)
				default:
					panic(fmt.Sprintf("unexpected arg type %v", function.Args.typeOf(i)))
				}
			}
		}
		return function.Compile(function, args)
	default:
		panic(fmt.Sprintf("compile(%T) is not implemented", e))
	}
}

/************************************************************************/

func asNodeSet(e Expr) Expr {
	if v, ok := e.(*variable); ok {
		v.returns = NodeSet
	} else if e.Returns() != NodeSet {
		panic("node-set expected")
	}
	return e
}

func asString(expr Expr) Expr {
	if expr.Returns() == String {
		return expr
	}
	return &stringFunc{expr}
}

func asNumber(expr Expr) Expr {
	if expr.Returns() == Number {
		return expr
	}
	return &numberFunc{expr}
}

func asBoolean(expr Expr) Expr {
	if expr.Returns() == Boolean {
		return expr
	}
	return &booleanFunc{expr}
}

/************************************************************************/

func (c *Compiler) nodeTest(axis xpath.Axis, nodeTest xpath.NodeTest) func(dom.Node) bool {
	switch test := nodeTest.(type) {
	case xpath.NodeType:
		switch test {
		case xpath.Node:
			return alwaysTrue
		case xpath.Comment:
			return isComment
		case xpath.Text:
			return isText
		}
	case xpath.PITest:
		return isProcInst(string(test))
	case *xpath.NameTest:
		uri, ok := c.resolvePrefix(test.Prefix)
		if !ok {
			panic(UnresolvedPrefixError(test.Prefix))
		}
		switch axis {
		case xpath.Attribute:
			if test.Local == "*" {
				if uri == "" {
					return alwaysTrue
				}
				return testAttrNs(uri)
			}
			return testAttrName(uri, test.Local)
		case xpath.Namespace:
			if test.Prefix == "" && test.Local == "*" {
				return alwaysTrue
			}
			return testNamespaceName(uri, test.Local)
		default:
			if test.Local == "*" {
				if uri == "" {
					return isElement
				}
				return testElementNS(uri)
			}
			return testElementName(uri, test.Local)
		}
	}
	panic(fmt.Sprintf("BUG: unexpected nodeTest %T", nodeTest))
}

func alwaysTrue(dom.Node) bool {
	return true
}

func isElement(n dom.Node) bool {
	_, ok := n.(*dom.Element)
	return ok
}

func isComment(n dom.Node) bool {
	_, ok := n.(*dom.Comment)
	return ok
}

func isText(n dom.Node) bool {
	_, ok := n.(*dom.Text)
	return ok
}

func isProcInst(name string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.ProcInst); ok {
			return name == "" || name == n.Target
		}
		return false
	}
}

func testElementNS(uri string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Element); ok {
			return n.URI == uri
		}
		return false
	}
}

func testElementName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Element); ok {
			return n.URI == uri && n.Local == local
		}
		return false
	}
}

func testAttrNs(uri string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Attr); ok {
			return n.URI == uri
		}
		return false
	}
}

func testAttrName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.Attr); ok {
			return n.URI == uri && n.Local == local
		}
		return false
	}
}

func testNamespaceName(uri, local string) func(dom.Node) bool {
	return func(n dom.Node) bool {
		if n, ok := n.(*dom.NameSpace); ok {
			return uri == "" && n.Prefix == local
		}
		return false
	}
}

/************************************************************************/

func (c *Compiler) resolvePrefix(prefix string) (string, bool) {
	if prefix == "" {
		return "", true
	}
	uri, ok := c.NS[prefix]
	return uri, ok
}

func (c *Compiler) compilePredicates(predicates []xpath.Expr) []Expr {
	var arr []Expr
	for _, p := range predicates {
		arr = append(arr, c.compile(p))
	}
	return arr
}

// Simplify returns the simplified expression.
//
// Simplification does evaluate all static expressions.
// An Expr that supports simplification implements: interface{ Simplify() Expr
func Simplify(e Expr) Expr {
	if e, ok := e.(interface {
		Simplify() Expr
	}); ok {
		return e.Simplify()
	}
	return e
}

// Literals returns true of all given expressions are literal expression.
//
// A literal expressions is an expression which wraps string, float64 or bool.
// An expression that includes only literals are candidates for simplification.
func Literals(exprs ...Expr) bool {
	for _, expr := range exprs {
		switch expr.(type) {
		case nil, stringVal, numberVal, booleanVal:
		// continue
		default:
			return false
		}
	}
	return true
}

/************************************************************************/

var arithmeticOp = []func(float64, float64) float64{
	func(x, y float64) float64 {
		return x + y
	},
	func(x, y float64) float64 {
		return x - y
	},
	func(x, y float64) float64 {
		return x * y
	},
	func(x, y float64) float64 {
		return math.Mod(x, y)
	},
	func(x, y float64) float64 {
		return x / y
	},
}

var equalityOp = []func(interface{}, interface{}) bool{
	func(v1, v2 interface{}) bool {
		return v1 == v2
	},
	func(v1, v2 interface{}) bool {
		return v1 != v2
	},
}

var relationalOp = []func(float64, float64) bool{
	func(v1, v2 float64) bool {
		return v1 < v2
	},
	func(v1, v2 float64) bool {
		return v1 <= v2
	},
	func(v1, v2 float64) bool {
		return v1 > v2
	},
	func(v1, v2 float64) bool {
		return v1 >= v2
	},
}
