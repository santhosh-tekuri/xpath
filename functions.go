// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xpatheng

import (
	"bytes"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/santhosh-tekuri/dom"
)

type Function struct {
	Returns   DataType
	Args      []DataType
	Mandatory int
	Variadic  bool
	Compile   func(f *Function, args []Expr) Expr
}

func (f *Function) canAccept(nArgs int) bool {
	return nArgs >= f.Mandatory && (f.Variadic || nArgs <= len(f.Args))
}

func (f *Function) argType(i int) DataType {
	if i < len(f.Args) {
		return f.Args[i]
	}
	return f.Args[len(f.Args)-1]
}

func CompileFunc(impl func(args []interface{}) interface{}) func(f *Function, args []Expr) Expr {
	return func(f *Function, args []Expr) Expr {
		return &funcCall{args, f.Returns, impl}
	}
}

var coreFunctions = map[string]*Function{
	"string": {
		String, []DataType{Unknown}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &stringFunc{ContextExpr{}}
			}
			return &stringFunc{args[0]}
		}},
	"number": {
		Number, []DataType{Unknown}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &numberFunc{ContextExpr{}}
			}
			return &numberFunc{args[0]}
		}},
	"boolean": {
		Boolean, []DataType{Unknown}, 0, false,
		func(f *Function, args []Expr) Expr {
			return &booleanFunc{args[0]}
		}},
	"name": {
		String, []DataType{NodeSet}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &qname{ContextExpr{}}
			}
			return &qname{args[0]}
		}},
	"local-name": {
		String, []DataType{NodeSet}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &localName{ContextExpr{}}
			}
			return &localName{args[0]}
		}},
	"namespace-uri": {
		String, []DataType{NodeSet}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &namespaceURI{ContextExpr{}}
			}
			return &namespaceURI{args[0]}
		}},
	"position": {
		Number, nil, 0, false,
		func(f *Function, args []Expr) Expr {
			return &position{}
		}},
	"last": {
		Number, nil, 0, false,
		func(f *Function, args []Expr) Expr {
			return &last{}
		}},
	"count": {
		Number, []DataType{NodeSet}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &count{args[0]}
		}},
	"sum": {
		Number, []DataType{NodeSet}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &sum{args[0]}
		}},
	"floor": {
		Number, []DataType{Number}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &floor{args[0]}
		}},
	"ceiling": {
		Number, []DataType{Number}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &ceiling{args[0]}
		}},
	"round": {
		Number, []DataType{Number}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &round{args[0]}
		}},
	"normalize-space": {
		String, []DataType{String}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &normalizeSpace{asString(ContextExpr{})}
			}
			return &normalizeSpace{args[0]}
		}},
	"string-length": {
		Number, []DataType{String}, 0, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 0 {
				return &stringLength{asString(ContextExpr{})}
			}
			return &stringLength{args[0]}
		}},
	"starts-with": {
		Boolean, []DataType{String, String}, 2, false,
		func(f *Function, args []Expr) Expr {
			return &startsWith{args[0], args[1]}
		}},
	"ends-with": {
		Boolean, []DataType{String, String}, 2, false,
		func(f *Function, args []Expr) Expr {
			return &endsWith{args[0], args[1]}
		}},
	"contains": {
		Boolean, []DataType{String, String}, 2, false,
		func(f *Function, args []Expr) Expr {
			return &contains{args[0], args[1]}
		}},
	"concat": {
		String, []DataType{String, String}, 2, true,
		func(f *Function, args []Expr) Expr {
			return &concat{args}
		}},
	"translate": {
		String, []DataType{String, String, String}, 3, false,
		func(f *Function, args []Expr) Expr {
			return &translate{args[0], args[1], args[2]}
		}},
	"substring": {
		String, []DataType{String, Number, Number}, 2, false,
		func(f *Function, args []Expr) Expr {
			if len(args) == 3 {
				return &substring{args[0], args[1], args[2]}
			}
			return &substring{args[0], args[1], nil}
		}},
	"substring-before": {
		String, []DataType{String, String}, 2, false,
		func(f *Function, args []Expr) Expr {
			return &substringBefore{args[0], args[1]}
		}},
	"substring-after": {
		String, []DataType{String, String}, 2, false,
		func(f *Function, args []Expr) Expr {
			return &substringAfter{args[0], args[1]}
		}},
	"true": {
		Boolean, nil, 0, false,
		func(f *Function, args []Expr) Expr {
			return booleanVal(true)
		}},
	"false": {
		Boolean, nil, 0, false,
		func(f *Function, args []Expr) Expr {
			return booleanVal(false)
		}},
	"not": {
		Boolean, []DataType{Boolean}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &not{args[0]}
		}},
	"lang": {
		Boolean, []DataType{String}, 1, false,
		func(f *Function, args []Expr) Expr {
			return &lang{args[0]}
		}},
}

/************************************************************************/

type ContextExpr struct{}

func (ContextExpr) ResultType() DataType {
	return NodeSet
}

func (ContextExpr) Eval(ctx *Context) interface{} {
	return []dom.Node{ctx.Node}
}

/************************************************************************/

type numberFunc struct {
	arg Expr
}

func (*numberFunc) ResultType() DataType {
	return Number
}

func (f *numberFunc) Eval(ctx *Context) interface{} {
	return Value2Number(f.arg.Eval(ctx))
}

/************************************************************************/

type booleanFunc struct {
	arg Expr
}

func (*booleanFunc) ResultType() DataType {
	return Boolean
}

func (f *booleanFunc) Eval(ctx *Context) interface{} {
	return Value2Boolean(f.arg.Eval(ctx))
}

/************************************************************************/

type stringFunc struct {
	arg Expr
}

func (*stringFunc) ResultType() DataType {
	return String
}

func (e *stringFunc) Eval(ctx *Context) interface{} {
	return Value2String(e.arg.Eval(ctx))
}

/************************************************************************/

type position struct{}

func (position) ResultType() DataType {
	return Number
}

func (position) Eval(ctx *Context) interface{} {
	return float64(ctx.Pos)
}

/************************************************************************/

type last struct{}

func (last) ResultType() DataType {
	return Number
}

func (last) Eval(ctx *Context) interface{} {
	return float64(ctx.Size)
}

/************************************************************************/

type count struct {
	arg Expr
}

func (*count) ResultType() DataType {
	return Number
}

func (e *count) Eval(ctx *Context) interface{} {
	return float64(len(e.arg.Eval(ctx).([]dom.Node)))
}

/************************************************************************/

type sum struct {
	arg Expr
}

func (*sum) ResultType() DataType {
	return Number
}

func (e *sum) Eval(ctx *Context) interface{} {
	var r float64
	for _, n := range e.arg.Eval(ctx).([]dom.Node) {
		r += Node2Number(n)
	}
	return r
}

/************************************************************************/

type localName struct {
	arg Expr
}

func (*localName) ResultType() DataType {
	return String
}

func (e *localName) Eval(ctx *Context) interface{} {
	ns := e.arg.Eval(ctx).([]dom.Node)
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
	arg Expr
}

func (*namespaceURI) ResultType() DataType {
	return String
}

func (e *namespaceURI) Eval(ctx *Context) interface{} {
	ns := e.arg.Eval(ctx).([]dom.Node)
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
	arg Expr
}

func (*qname) ResultType() DataType {
	return String
}

func (e *qname) Eval(ctx *Context) interface{} {
	ns := e.arg.Eval(ctx).([]dom.Node)
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
	arg Expr
}

func (*normalizeSpace) ResultType() DataType {
	return String
}

func (e *normalizeSpace) Eval(ctx *Context) interface{} {
	buf := []byte(e.arg.Eval(ctx).(string))
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
	str    Expr
	prefix Expr
}

func (*startsWith) ResultType() DataType {
	return Boolean
}

func (e *startsWith) Eval(ctx *Context) interface{} {
	return strings.HasPrefix(e.str.Eval(ctx).(string), e.prefix.Eval(ctx).(string))
}

/************************************************************************/

type endsWith struct {
	str    Expr
	prefix Expr
}

func (*endsWith) ResultType() DataType {
	return Boolean
}

func (e *endsWith) Eval(ctx *Context) interface{} {
	return strings.HasSuffix(e.str.Eval(ctx).(string), e.prefix.Eval(ctx).(string))
}

/************************************************************************/

type contains struct {
	str    Expr
	substr Expr
}

func (*contains) ResultType() DataType {
	return Boolean
}

func (e *contains) Eval(ctx *Context) interface{} {
	return strings.Contains(e.str.Eval(ctx).(string), e.substr.Eval(ctx).(string))
}

/************************************************************************/

type stringLength struct {
	str Expr
}

func (*stringLength) ResultType() DataType {
	return Number
}

func (e *stringLength) Eval(ctx *Context) interface{} {
	return float64(utf8.RuneCountInString(e.str.Eval(ctx).(string)))
}

/************************************************************************/

type concat struct {
	args []Expr
}

func (*concat) ResultType() DataType {
	return String
}

func (e *concat) Eval(ctx *Context) interface{} {
	buf := new(bytes.Buffer)
	for _, arg := range e.args {
		buf.WriteString(arg.Eval(ctx).(string))
	}
	return buf.String()
}

/************************************************************************/

type translate struct {
	str  Expr
	from Expr
	to   Expr
}

func (*translate) ResultType() DataType {
	return String
}

func (e *translate) Eval(ctx *Context) interface{} {
	from := []rune(e.from.Eval(ctx).(string))
	to := []rune(e.to.Eval(ctx).(string))
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

	str := e.str.Eval(ctx).(string)
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
	str   Expr
	match Expr
}

func (*substringBefore) ResultType() DataType {
	return String
}

func (e *substringBefore) Eval(ctx *Context) interface{} {
	str := e.str.Eval(ctx).(string)
	if i := strings.Index(str, e.match.Eval(ctx).(string)); i != -1 {
		return str[:i]
	}
	return ""
}

/************************************************************************/

type substringAfter struct {
	str   Expr
	match Expr
}

func (*substringAfter) ResultType() DataType {
	return String
}

func (e *substringAfter) Eval(ctx *Context) interface{} {
	str := e.str.Eval(ctx).(string)
	match := e.match.Eval(ctx).(string)
	if i := strings.Index(str, match); i != -1 {
		return str[i+len(match):]
	}
	return ""
}

/************************************************************************/

type substring struct {
	str    Expr
	from   Expr
	length Expr
}

func (*substring) ResultType() DataType {
	return String
}

func (e *substring) Eval(ctx *Context) interface{} {
	str := e.str.Eval(ctx).(string)
	strLength := utf8.RuneCountInString(str)
	if strLength == 0 {
		return ""
	}

	d1 := e.from.Eval(ctx).(float64)
	if math.IsNaN(d1) {
		return ""
	}
	start := roundToInt(d1) - 1
	substrLength := strLength
	if e.length != nil {
		d2 := e.length.Eval(ctx).(float64)
		if math.IsInf(d2, +1) {
			substrLength = math.MaxInt16
		} else if math.IsInf(d2, -1) {
			substrLength = math.MinInt16
		} else if math.IsNaN(d2) {
			substrLength = 0
		} else {
			substrLength = roundToInt(d2)
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

func roundToInt(val float64) int {
	if val != 0.5 {
		return int(math.Floor(val + 0.5))
	}
	return 0
}

/************************************************************************/

type not struct {
	arg Expr
}

func (*not) ResultType() DataType {
	return Boolean
}

func (e *not) Eval(ctx *Context) interface{} {
	return !e.arg.Eval(ctx).(bool)
}

/************************************************************************/

type lang struct {
	lang Expr
}

func (*lang) ResultType() DataType {
	return Boolean
}

func (e *lang) Eval(ctx *Context) interface{} {
	lang := e.lang.Eval(ctx).(string)
	n := ctx.Node
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

type floor struct {
	num Expr
}

func (*floor) ResultType() DataType {
	return Number
}

func (e *floor) Eval(ctx *Context) interface{} {
	return math.Floor(e.num.Eval(ctx).(float64))
}

/************************************************************************/

type ceiling struct {
	num Expr
}

func (*ceiling) ResultType() DataType {
	return Number
}

func (e *ceiling) Eval(ctx *Context) interface{} {
	return math.Ceil(e.num.Eval(ctx).(float64))
}

/************************************************************************/

type round struct {
	num Expr
}

func (*round) ResultType() DataType {
	return Number
}

func (e *round) Eval(ctx *Context) interface{} {
	num := e.num.Eval(ctx).(float64)
	switch {
	case math.IsNaN(num) || math.IsInf(num, 0):
		return num
	case num != 0.5:
		return math.Floor(num + 0.5)
	default:
		return float64(0)
	}
}
