// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
)

func (c *Compiler) compile(e xpath.Expr) expr {
	switch e := e.(type) {
	case xpath.Number:
		return numberVal(e)
	case xpath.String:
		return stringVal(e)
	case *xpath.NegateExpr:
		return &negateExpr{asNumber(c.compile(e.Expr))}
	case *xpath.BinaryExpr:
		lhs := c.compile(e.LHS)
		rhs := c.compile(e.RHS)
		switch e.Op {
		case xpath.Add:
			return &addExpr{asNumber(lhs), asNumber(rhs)}
		case xpath.Subtract:
			return &subtractExpr{asNumber(lhs), asNumber(rhs)}
		case xpath.Multiply:
			return &multiplyExpr{asNumber(lhs), asNumber(rhs)}
		case xpath.Div:
			return &divExpr{asNumber(lhs), asNumber(rhs)}
		case xpath.Mod:
			return &modExpr{asNumber(lhs), asNumber(rhs)}
		case xpath.EQ, xpath.NEQ:
			apply := equalityOp[e.Op]
			if lhs.resultType() == NodeSet && rhs.resultType() == NodeSet {
				panic("equalitiy on nodesets is not implemented")
			} else if lhs.resultType() != NodeSet && rhs.resultType() != NodeSet {
				if lhs.resultType() == Boolean || rhs.resultType() == Boolean {
					return &valueEqualityExpr{asBoolean(lhs), asBoolean(rhs), apply}
				}
				if lhs.resultType() == Number || rhs.resultType() == Number {
					return &valueEqualityExpr{asNumber(lhs), asNumber(rhs), apply}
				}
				return &valueEqualityExpr{asString(lhs), asString(rhs), apply}
			} else {
				var nodeSetExpr, valueExpr expr
				if lhs.resultType() == NodeSet {
					nodeSetExpr, valueExpr = lhs, rhs
				} else {
					valueExpr, nodeSetExpr = lhs, rhs
				}
				switch valueExpr.resultType() {
				case Boolean:
					return &valueEqualityExpr{valueExpr, asBoolean(nodeSetExpr), apply}
				case String:
					return &valuesEqualityExpr{nodeSetExpr, valueExpr, node2string, apply}
				case Number:
					return &valuesEqualityExpr{nodeSetExpr, valueExpr, node2number, apply}
				default:
					panic("impossible")
				}
			}
			panic(fmt.Sprintf("binaryOp %v for nodeset is not implemented", e.Op))
		case xpath.LT, xpath.LTE, xpath.GT, xpath.GTE:
			apply := relationalOp[e.Op]
			if lhs.resultType() != NodeSet && rhs.resultType() != NodeSet {
				return &valueRelationalExpr{asNumber(lhs), asNumber(rhs), apply}
			}
			panic(fmt.Sprintf("binaryOp %v for nodeset is not implemented", e.Op))
		default:
			panic(fmt.Sprintf("binaryOp %v is not implemented", e.Op))
		}
	case *xpath.LocationPath:
		lp := new(locationPath)
		lp.abs = e.Abs
		for _, estep := range e.Steps {
			s := new(step)
			lp.steps = append(lp.steps, s)
			s.iter = iterators[estep.Axis]
			if s.iter == nil {
				panic(fmt.Sprintf("axis %v is not implemented", estep.Axis))
			}
			switch test := estep.NodeTest.(type) {
			case xpath.NodeType:
				switch test {
				case xpath.Node:
					s.test = alwaysTrue
				case xpath.Comment:
					s.test = isComment
				case xpath.Text:
					s.test = isText
				}
			case xpath.PITest:
				s.test = isProcInst(string(test))
			case *xpath.NameTest:
				uri, ok := c.resolvePrefix(test.Prefix)
				if !ok {
					panic("unresolved prefix " + test.Prefix)
				}
				switch estep.Axis {
				case xpath.Attribute:
					if test.Local == "*" {
						if uri == "" {
							s.test = alwaysTrue
						} else {
							s.test = testAttrNs(uri)
						}
					} else {
						s.test = testAttrName(uri, test.Local)
					}
				case xpath.Namespace:
					if test.Prefix == "" && test.Local == "*" {
						s.test = alwaysTrue
					} else {
						s.test = testNamespaceName(uri, test.Local)
					}
				default:
					if test.Local == "*" {
						if uri == "" {
							s.test = isElement
						} else {
							s.test = testElementNS(uri)
						}
					} else {
						s.test = testElementName(uri, test.Local)
					}
				}
			}
		}
		return lp
	case *xpath.FuncCall:
		if e.Prefix == "" {
			switch e.Name {
			case "string":
				if len(e.Params) == 0 {
					return &stringFunc{contextExpr{}}
				} else if len(e.Params) == 1 {
					return &stringFunc{c.compile(e.Params[0])}
				} else {
					panic("string function with non-zero args is not implemented")
				}
			default:
				panic(fmt.Sprintf("function %s is not implemented", e.Name))
			}
		} else {
			panic("user functions is not implemented")
		}
	default:
		panic(fmt.Sprintf("compile(%T) is not implemented", e))
	}
}

type ResultType int

const (
	NodeSet ResultType = iota
	String
	Number
	Boolean
)

var resultTypeNames = []string{"node-set", "string", "number", "boolean"}

func (r ResultType) String() string {
	return resultTypeNames[r]
}

type expr interface {
	resultType() ResultType
	eval(ctx *context) interface{}
}

type contextExpr struct{}

func (contextExpr) resultType() ResultType {
	return NodeSet
}

func (contextExpr) eval(ctx *context) interface{} {
	return []dom.Node{ctx.node}
}

/************************************************************************/

type numberVal float64

func (numberVal) resultType() ResultType {
	return Number
}

func (e numberVal) eval(ctx *context) interface{} {
	return float64(e)
}

type stringVal string

func (stringVal) resultType() ResultType {
	return String
}

func (e stringVal) eval(ctx *context) interface{} {
	return string(e)
}

type booleanVal bool

func (booleanVal) resultType() ResultType {
	return Boolean
}

func (e booleanVal) eval(ctx *context) interface{} {
	return bool(e)
}

/************************************************************************/

func string2number(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}

func asNumber(expr expr) expr {
	if expr.resultType() == Number {
		return expr
	}
	return &numberFunc{expr}
}

type numberFunc struct {
	arg expr
}

func (*numberFunc) resultType() ResultType {
	return Number
}

func (f *numberFunc) eval(ctx *context) interface{} {
	r := f.arg.eval(ctx)
	switch r := r.(type) {
	case float64:
		return r
	case bool:
		if r {
			return float64(1)
		}
		return float64(0)
	case string:
		return string2number(r)
	default:
		panic(fmt.Sprintf("numberFunc(%T) is not implemented", r))
	}
}

/************************************************************************/

func asBoolean(expr expr) expr {
	if expr.resultType() == Boolean {
		return expr
	}
	return &booleanFunc{expr}
}

type booleanFunc struct {
	arg expr
}

func (*booleanFunc) resultType() ResultType {
	return Boolean
}

func (f *booleanFunc) eval(ctx *context) interface{} {
	r := f.arg.eval(ctx)
	switch r := r.(type) {
	case float64:
		if r == 0 || math.IsNaN(r) {
			return false
		}
		return true
	case bool:
		return r
	case string:
		return len(r) > 0
	case []dom.Node:
		return len(r) > 0
	}
	panic("impossible")
}

/************************************************************************/

func node2string(n dom.Node) interface{} {
	return textContent(n)
}

func node2number(n dom.Node) interface{} {
	return string2number(textContent(n))
}

func textContent(n dom.Node) string {
	switch n := n.(type) {
	case *dom.Comment:
		return n.Data
	case *dom.ProcInst:
		return n.Data
	case *dom.Text:
		return n.Data
	case *dom.NameSpace:
		return n.URI
	case *dom.Attr:
		return n.Value
	default:
		buf := new(bytes.Buffer)
		collectText(n, buf)
		return buf.String()
	}
}

func collectText(n dom.Node, buf *bytes.Buffer) {
	if t, ok := n.(*dom.Text); ok {
		buf.WriteString(t.Data)
	} else if p, ok := n.(dom.Parent); ok {
		for _, c := range p.Children() {
			collectText(c, buf)
		}
	}
}

func asString(expr expr) expr {
	if expr.resultType() == String {
		return expr
	}
	return &stringFunc{expr}
}

type stringFunc struct {
	arg expr
}

func (*stringFunc) resultType() ResultType {
	return String
}

func (f *stringFunc) eval(ctx *context) interface{} {
	r := f.arg.eval(ctx)
	switch r := r.(type) {
	case string:
		return r
	case float64:
		if math.IsNaN(r) {
			return "NaN"
		}
		if math.IsInf(r, +1) {
			return "Infinity"
		}
		if math.IsInf(r, -1) {
			return "-Infinity"
		}
		if r == math.Trunc(r) {
			return strconv.Itoa(int(r))
		}
		return strconv.FormatFloat(r, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(r)
	case []dom.Node:
		if len(r) == 0 {
			return ""
		}
		return textContent(r[0])
	default:
		panic(fmt.Sprintf("stringFunc(%T) is not implemented", r))
	}
}

/************************************************************************/

type negateExpr struct {
	num expr
}

func (*negateExpr) resultType() ResultType {
	return Number
}

func (e *negateExpr) eval(ctx *context) interface{} {
	return -e.num.eval(ctx).(float64)
}

/************************************************************************/

type addExpr struct {
	lhs expr
	rhs expr
}

func (*addExpr) resultType() ResultType {
	return Number
}

func (e *addExpr) eval(ctx *context) interface{} {
	return e.lhs.eval(ctx).(float64) + e.rhs.eval(ctx).(float64)
}

/************************************************************************/

type subtractExpr struct {
	lhs expr
	rhs expr
}

func (*subtractExpr) resultType() ResultType {
	return Number
}

func (s *subtractExpr) eval(ctx *context) interface{} {
	return s.lhs.eval(ctx).(float64) - s.rhs.eval(ctx).(float64)
}

/************************************************************************/

type multiplyExpr struct {
	lhs expr
	rhs expr
}

func (*multiplyExpr) resultType() ResultType {
	return Number
}

func (m *multiplyExpr) eval(ctx *context) interface{} {
	return m.lhs.eval(ctx).(float64) * m.rhs.eval(ctx).(float64)
}

/************************************************************************/

type divExpr struct {
	lhs expr
	rhs expr
}

func (*divExpr) resultType() ResultType {
	return Number
}

func (d *divExpr) eval(ctx *context) interface{} {
	return d.lhs.eval(ctx).(float64) / d.rhs.eval(ctx).(float64)
}

/************************************************************************/

type modExpr struct {
	lhs expr
	rhs expr
}

func (*modExpr) resultType() ResultType {
	return Number
}

func (m *modExpr) eval(ctx *context) interface{} {
	return math.Mod(m.lhs.eval(ctx).(float64), m.rhs.eval(ctx).(float64))
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

type valueEqualityExpr struct {
	lhs   expr
	rhs   expr
	apply func(interface{}, interface{}) bool
}

func (*valueEqualityExpr) resultType() ResultType {
	return Boolean
}

func (e *valueEqualityExpr) eval(ctx *context) interface{} {
	lhs := e.lhs.eval(ctx)
	rhs := e.rhs.eval(ctx)
	return e.apply(lhs, rhs)
}

type valuesEqualityExpr struct {
	nodeSetExpr expr
	valueExpr   expr
	convert     func(dom.Node) interface{}
	apply       func(interface{}, interface{}) bool
}

func (*valuesEqualityExpr) resultType() ResultType {
	return Boolean
}

func (e *valuesEqualityExpr) eval(ctx *context) interface{} {
	value := e.valueExpr.eval(ctx)
	nodeSet := e.nodeSetExpr.eval(ctx).([]dom.Node)
	for _, n := range nodeSet {
		if e.apply(value, e.convert(n)) {
			return true
		}
	}
	return false
}

/************************************************************************/

var relationalOp = []func(float64, float64) bool{
	nil, nil,
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

type valueRelationalExpr struct {
	lhs   expr
	rhs   expr
	apply func(float64, float64) bool
}

func (*valueRelationalExpr) resultType() ResultType {
	return Boolean
}

func (e *valueRelationalExpr) eval(ctx *context) interface{} {
	lhs := e.lhs.eval(ctx)
	rhs := e.rhs.eval(ctx)
	return e.apply(lhs.(float64), rhs.(float64))
}

/************************************************************************/

type locationPath struct {
	abs   bool
	steps []*step
}

func (*locationPath) resultType() ResultType {
	return NodeSet
}

func (e *locationPath) eval(ctx *context) interface{} {
	var ns []dom.Node
	if e.abs {
		ns = []dom.Node{document(ctx.node)}
	} else {
		ns = []dom.Node{ctx.node}
	}
	for _, s := range e.steps {
		ns = s.eval(ns)
	}
	if len(e.steps) > 1 {
		order(ns)
	}
	return ns
}

type step struct {
	iter    func(dom.Node) iterator
	test    func(dom.Node) bool
	reverse bool
}

func (s *step) eval(ctx []dom.Node) []dom.Node {
	var r []dom.Node
	unique := make(map[dom.Node]struct{})
	for _, c := range ctx {
		iter := s.iter(c)
		for {
			n := iter.next()
			if n == nil {
				break
			}
			if s.test(n) {
				if _, ok := unique[n]; !ok {
					unique[n] = struct{}{}
					r = append(r, n)
				}
			}
		}
	}
	if s.reverse {
		reverse(r)
	}
	return r
}

func document(n dom.Node) dom.Node {
	for {
		if _, ok := n.(*dom.Document); ok {
			return n
		}
		n = parent(n)
	}
}
