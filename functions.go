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
	Impl      func(ctx *Context, args []interface{}) interface{}
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

var coreFunctions = map[string]*Function{
	"string":           {String, []DataType{Unknown}, 0, false, nil},
	"boolean":          {Boolean, []DataType{Unknown}, 0, false, nil},
	"name":             {String, []DataType{NodeSet}, 0, false, nil},
	"local-name":       {String, []DataType{NodeSet}, 0, false, nil},
	"namespace-uri":    {String, []DataType{NodeSet}, 0, false, nil},
	"position":         {Number, nil, 0, false, nil},
	"last":             {Number, nil, 0, false, nil},
	"count":            {Number, []DataType{NodeSet}, 1, false, nil},
	"sum":              {Number, []DataType{NodeSet}, 1, false, nil},
	"normalize-space":  {String, []DataType{String}, 0, false, nil},
	"string-length":    {Number, []DataType{String}, 0, false, nil},
	"starts-with":      {Boolean, []DataType{String, String}, 2, false, nil},
	"ends-with":        {Boolean, []DataType{String, String}, 2, false, nil},
	"contains":         {Boolean, []DataType{String, String}, 2, false, nil},
	"concat":           {String, []DataType{String, String}, 2, true, nil},
	"translate":        {String, []DataType{String, String, String}, 3, false, nil},
	"substring":        {String, []DataType{String, Number, Number}, 2, false, nil},
	"substring-before": {String, []DataType{String, String}, 2, false, nil},
	"substring-after":  {String, []DataType{String, String}, 2, false, nil},
	"true":             {Boolean, nil, 0, false, nil},
	"false":            {Boolean, nil, 0, false, nil},
	"not":              {Boolean, []DataType{Boolean}, 1, false, nil},
	"lang":             {Boolean, []DataType{String}, 1, false, nil},
}

func coreFunction(name string, args []expr) expr {
	switch name {
	case "string":
		if len(args) == 0 {
			return &stringFunc{contextExpr{}}
		}
		return &stringFunc{args[0]}
	case "boolean":
		return &booleanFunc{args[0]}
	case "name":
		if len(args) == 0 {
			return &qname{contextExpr{}}
		}
		return &qname{args[0]}
	case "local-name":
		if len(args) == 0 {
			return &localName{contextExpr{}}
		}
		return &localName{args[0]}
	case "namespace-uri":
		if len(args) == 0 {
			return &namespaceURI{contextExpr{}}
		}
		return &namespaceURI{args[0]}
	case "normalize-space":
		if len(args) == 0 {
			return &normalizeSpace{asString(contextExpr{})}
		}
		return &normalizeSpace{args[0]}
	case "string-length":
		if len(args) == 0 {
			return &stringLength{asString(contextExpr{})}
		}
		return &stringLength{args[0]}
	case "substring":
		if len(args) == 3 {
			return &substring{args[0], args[1], args[2]}
		}
		return &substring{args[0], args[1], nil}
	case "substring-before":
		return &substringBefore{args[0], args[1]}
	case "substring-after":
		return &substringAfter{args[0], args[1]}
	case "position":
		return &position{}
	case "last":
		return &last{}
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
		return &concat{args}
	case "translate":
		return &translate{args[0], args[1], args[2]}
	}
	return nil
}

/************************************************************************/

type contextExpr struct{}

func (contextExpr) resultType() DataType {
	return NodeSet
}

func (contextExpr) eval(ctx *Context) interface{} {
	return []dom.Node{ctx.Node}
}

/************************************************************************/

type numberFunc struct {
	arg expr
}

func (*numberFunc) resultType() DataType {
	return Number
}

func (f *numberFunc) eval(ctx *Context) interface{} {
	return Value2Number(f.arg.eval(ctx))
}

/************************************************************************/

type booleanFunc struct {
	arg expr
}

func (*booleanFunc) resultType() DataType {
	return Boolean
}

func (f *booleanFunc) eval(ctx *Context) interface{} {
	return Value2Boolean(f.arg.eval(ctx))
}

/************************************************************************/

type stringFunc struct {
	arg expr
}

func (*stringFunc) resultType() DataType {
	return String
}

func (e *stringFunc) eval(ctx *Context) interface{} {
	return Value2String(e.arg.eval(ctx))
}

/************************************************************************/

type position struct{}

func (position) resultType() DataType {
	return Number
}

func (position) eval(ctx *Context) interface{} {
	return float64(ctx.Pos)
}

/************************************************************************/

type last struct{}

func (last) resultType() DataType {
	return Number
}

func (last) eval(ctx *Context) interface{} {
	return float64(ctx.Size)
}

/************************************************************************/

type count struct {
	arg expr
}

func (*count) resultType() DataType {
	return Number
}

func (e *count) eval(ctx *Context) interface{} {
	return float64(len(e.arg.eval(ctx).([]dom.Node)))
}

/************************************************************************/

type sum struct {
	arg expr
}

func (*sum) resultType() DataType {
	return Number
}

func (e *sum) eval(ctx *Context) interface{} {
	var r float64
	for _, n := range e.arg.eval(ctx).([]dom.Node) {
		r += Node2Number(n)
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

func (e *localName) eval(ctx *Context) interface{} {
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

func (e *namespaceURI) eval(ctx *Context) interface{} {
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

func (e *qname) eval(ctx *Context) interface{} {
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

func (e *normalizeSpace) eval(ctx *Context) interface{} {
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

func (e *startsWith) eval(ctx *Context) interface{} {
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

func (e *endsWith) eval(ctx *Context) interface{} {
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

func (e *contains) eval(ctx *Context) interface{} {
	return strings.Contains(e.str.eval(ctx).(string), e.substr.eval(ctx).(string))
}

/************************************************************************/

type stringLength struct {
	str expr
}

func (*stringLength) resultType() DataType {
	return Number
}

func (e *stringLength) eval(ctx *Context) interface{} {
	return float64(utf8.RuneCountInString(e.str.eval(ctx).(string)))
}

/************************************************************************/

type concat struct {
	args []expr
}

func (*concat) resultType() DataType {
	return String
}

func (e *concat) eval(ctx *Context) interface{} {
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

func (e *translate) eval(ctx *Context) interface{} {
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

func (e *substringBefore) eval(ctx *Context) interface{} {
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

func (e *substringAfter) eval(ctx *Context) interface{} {
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

func (e *substring) eval(ctx *Context) interface{} {
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

func (e *not) eval(ctx *Context) interface{} {
	return !e.arg.eval(ctx).(bool)
}

/************************************************************************/

type lang struct {
	lang expr
}

func (*lang) resultType() DataType {
	return Boolean
}

func (e *lang) eval(ctx *Context) interface{} {
	lang := e.lang.eval(ctx).(string)
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
