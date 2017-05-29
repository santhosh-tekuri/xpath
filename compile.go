// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/santhosh-tekuri/dom"
	"github.com/santhosh-tekuri/xpath"
)

func (c *Compiler) compile(e xpath.Expr) expr {
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
			s.predicates = c.compilePredicates(estep.Predicates)
		}
		return lp
	case *xpath.FilterExpr:
		return &filterExpr{c.compile(e.Expr), c.compilePredicates(e.Predicates)}
	case *xpath.PathExpr:
		return &pathExpr{c.compile(e.Filter), c.compile(e.LocationPath).(*locationPath)}
	case *xpath.FuncCall:
		if e.Prefix == "" {
			function, ok := coreFunctions[e.Name]
			if !ok {
				panic(UnresolvedFunctionError(e.Name))
			}
			if !function.canAccept(len(e.Params)) {
				panic(ArgCountError(e.Name))
			}
			var args []expr
			if len(e.Params) > 0 {
				args = make([]expr, len(e.Params))
				for i, arg := range e.Params {
					arg := c.compile(arg)
					switch function.argType(i) {
					case NodeSet:
						args[i] = asNodeSet(arg)
					case String:
						args[i] = asString(arg)
					case Number:
						args[i] = asNumber(arg)
					case Boolean:
						args[i] = asBoolean(arg)
					default:
						panic(fmt.Sprintf("unexpected arg type %v", function.argType(i)))
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
			case "string-length":
				if len(e.Params) == 0 {
					return &stringLength{asString(contextExpr{})}
				}
				return &stringLength{args[0]}
			case "substring":
				if len(e.Params) == 3 {
					return &substring{args[0], args[1], args[2]}
				}
				return &substring{args[0], args[1], nil}
			case "substring-before":
				return &substringBefore{args[0], args[1]}
			case "substring-after":
				return &substringAfter{args[0], args[1]}
			case "position":
				return &position{}
			case "true":
				return booleanVal(true)
			case "false":
				return booleanVal(false)
			case "count":
				return &count{args[0]}
			case "sum":
				return &sum{args[0]}
			case "not":
				return &not{args[0]}
			case "lang":
				return &lang{args[0]}
			case "starts-with":
				return &startsWith{args[0], args[1]}
			case "ends-with":
				return &endsWith{args[0], args[1]}
			case "contains":
				return &contains{args[0], args[1]}
			case "concat":
				if len(e.Params) == 2 {
					return &concat{args}
				}
				return &concat{args}
			case "translate":
				return &translate{args[0], args[1], args[2]}
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

func (c *Compiler) compilePredicates(predicates []xpath.Expr) []expr {
	var arr []expr
	for _, p := range predicates {
		predicate := c.compile(p)
		switch predicate.resultType() {
		case Number:
			predicate = &equalityExpr{position{}, predicate, equalityOp[0]}
		default:
			predicate = asBoolean(predicate)
		}
		arr = append(arr, predicate)
	}
	return arr
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
	return value2Number(f.arg.eval(ctx))
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
	return value2Boolean(f.arg.eval(ctx))
}

/************************************************************************/

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

func (e *stringFunc) eval(ctx *context) interface{} {
	return value2String(e.arg.eval(ctx))
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

type equalityExpr struct {
	lhs   expr
	rhs   expr
	apply func(interface{}, interface{}) bool
}

func (*equalityExpr) resultType() DataType {
	return Boolean
}

func (e *equalityExpr) eval(ctx *context) interface{} {
	lhs, rhs := e.lhs.eval(ctx), e.rhs.eval(ctx)
	lhsType, rhsType := typeOf(lhs), typeOf(rhs)
	switch {
	case lhsType == NodeSet && rhsType == NodeSet:
		lhs, rhs := lhs.([]dom.Node), rhs.([]dom.Node)
		if len(lhs) > 0 && len(rhs) > 0 {
			for _, n1 := range lhs {
				for _, n2 := range rhs {
					if e.apply(n1, n2) {
						return true
					}
				}
			}
		}
		return false
	case lhsType != NodeSet && rhsType != NodeSet:
		switch {
		case lhsType == Boolean || rhsType == Boolean:
			return e.apply(value2Boolean(lhs), value2Boolean(rhs))
		case lhsType == Number || rhsType == Number:
			return e.apply(value2Number(lhs), value2Number(rhs))
		default:
			return e.apply(value2String(lhs), value2String(rhs))
		}
	default:
		var val interface{}
		var nodeSet []dom.Node
		if lhsType == NodeSet {
			val, nodeSet = rhs, lhs.([]dom.Node)
		} else {
			val, nodeSet = lhs, rhs.([]dom.Node)
		}
		switch typeOf(val) {
		case Boolean:
			return e.apply(val, value2Boolean(nodeSet))
		case String:
			for _, n := range nodeSet {
				if e.apply(val, node2string(n)) {
					return true
				}
			}
			return false
		default:
			for _, n := range nodeSet {
				if e.apply(val, node2number(n)) {
					return true
				}
			}
			return false
		}
	}
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
	lhs   expr
	rhs   expr
	apply func(float64, float64) bool
}

func (*relationalExpr) resultType() DataType {
	return Boolean
}

func (e *relationalExpr) eval(ctx *context) interface{} {
	lhs, rhs := e.lhs.eval(ctx), e.rhs.eval(ctx)
	lhsType, rhsType := typeOf(lhs), typeOf(rhs)
	switch {
	case lhsType == NodeSet && rhsType == NodeSet:
		lhs, rhs := lhs.([]dom.Node), rhs.([]dom.Node)
		if len(lhs) > 0 && len(rhs) > 0 {
			for _, n1 := range lhs {
				for _, n2 := range rhs {
					if e.apply(node2number(n1), node2number(n2)) {
						return true
					}
				}
			}
		}
		return false
	case lhsType != NodeSet && rhsType != NodeSet:
		return e.apply(value2Number(lhs), value2Number(rhs))
	case lhsType == NodeSet:
		rhs := value2Number(rhs)
		for _, n := range lhs.([]dom.Node) {
			if e.apply(node2number(n), rhs) {
				return true
			}
		}
		return false
	default:
		lhs := value2Number(lhs)
		for _, n := range rhs.([]dom.Node) {
			if e.apply(lhs, node2number(n)) {
				return true
			}
		}
		return false
	}
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

type unionExpr struct {
	lhs expr
	rhs expr
}

func (*unionExpr) resultType() DataType {
	return NodeSet
}

func (e *unionExpr) eval(ctx *context) interface{} {
	lhs := e.lhs.eval(ctx).([]dom.Node)
	rhs := e.rhs.eval(ctx).([]dom.Node)
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
	return e.evalWith(ns, ctx)
}

func (e *locationPath) evalWith(ns []dom.Node, ctx *context) interface{} {
	for _, s := range e.steps {
		ns = s.eval(ns, ctx.vars)
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

func evalPredicates(predicates []expr, ns []dom.Node, vars Variables) []dom.Node {
	for _, predicate := range predicates {
		var pr []dom.Node
		scontext := &context{nil, 0, 1, vars}
		for _, n := range ns {
			scontext.node = n
			scontext.pos++
			if predicate.eval(scontext).(bool) {
				pr = append(pr, n)
			}
		}
		ns = pr
	}
	return ns
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

type filterExpr struct {
	expr       expr
	predicates []expr
}

func (*filterExpr) resultType() DataType {
	return NodeSet
}

func (e *filterExpr) eval(ctx *context) interface{} {
	return evalPredicates(e.predicates, e.expr.eval(ctx).([]dom.Node), ctx.vars)
}

/************************************************************************/

type pathExpr struct {
	filter       expr
	locationPath *locationPath
}

func (*pathExpr) resultType() DataType {
	return NodeSet
}

func (e *pathExpr) eval(ctx *context) interface{} {
	ns := e.filter.eval(ctx).([]dom.Node)
	return e.locationPath.evalWith(ns, ctx)
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
		r += node2number(n)
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

type startsWith struct {
	str    expr
	prefix expr
}

func (*startsWith) resultType() DataType {
	return Boolean
}

func (e *startsWith) eval(ctx *context) interface{} {
	return strings.HasPrefix(e.str.eval(ctx).(string), e.prefix.eval(ctx).(string))
}

/************************************************************************/

type endsWith struct {
	str    expr
	prefix expr
}

func (*endsWith) resultType() DataType {
	return Boolean
}

func (e *endsWith) eval(ctx *context) interface{} {
	return strings.HasSuffix(e.str.eval(ctx).(string), e.prefix.eval(ctx).(string))
}

/************************************************************************/

type contains struct {
	str    expr
	substr expr
}

func (*contains) resultType() DataType {
	return Boolean
}

func (e *contains) eval(ctx *context) interface{} {
	return strings.Contains(e.str.eval(ctx).(string), e.substr.eval(ctx).(string))
}

/************************************************************************/

type stringLength struct {
	str expr
}

func (*stringLength) resultType() DataType {
	return Number
}

func (e *stringLength) eval(ctx *context) interface{} {
	return float64(utf8.RuneCountInString(e.str.eval(ctx).(string)))
}

/************************************************************************/

type concat struct {
	args []expr
}

func (*concat) resultType() DataType {
	return String
}

func (e *concat) eval(ctx *context) interface{} {
	buf := new(bytes.Buffer)
	for _, arg := range e.args {
		buf.WriteString(arg.eval(ctx).(string))
	}
	return buf.String()
}

/************************************************************************/

type translate struct {
	str  expr
	from expr
	to   expr
}

func (*translate) resultType() DataType {
	return String
}

func (e *translate) eval(ctx *context) interface{} {
	from := []rune(e.from.eval(ctx).(string))
	to := []rune(e.to.eval(ctx).(string))
	replace := make(map[rune]rune)
	remove := make(map[rune]struct{})
	for i, frune := range from {
		if _, ok := replace[frune]; ok {
			continue
		}
		if _, ok := remove[frune]; ok {
			continue
		}
		if i < len(to) {
			replace[frune] = to[i]
		} else {
			remove[frune] = struct{}{}
		}
	}

	str := e.str.eval(ctx).(string)
	buf := bytes.NewBuffer(make([]byte, 0, len(str)))
	for _, r := range str {
		if _, ok := remove[r]; ok {
			continue
		}
		if v, ok := replace[r]; ok {
			buf.WriteRune(v)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

/************************************************************************/

type substringBefore struct {
	str   expr
	match expr
}

func (*substringBefore) resultType() DataType {
	return String
}

func (e *substringBefore) eval(ctx *context) interface{} {
	str := e.str.eval(ctx).(string)
	if i := strings.Index(str, e.match.eval(ctx).(string)); i != -1 {
		return str[:i]
	}
	return ""
}

/************************************************************************/

type substringAfter struct {
	str   expr
	match expr
}

func (*substringAfter) resultType() DataType {
	return String
}

func (e *substringAfter) eval(ctx *context) interface{} {
	str := e.str.eval(ctx).(string)
	match := e.match.eval(ctx).(string)
	if i := strings.Index(str, match); i != -1 {
		return str[i+len(match):]
	}
	return ""
}

/************************************************************************/

type substring struct {
	str    expr
	from   expr
	length expr
}

func (*substring) resultType() DataType {
	return String
}

func (e *substring) eval(ctx *context) interface{} {
	str := e.str.eval(ctx).(string)
	strLength := utf8.RuneCountInString(str)
	if strLength == 0 {
		return ""
	}

	d1 := e.from.eval(ctx).(float64)
	if math.IsNaN(d1) {
		return ""
	}
	start := round(d1) - 1
	substrLength := strLength
	if e.length != nil {
		d2 := e.length.eval(ctx).(float64)
		if math.IsInf(d2, +1) {
			substrLength = math.MaxInt16
		} else if math.IsInf(d2, -1) {
			substrLength = math.MinInt16
		} else if math.IsNaN(d2) {
			substrLength = 0
		} else {
			substrLength = round(d2)
		}
	}
	if substrLength < 0 {
		return ""
	}
	end := start + substrLength
	if e.length == nil {
		end = strLength
	}

	// negative start is treated as 0
	if start < 0 {
		start = 0
	} else if start > strLength {
		return ""
	}

	if end > strLength {
		end = strLength
	} else if end < start {
		return ""
	}

	if strLength == len(str) {
		return str[start:end]
	} else {
		return string([]rune(str)[start:end])
	}
}

func round(val float64) int {
	if val < 0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
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

type lang struct {
	lang expr
}

func (*lang) resultType() DataType {
	return Boolean
}

func (e *lang) eval(ctx *context) interface{} {
	lang := e.lang.eval(ctx).(string)
	n := ctx.node
	if _, ok := n.(*dom.Element); !ok {
		n = n.Parent()
	}
	for n != nil {
		if elem, ok := n.(*dom.Element); ok {
			attr := elem.GetAttr("http://www.w3.org/XML/1998/namespace", "lang")
			if attr != nil {
				sublang := attr.Value
				if strings.EqualFold(sublang, lang) {
					return true
				}
				ll := len(lang)
				return len(sublang) > ll && sublang[ll] == '-' && strings.EqualFold(sublang[:ll], lang)
			}
		} else {
			break
		}
		n = n.Parent()
	}
	return false
}

/************************************************************************/

func asNodeSet(e expr) expr {
	if v, ok := e.(*variable); ok {
		v.returns = NodeSet
	} else if e.resultType() != NodeSet {
		panic("node-set expected")
	}
	return e
}

type variable struct {
	name    string
	returns DataType
}

func (v *variable) resultType() DataType {
	return v.returns
}

func (v *variable) eval(ctx *context) interface{} {
	r := ctx.vars.eval(v.name)
	if r == nil {
		panic(UnresolvedVariableError(v.name))
	}
	if v.returns == NodeSet {
		if _, ok := r.([]dom.Node); !ok {
			panic(VarMustBeNodeSet(v.name))
		}
	}
	typeOf(r)
	return r
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
