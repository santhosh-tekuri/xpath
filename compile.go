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
		case xpath.Add, xpath.Subtract, xpath.Multiply, xpath.Div, xpath.Mod:
			return &arithmeticExpr{asNumber(lhs), asNumber(rhs), arithmeticOp[e.Op-xpath.Add]}
		case xpath.And:
			return &logicalExpr{asBoolean(lhs), asBoolean(rhs), false}
		case xpath.Or:
			return &logicalExpr{asBoolean(lhs), asBoolean(rhs), true}
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
				var valueExpr, nodeSetExpr expr
				if lhs.resultType() == NodeSet {
					valueExpr, nodeSetExpr = rhs, lhs
				} else {
					valueExpr, nodeSetExpr = lhs, rhs
				}
				switch valueExpr.resultType() {
				case Boolean:
					return &valueEqualityExpr{valueExpr, asBoolean(nodeSetExpr), apply}
				case String:
					return &valuesEqualityExpr{valueExpr, nodeSetExpr, node2string, apply}
				case Number:
					return &valuesEqualityExpr{valueExpr, nodeSetExpr, node2number, apply}
				default:
					panic("impossible")
				}
			}
		case xpath.LT, xpath.LTE, xpath.GT, xpath.GTE:
			apply := relationalOp[e.Op-xpath.LT]
			if lhs.resultType() == NodeSet && rhs.resultType() == NodeSet {
				panic("relationalOp on nodesets is not implemented")
			} else if lhs.resultType() != NodeSet && rhs.resultType() != NodeSet {
				return &valueRelationalExpr{asNumber(lhs), asNumber(rhs), apply}
			} else {
				if lhs.resultType() == NodeSet {
					var op xpath.Op
					switch e.Op {
					case xpath.LT:
						op = xpath.GTE
					case xpath.LTE:
						op = xpath.GT
					case xpath.GT:
						op = xpath.LTE
					case xpath.GTE:
						op = xpath.LT
					}
					apply = relationalOp[op-xpath.LT]
					return &not{&valuesRelationalExpr{asNumber(rhs), lhs, apply}}
				} else {
					return &valuesRelationalExpr{asNumber(lhs), rhs, apply}
				}
			}
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
			switch estep.Axis {
			case xpath.Preceding, xpath.PrecedingSibling, xpath.Ancestor, xpath.AncestorOrSelf:
				s.reverse = true
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
			for _, epredicate := range estep.Predicates {
				predicate := c.compile(epredicate)
				switch predicate.resultType() {
				case Number:
					predicate = &valueEqualityExpr{position{}, predicate, equalityOp[0]}
				default:
					predicate = asBoolean(predicate)
				}
				s.predicates = append(s.predicates, predicate)
			}
		}
		return lp
	case *xpath.FuncCall:
		if e.Prefix == "" {
			function, ok := coreFunctions[e.Name]
			if !ok {
				panic(fmt.Sprintf("no such function %s", e.Name))
			}
			if !function.canAccept(len(e.Params)) {
				panic(fmt.Sprintf("wrong number of args to function %s", e.Name))
			}
			var args []expr
			if len(e.Params) > 0 {
				args = make([]expr, len(e.Params))
				for i, arg := range e.Params {
					arg := c.compile(arg)
					switch function.argType(i) {
					case String:
						args[i] = asString(arg)
					case Number:
						args[i] = asNumber(arg)
					case Boolean:
						args[i] = asBoolean(arg)
					default:
						if arg.resultType() != NodeSet {
							panic(fmt.Sprintf("argument at %d to function %s must be node-set", i, e.Name))
						}
						args[i] = arg
					}
				}
			}
			switch e.Name {
			case "string":
				if len(e.Params) == 0 {
					return &stringFunc{contextExpr{}}
				}
				return &stringFunc{args[0]}
			case "name":
				if len(e.Params) == 0 {
					return &qname{contextExpr{}}
				}
				return &qname{args[0]}
			case "local-name":
				if len(e.Params) == 0 {
					return &localName{contextExpr{}}
				}
				return &localName{args[0]}
			case "namespace-uri":
				if len(e.Params) == 0 {
					return &namespaceURI{contextExpr{}}
				}
				return &namespaceURI{args[0]}
			case "normalize-space":
				if len(e.Params) == 0 {
					return &normalizeSpace{asString(contextExpr{})}
				}
				return &normalizeSpace{args[0]}
			case "position":
				return &position{}
			case "count":
				return &count{args[0]}
			case "sum":
				return &sum{args[0]}
			case "not":
				return &not{args[0]}
			default:
				return &funcCall{args, function.returns, function.impl}
			}
		} else {
			panic("user functions is not implemented")
		}
	default:
		panic(fmt.Sprintf("compile(%T) is not implemented", e))
	}
}

type DataType int

const (
	NodeSet DataType = iota
	String
	Number
	Boolean
)

var resultTypeNames = []string{"node-set", "string", "number", "boolean"}

func (r DataType) String() string {
	return resultTypeNames[r]
}

type expr interface {
	resultType() DataType
	eval(ctx *context) interface{}
}

type contextExpr struct{}

func (contextExpr) resultType() DataType {
	return NodeSet
}

func (contextExpr) eval(ctx *context) interface{} {
	return []dom.Node{ctx.node}
}

/************************************************************************/

type numberVal float64

func (numberVal) resultType() DataType {
	return Number
}

func (e numberVal) eval(ctx *context) interface{} {
	return float64(e)
}

type stringVal string

func (stringVal) resultType() DataType {
	return String
}

func (e stringVal) eval(ctx *context) interface{} {
	return string(e)
}

type booleanVal bool

func (booleanVal) resultType() DataType {
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

func (*numberFunc) resultType() DataType {
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

func (*booleanFunc) resultType() DataType {
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

func (*stringFunc) resultType() DataType {
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
	arg expr
}

func (*negateExpr) resultType() DataType {
	return Number
}

func (e *negateExpr) eval(ctx *context) interface{} {
	return -e.arg.eval(ctx).(float64)
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
	lhs   expr
	rhs   expr
	apply func(float64, float64) float64
}

func (*arithmeticExpr) resultType() DataType {
	return Number
}

func (e *arithmeticExpr) eval(ctx *context) interface{} {
	return e.apply(e.lhs.eval(ctx).(float64), e.rhs.eval(ctx).(float64))
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

func (*valueEqualityExpr) resultType() DataType {
	return Boolean
}

func (e *valueEqualityExpr) eval(ctx *context) interface{} {
	lhs := e.lhs.eval(ctx)
	rhs := e.rhs.eval(ctx)
	return e.apply(lhs, rhs)
}

type valuesEqualityExpr struct {
	valueExpr   expr
	nodeSetExpr expr
	convert     func(dom.Node) interface{}
	apply       func(interface{}, interface{}) bool
}

func (*valuesEqualityExpr) resultType() DataType {
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

func (*valueRelationalExpr) resultType() DataType {
	return Boolean
}

func (e *valueRelationalExpr) eval(ctx *context) interface{} {
	lhs := e.lhs.eval(ctx)
	rhs := e.rhs.eval(ctx)
	return e.apply(lhs.(float64), rhs.(float64))
}

type valuesRelationalExpr struct {
	valueExpr   expr
	nodeSetExpr expr
	apply       func(float64, float64) bool
}

func (*valuesRelationalExpr) resultType() DataType {
	return Boolean
}

func (e *valuesRelationalExpr) eval(ctx *context) interface{} {
	value := e.valueExpr.eval(ctx).(float64)
	nodeSet := e.nodeSetExpr.eval(ctx).([]dom.Node)
	for _, n := range nodeSet {
		if e.apply(value, node2number(n).(float64)) {
			return true
		}
	}
	return false
}

/************************************************************************/

type logicalExpr struct {
	lhs      expr
	rhs      expr
	lhsValue bool
}

func (*logicalExpr) resultType() DataType {
	return Boolean
}

func (e *logicalExpr) eval(ctx *context) interface{} {
	if e.lhs.eval(ctx) == e.lhsValue {
		return e.lhsValue
	}
	return e.rhs.eval(ctx)
}

/************************************************************************/

type locationPath struct {
	abs   bool
	steps []*step
}

func (*locationPath) resultType() DataType {
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
	iter       func(dom.Node) iterator
	test       func(dom.Node) bool
	predicates []expr
	reverse    bool
}

func (s *step) eval(ctx []dom.Node) []dom.Node {
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

		// eval predicates
		for _, predicate := range s.predicates {
			var pr []dom.Node
			scontext := &context{nil, 0}
			for _, n := range cr {
				scontext.node = n
				scontext.pos++
				if predicate.eval(scontext).(bool) {
					pr = append(pr, n)
				}
			}
			cr = pr
		}

		r = append(r, cr...)
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

/************************************************************************/

type position struct{}

func (position) resultType() DataType {
	return Number
}

func (position) eval(ctx *context) interface{} {
	return float64(ctx.pos)
}

/************************************************************************/

type count struct {
	arg expr
}

func (*count) resultType() DataType {
	return Number
}

func (e *count) eval(ctx *context) interface{} {
	return float64(len(e.arg.eval(ctx).([]dom.Node)))
}

/************************************************************************/

type sum struct {
	arg expr
}

func (*sum) resultType() DataType {
	return Number
}

func (e *sum) eval(ctx *context) interface{} {
	var r float64
	for _, n := range e.arg.eval(ctx).([]dom.Node) {
		r += node2number(n).(float64)
	}
	return r
}

/************************************************************************/

type localName struct {
	arg expr
}

func (*localName) resultType() DataType {
	return String
}

func (e *localName) eval(ctx *context) interface{} {
	ns := e.arg.eval(ctx).([]dom.Node)
	if len(ns) > 0 {
		switch n := ns[0].(type) {
		case *dom.Element:
			return n.Local
		case *dom.Attr:
			return n.Local
		case *dom.ProcInst:
			return n.Target
		case *dom.NameSpace:
			return n.Prefix
		}
	}
	return ""
}

/************************************************************************/

type namespaceURI struct {
	arg expr
}

func (*namespaceURI) resultType() DataType {
	return String
}

func (e *namespaceURI) eval(ctx *context) interface{} {
	ns := e.arg.eval(ctx).([]dom.Node)
	if len(ns) > 0 {
		switch n := ns[0].(type) {
		case *dom.Element:
			return n.URI
		case *dom.Attr:
			return n.URI
		}
	}
	return ""
}

/************************************************************************/

type qname struct {
	arg expr
}

func (*qname) resultType() DataType {
	return String
}

func (e *qname) eval(ctx *context) interface{} {
	ns := e.arg.eval(ctx).([]dom.Node)
	if len(ns) > 0 {
		switch n := ns[0].(type) {
		case *dom.Element:
			return n.Name.String()
		case *dom.Attr:
			return n.Name.String()
		case *dom.ProcInst:
			return n.Target
		case *dom.NameSpace:
			return n.Prefix
		}
	}
	return ""
}

/************************************************************************/

type normalizeSpace struct {
	arg expr
}

func (*normalizeSpace) resultType() DataType {
	return String
}

func (e *normalizeSpace) eval(ctx *context) interface{} {
	buf := []byte(e.arg.eval(ctx).(string))
	read, write, lastWrite := 0, 0, 0
	wroteOne := false
	for read < len(buf) {
		b := buf[read]
		if isSpace(b) {
			if wroteOne {
				buf[write] = ' '
				write++
			}
			read++
			for read < len(buf) && isSpace(buf[read]) {
				read++
			}
		} else {
			buf[write] = buf[read]
			write++
			read++
			wroteOne = true
			lastWrite = write
		}
	}
	return string(buf[:lastWrite])
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

/************************************************************************/

type not struct {
	arg expr
}

func (*not) resultType() DataType {
	return Boolean
}

func (e *not) eval(ctx *context) interface{} {
	return !e.arg.eval(ctx).(bool)
}

/************************************************************************/

type funcCall struct {
	args    []expr
	returns DataType
	impl    func(ctx *context, args []interface{}) interface{}
}

func (f *funcCall) resultType() DataType {
	return f.returns
}

func (f *funcCall) eval(ctx *context) interface{} {
	args := make([]interface{}, len(f.args))
	for i, arg := range f.args {
		args[i] = arg.eval(ctx)
	}
	return f.impl(ctx, args)
}
