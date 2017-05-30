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
		var function *Function
		if c.Functions != nil {
			function = c.Functions.resolve(fname)
		}
		if function == nil {
			if e.Prefix == "" {
				function = coreFunctions[fname]
			}
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

func (c *Compiler) compilePredicates(predicates []xpath.Expr) []Expr {
	var arr []Expr
	for _, p := range predicates {
		arr = append(arr, c.compile(p))
	}
	return arr
}

type Expr interface {
	Returns() DataType
	Eval(ctx *Context) interface{}
}

func Simplify(e Expr) Expr {
	if e, ok := e.(interface {
		Simplify() Expr
	}); ok {
		return e.Simplify()
	}
	return e
}

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

type numberVal float64

func (numberVal) Returns() DataType {
	return Number
}

func (e numberVal) Eval(ctx *Context) interface{} {
	return float64(e)
}

type stringVal string

func (stringVal) Returns() DataType {
	return String
}

func (e stringVal) Eval(ctx *Context) interface{} {
	return string(e)
}

type booleanVal bool

func (booleanVal) Returns() DataType {
	return Boolean
}

func (e booleanVal) Eval(ctx *Context) interface{} {
	return bool(e)
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

type negateExpr struct {
	arg Expr
}

func (*negateExpr) Returns() DataType {
	return Number
}

func (e *negateExpr) Eval(ctx *Context) interface{} {
	return -e.arg.Eval(ctx).(float64)
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

type arithmeticExpr struct {
	lhs   Expr
	rhs   Expr
	apply func(float64, float64) float64
}

func (*arithmeticExpr) Returns() DataType {
	return Number
}

func (e *arithmeticExpr) Eval(ctx *Context) interface{} {
	return e.apply(e.lhs.Eval(ctx).(float64), e.rhs.Eval(ctx).(float64))
}

func (e *arithmeticExpr) Simplify() Expr {
	e.lhs, e.rhs = Simplify(e.lhs), Simplify(e.rhs)
	if Literals(e.lhs, e.rhs) {
		return Value2Expr(e.Eval(nil))
	}
	return e
}

/************************************************************************/

var equalityOp = []func(interface{}, interface{}) bool{
	func(v1, v2 interface{}) bool {
		return v1 == v2
	},
	func(v1, v2 interface{}) bool {
		return v1 != v2
	},
}

type equalityExpr struct {
	lhs   Expr
	rhs   Expr
	apply func(interface{}, interface{}) bool
}

func (*equalityExpr) Returns() DataType {
	return Boolean
}

func (e *equalityExpr) Eval(ctx *Context) interface{} {
	lhs, rhs := e.lhs.Eval(ctx), e.rhs.Eval(ctx)
	lhsType, rhsType := TypeOf(lhs), TypeOf(rhs)
	switch {
	case lhsType == NodeSet && rhsType == NodeSet:
		lhs, rhs := lhs.([]dom.Node), rhs.([]dom.Node)
		if len(lhs) > 0 && len(rhs) > 0 {
			for _, n1 := range lhs {
				n1Str := Node2String(n1)
				for _, n2 := range rhs {
					if e.apply(n1Str, Node2String(n2)) {
						return true
					}
				}
			}
		}
		return false
	case lhsType != NodeSet && rhsType != NodeSet:
		switch {
		case lhsType == Boolean || rhsType == Boolean:
			return e.apply(Value2Boolean(lhs), Value2Boolean(rhs))
		case lhsType == Number || rhsType == Number:
			return e.apply(Value2Number(lhs), Value2Number(rhs))
		default:
			return e.apply(Value2String(lhs), Value2String(rhs))
		}
	default:
		var val interface{}
		var nodeSet []dom.Node
		if lhsType == NodeSet {
			val, nodeSet = rhs, lhs.([]dom.Node)
		} else {
			val, nodeSet = lhs, rhs.([]dom.Node)
		}
		switch TypeOf(val) {
		case Boolean:
			return e.apply(val, Value2Boolean(nodeSet))
		case String:
			for _, n := range nodeSet {
				if e.apply(val, Node2String(n)) {
					return true
				}
			}
			return false
		default:
			for _, n := range nodeSet {
				if e.apply(val, Node2Number(n)) {
					return true
				}
			}
			return false
		}
	}
}

func (e *equalityExpr) Simplify() Expr {
	e.lhs, e.rhs = Simplify(e.lhs), Simplify(e.rhs)
	if Literals(e.lhs, e.rhs) {
		return Value2Expr(e.Eval(nil))
	}
	return e
}

/************************************************************************/

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

type relationalExpr struct {
	lhs   Expr
	rhs   Expr
	apply func(float64, float64) bool
}

func (*relationalExpr) Returns() DataType {
	return Boolean
}

func (e *relationalExpr) Eval(ctx *Context) interface{} {
	lhs, rhs := e.lhs.Eval(ctx), e.rhs.Eval(ctx)
	lhsType, rhsType := TypeOf(lhs), TypeOf(rhs)
	switch {
	case lhsType == NodeSet && rhsType == NodeSet:
		lhs, rhs := lhs.([]dom.Node), rhs.([]dom.Node)
		if len(lhs) > 0 && len(rhs) > 0 {
			for _, n1 := range lhs {
				n1Num := Node2Number(n1)
				for _, n2 := range rhs {
					if e.apply(n1Num, Node2Number(n2)) {
						return true
					}
				}
			}
		}
		return false
	case lhsType != NodeSet && rhsType != NodeSet:
		return e.apply(Value2Number(lhs), Value2Number(rhs))
	case lhsType == NodeSet:
		rhs := Value2Number(rhs)
		for _, n := range lhs.([]dom.Node) {
			if e.apply(Node2Number(n), rhs) {
				return true
			}
		}
		return false
	default:
		lhs := Value2Number(lhs)
		for _, n := range rhs.([]dom.Node) {
			if e.apply(lhs, Node2Number(n)) {
				return true
			}
		}
		return false
	}
}

func (e *relationalExpr) Simplify() Expr {
	e.lhs, e.rhs = Simplify(e.lhs), Simplify(e.rhs)
	if Literals(e.lhs, e.rhs) {
		return Value2Expr(e.Eval(nil))
	}
	return e
}

/************************************************************************/

type logicalExpr struct {
	lhs      Expr
	rhs      Expr
	lhsValue bool
}

func (*logicalExpr) Returns() DataType {
	return Boolean
}

func (e *logicalExpr) Eval(ctx *Context) interface{} {
	if e.lhs.Eval(ctx) == e.lhsValue {
		return e.lhsValue
	}
	return e.rhs.Eval(ctx)
}

func (e *logicalExpr) Simplify() Expr {
	e.lhs, e.rhs = Simplify(e.lhs), Simplify(e.rhs)
	if Literals(e.lhs) && e.lhs.Eval(nil) == e.lhsValue {
		return Value2Expr(e.lhsValue)
	}
	if Literals(e.rhs) {
		return Value2Expr(e.rhs.Eval(nil))
	}
	return e
}

/************************************************************************/

type unionExpr struct {
	lhs Expr
	rhs Expr
}

func (*unionExpr) Returns() DataType {
	return NodeSet
}

func (e *unionExpr) Eval(ctx *Context) interface{} {
	lhs := e.lhs.Eval(ctx).([]dom.Node)
	rhs := e.rhs.Eval(ctx).([]dom.Node)
	unique := make(map[dom.Node]struct{})
	for _, n := range lhs {
		unique[n] = struct{}{}
	}
	for _, n := range rhs {
		if _, ok := unique[n]; !ok {
			lhs = append(lhs, n)
		}
	}
	order(lhs)
	return lhs
}

/************************************************************************/

type locationPath struct {
	abs   bool
	steps []*step
}

func (*locationPath) Returns() DataType {
	return NodeSet
}

func (e *locationPath) Eval(ctx *Context) interface{} {
	var ns []dom.Node
	if e.abs {
		ns = []dom.Node{ctx.Document()}
	} else {
		ns = []dom.Node{ctx.Node}
	}
	return e.evalWith(ns, ctx)
}

func (e *locationPath) evalWith(ns []dom.Node, ctx *Context) interface{} {
	for _, s := range e.steps {
		ns = s.eval(ns, ctx.Vars)
	}
	if len(e.steps) > 1 {
		order(ns)
	}
	return ns
}

type step struct {
	iter       func(dom.Node) iterator
	test       func(dom.Node) bool
	predicates []Expr
	reverse    bool
}

func (s *step) eval(ctx []dom.Node, vars Variables) []dom.Node {
	var r []dom.Node
	unique := make(map[dom.Node]struct{})

	for _, c := range ctx {
		var cr []dom.Node
		iter := s.iter(c)

		// eval test
		for {
			n := iter.next()
			if n == nil {
				break
			}
			if _, ok := unique[n]; !ok {
				if s.test(n) {
					unique[n] = struct{}{}
					cr = append(cr, n)
				}
			}
		}

		cr = evalPredicates(s.predicates, cr, vars)
		r = append(r, cr...)
	}

	if s.reverse {
		reverse(r)
	}
	return r
}

func evalPredicates(predicates []Expr, ns []dom.Node, vars Variables) []dom.Node {
	for _, predicate := range predicates {
		var pr []dom.Node
		scontext := &Context{nil, 0, len(ns), vars}
		for _, n := range ns {
			scontext.Node = n
			scontext.Pos++
			pval := predicate.Eval(scontext)
			if i, ok := pval.(float64); ok {
				if scontext.Pos == int(i) {
					pr = append(pr, n)
				}
			} else if Value2Boolean(pval) {
				pr = append(pr, n)
			}
		}
		ns = pr
	}
	return ns
}

/************************************************************************/

type filterExpr struct {
	expr       Expr
	predicates []Expr
}

func (*filterExpr) Returns() DataType {
	return NodeSet
}

func (e *filterExpr) Eval(ctx *Context) interface{} {
	return evalPredicates(e.predicates, e.expr.Eval(ctx).([]dom.Node), ctx.Vars)
}

/************************************************************************/

type pathExpr struct {
	filter       Expr
	locationPath *locationPath
}

func (*pathExpr) Returns() DataType {
	return NodeSet
}

func (e *pathExpr) Eval(ctx *Context) interface{} {
	ns := e.filter.Eval(ctx).([]dom.Node)
	return e.locationPath.evalWith(ns, ctx)
}

/************************************************************************/

type variable struct {
	name    string
	returns DataType
}

func (v *variable) Returns() DataType {
	return v.returns
}

func (v *variable) Eval(ctx *Context) interface{} {
	if ctx.Vars == nil {
		panic(UnresolvedVariableError(v.name))
	}
	r := ctx.Vars.eval(v.name)
	if r == nil {
		panic(UnresolvedVariableError(v.name))
	}
	if v.returns == NodeSet {
		if _, ok := r.([]dom.Node); !ok {
			panic(VarMustBeNodeSet(v.name))
		}
	}
	TypeOf(r)
	return r
}

/************************************************************************/

type funcCall struct {
	args    []Expr
	returns DataType
	impl    func(args []interface{}) interface{}
}

func (f *funcCall) Returns() DataType {
	return f.returns
}

func (f *funcCall) Eval(ctx *Context) interface{} {
	args := make([]interface{}, len(f.args))
	for i, arg := range f.args {
		args[i] = arg.Eval(ctx)
	}
	return f.impl(args)
}

func (e *funcCall) Simplify() Expr {
	for i := range e.args {
		e.args[i] = Simplify(e.args[i])
	}
	if Literals(e.args...) {
		return Value2Expr(e.Eval(nil))
	}
	return e
}
