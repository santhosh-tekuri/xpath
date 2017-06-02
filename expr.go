// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"github.com/santhosh-tekuri/dom"
	"math"
)

// Expr is interface used to represent a specific type of xpath expression.
type Expr interface {
	// Returns returns the DataType of the value that this expression evaluates to.
	Returns() DataType

	// Eval evaluates against the given context and returns the value
	// In case of any error, it will panic
	Eval(ctx *Context) interface{}
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

// ContextExpr represents current node in context.
type ContextExpr struct{}

// Returns returns the DataType of the value that this expression evaluates to.
func (ContextExpr) Returns() DataType {
	return NodeSet
}

// Eval returns []dom.Node of length 1 which contains the current node in context-set
func (ContextExpr) Eval(ctx *Context) interface{} {
	return []dom.Node{ctx.Node}
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
				if n1 := Node2Number(n1); !math.IsNaN(n1) {
					for _, n2 := range rhs {
						if e.apply(n1, Node2Number(n2)) {
							return true
						}
					}
				}
			}
		}
		return false
	case lhsType != NodeSet && rhsType != NodeSet:
		return e.apply(Value2Number(lhs), Value2Number(rhs))
	case lhsType == NodeSet:
		if rhs := Value2Number(rhs); !math.IsNaN(rhs) {
			for _, n := range lhs.([]dom.Node) {
				if e.apply(Node2Number(n), rhs) {
					return true
				}
			}
		}
		return false
	default:
		if lhs := Value2Number(lhs); !math.IsNaN(lhs) {
			for _, n := range rhs.([]dom.Node) {
				if e.apply(lhs, Node2Number(n)) {
					return true
				}
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
	lhs       Expr
	rhs       Expr
	skipValue bool
}

func (*logicalExpr) Returns() DataType {
	return Boolean
}

func (e *logicalExpr) Eval(ctx *Context) interface{} {
	if e.lhs.Eval(ctx) == e.skipValue {
		return e.skipValue
	}
	return e.rhs.Eval(ctx)
}

func (e *logicalExpr) Simplify() Expr {
	e.lhs, e.rhs = Simplify(e.lhs), Simplify(e.rhs)
	switch {
	case Literals(e.lhs):
		if e.lhs.Eval(nil) == e.skipValue {
			return Value2Expr(e.skipValue)
		}
		return e.rhs
	case Literals(e.rhs):
		if e.rhs.Eval(nil) == e.skipValue {
			return Value2Expr(e.skipValue)
		}
		return e.lhs
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
	switch {
	case len(lhs) == 0:
		return rhs
	case len(rhs) == 0:
		return lhs
	default:
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
}

/************************************************************************/

type predicates []Expr

func (p predicates) eval(ns []dom.Node, vars Variables) []dom.Node {
	for _, predicate := range p {
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
	orderReqd := len(ns) > 1 || len(e.steps) > 1
	for _, s := range e.steps {
		ns = s.eval(ns, ctx.Vars)
	}
	if orderReqd {
		order(ns)
	}
	return ns
}

func (e *locationPath) Simplify() Expr {
	for _, s := range e.steps {
		for i := range s.predicates {
			s.predicates[i] = Simplify(s.predicates[i])
		}
	}
	return e
}

type step struct {
	iter       func(dom.Node) Iterator
	test       func(dom.Node) bool
	predicates predicates
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
			n := iter.Next()
			if n == nil {
				break
			}
			if _, ok := unique[n]; !ok && s.test(n) {
				unique[n] = struct{}{}
				cr = append(cr, n)
			}
		}

		cr = s.predicates.eval(cr, vars)
		r = append(r, cr...)
	}

	if s.reverse {
		reverse(r)
	}
	return r
}

/************************************************************************/

type filterExpr struct {
	expr       Expr
	predicates predicates
}

func (*filterExpr) Returns() DataType {
	return NodeSet
}

func (e *filterExpr) Eval(ctx *Context) interface{} {
	return e.predicates.eval(e.expr.Eval(ctx).([]dom.Node), ctx.Vars)
}

func (e *filterExpr) Simplify() Expr {
	e.expr = Simplify(e.expr)
	for i := range e.predicates {
		e.predicates[i] = Simplify(e.predicates[i])
	}
	return e
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

func (e *pathExpr) Simplify() Expr {
	e.filter = Simplify(e.filter)
	e.locationPath = Simplify(e.locationPath).(*locationPath)
	return e
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
	r := ctx.Vars.Eval(v.name)
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

func (e *funcCall) Returns() DataType {
	return e.returns
}

func (e *funcCall) Eval(ctx *Context) interface{} {
	args := make([]interface{}, len(e.args))
	for i, arg := range e.args {
		args[i] = arg.Eval(ctx)
	}
	return e.impl(args)
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
